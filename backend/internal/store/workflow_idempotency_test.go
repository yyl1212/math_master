package store

import (
	"context"
	"database/sql"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"strings"
	"testing"
	"time"
)

func TestWorkflowIdempotencyRechecksPermission(t *testing.T) {
	s, db, a, id := workflowGuardFixture(t)
	ctx := context.Background()
	digest := strings.Repeat("a", 64)
	run := func(hash string, fail bool) error {
		return s.workflowTx(ctx, a, publication.SaveDraftAction, nil, func(ctx context.Context, tx *sql.Tx, u auth.User, now time.Time) error {
			result, found, err := s.workflowReplay(ctx, tx, u.ID, "saveDraft", a.IdempotencyKey, hash)
			if err != nil {
				return err
			}
			if found {
				if string(result) != `{"revision":2}` {
					t.Error("wrong historical result")
				}
				return nil
			}
			if _, err = tx.ExecContext(ctx, `INSERT INTO content_workflow_events(id,actor_user_id,action,object_kind,object_id,request_id,created_at) VALUES('33333333-3333-4333-8333-333333333333',$1,'saveDraft','draft','draft-test',$2,$3)`, id, a.RequestID, now); err != nil {
				return err
			}
			if err = s.workflowRemember(ctx, tx, id, "saveDraft", a.IdempotencyKey, hash, []byte(`{"revision":2}`)); err != nil {
				return err
			}
			if fail {
				return publication.ErrDraftConflict
			}
			return nil
		})
	}
	if err := run(digest, true); !errors.Is(err, publication.ErrDraftConflict) {
		t.Fatal(err)
	}
	for _, table := range []string{"content_idempotency", "content_workflow_events"} {
		var n int
		if err := db.QueryRow(`SELECT count(*) FROM ` + table).Scan(&n); err != nil || n != 0 {
			t.Fatalf("rollback leaked %s", table)
		}
	}
	if err := run(digest, false); err != nil {
		t.Fatal(err)
	}
	if err := run(digest, false); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM content_workflow_events`).Scan(&n); err != nil || n != 1 {
		t.Fatal("duplicate event")
	}
	if err := run(strings.Repeat("b", 64), false); !errors.Is(err, publication.ErrIdempotencyConflict) {
		t.Fatal(err)
	}
	if _, err := db.Exec(`DELETE FROM auth_user_roles WHERE user_id=$1 AND role='editor'`, id); err != nil {
		t.Fatal(err)
	}
	if err := run(digest, false); !errors.Is(err, auth.ErrForbidden) {
		t.Fatalf("replay bypassed revoked role: %v", err)
	}
	if _, err := db.Exec(`UPDATE auth_sessions SET revoked_at=clock_timestamp()`); err != nil {
		t.Fatal(err)
	}
	if err := run(digest, false); !errors.Is(err, auth.ErrAuthenticationRequired) {
		t.Fatalf("replay bypassed revoked session: %v", err)
	}
}
