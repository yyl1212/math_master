package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"strings"
	"time"
)

func questionDraftInputOf(d question.DraftView) question.DraftInput {
	return question.DraftInput{CatalogueVersion: d.CatalogueVersion, QuestionPackage: d.QuestionPackage, SourceMap: d.SourceMap}
}

// The client proof includes platform provenance and the saved revision, in addition to math and references.
func questionCheckDraft(ctx context.Context, tx *sql.Tx, d question.DraftView) (question.SealedPackage, question.ValidationReport, question.SourceResponsibility, error) {
	in := questionDraftInputOf(d)
	refs, err := questionReferences(ctx, tx, in)
	if err != nil {
		return question.SealedPackage{}, question.ValidationReport{}, question.SourceResponsibility{}, err
	}
	sealed, gate, err := question.ValidateAndSeal(ctx, in, refs)
	if err != nil && !errors.Is(err, question.ErrInvalid) && !errors.Is(err, question.ErrNotReady) && !errors.Is(err, question.ErrLimitExceeded) {
		return sealed, gate, question.SourceResponsibility{}, err
	}
	p := d.QuestionPackage
	if gate.ReadyToSubmit {
		p = sealed.Package
	}
	responsible, e := questionAuthors(ctx, tx, d.OwnerID, p, d.AuthorIDs, d.LegacyUnattributed)
	if e != nil {
		return sealed, gate, responsible, e
	}
	_, gate.Digest, e = questionCanonical("question-validation-proof-v1", struct {
		WorkspaceID      string                        `json:"workspaceId"`
		Revision         int64                         `json:"revision"`
		ValidationDigest string                        `json:"validationDigest"`
		Responsibility   question.SourceResponsibility `json:"responsibility"`
	}{d.ID, d.Revision, gate.Digest, responsible})
	return sealed, gate, responsible, e
}
func questionFrozen(d question.DraftView, sealed question.SealedPackage, gate question.ValidationReport, responsible question.SourceResponsibility) question.FrozenBody {
	ids := make([]question.Identity, 0, len(sealed.Instances))
	for _, i := range sealed.Instances {
		ids = append(ids, i.Identity)
	}
	generators, verifiers := question.UsedEngineVersions(sealed.Package, sealed.Instances)
	return question.FrozenBody{CatalogueVersion: d.CatalogueVersion, CatalogueSHA256: d.CatalogueSHA256, QuestionPackage: sealed.Package, SourceMap: d.SourceMap, AuthorIDs: responsible.AuthorIDs, LegacyUnattributed: responsible.LegacyUnattributed, Resolved: sealed.Resolved, Objectives: sealed.Objectives, Generation: sealed.Generation, InstanceIdentities: ids, Coverage: gate.Coverage, GeneratorVersions: generators, VerifierVersions: verifiers}
}
func questionFitPage[T any](page *question.Page[T]) error {
	for {
		raw, err := json.Marshal(page)
		if err != nil {
			return err
		}
		if len(raw) <= question.MaxResponseBytes {
			return nil
		}
		if len(page.Items) <= 1 {
			return question.ErrLimitExceeded
		}
		page.Items = page.Items[:len(page.Items)-1]
		page.Limit = len(page.Items)
	}
}

// Check the eventual reviewed DTO as well as today's pending DTO. The frozen limit remains unchanged.
func questionSubmissionBudget(out question.SubmissionView) error {
	const id = "ffffffff-ffff-4fff-bfff-ffffffffffff"
	note := strings.Repeat("<", 1000) // A legal note with the largest JSON escaping per rune.
	out.Status = "approved"
	out.Review = &question.ReviewDecision{ID: id, SubmissionID: out.ID, ReviewerID: id, FrozenDigest: out.Frozen.FrozenDigest, CreatedAt: "2099-12-31T23:59:59Z", Decision: "approve", IndependenceNote: note, GenerationNote: note, Note: note}
	raw, err := json.Marshal(out)
	if err != nil {
		return err
	}
	if len(raw) > question.MaxResponseBytes {
		return question.ErrLimitExceeded
	}
	return nil
}
func (s *Store) SubmitQuestionDraft(ctx context.Context, a question.Access, id string, input question.SubmitInput) (question.SubmissionView, error) {
	ctx, cancel := context.WithTimeout(ctx, questionTimeout)
	defer cancel()
	var out question.SubmissionView
	if !question.ValidID(id) || input.ExpectedRevision < 1 || !question.ValidSHA(input.ExpectedDigest) {
		return out, auth.ErrInvalidInput
	}
	// Discover normalized inherited authors before taking sorted account locks. Regenerate again under the content lock.
	var related []string
	err := s.questionReadTx(ctx, a, question.SubmitDraftAction, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		d, _, e := s.readQuestionDraft(ctx, tx, u, id, true)
		if e != nil {
			return e
		}
		related = d.AuthorIDs
		if d.Status == "editing" {
			_, _, responsible, e := questionCheckDraft(ctx, tx, d)
			if e != nil {
				return e
			}
			related = responsible.AuthorIDs
		}
		return nil
	})
	if err != nil {
		return out, err
	}
	err = s.questionTx(ctx, a, question.SubmitDraftAction, related, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		d, _, err := s.readQuestionDraft(ctx, tx, u, id, true)
		if err != nil {
			return err
		}
		prior, found, err := questionJSONReplay[question.SubmissionView](s, ctx, tx, u, a, question.SubmitDraftAction, id, input)
		if err != nil {
			return err
		}
		if found {
			out = prior
			return nil
		}
		if d.Status != "editing" || d.Revision != input.ExpectedRevision {
			return question.ErrDraftConflict
		}
		sealed, gate, responsible, err := questionCheckDraft(ctx, tx, d)
		if err != nil {
			return err
		}
		if gate.Digest != input.ExpectedDigest {
			return question.ErrDraftConflict
		}
		if !gate.ReadyToSubmit {
			return question.ErrNotReady
		}
		if !questionAuthorsLocked(responsible.AuthorIDs, related, u.ID) {
			return question.ErrDraftConflict
		}
		frozen := questionFrozen(d, sealed, gate, responsible)
		payload, digest, err := question.CanonicalFrozen(frozen, sealed.Instances)
		if err != nil {
			return err
		}
		if len(payload) > question.MaxResponseBytes {
			return question.ErrLimitExceeded
		}
		gate.FrozenBytes = len(payload)
		if _, err = questionStoreSealedTx(ctx, tx, sealed, d.SourceMap, responsible); err != nil {
			return err
		}
		// Only server-derived provenance may increase on the saved revision; no caller body is accepted here.
		if _, err = tx.ExecContext(ctx, `UPDATE question_workspaces SET gate=$2,legacy_unattributed=$3,updated_at=$4 WHERE id=$1`, id, body(gate), responsible.LegacyUnattributed, now); err != nil {
			return err
		}
		for _, author := range responsible.AuthorIDs {
			if _, err = tx.ExecContext(ctx, `INSERT INTO question_workspace_authors VALUES($1,$2) ON CONFLICT DO NOTHING`, id, author); err != nil {
				return err
			}
		}
		subID, err := workflowID()
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO question_submissions(id,workspace_id,owner_user_id,revision,package_id,package_version,package_sha256,catalogue_version,catalogue_sha256,frozen_body,frozen_bytes,frozen_digest,gate,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`, subID, id, u.ID, d.Revision, sealed.Package.ID, sealed.Package.Version, sealed.PackageSHA, d.CatalogueVersion, d.CatalogueSHA256, string(payload), payload, digest, body(gate), now)
		if err != nil {
			return err
		}
		for _, author := range responsible.AuthorIDs {
			if _, err = tx.ExecContext(ctx, `INSERT INTO question_submission_authors VALUES($1,$2)`, subID, author); err != nil {
				return err
			}
		}
		_, err = tx.ExecContext(ctx, `INSERT INTO question_submission_members(submission_id,kind,id,version,sha256,template_id,instance_id,blueprint_id)
   SELECT $1::uuid,'template',t.id,t.version,t.sha256,t.id,NULL,NULL FROM question_templates t JOIN question_packages p ON p.id=$2 AND p.version=$3 CROSS JOIN LATERAL jsonb_array_elements(p.body#>'{body,templates}') j WHERE t.id=j->>'id' AND t.version=(j->>'version')::integer
   UNION ALL SELECT $1::uuid,'blueprint',b.id,b.version,b.sha256,NULL,NULL,b.id FROM question_blueprints b JOIN question_packages p ON p.id=$2 AND p.version=$3 CROSS JOIN LATERAL jsonb_array_elements(p.body#>'{body,blueprints}') j WHERE b.id=j->>'id' AND b.version=(j->>'version')::integer
   UNION ALL SELECT $1::uuid,'instance',j->>'id',(j->>'version')::integer,j->>'sha256',NULL,j->>'id',NULL FROM question_packages p CROSS JOIN LATERAL jsonb_array_elements(p.instance_identities) j WHERE p.id=$2 AND p.version=$3`, subID, sealed.Package.ID, sealed.Package.Version)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE question_submissions SET sealed=true WHERE id=$1`, subID); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE question_workspaces SET status='submitted',updated_at=$2 WHERE id=$1`, id, now); err != nil {
			return err
		}
		frozen.FrozenDigest = digest
		out = question.SubmissionView{ID: subID, WorkspaceID: id, OwnerID: u.ID, Status: "pending", Revision: d.Revision, Frozen: frozen, Gate: gate, CreatedAt: now.UTC().Format(time.RFC3339)}
		if err = questionSubmissionBudget(out); err != nil {
			return err
		}
		if err = questionEvent(ctx, tx, u, a, question.SubmitDraftAction, "submission", subID, d.Gate.Digest, digest, "editing", "pending", "", now); err != nil {
			return err
		}
		return questionJSONRemember(s, ctx, tx, u, a, question.SubmitDraftAction, id, input, out)
	})
	return out, err
}
func (s *Store) readQuestionSubmission(ctx context.Context, tx *sql.Tx, u auth.User, id string, ownerOnly bool) (question.SubmissionView, []question.Instance, error) {
	var out question.SubmissionView
	var raw, gate []byte
	var digest string
	var created time.Time
	err := tx.QueryRowContext(ctx, `SELECT id::text,workspace_id::text,owner_user_id::text,revision,status,frozen_bytes,frozen_digest,gate,created_at FROM question_submissions WHERE id=$1 AND sealed`, id).Scan(&out.ID, &out.WorkspaceID, &out.OwnerID, &out.Revision, &out.Status, &raw, &digest, &gate, &created)
	if err != nil {
		return out, nil, workflowRowError(err)
	}
	if out.OwnerID != u.ID && (ownerOnly || !publication.HasRole(u, auth.RoleReviewer) && !publication.HasRole(u, auth.RoleAdmin)) {
		return question.SubmissionView{}, nil, auth.ErrNotFound
	}
	if len(raw) > question.MaxResponseBytes {
		return out, nil, question.ErrLimitExceeded
	}
	var envelope struct {
		Purpose string                 `json:"purpose"`
		Body    question.FrozenPayload `json:"body"`
	}
	if json.Unmarshal(raw, &envelope) != nil || json.Unmarshal(gate, &out.Gate) != nil {
		return out, nil, auth.ErrUnavailable
	}
	_, computed, err := question.CanonicalFrozen(envelope.Body.Body, envelope.Body.Instances)
	if err != nil || computed != digest || envelope.Purpose != "question-submission-v1" {
		return out, nil, auth.ErrUnavailable
	}
	out.Frozen = envelope.Body.Body
	out.Frozen.FrozenDigest = digest
	out.CreatedAt = created.UTC().Format(time.RFC3339)
	var review question.ReviewDecision
	var checks []byte
	var reviewed time.Time
	err = tx.QueryRowContext(ctx, `SELECT id::text,submission_id::text,reviewer_user_id::text,frozen_digest,decision,checks,independence_note,generation_note,note,created_at FROM question_review_decisions WHERE submission_id=$1`, id).Scan(&review.ID, &review.SubmissionID, &review.ReviewerID, &review.FrozenDigest, &review.Decision, &checks, &review.IndependenceNote, &review.GenerationNote, &review.Note, &reviewed)
	if errors.Is(err, sql.ErrNoRows) {
		return out, envelope.Body.Instances, nil
	}
	if err != nil {
		return out, nil, err
	}
	if json.Unmarshal(checks, &review.Checks) != nil {
		return out, nil, auth.ErrUnavailable
	}
	review.CreatedAt = reviewed.UTC().Format(time.RFC3339)
	out.Review = &review
	return out, envelope.Body.Instances, nil
}
func (s *Store) ReadQuestionSubmission(ctx context.Context, a question.Access, id string) (question.SubmissionView, error) {
	var out question.SubmissionView
	if !question.ValidID(id) {
		return out, auth.ErrInvalidInput
	}
	err := s.questionReadTx(ctx, a, question.ReadSubmissionAction, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		var e error
		out, _, e = s.readQuestionSubmission(ctx, tx, u, id, false)
		return e
	})
	return out, err
}
func (s *Store) ListQuestionInstances(ctx context.Context, a question.Access, id string, q question.ListQuery) (question.Page[question.Instance], error) {
	out := question.Page[question.Instance]{Items: []question.Instance{}}
	if !question.ValidID(id) || q.Scope != "" || q.Status != "" {
		return out, auth.ErrInvalidInput
	}
	pq, err := publication.ValidateList(publication.ListQuery{Limit: q.Limit, Offset: q.Offset})
	if err != nil {
		return out, err
	}
	out.Limit = pq.Limit
	out.Offset = pq.Offset
	err = s.questionReadTx(ctx, a, question.ListInstancesAction, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		_, instances, e := s.readQuestionSubmission(ctx, tx, u, id, false)
		if e != nil {
			return e
		}
		out.Total = len(instances)
		start := out.Offset
		if start > out.Total {
			start = out.Total
		}
		end := start + out.Limit
		if end > out.Total {
			end = out.Total
		}
		out.Items = instances[start:end]
		return questionFitPage(&out)
	})
	return out, err
}
func (s *Store) ReviseQuestionSubmission(ctx context.Context, a question.Access, id string) (question.DraftView, error) {
	ctx, cancel := context.WithTimeout(ctx, questionTimeout)
	defer cancel()
	var out question.DraftView
	if !question.ValidID(id) {
		return out, auth.ErrInvalidInput
	}
	sub, err := s.ReadQuestionSubmission(ctx, a, id)
	if err != nil {
		return out, err
	}
	related, err := s.questionRelated(ctx, sub.Frozen.QuestionPackage, sub.WorkspaceID)
	if err != nil {
		return out, err
	}
	err = s.questionTx(ctx, a, question.ReviseSubmissionAction, related, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		sub, _, err := s.readQuestionSubmission(ctx, tx, u, id, true)
		if err != nil {
			return err
		}
		prior, found, err := questionJSONReplay[question.DraftView](s, ctx, tx, u, a, question.ReviseSubmissionAction, id, struct{}{})
		if err != nil {
			return err
		}
		if found {
			out = prior
			return nil
		}
		if sub.Status != "approved" {
			return question.ErrReviewConflict
		}
		in := question.DraftInput{CatalogueVersion: sub.Frozen.CatalogueVersion, QuestionPackage: sub.Frozen.QuestionPackage, SourceMap: sub.Frozen.SourceMap}
		out, err = s.createQuestionDraft(ctx, tx, u, in, id, sub.Frozen.AuthorIDs, sub.Frozen.LegacyUnattributed, related, now)
		if err != nil {
			return err
		}
		if err = questionEvent(ctx, tx, u, a, question.ReviseSubmissionAction, "draft", out.ID, sub.Frozen.FrozenDigest, out.Gate.Digest, "approved", "editing:1", "", now); err != nil {
			return err
		}
		return questionJSONRemember(s, ctx, tx, u, a, question.ReviseSubmissionAction, id, struct{}{}, out)
	})
	return out, err
}
func (s *Store) ListQuestionSubmissions(ctx context.Context, a question.Access, q question.ListQuery) (question.Page[question.SubmissionSummary], error) {
	scope := q.Scope
	if scope == "review" {
		q.Scope = ""
	}
	pq, err := publication.ValidateList(publication.ListQuery{Scope: q.Scope, Status: q.Status, Limit: q.Limit, Offset: q.Offset}, "pending", "approved", "returned")
	q = question.ListQuery{Scope: scope, Status: pq.Status, Limit: pq.Limit, Offset: pq.Offset}
	out := question.Page[question.SubmissionSummary]{Items: []question.SubmissionSummary{}, Limit: q.Limit, Offset: q.Offset}
	if err != nil {
		return out, err
	}
	err = s.questionReadTx(ctx, a, question.ListSubmissionsAction, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		if q.Scope == "" {
			switch {
			case publication.HasRole(u, auth.RoleAdmin):
				q.Scope = "all"
			case publication.HasRole(u, auth.RoleReviewer):
				q.Scope = "review"
			default:
				q.Scope = "mine"
			}
		}
		if q.Scope == "all" && !publication.HasRole(u, auth.RoleAdmin) || q.Scope == "review" && !publication.HasRole(u, auth.RoleReviewer) {
			return auth.ErrForbidden
		}
		if q.Scope == "review" && q.Status == "" {
			q.Status = "pending"
		}
		where := `WHERE sealed AND ($1='all' OR ($1='mine' AND owner_user_id=$2) OR ($1='review' AND (($3='pending' AND NOT EXISTS(SELECT 1 FROM question_submission_authors a WHERE a.submission_id=question_submissions.id AND a.user_id=$2)) OR ($3<>'pending' AND EXISTS(SELECT 1 FROM question_review_decisions d WHERE d.submission_id=question_submissions.id AND d.reviewer_user_id=$2))))) AND ($3='' OR status=$3)`
		if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM question_submissions `+where, q.Scope, u.ID, q.Status).Scan(&out.Total); err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, `SELECT id::text,workspace_id::text,owner_user_id::text,package_id,package_version,catalogue_version,status,revision,frozen_digest,created_at FROM question_submissions `+where+` ORDER BY created_at DESC,id DESC LIMIT $4 OFFSET $5`, q.Scope, u.ID, q.Status, q.Limit, q.Offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var item question.SubmissionSummary
			var created time.Time
			if err = rows.Scan(&item.ID, &item.WorkspaceID, &item.OwnerID, &item.PackageID, &item.PackageVersion, &item.CatalogueVersion, &item.Status, &item.Revision, &item.FrozenDigest, &created); err != nil {
				return err
			}
			item.CreatedAt = created.UTC().Format(time.RFC3339)
			out.Items = append(out.Items, item)
		}
		return rows.Err()
	})
	return out, err
}
