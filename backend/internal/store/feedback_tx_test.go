package store

import (
	"context"
	"database/sql"
	"errors"
	"github.com/pressly/goose/v3"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"os"
	"testing"
	"time"
)

func TestFeedbackConfiguration(t *testing.T) {
	db := testutil.Database(t)
	p, e := goose.NewProvider(goose.DialectPostgres, db, os.DirFS("../../../db/migrations"))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = p.UpTo(context.Background(), 6); e != nil {
		t.Fatal(e)
	}
	check := func(want bool) {
		t.Helper()
		tx, _ := db.Begin()
		defer tx.Rollback()
		got, e := feedbackConfigured(context.Background(), tx)
		if got != want || (want && e != nil) || (!want && !errors.Is(e, feedback.ErrNotConfigured)) {
			t.Fatal(got, e)
		}
	}
	check(false)
	if _, e = p.Up(context.Background()); e != nil {
		t.Fatal(e)
	}
	check(true)
	if _, e = db.Exec(`ALTER TABLE feedback_events RENAME TO broken_feedback_events`); e != nil {
		t.Fatal(e)
	}
	check(false)
}
func TestFeedbackCommitIdentity(t *testing.T) {
	for _, c := range []struct {
		name, q string
		want    error
	}{
		{"role", `DELETE FROM auth_user_roles WHERE role='reviewer'`, auth.ErrForbidden},
		{"session", `UPDATE auth_sessions SET revoked_at=clock_timestamp()`, auth.ErrAuthenticationRequired},
		{"expires", `UPDATE auth_sessions SET absolute_expires_at=clock_timestamp()`, auth.ErrAuthenticationRequired},
		{"credential", `UPDATE auth_users SET credential_version=credential_version+1`, auth.ErrAuthenticationRequired},
		{"password", `UPDATE auth_users SET must_change_password=true`, auth.ErrPasswordChangeRequired},
		{"csrf", `UPDATE auth_sessions SET csrf=decode(repeat('03',32),'hex')`, auth.ErrCSRF},
	} {
		t.Run(c.name, func(t *testing.T) {
			s, db, a, id := workflowGuardFixture(t)
			if _, e := db.Exec(`INSERT INTO auth_user_roles VALUES($1,'reviewer')`, id); e != nil {
				t.Fatal(e)
			}
			called := false
			e := s.feedbackTx(context.Background(), a, feedback.TransitionAction, nil, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
				called = true
				if _, e := tx.ExecContext(ctx, c.q); e != nil {
					return e
				}
				_, e := tx.ExecContext(ctx, `INSERT INTO feedback_rate_limits(actor_user_id,scope,consumed_at) VALUES($1,'transition',$2)`, u.ID, now)
				return e
			})
			if !called || !errors.Is(e, c.want) {
				t.Fatal("commit identity", called, e)
			}
			var n int
			if e = db.QueryRow(`SELECT count(*) FROM feedback_rate_limits`).Scan(&n); e != nil || n != 0 {
				t.Fatal("partial commit", n, e)
			}
		})
	}
	t.Run("lock wait expires", func(t *testing.T) {
		s, db, a, _ := workflowGuardFixture(t)
		db.Exec(`UPDATE auth_sessions SET absolute_expires_at=clock_timestamp()+interval '200 milliseconds'`)
		block, _ := db.Begin()
		defer block.Rollback()
		block.Exec(`SELECT pg_advisory_xact_lock(1296127048)`)
		done := make(chan error, 1)
		go func() {
			done <- s.feedbackTx(context.Background(), a, feedback.CreateAction, nil, func(context.Context, *sql.Tx, auth.User, time.Time) error {
				t.Error("expired identity entered work")
				return nil
			})
		}()
		waitQuestionLock(t, db)
		time.Sleep(230 * time.Millisecond)
		block.Commit()
		if e := <-done; !errors.Is(e, auth.ErrAuthenticationRequired) {
			t.Fatal(e)
		}
	})
}
