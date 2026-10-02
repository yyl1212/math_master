package store

import (
	"context"
	"database/sql"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pressly/goose/v3"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"os"
	"testing"
	"time"
)

func TestLearningConfiguredLegacyAndPartial(t *testing.T) {
	db := testutil.Database(t)
	ctx := context.Background()
	p, e := goose.NewProvider(goose.DialectPostgres, db, os.DirFS("../../../db/migrations"))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = p.UpTo(ctx, 5); e != nil {
		t.Fatal(e)
	}
	check := func(want bool, wantErr error) {
		t.Helper()
		tx, e := db.Begin()
		if e != nil {
			t.Fatal(e)
		}
		defer tx.Rollback()
		got, e := learningConfigured(ctx, tx)
		if got != want || !errors.Is(e, wantErr) {
			t.Fatal(got, e)
		}
		if !got && e == nil {
			if e = questionConfigured(ctx, tx); e != nil {
				t.Fatal("old question service unavailable", e)
			}
		}
	}
	check(false, nil)
	if _, e = p.Up(ctx); e != nil {
		t.Fatal(e)
	}
	check(true, nil)
	if _, e = db.Exec(`ALTER TABLE learner_answer_exposures RENAME TO broken_learning_exposures`); e != nil {
		t.Fatal(e)
	}
	check(false, learning.ErrNotConfigured)
}
func TestLearningCommitProofRechecksAndRollsBack(t *testing.T) {
	for _, tt := range []struct {
		name, sql string
		want      error
	}{
		{"role", `DELETE FROM auth_user_roles WHERE role='learner'`, auth.ErrForbidden},
		{"session", `UPDATE auth_sessions SET revoked_at=clock_timestamp()`, auth.ErrAuthenticationRequired},
		{"expires", `UPDATE auth_sessions SET absolute_expires_at=clock_timestamp()`, auth.ErrAuthenticationRequired},
		{"credential", `UPDATE auth_users SET credential_version=credential_version+1`, auth.ErrAuthenticationRequired},
		{"password", `UPDATE auth_users SET must_change_password=true`, auth.ErrPasswordChangeRequired},
		{"csrf", `UPDATE auth_sessions SET csrf=decode(repeat('03',32),'hex')`, auth.ErrCSRF},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s, db, a, id := workflowGuardFixture(t)
			called := false
			e := s.learningTx(context.Background(), a, learning.StartKnowledgeAction, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
				called = true
				if _, e := tx.ExecContext(ctx, tt.sql); e != nil {
					return e
				}
				_, e := tx.ExecContext(ctx, `INSERT INTO learner_exposure_state(owner_user_id,sequence) VALUES($1,1)`, u.ID)
				return e
			})
			if !called || !errors.Is(e, tt.want) {
				t.Fatal("stale commit proof", called, e)
			}
			var n int
			if e = db.QueryRow(`SELECT count(*) FROM learner_exposure_state WHERE owner_user_id=$1`, id).Scan(&n); e != nil || n != 0 {
				t.Fatal("partial commit", n, e)
			}
		})
	}
	t.Run("cancel", func(t *testing.T) {
		s, db, a, _ := workflowGuardFixture(t)
		ctx, c := context.WithCancel(context.Background())
		defer c()
		e := s.learningTx(ctx, a, learning.StartKnowledgeAction, func(ctx context.Context, tx *sql.Tx, u auth.User, _ time.Time) error {
			_, e := tx.ExecContext(ctx, `INSERT INTO learner_exposure_state(owner_user_id,sequence) VALUES($1,1)`, u.ID)
			c()
			return e
		})
		var n int
		db.QueryRow(`SELECT count(*) FROM learner_exposure_state`).Scan(&n)
		if e == nil || n != 0 {
			t.Fatal("cancelled mutation committed", n, e)
		}
	})
}
func TestLearningCommitProofExpiresAfterLockWait(t *testing.T) {
	s, db, a, _ := workflowGuardFixture(t)
	db.Exec(`UPDATE auth_sessions SET absolute_expires_at=clock_timestamp()+interval '200 milliseconds'`)
	block, e := db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer block.Rollback()
	if _, e = block.Exec(`SELECT pg_advisory_xact_lock(1296127048)`); e != nil {
		t.Fatal(e)
	}
	done := make(chan error, 1)
	go func() {
		done <- s.learningTx(context.Background(), a, learning.StartKnowledgeAction, func(context.Context, *sql.Tx, auth.User, time.Time) error {
			t.Error("expired proof entered work")
			return nil
		})
	}()
	waitQuestionLock(t, db)
	time.Sleep(230 * time.Millisecond)
	block.Commit()
	if e = <-done; !errors.Is(e, auth.ErrAuthenticationRequired) {
		t.Fatal(e)
	}
}
func TestLearningSharedLocksDifferentUsersAndExclusiveOrder(t *testing.T) {
	s, db, a, _ := workflowGuardFixture(t)
	ctx := context.Background()
	tx, e := db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	other := "33333333-3333-4333-8333-333333333333"
	b := a
	b.TokenHash[0] = 4
	for _, q := range []string{`INSERT INTO auth_users(id,username,password_phc) VALUES('33333333-3333-4333-8333-333333333333','other_learner','isolated-test-only')`, `INSERT INTO auth_user_roles VALUES('33333333-3333-4333-8333-333333333333','learner')`} {
		if _, e = tx.Exec(q); e != nil {
			t.Fatal(e)
		}
	}
	if _, e = tx.Exec(`INSERT INTO auth_sessions(token_hash,user_id,credential_version,csrf,absolute_expires_at) VALUES($1,$2,1,$3,clock_timestamp()+interval '1 hour')`, b.TokenHash[:], other, b.CSRF[:]); e != nil {
		t.Fatal(e)
	}
	if e = tx.Commit(); e != nil {
		t.Fatal(e)
	}
	entered := make(chan string, 2)
	release := make(chan struct{})
	done := make(chan error, 2)
	for _, access := range []question.Access{a, b} {
		go func(access question.Access) {
			done <- s.learningTx(ctx, access, learning.StartKnowledgeAction, func(_ context.Context, _ *sql.Tx, u auth.User, _ time.Time) error {
				entered <- u.ID
				<-release
				return nil
			})
		}(access)
	}
	for j := 0; j < 2; j++ {
		select {
		case <-entered:
		case <-time.After(750 * time.Millisecond):
			close(release)
			t.Fatal("different learners serialized by exclusive locks")
		}
	}
	close(release)
	for j := 0; j < 2; j++ {
		if e = <-done; e != nil {
			t.Fatal(e)
		}
	}
	block, e := db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer block.Rollback()
	block.Exec(`SELECT pg_advisory_xact_lock(1296127049)`)
	go func() {
		done <- s.learningTx(ctx, a, learning.StartKnowledgeAction, func(context.Context, *sql.Tx, auth.User, time.Time) error { return nil })
	}()
	waitQuestionLock(t, db)
	var gotContentLock bool
	if e = db.QueryRow(`SELECT EXISTS(SELECT 1 FROM pg_locks WHERE database=(SELECT oid FROM pg_database WHERE datname=current_database()) AND locktype='advisory' AND objid=1296127048 AND granted)`).Scan(&gotContentLock); e != nil || gotContentLock {
		t.Fatal("content lock obtained before account lock", e)
	}
	block.Commit()
	if e = <-done; e != nil {
		t.Fatal(e)
	}
}

func TestLearningConfiguredRollbackMustNotDowngrade(t *testing.T) {
	db := testutil.Database(t)
	ctx := context.Background()
	p, err := goose.NewProvider(goose.DialectPostgres, db, os.DirFS("../../../db/migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = p.Up(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err = p.DownTo(ctx, 5); err != nil {
		t.Fatal(err)
	}
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	enabled, err := learningConfigured(ctx, tx)
	if enabled || !errors.Is(err, learning.ErrNotConfigured) {
		t.Fatal("enabled database silently downgraded to answer delivery without exposure", enabled, err)
	}

	tx.Rollback()
	if _, err = p.Up(ctx); err != nil {
		t.Fatal("explicit recovery migration failed", err)
	}
	recovered, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer recovered.Rollback()
	enabled, err = learningConfigured(ctx, recovered)
	if err != nil || !enabled {
		t.Fatal("explicit migration did not restore learning", enabled, err)
	}
}

func TestLearningCommitProofExpiryErrorIsClosed(t *testing.T) {
	got := learningError(&pgconn.PgError{Code: "M0001", Message: "private SQL expiry diagnostic"})
	if !errors.Is(got, learning.ErrAssessmentExpired) {
		t.Fatal("database expiry must return the closed assessment expiry fault", got)
	}
	unknown := learningError(&pgconn.PgError{Code: "P0001", Message: "private SQL diagnostic"})
	if !errors.Is(unknown, auth.ErrUnavailable) {
		t.Fatal("unknown database diagnostics must remain private", unknown)
	}
}
