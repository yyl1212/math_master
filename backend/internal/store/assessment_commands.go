package store

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

func learningAssessmentEligibility(ctx context.Context, tx *sql.Tx, actor string, k question.Identity, mode assessment.Mode) error {
	switch mode {
	case assessment.ModeDiagnostic:
		return nil
	case assessment.ModeNode:
		state, e := learningKnowledgeState(ctx, tx, actor, k)
		if e != nil {
			return e
		}
		if !state.CanEnter {
			return learning.ErrPrerequisitesUnmet
		}
		return nil
	case assessment.ModeReview:
		var had bool
		e := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM learning_records WHERE owner_user_id=$1 AND knowledge_id=$2) OR EXISTS(SELECT 1 FROM assessment_attempts WHERE owner_user_id=$1 AND knowledge_id=$2)`, actor, k.ID).Scan(&had)
		if e != nil {
			return e
		}
		if !had {
			return learning.ErrStateConflict
		}
		return nil
	}
	return auth.ErrInvalidInput
}
func (s *Store) CreateAssessment(ctx context.Context, a question.Access, in assessment.CreateInput) (assessment.AttemptView, error) {
	var out assessment.AttemptView
	if !question.ValidMathID(in.Knowledge.ID) || in.Knowledge.Version < 1 || !question.ValidSHA(in.Knowledge.SHA256) || !question.ValidMathID(in.Blueprint.ID) || in.Blueprint.Version < 1 || !question.ValidSHA(in.Blueprint.SHA256) || in.ExpectedKnowledgeHead == "" || !question.ValidID(in.ExpectedQuestionHead) || (in.Mode != assessment.ModeNode && in.Mode != assessment.ModeDiagnostic && in.Mode != assessment.ModeReview) {
		return out, auth.ErrInvalidInput
	}
	e := s.learningTx(ctx, a, learning.CreateAssessmentAction, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		digest := workflowRequestSHA(in.Knowledge.ID, in)
		receipt, found, e := learningReplay(ctx, tx, u.ID, string(learning.CreateAssessmentAction), in.Knowledge.ID, a.IdempotencyKey, digest)
		if e != nil {
			return e
		}
		if found {
			p, e := learningReadAssessment(ctx, tx, u.ID, receipt.ResourceID, true)
			if e != nil {
				return e
			}
			out, e = learningAssessmentView(ctx, tx, p, now, nil)
			return e
		}
		if e = correctionNewAttemptGuard(ctx, tx, u.ID, in.Knowledge, 1, now); e != nil {
			return e
		}
		// Read database time after the per-user locks, never a browser deadline.
		if e = tx.QueryRowContext(ctx, `SELECT clock_timestamp()`).Scan(&now); e != nil {
			return e
		}
		if _, e = tx.ExecContext(ctx, `UPDATE assessment_attempts SET state='expired',terminal_at=$2 WHERE owner_user_id=$1 AND state='active' AND expires_at<=$2`, u.ID, now); e != nil {
			return e
		}
		var active string
		e = tx.QueryRowContext(ctx, `SELECT id::text FROM assessment_attempts WHERE owner_user_id=$1 AND state='active' FOR UPDATE`, u.ID).Scan(&active)
		if e == nil {
			p, e := learningReadAssessment(ctx, tx, u.ID, active, false)
			if e != nil {
				return e
			}
			return &learning.ActiveAttemptError{Summary: p.Summary}
		}
		if !errors.Is(e, sql.ErrNoRows) {
			return e
		}
		pool, e := learningSourcePool(ctx, tx, u.ID, in.Knowledge, &in.Blueprint, now)
		if e != nil {
			return e
		}
		if pool.KnowledgeHead != in.ExpectedKnowledgeHead || pool.QuestionHead != in.ExpectedQuestionHead {
			return learning.ErrVersionStale
		}
		if e = learningAssessmentEligibility(ctx, tx, u.ID, in.Knowledge, in.Mode); e != nil {
			return e
		}
		seed, e := learningSeed()
		if e != nil {
			return e
		}
		ids, ready, e := assessment.SelectFive(ctx, pool.Blueprint.CoreObjectiveIndices, assessment.EligibleCandidates(pool.Candidates, now), seed)
		if e != nil {
			return e
		}
		if !ready {
			retry, e := assessment.EarliestReady(ctx, pool.Blueprint.CoreObjectiveIndices, pool.Candidates, now, seed)
			if e != nil {
				return e
			}
			return &learning.NotReadyError{RetryAt: retry}
		}
		bindings, items, e := learningFreezeItems(ctx, tx, pool, ids)
		if e != nil {
			return e
		}
		seal := assessment.Seal{Kind: "assessment", Knowledge: pool.Knowledge, Mode: &in.Mode, Blueprint: pool.BlueprintIdentity, KnowledgePublicationID: pool.KnowledgeHead, QuestionPublicationID: pool.QuestionHead, RuleVersion: 1, Core: append([]int{}, pool.Blueprint.CoreObjectiveIndices...), Seed: hex.EncodeToString(seed[:]), Items: bindings}
		raw, sha, e := assessment.CanonicalSeal(seal)
		if e != nil {
			return e
		}
		id, e := workflowID()
		if e != nil {
			return e
		}
		watermark, e := learningRecordExposure(ctx, tx, u.ID, nil, now)
		if e != nil {
			return e
		}
		if _, e = tx.ExecContext(ctx, `INSERT INTO assessment_attempts(id,owner_user_id,knowledge_id,knowledge_version,knowledge_sha256,knowledge_publication_id,question_publication_id,blueprint_id,blueprint_version,blueprint_sha256,mode,rule_version,core,seed,seal,seal_bytes,seal_sha256,created_at,expires_at,creation_exposure_sequence) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,1,$12,$13,$14,$15,$16,$17,$18,$19)`, id, u.ID, pool.Knowledge.ID, pool.Knowledge.Version, pool.Knowledge.SHA256, pool.KnowledgeHead, pool.QuestionHead, in.Blueprint.ID, in.Blueprint.Version, in.Blueprint.SHA256, string(in.Mode), body(seal.Core), seal.Seed, string(raw), raw, sha, now, now.Add(24*time.Hour), watermark); e != nil {
			return e
		}
		for _, b := range bindings {
			if _, e = tx.ExecContext(ctx, `INSERT INTO assessment_items(attempt_id,position,instance_id,instance_version,instance_sha256,binding) VALUES($1,$2,$3,$4,$5,$6)`, id, b.Position, b.Instance.ID, b.Instance.Version, b.Instance.SHA256, body(b)); e != nil {
				return e
			}
		}
		if e = learningSaveDependencies(ctx, tx, u.ID, "assessment", id, raw, false); e != nil {
			return e
		}
		if _, e = tx.ExecContext(ctx, `UPDATE assessment_attempts SET sealed=true WHERE id=$1`, id); e != nil {
			return e
		}
		if e = learningRecordViews(ctx, tx, u.ID, bindings, now); e != nil {
			return e
		}
		if in.Mode == assessment.ModeNode {
			if _, e = learningUnlock(ctx, tx, u.ID, pool.Knowledge, "assessment", id, now); e != nil {
				return e
			}
		}
		if e = learningRemember(ctx, tx, u.ID, string(learning.CreateAssessmentAction), in.Knowledge.ID, a.IdempotencyKey, digest, learning.Receipt{ResourceKind: "assessment", ResourceID: id, Status: 201}); e != nil {
			return e
		}
		p := assessmentRecord{Summary: assessment.AttemptSummary{ID: id, Kind: "assessment", Knowledge: pool.Knowledge, Mode: &in.Mode, State: "active", CreatedAt: now.UTC(), ExpiresAt: now.Add(24 * time.Hour).UTC()}, Seal: seal, CreationSequence: watermark}
		out, e = learningAssessmentView(ctx, tx, p, now, items)
		return e
	})
	if e != nil {
		return assessment.AttemptView{}, e
	}
	return out, nil
}
func (s *Store) SubmitAssessment(ctx context.Context, a question.Access, id string, in assessment.SubmitInput) (assessment.ResultView, error) {
	var out assessment.ResultView
	if !question.ValidID(id) {
		return out, auth.ErrInvalidInput
	}
	e := s.learningTx(ctx, a, learning.SubmitAssessmentAction, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		p, e := learningReadAssessment(ctx, tx, u.ID, id, true)
		if e != nil {
			return e
		}
		digest := workflowRequestSHA(id, in)
		_, found, e := learningReplay(ctx, tx, u.ID, string(learning.SubmitAssessmentAction), id, a.IdempotencyKey, digest)
		if e != nil {
			return e
		}
		if found {
			out, e = learningAssessmentResult(ctx, tx, u.ID, p, now, nil)
			return e
		}
		if p.Summary.State != "active" {
			return learning.ErrStateConflict
		}
		if e = tx.QueryRowContext(ctx, `SELECT clock_timestamp()`).Scan(&now); e != nil {
			return e
		}
		if !now.Before(p.Summary.ExpiresAt) {
			return learning.ErrAssessmentExpired
		}
		items, e := learningLoadItems(ctx, tx, p.Seal)
		if e != nil {
			return e
		}
		if e = assessment.ValidateAnswers(items, in); e != nil {
			return e
		}
		reasons, e := correctionEvidenceGuard(ctx, tx, u.ID, correction.EvidenceRef{Kind: correction.EvidenceKind("assessment"), ID: id})
		if e != nil {
			return e
		}
		current, e := learningIsCurrent(ctx, tx, p.Seal.Knowledge)
		if e != nil {
			return e
		}
		if !current && !learningHasReason(reasons, assessment.KnowledgeWithdrawn) {
			reasons = append(reasons, assessment.KnowledgeUpdated)
		}
		if current {
			e = learningAssessmentEligibility(ctx, tx, u.ID, p.Seal.Knowledge, *p.Seal.Mode)
			if errors.Is(e, learning.ErrPrerequisitesUnmet) {
				reasons = append(reasons, assessment.KnowledgeUpdated)
			} else if e != nil {
				return e
			}
		}
		exposed, e := learningExposureSince(ctx, tx, u.ID, p.CreationSequence, p.Seal.Items)
		if e != nil {
			return e
		}
		if exposed {
			reasons = append(reasons, assessment.ExposedAfterCreation)
		}
		reasons = learningUniqueReasons(reasons)
		outcome := assessment.Affected
		var score *int
		var passed *bool
		if len(reasons) == 0 {
			n, ok, e := assessment.GradeFive(items, in)
			if e != nil {
				return e
			}
			score, passed = &n, &ok
			outcome = assessment.Failed
			if ok {
				outcome = assessment.Passed
			}
		}
		// Original legal inputs and original score are immutable even after a withdrawal.
		for _, ans := range in.Answers {
			if _, e = tx.ExecContext(ctx, `INSERT INTO assessment_answers(attempt_id,owner_user_id,position,instance_id,instance_version,instance_sha256,answer) VALUES($1,$2,$3,$4,$5,$6,$7)`, id, u.ID, ans.Position, ans.Instance.ID, ans.Instance.Version, ans.Instance.SHA256, body(ans.Answer)); e != nil {
				return e
			}
		}
		sequence, e := learningRecordExposure(ctx, tx, u.ID, nil, now)
		if e != nil {
			return e
		}
		if e = tx.QueryRowContext(ctx, `SELECT clock_timestamp()`).Scan(&now); e != nil {
			return e
		}
		if !now.Before(p.Summary.ExpiresAt) {
			return learning.ErrAssessmentExpired
		}
		progress := assessment.ProgressUpdate{Knowledge: p.Seal.Knowledge, NewlyUnlocked: []question.Identity{}}
		if _, e = tx.ExecContext(ctx, `INSERT INTO assessment_results(attempt_id,owner_user_id,rule_version,outcome,score,passed,original_reasons,progress,submitted_at) VALUES($1,$2,1,$3,$4,$5,$6,$7,$8)`, id, u.ID, string(outcome), score, passed, body(reasons), body(progress), now); e != nil {
			return e
		}
		if _, e = tx.ExecContext(ctx, `UPDATE assessment_attempts SET state='submitted',terminal_at=$2,submission_exposure_sequence=$3 WHERE id=$1`, id, now, sequence); e != nil {
			return e
		}
		validity := assessment.Effective
		if len(reasons) > 0 {
			validity = assessment.Restricted
		}
		progress, e = learningApplyAssessment(ctx, tx, u, now, assessment.AttemptFact{ID: id, Knowledge: p.Seal.Knowledge, Mode: *p.Seal.Mode, Outcome: outcome, Score: score, Passed: passed, SubmittedAt: now, Validity: validity})
		if e != nil {
			return e
		}
		if _, e = tx.ExecContext(ctx, `UPDATE assessment_results SET progress=$2,sealed=true WHERE attempt_id=$1`, id, body(progress)); e != nil {
			return e
		}
		p.Summary.State = "submitted"
		t := now.UTC()
		p.Summary.SubmittedAt = &t
		out, e = learningAssessmentResult(ctx, tx, u.ID, p, now, items)
		if e != nil {
			return e
		}
		return learningRemember(ctx, tx, u.ID, string(learning.SubmitAssessmentAction), id, a.IdempotencyKey, digest, learning.Receipt{ResourceKind: "assessment", ResourceID: id, Status: 200})
	})
	if e != nil {
		return assessment.ResultView{}, e
	}
	return out, nil
}
func (s *Store) AbandonAssessment(ctx context.Context, a question.Access, id string) (assessment.AttemptView, error) {
	var out assessment.AttemptView
	if !question.ValidID(id) {
		return out, auth.ErrInvalidInput
	}
	e := s.learningTx(ctx, a, learning.AbandonAssessmentAction, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		p, e := learningReadAssessment(ctx, tx, u.ID, id, true)
		if e != nil {
			return e
		}
		digest := workflowRequestSHA(id, struct{}{})
		_, found, e := learningReplay(ctx, tx, u.ID, string(learning.AbandonAssessmentAction), id, a.IdempotencyKey, digest)
		if e != nil {
			return e
		}
		if !found {
			if p.Summary.State != "active" {
				return learning.ErrStateConflict
			}
			if e = tx.QueryRowContext(ctx, `SELECT clock_timestamp()`).Scan(&now); e != nil {
				return e
			}
			if !now.Before(p.Summary.ExpiresAt) {
				return learning.ErrAssessmentExpired
			}
			if _, e = tx.ExecContext(ctx, `UPDATE assessment_attempts SET state='abandoned',terminal_at=$2 WHERE id=$1`, id, now); e != nil {
				return e
			}
			p.Summary.State = "abandoned"
			if e = learningRemember(ctx, tx, u.ID, string(learning.AbandonAssessmentAction), id, a.IdempotencyKey, digest, learning.Receipt{ResourceKind: "assessment", ResourceID: id, Status: 200}); e != nil {
				return e
			}
		}
		out, e = learningAssessmentView(ctx, tx, p, now, nil)
		return e
	})
	if e != nil {
		return assessment.AttemptView{}, e
	}
	return out, nil
}
