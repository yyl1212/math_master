package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"time"
)

func (s *Store) ReadPreauth(ctx context.Context, hash auth.Digest) (auth.PreauthRecord, error) {
	ctx, c := context.WithTimeout(ctx, authDBTimeout)
	defer c()
	var raw []byte
	var record auth.PreauthRecord
	err := s.db.QueryRowContext(ctx, "SELECT csrf FROM auth_preauth WHERE token_hash=$1 AND consumed_at IS NULL AND expires_at>clock_timestamp()", hash[:]).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return record, auth.ErrCSRF
	}
	if err != nil || len(raw) != 32 {
		return record, auth.ErrUnavailable
	}
	copy(record.CSRF[:], raw)
	return record, nil
}
func (s *Store) CreatePreauth(ctx context.Context, input auth.NewPreauth) error {
	return s.authTx(ctx, nil, func(ctx context.Context, tx *sql.Tx) error {
		now, err := dbClock(ctx, tx)
		if err != nil {
			return err
		}
		_, err = tx.ExecContext(ctx, "INSERT INTO auth_preauth(token_hash,csrf,created_at,expires_at) VALUES($1,$2,$3,$4)", input.TokenHash[:], input.CSRF[:], now, now.Add(10*time.Minute))
		return err
	})
}

type sessionRow struct {
	UserID                  string
	Version                 int64
	CSRF                    auth.Secret
	Created, Seen, Absolute time.Time
	Revoked, Reauth         sql.NullTime
}

func readSessionRow(ctx context.Context, tx *sql.Tx, hash auth.Digest, lock bool) (sessionRow, error) {
	var row sessionRow
	var raw []byte
	q := "SELECT user_id::text,credential_version,csrf,created_at,last_seen_at,absolute_expires_at,revoked_at,reauthenticated_at FROM auth_sessions WHERE token_hash=$1"
	if lock {
		q += " FOR UPDATE"
	}
	err := tx.QueryRowContext(ctx, q, hash[:]).Scan(&row.UserID, &row.Version, &raw, &row.Created, &row.Seen, &row.Absolute, &row.Revoked, &row.Reauth)
	if errors.Is(err, sql.ErrNoRows) {
		return row, auth.ErrAuthenticationRequired
	}
	if err != nil {
		return row, err
	}
	if len(raw) != 32 {
		return row, auth.ErrUnavailable
	}
	copy(row.CSRF[:], raw)
	return row, nil
}
func validSession(row sessionRow, version int64, now time.Time) bool {
	return row.Version == version && !row.Revoked.Valid && now.Before(row.Absolute) && now.Before(row.Seen.Add(30*time.Minute))
}
func sessionRecord(row sessionRow, user auth.User) auth.SessionRecord {
	out := auth.SessionRecord{User: user, CSRF: row.CSRF, CredentialVersion: row.Version}
	if row.Reauth.Valid {
		out.ReauthenticatedAt = &row.Reauth.Time
	}
	return out
}
func (s *Store) ReadSession(ctx context.Context, hash auth.Digest, touch bool) (auth.SessionRecord, error) {
	if !touch {
		ctx, c := context.WithTimeout(ctx, authDBTimeout)
		defer c()
		var record auth.SessionRecord
		var raw, roles []byte
		var recent sql.NullTime
		err := s.db.QueryRowContext(ctx, `SELECT u.id::text,u.username,u.must_change_password,s.csrf,s.credential_version,s.reauthenticated_at,
   COALESCE((SELECT jsonb_agg(r.role ORDER BY CASE r.role WHEN 'learner' THEN 1 WHEN 'editor' THEN 2 WHEN 'reviewer' THEN 3 WHEN 'admin' THEN 4 END) FROM auth_user_roles r WHERE r.user_id=u.id),'[]'::jsonb)
   FROM auth_sessions s JOIN auth_users u ON u.id=s.user_id WHERE s.token_hash=$1 AND s.credential_version=u.credential_version AND s.revoked_at IS NULL AND s.absolute_expires_at>clock_timestamp() AND s.last_seen_at>clock_timestamp()-interval '30 minutes'`, hash[:]).Scan(&record.User.ID, &record.User.Username, &record.User.MustChangePassword, &raw, &record.CredentialVersion, &recent, &roles)
		if errors.Is(err, sql.ErrNoRows) {
			return record, auth.ErrAuthenticationRequired
		}
		if err != nil || len(raw) != 32 {
			return record, auth.ErrUnavailable
		}
		copy(record.CSRF[:], raw)
		if json.Unmarshal(roles, &record.User.Roles) != nil {
			return auth.SessionRecord{}, auth.ErrUnavailable
		}
		if recent.Valid {
			record.ReauthenticatedAt = &recent.Time
		}
		return record, nil
	}
	var record auth.SessionRecord
	err := s.authTx(ctx, nil, func(ctx context.Context, tx *sql.Tx) error {
		row, err := readSessionRow(ctx, tx, hash, true)
		if err != nil {
			return err
		}
		// This path never locks a user after the session; credential writes lock users first.
		account, err := readAccount(ctx, tx, row.UserID, false)
		if err != nil {
			return err
		}
		now, err := dbClock(ctx, tx)
		if err != nil {
			return err
		}
		if !validSession(row, account.Version, now) {
			return auth.ErrAuthenticationRequired
		}
		if !now.Before(row.Seen.Add(time.Minute)) {
			if _, err = tx.ExecContext(ctx, "UPDATE auth_sessions SET last_seen_at=$2 WHERE token_hash=$1", hash[:], now); err != nil {
				return err
			}
		}
		record = sessionRecord(row, account.User)
		return nil
	})
	return record, err
}
func (s *Store) LoginSession(ctx context.Context, proof auth.PreauthProof, userID string, version int64, next auth.NewSession, requestID string) (auth.User, error) {
	var user auth.User
	err := s.authTx(ctx, nil, func(ctx context.Context, tx *sql.Tx) error {
		account, err := readAccount(ctx, tx, userID, true)
		if err != nil {
			return err
		}
		if account.Version != version {
			return auth.ErrInvalidCredentials
		}
		if err = lockPreauth(ctx, tx, proof); err != nil {
			return err
		}
		now, err := dbClock(ctx, tx)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO auth_sessions(token_hash,user_id,credential_version,csrf,created_at,last_seen_at,absolute_expires_at) VALUES($1,$2,$3,$4,$5,$5,$6)", next.TokenHash[:], userID, version, next.CSRF[:], now, now.Add(7*24*time.Hour)); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE auth_sessions SET revoked_at=$2 WHERE token_hash IN (SELECT token_hash FROM auth_sessions WHERE user_id=$1 AND credential_version=$3 AND revoked_at IS NULL AND absolute_expires_at>$2 AND last_seen_at>$2-interval '30 minutes' ORDER BY created_at DESC,token_hash DESC OFFSET 5)`, userID, now, version); err != nil {
			return err
		}
		if err = insertAuthAudit(ctx, tx, "logged_in", userID, userID, account.User.Roles, account.User.Roles, "", "", nil, requestID, now); err != nil {
			return err
		}
		user = account.User
		return nil
	})
	return user, err
}
func lockedSessionProof(ctx context.Context, tx *sql.Tx, proof auth.SessionProof) (accountRow, sessionRow, time.Time, error) {
	var id string
	err := tx.QueryRowContext(ctx, "SELECT user_id::text FROM auth_sessions WHERE token_hash=$1", proof.TokenHash[:]).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return accountRow{}, sessionRow{}, time.Time{}, auth.ErrAuthenticationRequired
	}
	if err != nil {
		return accountRow{}, sessionRow{}, time.Time{}, err
	}
	account, err := readAccount(ctx, tx, id, true)
	if err != nil {
		return account, sessionRow{}, time.Time{}, err
	}
	row, err := readSessionRow(ctx, tx, proof.TokenHash, true)
	if err != nil {
		return account, row, time.Time{}, err
	}
	now, err := dbClock(ctx, tx)
	if err != nil {
		return account, row, now, err
	}
	if !validSession(row, account.Version, now) {
		return account, row, now, auth.ErrAuthenticationRequired
	}
	if !auth.EqualSecret(row.CSRF, proof.CSRF) {
		return account, row, now, auth.ErrCSRF
	}
	return account, row, now, nil
}
func (s *Store) LogoutSession(ctx context.Context, proof auth.SessionProof, all bool, requestID string) error {
	return s.authTx(ctx, nil, func(ctx context.Context, tx *sql.Tx) error {
		account, _, now, err := lockedSessionProof(ctx, tx, proof)
		if err != nil {
			return err
		}
		action := "logged_out"
		q := "UPDATE auth_sessions SET revoked_at=$2 WHERE token_hash=$1 AND revoked_at IS NULL"
		arg := any(proof.TokenHash[:])
		if all {
			action = "logged_out_all"
			q = "UPDATE auth_sessions SET revoked_at=$2 WHERE user_id=$1 AND revoked_at IS NULL"
			arg = account.User.ID
		}
		if _, err = tx.ExecContext(ctx, q, arg, now); err != nil {
			return err
		}
		return insertAuthAudit(ctx, tx, action, account.User.ID, account.User.ID, account.User.Roles, account.User.Roles, "", "", nil, requestID, now)
	})
}
func (s *Store) ChangePassword(ctx context.Context, proof auth.SessionProof, version int64, phc, requestID string) error {
	return s.authTx(ctx, nil, func(ctx context.Context, tx *sql.Tx) error {
		account, _, now, err := lockedSessionProof(ctx, tx, proof)
		if err != nil {
			return err
		}
		if account.Version != version {
			return auth.ErrInvalidCredentials
		}
		if _, err = tx.ExecContext(ctx, "UPDATE auth_users SET password_phc=$2,credential_version=credential_version+1,must_change_password=false WHERE id=$1", account.User.ID, phc); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "UPDATE auth_sessions SET revoked_at=$2 WHERE user_id=$1 AND revoked_at IS NULL", account.User.ID, now); err != nil {
			return err
		}
		return insertAuthAudit(ctx, tx, "password_changed", account.User.ID, account.User.ID, account.User.Roles, account.User.Roles, "", "", nil, requestID, now)
	})
}
func (s *Store) ReauthenticateSession(ctx context.Context, proof auth.SessionProof, version int64, requestID string) (time.Time, error) {
	var until time.Time
	err := s.authTx(ctx, nil, func(ctx context.Context, tx *sql.Tx) error {
		account, _, now, err := lockedSessionProof(ctx, tx, proof)
		if err != nil {
			return err
		}
		if account.User.MustChangePassword {
			return auth.ErrPasswordChangeRequired
		}
		if account.Version != version {
			return auth.ErrInvalidCredentials
		}
		if _, err = tx.ExecContext(ctx, "UPDATE auth_sessions SET reauthenticated_at=$2 WHERE token_hash=$1", proof.TokenHash[:], now); err != nil {
			return err
		}
		if err = insertAuthAudit(ctx, tx, "reauthenticated", account.User.ID, account.User.ID, account.User.Roles, account.User.Roles, "", "", nil, requestID, now); err != nil {
			return err
		}
		until = now.Add(5 * time.Minute)
		return nil
	})
	return until, err
}
