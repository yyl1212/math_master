package store

import (
	"context"
	"database/sql"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"sort"
	"time"
)

const workflowTimeout = 8 * time.Second

func workflowError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrImmutableConflict) {
		return publication.ErrImmutableConflict
	}
	for _, known := range []error{publication.ErrDraftConflict, publication.ErrReviewConflict, publication.ErrImmutableConflict, publication.ErrIdempotencyConflict, publication.ErrVersionConflict, publication.ErrPublicationStale, publication.ErrContentNotReady, publication.ErrContentInvalid, publication.ErrReviewRequired, publication.ErrContentLimitExceeded, publication.ErrContentNotConfigured} {
		if errors.Is(err, known) {
			return known
		}
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "42P01" {
		return publication.ErrContentNotConfigured
	}
	return authError(err)
}
func workflowConfigured(ctx context.Context, tx *sql.Tx) error {
	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT to_regclass('public.content_workspaces') IS NOT NULL`).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return publication.ErrContentNotConfigured
	}
	return nil
}
func workflowIdentity(ctx context.Context, tx *sql.Tx, a publication.Access, action publication.Action, lock bool, related []string) (auth.User, time.Time, error) {
	var id string
	err := tx.QueryRowContext(ctx, `SELECT user_id::text FROM auth_sessions WHERE token_hash=$1`, a.TokenHash[:]).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return auth.User{}, time.Time{}, auth.ErrAuthenticationRequired
	}
	if err != nil {
		return auth.User{}, time.Time{}, err
	}
	if lock {
		ids := append(append([]string{}, related...), id)
		sort.Strings(ids)
		for i, userID := range ids {
			if !publication.ValidID(userID) {
				return auth.User{}, time.Time{}, auth.ErrInvalidInput
			}
			if i > 0 && ids[i-1] == userID {
				continue
			}
			if _, err = readAccount(ctx, tx, userID, true); err != nil {
				return auth.User{}, time.Time{}, err
			}
		}
	}
	row, err := readSessionRow(ctx, tx, a.TokenHash, lock)
	if err != nil {
		return auth.User{}, time.Time{}, err
	}
	account, err := readAccount(ctx, tx, id, false)
	if err != nil {
		return auth.User{}, time.Time{}, err
	}
	now, err := dbClock(ctx, tx)
	if err != nil {
		return auth.User{}, now, err
	}
	if row.UserID != id || !validSession(row, account.Version, now) {
		return auth.User{}, now, auth.ErrAuthenticationRequired
	}
	if !publication.IsRead(action) && !auth.EqualSecret(a.CSRF, row.CSRF) {
		return auth.User{}, now, auth.ErrCSRF
	}
	if err = publication.Authorize(account.User, action); err != nil {
		return auth.User{}, now, err
	}
	if action == publication.ActivateReleaseAction || action == publication.WithdrawVersionAction {
		if !row.Reauth.Valid || now.Before(row.Reauth.Time) || !now.Before(row.Reauth.Time.Add(5*time.Minute)) {
			return auth.User{}, now, auth.ErrReauthRequired
		}
	}
	return account.User, now, nil
}
func (s *Store) workflowTx(ctx context.Context, a publication.Access, action publication.Action, related []string, fn func(context.Context, *sql.Tx, auth.User, time.Time) error) error {
	ctx, cancel := context.WithTimeout(ctx, workflowTimeout)
	defer cancel()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return workflowError(err)
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `SET LOCAL lock_timeout='1s'`); err != nil {
		return workflowError(err)
	}
	if err = workflowConfigured(ctx, tx); err != nil {
		return workflowError(err)
	}
	for _, lock := range []int64{adminLockID, 1296127048} {
		if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, lock); err != nil {
			return workflowError(err)
		}
	}
	user, now, err := workflowIdentity(ctx, tx, a, action, true, related)
	if err != nil {
		return workflowError(err)
	}
	if err = fn(ctx, tx, user, now); err != nil {
		return workflowError(err)
	}
	return workflowError(tx.Commit())
}
func (s *Store) workflowReadTx(ctx context.Context, a publication.Access, action publication.Action, fn func(context.Context, *sql.Tx, auth.User) error) error {
	ctx, cancel := context.WithTimeout(ctx, workflowTimeout)
	defer cancel()
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return workflowError(err)
	}
	defer tx.Rollback()
	if err = workflowConfigured(ctx, tx); err != nil {
		return workflowError(err)
	}
	user, _, err := workflowIdentity(ctx, tx, a, action, false, nil)
	if err != nil {
		return workflowError(err)
	}
	if err = fn(ctx, tx, user); err != nil {
		return workflowError(err)
	}
	return workflowError(tx.Commit())
}
func (s *Store) Preflight(ctx context.Context, a publication.Access, action publication.Action) (auth.User, error) {
	var out auth.User
	err := s.workflowReadTx(ctx, a, action, func(_ context.Context, _ *sql.Tx, u auth.User) error { out = u; return nil })
	return out, err
}
