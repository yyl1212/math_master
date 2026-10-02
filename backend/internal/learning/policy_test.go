package learning

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"testing"
	"time"
)

func TestLearningPolicyAllActionsAndSharedRates(t *testing.T) {
	actions := []Action{"readOverview", "listKnowledge", "readKnowledge", "startKnowledge", "completeKnowledge", "enrollPath", "listPaths", "readPath", "listPathNodes", "createPractice", "readPractice", "answerPractice", "revealPractice", "abandonPractice", "createAssessment", "readAssessment", "submitAssessment", "abandonAssessment", "readAssessmentResult", "listHistory", "readAsset"}
	reads := map[Action]bool{"readOverview": true, "listKnowledge": true, "readKnowledge": true, "listPaths": true, "readPath": true, "listPathNodes": true, "readPractice": true, "readAssessment": true, "readAssessmentResult": true, "listHistory": true, "readAsset": true}
	id := "11111111-1111-4111-8111-111111111111"
	for _, a := range actions {
		t.Run(string(a), func(t *testing.T) {
			u := auth.User{ID: id, Roles: []auth.Role{auth.RoleLearner}}
			if e := Authorize(u, a); e != nil {
				t.Fatal(e)
			}
			u.Roles = []auth.Role{auth.RoleAdmin}
			if !errors.Is(Authorize(u, a), auth.ErrForbidden) {
				t.Fatal("admin implied learner")
			}
			u.Roles = []auth.Role{auth.RoleLearner}
			u.MustChangePassword = true
			if !errors.Is(Authorize(u, a), auth.ErrPasswordChangeRequired) {
				t.Fatal("password change ignored")
			}
			heavy := a == "createPractice" || a == "createAssessment"
			if IsRead(a) != reads[a] || IsHeavy(a) != heavy || IsIdempotent(a) == reads[a] {
				t.Fatal("classification", a)
			}
			rates, e := Rates(id, a)
			if e != nil || len(rates) != 2 {
				t.Fatal(rates, e)
			}
			scope, userScope, g, n := "content_write", "content_write_user", 120, 30
			if heavy {
				scope, userScope, g, n = "content_heavy", "content_heavy_user", 40, 10
			} else if reads[a] {
				scope, userScope, g, n = "auth_read", "content_read_user", 600, 120
			}
			if rates[0].Scope != scope || rates[0].Limit != g || rates[1].Scope != userScope || rates[1].Limit != n || rates[1].Key != id || rates[0].Window != time.Minute {
				t.Fatal("second quota", rates)
			}
		})
	}
	if !errors.Is(Authorize(auth.User{}, "readOverview"), auth.ErrAuthenticationRequired) {
		t.Fatal("anonymous allowed")
	}
	if _, e := Rates(id, "bogus"); e == nil {
		t.Fatal("unknown action")
	}
}
