package store

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/notification"
)

// Test-only bridge: external fixtures exercise the exact production append,
// including source guards and ON CONFLICT, without adding a product API.
func CorrectionAppendNotificationForTest(ctx context.Context, tx *sql.Tx, owner string, source notification.Source) error {
	return notificationAppend(ctx, tx, owner, source)
}
