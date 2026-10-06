package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/study"
)

func studyReplay[T any](ctx context.Context, tx *sql.Tx, actor string, a study.Access, action study.Action, id string, in any) (T, bool, error) {
	var out T
	sha, e := study.InputDigest(in)
	if e != nil {
		return out, false, e
	}
	var prior string
	var raw []byte
	e = tx.QueryRowContext(ctx, "SELECT input_sha,receipt FROM study_idempotency WHERE owner_user_id=$1 AND action=$2 AND knowledge_id=$3 AND key=$4", actor, action, id, a.IdempotencyKey).Scan(&prior, &raw)
	if errors.Is(e, sql.ErrNoRows) {
		return out, false, nil
	}
	if e != nil {
		return out, false, e
	}
	if sha != prior {
		return out, false, study.ErrIdempotencyConflict
	}
	if len(raw) > 2<<20 || json.Unmarshal(raw, &out) != nil {
		return out, false, study.ErrNotConfigured
	}
	return out, true, nil
}
func studyRemember(ctx context.Context, tx *sql.Tx, actor string, a study.Access, action study.Action, id string, in, out any) error {
	if action == study.SaveNote || action == study.DeleteNote {
		if _, ok := out.(study.NoteReceipt); !ok {
			return study.ErrInvalid
		}
	}
	sha, e := study.InputDigest(in)
	if e != nil {
		return e
	}
	raw, e := json.Marshal(out)
	if e != nil || len(raw) > 2<<20 {
		return study.ErrInvalid
	}
	_, e = tx.ExecContext(ctx, "INSERT INTO study_idempotency(owner_user_id,action,knowledge_id,key,input_sha,receipt) VALUES($1,$2,$3,$4,$5,$6)", actor, action, id, a.IdempotencyKey, sha, string(raw))
	return e
}
