package store_test

import (
	"context"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"testing"
	"time"
)

func TestCorrectionProcessOwnerPasswordChangedDuringWait(t *testing.T) {
	f := newCorrectionFixture(t)
	f.cLegacyFailedFour()
	c := f.registerRule(nil, 1)
	f.cFinishRoots()
	f.cApproveAPI(c.ID)
	l, e := f.repo.ClaimCorrectionJob(f.ctx)
	if e != nil || l == nil {
		t.Fatal(e)
	}
	beforeResults := f.count(`SELECT count(*) FROM correction_results`)
	beforeNotices := f.count(`SELECT count(*) FROM notifications`)
	beforeEvents := f.count(`SELECT count(*) FROM correction_events`)
	hold, e := f.db.BeginTx(f.ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer hold.Rollback()
	if _, e = hold.ExecContext(f.ctx, `SELECT id FROM auth_users WHERE id=$1 FOR UPDATE`, f.ids["learner_a"]); e != nil {
		t.Fatal(e)
	}
	if _, e = hold.ExecContext(f.ctx, `UPDATE auth_users SET must_change_password=true WHERE id=$1`, f.ids["learner_a"]); e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(f.ctx, 3*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { _, e := f.repo.ProcessCorrectionJob(ctx, *l, 50); done <- e }()
	correctionWaitAccountLock(t, f, ctx)
	if e = hold.Commit(); e != nil {
		t.Fatal(e)
	}
	select {
	case e = <-done:
		if !errors.Is(e, auth.ErrPasswordChangeRequired) {
			t.Fatal("owner wait must observe new password-change requirement", e)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	if f.count(`SELECT count(*) FROM correction_results`) != beforeResults || f.count(`SELECT count(*) FROM notifications`) != beforeNotices || f.count(`SELECT count(*) FROM correction_events`) != beforeEvents || f.count(`SELECT processed_count FROM correction_jobs WHERE id=$1`, l.JobID) != 0 {
		t.Fatal("stopped owner had committed worker writes")
	}
}

func TestCorrectionProcessFinalAccountRecheckedBeforeCursor(t *testing.T) {
	f := newCorrectionFixture(t)
	f.cLegacyFailedFour()
	c := f.registerRule(nil, 1)
	f.cFinishRoots()
	f.cApproveAPI(c.ID)
	l, e := f.repo.ClaimCorrectionJob(f.ctx)
	if e != nil || l == nil {
		t.Fatal(e)
	}
	beforeResults := f.count(`SELECT count(*) FROM correction_results`)
	beforeNotices := f.count(`SELECT count(*) FROM notifications`)
	beforeEvents := f.count(`SELECT count(*) FROM correction_events`)
	f.exec(`CREATE FUNCTION test_stop_worker_owner() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN UPDATE auth_users SET must_change_password=true WHERE id=NEW.owner_user_id; RETURN NEW; END $$; CREATE TRIGGER test_stop_worker_owner AFTER INSERT ON notifications FOR EACH ROW EXECUTE FUNCTION test_stop_worker_owner()`)
	if _, e = f.repo.ProcessCorrectionJob(f.ctx, *l, 50); !errors.Is(e, auth.ErrPasswordChangeRequired) {
		t.Fatal("final owner state must stop before cursor", e)
	}
	if f.count(`SELECT count(*) FROM correction_results`) != beforeResults || f.count(`SELECT count(*) FROM notifications`) != beforeNotices || f.count(`SELECT count(*) FROM correction_events`) != beforeEvents || f.count(`SELECT processed_count FROM correction_jobs WHERE id=$1`, l.JobID) != 0 || f.count(`SELECT count(*) FROM auth_users WHERE id=$1 AND must_change_password`, f.ids["learner_a"]) != 0 {
		t.Fatal("final owner rejection did not roll back entire result/notice/cursor transaction")
	}
}
