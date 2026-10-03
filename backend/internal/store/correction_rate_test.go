package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"testing"
	"time"
)

func TestCorrectionSlidingBoundaryAndAllScopes(t *testing.T) {
	for _, scope := range []struct {
		name  string
		limit int
	}{{"create", 20}, {"process", 120}, {"retry", 20}, {"notification-read", 300}} {
		t.Run(scope.name, func(t *testing.T) {
			_, db, _, owner := workflowGuardFixture(t)
			tx, e := db.Begin()
			if e != nil {
				t.Fatal(e)
			}
			defer tx.Rollback()
			var now time.Time
			if e = tx.QueryRow(`SELECT clock_timestamp()`).Scan(&now); e != nil {
				t.Fatal(e)
			}
			left := now.Add(-time.Hour)
			for i := 0; i < scope.limit; i++ {
				if _, e = tx.Exec(`INSERT INTO correction_rate_limits(owner_user_id,scope,command_key,consumed_at) VALUES($1,$2,$3,$4)`, owner, scope.name, fmt.Sprintf("left:%d", i), left); e != nil {
					t.Fatal(e)
				}
			}
			ctx := correctionWithCommand(context.Background(), "createPlan", "cases:example", "11111111-1111-4111-8111-111111111111")
			if e = correctionConsumeRate(ctx, tx, owner, scope.name, now); e != nil {
				t.Fatal("inclusive left boundary counted", e)
			}
			for i := 0; i < scope.limit-1; i++ {
				if _, e = tx.Exec(`INSERT INTO correction_rate_limits(owner_user_id,scope,command_key,consumed_at) VALUES($1,$2,$3,$4)`, owner, scope.name, fmt.Sprintf("inside:%d", i), left.Add(time.Microsecond)); e != nil {
					t.Fatal(e)
				}
			}
			ctx = correctionWithCommand(context.Background(), "createPlan", "cases:example", "22222222-2222-4222-8222-222222222222")
			e = correctionConsumeRate(ctx, tx, owner, scope.name, now)
			var rate *correction.RateError
			if !errors.As(e, &rate) || !rate.RetryAt.Equal(now.Add(time.Microsecond)) {
				t.Fatal("wrong full-window retry boundary", e, rate)
			}
			var n int
			if e = tx.QueryRow(`SELECT count(*) FROM correction_rate_limits WHERE owner_user_id=$1 AND scope=$2 AND consumed_at>$3 AND consumed_at<=$4`, owner, scope.name, left, now).Scan(&n); e != nil || n != scope.limit {
				t.Fatal("failed command consumed quota", n, e)
			}
		})
	}
}

func TestCorrectionPlanUnknownRuleRemainsClosed(t *testing.T) {
	_, db, _, owner := workflowGuardFixture(t)
	tx, e := db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	ctx := correctionWithConfig(context.Background(), true)
	if e = correctionNewAttemptGuard(ctx, tx, owner, question.Identity{ID: "unknown-rule-scope", Version: 1}, 2, time.Now()); !errors.Is(e, learning.ErrAssessmentNotReady) {
		t.Fatal("unregistered grading rule opened", e)
	}
}
