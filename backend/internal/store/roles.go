package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"slices"
	"time"
)

const adminLockID int64 = 1296127049

var _ auth.AdminRepository = (*Store)(nil)

func lockAdmin(ctx context.Context, tx *sql.Tx) error {
	_, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1)", adminLockID)
	return err
}
func (s *Store) InitializeAdmin(ctx context.Context, id, username, phc, requestID string) (auth.User, error) {
	user := auth.User{ID: id, Username: username, Roles: []auth.Role{auth.RoleLearner, auth.RoleAdmin}}
	err := s.authTx(ctx, nil, func(ctx context.Context, tx *sql.Tx) error {
		if err := lockAdmin(ctx, tx); err != nil {
			return err
		}
		var exists bool
		if err := tx.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM auth_bootstrap)").Scan(&exists); err != nil {
			return err
		}
		if exists {
			return auth.ErrAlreadyInitialized
		}
		now, err := dbClock(ctx, tx)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO auth_users(id,username,password_phc,created_at) VALUES($1,$2,$3,$4)", id, username, phc, now); err != nil {
			return err
		}
		for _, role := range user.Roles {
			if _, err = tx.ExecContext(ctx, "INSERT INTO auth_user_roles(user_id,role) VALUES($1,$2)", id, role); err != nil {
				return err
			}
		}
		if _, err = tx.ExecContext(ctx, "INSERT INTO auth_bootstrap(user_id,created_at) VALUES($1,$2)", id, now); err != nil {
			return err
		}
		return insertAuthAudit(ctx, tx, "admin_initialized", id, id, nil, user.Roles, "", "", nil, requestID, now)
	})
	if err != nil {
		return auth.User{}, err
	}
	return user, nil
}
func requireAdmin(account accountRow, row sessionRow, now time.Time, recent bool) error {
	if !validSession(row, account.Version, now) {
		return auth.ErrAuthenticationRequired
	}
	if account.User.MustChangePassword {
		return auth.ErrPasswordChangeRequired
	}
	if !auth.HasRole(account.User, auth.RoleAdmin) {
		return auth.ErrForbidden
	}
	if recent && (!row.Reauth.Valid || now.Before(row.Reauth.Time) || !now.Before(row.Reauth.Time.Add(5*time.Minute))) {
		return auth.ErrReauthRequired
	}
	return nil
}
func lockAdminProof(ctx context.Context, tx *sql.Tx, proof auth.SessionProof, targetID string) (accountRow, accountRow, time.Time, error) {
	var actorID string
	err := tx.QueryRowContext(ctx, "SELECT user_id::text FROM auth_sessions WHERE token_hash=$1", proof.TokenHash[:]).Scan(&actorID)
	if errors.Is(err, sql.ErrNoRows) {
		return accountRow{}, accountRow{}, time.Time{}, auth.ErrAuthenticationRequired
	}
	if err != nil {
		return accountRow{}, accountRow{}, time.Time{}, err
	}
	rows, err := tx.QueryContext(ctx, "SELECT id::text FROM auth_users WHERE id=ANY($1::uuid[]) ORDER BY id FOR UPDATE", []string{actorID, targetID})
	if err != nil {
		return accountRow{}, accountRow{}, time.Time{}, err
	}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return accountRow{}, accountRow{}, time.Time{}, err
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return accountRow{}, accountRow{}, time.Time{}, err
	}
	actor, err := readAccount(ctx, tx, actorID, false)
	if err != nil {
		return actor, accountRow{}, time.Time{}, err
	}
	row, err := readSessionRow(ctx, tx, proof.TokenHash, true)
	if err != nil {
		return actor, accountRow{}, time.Time{}, err
	}
	now, err := dbClock(ctx, tx)
	if err != nil {
		return actor, accountRow{}, now, err
	}
	if err = requireAdmin(actor, row, now, true); err != nil {
		return actor, accountRow{}, now, err
	}
	if !auth.EqualSecret(row.CSRF, proof.CSRF) {
		return actor, accountRow{}, now, auth.ErrCSRF
	}
	target, err := readAccount(ctx, tx, targetID, false)
	return actor, target, now, err
}
func (s *Store) ReplaceAccountRoles(ctx context.Context, proof auth.SessionProof, targetID string, input auth.RolesInput, requestID string) (auth.RoleMutation, error) {
	roles, err := auth.NormalizeRoles(input.Roles)
	if err != nil {
		return auth.RoleMutation{}, err
	}
	if err = auth.ValidateNote(input.Reason); err != nil {
		return auth.RoleMutation{}, err
	}
	var result auth.RoleMutation
	err = s.authTx(ctx, nil, func(ctx context.Context, tx *sql.Tx) error {
		if err := lockAdmin(ctx, tx); err != nil {
			return err
		}
		actor, target, now, err := lockAdminProof(ctx, tx, proof, targetID)
		if err != nil {
			return err
		}
		changed := !slices.Equal(target.User.Roles, roles)
		if auth.HasRole(target.User, auth.RoleAdmin) && !slices.Contains(roles, auth.RoleAdmin) {
			var admins int
			if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM auth_user_roles WHERE role='admin'").Scan(&admins); err != nil {
				return err
			}
			if admins <= 1 {
				return auth.ErrLastAdminRequired
			}
		}
		if changed {
			if _, err = tx.ExecContext(ctx, "DELETE FROM auth_user_roles WHERE user_id=$1", targetID); err != nil {
				return err
			}
			for _, role := range roles {
				if _, err = tx.ExecContext(ctx, "INSERT INTO auth_user_roles(user_id,role) VALUES($1,$2)", targetID, role); err != nil {
					return err
				}
			}
			if _, err = tx.ExecContext(ctx, "UPDATE auth_users SET credential_version=credential_version+1 WHERE id=$1", targetID); err != nil {
				return err
			}
			if _, err = tx.ExecContext(ctx, "UPDATE auth_sessions SET revoked_at=$2 WHERE user_id=$1 AND revoked_at IS NULL", targetID, now); err != nil {
				return err
			}
		}
		if err = insertAuthAudit(ctx, tx, "roles_replaced", actor.User.ID, targetID, target.User.Roles, roles, input.Reason, "", nil, requestID, now); err != nil {
			return err
		}
		result = auth.RoleMutation{ActorID: actor.User.ID, Changed: changed}
		return nil
	})
	return result, err
}
func (s *Store) ResetAccountPassword(ctx context.Context, proof auth.SessionProof, targetID, phc, reason, note, requestID string) (auth.RoleMutation, error) {
	if err := auth.ValidateNote(reason); err != nil {
		return auth.RoleMutation{}, err
	}
	if err := auth.ValidateNote(note); err != nil {
		return auth.RoleMutation{}, err
	}
	var result auth.RoleMutation
	err := s.authTx(ctx, nil, func(ctx context.Context, tx *sql.Tx) error {
		if err := lockAdmin(ctx, tx); err != nil {
			return err
		}
		actor, target, now, err := lockAdminProof(ctx, tx, proof, targetID)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "UPDATE auth_users SET password_phc=$2,credential_version=credential_version+1,must_change_password=true WHERE id=$1", targetID, phc); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, "UPDATE auth_sessions SET revoked_at=$2 WHERE user_id=$1 AND revoked_at IS NULL", targetID, now); err != nil {
			return err
		}
		if err = insertAuthAudit(ctx, tx, "password_reset", actor.User.ID, targetID, target.User.Roles, target.User.Roles, reason, note, nil, requestID, now); err != nil {
			return err
		}
		result = auth.RoleMutation{ActorID: actor.User.ID, Changed: true}
		return nil
	})
	return result, err
}
func (s *Store) ListAccountUsers(ctx context.Context, hash auth.Digest, query auth.UserQuery) (auth.UserPage, error) {
	query, err := auth.NormalizeUserQuery(query)
	if err != nil {
		return auth.UserPage{}, err
	}
	page := auth.UserPage{Items: make([]auth.User, 0), Limit: query.Limit, Offset: query.Offset}
	err = s.authTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: true}, func(ctx context.Context, tx *sql.Tx) error {
		row, err := readSessionRow(ctx, tx, hash, false)
		if err != nil {
			return err
		}
		actor, err := readAccount(ctx, tx, row.UserID, false)
		if err != nil {
			return err
		}
		now, err := dbClock(ctx, tx)
		if err != nil {
			return err
		}
		if err = requireAdmin(actor, row, now, false); err != nil {
			return err
		}
		if err = tx.QueryRowContext(ctx, "SELECT count(*) FROM auth_users WHERE strpos(lower(username),lower($1))>0", query.Q).Scan(&page.Total); err != nil {
			return err
		}
		rows, err := tx.QueryContext(ctx, `SELECT u.id::text,u.username,u.must_change_password,
   COALESCE((SELECT jsonb_agg(r.role ORDER BY CASE r.role WHEN 'learner' THEN 1 WHEN 'editor' THEN 2 WHEN 'reviewer' THEN 3 WHEN 'admin' THEN 4 END) FROM auth_user_roles r WHERE r.user_id=u.id),'[]'::jsonb)
   FROM auth_users u WHERE strpos(lower(username),lower($1))>0 ORDER BY username,id LIMIT $2 OFFSET $3`, query.Q, query.Limit, query.Offset)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var user auth.User
			var roles []byte
			if err = rows.Scan(&user.ID, &user.Username, &user.MustChangePassword, &roles); err != nil {
				return err
			}
			if json.Unmarshal(roles, &user.Roles) != nil {
				return auth.ErrUnavailable
			}
			page.Items = append(page.Items, user)
		}
		return rows.Err()
	})
	if err != nil {
		return auth.UserPage{}, err
	}
	return page, nil
}
