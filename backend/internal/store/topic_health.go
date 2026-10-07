package store

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/study"
	"time"
)

func (s *Store) ReadTopicSchemaHealth(ctx context.Context) (study.SchemaHealth, error) {
	out := study.SchemaHealth{Taxonomy: true, Study: true, Retirement: true}
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	tx, e := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true, Isolation: sql.LevelRepeatableRead})
	if e != nil {
		return out, studyError(e)
	}
	defer tx.Rollback()
	if _, e = optionalTaxonomyConfigured(ctx, tx); e != nil {
		return out, nil
	}
	if _, e = optionalStudyConfigured(ctx, tx); e != nil {
		return out, nil
	}
	if _, e = optionalCutoverConfigured(ctx, tx); e != nil {
		return out, nil
	}
	var state bool
	if e = tx.QueryRowContext(ctx, "SELECT to_regclass('public.topic_learning_state') IS NOT NULL").Scan(&state); e != nil {
		return out, studyError(e)
	}
	if state {
		var mode string
		if e = tx.QueryRowContext(ctx, "SELECT experience_mode FROM topic_learning_state WHERE singleton").Scan(&mode); e != nil || mode != "legacy" && mode != "topics" {
			return out, nil
		}
		out.TopicsMode = mode == "topics"
		if out.TopicsMode {
			if e = cutoverConfigured(ctx, tx); e != nil {
				return out, nil
			}
		}
	}
	out.SchemaReady = true
	return out, studyError(tx.Commit())
}
