package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/question"
)

func correctionReplay(ctx context.Context, tx *sql.Tx, actor, action, resource, key, digest string) (correction.Receipt, bool, error) {
	var raw []byte
	var h string
	var r correction.Receipt
	e := tx.QueryRowContext(ctx, `SELECT digest,receipt FROM correction_idempotency WHERE owner_user_id=$1 AND action=$2 AND resource=$3 AND key=$4`, actor, action, resource, key).Scan(&h, &raw)
	if errors.Is(e, sql.ErrNoRows) {
		return r, false, nil
	}
	if e != nil {
		return r, false, e
	}
	if h != digest {
		return r, false, question.ErrIdempotencyConflict
	}
	if len(raw) > correction.MaxResponseBytes || json.Unmarshal(raw, &r) != nil {
		return r, false, auth.ErrUnavailable
	}
	return r, true, nil
}
func correctionRemember(ctx context.Context, tx *sql.Tx, actor, action, resource, key, digest string, r correction.Receipt) error {
	raw, e := json.Marshal(r)
	if e != nil {
		return e
	}
	if len(raw) > correction.MaxResponseBytes {
		return auth.ErrInvalidInput
	}
	_, e = tx.ExecContext(ctx, `INSERT INTO correction_idempotency(owner_user_id,action,resource,key,digest,status,receipt) VALUES($1,$2,$3,$4,$5,$6,$7)`, actor, action, resource, key, digest, r.Status, raw)
	return e
}
func correctionCommandDigest(action, resource string, in any) (string, error) {
	_, h, e := correction.Canonical("correction-command-v1", struct {
		Action   string `json:"action"`
		Resource string `json:"resource"`
		Input    any    `json:"input"`
	}{action, resource, in})
	return h, e
}
