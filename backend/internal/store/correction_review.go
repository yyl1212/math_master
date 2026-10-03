package store

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

func (s *Store) DecideCorrectionPlan(ctx context.Context, a question.Access, ref correction.PlanRef, in correction.DecisionInput) (correction.Envelope[correction.Receipt], error) {
	var out correction.Envelope[correction.Receipt]
	if !correction.ValidPlanRef(ref) || correction.ValidateDecision(in) != nil {
		return out, auth.ErrInvalidInput
	}
	action := string(correction.DecidePlanAction)
	resource := correctionPlanResource(ref)
	digest, e := correctionCommandDigest(action, resource, in)
	if e != nil {
		return out, e
	}
	ctx = correctionWithCommand(ctx, action, resource, a.IdempotencyKey)
	e = s.correctionTx(ctx, a, correction.DecidePlanAction, nil, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		out.ActorID = u.ID
		receipt, found, e := correctionReplay(ctx, tx, u.ID, action, resource, a.IdempotencyKey, digest)
		if e != nil {
			return e
		}
		if found {
			out.Data = receipt
			return nil
		}
		p, e := correctionReadPlan(ctx, tx, ref, true)
		if e != nil {
			return e
		}
		if p.Meta.Sequence != in.ExpectedSequence || p.Meta.Sequence >= correction.MaxSequence {
			return correction.ErrConflict
		}
		status, e := correction.NextPlanState(p.Meta.Status, correction.Action(in.Decision))
		if e != nil {
			return e
		}
		if !correction.RegisteredAlgorithm(p.Meta.AlgorithmVersion) || !correction.CanReview(u, p.Creator, p.Proof.Authors) {
			return auth.ErrForbidden
		}
		// Repeat source resolution under the shared content lock. New mathematical
		// authors cannot bypass independence through an old frozen author list.
		proof, e := correctionResolveSources(ctx, tx, p.Meta.CaseID, p.Input)
		if e != nil {
			return e
		}
		if !correction.CanReview(u, p.Creator, proof.Authors) {
			return auth.ErrForbidden
		}
		var valid bool
		if e = tx.QueryRowContext(ctx, `SELECT correction_plan_proof(frozen_body,case_id,body) FROM correction_plans WHERE id=$1 AND version=$2`, ref.ID, ref.Version).Scan(&valid); e != nil {
			return e
		}
		if !valid {
			return correction.ErrSourceStale
		}
		sequence := p.Meta.Sequence + 1
		if _, e = tx.ExecContext(ctx, `UPDATE correction_plans SET status=$3,sequence=$4,updated_at=$5 WHERE id=$1 AND version=$2`, ref.ID, ref.Version, status, sequence, now); e != nil {
			return e
		}
		if e = correctionEvent(ctx, tx, "plan", ref.ID, "plan_"+string(status), u.ID, p.Meta.CaseID, &ref.Version, sequence, in); e != nil {
			return e
		}
		if status == correction.Approved {
			if e = correctionEnqueueCase(ctx, tx, p.Meta.CaseID, &ref, now); e != nil {
				return e
			}
		}
		p, e = correctionReadPlan(ctx, tx, ref, false)
		if e != nil {
			return e
		}
		out.Data = correction.Receipt{Status: 200, Plan: &p.Meta}
		now, e = dbClock(ctx, tx)
		if e != nil {
			return e
		}
		if e = correctionConsumeRate(ctx, tx, u.ID, "process", now); e != nil {
			return e
		}
		return correctionRemember(ctx, tx, u.ID, action, resource, a.IdempotencyKey, digest, out.Data)
	})
	if e != nil {
		return correction.Envelope[correction.Receipt]{}, e
	}
	return out, nil
}
