package store

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
)

var publishTables = []string{"managed_knowledge_publish_jobs", "managed_knowledge_publish_items", "managed_knowledge_publish_receipts"}

const publishSchemaDigest = "b709ad5e16cd5e3f124a8135c323d0d2"

func publishSchemaIntegrity(ctx context.Context, tx *sql.Tx) error {
	var count int
	var installed bool
	if e := tx.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM unnest($1::text[]) n WHERE to_regclass('public.'||n) IS NOT NULL),EXISTS(SELECT 1 FROM public.goose_db_version WHERE version_id=16 AND is_applied)`, publishTables).Scan(&count, &installed); e != nil || count != 3 || !installed {
		return knowledgeadmin.ErrNotConfigured
	}
	var prior, digest string
	if e := tx.QueryRowContext(ctx, `SELECT current_setting('search_path')`).Scan(&prior); e != nil {
		return e
	}
	if _, e := tx.ExecContext(ctx, `SELECT set_config('search_path','pg_catalog,public',true)`); e != nil {
		return e
	}
	e := tx.QueryRowContext(ctx, managedSchemaDigestSQL, publishTables).Scan(&digest)
	_, restore := tx.ExecContext(ctx, `SELECT set_config('search_path',$1,true)`, prior)
	if e != nil || restore != nil || digest != publishSchemaDigest {
		return knowledgeadmin.ErrNotConfigured
	}
	return nil
}
