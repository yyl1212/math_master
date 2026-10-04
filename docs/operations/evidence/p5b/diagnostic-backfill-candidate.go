package store

import (
	"context"
	"database/sql"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"time"
)

// Metadata only: no answer, prompt or frozen mathematical body is projected.
const correctionEvidenceRowsSQL = `SELECT 'assessment'::text kind,id,owner_user_id owner,created_at,knowledge_id kid,knowledge_version kv,knowledge_sha256 kh,rule_version,state='submitted' terminal FROM assessment_attempts WHERE sealed
 UNION ALL SELECT 'practice',id,owner_user_id,created_at,knowledge_id,knowledge_version,knowledge_sha256,(seal#>>'{body,ruleVersion}')::integer,state='answered' FROM practice_attempts
 UNION ALL SELECT 'learning-event',id,owner_user_id,recorded_at,knowledge_id,knowledge_version,knowledge_sha256,NULL::integer,true FROM learning_events
 UNION ALL SELECT 'enrollment',id,owner_user_id,created_at,NULL::text,NULL::integer,NULL::text,NULL::integer,true FROM learning_path_enrollments`
const correctionGradingCaseAffectsSQL = `c.sealed AND c.kind='grading_rule' AND e.kind IN ('assessment','practice') AND e.rule_version=c.rule_version AND e.created_at<=c.cutoff AND (c.scope_kind='all' OR e.kid=c.knowledge_id AND e.kv=c.knowledge_version AND e.kh=c.knowledge_sha256)`
const correctionWithdrawalCaseAffectsSQL = `c.sealed AND c.kind='withdrawal' AND (
 EXISTS(SELECT 1 FROM learning_evidence_dependencies d WHERE d.evidence_kind=e.kind AND d.evidence_id=e.id AND d.owner_user_id=e.owner AND ((c.withdrawal_space='content' AND EXISTS(SELECT 1 FROM content_withdrawals w WHERE w.id=c.withdrawal_id AND w.kind=d.kind AND w.sha256=d.sha256 AND (d.kind='asset' OR w.target_id=d.id AND w.target_version=d.version))) OR (c.withdrawal_space='question' AND EXISTS(SELECT 1 FROM question_withdrawals w WHERE w.id=c.withdrawal_id AND w.kind=d.kind AND w.target_id=d.id AND w.target_version=d.version AND w.sha256=d.sha256))))
 OR EXISTS(SELECT 1 FROM correction_results r JOIN correction_dependencies d ON d.result_id=r.id AND d.role='effective' WHERE r.sealed AND r.owner_user_id=e.owner AND r.evidence_kind=e.kind AND r.evidence_id=e.id AND ((c.withdrawal_space='content' AND EXISTS(SELECT 1 FROM content_withdrawals w WHERE w.id=c.withdrawal_id AND w.kind=d.kind AND w.sha256=d.sha256 AND (d.kind='asset' OR w.target_id=d.id AND w.target_version=d.version))) OR (c.withdrawal_space='question' AND EXISTS(SELECT 1 FROM question_withdrawals w WHERE w.id=c.withdrawal_id AND w.kind=d.kind AND w.target_id=d.id AND w.target_version=d.version AND w.sha256=d.sha256))))
 OR (e.kind='enrollment' AND c.withdrawal_space='content' AND EXISTS(SELECT 1 FROM content_withdrawals w JOIN learning_path_enrollments p ON p.id=e.id AND p.owner_user_id=e.owner WHERE w.id=c.withdrawal_id AND ((w.kind='path' AND p.path_id=w.target_id AND p.path_version=w.target_version AND p.path_sha256=w.sha256) OR (w.kind='knowledge' AND EXISTS(SELECT 1 FROM learning_path_nodes n WHERE n.enrollment_id=p.id AND n.knowledge_id=w.target_id AND n.knowledge_version=w.target_version AND n.knowledge_sha256=w.sha256)))))
 )`
const correctionCaseAffectsSQL = `((` + correctionGradingCaseAffectsSQL + `) OR (` + correctionWithdrawalCaseAffectsSQL + `))`

// The case kind is immutable and checked inside the transaction. Selecting its
// exact predicate avoids planning unrelated withdrawal joins for grading scans.
func correctionCasePredicateSQL(kind correction.CaseKind) string {
	switch kind {
	case correction.GradingRuleCase:
		return correctionGradingCaseAffectsSQL
	case correction.WithdrawalCase:
		return correctionWithdrawalCaseAffectsSQL
	default:
		return `FALSE`
	}
}

func correctionEnqueueOneTerminal(ctx context.Context, tx *sql.Tx, caseID, owner string, ref correction.EvidenceRef, now time.Time) (bool, error) {
	id, e := workflowID()
	if e != nil {
		return false, e
	}
	source := "terminal:" + caseID + ":" + string(ref.Kind) + ":" + ref.ID
	e = tx.QueryRowContext(ctx, `INSERT INTO correction_jobs(id,source_key,case_id,type,evidence_kind,evidence_id,owner_user_id,created_at,next_run_at) VALUES($1,$2,$3,'attempt_terminal',$4,$5,$6,$7,$7) ON CONFLICT(source_key) DO NOTHING RETURNING id::text`, id, source, caseID, ref.Kind, ref.ID, owner, now).Scan(&id)
	if errors.Is(e, sql.ErrNoRows) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	e = correctionEvent(ctx, tx, "job", id, "job_created", "", caseID, nil, 1, map[string]any{"evidence": ref})
	return e == nil, e
}
func (s *Store) BackfillCorrections(ctx context.Context, limit int) (int, error) {
	if limit < 1 || limit > 50 {
		return 0, auth.ErrInvalidInput
	}
	total := 0
	e := s.correctionSystemTx(ctx, true, func(ctx context.Context, tx *sql.Tx, now time.Time) error {
		rows, e := tx.QueryContext(ctx, `WITH candidates AS (SELECT id::text case_id,NULL::text space,NULL::text source_id,created_at,last_backfill_at FROM correction_cases WHERE sealed UNION ALL SELECT NULL,'content',w.id::text,w.created_at,NULL::timestamptz FROM content_withdrawals w WHERE NOT EXISTS(SELECT 1 FROM correction_cases c WHERE c.withdrawal_space='content' AND c.withdrawal_id=w.id) UNION ALL SELECT NULL,'question',w.id::text,w.created_at,NULL::timestamptz FROM question_withdrawals w WHERE NOT EXISTS(SELECT 1 FROM correction_cases c WHERE c.withdrawal_space='question' AND c.withdrawal_id=w.id)) SELECT case_id,space,source_id FROM candidates ORDER BY last_backfill_at NULLS FIRST,created_at,coalesce(case_id,source_id),space LIMIT $1`, limit)
		if e != nil {
			return e
		}
		type candidate struct{ caseID, space, sourceID sql.NullString }
		candidates := []candidate{}
		for rows.Next() {
			var c candidate
			if e = rows.Scan(&c.caseID, &c.space, &c.sourceID); e != nil {
				rows.Close()
				return e
			}
			candidates = append(candidates, c)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		for _, candidate := range candidates {
			if total >= limit {
				break
			}
			caseID := candidate.caseID.String
			if !candidate.caseID.Valid {
				if e = correctionEnqueueWithdrawal(ctx, tx, candidate.space.String, candidate.sourceID.String, now); e != nil {
					return e
				}
				if e = tx.QueryRowContext(ctx, `SELECT id::text FROM correction_cases WHERE withdrawal_space=$1 AND withdrawal_id=$2 AND sealed`, candidate.space.String, candidate.sourceID.String).Scan(&caseID); e != nil {
					return e
				}
				total++
			} else {
				var locked string
				e = tx.QueryRowContext(ctx, `SELECT id::text FROM correction_cases WHERE id=$1 AND sealed FOR UPDATE SKIP LOCKED`, caseID).Scan(&locked)
				if errors.Is(e, sql.ErrNoRows) {
					continue
				}
				if e != nil {
					return e
				}
				var exists bool
				if e = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM correction_jobs WHERE source_key=$1)`, "case:"+caseID).Scan(&exists); e != nil {
					return e
				}
				if !exists {
					if e = correctionEnqueueCase(ctx, tx, caseID, nil, now); e != nil {
						return e
					}
					total++
				}
			}
			if total < limit {
				plans, e := tx.QueryContext(ctx, `SELECT id::text,version FROM correction_plans p WHERE case_id=$1 AND status='approved' AND sealed AND NOT EXISTS(SELECT 1 FROM correction_jobs j WHERE j.source_key='plan:'||p.id::text||':'||p.version::text) ORDER BY created_at,id,version LIMIT $2`, caseID, limit-total)
				if e != nil {
					return e
				}
				refs := []correction.PlanRef{}
				for plans.Next() {
					var p correction.PlanRef
					if e = plans.Scan(&p.ID, &p.Version); e != nil {
						plans.Close()
						return e
					}
					refs = append(refs, p)
				}
				e = plans.Err()
				plans.Close()
				if e != nil {
					return e
				}
				for _, p := range refs {
					if e = correctionEnqueueCase(ctx, tx, caseID, &p, now); e != nil {
						return e
					}
					total++
				}
			}
			if total < limit {
				c, e := correctionReadCase(ctx, tx, caseID)
				if e != nil {
					return e
				}
				items, e := tx.QueryContext(ctx, `WITH evidence AS (`+correctionEvidenceRowsSQL+`) SELECT e.kind,e.id::text,e.owner::text FROM correction_cases c CROSS JOIN LATERAL (SELECT e.kind,e.id,e.owner FROM evidence e WHERE e.terminal AND e.kind IN ('assessment','practice') AND `+correctionCasePredicateSQL(c.Kind)+` AND NOT EXISTS(SELECT 1 FROM correction_jobs j WHERE j.source_key='terminal:'||c.id::text||':'||e.kind||':'||e.id::text) ORDER BY e.kind,e.id LIMIT $2 OFFSET 0) e WHERE c.id=$1 ORDER BY e.kind,e.id`, caseID, limit-total)
				if e != nil {
					return e
				}
				type owned struct {
					ref   correction.EvidenceRef
					owner string
				}
				batch := []owned{}
				for items.Next() {
					var item owned
					if e = items.Scan(&item.ref.Kind, &item.ref.ID, &item.owner); e != nil {
						items.Close()
						return e
					}
					batch = append(batch, item)
				}
				e = items.Err()
				items.Close()
				if e != nil {
					return e
				}
				for _, item := range batch {
					created, e := correctionEnqueueOneTerminal(ctx, tx, caseID, item.owner, item.ref, now)
					if e != nil {
						return e
					}
					if created {
						total++
					}
				}
			}
			if _, e = tx.ExecContext(ctx, `UPDATE correction_cases SET last_backfill_at=clock_timestamp() WHERE id=$1`, caseID); e != nil {
				return e
			}
		}
		return nil
	})
	if e != nil {
		return 0, e
	}
	return total, nil
}
