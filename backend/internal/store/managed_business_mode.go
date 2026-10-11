package store

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/auth"
)

// Legacy schema compatibility is read-only knowledge of the mode; it never grants a role.
func managedBusinessMode(ctx context.Context, tx *sql.Tx) (bool, error) {
	var present bool
	if e := tx.QueryRowContext(ctx, "SELECT to_regclass('public.knowledge_admin_state') IS NOT NULL").Scan(&present); e != nil {
		return false, e
	}
	if !present {
		return false, nil
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
