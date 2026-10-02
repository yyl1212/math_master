package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

func questionHead(ctx context.Context, tx *sql.Tx) (*string, error) {
	var id string
	err := tx.QueryRowContext(ctx, `SELECT publication_id::text FROM question_heads WHERE singleton`).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &id, err
}
func questionLoadCandidate(ctx context.Context, tx *sql.Tx, id string) (question.Candidate, error) {
	c := question.Candidate{Templates: []question.Template{}, Instances: []question.Instance{}, Blueprints: []question.Blueprint{}, Changes: []question.Change{}}
	var raw, diff, changes []byte
	var sha string
	err := tx.QueryRowContext(ctx, `SELECT manifest_bytes,manifest_sha256,diff,changes FROM question_publications WHERE id=$1 AND sealed`, id).Scan(&raw, &sha, &diff, &changes)
	if err != nil {
		return c, workflowRowError(err)
	}
	if len(raw) > question.MaxManifestBytes {
		return c, question.ErrLimitExceeded
	}
	var envelope struct {
		Purpose string            `json:"purpose"`
		Body    question.Manifest `json:"body"`
	}
	if json.Unmarshal(raw, &envelope) != nil || json.Unmarshal(diff, &c.Diff) != nil || json.Unmarshal(changes, &c.Changes) != nil {
		return c, auth.ErrUnavailable
	}
	manifestBytes := len(raw)
	c.Manifest = envelope.Body
	_, computed, err := question.CanonicalManifest(c.Manifest)
	if err != nil || sha != computed || envelope.Purpose != "question-manifest-v1" {
		return c, auth.ErrUnavailable
	}
	rows, err := tx.QueryContext(ctx, `SELECT m.kind,m.sha256,CASE m.kind WHEN 'template' THEN t.body_bytes WHEN 'instance' THEN i.body_bytes ELSE b.body_bytes END FROM question_publication_members m LEFT JOIN question_templates t ON m.kind='template' AND t.id=m.id AND t.version=m.version LEFT JOIN question_instances i ON m.kind='instance' AND i.id=m.id AND i.version=m.version AND i.sealed LEFT JOIN question_blueprints b ON m.kind='blueprint' AND b.id=m.id AND b.version=m.version AND b.sealed WHERE m.publication_id=$1 ORDER BY m.kind,m.id LIMIT 11201`, id)
	if err != nil {
		return c, err
	}
	defer rows.Close()
	bodyBytes := 0
	for rows.Next() {
		var kind, sha string
		var raw []byte
		if err = rows.Scan(&kind, &sha, &raw); err != nil {
			return c, err
		}
		if kind != "blueprint" {
			bodyBytes += len(raw)
			if bodyBytes > question.MaxBankBodyBytes {
				return c, question.ErrLimitExceeded
			}
		}
		var envelope struct {
			Body json.RawMessage `json:"body"`
		}
		if json.Unmarshal(raw, &envelope) != nil {
			return c, auth.ErrUnavailable
		}
		switch kind {
		case "template":
			var t question.Template
			if json.Unmarshal(envelope.Body, &t) != nil {
				return c, auth.ErrUnavailable
			}
			c.Templates = append(c.Templates, t)
		case "instance":
			var i question.Instance
			if json.Unmarshal(envelope.Body, &i) != nil {
				return c, auth.ErrUnavailable
			}
			i.Identity.SHA256 = sha
			c.Instances = append(c.Instances, i)
		case "blueprint":
			var b question.Blueprint
			if json.Unmarshal(envelope.Body, &b) != nil {
				return c, auth.ErrUnavailable
			}
			c.Blueprints = append(c.Blueprints, b)
		default:
			return c, auth.ErrUnavailable
		}
		if err = question.CheckBankLimits(len(c.Templates), len(c.Instances), len(c.Blueprints), bodyBytes, manifestBytes); err != nil {
			return c, err
		}
	}
	return c, rows.Err()
}
func questionEvidence(ctx context.Context, tx *sql.Tx, m question.Manifest, currentReviewers bool) error {
	const statement = `SELECT count(*) FROM jsonb_to_recordset($1::jsonb->'members') e(identity jsonb,evidence jsonb)
 JOIN question_submission_members sm ON sm.submission_id=(e.evidence->>'submissionId')::uuid AND sm.kind=e.identity->>'kind' AND sm.id=e.identity->>'id' AND sm.version=(e.identity->>'version')::integer AND sm.sha256=e.identity->>'sha256'
 JOIN question_submissions s ON s.id=sm.submission_id AND s.sealed AND s.status='approved' AND s.package_id=e.identity->>'packageId' AND s.package_version=(e.identity->>'packageVersion')::integer AND s.frozen_digest=e.evidence->>'frozenDigest' AND s.catalogue_version=$2 AND s.catalogue_sha256=$3
 JOIN question_review_decisions d ON d.id=(e.evidence->>'decisionId')::uuid AND d.submission_id=s.id AND d.frozen_digest=s.frozen_digest AND d.decision='approve'
 WHERE d.checks @> '{"mathematics":true,"explanations":true,"objectives":true,"sources":true,"illustrations":true,"generation":true}' AND NOT EXISTS(SELECT 1 FROM question_submission_authors a WHERE a.submission_id=s.id AND a.user_id=d.reviewer_user_id)
 AND (NOT $4 OR e.evidence->>'inheritedFrom' IS NOT NULL OR EXISTS(SELECT 1 FROM auth_user_roles r WHERE r.user_id=d.reviewer_user_id AND r.role='reviewer'))`
	var count int
	if err := tx.QueryRowContext(ctx, statement, body(m), m.CatalogueVersion, m.CatalogueSHA256, currentReviewers).Scan(&count); err != nil {
		return err
	}
	if count != len(m.Members) {
		return question.ErrReviewRequired
	}
	inherited := 0
	for _, item := range m.Members {
		if item.Evidence.InheritedFrom != nil {
			inherited++
			if !sameWorkflowHead(item.Evidence.InheritedFrom, m.BaseQuestionHead) {
				return question.ErrReviewRequired
			}
		}
	}
	if inherited == 0 {
		return nil
	}
	const previous = `WITH previous AS MATERIALIZED (SELECT m.kind,m.id,m.version,m.sha256,m.package_id,m.package_version,m.evidence-'inheritedFrom' evidence FROM question_publication_members m JOIN question_publications p ON p.id=m.publication_id AND p.sealed AND p.status='published' WHERE m.publication_id=$2::uuid) SELECT count(*) FROM jsonb_to_recordset($1::jsonb->'members') e(identity jsonb,evidence jsonb) JOIN previous p ON p.kind=e.identity->>'kind' AND p.id=e.identity->>'id' AND p.version=(e.identity->>'version')::integer AND p.sha256=e.identity->>'sha256' AND p.package_id=e.identity->>'packageId' AND p.package_version=(e.identity->>'packageVersion')::integer AND p.evidence=e.evidence-'inheritedFrom' WHERE e.evidence->>'inheritedFrom'=($2::uuid)::text`
	if err := tx.QueryRowContext(ctx, previous, body(m), *m.BaseQuestionHead).Scan(&count); err != nil {
		return err
	}
	if count != inherited {
		return question.ErrReviewRequired
	}
	return nil
}
func questionBlacklisted(ctx context.Context, tx *sql.Tx, m question.Manifest) error {
	var found bool
	err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM jsonb_to_recordset($1::jsonb->'members') e(identity jsonb,evidence jsonb) JOIN question_withdrawals w ON w.kind=e.identity->>'kind' AND w.target_id=e.identity->>'id' AND w.target_version=(e.identity->>'version')::integer)`, body(m)).Scan(&found)
	if err != nil {
		return err
	}
	if found {
		return question.ErrInvalid
	}
	return nil
}
func questionValidateCandidate(ctx context.Context, tx *sql.Tx, c question.Candidate, ready bool) error {
	refs, err := questionReferences(ctx, tx, question.CandidateDraft(c))
	if err != nil {
		return err
	}
	if err = question.CheckCandidate(ctx, c, refs, ready); err != nil {
		return err
	}
	if err = questionEvidence(ctx, tx, c.Manifest, ready); err != nil {
		return err
	}
	return questionBlacklisted(ctx, tx, c.Manifest)
}
func (s *Store) questionReleaseReviewers(ctx context.Context, a question.Access, ids []string, publicationID string) ([]string, error) {
	out := []string{}
	err := s.questionReadTx(ctx, a, question.ReadPublicationAction, func(ctx context.Context, tx *sql.Tx, _ auth.User) error {
		rows, err := tx.QueryContext(ctx, `SELECT DISTINCT d.reviewer_user_id::text FROM question_review_decisions d WHERE d.decision='approve' AND (d.submission_id=ANY($1::uuid[]) OR EXISTS(SELECT 1 FROM question_publication_members m WHERE m.publication_id=$2::uuid AND m.review_id=d.id AND m.evidence->>'inheritedFrom' IS NULL)) ORDER BY d.reviewer_user_id::text`, ids, workflowNullableID(publicationID))
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var id string
			if err = rows.Scan(&id); err != nil {
				return err
			}
			out = append(out, id)
		}
		return rows.Err()
	})
	return out, err
}
func (s *Store) questionApproved(ctx context.Context, tx *sql.Tx, u auth.User, id string) (question.ApprovedSubmission, error) {
	sub, instances, err := s.readQuestionSubmission(ctx, tx, u, id, false)
	if err != nil {
		return question.ApprovedSubmission{}, err
	}
	if sub.Status != "approved" || sub.Review == nil || sub.Review.Decision != "approve" {
		return question.ApprovedSubmission{}, question.ErrReviewRequired
	}
	var role bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM auth_user_roles WHERE user_id=$1 AND role='reviewer')`, sub.Review.ReviewerID).Scan(&role); err != nil {
		return question.ApprovedSubmission{}, err
	}
	if !role {
		return question.ApprovedSubmission{}, question.ErrReviewRequired
	}
	return question.ApprovedSubmission{SubmissionID: id, Frozen: sub.Frozen, Instances: instances, Decision: *sub.Review}, nil
}
func questionInsertPublication(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time, c question.Candidate, status string) (question.PublicationSummary, error) {
	var out question.PublicationSummary
	raw, sha, err := question.CanonicalManifest(c.Manifest)
	if err != nil {
		return out, err
	}
	id, err := workflowID()
	if err != nil {
		return out, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO question_publications(id,base_knowledge_head,base_question_head,catalogue_version,catalogue_sha256,manifest,manifest_bytes,manifest_sha256,diff,changes,creator_user_id,status,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`, id, c.Manifest.BaseKnowledgeHead, c.Manifest.BaseQuestionHead, c.Manifest.CatalogueVersion, c.Manifest.CatalogueSHA256, string(raw), raw, sha, body(c.Diff), body(c.Changes), u.ID, status, now)
	if err != nil {
		return out, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO question_publication_members(publication_id,kind,id,version,sha256,package_id,package_version,template_id,instance_id,blueprint_id,submission_id,review_id,evidence)
 SELECT $1::uuid,e.identity->>'kind',e.identity->>'id',(e.identity->>'version')::integer,e.identity->>'sha256',e.identity->>'packageId',(e.identity->>'packageVersion')::integer,CASE WHEN e.identity->>'kind'='template' THEN e.identity->>'id' END,CASE WHEN e.identity->>'kind'='instance' THEN e.identity->>'id' END,CASE WHEN e.identity->>'kind'='blueprint' THEN e.identity->>'id' END,(e.evidence->>'submissionId')::uuid,(e.evidence->>'decisionId')::uuid,e.evidence FROM jsonb_to_recordset($2::jsonb->'members') e(identity jsonb,evidence jsonb)`, id, body(c.Manifest))
	if err != nil {
		return out, err
	}
	if _, err = tx.ExecContext(ctx, `UPDATE question_publications SET sealed=true WHERE id=$1`, id); err != nil {
		return out, err
	}
	out = question.PublicationSummary{ID: id, Status: status, ManifestSHA: sha, CreatedAt: now.UTC().Format(time.RFC3339), BaseKnowledgeHead: c.Manifest.BaseKnowledgeHead, BaseQuestionHead: c.Manifest.BaseQuestionHead, CatalogueVersion: c.Manifest.CatalogueVersion, CatalogueSHA256: c.Manifest.CatalogueSHA256, TemplateCount: len(c.Templates), InstanceCount: len(c.Instances), BlueprintCount: len(c.Blueprints), Diff: c.Diff}
	future := out
	future.Status = "published"
	envelope := question.PublicationPage{Page: question.Page[question.PublicationSummary]{Items: []question.PublicationSummary{future}, Total: 2147483647, Limit: 100, Offset: 100000}, Head: &id}
	size, _ := json.Marshal(envelope)
	if len(size) > question.MaxResponseBytes {
		return out, question.ErrLimitExceeded
	}
	return out, nil
}
func (s *Store) PrepareQuestionRelease(ctx context.Context, a question.Access, input question.PrepareInput) (question.PublicationSummary, error) {
	ctx, cancel := context.WithTimeout(ctx, questionTimeout)
	defer cancel()
	var out question.PublicationSummary
	if len(input.SubmissionIDs) == 0 || !validWorkflowHead(input.ExpectedKnowledgeHead) || !validWorkflowHead(input.ExpectedQuestionHead) || !publication.ValidNote(input.Reason) {
		return out, auth.ErrInvalidInput
	}
	if len(input.SubmissionIDs) > 20 {
		return out, question.ErrLimitExceeded
	}
	seen := map[string]bool{}
	for _, id := range input.SubmissionIDs {
		if !question.ValidID(id) || seen[id] {
			return out, auth.ErrInvalidInput
		}
		seen[id] = true
	}
	related, err := s.questionReleaseReviewers(ctx, a, input.SubmissionIDs, "")
	if err != nil {
		return out, err
	}
	err = s.questionTx(ctx, a, question.PrepareReleaseAction, related, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		replay, found, err := questionJSONReplay[question.PublicationSummary](s, ctx, tx, u, a, question.PrepareReleaseAction, "", input)
		if err != nil {
			return err
		}
		if found {
			out = replay
			return nil
		}
		khead, err := workflowHead(ctx, tx)
		if err != nil {
			return err
		}
		qhead, err := questionHead(ctx, tx)
		if err != nil {
			return err
		}
		if !sameWorkflowHead(khead, input.ExpectedKnowledgeHead) || !sameWorkflowHead(qhead, input.ExpectedQuestionHead) {
			return question.ErrPublicationStale
		}
		base := question.BaseManifest{Head: qhead}
		if qhead != nil {
			prior, err := questionLoadCandidate(ctx, tx, *qhead)
			if err != nil {
				return err
			}
			base.Manifest = &prior.Manifest
			base.Templates = prior.Templates
			base.Instances = prior.Instances
			base.Blueprints = prior.Blueprints
		}
		selected := []question.ApprovedSubmission{}
		draft := question.DraftInput{QuestionPackage: question.QuestionPackage{Templates: append([]question.Template{}, base.Templates...), Blueprints: append([]question.Blueprint{}, base.Blueprints...), FixedQuestions: []question.FixedQuestion{}}}
		for _, i := range base.Instances {
			draft.QuestionPackage.FixedQuestions = append(draft.QuestionPackage.FixedQuestions, question.FixedQuestion{ID: i.Identity.ID, Version: i.Identity.Version, Body: i.Body})
		}
		for _, id := range input.SubmissionIDs {
			sub, err := s.questionApproved(ctx, tx, u, id)
			if err != nil {
				return err
			}
			selected = append(selected, sub)
			if !questionAuthorsLocked([]string{sub.Decision.ReviewerID}, related, u.ID) {
				return question.ErrReviewRequired
			}
			if draft.CatalogueVersion != 0 && draft.CatalogueVersion != sub.Frozen.CatalogueVersion {
				return question.ErrNotReady
			}
			draft.CatalogueVersion = sub.Frozen.CatalogueVersion
			draft.QuestionPackage.Templates = append(draft.QuestionPackage.Templates, sub.Frozen.QuestionPackage.Templates...)
			draft.QuestionPackage.Blueprints = append(draft.QuestionPackage.Blueprints, sub.Frozen.QuestionPackage.Blueprints...)
			for _, i := range sub.Instances {
				draft.QuestionPackage.FixedQuestions = append(draft.QuestionPackage.FixedQuestions, question.FixedQuestion{ID: i.Identity.ID, Version: i.Identity.Version, Body: i.Body})
			}
		}
		refs, err := questionReferences(ctx, tx, draft)
		if err != nil {
			return err
		}
		c, err := question.BuildCandidate(ctx, base, selected, refs)
		if err != nil {
			return err
		}
		if err = questionEvidence(ctx, tx, c.Manifest, true); err != nil {
			return err
		}
		if err = questionBlacklisted(ctx, tx, c.Manifest); err != nil {
			return err
		}
		out, err = questionInsertPublication(ctx, tx, u, now, c, "prepared")
		if err != nil {
			return err
		}
		if err = questionEvent(ctx, tx, u, a, question.PrepareReleaseAction, "publication", out.ID, "", out.ManifestSHA, "", "prepared", input.Reason, now); err != nil {
			return err
		}
		return questionJSONRemember(s, ctx, tx, u, a, question.PrepareReleaseAction, "", input, out)
	})
	return out, err
}
func (s *Store) ActivateQuestionRelease(ctx context.Context, a question.Access, id string, input question.ActivateInput) (question.PublicationSummary, error) {
	ctx, cancel := context.WithTimeout(ctx, questionTimeout)
	defer cancel()
	var out question.PublicationSummary
	if !question.ValidID(id) || !validWorkflowHead(input.ExpectedKnowledgeHead) || !validWorkflowHead(input.ExpectedQuestionHead) || !question.ValidSHA(input.ExpectedManifestSHA) || !publication.ValidNote(input.Reason) {
		return out, auth.ErrInvalidInput
	}
	related, err := s.questionReleaseReviewers(ctx, a, nil, id)
	if err != nil {
		return out, err
	}
	err = s.questionTx(ctx, a, question.ActivateReleaseAction, related, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		replay, found, err := questionJSONReplay[question.PublicationSummary](s, ctx, tx, u, a, question.ActivateReleaseAction, id, input)
		if err != nil {
			return err
		}
		if found {
			out = replay
			return nil
		}
		out, err = questionReadSummary(ctx, tx, id)
		if err != nil {
			return err
		}
		khead, err := workflowHead(ctx, tx)
		if err != nil {
			return err
		}
		qhead, err := questionHead(ctx, tx)
		if err != nil {
			return err
		}
		if out.Status != "prepared" || out.ManifestSHA != input.ExpectedManifestSHA || !sameWorkflowHead(khead, input.ExpectedKnowledgeHead) || !sameWorkflowHead(qhead, input.ExpectedQuestionHead) || !sameWorkflowHead(out.BaseKnowledgeHead, khead) || !sameWorkflowHead(out.BaseQuestionHead, qhead) {
			return question.ErrPublicationStale
		}
		c, err := questionLoadCandidate(ctx, tx, id)
		if err != nil {
			return err
		}
		if err = questionValidateCandidate(ctx, tx, c, true); err != nil {
			return err
		}

		if _, err = tx.ExecContext(ctx, `UPDATE question_publications SET status='published' WHERE id=$1 AND status='prepared'`, id); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO question_heads VALUES(true,$1) ON CONFLICT(singleton) DO UPDATE SET publication_id=EXCLUDED.publication_id`, id); err != nil {
			return err
		}
		out.Status = "published"
		if err = questionEvent(ctx, tx, u, a, question.ActivateReleaseAction, "publication", id, "", out.ManifestSHA, "prepared", "published", input.Reason, now); err != nil {
			return err
		}
		return questionJSONRemember(s, ctx, tx, u, a, question.ActivateReleaseAction, id, input, out)
	})
	return out, err
}
