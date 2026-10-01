package store

import (
	"context"
	"database/sql"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"time"
)

var _ auth.AccountRepository = (*Store)(nil)

const authDBTimeout = 3 * time.Second

func authError(err error) error {
	if err == nil {
		return nil
	}
	for _, known := range []error{auth.ErrInvalidInput, auth.ErrInvalidCookie, auth.ErrInvalidCredentials, auth.ErrAuthenticationRequired, auth.ErrCSRF, auth.ErrForbidden, auth.ErrUsernameUnavailable, auth.ErrAlreadyAuthenticated, auth.ErrLastAdminRequired, auth.ErrPasswordChangeRequired, auth.ErrReauthRequired, auth.ErrNotFound, auth.ErrUnavailable, auth.ErrAlreadyInitialized} {
		if errors.Is(err, known) {
			return known
		}
	}
	var limited *auth.RateLimitError
	if errors.As(err, &limited) {
		return limited
	}
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" && pg.ConstraintName == "auth_users_username_key" {
		return auth.ErrUsernameUnavailable
	}
	return auth.ErrUnavailable
}
func (s *Store) authTx(ctx context.Context, options *sql.TxOptions, fn func(context.Context, *sql.Tx) error) error {
	ctx, cancel := context.WithTimeout(ctx, authDBTimeout)
	defer cancel()
	tx, err := s.db.BeginTx(ctx, options)
	if err != nil {
		return auth.ErrUnavailable
	}
	defer tx.Rollback()
	if err = fn(ctx, tx); err != nil {
		return authError(err)
	}
	return authError(tx.Commit())
}
func dbClock(ctx context.Context, tx *sql.Tx) (time.Time, error) {
	var now time.Time
	err := tx.QueryRowContext(ctx, "SELECT clock_timestamp()").Scan(&now)
	return now, err
}

type accountRow struct {
	User    auth.User
	PHC     string
	Version int64
}

func readAccount(ctx context.Context, tx *sql.Tx, id string, lock bool) (accountRow, error) {
	var a accountRow
	q := "SELECT id::text,username,password_phc,credential_version,must_change_password FROM auth_users WHERE id=$1"
	if lock {
		q += " FOR UPDATE"
	}
	if err := tx.QueryRowContext(ctx, q, id).Scan(&a.User.ID, &a.User.Username, &a.PHC, &a.Version, &a.User.MustChangePassword); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return a, auth.ErrNotFound
		}
		return a, err
	}
	rows, err := tx.QueryContext(ctx, "SELECT role FROM auth_user_roles WHERE user_id=$1 ORDER BY CASE role WHEN 'learner' THEN 1 WHEN 'editor' THEN 2 WHEN 'reviewer' THEN 3 WHEN 'admin' THEN 4 END", id)
	if err != nil {
		return a, err
	}
	defer rows.Close()
	a.User.Roles = make([]auth.Role, 0, 4)
	for rows.Next() {
		var role auth.Role
		if err = rows.Scan(&role); err != nil {
			return a, err
		}
		a.User.Roles = append(a.User.Roles, role)
	}
	return a, rows.Err()
}
func (s *Store) ReadCredential(ctx context.Context, username string) (auth.Credential, error) {
	ctx, cancel := context.WithTimeout(ctx, authDBTimeout)
	defer cancel()
	var c auth.Credential
	err := s.db.QueryRowContext(ctx, "SELECT id::text,password_phc,credential_version FROM auth_users WHERE username=$1", username).Scan(&c.UserID, &c.PHC, &c.Version)
	if errors.Is(err, sql.ErrNoRows) {
		return c, auth.ErrNotFound
	}
	return c, authError(err)
}
func lockPreauth(ctx context.Context, tx *sql.Tx, proof auth.PreauthProof) error {
	var csrf []byte
	var expires time.Time
	var consumed sql.NullTime
	err := tx.QueryRowContext(ctx, "SELECT csrf,expires_at,consumed_at FROM auth_preauth WHERE token_hash=$1 FOR UPDATE", proof.TokenHash[:]).Scan(&csrf, &expires, &consumed)
	if errors.Is(err, sql.ErrNoRows) {
		return auth.ErrCSRF
	}
	if err != nil {
		return err
	}
	// Refresh the database clock after obtaining the row lock.
	now, err := dbClock(ctx, tx)
	if err != nil {
		return err
	}
	var expected auth.Secret
	if len(csrf) != 32 {
		return auth.ErrUnavailable
	}
	copy(expected[:], csrf)
	if consumed.Valid || !now.Before(expires) || !auth.EqualSecret(proof.CSRF, expected) {
		return auth.ErrCSRF
	}
	_, err = tx.ExecContext(ctx, "UPDATE auth_preauth SET consumed_at=$2 WHERE token_hash=$1", proof.TokenHash[:], now)
	return err
}
func (s *Store) RegisterLearner(ctx context.Context, proof auth.PreauthProof, id, username, phc, requestID string) (auth.User, error) {
	user := auth.User{ID: id, Username: username, Roles: []auth.Role{auth.RoleLearner}}
	err := s.authTx(ctx, nil, func(ctx context.Context, tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, "INSERT INTO auth_users(id,username,password_phc) VALUES($1,$2,$3)", id, username, phc); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, "INSERT INTO auth_user_roles(user_id,role) VALUES($1,'learner')", id); err != nil {
			return err
		}
		if err := lockPreauth(ctx, tx, proof); err != nil {
			return err
		}
		now, err := dbClock(ctx, tx)
		if err != nil {
			return err
		}
		return insertAuthAudit(ctx, tx, "registered", id, id, nil, user.Roles, "", "", nil, requestID, now)
	})
	if err != nil {
		return auth.User{}, err
	}
	return user, nil
}
