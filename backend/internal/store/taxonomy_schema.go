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
	e = tx.QueryRowContext(ctx, `SELECT count(*) FROM (VALUES
 ('taxonomy_batch_immutable','taxonomy_source_batches','reject_content_update()',27),
 ('taxonomy_version_immutable','taxonomy_versions','reject_content_update()',27),
 ('taxonomy_node_immutable','taxonomy_nodes','reject_content_update()',27),
 ('taxonomy_submission_immutable','taxonomy_submission_assignments','reject_content_update()',27),
 ('taxonomy_review_immutable','taxonomy_review_bindings','reject_content_update()',27),
 ('taxonomy_receipt_immutable','taxonomy_idempotency','reject_content_update()',27),
 ('taxonomy_release_guard','taxonomy_releases','guard_taxonomy_release()',27),
 ('taxonomy_member_guard','taxonomy_release_assignments','guard_taxonomy_members()',31)
 ) expected(name,table_name,function_name,event_bits)
 JOIN pg_trigger t ON t.tgname=expected.name AND t.tgrelid=to_regclass('public.'||expected.table_name) AND t.tgfoid=to_regprocedure('public.'||expected.function_name) AND t.tgtype=expected.event_bits AND NOT t.tgisinternal AND t.tgenabled IN ('O','A')`).Scan(&n)
	if e != nil {
		return e
	}
	if n != 8 {
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
