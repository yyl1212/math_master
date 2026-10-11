package store

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
)

// Legacy schema compatibility is read-only knowledge of the mode; it never grants a role.
func managedBusinessMode(ctx context.Context, tx *sql.Tx) (bool, error) {
	var tables int
	var markerColumn, applied bool
	if e := tx.QueryRowContext(ctx, `SELECT
 (SELECT count(*) FROM unnest($1::text[]) name WHERE to_regclass('public.'||name) IS NOT NULL),
 EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name='goose_db_version' AND column_name='managed_knowledge_enabled'),
 EXISTS(SELECT 1 FROM goose_db_version WHERE version_id=13 AND is_applied)`, managedTables).Scan(&tables, &markerColumn, &applied); e != nil {
		return false, e
	}
	if tables == 0 && !markerColumn && !applied {
		return false, nil
	}
	if tables != len(managedTables) || !markerColumn || !applied {
		return false, auth.ErrUnavailable
	}
	var mode string
	var once, marker bool
	if e := tx.QueryRowContext(ctx, "SELECT content_mode,enabled_once,(SELECT managed_knowledge_enabled FROM goose_db_version WHERE version_id=0) FROM knowledge_admin_state WHERE singleton").Scan(&mode, &once, &marker); e != nil {
		return false, e
	}
	if mode == "managed" && once && marker {
		return true, nil
	}
	if mode == "legacy" && !once && !marker {
		return false, nil
	}
	return false, auth.ErrUnavailable
}

// Same fence order as permanent managed activation: admin, content, knowledge.
func managedCorrectionGuard(ctx context.Context, tx *sql.Tx) error {
	if e := learningLocks(ctx, tx); e != nil {
		return e
	}
	if _, e := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock_shared($1)", knowledgeLockID); e != nil {
		return e
	}
	managed, e := managedBusinessMode(ctx, tx)
	if e != nil {
		return e
	}
	if managed {
		return correction.ErrRetired
	}
	return nil
}
