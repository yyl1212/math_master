package store

import (
	"context"
	"database/sql"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/notification"
	"time"
)

func correctionAdvanceCursor(ctx context.Context, tx *sql.Tx, l correction.Lease, ref correction.EvidenceRef) error {
	j, e := correctionReadJob(ctx, tx, l.JobID, true)
	if e != nil {
		return e
	}
	now, e := dbClock(ctx, tx)
	if e != nil {
		return e
	}
	if !correctionLeaseMatches(j, l, now) {
		return correction.ErrLeaseLost
	}
	if j.Meta.Sequence >= correction.MaxSequence || j.Meta.ProcessedCount >= correction.MaxSequence {
		return correction.ErrConflict
	}
	if j.Cursor != nil && (string(ref.Kind) < string(j.Cursor.Kind) || ref.Kind == j.Cursor.Kind && ref.ID <= j.Cursor.ID) {
		return correction.ErrConflict
	}
	seq := j.Meta.Sequence + 1
	_, e = tx.ExecContext(ctx, `UPDATE correction_jobs SET cursor_kind=$2,cursor_id=$3,processed_count=processed_count+1,sequence=$4 WHERE id=$1`, l.JobID, ref.Kind, ref.ID, seq)
	if e != nil {
		return e
	}
	return correctionEvent(ctx, tx, "job", l.JobID, "job_continued", "", j.Meta.CaseID, nil, seq, map[string]any{"cursor": correction.ScanKey{Kind: ref.Kind, ID: ref.ID}, "processedCount": j.Meta.ProcessedCount + 1})
}
func correctionProcessEvidence(ctx context.Context, tx *sql.Tx, l correction.Lease, ref correction.EvidenceRef, now time.Time) error {
	j, e := correctionReadJob(ctx, tx, l.JobID, false)
	if e != nil {
		return e
	}
	if !correctionLeaseMatches(j, l, now) {
		return correction.ErrLeaseLost
	}
	meta, e := correctionReadEvidenceMetadata(ctx, tx, ref)
	if e != nil {
		return e
	}
	if j.Evidence != nil && (*j.Evidence != ref || j.Owner == nil || *j.Owner != meta.Owner) {
		return auth.ErrNotFound
	}
	account, e := readAccount(ctx, tx, meta.Owner, true)
	if e != nil {
		return e
	}
	if e = correction.Authorize(account.User, correction.ReadOwnAction); e != nil {
		return e
	}
	ctx = learningWithProjection(ctx, tx, meta.Owner)
	if e = correctionLockCase(ctx, tx, j.Meta.CaseID); e != nil {
		return e
	}
	ctx = context.WithValue(ctx, correctionProcessingKey{}, j)
	if !meta.Terminal && (ref.Kind == correction.AssessmentEvidence || ref.Kind == correction.PracticeEvidence) {
		if e = notificationAppend(ctx, tx, meta.Owner, notification.Source{DedupKey: "checking:" + j.Meta.CaseID + ":" + string(ref.Kind) + ":" + ref.ID, Type: notification.Checking, Evidence: ref, CaseID: j.Meta.CaseID}); e != nil {
			return e
		}
		return correctionAdvanceCursor(ctx, tx, l, ref)
	}
	plan := j.Meta.Plan
	if j.Meta.Type == correction.AttemptTerminal {
		var p correction.PlanRef
		e = tx.QueryRowContext(ctx, `SELECT p.id::text,p.version FROM correction_plans p WHERE p.case_id=$1 AND p.status='approved' AND p.sealed AND p.algorithm_version=1 AND NOT EXISTS(SELECT 1 FROM correction_plans child WHERE child.id=p.id AND child.status='approved' AND child.version>p.version) ORDER BY p.id,p.version LIMIT 1`, j.Meta.CaseID).Scan(&p.ID, &p.Version)
		if e == nil {
			plan = &p
		} else if !errors.Is(e, sql.ErrNoRows) {
			return e
		}
	}
	b, buildError := correctionBuildBasis(ctx, tx, j.Meta.CaseID, plan, ref)
	v := correction.Evaluation{Status: correction.AwaitingReview, Reason: correction.NoApprovedBasis, Correct: []bool{}}
	if ref.Kind == correction.LearningEventEvidence || ref.Kind == correction.EnrollmentEvidence {
		v.Status = correction.ReviewMaterial
		v.Reason = correction.SourceInvalid
		if ref.Kind == correction.EnrollmentEvidence {
			v.Reason = correction.PathWithdrawn
		}
	}
	if buildError != nil {
		if errors.Is(buildError, correction.ErrConflict) {
			v.Status = correction.AwaitingReview
			v.Reason = correction.ConflictingBasis
		} else if errors.Is(buildError, correction.ErrSourceStale) {
			v.Status = correction.RetakeRequired
			v.Reason = correction.SourceInvalid
		} else {
			return buildError
		}
	} else if plan != nil && (ref.Kind == correction.AssessmentEvidence || ref.Kind == correction.PracticeEvidence) {
		v, e = correction.Evaluate(b)
		if e != nil {
			return e
		}
		if c, e := correctionReadCase(ctx, tx, j.Meta.CaseID); e != nil {
			return e
		} else if c.Kind == correction.GradingRuleCase && v.Status != correction.RetakeRequired && v.Status != correction.AwaitingReview {
			v.Reason = correction.RuleRegraded
		}
		deps := []learning.EvidenceDependency{}
		for _, d := range b.EffectiveDeps {
			deps = append(deps, learning.EvidenceDependency{Kind: d.Kind, ID: d.ID, Version: d.Version, SHA256: d.SHA256})
		}
		rs, e := learningEvidenceRestrictions(ctx, tx, deps)
		if e != nil {
			return e
		}
		current, e := learningIsCurrent(ctx, tx, b.OriginalSeal.Knowledge)
		if e != nil {
			return e
		}
		if len(rs) > 0 || !current {
			v = correction.Evaluation{Status: correction.RetakeRequired, Reason: correction.SourceInvalid, Correct: []bool{}}
			if !current {
				v.Reason = correction.KnowledgeChanged
			}
		}
		if ref.Kind == correction.AssessmentEvidence {
			var exposed bool
			if e = tx.QueryRowContext(ctx, `SELECT original_reasons @> '["exposed-after-creation"]'::jsonb FROM assessment_results WHERE attempt_id=$1`, ref.ID).Scan(&exposed); e != nil {
				return e
			}
			if exposed {
				v = correction.Evaluation{Status: correction.RetakeRequired, Reason: correction.SourceInvalid, Correct: []bool{}}
			}
		}
	}
	rid, e := correctionWriteResult(ctx, tx, j, meta, plan, b, v)
	if e != nil {
		return e
	}
	if meta.Knowledge != nil {
		learningInvalidateProjection(ctx, tx, meta.Owner, *meta.Knowledge)
	}
	if ref.Kind == correction.AssessmentEvidence && v.Status == correction.CorrectedPassed && meta.Knowledge != nil {
		q, e := learningCurrentEvidence(ctx, tx, meta.Owner, *meta.Knowledge)
		if e != nil {
			return e
		}
		if q.Qualified && q.Qualification.CorrectionID != nil {
			if e = correctionGrantEvidence(ctx, tx, meta.Owner, *meta.Knowledge, *q.Qualification, now); e != nil {
				return e
			}
			if q.Qualification.Kind == "diagnostic" {
				if _, e = learningUnlock(ctx, tx, meta.Owner, *meta.Knowledge, "assessment", ref.ID, now); e != nil {
					return e
				}
			}
			if _, e = learningUnlockSuccessors(ctx, tx, meta.Owner, *meta.Knowledge, "assessment", ref.ID, now); e != nil {
				return e
			}
		}
	}
	typ := notification.Checking
	switch v.Status {
	case correction.CorrectedPassed, correction.CorrectedFailed, correction.CheckedUnaffected:
		typ = notification.Corrected
	case correction.RetakeRequired:
		typ = notification.Retake
	case correction.ReviewMaterial:
		typ = notification.ReviewMaterial
		if ref.Kind == correction.EnrollmentEvidence {
			typ = notification.PathUnavailable
		}
	}
	if e = notificationAppend(ctx, tx, meta.Owner, notification.Source{DedupKey: "result:" + rid, Type: typ, Evidence: ref, CaseID: j.Meta.CaseID, ResultID: &rid}); e != nil {
		return e
	}
	// Content and owner locks remain held through the final database-time fence.
	account, e = readAccount(ctx, tx, meta.Owner, false)
	if e != nil {
		return e
	}
	if e = correction.Authorize(account.User, correction.ReadOwnAction); e != nil {
		return e
	}
	return correctionAdvanceCursor(ctx, tx, l, ref)
}
func (s *Store) ProcessCorrectionJob(ctx context.Context, l correction.Lease, limit int) (correction.Batch, error) {
	out := correction.Batch{Claimed: true, State: correction.Running}
	if !correctionValidLease(l) || limit < 1 || limit > 50 {
		return correction.Batch{}, auth.ErrInvalidInput
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	refs := []correction.EvidenceRef{}
	e := s.correctionSystemTx(ctx, true, func(ctx context.Context, tx *sql.Tx, now time.Time) error {
		j, e := correctionReadJob(ctx, tx, l.JobID, false)
		if e != nil {
			return e
		}
		if !correctionLeaseMatches(j, l, now) {
			return correction.ErrLeaseLost
		}
		var kind, id any
		if j.Cursor != nil {
			kind = string(j.Cursor.Kind)
			id = j.Cursor.ID
		}
		rows, e := tx.QueryContext(ctx, `WITH evidence AS (`+correctionEvidenceRowsSQL+`) SELECT e.kind,e.id::text FROM evidence e CROSS JOIN correction_cases c WHERE c.id=$1 AND `+correctionCaseAffectsSQL+` AND ($2::text IS NULL OR (e.kind,e.id)>($2::text,$3::uuid)) AND ($4::uuid IS NULL OR e.kind=$5::text AND e.id=$4 AND e.owner=$6) ORDER BY e.kind,e.id LIMIT $7`, j.Meta.CaseID, kind, id, func() any {
			if j.Evidence != nil {
				return j.Evidence.ID
			}
			return nil
		}(), func() any {
			if j.Evidence != nil {
				return j.Evidence.Kind
			}
			return nil
		}(), j.Owner, limit+1)
		if e != nil {
			return e
		}
		for rows.Next() {
			var ref correction.EvidenceRef
			if e = rows.Scan(&ref.Kind, &ref.ID); e != nil {
				rows.Close()
				return e
			}
			refs = append(refs, ref)
		}
		e = rows.Err()
		rows.Close()
		return e
	})
	if e != nil {
		return out, e
	}
	out.Remaining = len(refs) > limit
	if out.Remaining {
		refs = refs[:limit]
	}
	for _, ref := range refs {
		e = s.correctionSystemTx(ctx, true, func(ctx context.Context, tx *sql.Tx, now time.Time) error {
			return correctionProcessEvidence(ctx, tx, l, ref, now)
		})
		if e != nil {
			return out, e
		}
		out.Processed++
	}
	out.State = correction.Succeeded
	if out.Remaining {
		out.State = correction.Queued
	}
	if e = s.FinishCorrectionJob(ctx, l, out.State, ""); e != nil {
		return out, e
	}
	return out, nil
}
