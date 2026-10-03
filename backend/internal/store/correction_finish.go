package store

import (
	"context"
	"database/sql"
	"github.com/jackc/pgx/v5"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/notification"
	"time"
)

// 通知、当前账户策略和job锁后时钟仍是原顺序的独立语句；完整结果关闭后推进断点。
type correctionTailFactsKey struct{}
type correctionTailFacts struct {
	tx  *sql.Tx
	j   correctionJobRecord
	now time.Time
}

func correctionNotifyAndFinish(ctx context.Context, tx *sql.Tx, l correction.Lease, owner string, in notification.Source) (context.Context, error) {
	args, e := notificationAppendArgs(owner, in)
	if e != nil {
		return ctx, e
	}
	b := &pgx.Batch{}
	b.Queue(notificationInsertSQL, args...)
	b.Queue(`SELECT id::text,must_change_password FROM auth_users WHERE id=$1`, owner)
	b.Queue(`WITH locked_job AS MATERIALIZED (SELECT `+correctionJobColumns+` FROM correction_jobs WHERE id=$1 AND EXISTS(SELECT 1 FROM auth_users WHERE id=$2 AND NOT must_change_password) FOR UPDATE) SELECT locked_job.*,clock_timestamp() FROM locked_job`, l.JobID, owner)
	facts := &correctionTailFacts{tx: tx}
	var account accountRow
	supported, e := correctionReadBatch(ctx, tx, b, func(results pgx.BatchResults) error {
		if _, e := results.Exec(); e != nil {
			return e
		}
		if e := results.QueryRow().Scan(&account.User.ID, &account.User.MustChangePassword); e != nil {
			return workflowRowError(e)
		}
		if e := correction.Authorize(account.User, correction.ReadOwnAction); e != nil {
			return e
		}
		var e error
		facts.j, e = correctionScanJob(results.QueryRow(), &facts.now)
		return workflowRowError(e)
	})
	if !supported && e == nil {
		if e = notificationAppend(ctx, tx, owner, in); e != nil {
			return ctx, e
		}
		account, e = correctionReadWorkerOwner(ctx, tx, owner, false)
		if e != nil {
			return ctx, e
		}
		if e = correction.Authorize(account.User, correction.ReadOwnAction); e != nil {
			return ctx, e
		}
		facts.j, facts.now, e = correctionReadJobTime(ctx, tx, l.JobID)
	}
	if e != nil {
		return ctx, e
	}
	return context.WithValue(ctx, correctionTailFactsKey{}, facts), nil
}
