package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

type correctionPlanRecord struct {
	Meta    correction.PlanMetadata
	Creator string
	Input   correction.PlanInput
	Frozen  []byte
	Proof   correction.PlanProof
}

func correctionLockCase(ctx context.Context, tx *sql.Tx, id string) error {
	_, e := correctionLockCaseKind(ctx, tx, id)
	return e
}
func correctionLockCaseKind(ctx context.Context, tx *sql.Tx, id string) (correction.CaseKind, error) {
	var kind correction.CaseKind
	e := tx.QueryRowContext(ctx, `SELECT kind FROM correction_cases WHERE id=$1 AND sealed FOR UPDATE`, id).Scan(&kind)
	return kind, workflowRowError(e)
}
func correctionPlanResource(ref correction.PlanRef) string {
	return fmt.Sprintf("plans:%s:%d", ref.ID, ref.Version)
}
func correctionReadPlan(ctx context.Context, tx *sql.Tx, ref correction.PlanRef, lock bool) (correctionPlanRecord, error) {
	var p correctionPlanRecord
	p.Meta.Ref = ref
	if lock {
		var caseID string
		if e := tx.QueryRowContext(ctx, `SELECT case_id::text FROM correction_plans WHERE id=$1 AND version=$2`, ref.ID, ref.Version).Scan(&caseID); e != nil {
			return p, workflowRowError(e)
		}
		if e := correctionLockCase(ctx, tx, caseID); e != nil {
			return p, e
		}
	}
	suffix := ""
	if lock {
		suffix = " FOR UPDATE"
	}
	var raw []byte
	var digest sql.NullString
	e := tx.QueryRowContext(ctx, `SELECT case_id::text,creator_user_id::text,status,sequence,algorithm_version,jsonb_array_length(body->'mappings'),body,frozen_bytes,frozen_digest,created_at,updated_at FROM correction_plans WHERE id=$1 AND version=$2`+suffix, ref.ID, ref.Version).Scan(&p.Meta.CaseID, &p.Creator, &p.Meta.Status, &p.Meta.Sequence, &p.Meta.AlgorithmVersion, &p.Meta.MappingCount, &raw, &p.Frozen, &digest, &p.Meta.CreatedAt, &p.Meta.UpdatedAt)
	if e != nil {
		return p, workflowRowError(e)
	}
	if json.Unmarshal(raw, &p.Input) != nil {
		return p, auth.ErrUnavailable
	}
	if digest.Valid {
		p.Meta.Digest = &digest.String
	}
	p.Meta.CreatedAt = p.Meta.CreatedAt.UTC()
	p.Meta.UpdatedAt = p.Meta.UpdatedAt.UTC()
	if p.Frozen != nil {
		p.Proof, e = correctionDecodeFrozenPlan(p.Frozen, p.Input)
		if e != nil {
			return p, e
		}
	}
	return p, nil
}
func correctionDecodeFrozenPlan(frozen []byte, input correction.PlanInput) (correction.PlanProof, error) {
	if frozen == nil {
		return correction.PlanProof{}, nil
	}
	var env struct {
		Purpose string `json:"purpose"`
		Body    struct {
			Input correction.PlanInput `json:"input"`
			Proof correction.PlanProof `json:"proof"`
		} `json:"body"`
	}
	if json.Unmarshal(frozen, &env) != nil || env.Purpose != "correction-plan-v1" || body(env.Body.Input) != body(input) {
		return correction.PlanProof{}, auth.ErrUnavailable
	}
	return env.Body.Proof, nil
}

func (s *Store) CreateCorrectionPlan(ctx context.Context, a question.Access, caseID string, in correction.PlanInput) (correction.Envelope[correction.Receipt], error) {
	var out correction.Envelope[correction.Receipt]
	if !question.ValidID(caseID) || correction.ValidatePlan(in) != nil || in.ExpectedSequence != nil {
		return out, auth.ErrInvalidInput
	}
	action := string(correction.CreatePlanAction)
	resource := "cases:" + caseID
	digest, e := correctionCommandDigest(action, resource, in)
	if e != nil {
		return out, e
	}
	ctx = correctionWithCommand(ctx, action, resource, a.IdempotencyKey)
	e = s.correctionTx(ctx, a, correction.CreatePlanAction, nil, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		out.ActorID = u.ID
		receipt, found, e := correctionReplay(ctx, tx, u.ID, action, resource, a.IdempotencyKey, digest)
		if e != nil {
			return e
		}
		if found {
			out.Data = receipt
			return nil
		}
		if e = correctionLockCase(ctx, tx, caseID); e != nil {
			return e
		}
		ref := correction.PlanRef{Version: 1}
		var parentID, parentVersion any
		if in.Parent != nil {
			parent, e := correctionReadPlan(ctx, tx, *in.Parent, true)
			if e != nil {
				return e
			}
			if parent.Meta.CaseID != caseID || parent.Meta.Status == correction.Draft {
				return correction.ErrConflict
			}
			if parent.Creator != u.ID {
				return auth.ErrForbidden
			}
			ref.ID = parent.Meta.Ref.ID
			parentID = ref.ID
			parentVersion = in.Parent.Version
			if e = tx.QueryRowContext(ctx, `SELECT max(version)+1 FROM correction_plans WHERE id=$1`, ref.ID).Scan(&ref.Version); e != nil {
				return e
			}
			if ref.Version > correction.MaxVersion {
				return correction.ErrConflict
			}
		} else {
			ref.ID, e = workflowID()
			if e != nil {
				return e
			}
		}
		if _, e = correctionResolveSources(ctx, tx, caseID, in); e != nil {
			return e
		}
		_, e = tx.ExecContext(ctx, `INSERT INTO correction_plans(id,version,case_id,parent_id,parent_version,creator_user_id,algorithm_version,body,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$9)`, ref.ID, ref.Version, caseID, parentID, parentVersion, u.ID, in.AlgorithmVersion, body(in), now)
		if e != nil {
			return e
		}
		if e = correctionEvent(ctx, tx, "plan", ref.ID, "plan_created", u.ID, caseID, &ref.Version, 1, map[string]any{"mappingCount": len(in.Mappings)}); e != nil {
			return e
		}
		p, e := correctionReadPlan(ctx, tx, ref, false)
		if e != nil {
			return e
		}
		out.Data = correction.Receipt{Status: 201, Plan: &p.Meta}
		now, e = dbClock(ctx, tx)
		if e != nil {
			return e
		}
		if e = correctionConsumeRate(ctx, tx, u.ID, "create", now); e != nil {
			return e
		}
		return correctionRemember(ctx, tx, u.ID, action, resource, a.IdempotencyKey, digest, out.Data)
	})
	if e != nil {
		return correction.Envelope[correction.Receipt]{}, e
	}
	return out, nil
}
func (s *Store) UpdateCorrectionPlan(ctx context.Context, a question.Access, ref correction.PlanRef, in correction.PlanInput) (correction.Envelope[correction.Receipt], error) {
	var out correction.Envelope[correction.Receipt]
	if !correction.ValidPlanRef(ref) || correction.ValidatePlan(in) != nil || in.ExpectedSequence == nil {
		return out, auth.ErrInvalidInput
	}
	action := string(correction.UpdatePlanAction)
	resource := correctionPlanResource(ref)
	digest, e := correctionCommandDigest(action, resource, in)
	if e != nil {
		return out, e
	}
	ctx = correctionWithCommand(ctx, action, resource, a.IdempotencyKey)
	e = s.correctionTx(ctx, a, correction.UpdatePlanAction, nil, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
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
		if p.Creator != u.ID && !auth.HasRole(u, auth.RoleAdmin) {
			return auth.ErrForbidden
		}
		if p.Meta.Sequence != *in.ExpectedSequence || p.Meta.Sequence >= correction.MaxSequence || body(p.Input.Parent) != body(in.Parent) {
			return correction.ErrConflict
		}
		if _, e = correction.NextPlanState(p.Meta.Status, correction.UpdatePlanAction); e != nil {
			return e
		}
		if _, e = correctionResolveSources(ctx, tx, p.Meta.CaseID, in); e != nil {
			return e
		}
		sequence := p.Meta.Sequence + 1
		if _, e = tx.ExecContext(ctx, `UPDATE correction_plans SET body=$3,sequence=$4,updated_at=$5 WHERE id=$1 AND version=$2`, ref.ID, ref.Version, body(in), sequence, now); e != nil {
			return e
		}
		if e = correctionEvent(ctx, tx, "plan", ref.ID, "plan_updated", u.ID, p.Meta.CaseID, &ref.Version, sequence, map[string]any{"mappingCount": len(in.Mappings)}); e != nil {
			return e
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
func (s *Store) SubmitCorrectionPlan(ctx context.Context, a question.Access, ref correction.PlanRef, in correction.SubmitInput) (correction.Envelope[correction.Receipt], error) {
	var out correction.Envelope[correction.Receipt]
	if !correction.ValidPlanRef(ref) || correction.ValidateSubmit(in) != nil {
		return out, auth.ErrInvalidInput
	}
	action := string(correction.SubmitPlanAction)
	resource := correctionPlanResource(ref)
	digest, e := correctionCommandDigest(action, resource, in)
	if e != nil {
		return out, e
	}
	ctx = correctionWithCommand(ctx, action, resource, a.IdempotencyKey)
	e = s.correctionTx(ctx, a, correction.SubmitPlanAction, nil, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
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
		if p.Creator != u.ID {
			return auth.ErrForbidden
		}
		if p.Meta.Sequence != in.ExpectedSequence || p.Meta.Sequence >= correction.MaxSequence {
			return correction.ErrConflict
		}
		status, e := correction.NextPlanState(p.Meta.Status, correction.SubmitPlanAction)
		if e != nil {
			return e
		}
		proof, e := correctionResolveSources(ctx, tx, p.Meta.CaseID, p.Input)
		if e != nil {
			return e
		}
		if e = correctionPlanEditors(ctx, tx, ref, &proof); e != nil {
			return e
		}
		frozen, h, e := correction.Canonical("correction-plan-v1", struct {
			Input correction.PlanInput `json:"input"`
			Proof correction.PlanProof `json:"proof"`
		}{p.Input, proof})
		if e != nil {
			return e
		}
		if len(frozen) > correction.MaxResponseBytes {
			return auth.ErrInvalidInput
		}
		sequence := p.Meta.Sequence + 1
		if _, e = tx.ExecContext(ctx, `UPDATE correction_plans SET status=$3,sequence=$4,sealed=true,frozen_body=$5,frozen_bytes=$6,frozen_digest=$7,updated_at=$8 WHERE id=$1 AND version=$2`, ref.ID, ref.Version, status, sequence, string(frozen), frozen, h, now); e != nil {
			return e
		}
		if e = correctionEvent(ctx, tx, "plan", ref.ID, "plan_submitted", u.ID, p.Meta.CaseID, &ref.Version, sequence, map[string]any{"digest": h}); e != nil {
			return e
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

func correctionPlanEditors(ctx context.Context, tx *sql.Tx, ref correction.PlanRef, proof *correction.PlanProof) error {
	authors := map[string]bool{}
	for _, id := range proof.Authors {
		authors[id] = true
	}
	rows, e := tx.QueryContext(ctx, `SELECT DISTINCT actor_user_id::text FROM correction_events WHERE subject_kind='plan' AND subject_id=$1 AND subject_version=$2 AND kind IN ('plan_created','plan_updated') AND actor_user_id IS NOT NULL`, ref.ID, ref.Version)
	if e != nil {
		return e
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			return e
		}
		authors[id] = true
	}
	if e = rows.Err(); e != nil {
		return e
	}
	proof.Authors = sortedQuestionAuthors(authors)
	return nil
}
