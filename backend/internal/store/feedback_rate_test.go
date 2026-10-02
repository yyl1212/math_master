package store

import (
	"context"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"testing"
	"time"
)

func TestFeedbackSlidingBoundary(t *testing.T) {
	_, db, _, id := workflowGuardFixture(t)
	tx, _ := db.Begin()
	defer tx.Rollback()
	var now time.Time
	if e := tx.QueryRow(`SELECT clock_timestamp()`).Scan(&now); e != nil {
		t.Fatal(e)
	}
	left := now.Add(-15 * time.Minute)
	for n := 0; n < 5; n++ {
		if _, e := tx.Exec(`INSERT INTO feedback_rate_limits(actor_user_id,scope,consumed_at) VALUES($1,'create',$2)`, id, left); e != nil {
			t.Fatal(e)
		}
	}
	if _, e := tx.Exec(`INSERT INTO feedback_rate_limits(actor_user_id,scope,consumed_at) VALUES($1,'create',$2)`, id, left.Add(time.Microsecond)); e != nil {
		t.Fatal(e)
	}
	if e := feedbackConsumeRates(context.Background(), tx, id, feedback.CreateAction, now); e != nil {
		t.Fatal("left boundary counted", e)
	}
	var n int
	if e := tx.QueryRow(`SELECT count(*) FROM feedback_rate_limits WHERE consumed_at>$1 AND consumed_at<=$2`, left, now).Scan(&n); e != nil || n != 2 {
		t.Fatal(n, e)
	}
}
func TestFeedbackRatesLimits(t *testing.T) {
	for _, c := range []struct {
		action      feedback.Action
		n           int
		window, age time.Duration
	}{{feedback.CreateAction, 5, 15 * time.Minute, time.Minute}, {feedback.CreateAction, 20, 24 * time.Hour, time.Hour}, {feedback.ReplyAction, 30, time.Hour, time.Minute}, {feedback.TransitionAction, 120, time.Hour, time.Minute}} {
		t.Run(string(c.action)+c.window.String(), func(t *testing.T) {
			_, db, _, id := workflowGuardFixture(t)
			tx, _ := db.Begin()
			defer tx.Rollback()
			var now time.Time
			tx.QueryRow(`SELECT clock_timestamp()`).Scan(&now)
			when := now.Add(-c.age)
			for n := 0; n < c.n; n++ {
				if _, e := tx.Exec(`INSERT INTO feedback_rate_limits(actor_user_id,scope,consumed_at) VALUES($1,$2,$3)`, id, c.action, when); e != nil {
					t.Fatal(e)
				}
			}
			e := feedbackConsumeRates(context.Background(), tx, id, c.action, now)
			var r *feedback.RateError
			if !errors.As(e, &r) || !r.RetryAt.Equal(when.Add(c.window)) {
				t.Fatal("window", e, r)
			}
		})
	}
}
