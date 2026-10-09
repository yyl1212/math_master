package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"strings"
	"time"
)

const knowledgeLockID int64 = 1296127059

func knowledgeError(e error) error {
	for _, known := range []error{knowledgeadmin.ErrInvalid, knowledgeadmin.ErrConflict, knowledgeadmin.ErrStale, knowledgeadmin.ErrIdempotency, knowledgeadmin.ErrNotConfigured, knowledgeadmin.ErrNotFound, knowledgeadmin.ErrBusy, knowledgeadmin.ErrRetired} {
		if errors.Is(e, known) {
			return known
		}
	}
	if errors.Is(e, sql.ErrNoRows) {
		return knowledgeadmin.ErrNotFound
	}
	var pg *pgconn.PgError
	if errors.As(e, &pg) {
		switch pg.Code {
		case "42P01", "42703":
			return knowledgeadmin.ErrNotConfigured
		case "23505":
			return knowledgeadmin.ErrConflict
		}
	}
	return authError(e)
}
func knowledgeIdentity(ctx context.Context, tx *sql.Tx, a knowledgeadmin.Access, write, lock, admin bool) (auth.User, error) {
	u, _, _, e := managedIdentity(ctx, tx, publication.Access{TokenHash: a.TokenHash, CSRF: a.CSRF, IdempotencyKey: a.IdempotencyKey, RequestID: a.RequestID}, write, lock, nil)
	if e != nil {
		return u, e
	}
	if u.MustChangePassword {
		return u, auth.ErrPasswordChangeRequired
	}
	if admin && !auth.HasRole(u, auth.RoleAdmin) {
		return u, auth.ErrForbidden
	}
	if write && (strings.TrimSpace(a.IdempotencyKey) == "" || len(a.IdempotencyKey) > 128 || a.RequestID == "" || len(a.RequestID) > 200) {
		return u, auth.ErrInvalidInput
	}
	return u, nil
}
func knowledgeConfigured(ctx context.Context, tx *sql.Tx) error {
	return managedSchemaIntegrity(ctx, tx)
}
func (s *Store) knowledgeTx(ctx context.Context, a knowledgeadmin.Access, write, admin bool, fn func(context.Context, *sql.Tx, auth.User) error) error {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	opts := &sql.TxOptions{}
	if !write {
		opts.ReadOnly = true
		opts.Isolation = sql.LevelRepeatableRead
	}
	tx, e := s.db.BeginTx(ctx, opts)
	if e != nil {
		return knowledgeError(e)
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(ctx, "SET LOCAL lock_timeout='1s'"); e != nil {
		return knowledgeError(e)
	}
	if write {
		for _, id := range []int64{adminLockID, 1296127048} {
			if _, e = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock_shared($1)", id); e != nil {
				return knowledgeError(e)
			}
		}
	}
	u, e := knowledgeIdentity(ctx, tx, a, write, write, admin)
	if e != nil {
		return knowledgeError(e)
	}
	if e = knowledgeConfigured(ctx, tx); e != nil {
		return knowledgeError(e)
	}
	if write && admin {
		if _, e = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1)", knowledgeLockID); e != nil {
			return knowledgeError(e)
		}
	}
	if e = fn(ctx, tx, u); e != nil {
		return knowledgeError(e)
	}
	if _, e = knowledgeIdentity(ctx, tx, a, write, false, admin); e != nil {
		return knowledgeError(e)
	}
	return knowledgeError(tx.Commit())
}
func (s *Store) KnowledgePreflight(ctx context.Context, a knowledgeadmin.Access) (u auth.User, e error) {
	e = s.knowledgeTx(ctx, a, true, true, func(_ context.Context, _ *sql.Tx, actor auth.User) error { u = actor; return nil })
	return
}
func knowledgeJSON(v any) []byte { b, _ := json.Marshal(v); return b }
func knowledgeFingerprint(v any) string {
	b, e := json.Marshal(v)
	if e != nil {
		return ""
	}
	b, e = knowledgeadmin.CanonicalJSON(b)
	if e != nil {
		return ""
	}
	return knowledgeadmin.DigestBytes(b)
}
func knowledgeReplay(ctx context.Context, tx *sql.Tx, u auth.User, a knowledgeadmin.Access, action, resource, digest string, out any) (bool, error) {
	var sha, storedResource string
	var b []byte
	e := tx.QueryRowContext(ctx, "SELECT input_sha256,resource,receipt FROM managed_knowledge_imports WHERE owner_user_id=$1 AND action=$2 AND key=$3", u.ID, action, a.IdempotencyKey).Scan(&sha, &storedResource, &b)
	if errors.Is(e, sql.ErrNoRows) {
		return false, nil
	}
	if e != nil {
		return false, e
	}
	if sha != digest || resource != storedResource {
		return false, knowledgeadmin.ErrIdempotency
	}
	if len(b) == 0 {
		return false, knowledgeadmin.ErrConflict
	}
	if e = json.Unmarshal(b, out); e != nil {
		return false, knowledgeadmin.ErrNotConfigured
	}
	return true, nil
}
func knowledgeReceipt(ctx context.Context, tx *sql.Tx, u auth.User, a knowledgeadmin.Access, action, resource, digest string, v any) error {
	_, e := tx.ExecContext(ctx, `INSERT INTO managed_knowledge_imports(owner_user_id,action,key,resource,input_sha256,state,receipt) VALUES($1,$2,$3,$4,$5,'applied',$6)`, u.ID, action, a.IdempotencyKey, resource, digest, knowledgeJSON(v))
	return e
}
