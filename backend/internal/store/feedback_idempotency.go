package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"github.com/yyl1212/math_master/backend/internal/question"
)

func feedbackReplay(ctx context.Context, tx *sql.Tx, actor, action, resource, key, digest string) (feedback.Receipt, bool, error) {
	var raw []byte
	var sha string
	r := feedback.Receipt{}
	e := tx.QueryRowContext(ctx, `SELECT request_sha256,receipt FROM feedback_idempotency WHERE actor_user_id=$1 AND action=$2 AND resource=$3 AND key=$4`, actor, action, resource, key).Scan(&sha, &raw)
	if errors.Is(e, sql.ErrNoRows) {
		return r, false, nil
	}
	if e != nil {
		return r, false, e
	}
	if sha != digest {
		return r, false, question.ErrIdempotencyConflict
	}
	if json.Unmarshal(raw, &r) != nil {
		return r, false, ErrUnavailable
	}
	return r, true, nil
}
func feedbackRemember(ctx context.Context, tx *sql.Tx, actor, action, resource, key, digest string, r feedback.Receipt) error {
	_, e := tx.ExecContext(ctx, `INSERT INTO feedback_idempotency(actor_user_id,action,resource,key,request_sha256,ticket_id,event_sequence,receipt) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, actor, action, resource, key, digest, r.Ticket.ID, r.Ticket.Sequence, body(r))
	return e
}
