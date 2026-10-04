package store_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/question"
	"github.com/yyl1212/math_master/backend/internal/store"
	"testing"
	"time"
)

// Real held locks prove ordering and that the transaction time is sampled after
// its entry fence, independent of SQL shape or method call counts.
func TestCorrectionSystemEntryOrderedLocks(t *testing.T) {
	for index, key := range store.CorrectionSystemLockIDsForTest() {
		t.Run(fmt.Sprint(index), func(t *testing.T) {
			f := newCorrectionFixture(t)
			hold, e := f.db.BeginTx(f.ctx, nil)
			if e != nil {
				t.Fatal(e)
			}
			defer hold.Rollback()
			if _, e = hold.ExecContext(f.ctx, `SELECT pg_advisory_xact_lock($1)`, key); e != nil {
				t.Fatal(e)
			}
			ctx, cancel := context.WithTimeout(f.ctx, 3*time.Second)
			defer cancel()
			type result struct {
				now time.Time
				err error
			}
			done := make(chan result, 1)
			go func() { now, e := store.CorrectionSystemEntryForTest(ctx, f.repo, true); done <- result{now, e} }()
			waiting := false
			end := time.Now().Add(700 * time.Millisecond)
			for time.Now().Before(end) {
				if e = f.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pg_locks WHERE locktype='advisory' AND NOT granted AND database=(SELECT oid FROM pg_database WHERE datname=current_database()) AND classid=$1::oid AND objid=$2::oid)`, uint32(uint64(key)>>32), uint32(key)).Scan(&waiting); e != nil {
					t.Fatal(e)
				}
				if waiting {
					break
				}
				time.Sleep(5 * time.Millisecond)
			}
			if !waiting {
				t.Fatal("entry must wait on held lock")
			}
			for position, other := range store.CorrectionSystemLockIDsForTest() {
				if position == index {
					continue
				}
				probe, e := f.db.BeginTx(ctx, nil)
				if e != nil {
					t.Fatal(e)
				}
				var acquired bool
				e = probe.QueryRowContext(ctx, `SELECT pg_try_advisory_xact_lock($1)`, other).Scan(&acquired)
				probe.Rollback()
				if e != nil {
					t.Fatal(e)
				}
				if position < index && acquired {
					t.Fatal("prior shared lock must already be held", position)
				}
				if position > index && !acquired {
					t.Fatal("later lock must not precede its dependency", position)
				}
			}
			var releasedAfter time.Time
			if e = hold.QueryRowContext(ctx, `SELECT clock_timestamp()`).Scan(&releasedAfter); e != nil {
				t.Fatal(e)
			}
			if e = hold.Commit(); e != nil {
				t.Fatal(e)
			}
			select {
			case v := <-done:
				if v.err != nil || v.now.Before(releasedAfter) {
					t.Fatal("entry time must follow lock release", v.err, v.now, releasedAfter)
				}
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
		})
	}
}
func TestCorrectionSystemEntryWithoutContentLocks(t *testing.T) {
	f := newCorrectionFixture(t)
	hold, e := f.db.BeginTx(f.ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer hold.Rollback()
	for _, key := range store.CorrectionSystemLockIDsForTest() {
		if _, e = hold.ExecContext(f.ctx, `SELECT pg_advisory_xact_lock($1)`, key); e != nil {
			t.Fatal(e)
		}
	}
	ctx, cancel := context.WithTimeout(f.ctx, 500*time.Millisecond)
	defer cancel()
	if now, e := store.CorrectionSystemEntryForTest(ctx, f.repo, false); e != nil || now.IsZero() {
		t.Fatal("lease bookkeeping must keep its no-content-lock mode", e)
	}
}

func TestCorrectionJobClockAfterRowWait(t *testing.T) {
	f := newCorrectionFixture(t)
	f.registerRule(nil, 1)
	l, e := f.repo.ClaimCorrectionJob(f.ctx)
	if e != nil || l == nil {
		t.Fatal(e)
	}
	hold, e := f.db.BeginTx(f.ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer hold.Rollback()
	if _, e = hold.ExecContext(f.ctx, `SELECT id FROM correction_jobs WHERE id=$1 FOR UPDATE`, l.JobID); e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(f.ctx, 3*time.Second)
	defer cancel()
	type result struct {
		now time.Time
		err error
	}
	done := make(chan result, 1)
	go func() { now, e := store.CorrectionJobLockedTimeForTest(ctx, f.db, l.JobID); done <- result{now, e} }()
	waiting := false
	end := time.Now().Add(700 * time.Millisecond)
	for time.Now().Before(end) {
		if e = f.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND query LIKE '%correction_jobs%' AND query LIKE '%FOR UPDATE%')`).Scan(&waiting); e != nil {
			t.Fatal(e)
		}
		if waiting {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !waiting {
		t.Fatal("clock query must be blocked on the current job row")
	}
	var release time.Time
	if e = hold.QueryRowContext(ctx, `SELECT clock_timestamp()`).Scan(&release); e != nil {
		t.Fatal(e)
	}
	if e = hold.Commit(); e != nil {
		t.Fatal(e)
	}
	select {
	case v := <-done:
		if v.err != nil || v.now.Before(release) {
			t.Fatal("lease time sampled before row lock", v.err, v.now, release)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestCorrectionCombinedDependenciesExactSources(t *testing.T) {
	f := newCorrectionFixture(t)
	i := question.Instance{Identity: f.items[0].Instance, Body: question.QuestionBody{Knowledge: question.Ref{ID: f.knowledge.ID, Version: f.knowledge.Version}}}
	for _, name := range []string{"nil", "empty", "missing-knowledge", "missing-unit"} {
		t.Run(name, func(t *testing.T) {
			v := i
			switch name {
			case "empty":
				v.Body.Units = []question.Ref{}
			case "missing-knowledge":
				v.Body.Knowledge.Version++
			case "missing-unit":
				v.Body.Units = []question.Ref{{ID: "missing-unit", Version: 1}}
			}
			deps, e := store.CorrectionInstanceDependenciesForTest(f.ctx, f.db, v)
			if name == "missing-knowledge" || name == "missing-unit" {
				if !errors.Is(e, sql.ErrNoRows) {
					t.Fatal("missing actual source must fail closed", e)
				}
				return
			}
			if e != nil {
				t.Fatal("no unit dependencies is valid", e)
			}
			if len(deps) != 2 {
				t.Fatal("expected exact instance and actual knowledge", deps)
			}
			found := false
			for _, d := range deps {
				if d.Kind == "knowledge" {
					found = d.ID == f.knowledge.ID && d.Version != nil && *d.Version == f.knowledge.Version && d.SHA256 == f.knowledge.SHA256
				}
			}
			if !found {
				t.Fatal("actual exact knowledge dependency missing", deps)
			}
		})
	}
}
