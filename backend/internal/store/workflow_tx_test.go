package store

import (
	"context"
	"database/sql"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"testing"
	"time"
)

func workflowGuardFixture(t *testing.T) (*Store, *sql.DB, publication.Access, string) {
	t.Helper()
	db := testutil.Database(t)
	ctx := context.Background()
	if err := Up(ctx, db, "../../../db/migrations"); err != nil {
		t.Fatal(err)
	}
	id := "11111111-1111-4111-8111-111111111111"
	var token auth.Digest
	token[0] = 1
	var csrf auth.Secret
	csrf[0] = 2
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	for _, q := range []string{
		`INSERT INTO auth_users(id,username,password_phc) VALUES ('11111111-1111-4111-8111-111111111111','guard_editor','isolated-test-only')`,
		`INSERT INTO auth_user_roles(user_id,role) VALUES ('11111111-1111-4111-8111-111111111111','learner'),('11111111-1111-4111-8111-111111111111','editor')`,
	} {
		if _, err = tx.ExecContext(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO auth_sessions(token_hash,user_id,credential_version,csrf,created_at,last_seen_at,absolute_expires_at) VALUES($1,$2,1,$3,clock_timestamp(),clock_timestamp(),clock_timestamp()+interval '1 hour')`, token[:], id, csrf[:]); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	return New(db), db, publication.Access{TokenHash: token, CSRF: csrf, IdempotencyKey: "22222222-2222-4222-8222-222222222222", RequestID: "guard-request"}, id
}
func TestWorkflowSessionExpiresWhileWaiting(t *testing.T) {
	for _, lock := range []string{"content", "user"} {
		t.Run(lock, func(t *testing.T) {
			s, db, access, id := workflowGuardFixture(t)
			ctx := context.Background()
			if _, err := db.ExecContext(ctx, `UPDATE auth_sessions SET absolute_expires_at=clock_timestamp()+interval '250 milliseconds'`); err != nil {
				t.Fatal(err)
			}
			tx, err := db.BeginTx(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer tx.Rollback()
			q := `SELECT pg_advisory_xact_lock(1296127048)`
			if lock == "user" {
				q = `SELECT id FROM auth_users WHERE id='11111111-1111-4111-8111-111111111111' FOR UPDATE`
			}
			if _, err = tx.ExecContext(ctx, q); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() {
				done <- s.workflowTx(ctx, access, publication.SaveDraftAction, []string{id}, func(context.Context, *sql.Tx, auth.User, time.Time) error {
					t.Error("expired proof reached mutation")
					return nil
				})
			}()
			time.Sleep(400 * time.Millisecond)
			if err = tx.Commit(); err != nil {
				t.Fatal(err)
			}
			if err = <-done; !errors.Is(err, auth.ErrAuthenticationRequired) {
				t.Fatalf("expired proof accepted: %v", err)
			}
		})
	}
	t.Run("csrf", func(t *testing.T) {
		s, _, a, _ := workflowGuardFixture(t)
		a.CSRF[0]++
		if err := s.workflowTx(context.Background(), a, publication.SaveDraftAction, nil, func(context.Context, *sql.Tx, auth.User, time.Time) error {
			t.Error("old CSRF reached mutation")
			return nil
		}); !errors.Is(err, auth.ErrCSRF) {
			t.Fatal(err)
		}
	})
	t.Run("lock timeout", func(t *testing.T) {
		s, db, a, _ := workflowGuardFixture(t)
		tx, err := db.Begin()
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback()
		if _, err = tx.Exec(`SELECT pg_advisory_xact_lock(1296127048)`); err != nil {
			t.Fatal(err)
		}
		start := time.Now()
		err = s.workflowTx(context.Background(), a, publication.SaveDraftAction, nil, func(context.Context, *sql.Tx, auth.User, time.Time) error {
			t.Error("blocked mutation executed")
			return nil
		})
		if !errors.Is(err, auth.ErrUnavailable) || time.Since(start) > 2*time.Second {
			t.Fatalf("unbounded lock: %v", err)
		}
	})
	t.Run("content deadline", func(t *testing.T) {
		s, _, a, _ := workflowGuardFixture(t)
		err := s.workflowTx(context.Background(), a, publication.SaveDraftAction, nil, func(ctx context.Context, _ *sql.Tx, _ auth.User, _ time.Time) error {
			d, ok := ctx.Deadline()
			if !ok || time.Until(d) < 7*time.Second || time.Until(d) > 8*time.Second {
				t.Error("content inherited account timeout")
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	})
}

func TestWorkflowSessionExpiresDuringContentWork(t *testing.T) {
	s, db, a, id := workflowGuardFixture(t)
	if _, err := db.Exec(`UPDATE auth_sessions SET absolute_expires_at=clock_timestamp()+interval '300 milliseconds'`); err != nil {
		t.Fatal(err)
	}
	err := s.workflowTx(context.Background(), a, publication.SaveDraftAction, nil, func(ctx context.Context, tx *sql.Tx, _ auth.User, now time.Time) error {
		if _, err := tx.ExecContext(ctx, `INSERT INTO content_workflow_events(id,actor_user_id,action,object_kind,object_id,request_id,created_at) VALUES('44444444-4444-4444-8444-444444444444',$1,'saveDraft','draft','expiry-fixture','expiry-fixture',$2)`, id, now); err != nil {
			return err
		}
		time.Sleep(350 * time.Millisecond)
		return nil
	})
	if !errors.Is(err, auth.ErrAuthenticationRequired) {
		t.Fatalf("expired session committed after content work: %v", err)
	}
	var n int
	if err = db.QueryRow(`SELECT count(*) FROM content_workflow_events`).Scan(&n); err != nil || n != 0 {
		t.Fatal("expired work leaked an audit")
	}
}
