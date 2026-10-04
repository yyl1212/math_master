package store_test

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/question"
	"testing"
)

func TestCorrectionReplaySameKeyDifferentResourceConsumesTwice(t *testing.T) {
	f := newCorrectionFixture(t)
	first := f.registerRule(nil, 1)
	second := f.registerRule(&f.knowledge, 1)
	key := f.Access("author_b", false)
	in := correctionPlanInput()
	a, e := f.repo.CreateCorrectionPlan(f.ctx, key, first.ID, in)
	if e != nil {
		t.Fatal(e)
	}
	b, e := f.repo.CreateCorrectionPlan(f.ctx, key, second.ID, in)
	if e != nil || a.Data.Plan.Ref.ID == b.Data.Plan.Ref.ID {
		t.Fatal("resource isolated key collapsed", e)
	}
	if f.count(`SELECT count(*) FROM correction_rate_limits WHERE owner_user_id=$1 AND scope='create'`, f.ids["author_b"]) != 2 {
		t.Fatal("same UUID across resource deduplicated successful quota")
	}
}
func TestCorrectionPlanCreateQuotaRollsBackAndReplayFree(t *testing.T) {
	f := newCorrectionFixture(t)
	c := f.registerRule(nil, 1)
	firstKey := f.Access("author_b", false)
	in := correctionPlanInput()
	first, e := f.repo.CreateCorrectionPlan(f.ctx, firstKey, c.ID, in)
	if e != nil {
		t.Fatal(e)
	}
	for i := 1; i < 20; i++ {
		if _, e = f.repo.CreateCorrectionPlan(f.ctx, f.Access("author_b", false), c.ID, in); e != nil {
			t.Fatal(i, e)
		}
	}
	_, e = f.repo.CreateCorrectionPlan(f.ctx, f.Access("author_b", false), c.ID, in)
	var rate *correction.RateError
	if !errors.As(e, &rate) || rate.RetryAt.IsZero() {
		t.Fatal("create hourly limit missing", e)
	}
	replay, e := f.repo.CreateCorrectionPlan(f.ctx, firstKey, c.ID, in)
	if e != nil || correctionReceiptJSON(t, replay) != correctionReceiptJSON(t, first) {
		t.Fatal("full-window replay charged", e)
	}
	in.Reason = "A different valid draft cannot reuse an existing successful command."
	if _, e = f.repo.CreateCorrectionPlan(f.ctx, firstKey, c.ID, in); !errors.Is(e, question.ErrIdempotencyConflict) {
		t.Fatal("different body reused successful key", e)
	}
	for _, q := range []string{`SELECT count(*) FROM correction_plans`, `SELECT count(*) FROM correction_events WHERE kind='plan_created'`, `SELECT count(*) FROM correction_idempotency WHERE action='createPlan'`, `SELECT count(*) FROM correction_rate_limits WHERE scope='create' AND owner_user_id='` + f.ids["author_b"] + `'`} {
		if f.count(q) != 20 {
			t.Fatal("failed or replay command committed", q)
		}
	}
}
