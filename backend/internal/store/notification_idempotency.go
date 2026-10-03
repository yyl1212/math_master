package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/notification"
	"github.com/yyl1212/math_master/backend/internal/question"
)

func notificationReplay(ctx context.Context, tx *sql.Tx, actor, resource, key, digest string) (notification.ReadReceipt, bool, error) {
	var r notification.ReadReceipt
	var raw []byte
	var h string
	e := tx.QueryRowContext(ctx, `SELECT digest,receipt FROM correction_idempotency WHERE owner_user_id=$1 AND action='notification-read' AND resource=$2 AND key=$3`, actor, resource, key).Scan(&h, &raw)
	if errors.Is(e, sql.ErrNoRows) {
		return r, false, nil
	}
	if e != nil {
		return r, false, e
	}
	if h != digest {
		return r, false, question.ErrIdempotencyConflict
	}
	if len(raw) > correction.MaxResponseBytes || json.Unmarshal(raw, &r) != nil || r.Status != 200 || r.NotificationID != resource || r.ReadAt.IsZero() {
		return r, false, auth.ErrUnavailable
	}
	return r, true, nil
}
func notificationRemember(ctx context.Context, tx *sql.Tx, actor, resource, key, digest string, r notification.ReadReceipt) error {
	raw, e := json.Marshal(r)
	if e != nil {
		return e
	}
	if len(raw) > correction.MaxResponseBytes {
		return question.ErrLimitExceeded
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO correction_idempotency(owner_user_id,action,resource,key,digest,status,receipt) VALUES($1,'notification-read',$2,$3,$4,200,$5)`, actor, resource, key, digest, raw)
	return e
}
