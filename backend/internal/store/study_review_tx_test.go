package store

import (
	"context"
	"database/sql"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/studyreview"
	"testing"
	"time"
)

func reviewIdentityFixture(t *testing.T) (*Store, *sql.DB, studyreview.Access) {
	t.Helper()
	db := reviewSchemaDB(t)
	sharingSchemaSeed(t, db)
	var a studyreview.Access
	for i := range a.TokenHash {
		a.TokenHash[i] = 1
		a.CSRF[i] = 2
	}
	a.IdempotencyKey = "identity-fixture"
	a.RequestID = "review-identity-test"
	if _, e := db.Exec(`INSERT INTO auth_sessions(token_hash,user_id,credential_version,csrf,absolute_expires_at) VALUES($1,'11111111-1111-4111-8111-111111111111',1,$2,clock_timestamp()+interval '1 hour')`, a.TokenHash[:], a.CSRF[:]); e != nil {
		t.Fatal(e)
	}
	if _, e := db.Exec(`UPDATE knowledge_admin_state SET content_mode='managed',enabled_once=true,activated_at=clock_timestamp() WHERE singleton;UPDATE goose_db_version SET managed_knowledge_enabled=true WHERE version_id=0`); e != nil {
		t.Fatal(e)
	}
	return New(db), db, a
}
func waitReviewBlocked(t *testing.T, db *sql.DB) {
	t.Helper()
	deadline := time.Now().Add(750 * time.Millisecond)
	for time.Now().Before(deadline) {
		var n int
		e := db.QueryRow(`SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND pid<>pg_backend_pid() AND wait_event_type='Lock'`).Scan(&n)
		if e != nil {
			t.Fatal(e)
		}
		if n > 0 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("expected identity lock wait absent")
}
func TestReviewReadRevocationOrdering(t *testing.T) {
	t.Run("read-first", func(t *testing.T) {
		s, db, a := reviewIdentityFixture(t)
		entered := make(chan struct{})
		release := make(chan struct{})
		done := make(chan error, 1)
		go func() {
			done <- s.studyReviewReadTx(context.Background(), a, func(context.Context, *sql.Tx, auth.User) error { close(entered); <-release; return nil })
		}()
		select {
		case <-entered:
		case e := <-done:
			t.Fatal(e)
		case <-time.After(2 * time.Second):
			t.Fatal("read did not enter")
		}
		released := false
		defer func() {
			if !released {
				close(release)
			}
		}()
		revoke := make(chan error, 1)
		go func() {
			tx, e := db.Begin()
			if e != nil {
				revoke <- e
				return
			}
			defer tx.Rollback()
			_, e = tx.Exec(`SET LOCAL lock_timeout='1s'`)
			if e == nil {
				_, e = tx.Exec(`UPDATE auth_sessions SET revoked_at=clock_timestamp() WHERE token_hash=$1`, a.TokenHash[:])
			}
			if e == nil {
				e = tx.Commit()
			}
			revoke <- e
		}()
		waitReviewBlocked(t, db)
		select {
		case e := <-revoke:
			t.Fatalf("revocation bypassed read lock: %v", e)
		default:
		}
		close(release)
		released = true
		if e := <-done; e != nil {
			t.Fatal(e)
		}
		if e := <-revoke; e != nil {
			t.Fatal(e)
		}
		called := false
		e := s.studyReviewReadTx(context.Background(), a, func(context.Context, *sql.Tx, auth.User) error { called = true; return nil })
		if !errors.Is(e, auth.ErrAuthenticationRequired) || called {
			t.Fatal("revocation committed but read approved", called, e)
		}
	})
	t.Run("revocation-first", func(t *testing.T) {
		s, db, a := reviewIdentityFixture(t)
		block, e := db.Begin()
		if e != nil {
			t.Fatal(e)
		}
		defer block.Rollback()
		if _, e = block.Exec(`UPDATE auth_sessions SET revoked_at=clock_timestamp() WHERE token_hash=$1`, a.TokenHash[:]); e != nil {
			t.Fatal(e)
		}
		called := make(chan struct{}, 1)
		done := make(chan error, 1)
		go func() {
			done <- s.studyReviewReadTx(context.Background(), a, func(context.Context, *sql.Tx, auth.User) error { called <- struct{}{}; return nil })
		}()
		waitReviewBlocked(t, db)
		if e = block.Commit(); e != nil {
			t.Fatal(e)
		}
		if e = <-done; !errors.Is(e, auth.ErrAuthenticationRequired) {
			t.Fatal("revoked snapshot approved", e)
		}
		select {
		case <-called:
			t.Fatal("revoked body delivered")
		default:
		}
	})
}
func TestReviewIdentityAndCommitRace(t *testing.T) {
	for _, change := range []struct {
		name, sql string
		want      error
	}{{"revoked", `UPDATE auth_sessions SET revoked_at=clock_timestamp()`, auth.ErrAuthenticationRequired}, {"credentials", `UPDATE auth_users SET credential_version=credential_version+1`, auth.ErrAuthenticationRequired}, {"password", `UPDATE auth_users SET must_change_password=true`, auth.ErrPasswordChangeRequired}, {"expired", `UPDATE auth_sessions SET absolute_expires_at=clock_timestamp()`, auth.ErrAuthenticationRequired}} {
		t.Run(change.name, func(t *testing.T) {
			s, db, a := reviewIdentityFixture(t)
			e := s.knowledgeTx(context.Background(), a, true, false, func(ctx context.Context, tx *sql.Tx, u auth.User) error {
				_, e := tx.ExecContext(ctx, change.sql)
				if e != nil {
					return e
				}
				_, e = tx.ExecContext(ctx, `UPDATE managed_study_notes SET body='must roll back',revision=revision+1,updated_at=clock_timestamp()`)
				return e
			})
			if !errors.Is(e, change.want) {
				t.Fatal("commit identity stale", e)
			}
			var body string
			if e = db.QueryRow(`SELECT body FROM managed_study_notes`).Scan(&body); e != nil || body != "自己整理的数学笔记" {
				t.Fatal("partial commit", e)
			}
		})
	}
	s, db, a := reviewIdentityFixture(t)
	tx, e := db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	if _, e = tx.Exec(`SELECT id FROM auth_users FOR UPDATE`); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(`UPDATE auth_sessions SET absolute_expires_at=clock_timestamp()+interval '200 milliseconds'`); e != nil {
		t.Fatal(e)
	}
	done := make(chan error, 1)
	go func() {
		done <- s.studyReviewReadTx(context.Background(), a, func(context.Context, *sql.Tx, auth.User) error { return errors.New("expired read entered body") })
	}()
	waitReviewBlocked(t, db)
	time.Sleep(220 * time.Millisecond)
	if e = tx.Commit(); e != nil {
		t.Fatal(e)
	}
	if e = <-done; !errors.Is(e, auth.ErrAuthenticationRequired) {
		t.Fatal("fresh time after wait", e)
	}
}
