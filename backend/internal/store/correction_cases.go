package store

import (
	"context"
	"database/sql"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

func correctionEvent(ctx context.Context, tx *sql.Tx, subject, id, kind, actor, caseID string, version *int, sequence int64, value any) error {
	raw, h, e := correction.Canonical("correction-command-v1", value)
	if e != nil {
		return e
	}
	eid, e := workflowID()
	if e != nil {
		return e
	}
	var actorID any
	if actor != "" {
		actorID = actor
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO correction_events(id,subject_kind,subject_id,subject_version,case_id,kind,sequence,actor_user_id,body,body_bytes,body_digest) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, eid, subject, id, version, caseID, kind, sequence, actorID, string(raw), raw, h)
	return e
}

const correctionCaseSQL = `SELECT id::text,kind,withdrawal_space,withdrawal_id::text,rule_version,scope_kind,knowledge_id,knowledge_version,knowledge_sha256,cutoff,sequence,created_at,EXISTS(SELECT 1 FROM correction_plans p WHERE p.case_id=c.id AND p.status='approved' AND p.sealed AND p.algorithm_version=1) FROM correction_cases c WHERE c.id=$1 AND c.sealed`

func correctionReadCase(ctx context.Context, tx *sql.Tx, id string) (correction.CaseMetadata, error) {
	var c correction.CaseMetadata
	var space, wid, scope, kid, kh sql.NullString
	var rule, kv sql.NullInt64
	var cutoff sql.NullTime
	e := tx.QueryRowContext(ctx, correctionCaseSQL, id).Scan(&c.ID, &c.Kind, &space, &wid, &rule, &scope, &kid, &kv, &kh, &cutoff, &c.Sequence, &c.CreatedAt, &c.HasApprovedPlan)
	if e != nil {
		return c, workflowRowError(e)
	}
	if wid.Valid {
		c.Withdrawal = &correction.WithdrawalRef{Space: space.String, ID: wid.String}
	}
	if rule.Valid {
		c.Rule = &correction.RuleScope{RuleVersion: int(rule.Int64), Kind: scope.String}
		if kid.Valid {
			c.Rule.Knowledge = &question.Identity{ID: kid.String, Version: int(kv.Int64), SHA256: kh.String}
		}
	}
	if cutoff.Valid {
		v := cutoff.Time.UTC()
		c.Cutoff = &v
	}
	c.CreatedAt = c.CreatedAt.UTC()
	return c, nil
}
func (s *Store) CreateCorrectionCase(ctx context.Context, a question.Access, in correction.CaseInput) (correction.Envelope[correction.Receipt], error) {
	var out correction.Envelope[correction.Receipt]
	if e := correction.ValidateCase(in); e != nil {
		return out, e
	}
	action := string(correction.CreateCaseAction)
	digest, e := correctionCommandDigest(action, "cases", in)
	if e != nil {
		return out, e
	}
	ctx = correctionWithCommand(ctx, action, "cases", a.IdempotencyKey)
	e = s.correctionTx(ctx, a, correction.CreateCaseAction, nil, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
		out.ActorID = u.ID
		r, found, e := correctionReplay(ctx, tx, u.ID, action, "cases", a.IdempotencyKey, digest)
		if e != nil {
			return e
		}
		if found {
			out.Data = r
			return nil
		}
		if in.Withdrawal != nil {
			if e = correctionWithdrawalSource(ctx, tx, *in.Withdrawal); e != nil {
				return e
			}
		}
		if in.Rule != nil && in.Rule.Knowledge != nil {
			var exists bool
			k := in.Rule.Knowledge
			if e = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM knowledge_versions WHERE id=$1 AND version=$2 AND sha256=$3)`, k.ID, k.Version, k.SHA256).Scan(&exists); e != nil {
				return e
			}
			if !exists {
				return correction.ErrSourceStale
			}
		}
		id, e := workflowID()
		if e != nil {
			return e
		}
		var space, wid, cwid, qwid, rule, scope, kid, kv, kh, cutoff any
		if w := in.Withdrawal; w != nil {
			space = w.Space
			wid = w.ID
			if w.Space == "content" {
				cwid = w.ID
			} else {
				qwid = w.ID
			}
		}
		if v := in.Rule; v != nil {
			rule = v.RuleVersion
			scope = v.Kind
			cutoff = now
			if v.Knowledge != nil {
				kid = v.Knowledge.ID
				kv = v.Knowledge.Version
				kh = v.Knowledge.SHA256
			}
		}
		_, e = tx.ExecContext(ctx, `INSERT INTO correction_cases(id,kind,withdrawal_space,withdrawal_id,content_withdrawal_id,question_withdrawal_id,rule_version,scope_kind,knowledge_id,knowledge_version,knowledge_sha256,cutoff,creator_user_id,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`, id, in.Kind, space, wid, cwid, qwid, rule, scope, kid, kv, kh, cutoff, u.ID, now)
		if e != nil {
			return e
		}
		if e = correctionEvent(ctx, tx, "case", id, "case_registered", u.ID, id, nil, 1, in); e != nil {
			return e
		}
		if _, e = tx.ExecContext(ctx, `UPDATE correction_cases SET sealed=true WHERE id=$1`, id); e != nil {
			return e
		}
		c, e := correctionReadCase(ctx, tx, id)
		if e != nil {
			return e
		}
		if e = correctionEnqueueCase(ctx, tx, id, nil, c.CreatedAt); e != nil {
			return e
		}
		now, e = dbClock(ctx, tx)
		if e != nil {
			return e
		}
		if e = correctionConsumeRate(ctx, tx, u.ID, "create", now); e != nil {
			return e
		}
		out.Data = correction.Receipt{Status: 201, Case: &c}
		return correctionRemember(ctx, tx, u.ID, action, "cases", a.IdempotencyKey, digest, out.Data)
	})
	if e != nil {
		return correction.Envelope[correction.Receipt]{}, e
	}
	return out, nil
}
func correctionEnqueueCase(ctx context.Context, tx *sql.Tx, caseID string, plan *correction.PlanRef, now time.Time) error {
	c, e := correctionReadCase(ctx, tx, caseID)
	if e != nil {
		return e
	}
	source := "case:" + caseID
	kind := correction.WithdrawalImpact
	var pid, pv any
	if c.Kind == correction.GradingRuleCase {
		kind = correction.RuleImpact
	}
	if plan != nil {
		pid = plan.ID
		pv = plan.Version
		source = "plan:" + plan.ID + ":" + fmtVersion(plan.Version)
		kind = correction.ApprovedPlan
	}
	id, e := workflowID()
	if e != nil {
		return e
	}
	e = tx.QueryRowContext(ctx, `INSERT INTO correction_jobs(id,source_key,case_id,plan_id,plan_version,type,created_at,next_run_at) VALUES($1,$2,$3,$4,$5,$6,$7,$7) ON CONFLICT(source_key) DO NOTHING RETURNING id::text`, id, source, caseID, pid, pv, kind, now).Scan(&id)
	if errors.Is(e, sql.ErrNoRows) {
		return nil
	}
	if e != nil {
		return e
	}
	return correctionEvent(ctx, tx, "job", id, "job_created", "", caseID, nil, 1, map[string]any{"sourceKey": source})
}
