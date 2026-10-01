package question

import (
	"github.com/yyl1212/math_master/backend/internal/auth"
	"time"
)

func Rates(actor string, a Action) ([]auth.RateKey, error) {
	if !ValidID(actor) {
		return nil, auth.ErrInvalidInput
	}
	if err := Authorize(auth.User{ID: actor, Roles: []auth.Role{auth.RoleEditor, auth.RoleReviewer, auth.RoleAdmin}}, a); err != nil {
		return nil, err
	}
	scope, userScope, global, user := "content_write", "content_write_user", 120, 30
	if IsHeavy(a) {
		scope, userScope, global, user = "content_heavy", "content_heavy_user", 40, 10
	} else if IsRead(a) {
		scope, userScope, global, user = "auth_read", "content_read_user", 600, 120
	}
	return []auth.RateKey{{Scope: scope, Limit: global, Window: time.Minute}, {Scope: userScope, Key: actor, Limit: user, Window: time.Minute}}, nil
}
