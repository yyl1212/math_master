package store

import (
	"context"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"time"
)

func roleStrings(roles []auth.Role) []string {
	out := make([]string, 0, len(roles))
	for _, r := range roles {
		out = append(out, string(r))
	}
	return out
}
func nullableID(value string) any {
	if value == "" {
		return nil
	}
	return value
}
func insertAuthAudit(ctx context.Context, tx *sql.Tx, action, actor, target string, before, after []auth.Role, reason, note string, usernameHash *auth.Digest, requestID string, now time.Time) error {
	var hash any
	if usernameHash != nil {
		hash = usernameHash[:]
	}
	_, err := tx.ExecContext(ctx, "INSERT INTO auth_audit_events(action,actor_id,target_id,before_roles,after_roles,reason,ownership_note,username_hash,request_id,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)", action, nullableID(actor), nullableID(target), roleStrings(before), roleStrings(after), reason, note, hash, requestID, now)
	return err
}
func (s *Store) RecordLoginFailure(ctx context.Context, hash auth.Digest, requestID string) error {
	return s.authTx(ctx, nil, func(ctx context.Context, tx *sql.Tx) error {
		now, err := dbClock(ctx, tx)
		if err != nil {
			return err
		}
		return insertAuthAudit(ctx, tx, "login_failed", "", "", nil, nil, "", "", &hash, requestID, now)
	})
}
