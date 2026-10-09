package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"regexp"
	"sort"
)

var managedSHA = regexp.MustCompile(`^[0-9a-f]{64}$`)

func sourceCurrent(d knowledgeadmin.SourceDocument, p knowledgeadmin.SourcePoint) knowledgeadmin.CurrentInput {
	var url *string
	if d.Source.URL != "" {
		v := d.Source.URL
		url = &v
	}
	return knowledgeadmin.CurrentInput{ExternalID: p.ID, Point: p, Sources: []knowledgeadmin.PublicSource{{SourceID: d.Source.SourceID, Title: d.Source.Title, Citation: d.Source.Attribution, URL: url}}}
}
func countPreview(out *knowledgeadmin.Preview, item knowledgeadmin.PreviewItem) {
	switch item.Action {
	case "create":
		out.Counts.CreatedKnowledge++
		out.Counts.LinkedTopics += len(item.TopicKeys)
	case "link":
		out.Counts.LinkedTopics += len(item.TopicKeys)
	case "skip":
		out.Counts.SkippedItems++
	case "conflict":
		out.Counts.Conflicts++
	case "invalid":
		out.Counts.InvalidItems++
	}
}
func previewManaged(ctx context.Context, tx *sql.Tx, d knowledgeadmin.SourceDocument) ([]knowledgeadmin.PreviewItem, error) {
	index, e := knowledgeadmin.LoadClassificationIndex()
	if e != nil {
		return nil, e
	}
	out := []knowledgeadmin.PreviewItem{}
	pending := map[string]knowledgeadmin.SourcePoint{}
	pendingTopics := map[string]map[string]bool{}
	for n, p := range d.KnowledgePoints {
		item := knowledgeadmin.PreviewItem{Index: n, ExternalID: p.ID, Action: "create", TopicKeys: knowledgeadmin.TopicKeys(p)}
		if e = knowledgeadmin.ValidatePoint(p, index); e != nil {
			item.Action = "invalid"
			item.ErrorCode = "INVALID_FIELD"
			if de, ok := e.(*knowledgeadmin.DecodeError); ok {
				item.ErrorPath = de.Path
			}
			out = append(out, item)
			continue
		}
		old, exists, e := existingExternal(ctx, tx, p.ID)
		if e != nil {
			return nil, e
		}
		if prior, ok := pending[p.ID]; ok {
			a, _ := knowledgeadmin.SourceCoreSHA(prior)
			b, _ := knowledgeadmin.SourceCoreSHA(p)
			if a != b {
				item.Action = "conflict"
				item.ErrorCode = "ID_CONTENT_CONFLICT"
			} else {
				newTopics := []string{}
				for _, t := range item.TopicKeys {
					if !pendingTopics[p.ID][t] {
						newTopics = append(newTopics, t)
						pendingTopics[p.ID][t] = true
					}
				}
				item.TopicKeys = newTopics
				if len(newTopics) == 0 {
					item.Action = "skip"
				} else {
					item.Action = "link"
				}
			}
		} else if exists {
			if old.Deleted {
				item.Action = "skip"
				item.TopicKeys = []string{}
			} else {
				known, e := knownSourceCore(ctx, tx, old, p)
				if e != nil {
					return nil, e
				}
				if !known {
					item.Action = "conflict"
					item.ErrorCode = "ID_CONTENT_CONFLICT"
				} else {
					newTopics := []string{}
					for _, t := range item.TopicKeys {
						var present bool
						e = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM managed_knowledge_topics WHERE external_id=$1 AND topic_key=$2)", p.ID, t).Scan(&present)
						if e != nil {
							return nil, e
						}
						if !present {
							newTopics = append(newTopics, t)
						}
					}
					item.TopicKeys = newTopics
					if len(newTopics) == 0 {
						item.Action = "skip"
					} else {
						item.Action = "link"
					}
				}
			}
		}
		if item.Action != "conflict" && item.Action != "invalid" {
			pending[p.ID] = p
			if pendingTopics[p.ID] == nil {
				pendingTopics[p.ID] = map[string]bool{}
			}
			if exists {
				rows, e := tx.QueryContext(ctx, "SELECT topic_key FROM managed_knowledge_topics WHERE external_id=$1", p.ID)
				if e != nil {
					return nil, e
				}
				for rows.Next() {
					var t string
					if e = rows.Scan(&t); e != nil {
						rows.Close()
						return nil, e
					}
					pendingTopics[p.ID][t] = true
				}
				e = rows.Err()
				rows.Close()
				if e != nil {
					return nil, e
				}
			}
			for _, t := range item.TopicKeys {
				pendingTopics[p.ID][t] = true
			}
		}
		out = append(out, item)
	}
	return out, nil
}
func (s *Store) PreviewManagedImport(ctx context.Context, a knowledgeadmin.Access, d knowledgeadmin.SourceDocument, inputSHA string) (out knowledgeadmin.Preview, e error) {
	if !managedSHA.MatchString(inputSHA) || len(d.KnowledgePoints) > 100 || len(d.KnowledgePoints) == 0 {
		return out, knowledgeadmin.ErrInvalid
	}
	digest := knowledgeFingerprint(struct {
		Doc    knowledgeadmin.SourceDocument
		RawSHA string
	}{d, inputSHA})
	e = s.knowledgeTx(ctx, a, true, true, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		var priorSHA string
		var b []byte
		e := tx.QueryRowContext(ctx, "SELECT input_sha256,preview FROM managed_knowledge_imports WHERE owner_user_id=$1 AND action='preview' AND key=$2", u.ID, a.IdempotencyKey).Scan(&priorSHA, &b)
		if e == nil {
			if priorSHA != digest {
				return knowledgeadmin.ErrIdempotency
			}
			return json.Unmarshal(b, &out)
		}
		if !errors.Is(e, sql.ErrNoRows) {
			return e
		}
		out = knowledgeadmin.Preview{InputSHA256: inputSHA}
		out.Items, e = previewManaged(ctx, tx, d)
		if e != nil {
			return e
		}
		for _, item := range out.Items {
			countPreview(&out, item)
		}
		e = tx.QueryRowContext(ctx, "SELECT gen_random_uuid()::text,gen_random_uuid()::text").Scan(&out.ImportID, &out.PreviewToken)
		if e != nil {
			return e
		}
		_, e = tx.ExecContext(ctx, `INSERT INTO managed_knowledge_imports(import_id,owner_user_id,action,key,resource,input_sha256,source_body,preview,preview_token,state) VALUES($1,$2,'preview',$3,$8,$4,$5,$6,$7,'preview')`, out.ImportID, u.ID, a.IdempotencyKey, digest, knowledgeJSON(d), knowledgeJSON(out), out.PreviewToken, out.ImportID)
		return e
	})
	return
}
func (s *Store) ReadManagedImport(ctx context.Context, a knowledgeadmin.Access, id string) (out knowledgeadmin.Preview, receipt *knowledgeadmin.Receipt, e error) {
	e = s.knowledgeTx(ctx, a, false, true, func(ctx context.Context, tx *sql.Tx, _ auth.User) error {
		var p, r []byte
		e := tx.QueryRowContext(ctx, "SELECT preview,receipt FROM managed_knowledge_imports WHERE import_id=$1 AND action='preview'", id).Scan(&p, &r)
		if e != nil {
			return e
		}
		if e = json.Unmarshal(p, &out); e != nil {
			return e
		}
		if len(r) > 0 {
			receipt = &knowledgeadmin.Receipt{}
			return json.Unmarshal(r, receipt)
		}
		return nil
	})
	return
}
func (s *Store) ApplyManagedImport(ctx context.Context, a knowledgeadmin.Access, id string, in knowledgeadmin.ApplyInput) (out knowledgeadmin.Receipt, e error) {
	selected := append([]int{}, in.SelectedIndexes...)
	sort.Ints(selected)
	for n, i := range selected {
		if i < 0 || n > 0 && selected[n-1] == i {
			return out, knowledgeadmin.ErrInvalid
		}
	}
	in.SelectedIndexes = selected
	digest := knowledgeFingerprint(struct {
		ID    string
		Input knowledgeadmin.ApplyInput
	}{id, in})
	e = s.knowledgeTx(ctx, a, true, true, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		if ok, e := knowledgeReplay(ctx, tx, u, a, "apply", id, digest, &out); e != nil || ok {
			return e
		}
		var docJSON, pJSON, rJSON []byte
		var token, rawState string
		e := tx.QueryRowContext(ctx, "SELECT source_body,preview,preview_token::text,state,receipt FROM managed_knowledge_imports WHERE import_id=$1 AND action='preview' FOR UPDATE", id).Scan(&docJSON, &pJSON, &token, &rawState, &rJSON)
		if e != nil {
			return e
		}
		if token != in.PreviewToken {
			return knowledgeadmin.ErrStale
		}
		if rawState == "applied" {
			return knowledgeadmin.ErrConflict
		}
		var d knowledgeadmin.SourceDocument
		var p knowledgeadmin.Preview
		if json.Unmarshal(docJSON, &d) != nil || json.Unmarshal(pJSON, &p) != nil {
			return knowledgeadmin.ErrNotConfigured
		}
		out = knowledgeadmin.Receipt{OperationID: id, Items: []knowledgeadmin.ItemReceipt{}}
		allowed := map[int]bool{}
		for _, v := range p.Items {
			if v.Action == "create" || v.Action == "link" {
				allowed[v.Index] = true
			}
		}
		for _, n := range selected {
			if n >= len(d.KnowledgePoints) || !allowed[n] {
				return knowledgeadmin.ErrInvalid
			}
		}
		for _, n := range selected {
			point := d.KnowledgePoints[n]
			item := knowledgeadmin.ItemReceipt{Index: n, ExternalID: point.ID, KnowledgeID: knowledgeadmin.KnowledgeID(point.ID), TopicKeys: []string{}}
			k, exists, e := existingExternal(ctx, tx, point.ID)
			if e != nil {
				return e
			}
			if exists && k.Deleted {
				item.Action = "skip"
				out.Counts.SkippedItems++
				out.Items = append(out.Items, item)
				continue
			}
			if !exists {
				i := sourceCurrent(d, point)
				if e = knowledgeadmin.ValidateCurrent(i); e != nil {
					return e
				}
				k, e = createManaged(ctx, tx, u, i, in.Publish)
				if e != nil {
					return e
				}
				item.Action = "create"
				item.TopicKeys = knowledgeadmin.TopicKeys(point)
				out.Counts.CreatedKnowledge++
				out.Counts.LinkedTopics += len(item.TopicKeys)
				if e = managedEvent(ctx, tx, u, "import", nil, k, []string{}, item.TopicKeys, map[string]any{"importId": id, "sourceId": d.Source.SourceID}); e != nil {
					return e
				}
			} else {
				known, e := knownSourceCore(ctx, tx, k, point)
				if e != nil {
					return e
				}
				if !known {
					return knowledgeadmin.ErrConflict
				}
				before, e := managedTopics(ctx, tx, k.ID)
				if e != nil {
					return e
				}
				for _, t := range knowledgeadmin.TopicKeys(point) {
					r, e := tx.ExecContext(ctx, `INSERT INTO managed_knowledge_topics(internal_id,external_id,topic_key) VALUES($1,$2,$3) ON CONFLICT(external_id,topic_key) DO NOTHING`, k.ID, k.ExternalID, t)
					if e != nil {
						return e
					}
					count, e := r.RowsAffected()
					if e != nil {
						return e
					}
					if count == 1 {
						item.TopicKeys = append(item.TopicKeys, t)
					}
				}
				if len(item.TopicKeys) > 0 {
					after, e := managedTopics(ctx, tx, k.ID)
					if e != nil {
						return e
					}
					input := knowledgeadmin.CurrentInput{ExternalID: k.ExternalID, Point: k.Point, Sources: k.Sources}

					sha, e := knowledgeadmin.CurrentSHA(input, after)
					if e != nil {
						return e
					}
					if _, e = tx.ExecContext(ctx, `UPDATE managed_knowledge SET content_sha256=$2,updated_by=$3,updated_at=clock_timestamp(),edit_token=gen_random_uuid() WHERE internal_id=$1`, k.ID, sha, u.ID); e != nil {
						return e
					}
					next, e := readManaged(ctx, tx, k.ID, false)
					if e != nil {
						return e
					}
					if e = managedEvent(ctx, tx, u, "link", &k, next, before, after, map[string]any{"importId": id}); e != nil {
						return e
					}
					item.Action = "link"
					out.Counts.LinkedTopics += len(item.TopicKeys)
				} else {
					item.Action = "skip"
					out.Counts.SkippedItems++
				}
			}
			if e = recordManagedSource(ctx, tx, k.ID, d, point, p.InputSHA256); e != nil {
				return e
			}
			out.Items = append(out.Items, item)
		}
		// Preview skips and conflicts remain visible even when only valid new items were selected.
		chosen := map[int]bool{}
		for _, n := range selected {
			chosen[n] = true
		}
		for _, v := range p.Items {
			if chosen[v.Index] {
				continue
			}
			switch v.Action {
			case "skip":
				out.Counts.SkippedItems++
				point := d.KnowledgePoints[v.Index]
				k, exists, e := existingExternal(ctx, tx, point.ID)
				if e != nil {
					return e
				}
				if exists && !k.Deleted {
					known, e := knownSourceCore(ctx, tx, k, point)
					if e != nil {
						return e
					}
					if known {
						if e = recordManagedSource(ctx, tx, k.ID, d, point, p.InputSHA256); e != nil {
							return e
						}
					}
				}

			case "conflict":
				out.Counts.Conflicts++
			case "invalid":
				out.Counts.InvalidItems++
			}
			out.Items = append(out.Items, knowledgeadmin.ItemReceipt{Index: v.Index, ExternalID: v.ExternalID, KnowledgeID: knowledgeadmin.KnowledgeID(v.ExternalID), Action: v.Action, TopicKeys: v.TopicKeys, ErrorCode: v.ErrorCode})
		}
		sort.Slice(out.Items, func(i, j int) bool { return out.Items[i].Index < out.Items[j].Index })
		if _, e = tx.ExecContext(ctx, "UPDATE managed_knowledge_imports SET state='applied',receipt=$2 WHERE import_id=$1", id, knowledgeJSON(out)); e != nil {
			return e
		}
		return knowledgeReceipt(ctx, tx, u, a, "apply", id, digest, out)
	})
	return
}

func recordManagedSource(ctx context.Context, tx *sql.Tx, id string, d knowledgeadmin.SourceDocument, p knowledgeadmin.SourcePoint, rawSHA string) error {
	core, e := knowledgeadmin.SourceCoreSHA(p)
	if e != nil {
		return e
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO managed_knowledge_sources(internal_id,source_id,source_core_sha,source_file_sha,source_profile,source_point) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT DO NOTHING`, id, d.Source.SourceID, core, rawSHA, knowledgeJSON(d.Source), knowledgeJSON(p))
	return e
}
