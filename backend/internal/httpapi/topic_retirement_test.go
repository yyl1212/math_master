package httpapi

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/study"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTopicRetirementExactRouteMatrix(t *testing.T) {
	id := "11111111-1111-4111-8111-111111111111"
	retired := []RetirementRoute{
		{Method: "POST", Path: "/api/v1/learning/knowledge/fractions/start"}, {Method: "POST", Path: "/api/v1/learning/knowledge/fractions/complete"}, {Method: "POST", Path: "/api/v1/learning/paths/fractions/enroll"}, {Method: "POST", Path: "/api/v1/learning/practice"}, {Method: "POST", Path: "/api/v1/learning/assessments"},
		{Method: "POST", Path: "/api/v1/question-bank/drafts"}, {Method: "POST", Path: "/api/v1/question-bank/drafts/adopt"}, {Method: "PUT", Path: "/api/v1/question-bank/drafts/" + id}, {Method: "POST", Path: "/api/v1/question-bank/drafts/" + id + "/validate"}, {Method: "POST", Path: "/api/v1/question-bank/drafts/" + id + "/submit"}, {Method: "POST", Path: "/api/v1/question-bank/submissions/" + id + "/revision"}, {Method: "POST", Path: "/api/v1/question-bank/submissions/" + id + "/decision"}, {Method: "POST", Path: "/api/v1/question-bank/publications/prepare"}, {Method: "POST", Path: "/api/v1/question-bank/publications/" + id + "/activate"}, {Method: "POST", Path: "/api/v1/question-bank/withdrawals"}, {Method: "POST", Path: "/api/v1/question-bank/withdrawals/preview"},
		{Method: "POST", Path: "/api/v1/corrections/cases"}, {Method: "POST", Path: "/api/v1/corrections/cases/" + id + "/plans"}, {Method: "PUT", Path: "/api/v1/corrections/plans/" + id + "/versions/1"}, {Method: "POST", Path: "/api/v1/corrections/plans/" + id + "/versions/1/submit"}, {Method: "POST", Path: "/api/v1/corrections/plans/" + id + "/versions/1/decision"},
	}
	for _, verb := range []string{"answer", "reveal", "abandon"} {
		retired = append(retired, RetirementRoute{Method: "POST", Path: "/api/v1/learning/practice/" + id + "/" + verb})
	}
	for _, verb := range []string{"submit", "abandon"} {
		retired = append(retired, RetirementRoute{Method: "POST", Path: "/api/v1/learning/assessments/" + id + "/" + verb})
	}
	for _, r := range retired {
		if retirementPolicy(taxonomy.ModeTopics, r) != "retired" {
			t.Error("write accepted", r.Method, r.Path)
		}
		if retirementPolicy(taxonomy.ModeLegacy, r) != "allow" {
			t.Error("legacy changed", r.Path)
		}
	}
	for _, r := range []RetirementRoute{{Method: "GET", Path: "/api/v1/learning/history"}, {Method: "GET", Path: "/api/v1/learning/practice/" + id}, {Method: "GET", Path: "/api/v1/learning/assessments/" + id + "/result"}, {Method: "GET", Path: "/api/v1/question-bank/submissions/" + id}, {Method: "GET", Path: "/api/v1/corrections/evidence"}} {
		if retirementPolicy(taxonomy.ModeTopics, r) != "historical-read" {
			t.Error("history inaccessible", r.Path)
		}
	}
	for _, r := range []RetirementRoute{{Method: "POST", Path: "/api/v1/auth/login"}, {Method: "POST", Path: "/api/v2/admin/publications/prepare"}, {Method: "POST", Path: "/api/v1/feedback/tickets"}, {Method: "POST", Path: "/api/v1/corrections/jobs/" + id + "/retry"}, {Method: "DELETE", Path: "/api/v1/learning/practice/" + id}, {Method: "POST", Path: "/api/v1/learning/unknown"}} {
		if retirementPolicy(taxonomy.ModeTopics, r) != "allow" {
			t.Error("unrelated route intercepted", r.Path)
		}
	}
}
func TestTopicRetirementErrorContract(t *testing.T) {
	for _, write := range []func(*httptest.ResponseRecorder){func(w *httptest.ResponseRecorder) {
		learningError(w, httptest.NewRequest("POST", "/", nil), study.ErrModuleRetired)
	}, func(w *httptest.ResponseRecorder) {
		questionError(w, httptest.NewRequest("POST", "/", nil), study.ErrModuleRetired)
	}, func(w *httptest.ResponseRecorder) {
		correctionError(w, httptest.NewRequest("POST", "/", nil), study.ErrModuleRetired)
	}} {
		w := httptest.NewRecorder()
		write(w)
		if w.Code != 410 || w.Header().Get("Cache-Control") != "private, no-store" {
			t.Error("retirement not explicit", w.Code)
		}
	}
}

type retirementModeFixture struct {
	mode  taxonomy.ExperienceMode
	err   error
	calls int
}

func (m *retirementModeFixture) ReadExperienceMode(context.Context) (taxonomy.ExperienceMode, error) {
	m.calls++
	return m.mode, m.err
}
func TestTopicRetirementHTTPHeaderAndFailClosed(t *testing.T) {
	m := &retirementModeFixture{mode: taxonomy.ModeTopics}
	h := NewApplicationHandler(nil, nil, AuthOptions{ExperienceMode: m})
	r := httptest.NewRequest("POST", "/api/v1/learning/practice", strings.NewReader("{}"))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 410 || w.Header().Get("Cache-Control") != "private, no-store" || len(w.Header().Get("X-Request-ID")) != 32 || !strings.Contains(w.Body.String(), "MODULE_RETIRED") {
		t.Fatal("invalid retired transport", w.Code)
	}
	m.err = study.ErrNotConfigured
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 503 {
		t.Fatal("missing capability fell back", w.Code)
	}
	m.err = nil
	calls := m.calls
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("POST", "/api/v2/study/unknown", nil))
	if m.calls != calls {
		t.Fatal("unrelated v2 route read retirement mode")
	}
}
