package store

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/study"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
)

func (s *Store) ReadExperienceMode(ctx context.Context) (taxonomy.ExperienceMode, error) {
	ctx, cancel := context.WithTimeout(ctx, studyTimeout)
	defer cancel()
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if e != nil {
		return "", studyError(e)
	}
	defer tx.Rollback()
	mode, e := readExperienceModeTx(ctx, tx, false)
	if e != nil {
		return "", studyError(e)
	}
	return mode, studyError(tx.Commit())
}
func readExperienceModeTx(ctx context.Context, tx *sql.Tx, lock bool) (taxonomy.ExperienceMode, error) {
	var exists bool
	var e error
	if e = tx.QueryRowContext(ctx, "SELECT to_regclass('public.topic_learning_state') IS NOT NULL").Scan(&exists); e != nil {
		return "", studyError(e)
	}
	if !exists {
		enabled, e := optionalStudyConfigured(ctx, tx)
		if e != nil || enabled {
			return "", study.ErrNotConfigured
		}
		var n int
		if e = tx.QueryRowContext(ctx, "SELECT count(*) FROM unnest($1::text[]) name WHERE to_regclass('public.'||name) IS NOT NULL", taxonomyTables).Scan(&n); e != nil {
			return "", studyError(e)
		}
		if n != 0 {
			return "", study.ErrNotConfigured
		}
		return taxonomy.ModeLegacy, nil
	}
	var mode taxonomy.ExperienceMode
	query := "SELECT experience_mode FROM topic_learning_state WHERE singleton"
	if lock {
		query += " FOR SHARE"
	}
	if e := tx.QueryRowContext(ctx, query).Scan(&mode); e != nil {
		return "", study.ErrNotConfigured
	}
	if mode != taxonomy.ModeLegacy && mode != taxonomy.ModeTopics {
		return "", study.ErrNotConfigured
	}
	if e = taxonomyConfigured(ctx, tx); e != nil {
		return "", study.ErrNotConfigured
	}
	if _, e = optionalStudyConfigured(ctx, tx); e != nil {
		return "", studyError(e)
	}
	if mode == taxonomy.ModeTopics {
		if e = studyConfigured(ctx, tx); e != nil {
			return "", studyError(e)
		}
		if _, e = studyScopeTx(ctx, tx); e != nil {
			return "", studyError(e)
		}
	}
	return mode, nil
}

// Lock experience configuration before all established global and actor locks.
// Cutover takes FOR UPDATE on this same row, so an old write cannot outlive it.
func topicLegacyWriteGuard(ctx context.Context, tx *sql.Tx, lock bool) error {
	if lock {
		if _, e := tx.ExecContext(ctx, "SET LOCAL lock_timeout='1s'"); e != nil {
			return e
		}
	}
	mode, e := readExperienceModeTx(ctx, tx, lock)
	if e != nil {
		return studyError(e)
	}
	if mode == taxonomy.ModeTopics {
		return study.ErrModuleRetired
	}
	return nil
}
