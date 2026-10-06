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
	var exists bool
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
		return taxonomy.ModeLegacy, studyError(tx.Commit())
	}
	var mode taxonomy.ExperienceMode
	if e = tx.QueryRowContext(ctx, "SELECT experience_mode FROM topic_learning_state WHERE singleton").Scan(&mode); e != nil {
		return "", study.ErrNotConfigured
	}
	if mode != taxonomy.ModeLegacy && mode != taxonomy.ModeTopics {
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
	return mode, studyError(tx.Commit())
}
