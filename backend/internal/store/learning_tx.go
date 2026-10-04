package store

import (
	"context"
	"database/sql"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

const learningTimeout = 8 * time.Second

var learningTables = []string{"learning_records", "learning_events", "learning_path_enrollments", "learning_path_nodes", "learning_unlocks", "learning_qualification_events", "practice_attempts", "assessment_attempts", "assessment_items", "assessment_answers", "assessment_results", "learning_evidence_dependencies", "learner_answer_exposures", "learner_exposure_state", "learner_question_views", "learning_idempotency"}

// Only a never-enabled database may use legacy answer delivery without exposure.
func learningConfigured(ctx context.Context, tx *sql.Tx) (bool, error) {
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT count(*) FROM unnest($1::text[]) name WHERE to_regclass('public.'||name) IS NOT NULL`, learningTables).Scan(&count); err != nil {
		return false, err
	}
	if count == len(learningTables) {
		return true, nil
	}
	var gooseExists bool
	if err := tx.QueryRowContext(ctx, `SELECT to_regclass('public.goose_db_version') IS NOT NULL`).Scan(&gooseExists); err != nil {
		return false, err
	}
	everEnabled := false
	if gooseExists {
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM goose_db_version WHERE version_id=6)`).Scan(&everEnabled); err != nil {
			return false, err
		}
	}
	if gooseExists && !everEnabled {
		var hasMarker bool
		if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name='goose_db_version' AND column_name='learning_enabled')`).Scan(&hasMarker); err != nil {
			return false, err
		}
		if hasMarker {
			if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM goose_db_version WHERE learning_enabled)`).Scan(&everEnabled); err != nil {
				return false, err
			}
		}
	}
	if count == 0 && !everEnabled {
		return false, nil
	}
	return false, learning.ErrNotConfigured
}
func learningError(err error) error {
	if err == nil {
		return nil
	}
	for _, known := range []error{correction.ErrNotConfigured, learning.ErrNotConfigured, learning.ErrVersionStale, learning.ErrPrerequisitesUnmet, learning.ErrAssessmentNotReady, learning.ErrAssessmentActive, learning.ErrAssessmentExpired, learning.ErrStateConflict, learning.ErrAnswerFormatInvalid, question.ErrIdempotencyConflict, question.ErrInvalid, question.ErrLimitExceeded, question.ErrImmutableConflict} {
		if errors.Is(err, known) {
			return err
		}
	}
	var format *question.NumericFormatError
	if errors.As(err, &format) {
		return err
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) {
		switch pg.Code {
		case "42P01":
			return learning.ErrNotConfigured
		case "M0001":
			return learning.ErrAssessmentExpired
		}
	}
	return authError(err)
}
func learningIdentity(ctx context.Context, tx *sql.Tx, a question.Access, action learning.Action, lock bool) (auth.User, time.Time, error) {
	u, _, now, err := managedIdentity(ctx, tx, a, !learning.IsRead(action), lock, nil)
	if err != nil {
		return u, now, err
	}
	if err = learning.Authorize(u, action); err != nil {
		return u, now, err
	}
	if learning.IsIdempotent(action) && !question.ValidID(a.IdempotencyKey) {
		return u, now, auth.ErrInvalidInput
	}
	return u, now, nil
}
func learningLocks(ctx context.Context, tx *sql.Tx) error {
	if _, err := tx.ExecContext(ctx, `SET LOCAL lock_timeout='1s'`); err != nil {
		return err
	}
	for _, id := range []int64{adminLockID, 1296127048} {
		if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock_shared($1)`, id); err != nil {
			return err
		}
	}
	return nil
}
func (s *Store) learningTx(ctx context.Context, a question.Access, action learning.Action, fn func(context.Context, *sql.Tx, auth.User, time.Time) error) error {
	ctx, cancel := context.WithTimeout(ctx, learningTimeout)
	defer cancel()
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return learningError(err)
	}
	defer tx.Rollback()
	enabled, err := learningConfigured(ctx, tx)
	if err != nil {
		return learningError(err)
	}
	if !enabled {
		return learning.ErrNotConfigured
	}
	if err = questionConfigured(ctx, tx); err != nil {
		return learning.ErrNotConfigured
	}
	correctionOn, err := correctionConfigured(ctx, tx)
	if err != nil {
		return learningError(err)
	}
	ctx = correctionWithConfig(ctx, correctionOn)
	if err = learningLocks(ctx, tx); err != nil {
		return learningError(err)
	}
	if err = correctionRegistrationFence(ctx, tx, false); err != nil {
		return learningError(err)
	}
	u, now, err := learningIdentity(ctx, tx, a, action, true)
	if err != nil {
		return learningError(err)
	}
	var sequence int64
	if err = tx.QueryRowContext(ctx, `SELECT sequence FROM learner_exposure_state WHERE owner_user_id=$1 FOR UPDATE`, u.ID).Scan(&sequence); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return learningError(err)
	}
	if err = fn(ctx, tx, u, now); err != nil {
		return learningError(err)
	}
	if _, _, err = learningIdentity(ctx, tx, a, action, false); err != nil {
		return learningError(err)
	}
	return learningError(tx.Commit())
}
func (s *Store) LearningPreflight(ctx context.Context, a question.Access, action learning.Action) (auth.User, error) {
	ctx, cancel := context.WithTimeout(ctx, learningTimeout)
	defer cancel()
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted, ReadOnly: true})
	if err != nil {
		return auth.User{}, learningError(err)
	}
	defer tx.Rollback()
	enabled, err := learningConfigured(ctx, tx)
	if err != nil {
		return auth.User{}, learningError(err)
	}
	if !enabled {
		return auth.User{}, learning.ErrNotConfigured
	}
	if err = questionConfigured(ctx, tx); err != nil {
		return auth.User{}, learning.ErrNotConfigured
	}
	if err = learningLocks(ctx, tx); err != nil {
		return auth.User{}, learningError(err)
	}
	correctionOn, err := correctionConfigured(ctx, tx)
	if err != nil {
		return auth.User{}, learningError(err)
	}
	ctx = correctionWithConfig(ctx, correctionOn)
	if err = correctionRegistrationFence(ctx, tx, false); err != nil {
		return auth.User{}, learningError(err)
	}
	u, _, err := learningIdentity(ctx, tx, a, action, false)
	if err != nil {
		return auth.User{}, learningError(err)
	}
	return u, learningError(tx.Commit())
}
