package e2etest

import (
	"context"
	"database/sql"
	"errors"
)

func knowledgePublishChange(ctx context.Context, db *sql.DB, operation string) error {
	var q string
	switch operation {
	case "stale":
		q = `UPDATE managed_knowledge SET edit_token=gen_random_uuid() WHERE NOT published`
	case "revoke-admin":
		q = `DELETE FROM auth_user_roles WHERE role='admin' AND user_id IN(SELECT id FROM auth_users WHERE username='auth_admin')`
	default:
		return errors.New("invalid test operation")
	}
	_, e := db.ExecContext(ctx, q)
	return e
}
