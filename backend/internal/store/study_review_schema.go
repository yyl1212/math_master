package store

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/studyreview"
	"strings"
)

var studyReviewTables = []string{"study_review_sessions", "study_review_snapshots", "study_review_schedules", "study_review_events", "study_review_receipts"}

// Recorded from the real migration with pg_catalog,public search_path; never auto-accept drift.
var studyReviewSchemaDigestSQL = strings.TrimSuffix(managedSchemaDigestSQL, ") checks") + `
UNION ALL SELECT 'RF:'||p.oid::regprocedure::text||':'||md5(pg_get_functiondef(p.oid)) FROM pg_proc p WHERE p.pronamespace='public'::regnamespace AND p.proname IN ('study_review_blank','study_review_material_bytes','study_review_session_guard','study_review_schedule_guard')
UNION ALL SELECT 'RI:'||i.indexrelid::regclass::text||':'||pg_get_indexdef(i.indexrelid)||':'||i.indisvalid::text||':'||i.indisready::text FROM pg_index i WHERE i.indrelid IN (SELECT oid FROM pg_class WHERE relnamespace='public'::regnamespace AND relname=ANY($1::text[]))
) checks`

const studyReviewDigest = "2da021ff441d7e514c792890c9f76a3c"

func studyReviewSchemaIntegrity(ctx context.Context, tx *sql.Tx) error {
	var count int
	var applied bool
	if e := tx.QueryRowContext(ctx, `SELECT (SELECT count(*) FROM unnest($1::text[]) name WHERE to_regclass('public.'||name) IS NOT NULL),EXISTS(SELECT 1 FROM public.goose_db_version WHERE version_id=15 AND is_applied)`, studyReviewTables).Scan(&count, &applied); e != nil || count != 5 || !applied {
		return studyreview.ErrNotConfigured
	}
	var prior, digest string
	if e := tx.QueryRowContext(ctx, `SELECT current_setting('search_path')`).Scan(&prior); e != nil {
		return studyreview.ErrNotConfigured
	}
	if _, e := tx.ExecContext(ctx, `SELECT set_config('search_path','pg_catalog,public',true)`); e != nil {
		return studyreview.ErrNotConfigured
	}
	e := tx.QueryRowContext(ctx, studyReviewSchemaDigestSQL, studyReviewTables).Scan(&digest)
	_, restore := tx.ExecContext(ctx, `SELECT set_config('search_path',$1,true)`, prior)
	if e != nil || restore != nil || digest != studyReviewDigest {
		return studyreview.ErrNotConfigured
	}
	return nil
}
func studyReviewConfigured(ctx context.Context, tx *sql.Tx) error {
	if e := managedStudyMode(ctx, tx); e != nil {
		return e
	}
	return studyReviewSchemaIntegrity(ctx, tx)
}
