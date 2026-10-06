package store

import (
	"context"
	"database/sql"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"time"
)

const workflowTimeout = 8 * time.Second

func workflowError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, content.ErrLimit) {
		return publication.ErrContentLimitExceeded
	}
	if errors.Is(err, content.ErrValidation) || errors.Is(err, ErrInvalidPackage) {
		return publication.ErrContentInvalid
	}
	if errors.Is(err, ErrImmutableConflict) {
		return publication.ErrImmutableConflict
	}
	for _, known := range []error{taxonomy.ErrInvalid, taxonomy.ErrNotConfigured, taxonomy.ErrLimit, taxonomy.ErrHeadStale, taxonomy.ErrConflict, taxonomy.ErrIdempotencyConflict, correction.ErrNotConfigured, publication.ErrDraftConflict, publication.ErrReviewConflict, publication.ErrImmutableConflict, publication.ErrIdempotencyConflict, publication.ErrVersionConflict, publication.ErrPublicationStale, publication.ErrContentNotReady, publication.ErrContentInvalid, publication.ErrReviewRequired, publication.ErrContentLimitExceeded, publication.ErrContentNotConfigured} {
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
	user, session, now, err := managedIdentity(ctx, tx, a, !publication.IsRead(action), lock, related)
	if err != nil {
		return user, now, err
	}
	if err = publication.Authorize(user, action); err != nil {
		return user, now, err
	}
	if action == publication.ActivateReleaseAction || action == publication.WithdrawVersionAction {
		if err = managedReauth(session, now); err != nil {
			return user, now, err
		}
	}
	return user, now, nil
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
	// Content row waits and validation can outlive the proof checked at entry.
	if _, _, err = workflowIdentity(ctx, tx, a, action, false, nil); err != nil {
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
