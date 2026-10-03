package correction

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"testing"
)

func TestCorrectionPolicy(t *testing.T) {
	learner := auth.User{ID: testUUID, Roles: []auth.Role{auth.RoleLearner}}
	if !errors.Is(Authorize(learner, CreatePlanAction), auth.ErrForbidden) {
		t.Fatal("learner can propose correction")
	}
	editor := learner
	editor.Roles = []auth.Role{auth.RoleEditor}
	if Authorize(editor, CreatePlanAction) != nil {
		t.Fatal("editor cannot propose")
	}
	if Authorize(editor, CreateCaseAction) == nil {
		t.Fatal("editor can register grading problem")
	}
	reviewer := learner
	reviewer.Roles = []auth.Role{auth.RoleReviewer}
	if CanReview(reviewer, testUUID, nil) {
		t.Fatal("creator may self-review")
	}
	if CanReview(reviewer, "22222222-2222-4222-8222-222222222222", []string{testUUID}) {
		t.Fatal("source author may self-review")
	}
	if !CanReview(reviewer, "22222222-2222-4222-8222-222222222222", nil) {
		t.Fatal("independent review denied")
	}
	for _, a := range []Action{ListOwnAction, ReadOwnAction, ReadOwnDetailAction} {
		if Authorize(learner, a) != nil {
			t.Fatal("learner cannot read own result", a)
		}
	}
	learner.MustChangePassword = true
	if !errors.Is(Authorize(learner, ReadOwnAction), auth.ErrPasswordChangeRequired) {
		t.Fatal("must-change account accepted")
	}
	if !errors.Is(Authorize(auth.User{}, ReadOwnAction), auth.ErrAuthenticationRequired) {
		t.Fatal("anonymous accepted")
	}
}
