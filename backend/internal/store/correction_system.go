package store

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"time"
)

const correctionSystemExistenceSQL = `SELECT
 (SELECT count(*) FROM unnest($1::text[]) name WHERE to_regclass('public.'||name) IS NOT NULL),
 to_regclass('public.goose_db_version') IS NOT NULL,
 EXISTS(SELECT 1 FROM pg_attribute WHERE attrelid=to_regclass('public.goose_db_version') AND attname='correction_enabled' AND NOT attisdropped),
 (SELECT count(*) FROM unnest($2::text[]) name WHERE to_regclass('public.'||name) IS NOT NULL),
 (SELECT count(*) FROM unnest($3::text[]) name WHERE to_regclass('public.'||name) IS NOT NULL)`

var correctionSystemHealthSQL = `SELECT EXISTS(SELECT 1 FROM goose_db_version WHERE version_id=8 AND is_applied),EXISTS(SELECT 1 FROM goose_db_version g WHERE version_id=0 AND coalesce((to_jsonb(g)->>'correction_enabled')::boolean,false)),(` + correctionSchemaIntegritySQL + `)`

// Every system transaction checks the same complete schema predicate. Combine
// only independent existence reads; never reuse the result in another tx.
func correctionSystemConfigured(ctx context.Context, tx *sql.Tx) (bool, error) {
	var n, learningCount, questionCount int
	var goose, markerColumn, ever, version, intact bool
	e := tx.QueryRowContext(ctx, correctionSystemExistenceSQL, correctionTables, learningTables, questionTables).Scan(&n, &goose, &markerColumn, &learningCount, &questionCount)
	if e != nil {
		return false, e
	}
	if goose {
		// to_jsonb preserves never-enabled schemas where the marker column is absent.
		// The column's physical presence remains part of the fail-closed decision.
		e = tx.QueryRowContext(ctx, correctionSystemHealthSQL, correctionSchemaIntegrityArgs()...).Scan(&version, &ever, &intact)
		if e != nil {
			return false, e
		}
	}
	if n == 0 && !ever && !version && !markerColumn {
		return false, nil
	}
	if n != len(correctionTables) || !ever || !version || !intact || learningCount != len(learningTables) || questionCount != len(questionTables) {
		return false, correction.ErrNotConfigured
	}
	return true, nil
}

func correctionSystemEnter(ctx context.Context, tx *sql.Tx, locks bool) (time.Time, error) {
	var now time.Time
	query := `WITH settings AS MATERIALIZED (SELECT set_config('lock_timeout','1s',true)) SELECT clock_timestamp() FROM settings`
	args := []any{}
	if locks {
		// Materialized producer dependencies finish each volatile lock operation
		// before the next level can obtain a row. Sample time after the final lock.
		query = `WITH settings AS MATERIALIZED (SELECT set_config('lock_timeout','1s',true)),
  admin_fence AS MATERIALIZED (SELECT pg_advisory_xact_lock_shared($1) FROM settings),
  content_fence AS MATERIALIZED (SELECT pg_advisory_xact_lock_shared($2) FROM admin_fence),
  registration_fence AS MATERIALIZED (SELECT pg_advisory_xact_lock_shared($3) FROM content_fence)
  SELECT clock_timestamp() FROM registration_fence`
		args = []any{adminLockID, int64(1296127048), correctionRegistrationLock}
	}
	e := tx.QueryRowContext(ctx, query, args...).Scan(&now)
	return now, e
}
