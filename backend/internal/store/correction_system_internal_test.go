package store

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

func CorrectionSystemEntryForTest(ctx context.Context, s *Store, locks bool) (time.Time, error) {
	var out time.Time
	e := s.correctionSystemTx(ctx, locks, func(_ context.Context, _ *sql.Tx, now time.Time) error { out = now; return nil })
	return out, e
}
func CorrectionSystemLockIDsForTest() []int64 {
	return []int64{adminLockID, 1296127048, correctionRegistrationLock}
}

func CorrectionJobLockedTimeForTest(ctx context.Context, db *sql.DB, id string) (time.Time, error) {
	tx, e := db.BeginTx(ctx, nil)
	if e != nil {
		return time.Time{}, e
	}
	defer tx.Rollback()
	_, now, e := correctionReadJobTime(ctx, tx, id)
	return now, e
}

func CorrectionInstanceDependenciesForTest(ctx context.Context, db *sql.DB, i question.Instance) ([]correction.Dependency, error) {
	tx, e := db.BeginTx(ctx, nil)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback()
	return correctionInstanceDeps(ctx, tx, i)
}
