package store_test

import (
	"context"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/store"
	"testing"
	"time"
)

// 必须等待实际账户行锁，不能用账户与角色的同语句快照验证权限。
func correctionWaitAccountLock(t *testing.T, f *correctionFixture, ctx context.Context) {
	t.Helper()
	end := time.Now().Add(700 * time.Millisecond)
	for time.Now().Before(end) {
		var waiting bool
		if e := f.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND query LIKE '%FROM auth_users WHERE id=$1%' AND query LIKE '%FOR UPDATE%')`).Scan(&waiting); e != nil {
			t.Fatal(e)
		}
		if waiting {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("worker did not wait on the actual owner row")
}

func TestCorrectionWorkerAccountFreshRoleAfterWait(t *testing.T) {
	f := newCorrectionFixture(t)
	hold, e := f.db.BeginTx(f.ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer hold.Rollback()
	if _, e = hold.ExecContext(f.ctx, `SELECT id FROM auth_users WHERE id=$1 FOR UPDATE`, f.ids["author_b"]); e != nil {
		t.Fatal(e)
	}
	if _, e = hold.ExecContext(f.ctx, `DELETE FROM auth_user_roles WHERE user_id=$1 AND role='editor'`, f.ids["author_b"]); e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(f.ctx, 3*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- store.CorrectionWorkerAccountForTest(ctx, f.repo, f.ids["author_b"], correction.CreatePlanAction)
	}()
	correctionWaitAccountLock(t, f, ctx)
	if e = hold.Commit(); e != nil {
		t.Fatal(e)
	}
	select {
	case e = <-done:
		if !errors.Is(e, auth.ErrForbidden) {
			t.Fatal("worker must read the revoked role after owner wait", e)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestCorrectionWorkerAccountCancelledBatchRecovery(t *testing.T) {
	f := newCorrectionFixture(t)
	hold, e := f.db.BeginTx(f.ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer hold.Rollback()
	if _, e = hold.ExecContext(f.ctx, `SELECT id FROM auth_users WHERE id=$1 FOR UPDATE`, f.ids["learner_a"]); e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithCancel(f.ctx)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- store.CorrectionWorkerAccountForTest(ctx, f.repo, f.ids["learner_a"], correction.ReadOwnAction)
	}()
	correctionWaitAccountLock(t, f, f.ctx)
	cancel()
	select {
	case e = <-done:
		if e == nil {
			t.Fatal("cancelled owner query authorized")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("cancelled query did not release its results and transaction")
	}
	if e = hold.Rollback(); e != nil {
		t.Fatal(e)
	}
	f.db.SetMaxOpenConns(1)
	f.db.SetMaxIdleConns(1)
	retry, stop := context.WithTimeout(f.ctx, 3*time.Second)
	defer stop()
	if e = store.CorrectionWorkerAccountForTest(retry, f.repo, f.ids["learner_a"], correction.ReadOwnAction); e != nil {
		t.Fatal("single-connection pool did not recover", e)
	}
}
