package store

import (
	"context"
	"database/sql"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
)

var taxonomyTables = []string{"taxonomy_source_batches", "taxonomy_versions", "taxonomy_nodes", "taxonomy_assignment_drafts", "taxonomy_submission_assignments", "taxonomy_review_bindings", "taxonomy_releases", "taxonomy_release_assignments", "taxonomy_heads", "taxonomy_idempotency", "topic_learning_state"}

func taxonomyConfigured(ctx context.Context, tx *sql.Tx) error {
	var n int
	e := tx.QueryRowContext(ctx, "SELECT count(*) FROM unnest($1::text[]) name WHERE to_regclass('public.'||name) IS NOT NULL", taxonomyTables).Scan(&n)
	if e != nil {
		return e
	}
	if n != len(taxonomyTables) {
		return taxonomy.ErrNotConfigured
	}
	return nil
}
func taxonomyError(e error) error {
	if e == nil {
		return nil
	}
	if errors.Is(e, sql.ErrNoRows) {
		return ErrNotFound
	}
	if errors.Is(e, taxonomy.ErrInvalid) || errors.Is(e, taxonomy.ErrNotConfigured) || errors.Is(e, taxonomy.ErrConflict) || errors.Is(e, taxonomy.ErrHeadStale) || errors.Is(e, taxonomy.ErrIdempotencyConflict) {
		return e
	}
	return readError(e)
}
