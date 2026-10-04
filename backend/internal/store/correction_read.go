package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

func correctionResponseSize(v any) error {
	raw, e := json.Marshal(v)
	if e != nil {
		return e
	}
	if len(raw) > correction.MaxResponseBytes {
		return question.ErrLimitExceeded
	}
	return nil
}

type correctionPageKey struct {
	ID        string
	Version   int
	CreatedAt time.Time
}

// query and its aliases are internal constants. Ownership predicates precede
// cursor bounds; a cursor from another account never widens permission.
func correctionPageKeys(ctx context.Context, tx *sql.Tx, query string, args []any, q correction.Query) ([]correctionPageKey, *string, error) {
	keys := []correctionPageKey{}
	if e := correction.ValidateQuery(q); e != nil {
		return nil, nil, e
	}
	limit := q.Limit
	if limit == 0 {
		limit = correction.DefaultLimit
	}
	var at any
	var id any
	if q.Cursor != "" {
		c, e := correction.DecodeCursor(q.Cursor)
		if e != nil {
			return nil, nil, e
		}
		at = c.CreatedAt
		id = c.ID
	}
	n := len(args)
	query += fmt.Sprintf(` AND ($%d::timestamptz IS NULL OR (t.created_at,t.id)<($%d::timestamptz,$%d::uuid)) ORDER BY t.created_at DESC,t.id DESC LIMIT $%d`, n+1, n+1, n+2, n+3)
	args = append(args, at, id, limit+1)
	rows, e := tx.QueryContext(ctx, query, args...)
	if e != nil {
		return nil, nil, e
	}
	defer rows.Close()
	for rows.Next() {
		var k correctionPageKey
		if e = rows.Scan(&k.ID, &k.Version, &k.CreatedAt); e != nil {
			return nil, nil, e
		}
		k.CreatedAt = k.CreatedAt.UTC()
		keys = append(keys, k)
	}
	if e = rows.Err(); e != nil {
		return nil, nil, e
	}
	var next *string
	if len(keys) > limit {
		last := keys[limit-1]
		c, e := correction.EncodeCursor(correction.ListCursor{Version: 1, CreatedAt: last.CreatedAt, ID: last.ID})
		if e != nil {
			return nil, nil, e
		}
		next = &c
		keys = keys[:limit]
	}
	return keys, next, nil
}
func correctionList[T any](ctx context.Context, s *Store, a question.Access, action correction.Action, q correction.Query, query string, args []any, check func(context.Context, *sql.Tx, auth.User) error, read func(context.Context, *sql.Tx, correctionPageKey, auth.User) (T, error)) (correction.Envelope[correction.Page[T]], error) {
	var out correction.Envelope[correction.Page[T]]
	if e := correction.ValidateQuery(q); e != nil {
		return out, e
	}
	e := s.correctionTx(ctx, a, action, nil, func(ctx context.Context, tx *sql.Tx, u auth.User, _ time.Time) error {
		if check != nil {
			if e := check(ctx, tx, u); e != nil {
				return e
			}
		}
		actualArgs := append([]any{}, args...)
		for n, v := range actualArgs {
			if marker, ok := v.(correctionOwnerArgument); ok && bool(marker) {
				actualArgs[n] = u.ID
			}
		}
		keys, next, e := correctionPageKeys(ctx, tx, query, actualArgs, q)
		if e != nil {
			return e
		}
		out.ActorID = u.ID
		out.Data.Items = []T{}
		out.Data.NextCursor = next
		for _, k := range keys {
			v, e := read(ctx, tx, k, u)
			if e != nil {
				return e
			}
			out.Data.Items = append(out.Data.Items, v)
		}
		return correctionResponseSize(out)
	})
	if e != nil {
		return correction.Envelope[correction.Page[T]]{}, e
	}
	return out, nil
}

type correctionOwnerArgument bool

func (s *Store) ListCorrectionCases(ctx context.Context, a question.Access, q correction.Query) (correction.Envelope[correction.Page[correction.CaseMetadata]], error) {
	return correctionList(ctx, s, a, correction.ListCasesAction, q, `SELECT t.id::text,0,t.created_at FROM correction_cases t WHERE t.sealed`, nil, nil, func(ctx context.Context, tx *sql.Tx, k correctionPageKey, _ auth.User) (correction.CaseMetadata, error) {
		return correctionReadCase(ctx, tx, k.ID)
	})
}
func (s *Store) ReadCorrectionCase(ctx context.Context, a question.Access, id string) (correction.Envelope[correction.CaseMetadata], error) {
	var out correction.Envelope[correction.CaseMetadata]
	if !question.ValidID(id) {
		return out, auth.ErrInvalidInput
	}
	e := s.correctionTx(ctx, a, correction.ReadCaseAction, nil, func(ctx context.Context, tx *sql.Tx, u auth.User, _ time.Time) error {
		v, e := correctionReadCase(ctx, tx, id)
		if e != nil {
			return e
		}
		out = correction.Envelope[correction.CaseMetadata]{ActorID: u.ID, Data: v}
		return correctionResponseSize(out)
	})
	if e != nil {
		return correction.Envelope[correction.CaseMetadata]{}, e
	}
	return out, nil
}
func correctionCaseReadCheck(id string) func(context.Context, *sql.Tx, auth.User) error {
	return func(ctx context.Context, tx *sql.Tx, _ auth.User) error {
		_, e := correctionReadCase(ctx, tx, id)
		return e
	}
}
func correctionPlanMetadata(ctx context.Context, tx *sql.Tx, ref correction.PlanRef) (correction.PlanDetail, error) {
	out := correction.PlanDetail{Mappings: []correction.Mapping{}}
	out.Plan.Ref = ref
	var parent *string
	var pv *int
	e := tx.QueryRowContext(ctx, `SELECT case_id::text,status,sequence,algorithm_version,jsonb_array_length(body->'mappings'),created_at,updated_at,frozen_digest,parent_id::text,parent_version FROM correction_plans WHERE id=$1 AND version=$2`, ref.ID, ref.Version).Scan(&out.Plan.CaseID, &out.Plan.Status, &out.Plan.Sequence, &out.Plan.AlgorithmVersion, &out.Plan.MappingCount, &out.Plan.CreatedAt, &out.Plan.UpdatedAt, &out.Plan.Digest, &parent, &pv)
	if e != nil {
		return out, workflowRowError(e)
	}
	out.Plan.CreatedAt = out.Plan.CreatedAt.UTC()
	out.Plan.UpdatedAt = out.Plan.UpdatedAt.UTC()
	if parent != nil && pv != nil {
		out.Parent = &correction.PlanRef{ID: *parent, Version: *pv}
	}
	return out, nil
}
func (s *Store) ListCorrectionPlans(ctx context.Context, a question.Access, id string, q correction.Query) (correction.Envelope[correction.Page[correction.PlanMetadata]], error) {
	if !question.ValidID(id) {
		return correction.Envelope[correction.Page[correction.PlanMetadata]]{}, auth.ErrInvalidInput
	}
	return correctionList(ctx, s, a, correction.ListPlansAction, q, `SELECT t.id::text,t.version,t.created_at FROM correction_plans t WHERE t.case_id=$1`, []any{id}, correctionCaseReadCheck(id), func(ctx context.Context, tx *sql.Tx, k correctionPageKey, _ auth.User) (correction.PlanMetadata, error) {
		v, e := correctionPlanMetadata(ctx, tx, correction.PlanRef{ID: k.ID, Version: k.Version})
		return v.Plan, e
	})
}
func (s *Store) ReadCorrectionPlan(ctx context.Context, a question.Access, ref correction.PlanRef, detail bool) (correction.Envelope[correction.PlanDetail], error) {
	var out correction.Envelope[correction.PlanDetail]
	if !correction.ValidPlanRef(ref) {
		return out, auth.ErrInvalidInput
	}
	e := s.correctionTx(ctx, a, correction.ReadPlanAction, nil, func(ctx context.Context, tx *sql.Tx, u auth.User, _ time.Time) error {
		v, e := correctionPlanMetadata(ctx, tx, ref)
		if e != nil {
			return e
		}
		out = correction.Envelope[correction.PlanDetail]{ActorID: u.ID, Data: v}
		if !detail {
			return correctionResponseSize(out)
		}
		var size int
		if e = tx.QueryRowContext(ctx, `SELECT octet_length(body::text)+coalesce(octet_length(frozen_bytes),0) FROM correction_plans WHERE id=$1 AND version=$2`, ref.ID, ref.Version).Scan(&size); e != nil {
			return e
		}
		if size > correction.MaxResponseBytes {
			return question.ErrLimitExceeded
		}
		p, e := correctionReadPlan(ctx, tx, ref, false)
		if e != nil {
			return e
		}
		proof := p.Proof
		if p.Meta.Status == correction.Draft {
			proof, e = correctionResolveSources(ctx, tx, p.Meta.CaseID, p.Input)
			if e != nil {
				return e
			}
		}
		out.Data.Mappings = p.Input.Mappings
		reason := p.Input.Reason
		out.Data.Reason = &reason
		if e = tx.QueryRowContext(ctx, `SELECT body#>>'{body,reason}' FROM correction_events WHERE subject_kind='plan' AND subject_id=$1 AND subject_version=$2 AND kind IN ('plan_approved','plan_rejected') ORDER BY sequence DESC LIMIT 1`, ref.ID, ref.Version).Scan(&out.Data.DecisionReason); e != nil && e != sql.ErrNoRows {
			return e
		}
		if e = correctionResponseSize(out); e != nil {
			return e
		}
		refs := []correctionExposureRef{}
		for _, m := range proof.Mappings {
			refs = append(refs, correctionInstanceExposure(m.Original)...)
			refs = append(refs, correctionInstanceExposure(m.Replacement)...)
		}
		return correctionDetailExposure(ctx, tx, u.ID, p.Meta.CaseID, refs)
	})
	if e != nil {
		return correction.Envelope[correction.PlanDetail]{}, e
	}
	return out, nil
}
func (s *Store) ListCorrectionJobs(ctx context.Context, a question.Access, id string, q correction.Query) (correction.Envelope[correction.Page[correction.JobMetadata]], error) {
	if !question.ValidID(id) {
		return correction.Envelope[correction.Page[correction.JobMetadata]]{}, auth.ErrInvalidInput
	}
	return correctionList(ctx, s, a, correction.ListJobsAction, q, `SELECT t.id::text,0,t.created_at FROM correction_jobs t WHERE t.case_id=$1`, []any{id}, correctionCaseReadCheck(id), func(ctx context.Context, tx *sql.Tx, k correctionPageKey, _ auth.User) (correction.JobMetadata, error) {
		v, e := correctionReadJob(ctx, tx, k.ID, false)
		return v.Meta, e
	})
}
func correctionOwnResultMetadata(ctx context.Context, tx *sql.Tx, owner, id string) (correction.ResultMetadata, error) {
	var v correction.ResultMetadata
	var pid, kid, kh *string
	var pv, kv *int
	var handled []byte
	var valid bool
	e := tx.QueryRowContext(ctx, `SELECT cr.id::text,cr.case_id::text,cr.plan_id::text,cr.plan_version,cr.parent_result_id::text,cr.evidence_kind,cr.evidence_id::text,cr.knowledge_id,cr.knowledge_version,cr.knowledge_sha256,cr.status,cr.reason,cr.score,cr.passed,cr.handled_case_ids,cr.created_at,(`+correctionEffectiveResultSQL("cr")+`) FROM correction_results cr WHERE cr.id=$1 AND cr.owner_user_id=$2 AND cr.sealed`, id, owner).Scan(&v.ID, &v.CaseID, &pid, &pv, &v.ParentResultID, &v.Evidence.Kind, &v.Evidence.ID, &kid, &kv, &kh, &v.Status, &v.Reason, &v.Score, &v.Passed, &handled, &v.CreatedAt, &valid)
	if e != nil {
		return v, workflowRowError(e)
	}
	if json.Unmarshal(handled, &v.HandledCaseIDs) != nil || v.HandledCaseIDs == nil {
		return v, auth.ErrUnavailable
	}
	if pid != nil && pv != nil {
		v.Plan = &correction.PlanRef{ID: *pid, Version: *pv}
	}
	if kid != nil && kv != nil && kh != nil {
		v.Knowledge = &question.Identity{ID: *kid, Version: *kv, SHA256: *kh}
		current, e := learningIsCurrent(ctx, tx, *v.Knowledge)
		if e != nil {
			return v, e
		}
		valid = valid && current
	}
	v.Validity = assessment.Restricted
	if valid {
		v.Validity = assessment.Effective
	}
	v.CreatedAt = v.CreatedAt.UTC()
	return v, nil
}
func (s *Store) ListOwnCorrections(ctx context.Context, a question.Access, ref correction.EvidenceRef, q correction.Query) (correction.Envelope[correction.Page[correction.ResultMetadata]], error) {
	if !correction.ValidEvidence(ref, true) {
		return correction.Envelope[correction.Page[correction.ResultMetadata]]{}, auth.ErrInvalidInput
	}
	return correctionList(ctx, s, a, correction.ListOwnAction, q, `SELECT t.id::text,0,t.created_at FROM correction_results t WHERE t.owner_user_id=$1 AND t.evidence_kind=$2 AND t.evidence_id=$3 AND t.sealed`, []any{correctionOwnerArgument(true), ref.Kind, ref.ID}, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
		meta, e := correctionReadEvidenceMetadata(ctx, tx, ref)
		if e != nil {
			return e
		}
		if meta.Owner != u.ID {
			return auth.ErrNotFound
		}
		return nil
	}, func(ctx context.Context, tx *sql.Tx, k correctionPageKey, u auth.User) (correction.ResultMetadata, error) {
		return correctionOwnResultMetadata(ctx, tx, u.ID, k.ID)
	})
}
func (s *Store) ReadOwnCorrection(ctx context.Context, a question.Access, id string, detail bool) (correction.Envelope[correction.ResultDetail], error) {
	var out correction.Envelope[correction.ResultDetail]
	if !question.ValidID(id) {
		return out, auth.ErrInvalidInput
	}
	action := correction.ReadOwnAction
	if detail {
		action = correction.ReadOwnDetailAction
	}
	e := s.correctionTx(ctx, a, action, nil, func(ctx context.Context, tx *sql.Tx, u auth.User, _ time.Time) error {
		v, e := correctionOwnResultMetadata(ctx, tx, u.ID, id)
		if e != nil {
			return e
		}
		out = correction.Envelope[correction.ResultDetail]{ActorID: u.ID, Data: correction.ResultDetail{Result: v, Items: []correction.CorrectedItem{}}}
		if !detail {
			return correctionResponseSize(out)
		}
		var size int
		if e = tx.QueryRowContext(ctx, `SELECT octet_length(basis_bytes) FROM correction_results WHERE id=$1 AND owner_user_id=$2 AND sealed`, id, u.ID).Scan(&size); e != nil {
			return workflowRowError(e)
		}
		if size > correction.MaxResponseBytes {
			return question.ErrLimitExceeded
		}
		var raw, correct []byte
		if e = tx.QueryRowContext(ctx, `SELECT basis_bytes,correctness FROM correction_results WHERE id=$1 AND owner_user_id=$2 AND sealed`, id, u.ID).Scan(&raw, &correct); e != nil {
			return workflowRowError(e)
		}
		var env struct {
			Purpose string           `json:"purpose"`
			Body    correction.Basis `json:"body"`
		}
		var flags []bool
		if json.Unmarshal(raw, &env) != nil || env.Purpose != "correction-result-v1" || json.Unmarshal(correct, &flags) != nil {
			return auth.ErrUnavailable
		}
		b := env.Body
		if v.Plan != nil {
			var reason string
			if e = tx.QueryRowContext(ctx, `SELECT body->>'reason' FROM correction_plans WHERE id=$1 AND version=$2`, v.Plan.ID, v.Plan.Version).Scan(&reason); e != nil {
				return e
			}
			out.Data.PlanReason = &reason
		}
		if v.Status == correction.CorrectedPassed || v.Status == correction.CorrectedFailed {
			if len(b.OriginalItems) != len(b.EffectiveItems) || len(b.EffectiveItems) != len(b.OriginalAnswers) || len(flags) != len(b.EffectiveItems) {
				return auth.ErrUnavailable
			}
			for n, i := range b.EffectiveItems {
				choices := i.Body.Choices
				if choices == nil {
					choices = []question.Choice{}
				}
				assets := i.Body.Assets
				if assets == nil {
					assets = []question.AssetRef{}
				}
				out.Data.Items = append(out.Data.Items, correction.CorrectedItem{Position: n + 1, Original: b.OriginalItems[n].Identity, Effective: i.Identity, OriginalTemplate: b.OriginalItems[n].Template, EffectiveTemplate: i.Template, Prompt: i.Body.Prompt, Choices: choices, Answer: b.OriginalAnswers[n], Correct: flags[n], Explanation: i.Body.Explanation, Assets: assets})
			}
		}
		if e = correctionResponseSize(out); e != nil {
			return e
		}
		refs := []correctionExposureRef{}
		for _, i := range append(append([]question.Instance{}, b.OriginalItems...), b.EffectiveItems...) {
			refs = append(refs, correctionInstanceExposure(i)...)
		}
		return correctionDetailExposure(ctx, tx, u.ID, v.CaseID, refs)
	})
	if e != nil {
		return correction.Envelope[correction.ResultDetail]{}, e
	}
	return out, nil
}
