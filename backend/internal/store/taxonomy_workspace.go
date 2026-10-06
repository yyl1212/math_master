package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"sort"
	"time"
)

func topicKnowledgeBinding(d publication.DraftView, id string) (string, bool) {
	for _, k := range d.Package.Knowledge {
		if k.ID == id {
			links := []publication.SourceLink{}
			for _, link := range d.SourceMap {
				if link.Knowledge.ID == id && link.Knowledge.Version == k.Version {
					links = append(links, link)
				}
			}
			return content.Digest(struct {
				Knowledge content.Knowledge
				Sources   []publication.SourceLink
			}{k, links}), true
		}
	}
	return "", false
}
func topicDraftViewTx(ctx context.Context, tx *sql.Tx, d publication.DraftView) (taxonomy.DraftTopicView, error) {
	out := taxonomy.DraftTopicView{DraftID: d.ID, DraftRevision: d.Revision, Members: []taxonomy.AssignmentInput{}}
	rows, e := tx.QueryContext(ctx, "SELECT knowledge_id,draft_revision,assignment_revision,taxonomy_version_id,knowledge_sha,body FROM taxonomy_assignment_drafts WHERE workspace_id=$1 ORDER BY knowledge_id", d.ID)
	if e != nil {
		return out, e
	}
	defer rows.Close()
	valid := 0
	for rows.Next() {
		var kid, version, bind string
		var rev, ar int64
		var raw []byte
		if e = rows.Scan(&kid, &rev, &ar, &version, &bind, &raw); e != nil {
			return out, e
		}
		var member taxonomy.AssignmentInput
		if e = json.Unmarshal(raw, &member); e != nil {
			return out, e
		}
		if ar > out.AssignmentRevision {
			out.AssignmentRevision = ar
		}
		expected, ok := topicKnowledgeBinding(d, kid)
		if !ok {
			continue
		}
		out.Members = append(out.Members, member)
		if out.TaxonomyVersionID == "" {
			out.TaxonomyVersionID = version
		} else if out.TaxonomyVersionID != version {
			return out, taxonomy.ErrConflict
		}
		if ok && rev == d.Revision && bind == expected {
			valid++
		}
	}
	if e = rows.Err(); e != nil {
		return out, e
	}
	rows.Close()
	if out.TaxonomyVersionID == "" {
		e = tx.QueryRowContext(ctx, "SELECT id FROM taxonomy_versions ORDER BY created_at DESC,id DESC LIMIT 1").Scan(&out.TaxonomyVersionID)
		if e != nil && !errors.Is(e, sql.ErrNoRows) {
			return out, e
		}
	}
	out.ReadyToSubmit = valid == len(d.Package.Knowledge) && valid > 0
	out.Digest, e = taxonomy.Digest(struct {
		Version string
		Members []taxonomy.AssignmentInput
	}{out.TaxonomyVersionID, out.Members})
	return out, e
}
func validateTopicSourceTx(ctx context.Context, tx *sql.Tx, version string, in taxonomy.AssignmentInput, d publication.DraftView) error {
	var raw []byte
	if e := tx.QueryRowContext(ctx, "SELECT b.body FROM taxonomy_versions v JOIN taxonomy_source_batches b ON b.snapshot_id=v.snapshot_id WHERE v.id=$1", version).Scan(&raw); e != nil {
		return taxonomy.ErrInvalid
	}
	var batch taxonomy.CapturedBatch
	if json.Unmarshal(raw, &batch) != nil || batch.Manifest.SnapshotID != in.SourceBatchSHA {
		return taxonomy.ErrInvalid
	}
	allowed := map[taxonomy.SourceRecordRef]bool{}
	for _, r := range batch.SourceRecordIndex {
		allowed[r] = true
	}
	for _, r := range in.SourceRefs {
		if !allowed[r] {
			return taxonomy.ErrInvalid
		}
		matched := false
		for _, link := range d.SourceMap {
			if link.Knowledge == in.Knowledge && link.RelativePath == r.Path && link.LegacyID == r.RecordID && link.SHA256 == r.SHA256 && link.BatchSHA256 == in.SourceBatchSHA {
				matched = true
			}
		}
		if !matched {
			return taxonomy.ErrInvalid
		}
	}
	for _, topic := range in.TopicIDs {
		var valid bool
		if e := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM taxonomy_nodes WHERE taxonomy_version_id=$1 AND id=$2 AND kind='primary' AND level=3)", version, topic).Scan(&valid); e != nil {
			return e
		}
		if !valid {
			return taxonomy.ErrInvalid
		}
	}
	return nil
}
func (s *Store) ReadDraftTopics(ctx context.Context, a publication.Access, id string) (taxonomy.DraftTopicView, error) {
	var out taxonomy.DraftTopicView
	e := s.workflowReadTx(ctx, a, publication.ReadDraftAction, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		if e := taxonomyConfigured(ctx, tx); e != nil {
			return e
		}
		d, _, e := s.readWorkflowDraft(ctx, tx, u, id, false)
		if e != nil {
			return e
		}
		out, e = topicDraftViewTx(ctx, tx, d)
		if e == nil && out.TaxonomyVersionID == "" {
			return taxonomy.ErrNotConfigured
		}
		return e
	})
	return out, e
}
func (s *Store) SaveDraftTopics(ctx context.Context, a publication.Access, id string, in taxonomy.DraftTopicInput) (taxonomy.DraftTopicView, error) {
	var out taxonomy.DraftTopicView
	raw, e := json.Marshal(in)
	if e != nil || len(raw) > 8192 || !publication.ValidID(id) || !taxonomy.ValidSHA(in.TaxonomyVersionID) || taxonomy.ValidateAssignment(in.Member) != nil {
		return out, taxonomy.ErrInvalid
	}
	inputSHA, _ := taxonomy.Digest(in)
	e = s.workflowTx(ctx, a, publication.SaveDraftAction, nil, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		if e := taxonomyConfigured(ctx, tx); e != nil {
			return e
		}
		d, _, e := s.readWorkflowDraft(ctx, tx, u, id, true)
		if e != nil {
			return e
		}
		var priorSHA string
		var receipt []byte
		e = tx.QueryRowContext(ctx, "SELECT input_sha,receipt FROM taxonomy_idempotency WHERE actor_user_id=$1 AND action='saveDraftTopics' AND target=$2 AND key=$3", u.ID, id, a.IdempotencyKey).Scan(&priorSHA, &receipt)
		if e == nil {
			if priorSHA != inputSHA {
				return publication.ErrIdempotencyConflict
			}
			if json.Unmarshal(receipt, &out) != nil {
				return taxonomy.ErrInvalid
			}
			return nil
		}
		if !errors.Is(e, sql.ErrNoRows) {
			return e
		}
		current, e := topicDraftViewTx(ctx, tx, d)
		if e != nil {
			return e
		}
		if d.Status != "editing" || d.Revision != in.ExpectedDraftRevision || current.AssignmentRevision != in.ExpectedAssignmentRevision || (len(current.Members) > 0 && current.TaxonomyVersionID != in.TaxonomyVersionID) {
			return publication.ErrDraftConflict
		}
		bind, ok := topicKnowledgeBinding(d, in.Member.Knowledge.ID)
		if !ok {
			return taxonomy.ErrInvalid
		}
		found := false
		for _, k := range d.Package.Knowledge {
			if k.ID == in.Member.Knowledge.ID && k.Version == in.Member.Knowledge.Version {
				found = true
			}
		}
		if !found {
			return taxonomy.ErrInvalid
		}
		if e = validateTopicSourceTx(ctx, tx, in.TaxonomyVersionID, in.Member, d); e != nil {
			return e
		}
		member, digest, e := taxonomy.CanonicalAssignment(in.Member)
		if e != nil {
			return e
		}
		_, e = tx.ExecContext(ctx, `INSERT INTO taxonomy_assignment_drafts(workspace_id,knowledge_id,draft_revision,assignment_revision,taxonomy_version_id,knowledge_sha,body,digest,updated_at)
   VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT(workspace_id,knowledge_id) DO UPDATE SET draft_revision=excluded.draft_revision,assignment_revision=excluded.assignment_revision,taxonomy_version_id=excluded.taxonomy_version_id,knowledge_sha=excluded.knowledge_sha,body=excluded.body,digest=excluded.digest,updated_at=excluded.updated_at`, id, in.Member.Knowledge.ID, d.Revision, current.AssignmentRevision+1, in.TaxonomyVersionID, bind, string(member), digest, now)
		if e != nil {
			return e
		}
		out, e = topicDraftViewTx(ctx, tx, d)
		if e != nil {
			return e
		}
		b, e := json.Marshal(out)
		if e != nil {
			return e
		}
		_, e = tx.ExecContext(ctx, "INSERT INTO taxonomy_idempotency(actor_user_id,action,target,key,input_sha,receipt) VALUES($1,'saveDraftTopics',$2,$3,$4,$5)", u.ID, id, a.IdempotencyKey, inputSHA, string(b))
		return e
	})
	return out, e
}
func refreshDraftTopicsTx(ctx context.Context, tx *sql.Tx, before, after publication.DraftView) error {
	if e := taxonomyConfigured(ctx, tx); errors.Is(e, taxonomy.ErrNotConfigured) {
		return nil
	} else if e != nil {
		return e
	}
	for _, k := range after.Package.Knowledge {
		a, ok := topicKnowledgeBinding(before, k.ID)
		b, found := topicKnowledgeBinding(after, k.ID)
		if ok && found && a == b {
			if _, e := tx.ExecContext(ctx, "UPDATE taxonomy_assignment_drafts SET draft_revision=$2 WHERE workspace_id=$1 AND knowledge_id=$3 AND knowledge_sha=$4", after.ID, after.Revision, k.ID, b); e != nil {
				return e
			}
		}
	}
	return nil
}
func freezeDraftTopicsTx(ctx context.Context, tx *sql.Tx, draftID, submissionID string, revision int64, d publication.DraftView) error {
	if e := taxonomyConfigured(ctx, tx); errors.Is(e, taxonomy.ErrNotConfigured) {
		return nil
	} else if e != nil {
		return e
	}
	view, e := topicDraftViewTx(ctx, tx, d)
	if e != nil {
		return e
	}
	if len(view.Members) == 0 {
		var mode string
		if e = tx.QueryRowContext(ctx, "SELECT experience_mode FROM topic_learning_state WHERE singleton").Scan(&mode); e != nil {
			return e
		}
		if mode == "topics" {
			return publication.ErrContentNotReady
		}
		return nil
	}
	if !view.ReadyToSubmit || d.Revision != revision {
		return publication.ErrContentNotReady
	}
	sort.Slice(view.Members, func(i, j int) bool { return view.Members[i].Knowledge.ID < view.Members[j].Knowledge.ID })
	raw, e := json.Marshal(view)
	if e != nil || len(raw) > 2<<20 {
		return taxonomy.ErrLimit
	}
	_, e = tx.ExecContext(ctx, "INSERT INTO taxonomy_submission_assignments(submission_id,taxonomy_version_id,body,digest) VALUES($1,$2,$3,$4)", submissionID, view.TaxonomyVersionID, string(raw), view.Digest)
	return e
}
