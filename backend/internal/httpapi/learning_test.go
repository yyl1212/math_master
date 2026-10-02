package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestLearningSharedBoundary(t *testing.T) {
	raw, e := os.ReadFile("../../../api/learning-boundary-cases.json")
	if e != nil {
		t.Fatal(e)
	}
	var cases []struct {
		Name      string
		Action    learning.Action
		Raw       string
		Accepted  bool
		ErrorCode *string
	}
	if json.Unmarshal(raw, &cases) != nil || len(cases) < 60 {
		t.Fatal("cases")
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			_, e := DecodeLearningInput([]byte(c.Raw), c.Action)
			if (e == nil) != c.Accepted {
				t.Fatal(e)
			}
			if e != nil {
				w := httptest.NewRecorder()
				r := httptest.NewRequest("POST", "/api/v1/learning/practice", nil)
				learningError(w, r, e)
				var got struct{ Error struct{ Code string } }
				json.Unmarshal(w.Body.Bytes(), &got)
				if c.ErrorCode == nil || got.Error.Code != *c.ErrorCode {
					t.Fatal(got, e)
				}
			}
		})
	}
}
func TestLearningRoutesAndQueries(t *testing.T) {
	id := "11111111-1111-4111-8111-111111111111"
	sha := strings.Repeat("a", 64)
	paths := []struct {
		path, method string
		action       learning.Action
	}{{"overview", "GET", learning.ReadOverviewAction}, {"knowledge", "GET", learning.ListKnowledgeAction}, {"knowledge/math-root", "GET", learning.ReadKnowledgeAction}, {"knowledge/math-root/start", "POST", learning.StartKnowledgeAction}, {"knowledge/math-root/complete", "POST", learning.CompleteKnowledgeAction}, {"paths/route/enroll", "POST", learning.EnrollPathAction}, {"paths", "GET", learning.ListPathsAction}, {"paths/" + id, "GET", learning.ReadPathAction}, {"paths/" + id + "/nodes", "GET", learning.ListPathNodesAction}, {"practice", "POST", learning.CreatePracticeAction}, {"practice/" + id, "GET", learning.ReadPracticeAction}, {"practice/" + id + "/answer", "POST", learning.AnswerPracticeAction}, {"practice/" + id + "/reveal", "POST", learning.RevealPracticeAction}, {"practice/" + id + "/abandon", "POST", learning.AbandonPracticeAction}, {"assessments", "POST", learning.CreateAssessmentAction}, {"assessments/" + id, "GET", learning.ReadAssessmentAction}, {"assessments/" + id + "/submit", "POST", learning.SubmitAssessmentAction}, {"assessments/" + id + "/abandon", "POST", learning.AbandonAssessmentAction}, {"assessments/" + id + "/result", "GET", learning.ReadAssessmentResultAction}, {"history", "GET", learning.ListHistoryAction}, {"assets/" + id + "/" + sha, "GET", learning.ReadAssetAction}}
	for _, p := range paths {
		r, e := routeLearning("/api/v1/learning/"+p.path, p.method)
		if e != nil || r.Action != p.action {
			t.Fatal(p, e)
		}
		if _, e = routeLearning("/api/v1/learning/"+p.path, "PUT"); e != errPrivateMethod {
			t.Fatal(p, e)
		}
	}
	for _, raw := range []string{"version=1&version=1", "version=01", "version=1.0", "version=1e0", "version=%31", "version=1&limit=1", "owner=x", ""} {
		if _, _, e := learningQuery(raw, learning.ReadKnowledgeAction); e == nil {
			t.Fatal(raw)
		}
	}
	for _, raw := range []string{"limit=101", "offset=100001", "limit=0", "limit=20&limit=20", "scope=mine", "limit=+1"} {
		if _, _, e := learningQuery(raw, learning.ListHistoryAction); e == nil {
			t.Fatal(raw)
		}
	}
	if q, _, e := learningQuery("limit=100&offset=100000", learning.ListHistoryAction); e != nil || q.Limit != 100 {
		t.Fatal(q, e)
	}
}
func TestLearningResponseBudget(t *testing.T) {
	for _, n := range []int{(4 << 20) - 1, 4 << 20, (4 << 20) + 1} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/api/v1/learning/overview", nil)
		learningResponse(w, r, 200, strings.Repeat("a", n-2))
		if (w.Code == 200) != (n <= 4<<20) {
			t.Fatal(n, w.Code)
		}
		if len(w.Body.Bytes()) > 4<<20 || w.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatal(n)
		}
	}
}

type learningHTTPRepo struct {
	learning.Repository
	user                       auth.User
	fault                      error
	preflights, rates, creates int
	deadline                   time.Duration
}

func (r *learningHTTPRepo) LearningPreflight(ctx context.Context, _ question.Access, _ learning.Action) (auth.User, error) {
	r.preflights++
	if d, ok := ctx.Deadline(); ok {
		r.deadline = time.Until(d)
	}
	return r.user, r.fault
}
func (r *learningHTTPRepo) ConsumeRates(context.Context, []auth.RateKey) error { r.rates++; return nil }
func (r *learningHTTPRepo) ReadLearningOverview(context.Context, question.Access) (learning.Overview, error) {
	return learning.Overview{AvailablePaths: []learning.PublishedPathSummary{}, Recent: []learning.HistoryEntry{}}, nil
}
func TestLearningHTTPPrivateBoundary(t *testing.T) {
	repo := &learningHTTPRepo{user: auth.User{ID: contentFixtureID, Roles: []auth.Role{auth.RoleLearner}}}
	slots := publication.NewService(nil)
	service, _ := learning.NewService(repo, slots.AcquireValidation)
	h := questionHTTPFixture(t, &httpQuestionRepo{}, nil, func(o *AuthOptions) { o.Learning = &LearningOptions{Learning: service, PublicOrigin: privateOrigin} })
	headers := contentHeaders()
	for _, tc := range []struct {
		path, method string
		mutate       func(map[string]string)
		status       int
		code         string
	}{{"/api/v1/learning/overview", "GET", func(h map[string]string) {}, 200, ""}, {"/api/v1/learning/overview", "POST", func(h map[string]string) {}, 405, "METHOD_NOT_ALLOWED"}, {"/api/v1/learning", "GET", func(h map[string]string) {}, 404, "NOT_FOUND"}, {"/api/v1/learning/overview?owner=x", "GET", func(h map[string]string) {}, 400, "INVALID_REQUEST"}, {"/api/v1/learning/practice", "POST", func(h map[string]string) { delete(h, "X-CSRF-Token") }, 403, "CSRF_FAILED"}, {"/api/v1/learning/practice", "POST", func(h map[string]string) { h["Origin"] = "https://wrong.example" }, 403, "CSRF_FAILED"}, {"/api/v1/learning/overview", "GET", func(h map[string]string) { h["Sec-Fetch-Site"] = "cross-site" }, 403, "CSRF_FAILED"}, {"/api/v1/learning/practice", "POST", func(h map[string]string) { delete(h, "Idempotency-Key") }, 400, "INVALID_REQUEST"}, {"/api/v1/learning/practice", "POST", func(h map[string]string) { h["Content-Type"] = "text/plain" }, 400, "INVALID_REQUEST"}, {"/api/v1/learning/knowledge/%6dath-root?version=1", "GET", func(h map[string]string) {}, 400, "INVALID_REQUEST"}, {"/api/v1/learning/assets/bad/bad", "GET", func(h map[string]string) {}, 400, "INVALID_REQUEST"}} {
		copy := map[string]string{}
		for k, v := range headers {
			copy[k] = v
		}
		tc.mutate(copy)
		body := ""
		if tc.method == "POST" {
			body = "{}"
		}
		w := privateRequest(h, tc.method, tc.path, body, copy)
		if w.Code != tc.status {
			t.Fatal(tc.path, w.Code, w.Body.String())
		}
		if tc.code != "" && !strings.Contains(w.Body.String(), tc.code) {
			t.Fatal(tc.path, w.Body.String())
		}
		if w.Header().Get("Cache-Control") != "private, no-store" || len(w.Result().Cookies()) != 0 {
			t.Fatal("private headers")
		}
	}
	if repo.deadline <= 7*time.Second || repo.deadline > 8*time.Second {
		t.Fatal(repo.deadline)
	}
	before := repo.rates
	repo.user.MustChangePassword = true
	w := privateRequest(h, "GET", "/api/v1/learning/overview", "", headers)
	if w.Code != 403 || repo.rates != before {
		t.Fatal(w.Code)
	}
	repo.user.MustChangePassword = false
	one, _ := slots.AcquireValidation(context.Background())
	two, _ := slots.AcquireValidation(context.Background())
	w = privateRequest(h, "POST", "/api/v1/learning/practice", "{}", headers)
	if w.Code != 503 || repo.rates != before+1 {
		t.Fatal(w.Code, repo.rates)
	}
	one()
	two()
	repo.fault = learning.ErrNotConfigured
	w = privateRequest(h, "GET", "/api/v1/learning/overview", "", headers)
	if w.Code != 503 || !strings.Contains(w.Body.String(), "LEARNING_NOT_CONFIGURED") {
		t.Fatal(w.Code)
	}
}
func TestLearningHTTPDeadlineIncludesReadingBody(t *testing.T) {
	repo := &learningHTTPRepo{user: auth.User{ID: contentFixtureID, Roles: []auth.Role{auth.RoleLearner}}}
	service, _ := learning.NewService(repo, publication.NewService(nil).AcquireValidation)
	h := questionHTTPFixture(t, &httpQuestionRepo{}, nil, func(o *AuthOptions) { o.Learning = &LearningOptions{Learning: service, PublicOrigin: privateOrigin} })
	server := httptest.NewServer(h)
	defer server.Close()
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	req, _ := http.NewRequest("POST", server.URL+"/api/v1/learning/practice", reader)
	for k, v := range contentHeaders() {
		req.Header.Set(k, v)
	}
	go func() { _, _ = writer.Write([]byte("{")) }()
	begin := time.Now()
	client := &http.Client{Timeout: 10 * time.Second}
	res, e := client.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	defer res.Body.Close()
	raw, e := io.ReadAll(res.Body)
	if e != nil || res.StatusCode != 503 || time.Since(begin) < 7*time.Second || time.Since(begin) > 10*time.Second || !strings.Contains(string(raw), "SERVICE_UNAVAILABLE") {
		t.Fatal(res.StatusCode, e, time.Since(begin))
	}
	if repo.preflights != 1 || repo.rates != 1 {
		t.Fatal(repo.preflights, repo.rates)
	}
}
func TestLearningHTTPErrorExtrasAreClosed(t *testing.T) {
	for _, fault := range []error{&learning.NotReadyError{}, learning.ErrStateConflict, &question.NumericFormatError{Code: "ZERO_DENOMINATOR"}, errors.New("private SQL correctNumeric sourceMap should never appear")} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/api/v1/learning/practice", nil)
		learningError(w, r, fault)
		if strings.Contains(w.Body.String(), "private SQL") || strings.Contains(w.Body.String(), "sourceMap") || strings.Contains(w.Body.String(), "correctNumeric") {
			t.Fatal(w.Body.String())
		}
	}
}
