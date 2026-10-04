package store_test

import (
	"context"
	"errors"
	"github.com/pressly/goose/v3"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"os"
	"testing"
	"time"
)

func TestCorrectionWorkerDisabledKeepsRestriction(t *testing.T) {
	f := newCorrectionFixture(t)
	f.cSubmittedBasis("learner_a")
	f.registerRule(nil, 1)
	r, e := correction.NewRunner(f.repo, correction.Options{Enabled: false, Limit: 50})
	if e != nil {
		t.Fatal(e)
	}
	ctx, cancel := context.WithTimeout(f.ctx, 10*time.Millisecond)
	defer cancel()
	if e = r.Run(ctx); e != nil {
		t.Fatal(e)
	}
	v, e := f.repo.ReadLearningOverview(f.ctx, f.Access("learner_a", false))
	if e != nil || v.EffectivePassedCount != 0 || f.count(`SELECT count(*) FROM correction_results`) != 0 {
		t.Fatal("disabling worker bypassed synchronous restriction", v, e)
	}
}
func TestCorrectionWorkerSchemaStateRealDatabase(t *testing.T) {
	db := testutil.Database(t)
	ctx := context.Background()
	p, e := goose.NewProvider(goose.DialectPostgres, db, os.DirFS("../../../db/migrations"))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = p.UpTo(ctx, 7); e != nil {
		t.Fatal(e)
	}
	r, e := correction.NewRunner(store.New(db), correction.Options{Enabled: true, Limit: 50})
	if e != nil {
		t.Fatal(e)
	}
	idle, stop := context.WithTimeout(ctx, 10*time.Millisecond)
	if e = r.Run(idle); e != nil {
		t.Fatal(e)
	}
	stop()
	if _, e = r.RunOnce(ctx); !errors.Is(e, correction.ErrNeverEnabled) {
		t.Fatal("legacy state lost", e)
	}
	if e = store.Up(ctx, db, "../../../db/migrations"); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(`DROP TABLE notification_reads`); e != nil {
		t.Fatal(e)
	}
	if _, e = r.RunOnce(ctx); !errors.Is(e, correction.ErrNotConfigured) || errors.Is(e, correction.ErrNeverEnabled) {
		t.Fatal("damaged enabled database idled", e)
	}
}
