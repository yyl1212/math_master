package store

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"time"
)

func feedbackConsumeRates(ctx context.Context, tx *sql.Tx, actor string, action feedback.Action, now time.Time) error {
	type window struct {
		span  time.Duration
		limit int
	}
	var windows []window
	switch action {
	case feedback.CreateAction:
		windows = []window{{15 * time.Minute, 5}, {24 * time.Hour, 20}}
	case feedback.ReplyAction:
		windows = []window{{time.Hour, 30}}
	case feedback.TransitionAction:
		windows = []window{{time.Hour, 120}}
	default:
		return auth.ErrInvalidInput
	}
	retry := time.Time{}
	for _, w := range windows {
		var n int
		e := tx.QueryRowContext(ctx, `SELECT count(*) FROM feedback_rate_limits WHERE actor_user_id=$1 AND scope=$2 AND consumed_at>$3 AND consumed_at<=$4`, actor, action, now.Add(-w.span), now).Scan(&n)
		if e != nil {
			return e
		}
		if n >= w.limit {
			var earliest time.Time
			e = tx.QueryRowContext(ctx, `SELECT consumed_at FROM feedback_rate_limits WHERE actor_user_id=$1 AND scope=$2 AND consumed_at>$3 AND consumed_at<=$4 ORDER BY consumed_at,id OFFSET $5 LIMIT 1`, actor, action, now.Add(-w.span), now, n-w.limit).Scan(&earliest)
			if e != nil {
				return e
			}
			at := earliest.Add(w.span)
			if at.After(retry) {
				retry = at
			}
		}
	}
	if !retry.IsZero() {
		return &feedback.RateError{RetryAt: retry.UTC()}
	}
	_, e := tx.ExecContext(ctx, `INSERT INTO feedback_rate_limits(actor_user_id,scope,consumed_at) VALUES($1,$2,$3)`, actor, action, now)
	return e
}
