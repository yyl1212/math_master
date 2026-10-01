package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/publication"
)

func workflowKey(actor, route, key, digest string) error {
	if !publication.ValidID(actor) || !publication.ValidID(key) || !publication.ValidSHA(digest) || !publication.IsIdempotent(publication.Action(route)) {
		return auth.ErrInvalidInput
	}
	if err := publication.Authorize(auth.User{ID: actor, Roles: []auth.Role{auth.RoleEditor, auth.RoleReviewer, auth.RoleAdmin}}, publication.Action(route)); err != nil {
		return err
	}
	return nil
}

// Call only inside workflowTx after current identity and roles have been checked.
func (s *Store) workflowReplay(ctx context.Context, tx *sql.Tx, actor, route, key, digest string) ([]byte, bool, error) {
	if err := workflowKey(actor, route, key, digest); err != nil {
		return nil, false, err
	}
	var hash string
	var result []byte
	err := tx.QueryRowContext(ctx, `SELECT request_sha256,result_bytes FROM content_idempotency WHERE actor_user_id=$1 AND route=$2 AND key=$3`, actor, route, key).Scan(&hash, &result)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if hash != digest {
		return nil, false, publication.ErrIdempotencyConflict
	}
	if len(result) > 4<<20 || !json.Valid(result) {
		return nil, false, auth.ErrUnavailable
	}
	return result, true, nil
}
func (s *Store) workflowRemember(ctx context.Context, tx *sql.Tx, actor, route, key, digest string, result []byte) error {
	if err := workflowKey(actor, route, key, digest); err != nil {
		return err
	}
	if len(result) > 4<<20 {
		return publication.ErrContentLimitExceeded
	}
	if !json.Valid(result) {
		return auth.ErrInvalidInput
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO content_idempotency(actor_user_id,route,key,request_sha256,result_bytes) VALUES($1,$2,$3,$4,$5)`, actor, route, key, digest, result)
	return err
}
