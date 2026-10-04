package store

import (
	"context"
	"database/sql"
	"github.com/jackc/pgx/v5"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

// 每次实际核验完整schema。前置读取只绑定当前事务、job与证据，不能跨事务复用。
type correctionStartHintKey struct{}
type correctionStartHint struct {
	job string
	ref correction.EvidenceRef
}
type correctionStartFactsKey struct{}
type correctionStartFacts struct {
	tx              *sql.Tx
	hint            correctionStartHint
	j               correctionJobRecord
	meta            correctionEvidenceMetadata
	jobErr, metaErr error
}

func correctionWithStartHint(ctx context.Context, l correction.Lease, ref correction.EvidenceRef) context.Context {
	return context.WithValue(ctx, correctionStartHintKey{}, correctionStartHint{job: l.JobID, ref: ref})
}
func correctionSystemStart(ctx context.Context, tx *sql.Tx, locks bool) (bool, time.Time, *correctionStartFacts, error) {
	var n, lc, qc int
	var goose, marker bool
	e := tx.QueryRowContext(ctx, correctionSystemExistenceSQL, correctionTables, learningTables, questionTables).Scan(&n, &goose, &marker, &lc, &qc)
	if e != nil {
		return false, time.Time{}, nil, e
	}
	if !goose || !marker || n != len(correctionTables) || lc != len(learningTables) || qc != len(questionTables) {
		on, e := correctionSystemConfigured(ctx, tx)
		if e != nil || !on {
			return on, time.Time{}, nil, e
		}
		now, e := correctionSystemEnter(ctx, tx, locks)
		return on, now, nil, e
	}
	query := `WITH health(version,ever,intact) AS MATERIALIZED (` + correctionSystemHealthSQL + `),settings AS MATERIALIZED (SELECT set_config('lock_timeout','1s',true) FROM health WHERE version AND ever AND intact)`
	args := correctionSchemaIntegrityArgs()
	if locks {
		query += `,admin_fence AS MATERIALIZED(SELECT pg_advisory_xact_lock_shared($6) FROM settings),content_fence AS MATERIALIZED(SELECT pg_advisory_xact_lock_shared($7) FROM admin_fence),registration_fence AS MATERIALIZED(SELECT pg_advisory_xact_lock_shared($8) FROM content_fence)`
		args = append(args, adminLockID, int64(1296127048), correctionRegistrationLock)
		query += ` SELECT version,ever,intact,(SELECT clock_timestamp() FROM registration_fence) FROM health`
	} else {
		query += ` SELECT version,ever,intact,(SELECT clock_timestamp() FROM settings) FROM health`
	}
	var version, ever, intact bool
	var now *time.Time
	check := func() error {
		if !version || !ever || !intact || now == nil {
			return correction.ErrNotConfigured
		}
		return nil
	}
	hint, hasHint := ctx.Value(correctionStartHintKey{}).(correctionStartHint)
	if !hasHint {
		if e = tx.QueryRowContext(ctx, query, args...).Scan(&version, &ever, &intact, &now); e != nil {
			return false, time.Time{}, nil, e
		}
		if e = check(); e != nil {
			return false, time.Time{}, nil, e
		}
		return true, *now, nil, nil
	}
	facts := &correctionStartFacts{tx: tx, hint: hint, meta: correctionEvidenceMetadata{Ref: hint.ref}}
	batch := &pgx.Batch{}
	batch.Queue(query, args...)
	batch.Queue(`SELECT `+correctionJobColumns+` FROM correction_jobs WHERE id=$1`, hint.job)
	batch.Queue(`WITH evidence AS (`+correctionEvidenceRowsSQL+`) SELECT owner::text,kid,kv,kh,terminal FROM evidence WHERE kind=$1 AND id=$2`, hint.ref.Kind, hint.ref.ID)
	supported, e := correctionReadBatch(ctx, tx, batch, func(results pgx.BatchResults) error {
		if e := results.QueryRow().Scan(&version, &ever, &intact, &now); e != nil {
			return e
		}
		if e := check(); e != nil {
			return e
		}
		facts.j, facts.jobErr = correctionScanJob(results.QueryRow())
		facts.jobErr = workflowRowError(facts.jobErr)
		var kid, kh *string
		var kv *int
		facts.metaErr = results.QueryRow().Scan(&facts.meta.Owner, &kid, &kv, &kh, &facts.meta.Terminal)
		facts.metaErr = workflowRowError(facts.metaErr)
		if kid != nil && kv != nil && kh != nil {
			facts.meta.Knowledge = &question.Identity{ID: *kid, Version: *kv, SHA256: *kh}
		}
		return nil
	})
	if !supported && e == nil {
		if e = tx.QueryRowContext(ctx, query, args...).Scan(&version, &ever, &intact, &now); e != nil {
			return false, time.Time{}, nil, e
		}
		if e = check(); e != nil {
			return false, time.Time{}, nil, e
		}
		return true, *now, nil, nil
	}
	if e != nil {
		return false, time.Time{}, nil, e
	}
	return true, *now, facts, nil
}
