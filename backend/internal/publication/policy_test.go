package publication

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"testing"
)

func TestWorkflowRoleMatrix(t *testing.T) {
	id := "11111111-1111-4111-8111-111111111111"
	for _, tc := range []struct {
		roles  []auth.Role
		action Action
		want   error
	}{
		{[]auth.Role{auth.RoleLearner}, CreateDraftAction, auth.ErrForbidden},
		{[]auth.Role{auth.RoleLearner, auth.RoleAdmin}, CreateDraftAction, auth.ErrForbidden},
		{[]auth.Role{auth.RoleLearner, auth.RoleEditor}, ActivateReleaseAction, auth.ErrForbidden},
		{[]auth.Role{auth.RoleLearner, auth.RoleEditor}, CreateDraftAction, nil},
		{[]auth.Role{auth.RoleLearner, auth.RoleReviewer}, DecideReviewAction, nil},
		{[]auth.Role{auth.RoleLearner, auth.RoleAdmin}, ReadDraftAction, nil},
	} {
		err := Authorize(auth.User{ID: id, Roles: tc.roles}, tc.action)
		if !errors.Is(err, tc.want) {
			t.Fatalf("role policy mismatch for %s", tc.action)
		}
	}
	if err := Authorize(auth.User{ID: id, Roles: []auth.Role{auth.RoleEditor}, MustChangePassword: true}, CreateDraftAction); !errors.Is(err, auth.ErrPasswordChangeRequired) {
		t.Fatal("temporary password bypassed content policy")
	}
	if err := Authorize(auth.User{}, ReadDraftAction); !errors.Is(err, auth.ErrAuthenticationRequired) {
		t.Fatal("missing account accepted")
	}
	if err := Authorize(auth.User{ID: id, Roles: []auth.Role{auth.RoleAdmin}}, Action("unknown")); !errors.Is(err, auth.ErrInvalidInput) {
		t.Fatal("unknown action accepted")
	}
}
