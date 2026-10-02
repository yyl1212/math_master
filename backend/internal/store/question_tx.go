package store

import (
	"context"
	"database/sql"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

const questionTimeout = 8 * time.Second

func questionError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrImmutableConflict) {
		return question.ErrImmutableConflict
	}
	for _, known := range []error{question.ErrDraftConflict, question.ErrPublicationStale, question.ErrReviewConflict, question.ErrIdempotencyConflict, question.ErrImmutableConflict, question.ErrVersionConflict, question.ErrInvalid, question.ErrNotReady, question.ErrLimitExceeded, question.ErrReviewRequired, question.ErrNotConfigured} {
		if errors.Is(err, known) {
			return known
		}
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "42P01" {
		return question.ErrNotConfigured
	}
	return authError(err)
}

var questionTables = []string{"question_workspaces", "question_workspace_authors", "question_packages", "question_templates", "question_instances", "question_blueprints", "question_instance_coverage", "question_blueprint_sources", "question_submissions", "question_submission_authors", "question_submission_members", "question_review_decisions", "question_publications", "question_publication_members", "question_heads", "question_withdrawals", "question_events", "question_idempotency"}

func questionConfigured(ctx context.Context, tx *sql.Tx) error {
	var n int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM unnest($1::text[]) name WHERE to_regclass('public.'||name) IS NOT NULL`, questionTables).Scan(&n); err != nil {
		return err
	}
	if n != len(questionTables) {
		return question.ErrNotConfigured
	}
	return nil
}
func questionIdentity(ctx context.Context, tx *sql.Tx, a question.Access, action question.Action, lock bool, related []string) (auth.User, time.Time, error) {
	user, session, now, err := managedIdentity(ctx, tx, a, !question.IsRead(action), lock, related)
	if err != nil {
		return user, now, err
	}
	if err = question.Authorize(user, action); err != nil {
		return user, now, err
	}
	if action == question.ActivateReleaseAction || action == question.WithdrawVersionAction {
		if err = managedReauth(session, now); err != nil {
			return user, now, err
		}
	}
	return user, now, nil
}
func (s *Store) questionTx(ctx context.Context, a question.Access, action question.Action, related []string, fn func(context.Context, *sql.Tx, auth.User, time.Time) error) error {
	ctx, cancel := context.WithTimeout(ctx, questionTimeout)
	defer cancel()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return questionError(err)
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `SET LOCAL lock_timeout='1s'`); err != nil {
		return questionError(err)
	}
	if err = questionConfigured(ctx, tx); err != nil {
		return questionError(err)
	}
	for _, lock := range []int64{adminLockID, 1296127048} {
		if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, lock); err != nil {
			return questionError(err)
		}
	}
	user, now, err := questionIdentity(ctx, tx, a, action, true, related)
	if err != nil {
		return questionError(err)
	}
	if err = fn(ctx, tx, user, now); err != nil {
		return questionError(err)
	}
	// Content row waits and validation can outlive the proof checked at entry.
	if _, _, err = questionIdentity(ctx, tx, a, action, false, nil); err != nil {
		return questionError(err)
	}
	return questionError(tx.Commit())
}
func (s *Store) questionReadTx(ctx context.Context, a question.Access, action question.Action, fn func(context.Context, *sql.Tx, auth.User) error) error {
	ctx, cancel := context.WithTimeout(ctx, questionTimeout)
	defer cancel()
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true})
	if err != nil {
		return questionError(err)
	}
	defer tx.Rollback()
	if err = questionConfigured(ctx, tx); err != nil {
		return questionError(err)
	}
	user, _, err := questionIdentity(ctx, tx, a, action, false, nil)
	if err != nil {
		return questionError(err)
	}
	if err = fn(ctx, tx, user); err != nil {
		return questionError(err)
	}
	return questionError(tx.Commit())
}
func (s *Store) QuestionPreflight(ctx context.Context, a question.Access, action question.Action) (auth.User, error) {
	var out auth.User
	err := s.questionReadTx(ctx, a, action, func(_ context.Context, _ *sql.Tx, u auth.User) error { out = u; return nil })
	return out, err
}
