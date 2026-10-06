package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"reflect"
	"sort"
	"time"
)

func rejectIndependentActivationTx(ctx context.Context, tx *sql.Tx) error {
	var exists bool
	if e := tx.QueryRowContext(ctx, "SELECT to_regclass('public.taxonomy_heads') IS NOT NULL").Scan(&exists); e != nil {
		return e
	}
	if !exists {
		return nil
	}
	if e := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM taxonomy_heads)").Scan(&exists); e != nil {
		return e
	}
	if exists {
		return publication.ErrPublicationStale
	}
	return nil
}
func topicCurrentPairTx(ctx context.Context, tx *sql.Tx, version string) (taxonomy.PairRef, error) {
	out := taxonomy.PairRef{TaxonomyVersionID: version}
	var e error
	out.KnowledgeHead, e = workflowHead(ctx, tx)
	if e != nil {
		return out, e
	}
	var id string
	var kh sql.NullString
	e = tx.QueryRowContext(ctx, "SELECT r.id::text,r.knowledge_publication_id FROM taxonomy_heads h JOIN taxonomy_releases r ON r.id=h.release_id AND r.status='published'").Scan(&id, &kh)
	if errors.Is(e, sql.ErrNoRows) {
		return out, nil
	}
	if e != nil {
		return out, e
	}
	out.TaxonomyHead = &id
	if kh.Valid != (out.KnowledgeHead != nil) || kh.Valid && kh.String != *out.KnowledgeHead {
		return out, publication.ErrPublicationStale
	}
	return out, nil
}
func validTopicPair(p taxonomy.PairRef) bool {
	return taxonomy.ValidSHA(p.TaxonomyVersionID) && validWorkflowHead(p.KnowledgeHead) && validWorkflowHead(p.TaxonomyHead)
}
func topicReplay[T any](ctx context.Context, tx *sql.Tx, u auth.User, a publication.Access, action, target string, in any) (T, bool, error) {
	var out T
	sha, e := taxonomy.Digest(in)
	if e != nil {
		return out, false, e
	}
	var prior string
	var raw []byte
	e = tx.QueryRowContext(ctx, "SELECT input_sha,receipt FROM taxonomy_idempotency WHERE actor_user_id=$1 AND action=$2 AND target=$3 AND key=$4", u.ID, action, target, a.IdempotencyKey).Scan(&prior, &raw)
	if errors.Is(e, sql.ErrNoRows) {
		return out, false, nil
	}
	if e != nil {
		return out, false, e
	}
	if prior != sha {
		return out, false, publication.ErrIdempotencyConflict
	}
	e = json.Unmarshal(raw, &out)
	return out, true, e
}
func topicRemember(ctx context.Context, tx *sql.Tx, u auth.User, a publication.Access, action, target string, in, out any) error {
	sha, e := taxonomy.Digest(in)
	if e != nil {
		return e
	}
	raw, e := json.Marshal(out)
	if e != nil || len(raw) > 2<<20 {
		return taxonomy.ErrLimit
	}
	_, e = tx.ExecContext(ctx, "INSERT INTO taxonomy_idempotency(actor_user_id,action,target,key,input_sha,receipt) VALUES($1,$2,$3,$4,$5,$6)", u.ID, action, target, a.IdempotencyKey, sha, string(raw))
	return e
}
func readTopicReleaseTx(ctx context.Context, tx *sql.Tx, id string) (taxonomy.ReleaseDocument, error) {
	var out taxonomy.ReleaseDocument
	var raw []byte
	var status, manifestSHA, assignmentsSHA, version string
	var kh, bkh, bth sql.NullString
	e := tx.QueryRowContext(ctx, "SELECT body,status,manifest_sha,assignments_sha,taxonomy_version_id,knowledge_publication_id,base_knowledge_head,base_taxonomy_head::text FROM taxonomy_releases WHERE id=$1", id).Scan(&raw, &status, &manifestSHA, &assignmentsSHA, &version, &kh, &bkh, &bth)
	if e != nil {
		return out, workflowRowError(e)
	}
	if len(raw) > 32<<20 || json.Unmarshal(raw, &out) != nil {
		return out, taxonomy.ErrInvalid
	}
	sha, e := taxonomy.ReleaseDocumentDigest(out)
	if e != nil || sha != manifestSHA || out.View.ID != id || out.View.ManifestSHA != sha || out.View.AssignmentsSHA != assignmentsSHA || out.View.Pair.TaxonomyVersionID != version {
		return out, taxonomy.ErrInvalid
	}
	nullable := func(n sql.NullString, p *string) bool { return n.Valid == (p != nil) && (!n.Valid || n.String == *p) }
	if !nullable(kh, out.View.KnowledgePublicationID) || !nullable(bkh, out.View.Pair.KnowledgeHead) || !nullable(bth, out.View.Pair.TaxonomyHead) {
		return out, taxonomy.ErrInvalid
	}
	canonical, digest, e := taxonomy.CanonicalReleasedAssignments(out.Assignments)
	if e != nil || digest != assignmentsSHA || !reflect.DeepEqual(canonical, out.Assignments) {
		return out, taxonomy.ErrInvalid
	}
	out.View.Status = status
	return out, nil
}
func topicKnowledgeSetTx(ctx context.Context, tx *sql.Tx, head *string) (taxonomy.KnowledgeSet, error) {
	out := taxonomy.KnowledgeSet{}
	if head == nil {
		return out, nil
	}
	rows, e := tx.QueryContext(ctx, "SELECT k.id,k.version,k.sha256 FROM publication_members m JOIN knowledge_versions k ON k.id=m.id AND k.version=m.version WHERE m.snapshot_id=$1 AND m.kind='knowledge' AND m.availability='active'", *head)
	if e != nil {
		return out, e
	}
	defer rows.Close()
	for rows.Next() {
		var k taxonomy.KnowledgeRef
		if e = rows.Scan(&k.ID, &k.Version, &k.SHA256); e != nil {
			return out, e
		}
		if _, ok := out[k.ID]; ok {
			return out, taxonomy.ErrInvalid
		}
		out[k.ID] = k
		if len(out) > 1000 {
			return out, taxonomy.ErrLimit
		}
	}
	return out, rows.Err()
}
func topicReleaseEvidenceTx(ctx context.Context, tx *sql.Tx, d taxonomy.ReleaseDocument, current bool) error {
	// 一次性核验冻结成员、同一审核决定和当前资格，避免逐知识点的数据库往返。
	raw, e := json.Marshal(d.Assignments)
	if e != nil {
		return e
	}
	var n int
	const query = `SELECT count(*) FROM jsonb_array_elements($1::jsonb) a
 JOIN content_submission_members sm ON sm.submission_id=(a->'evidence'->>'submissionId')::uuid AND sm.kind='knowledge' AND sm.id=a->'knowledge'->>'id' AND sm.version=(a->'knowledge'->>'version')::integer AND sm.sha256=a->'knowledge'->>'sha256'
 JOIN taxonomy_submission_assignments frozen ON frozen.submission_id=sm.submission_id AND frozen.taxonomy_version_id=a->'evidence'->>'taxonomyVersionId' AND frozen.digest=a->'evidence'->>'assignmentDigest'
 JOIN taxonomy_review_bindings binding ON binding.submission_id=frozen.submission_id AND binding.digest=frozen.digest AND binding.decision_id=(a->'evidence'->>'decisionId')::uuid
 JOIN content_review_decisions decision ON decision.id=binding.decision_id AND decision.decision='approve'
 JOIN content_submissions sub ON sub.id=frozen.submission_id AND sub.sealed AND sub.status='approved' AND sub.frozen_digest=decision.frozen_digest
 WHERE decision.checks @> '{"mathematics":true,"explanations":true,"relationships":true,"sources":true,"illustrations":true}'
 AND EXISTS(SELECT 1 FROM jsonb_array_elements(frozen.body->'members') m WHERE m->'knowledge'=jsonb_build_object('id',a->'knowledge'->>'id','version',(a->'knowledge'->>'version')::integer) AND m->'topicIds'=a->'topicIds' AND m->'sourceRefs'=a->'sourceRefs' AND m->>'sourceBatchSHA'=a->>'sourceBatchSHA')
 AND (NOT $2 OR a->>'inheritedFrom' IS NOT NULL OR EXISTS(SELECT 1 FROM auth_user_roles r WHERE r.user_id=decision.reviewer_user_id AND r.role='reviewer'))
 AND (NOT $2 OR a->>'inheritedFrom' IS NOT NULL OR NOT EXISTS(SELECT 1 FROM content_submission_authors au WHERE au.submission_id=sub.id AND au.user_id=decision.reviewer_user_id) OR EXISTS(SELECT 1 FROM auth_user_roles r WHERE r.user_id=decision.reviewer_user_id AND r.role='admin'))`
	if e = tx.QueryRowContext(ctx, query, string(raw), current).Scan(&n); e != nil {
		return e
	}
	if n != len(d.Assignments) {
		return publication.ErrReviewRequired
	}
	inherited := 0
	for _, a := range d.Assignments {
		if a.InheritedFrom != nil {
			if !sameWorkflowHead(a.InheritedFrom, d.View.Pair.TaxonomyHead) {
				return publication.ErrReviewRequired
			}
			inherited++
		}
	}
	if inherited > 0 {
		e = tx.QueryRowContext(ctx, `SELECT count(*) FROM jsonb_array_elements($1::jsonb) a JOIN taxonomy_releases previous ON previous.id=$2 AND previous.status='published' WHERE a->>'inheritedFrom'=$2::text AND EXISTS(SELECT 1 FROM jsonb_array_elements(previous.body->'assignments') old WHERE old-'inheritedFrom'=a-'inheritedFrom')`, string(raw), *d.View.Pair.TaxonomyHead).Scan(&n)
		if e != nil {
			return e
		}
		if n != inherited {
			return publication.ErrReviewRequired
		}
	}
	set, e := topicKnowledgeSetTx(ctx, tx, d.View.KnowledgePublicationID)
	if e != nil {
		return e
	}
	for _, a := range d.Assignments {
		if set[a.Knowledge.ID] != a.Knowledge {
			return taxonomy.ErrInvalid
		}
	}
	return nil
}
func insertTopicReleaseTx(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time, d taxonomy.ReleaseDocument) (taxonomy.ReleaseDocument, error) {
	var e error
	d.View.ID, e = workflowID()
	if e != nil {
		return d, e
	}
	d.View.Status = "draft"
	d.View.CreatedAt = now.UTC().Format(time.RFC3339)
	d.Assignments, d.View.AssignmentsSHA, e = taxonomy.CanonicalReleasedAssignments(d.Assignments)
	if e != nil {
		return d, e
	}
	d.View.ManifestSHA, e = taxonomy.ReleaseDocumentDigest(d)
	if e != nil {
		return d, e
	}
	raw, e := json.Marshal(d)
	view, e2 := json.Marshal(d.View)
	if e != nil {
		return d, e
	}
	if e2 != nil {
		return d, e2
	}
	if len(raw) > 32<<20 || len(view) > 2<<20 {
		return d, taxonomy.ErrLimit
	}
	_, e = tx.ExecContext(ctx, "INSERT INTO taxonomy_releases(id,taxonomy_version_id,knowledge_publication_id,base_knowledge_head,base_taxonomy_head,status,manifest_sha,assignments_sha,body,creator_user_id,created_at) VALUES($1,$2,$3,$4,$5,'draft',$6,$7,$8,$9,$10)", d.View.ID, d.View.Pair.TaxonomyVersionID, d.View.KnowledgePublicationID, d.View.Pair.KnowledgeHead, d.View.Pair.TaxonomyHead, d.View.ManifestSHA, d.View.AssignmentsSHA, string(raw), u.ID, now)
	if e != nil {
		return d, e
	}
	entries := []map[string]any{}
	for _, a := range d.Assignments {
		for _, id := range a.TopicIDs {
			entries = append(entries, map[string]any{"id": a.Knowledge.ID, "version": a.Knowledge.Version, "sha": a.Knowledge.SHA256, "topic": id, "submission": a.Evidence.SubmissionID})
		}
	}
	data, e := json.Marshal(entries)
	if e != nil {
		return d, e
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO taxonomy_release_assignments(release_id,taxonomy_version_id,knowledge_id,knowledge_version,knowledge_sha,topic_id,evidence_submission_id) SELECT $1,$2,x.id,x.version,x.sha,x.topic,x.submission::uuid FROM jsonb_to_recordset($3::jsonb) x(id text,version integer,sha text,topic text,submission text)`, d.View.ID, d.View.Pair.TaxonomyVersionID, string(data))
	return d, e
}
func activateTopicHeadTx(ctx context.Context, tx *sql.Tx, id string) error {
	if _, e := tx.ExecContext(ctx, "UPDATE taxonomy_releases SET status='published' WHERE id=$1 AND status='draft'", id); e != nil {
		return e
	}
	_, e := tx.ExecContext(ctx, "INSERT INTO taxonomy_heads VALUES(true,$1) ON CONFLICT(singleton) DO UPDATE SET release_id=excluded.release_id", id)
	return e
}
func (s *Store) ReadTopicRelease(ctx context.Context, a publication.Access, id string) (taxonomy.ReleaseView, error) {
	var out taxonomy.ReleaseView
	if !publication.ValidID(id) {
		return out, taxonomy.ErrInvalid
	}
	e := s.workflowReadTx(ctx, a, publication.ReadPublicationAction, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		if e := taxonomyConfigured(ctx, tx); e != nil {
			return e
		}
		d, e := readTopicReleaseTx(ctx, tx, id)
		out = d.View
		return e
	})
	return out, e
}
func (s *Store) PrepareTopicRelease(ctx context.Context, a publication.Access, in taxonomy.PrepareInput) (taxonomy.ReleaseView, error) {
	var out taxonomy.ReleaseView
	ctx, cancel := context.WithTimeout(ctx, workflowTimeout)
	defer cancel()
	if !validTopicPair(in.ExpectedPair) || !publication.ValidNote(in.Reason) || len(in.SubmissionIDs) > 20 {
		return out, taxonomy.ErrInvalid
	}
	seen := map[string]bool{}
	for _, id := range in.SubmissionIDs {
		if !publication.ValidID(id) || seen[id] {
			return out, taxonomy.ErrInvalid
		}
		seen[id] = true
	}
	related, e := s.workflowReleaseReviewers(ctx, a, append([]string{}, in.SubmissionIDs...), "")
	if e != nil {
		return out, e
	}
	e = s.workflowTx(ctx, a, publication.PrepareReleaseAction, related, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		if e := taxonomyConfigured(ctx, tx); e != nil {
			return e
		}
		prior, found, e := topicReplay[taxonomy.ReleaseView](ctx, tx, u, a, "prepareTopicRelease", "", in)
		if e != nil {
			return e
		}
		if found {
			out = prior
			return nil
		}
		pair, e := topicCurrentPairTx(ctx, tx, in.ExpectedPair.TaxonomyVersionID)
		if e != nil {
			return e
		}
		if !taxonomy.SamePair(pair, in.ExpectedPair) {
			return publication.ErrPublicationStale
		}
		var exists bool
		if e = tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM taxonomy_versions WHERE id=$1)", pair.TaxonomyVersionID).Scan(&exists); e != nil {
			return e
		}
		if !exists {
			return taxonomy.ErrInvalid
		}
		d := taxonomy.ReleaseDocument{View: taxonomy.ReleaseView{Pair: pair, KnowledgePublicationID: pair.KnowledgeHead}, Assignments: []taxonomy.ReleasedAssignment{}}
		old := []taxonomy.ReleasedAssignment{}
		if pair.TaxonomyHead != nil {
			base, e := readTopicReleaseTx(ctx, tx, *pair.TaxonomyHead)
			if e != nil {
				return e
			}
			old = base.Assignments
			for _, a := range old {
				a.InheritedFrom = pair.TaxonomyHead
				d.Assignments = append(d.Assignments, a)
			}
		}
		if pair.KnowledgeHead != nil {
			var sha sql.NullString
			if e = tx.QueryRowContext(ctx, "SELECT (SELECT sha256 FROM content_publication_manifests WHERE snapshot_id=$1)", *pair.KnowledgeHead).Scan(&sha); e != nil {
				return e
			}
			d.KnowledgeManifestSHA = sha.String
		}
		batches := []publication.ReviewedBatch{}
		chosen := map[string]taxonomy.ReleasedAssignment{}
		for _, id := range in.SubmissionIDs {
			b, e := s.workflowReviewedBatch(ctx, tx, u, id)
			if e != nil {
				return e
			}
			batches = append(batches, b)
			var raw []byte
			var digest, version string
			e = tx.QueryRowContext(ctx, "SELECT body,digest,taxonomy_version_id FROM taxonomy_submission_assignments WHERE submission_id=$1", id).Scan(&raw, &digest, &version)
			if errors.Is(e, sql.ErrNoRows) {
				return publication.ErrContentNotReady
			}
			if e != nil {
				return e
			}
			var frozen taxonomy.DraftTopicView
			if json.Unmarshal(raw, &frozen) != nil || !frozen.ReadyToSubmit || version != pair.TaxonomyVersionID {
				return taxonomy.ErrInvalid
			}
			for _, m := range frozen.Members {
				var ref taxonomy.KnowledgeRef
				for _, identity := range b.Members {
					if identity.Kind == "knowledge" && identity.ID == m.Knowledge.ID && identity.Version == m.Knowledge.Version {
						ref = taxonomy.KnowledgeRef{ID: identity.ID, Version: identity.Version, SHA256: identity.SHA256}
					}
				}
				if ref.ID == "" {
					return taxonomy.ErrInvalid
				}
				v := taxonomy.ReleasedAssignment{Knowledge: ref, TopicIDs: m.TopicIDs, SourceRefs: m.SourceRefs, SourceBatchSHA: m.SourceBatchSHA, Evidence: taxonomy.ReleaseEvidence{SubmissionID: id, DecisionID: b.Submission.Review.ID, AssignmentDigest: digest, TaxonomyVersionID: version}}
				if prior, ok := chosen[ref.ID]; ok && !reflect.DeepEqual(prior, v) {
					return taxonomy.ErrConflict
				}
				chosen[ref.ID] = v
			}
		}
		if len(batches) > 0 {
			base, e := s.loadWorkflowCandidate(ctx, tx, pair.KnowledgeHead)
			if e != nil {
				return e
			}
			candidate, e := publication.BuildCandidate(base, batches)
			if e != nil {
				return e
			}
			if e = workflowValidateCandidate(ctx, tx, candidate, true); e != nil {
				return e
			}
			identities := func(m publication.Manifest) []publication.MemberIdentity {
				items := []publication.MemberIdentity{}
				for _, a := range m.Members {
					items = append(items, a.Identity)
				}
				sort.Slice(items, func(i, j int) bool { return items[i].Kind+"/"+items[i].ID < items[j].Kind+"/"+items[j].ID })
				return items
			}
			if pair.KnowledgeHead == nil || !reflect.DeepEqual(identities(base.Manifest), identities(candidate.Manifest)) || !reflect.DeepEqual(base.Manifest.Bindings, candidate.Manifest.Bindings) {
				p, e := s.insertWorkflowPublication(ctx, tx, u, now, candidate, "draft")
				if e != nil {
					return e
				}
				d.View.KnowledgePublicationID = &p.ID
				d.KnowledgeManifestSHA = p.ManifestSHA
			}
		}
		set, e := topicKnowledgeSetTx(ctx, tx, d.View.KnowledgePublicationID)
		if e != nil {
			return e
		}
		kept := []taxonomy.ReleasedAssignment{}
		for _, v := range d.Assignments {
			if _, replaced := chosen[v.Knowledge.ID]; !replaced && set[v.Knowledge.ID] == v.Knowledge {
				kept = append(kept, v)
			}
		}
		for _, v := range chosen {
			kept = append(kept, v)
		}
		d.Assignments = kept
		d.Assignments, _, e = taxonomy.CanonicalReleasedAssignments(d.Assignments)
		if e != nil {
			return e
		}
		d.View.Diff = taxonomy.AssignmentDiff(old, d.Assignments)
		if e = topicReleaseEvidenceTx(ctx, tx, d, true); e != nil {
			return e
		}
		d, e = insertTopicReleaseTx(ctx, tx, u, now, d)
		if e != nil {
			return e
		}
		out = d.View
		if e = workflowEvent(ctx, tx, u, a, publication.PrepareReleaseAction, "taxonomy", out.ID, "", out.ManifestSHA, "", "draft", in.Reason, now); e != nil {
			return e
		}
		return topicRemember(ctx, tx, u, a, "prepareTopicRelease", "", in, out)
	})
	return out, e
}
func (s *Store) ActivateTopicRelease(ctx context.Context, a publication.Access, id string, in taxonomy.ActivateInput) (taxonomy.ReleaseView, error) {
	var out taxonomy.ReleaseView
	ctx, cancel := context.WithTimeout(ctx, workflowTimeout)
	defer cancel()
	if !publication.ValidID(id) || !validTopicPair(in.ExpectedPair) || !taxonomy.ValidSHA(in.ManifestSHA) || !publication.ValidNote(in.Reason) {
		return out, taxonomy.ErrInvalid
	}
	var d taxonomy.ReleaseDocument
	e := s.workflowReadTx(ctx, a, publication.ReadPublicationAction, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		var e error
		if e = taxonomyConfigured(ctx, tx); e != nil {
			return e
		}
		d, e = readTopicReleaseTx(ctx, tx, id)
		return e
	})
	if e != nil {
		return out, e
	}
	ids := []string{}
	for _, v := range d.Assignments {
		if v.InheritedFrom == nil {
			ids = append(ids, v.Evidence.SubmissionID)
		}
	}
	pubID := ""
	if d.View.KnowledgePublicationID != nil && !sameWorkflowHead(d.View.KnowledgePublicationID, d.View.Pair.KnowledgeHead) {
		pubID = *d.View.KnowledgePublicationID
	}
	related, e := s.workflowReleaseReviewers(ctx, a, ids, pubID)
	if e != nil {
		return out, e
	}
	e = s.workflowTx(ctx, a, publication.ActivateReleaseAction, related, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		if e := taxonomyConfigured(ctx, tx); e != nil {
			return e
		}
		prior, found, e := topicReplay[taxonomy.ReleaseView](ctx, tx, u, a, "activateTopicRelease", id, in)
		if e != nil {
			return e
		}
		if found {
			out = prior
			return nil
		}
		d, e := readTopicReleaseTx(ctx, tx, id)
		if e != nil {
			return e
		}
		pair, e := topicCurrentPairTx(ctx, tx, in.ExpectedPair.TaxonomyVersionID)
		if e != nil {
			return e
		}
		if d.View.Status != "draft" || d.View.ManifestSHA != in.ManifestSHA || !taxonomy.SamePair(d.View.Pair, pair) || !taxonomy.SamePair(pair, in.ExpectedPair) {
			return publication.ErrPublicationStale
		}
		if e = topicReleaseEvidenceTx(ctx, tx, d, true); e != nil {
			return e
		}
		if !sameWorkflowHead(d.View.KnowledgePublicationID, pair.KnowledgeHead) {
			if d.View.KnowledgePublicationID == nil {
				return taxonomy.ErrInvalid
			}
			if _, e = s.activateWorkflowReleaseTx(ctx, tx, *d.View.KnowledgePublicationID, publication.ActivateInput{ExpectedHead: pair.KnowledgeHead, ExpectedManifestSHA: d.KnowledgeManifestSHA, Reason: in.Reason}); e != nil {
				return e
			}
		}
		if e = activateTopicHeadTx(ctx, tx, id); e != nil {
			return e
		}
		out = d.View
		out.Status = "published"
		if e = workflowEvent(ctx, tx, u, a, publication.ActivateReleaseAction, "taxonomy", id, "", out.ManifestSHA, "draft", "published", in.Reason, now); e != nil {
			return e
		}
		return topicRemember(ctx, tx, u, a, "activateTopicRelease", id, in, out)
	})
	return out, e
}
func applyTopicWithdrawalTx(ctx context.Context, tx *sql.Tx, knowledgeHead string, removed []taxonomy.KnowledgeRef) error {
	if e := taxonomyConfigured(ctx, tx); errors.Is(e, taxonomy.ErrNotConfigured) {
		for _, table := range []string{"taxonomy_heads", "taxonomy_releases", "topic_learning_state"} {
			var exists bool
			if err := tx.QueryRowContext(ctx, "SELECT to_regclass($1) IS NOT NULL", "public."+table).Scan(&exists); err != nil {
				return err
			}
			if !exists {
				continue
			}
			query := map[string]string{"taxonomy_heads": "SELECT EXISTS(SELECT 1 FROM taxonomy_heads)", "taxonomy_releases": "SELECT EXISTS(SELECT 1 FROM taxonomy_releases WHERE status='published')", "topic_learning_state": "SELECT EXISTS(SELECT 1 FROM topic_learning_state WHERE experience_mode='topics' OR study_enabled)"}[table]
			if err := tx.QueryRowContext(ctx, query).Scan(&exists); err != nil {
				return err
			}
			if exists {
				return taxonomy.ErrNotConfigured
			}
		}
		return nil
	} else if e != nil {
		return e
	}
	var head string
	e := tx.QueryRowContext(ctx, "SELECT release_id::text FROM taxonomy_heads").Scan(&head)
	if errors.Is(e, sql.ErrNoRows) {
		return nil
	}
	if e != nil {
		return e
	}
	old, e := readTopicReleaseTx(ctx, tx, head)
	if e != nil {
		return e
	}
	p, e := readWorkflowPublication(ctx, tx, knowledgeHead)
	if e != nil {
		return e
	}
	if !sameWorkflowHead(old.View.KnowledgePublicationID, p.Manifest.BaseHead) {
		return publication.ErrPublicationStale
	}
	var u auth.User
	var now time.Time
	if e = tx.QueryRowContext(ctx, "SELECT creator_user_id::text,created_at FROM content_publication_manifests WHERE snapshot_id=$1", knowledgeHead).Scan(&u.ID, &now); e != nil {
		return e
	}
	next := &knowledgeHead
	d := taxonomy.ReleaseDocument{View: taxonomy.ReleaseView{Pair: taxonomy.PairRef{KnowledgeHead: old.View.KnowledgePublicationID, TaxonomyHead: &head, TaxonomyVersionID: old.View.Pair.TaxonomyVersionID}, KnowledgePublicationID: next}, KnowledgeManifestSHA: p.ManifestSHA, Assignments: []taxonomy.ReleasedAssignment{}}
	set, e := topicKnowledgeSetTx(ctx, tx, next)
	if e != nil {
		return e
	}
	gone := map[taxonomy.KnowledgeRef]bool{}
	for _, ref := range removed {
		gone[ref] = true
	}
	for _, v := range old.Assignments {
		if !gone[v.Knowledge] && set[v.Knowledge.ID] == v.Knowledge {
			v.InheritedFrom = &head
			d.Assignments = append(d.Assignments, v)
		}
	}
	d.View.Diff = taxonomy.AssignmentDiff(old.Assignments, d.Assignments)
	if e = topicReleaseEvidenceTx(ctx, tx, d, false); e != nil {
		return e
	}
	d, e = insertTopicReleaseTx(ctx, tx, u, now, d)
	if e != nil {
		return e
	}
	return activateTopicHeadTx(ctx, tx, d.View.ID)
}
func (s *Store) ListTopicReleases(ctx context.Context, a publication.Access, q taxonomy.Query) (taxonomy.ReleasePage, error) {
	out := taxonomy.ReleasePage{Items: []taxonomy.ReleaseView{}}
	if q.Limit < 0 || q.Limit > 100 || q.Offset < 0 || q.Offset > 100000 || q.Q != "" || q.Kind != "" || q.ParentID != "" || q.Level != 0 {
		return out, taxonomy.ErrInvalid
	}
	if q.Limit == 0 {
		q.Limit = 20
	}
	out.Limit = q.Limit
	out.Offset = q.Offset
	e := s.workflowReadTx(ctx, a, publication.ListPublicationsAction, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		if e := taxonomyConfigured(ctx, tx); e != nil {
			return e
		}
		var version string
		e := tx.QueryRowContext(ctx, "SELECT COALESCE((SELECT r.taxonomy_version_id FROM taxonomy_heads h JOIN taxonomy_releases r ON r.id=h.release_id AND r.status='published'),(SELECT id FROM taxonomy_versions ORDER BY created_at DESC,id DESC LIMIT 1))").Scan(&version)
		if e != nil {
			return taxonomy.ErrNotConfigured
		}
		out.Pair, e = topicCurrentPairTx(ctx, tx, version)
		if e != nil {
			return e
		}
		if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM taxonomy_releases").Scan(&out.Total); e != nil {
			return e
		}
		rows, e := tx.QueryContext(ctx, "SELECT id::text FROM taxonomy_releases ORDER BY created_at DESC,id DESC LIMIT $1 OFFSET $2", q.Limit, q.Offset)
		if e != nil {
			return e
		}
		ids := []string{}
		for rows.Next() {
			var id string
			if e = rows.Scan(&id); e != nil {
				rows.Close()
				return e
			}
			ids = append(ids, id)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		for _, id := range ids {
			d, e := readTopicReleaseTx(ctx, tx, id)
			if e != nil {
				return e
			}
			out.Items = append(out.Items, d.View)
			raw, e := json.Marshal(out)
			if e != nil {
				return e
			}
			if len(raw) > 2<<20 {
				out.Items = out.Items[:len(out.Items)-1]
				if len(out.Items) == 0 {
					return taxonomy.ErrLimit
				}
				out.Limit = len(out.Items)
				break
			}
		}
		return nil
	})
	return out, e
}
