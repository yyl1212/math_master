package store

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"time"
)

type correctionCommandKey struct{}

func correctionWithCommand(ctx context.Context, action, resource, key string) context.Context {
	return context.WithValue(ctx, correctionCommandKey{}, action+":"+resource+":"+key)
}
func correctionConsumeRate(ctx context.Context, tx *sql.Tx, actor, scope string, now time.Time) error {
	limit := 0
	switch scope {
	case "create", "retry":
		limit = 20
	case "process":
		limit = 120
	case "notification-read":
		limit = 300
	default:
		return auth.ErrInvalidInput
	}
	key, ok := ctx.Value(correctionCommandKey{}).(string)
	if !ok || key == "" {
		return auth.ErrInvalidInput
	}
	var n int
	if e := tx.QueryRowContext(ctx, `SELECT count(*) FROM correction_rate_limits WHERE owner_user_id=$1 AND scope=$2 AND consumed_at>$3 AND consumed_at<=$4`, actor, scope, now.Add(-time.Hour), now).Scan(&n); e != nil {
		return e
	}
	if n >= limit {
		var at time.Time
		if e := tx.QueryRowContext(ctx, `SELECT consumed_at FROM correction_rate_limits WHERE owner_user_id=$1 AND scope=$2 AND consumed_at>$3 AND consumed_at<=$4 ORDER BY consumed_at,id OFFSET $5 LIMIT 1`, actor, scope, now.Add(-time.Hour), now, n-limit).Scan(&at); e != nil {
			return e
		}
		return &correction.RateError{RetryAt: at.Add(time.Hour).UTC()}
	}
	_, e := tx.ExecContext(ctx, `INSERT INTO correction_rate_limits(owner_user_id,scope,command_key,consumed_at) VALUES($1,$2,$3,$4)`, actor, scope, key, now)
	return e
}
