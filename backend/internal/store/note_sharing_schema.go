package store

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/notesharing"
)

var noteSharingTables = []string{"note_share_heads", "note_share_submissions", "note_share_events", "note_share_receipts"}

const noteSharingDigest = "8599fcfdbdc6636bfed2071391026efb"

func noteSharingInstalled(ctx context.Context, tx *sql.Tx) (bool, error) {
	var count int
	var applied bool
	if e := tx.QueryRowContext(ctx, `SELECT count(*) FROM unnest($1::text[]) name WHERE to_regclass('public.'||name) IS NOT NULL`, noteSharingTables).Scan(&count); e != nil {
		return false, notesharing.ErrNotReady
	}
	if e := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM public.goose_db_version WHERE version_id=14 AND is_applied)`).Scan(&applied); e != nil {
		return false, notesharing.ErrNotReady
	}
	if count == 0 && !applied {
		return false, nil
	}
	if count != len(noteSharingTables) || !applied {
		return false, notesharing.ErrNotReady
	}
	return true, nil
}
func noteSharingSchemaIntegrity(ctx context.Context, tx *sql.Tx) error {
	installed, e := noteSharingInstalled(ctx, tx)
	if e != nil || !installed {
		return notesharing.ErrNotReady
	}
	var previous, digest string
	if e = tx.QueryRowContext(ctx, "SELECT current_setting('search_path')").Scan(&previous); e != nil {
		return notesharing.ErrNotReady
	}
	if _, e = tx.ExecContext(ctx, "SELECT set_config('search_path','pg_catalog,public',true)"); e != nil {
		return notesharing.ErrNotReady
	}
	e = tx.QueryRowContext(ctx, managedSchemaDigestSQL, noteSharingTables).Scan(&digest)
	_, restore := tx.ExecContext(ctx, "SELECT set_config('search_path',$1,true)", previous)
	if e != nil || restore != nil || digest != noteSharingDigest {
		return notesharing.ErrNotReady
	}
	return nil
}
