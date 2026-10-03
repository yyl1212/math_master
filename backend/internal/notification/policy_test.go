package notification

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"testing"
)

func TestNotificationPolicy(t *testing.T) {
	u := auth.User{ID: "11111111-1111-4111-8111-111111111111", Roles: []auth.Role{auth.RoleLearner}}
	for _, a := range []Action{ListAction, CountAction, ReadAction, MarkReadAction} {
		if Authorize(u, a) != nil {
			t.Fatal("owner denied", a)
		}
	}
	if !errors.Is(Authorize(auth.User{}, ListAction), auth.ErrAuthenticationRequired) {
		t.Fatal("anonymous accepted")
	}
	u.MustChangePassword = true
	if !errors.Is(Authorize(u, MarkReadAction), auth.ErrPasswordChangeRequired) {
		t.Fatal("unsafe account accepted")
	}
	if Authorize(u, "send") == nil {
		t.Fatal("external sending authorized")
	}
}
