package store

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/studyreview"
	"time"
)

// Private reads take the original identity locks under RC so commit-time revocation is fresh.
// GET never requires write CSRF/idempotency and never updates business rows.
func (s *Store) studyReviewReadTx(ctx context.Context, a studyreview.Access, fn func(context.Context, *sql.Tx, auth.User) error) error {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if e != nil {
		return knowledgeError(e)
	}
	defer tx.Rollback()
	if _, e = tx.ExecContext(ctx, "SET LOCAL lock_timeout='1s'"); e != nil {
		return knowledgeError(e)
	}
	for _, id := range []int64{adminLockID, 1296127048} {
		if _, e = tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock_shared($1)", id); e != nil {
			return knowledgeError(e)
		}
	}
	u, e := knowledgeIdentity(ctx, tx, a, false, true, false)
	if e != nil {
		return knowledgeError(e)
	}
	if e = knowledgeConfigured(ctx, tx); e != nil {
		return knowledgeError(e)
	}
	if e = studyReviewConfigured(ctx, tx); e != nil {
		return knowledgeError(e)
	}
	if e = fn(ctx, tx, u); e != nil {
		return knowledgeError(e)
	}
	if _, e = knowledgeIdentity(ctx, tx, a, false, false, false); e != nil {
		return knowledgeError(e)
	}
	return knowledgeError(tx.Commit())
}
