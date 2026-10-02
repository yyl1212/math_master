package store

import (
	"context"
	"database/sql"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/question"
	"strings"
	"testing"
	"time"
)

func TestQuestionSessionExpiresWhileWaiting(t *testing.T) {
	for _, lock := range []string{"content", "user"} {
		t.Run(lock, func(t *testing.T) {
			s, db, a, id := workflowGuardFixture(t)
			ctx := context.Background()
			if _, err := db.Exec(`UPDATE auth_sessions SET absolute_expires_at=clock_timestamp()+interval '250 milliseconds'`); err != nil {
				t.Fatal(err)
			}
			blocker, err := db.Begin()
			if err != nil {
				t.Fatal(err)
			}
			defer blocker.Rollback()
			q := `SELECT pg_advisory_xact_lock(1296127048)`
			if lock == "user" {
				q = `SELECT id FROM auth_users WHERE id='11111111-1111-4111-8111-111111111111' FOR UPDATE`
			}
			if _, err = blocker.Exec(q); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() {
				done <- s.questionTx(ctx, a, question.SaveDraftAction, []string{id}, func(context.Context, *sql.Tx, auth.User, time.Time) error {
					t.Error("expired proof reached work")
					return nil
				})
			}()
			waitQuestionLock(t, db)
			time.Sleep(300 * time.Millisecond)
			if err = blocker.Commit(); err != nil {
				t.Fatal(err)
			}
			if err = <-done; !errors.Is(err, auth.ErrAuthenticationRequired) {
				t.Fatalf("expired proof accepted: %v", err)
			}
		})
	}
	for _, tc := range []struct {
		name, sql string
		want      error
	}{
		{"role", `DELETE FROM auth_user_roles WHERE role='editor'`, auth.ErrForbidden},
		{"credential", `UPDATE auth_users SET credential_version=credential_version+1`, auth.ErrAuthenticationRequired},
		{"csrf", `UPDATE auth_sessions SET csrf=decode(repeat('03',32),'hex')`, auth.ErrCSRF},
		{"password", `UPDATE auth_users SET must_change_password=true`, auth.ErrPasswordChangeRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, db, a, _ := workflowGuardFixture(t)
			blocker, err := db.Begin()
			if err != nil {
				t.Fatal(err)
			}
			defer blocker.Rollback()
			if _, err = blocker.Exec(`SELECT id FROM auth_users FOR UPDATE`); err != nil {
				t.Fatal(err)
			}
			if _, err = blocker.Exec(tc.sql); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() {
				done <- s.questionTx(context.Background(), a, question.SaveDraftAction, nil, func(context.Context, *sql.Tx, auth.User, time.Time) error {
					t.Error("invalid proof reached work")
					return nil
				})
			}()
			waitQuestionLock(t, db)
			if err = blocker.Commit(); err != nil {
				t.Fatal(err)
			}
			err = <-done
			if !errors.Is(err, tc.want) {
				t.Fatal(err)
			}
		})
	}
	t.Run("before commit", func(t *testing.T) {
		s, db, a, id := workflowGuardFixture(t)
		if _, err := db.Exec(`UPDATE auth_sessions SET absolute_expires_at=clock_timestamp()+interval '250 milliseconds'`); err != nil {
			t.Fatal(err)
		}
		err := s.questionTx(context.Background(), a, question.SaveDraftAction, nil, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
			_, err := tx.ExecContext(ctx, `INSERT INTO question_events(id,actor_user_id,action,object_kind,object_id,request_id) VALUES('33333333-3333-4333-8333-333333333333',$1,'saveDraft','draft','test','test')`, id)
			time.Sleep(350 * time.Millisecond)
			return err
		})
		if !errors.Is(err, auth.ErrAuthenticationRequired) {
			t.Fatal(err)
		}
		var n int
		if err = db.QueryRow(`SELECT count(*) FROM question_events`).Scan(&n); err != nil || n != 0 {
			t.Fatal("expired work committed")
		}
	})
	t.Run("read needs no csrf", func(t *testing.T) {
		s, _, a, _ := workflowGuardFixture(t)
		a.CSRF = auth.Secret{}
		if err := s.questionReadTx(context.Background(), a, question.ListDraftsAction, func(context.Context, *sql.Tx, auth.User) error { return nil }); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("lock deadline", func(t *testing.T) {
		s, db, a, _ := workflowGuardFixture(t)
		blocker, err := db.Begin()
		if err != nil {
			t.Fatal(err)
		}
		defer blocker.Rollback()
		if _, err = blocker.Exec(`SELECT pg_advisory_xact_lock(1296127048)`); err != nil {
			t.Fatal(err)
		}
		start := time.Now()
		err = s.questionTx(context.Background(), a, question.SaveDraftAction, nil, func(context.Context, *sql.Tx, auth.User, time.Time) error {
			t.Error("blocked work executed")
			return nil
		})
		if !errors.Is(err, auth.ErrUnavailable) || time.Since(start) > 2*time.Second {
			t.Fatal(err)
		}
	})
	t.Run("reauth boundary", func(t *testing.T) {
		s, db, a, id := workflowGuardFixture(t)
		if _, err := db.Exec(`INSERT INTO auth_user_roles VALUES($1,'admin')`, id); err != nil {
			t.Fatal(err)
		}
		for _, interval := range []string{"300 seconds", "-1 seconds"} {
			if _, err := db.Exec(`UPDATE auth_sessions SET reauthenticated_at=clock_timestamp()-$1::interval`, interval); err != nil {
				t.Fatal(err)
			}
			err := s.questionTx(context.Background(), a, question.ActivateReleaseAction, nil, func(context.Context, *sql.Tx, auth.User, time.Time) error {
				t.Error("stale reauth accepted")
				return nil
			})
			if !errors.Is(err, auth.ErrReauthRequired) {
				t.Fatal(err)
			}
		}
		if _, err := db.Exec(`UPDATE auth_sessions SET reauthenticated_at=clock_timestamp()`); err != nil {
			t.Fatal(err)
		}
		if err := s.questionTx(context.Background(), a, question.ActivateReleaseAction, nil, func(ctx context.Context, _ *sql.Tx, _ auth.User, _ time.Time) error {
			deadline, ok := ctx.Deadline()
			if !ok || time.Until(deadline) > 8*time.Second || time.Until(deadline) < 7*time.Second {
				t.Error("wrong deadline")
			}
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	})
}

func TestQuestionIdempotencyAtomicity(t *testing.T) {
	s, db, a, id := workflowGuardFixture(t)
	ctx := context.Background()
	digest := strings.Repeat("a", 64)
	run := func(hash string, fail bool) error {
		return s.questionTx(ctx, a, question.SaveDraftAction, nil, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
			raw, found, err := s.questionReplay(ctx, tx, u.ID, "saveDraft", a.IdempotencyKey, hash)
			if err != nil {
				return err
			}
			if found {
				if string(raw) != `{"revision":2}` {
					t.Error("wrong replay")
				}
				return nil
			}
			if err = s.questionRemember(ctx, tx, id, "saveDraft", a.IdempotencyKey, hash, []byte(`{"revision":2}`)); err != nil {
				return err
			}
			actor := id
			if fail {
				actor = "99999999-9999-4999-8999-999999999999"
			}
			_, err = tx.ExecContext(ctx, `INSERT INTO question_events(id,actor_user_id,action,object_kind,object_id,request_id) VALUES('33333333-3333-4333-8333-333333333333',$1,'saveDraft','draft','test','test')`, actor)
			return err
		})
	}
	if err := run(digest, true); err == nil {
		t.Fatal("audit FK failure succeeded")
	}
	for _, table := range []string{"question_idempotency", "question_events"} {
		var n int
		if err := db.QueryRow(`SELECT count(*) FROM ` + table).Scan(&n); err != nil || n != 0 {
			t.Fatalf("rollback leaked %s", table)
		}
	}
	for i := 0; i < 2; i++ {
		if err := run(digest, false); err != nil {
			t.Fatal(err)
		}
	}
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM question_events`).Scan(&n); err != nil || n != 1 {
		t.Fatal("duplicate event")
	}
	if err := run(strings.Repeat("b", 64), false); !errors.Is(err, question.ErrIdempotencyConflict) {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM auth_user_roles WHERE role='editor'`); err != nil {
		t.Fatal(err)
	}
	if err := run(digest, false); !errors.Is(err, auth.ErrForbidden) {
		t.Fatal(err)
	}
}

// Observe a real DB lock wait before allowing the proof to expire.
func waitQuestionLock(t *testing.T, db *sql.DB) {
	t.Helper()
	deadline := time.Now().Add(800 * time.Millisecond)
	for time.Now().Before(deadline) {
		var n int
		if err := db.QueryRow(`SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock'`).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n > 0 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("question transaction never reached lock barrier")
}
