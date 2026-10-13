package store_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/store"
)

func managedFenceBlock(t *testing.T, db *sql.DB, id int64) *sql.Tx {
	t.Helper()
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { tx.Rollback() })
	if _, err = tx.Exec(`SELECT pg_advisory_xact_lock($1)`, id); err != nil {
		t.Fatal(err)
	}
	return tx
}

func managedFenceWaiter(t *testing.T, db *sql.DB, id int64) int {
	t.Helper()
	deadline := time.Now().Add(800 * time.Millisecond)
	for time.Now().Before(deadline) {
		var pid int
		err := db.QueryRow(`SELECT l.pid FROM pg_locks l JOIN pg_stat_activity a ON a.pid=l.pid WHERE a.datname=current_database() AND l.locktype='advisory' AND NOT l.granted AND l.classid=($1::bigint>>32)::oid AND l.objid=($1::bigint & 4294967295)::oid AND l.objsubid=1 LIMIT 1`, id).Scan(&pid)
		if err == nil {
			return pid
		}
		if !errors.Is(err, sql.ErrNoRows) {
			t.Fatal(err)
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("guard did not wait at expected fence")
	return 0
}

func managedFenceHeld(t *testing.T, db *sql.DB, pid int, id int64) bool {
	t.Helper()
	var held bool
	if err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM pg_locks WHERE pid=$1 AND locktype='advisory' AND granted AND mode='ShareLock' AND classid=($2::bigint>>32)::oid AND objid=($2::bigint & 4294967295)::oid AND objsubid=1)`, pid, id).Scan(&held); err != nil {
		t.Fatal(err)
	}
	return held
}

func TestManagedCorrectionFenceTransportBound(t *testing.T) {
	db, _, ctx := setup(t)
	repo, count, close, err := store.ManagedCorrectionCountingStoreForTest(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	defer close()
	if err = store.ManagedCorrectionGuardForTest(ctx, repo, true); err != nil {
		t.Fatal(err)
	}
	if got := count.Load(); got != 1 {
		t.Fatalf("complete fence used %d Exec round trips, want 1", got)
	}
}

func TestManagedCorrectionFenceOrderedLocks(t *testing.T) {
	for _, blocked := range []int{1, 2} {
		t.Run([]string{"content", "knowledge"}[blocked-1], func(t *testing.T) {
			db, repo, ctx := setup(t)
			ids := store.ManagedCorrectionLockIDsForTest()
			control := managedFenceBlock(t, db, ids[blocked])
			done := make(chan error, 1)
			go func() { done <- store.ManagedCorrectionGuardForTest(ctx, repo, true) }()
			pid := managedFenceWaiter(t, db, ids[blocked])
			for index, id := range ids {
				if got, want := managedFenceHeld(t, db, pid, id), index < blocked; got != want {
					t.Fatalf("lock %d held=%v, want %v before blocked lock", index, got, want)
				}
			}
			if err := control.Commit(); err != nil {
				t.Fatal(err)
			}
			if err := <-done; err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestManagedCorrectionFenceFreshModeAfterWait(t *testing.T) {
	db, repo, ctx := setup(t)
	ids := store.ManagedCorrectionLockIDsForTest()
	control := managedFenceBlock(t, db, ids[2])
	done := make(chan error, 1)
	go func() { done <- store.ManagedCorrectionGuardForTest(ctx, repo, true) }()
	managedFenceWaiter(t, db, ids[2])
	if _, err := control.Exec(`UPDATE knowledge_admin_state SET content_mode='managed',enabled_once=true,activated_at=clock_timestamp()`); err != nil {
		t.Fatal(err)
	}
	if _, err := control.Exec(`UPDATE goose_db_version SET managed_knowledge_enabled=true WHERE version_id=0`); err != nil {
		t.Fatal(err)
	}
	if err := control.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := <-done; !errors.Is(err, correction.ErrRetired) {
		t.Fatal("mode read did not see activation committed during wait", err)
	}
}

func TestManagedCorrectionFenceBookkeepingContract(t *testing.T) {
	db, repo, ctx := setup(t)
	ids := store.ManagedCorrectionLockIDsForTest()
	control := managedFenceBlock(t, db, ids[0])
	if _, err := control.Exec(`SELECT pg_advisory_xact_lock($1)`, ids[1]); err != nil {
		t.Fatal(err)
	}
	short, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()
	if err := store.ManagedCorrectionGuardForTest(short, repo, false); err != nil {
		t.Fatal("bookkeeping waited on admin/content", err)
	}
}

func TestManagedCorrectionFenceCancellationRollback(t *testing.T) {
	for _, cancelled := range []bool{false, true} {
		t.Run(map[bool]string{false: "lock-timeout", true: "cancel"}[cancelled], func(t *testing.T) {
			db, repo, ctx := setup(t)
			ids := store.ManagedCorrectionLockIDsForTest()
			control := managedFenceBlock(t, db, ids[2])
			request, cancel := context.WithCancel(ctx)
			defer cancel()
			done := make(chan error, 1)
			go func() { done <- store.ManagedCorrectionGuardForTest(request, repo, true) }()
			managedFenceWaiter(t, db, ids[2])
			if cancelled {
				cancel()
			}
			err := <-done
			if cancelled {
				if !errors.Is(err, context.Canceled) {
					t.Fatal("cancel error changed", err)
				}
			} else {
				var pg *pgconn.PgError
				if !errors.As(err, &pg) || pg.Code != "55P03" {
					t.Fatal("lock timeout error changed", err)
				}
			}
			for _, id := range ids[:2] {
				var acquired bool
				// Cancellation can return before PostgreSQL consumes the connection's
				// rollback/termination; observe completion within a bounded interval.
				deadline := time.Now().Add(500 * time.Millisecond)
				for time.Now().Before(deadline) {
					if err := control.QueryRow(`SELECT pg_try_advisory_xact_lock($1)`, id).Scan(&acquired); err != nil {
						t.Fatal(err)
					}
					if acquired {
						break
					}
					time.Sleep(5 * time.Millisecond)
				}
				if !acquired {
					t.Fatal("rollback retained preceding lock")
				}
			}
		})
	}
}
