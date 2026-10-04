package store

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"time"
)

func CorrectionWorkerAccountForTest(ctx context.Context, s *Store, id string, action correction.Action) error {
	return s.correctionSystemTx(ctx, true, func(ctx context.Context, tx *sql.Tx, _ time.Time) error {
		a, e := correctionReadAccount(ctx, tx, id, true)
		if e != nil {
			return e
		}
		return correction.Authorize(a.User, action)
	})
}
