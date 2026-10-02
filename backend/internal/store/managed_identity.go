package store

import (
	"context"
	"database/sql"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"sort"
	"time"
)

// managedIdentity checks fresh DB time after all account/session waits. Policies stay in wrappers.
func managedIdentity(ctx context.Context, tx *sql.Tx, a publication.Access, write, lock bool, related []string) (auth.User, auth.SessionRecord, time.Time, error) {
	var id string
	err := tx.QueryRowContext(ctx, `SELECT user_id::text FROM auth_sessions WHERE token_hash=$1`, a.TokenHash[:]).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return auth.User{}, auth.SessionRecord{}, time.Time{}, auth.ErrAuthenticationRequired
	}
	if err != nil {
		return auth.User{}, auth.SessionRecord{}, time.Time{}, err
	}
	if lock {
		ids := append(append([]string{}, related...), id)
		sort.Strings(ids)
		for i, userID := range ids {
			if !publication.ValidID(userID) {
				return auth.User{}, auth.SessionRecord{}, time.Time{}, auth.ErrInvalidInput
			}
			if i > 0 && ids[i-1] == userID {
				continue
			}
			if _, err = readAccount(ctx, tx, userID, true); err != nil {
				return auth.User{}, auth.SessionRecord{}, time.Time{}, err
			}
		}
	}
	row, err := readSessionRow(ctx, tx, a.TokenHash, lock)
	if err != nil {
		return auth.User{}, auth.SessionRecord{}, time.Time{}, err
	}
	account, err := readAccount(ctx, tx, id, false)
	if err != nil {
		return auth.User{}, auth.SessionRecord{}, time.Time{}, err
	}
	now, err := dbClock(ctx, tx)
	if err != nil {
		return auth.User{}, auth.SessionRecord{}, now, err
	}
	if row.UserID != id || !validSession(row, account.Version, now) {
		return auth.User{}, auth.SessionRecord{}, now, auth.ErrAuthenticationRequired
	}
	if write && !auth.EqualSecret(a.CSRF, row.CSRF) {
		return auth.User{}, auth.SessionRecord{}, now, auth.ErrCSRF
	}
	return account.User, sessionRecord(row, account.User), now, nil
}
func managedReauth(row auth.SessionRecord, now time.Time) error {
	if row.ReauthenticatedAt == nil || now.Before(*row.ReauthenticatedAt) || !now.Before(row.ReauthenticatedAt.Add(5*time.Minute)) {
		return auth.ErrReauthRequired
	}
	return nil
}
