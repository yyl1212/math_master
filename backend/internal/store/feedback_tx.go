package store

import (
	"context"
	"database/sql"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

var feedbackTables = []string{"feedback_tickets", "feedback_events", "feedback_idempotency", "feedback_rate_limits"}

func feedbackConfigured(ctx context.Context, tx *sql.Tx) (bool, error) {
	var n int
	if e := tx.QueryRowContext(ctx, `SELECT count(*) FROM unnest($1::text[]) name WHERE to_regclass('public.'||name) IS NOT NULL`, feedbackTables).Scan(&n); e != nil {
		return false, e
	}
	if n != 4 {
		return false, feedback.ErrNotConfigured
	}
	ok, e := learningConfigured(ctx, tx)
	if e != nil || !ok {
		return false, feedback.ErrNotConfigured
	}
	if e = questionConfigured(ctx, tx); e != nil {
		return false, feedback.ErrNotConfigured
	}
	return true, nil
}
func feedbackError(e error) error {
	if e == nil {
		return nil
	}
	for _, known := range []error{feedback.ErrNotConfigured, feedback.ErrConflict, feedback.ErrTargetStale, feedback.ErrAnswerOverlap, question.ErrIdempotencyConflict, auth.ErrInvalidInput, auth.ErrNotFound} {
		if errors.Is(e, known) {
			return e
		}
	}
	var rate *feedback.RateError
	if errors.As(e, &rate) {
		return e
	}
	var pg *pgconn.PgError
	if errors.As(e, &pg) && (pg.Code == "42P01" || pg.Code == "42883") {
		return feedback.ErrNotConfigured
	}
	return authError(e)
}
func feedbackIdentity(ctx context.Context, tx *sql.Tx, a question.Access, action feedback.Action, lock bool, related []string) (auth.User, time.Time, error) {
	u, _, now, e := managedIdentity(ctx, tx, a, feedback.IsWrite(action), lock, related)
	if e != nil {
		return u, now, e
	}
	if e = feedback.Authorize(u, action); e != nil {
		return u, now, e
	}
	if feedback.IsWrite(action) && !question.ValidID(a.IdempotencyKey) {
		return u, now, auth.ErrInvalidInput
	}
	return u, now, nil
}
func (s *Store) feedbackTx(ctx context.Context, a question.Access, action feedback.Action, related []string, fn func(context.Context, *sql.Tx, auth.User, time.Time) error) error {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if e != nil {
		return feedbackError(e)
	}
	defer tx.Rollback()
	if _, e = feedbackConfigured(ctx, tx); e != nil {
		return feedbackError(e)
	}
	if e = learningLocks(ctx, tx); e != nil {
		return feedbackError(e)
	}
	u, now, e := feedbackIdentity(ctx, tx, a, action, true, related)
	if e != nil {
		return feedbackError(e)
	}
	if e = fn(ctx, tx, u, now); e != nil {
		return feedbackError(e)
	}
	if _, _, e = feedbackIdentity(ctx, tx, a, action, false, nil); e != nil {
		return feedbackError(e)
	}
	return feedbackError(tx.Commit())
}
func (s *Store) FeedbackPreflight(ctx context.Context, a question.Access, action feedback.Action) (auth.User, error) {
	// Preflight never reserves or consumes a command quota.
	var user auth.User
	e := s.feedbackTx(ctx, a, action, nil, func(_ context.Context, _ *sql.Tx, u auth.User, _ time.Time) error { user = u; return nil })
	return user, e
}

var _ feedback.Repository = (*Store)(nil)
