package question

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"testing"
	"time"
)

const actorID = "11111111-1111-4111-8111-111111111111"

func TestQuestionRoleAndRates(t *testing.T) {
	for _, c := range []struct {
		action  Action
		roles   []auth.Role
		allowed bool
	}{
		{CreateDraftAction, []auth.Role{auth.RoleEditor}, true}, {CreateDraftAction, []auth.Role{auth.RoleAdmin}, false},
		{DecideReviewAction, []auth.Role{auth.RoleReviewer}, true}, {DecideReviewAction, []auth.Role{auth.RoleAdmin}, false},
		{ReadCoverageAction, []auth.Role{auth.RoleReviewer}, true}, {ReadCoverageAction, []auth.Role{auth.RoleEditor}, false},
		{PrepareReleaseAction, []auth.Role{auth.RoleAdmin}, true}, {PrepareReleaseAction, []auth.Role{auth.RoleLearner}, false},
		{ReadSubmissionAction, []auth.Role{auth.RoleLearner}, false},
	} {
		if e := Authorize(auth.User{ID: actorID, Roles: c.roles}, c.action); (e == nil) != c.allowed {
			t.Fatalf("role %v action %s: %v", c.roles, c.action, e)
		}
	}
	if !errors.Is(Authorize(auth.User{ID: actorID, Roles: []auth.Role{auth.RoleEditor}, MustChangePassword: true}, CreateDraftAction), auth.ErrPasswordChangeRequired) {
		t.Fatal("password change bypass")
	}
	if !errors.Is(Authorize(auth.User{}, ReadCoverageAction), auth.ErrAuthenticationRequired) {
		t.Fatal("anonymous")
	}
	if !IsRead(ReadCoverageAction) || !IsHeavy(ReadCoverageAction) || IsIdempotent(ReadCoverageAction) || IsIdempotent(ValidateDraftAction) || IsIdempotent(PreviewWithdrawalAction) {
		t.Fatal("action classification")
	}
	for _, c := range []struct {
		action       Action
		scope, user  string
		total, limit int
	}{{SaveDraftAction, "content_write", "content_write_user", 120, 30}, {ReadDraftAction, "auth_read", "content_read_user", 600, 120}, {ValidateDraftAction, "content_heavy", "content_heavy_user", 40, 10}, {ReadCoverageAction, "content_heavy", "content_heavy_user", 40, 10}} {
		r, e := Rates(actorID, c.action)
		if e != nil || len(r) != 2 || r[0].Scope != c.scope || r[1].Scope != c.user || r[0].Limit != c.total || r[1].Limit != c.limit || r[1].Key != actorID || r[0].Window != time.Minute {
			t.Fatalf("rates %s: %v %v", c.action, r, e)
		}
	}
	if _, e := Rates(actorID, Action("unknown")); e == nil || IsIdempotent(Action("unknown")) {
		t.Fatal("unknown action")
	}
}
