package feedback

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"testing"
)

func TestFeedbackPolicyAccountsAndReview(t *testing.T) {
	own := []Action{"readContext", "create", "reply", "listOwn", "readOwn", "discussOwn"}
	review := []Action{"transition", "listReview", "readReview", "discussReview"}
	for _, role := range []auth.Role{auth.RoleLearner, auth.RoleEditor, auth.RoleReviewer, auth.RoleAdmin} {
		u := auth.User{ID: testUUID, Roles: []auth.Role{role}}
		for _, a := range own {
			if e := Authorize(u, a); e != nil {
				t.Fatal(role, a, e)
			}
		}
		for _, a := range review {
			want := role == auth.RoleReviewer || role == auth.RoleAdmin
			if e := Authorize(u, a); (e == nil) != want {
				t.Fatal(role, a, e)
			}
		}
		u.MustChangePassword = true
		for _, a := range append(own, review...) {
			if !errors.Is(Authorize(u, a), auth.ErrPasswordChangeRequired) {
				t.Fatal("must change", role, a)
			}
		}
	}
	if !errors.Is(Authorize(auth.User{}, "create"), auth.ErrAuthenticationRequired) {
		t.Fatal("anonymous")
	}
	if !errors.Is(Authorize(auth.User{ID: testUUID}, "unknown"), auth.ErrInvalidInput) {
		t.Fatal("unknown action")
	}
	if Authorize(auth.User{ID: testUUID}, "create") != nil {
		t.Fatal("valid account without handling role")
	}
}
func TestFeedbackPolicyNoSelfHandling(t *testing.T) {
	for _, role := range []auth.Role{auth.RoleReviewer, auth.RoleAdmin} {
		u := auth.User{ID: testUUID, Roles: []auth.Role{role}}
		if CanHandle(u, testUUID) || !CanHandle(u, otherUUID) || CanHandle(u, "") {
			t.Fatal(role)
		}
		u.MustChangePassword = true
		if CanHandle(u, otherUUID) {
			t.Fatal("password proof")
		}
	}
	if CanHandle(auth.User{ID: testUUID, Roles: []auth.Role{auth.RoleEditor}}, otherUUID) {
		t.Fatal("editor")
	}
}
