package httpapi

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/question"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type correctionHTTPRepo struct {
	correction.Repository
	calls      []string
	fault      error
	preflights int
	deadline   time.Duration
}

func (r *correctionHTTPRepo) CorrectionPreflight(ctx context.Context, _ question.Access, a correction.Action) (auth.User, error) {
	r.preflights++
	d, _ := ctx.Deadline()
	r.deadline = time.Until(d)
	return auth.User{ID: contentFixtureID, Roles: []auth.Role{auth.RoleAdmin}}, r.fault
}
func (r *correctionHTTPRepo) command(name string) (correction.Envelope[correction.Receipt], error) {
	r.calls = append(r.calls, name)
	status := 200
	if name == "createCase" || name == "createPlan" {
		status = 201
	}
	return correction.Envelope[correction.Receipt]{ActorID: contentFixtureID, Data: correction.Receipt{Status: status}}, nil
}
func (r *correctionHTTPRepo) CreateCorrectionCase(context.Context, question.Access, correction.CaseInput) (correction.Envelope[correction.Receipt], error) {
	return r.command("createCase")
}
func (r *correctionHTTPRepo) CreateCorrectionPlan(context.Context, question.Access, string, correction.PlanInput) (correction.Envelope[correction.Receipt], error) {
	return r.command("createPlan")
}
func (r *correctionHTTPRepo) UpdateCorrectionPlan(context.Context, question.Access, correction.PlanRef, correction.PlanInput) (correction.Envelope[correction.Receipt], error) {
	return r.command("updatePlan")
}
func (r *correctionHTTPRepo) SubmitCorrectionPlan(context.Context, question.Access, correction.PlanRef, correction.SubmitInput) (correction.Envelope[correction.Receipt], error) {
	return r.command("submitPlan")
}
func (r *correctionHTTPRepo) DecideCorrectionPlan(context.Context, question.Access, correction.PlanRef, correction.DecisionInput) (correction.Envelope[correction.Receipt], error) {
	return r.command("decidePlan")
}
func (r *correctionHTTPRepo) RetryCorrectionJob(context.Context, question.Access, string, correction.RetryInput) (correction.Envelope[correction.Receipt], error) {
	return r.command("retryJob")
}
func (r *correctionHTTPRepo) ListCorrectionCases(context.Context, question.Access, correction.Query) (correction.Envelope[correction.Page[correction.CaseMetadata]], error) {
	r.calls = append(r.calls, "listCases")
	return correction.Envelope[correction.Page[correction.CaseMetadata]]{}, nil
}
func (r *correctionHTTPRepo) ReadCorrectionCase(context.Context, question.Access, string) (correction.Envelope[correction.CaseMetadata], error) {
	r.calls = append(r.calls, "readCase")
	return correction.Envelope[correction.CaseMetadata]{}, nil
}
func (r *correctionHTTPRepo) ListCorrectionPlans(context.Context, question.Access, string, correction.Query) (correction.Envelope[correction.Page[correction.PlanMetadata]], error) {
	r.calls = append(r.calls, "listPlans")
	return correction.Envelope[correction.Page[correction.PlanMetadata]]{}, nil
}
func (r *correctionHTTPRepo) ReadCorrectionPlan(_ context.Context, _ question.Access, _ correction.PlanRef, detail bool) (correction.Envelope[correction.PlanDetail], error) {
	name := "readPlan"
	if detail {
		name = "readPlanDetail"
	}
	r.calls = append(r.calls, name)
	return correction.Envelope[correction.PlanDetail]{}, nil
}
func (r *correctionHTTPRepo) ListCorrectionJobs(context.Context, question.Access, string, correction.Query) (correction.Envelope[correction.Page[correction.JobMetadata]], error) {
	r.calls = append(r.calls, "listJobs")
	return correction.Envelope[correction.Page[correction.JobMetadata]]{}, nil
}
func (r *correctionHTTPRepo) ListOwnCorrections(context.Context, question.Access, correction.EvidenceRef, correction.Query) (correction.Envelope[correction.Page[correction.ResultMetadata]], error) {
	r.calls = append(r.calls, "listOwn")
	return correction.Envelope[correction.Page[correction.ResultMetadata]]{}, nil
}
func (r *correctionHTTPRepo) ReadOwnCorrection(_ context.Context, _ question.Access, _ string, detail bool) (correction.Envelope[correction.ResultDetail], error) {
	name := "readOwn"
	if detail {
		name = "readOwnDetail"
	}
	r.calls = append(r.calls, name)
	return correction.Envelope[correction.ResultDetail]{}, nil
}
func correctionHTTPFixture(t *testing.T, repo *correctionHTTPRepo) http.Handler {
	t.Helper()
	svc, e := correction.NewService(repo)
	if e != nil {
		t.Fatal(e)
	}
	return questionHTTPFixture(t, &httpQuestionRepo{}, nil, func(o *AuthOptions) { o.Correction = &CorrectionOptions{Service: svc, PublicOrigin: privateOrigin} })
}
func TestCorrectionHTTPRoutesAndDispatch(t *testing.T) {
	id := contentFixtureID
	repo := &correctionHTTPRepo{}
	h := correctionHTTPFixture(t, repo)
	plan := `{"expectedSequence":null,"parent":null,"algorithmVersion":1,"mappings":[],"reason":"Original correction."}`
	paths := []struct {
		Method, Path, Body, Call string
		Status                   int
	}{
		{"GET", "cases", "", "listCases", 200}, {"POST", "cases", `{"kind":"grading_rule","withdrawal":null,"rule":{"ruleVersion":1,"kind":"all","knowledge":null}}`, "createCase", 201}, {"GET", "cases/" + id, "", "readCase", 200}, {"GET", "cases/" + id + "/plans", "", "listPlans", 200}, {"POST", "cases/" + id + "/plans", plan, "createPlan", 201}, {"GET", "plans/" + id + "/versions/1", "", "readPlan", 200}, {"PUT", "plans/" + id + "/versions/1", strings.Replace(plan, "null", "1", 1), "updatePlan", 200}, {"GET", "plans/" + id + "/versions/1/detail", "", "readPlanDetail", 200}, {"POST", "plans/" + id + "/versions/1/submit", `{"expectedSequence":1}`, "submitPlan", 200}, {"POST", "plans/" + id + "/versions/1/decision", `{"expectedSequence":1,"decision":"approve","reason":"Independently verified."}`, "decidePlan", 200}, {"GET", "cases/" + id + "/jobs", "", "listJobs", 200}, {"POST", "jobs/" + id + "/retry", `{"expectedSequence":1}`, "retryJob", 200}, {"GET", "evidence?kind=assessment&id=" + id, "", "listOwn", 200}, {"GET", "results/" + id, "", "readOwn", 200}, {"GET", "results/" + id + "/detail", "", "readOwnDetail", 200}}
	for _, c := range paths {
		n := len(repo.calls)
		w := privateRequest(h, c.Method, "/api/v1/corrections/"+c.Path, c.Body, contentHeaders())
		if w.Code != c.Status || len(repo.calls) != n+1 || repo.calls[n] != c.Call || w.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatal(c.Call, w.Code, w.Body.String())
		}
	}
	if repo.deadline > 8*time.Second || repo.deadline < 7*time.Second {
		t.Fatal("whole request deadline", repo.deadline)
	}
	for _, p := range []string{"plans/" + id + "/versions/01", "plans/" + id + "/versions/0", "plans/" + id + "/versions/2147483648", "cases/bad", "cases?limit=01", "cases?limit=51", "cases?limit=1&limit=2", "cases?cursor=bad", "cases?unknown=1", "cases?", "cases/?limit=1", "evidence?kind=assessment", "evidence?kind=foreign&id=" + id, "results/" + id + "?detail=true"} {
		w := privateRequest(h, "GET", "/api/v1/corrections/"+p, "", contentHeaders())
		if w.Code != 400 && w.Code != 404 {
			t.Fatal("strict route/query", p, w.Code)
		}
	}
	w := privateRequest(h, "DELETE", "/api/v1/corrections/cases", "", contentHeaders())
	if w.Code != 405 || w.Header().Get("Allow") != "GET, POST" {
		t.Fatal("Allow", w.Code, w.Header())
	}
}
func TestCorrectionHTTPPrivateBoundaries(t *testing.T) {
	repo := &correctionHTTPRepo{}
	h := correctionHTTPFixture(t, repo)
	body := `{"kind":"grading_rule","withdrawal":null,"rule":{"ruleVersion":1,"kind":"all","knowledge":null}}`
	for _, c := range []struct {
		Name   string
		Mutate func(map[string]string)
		Status int
	}{{"origin", func(h map[string]string) { h["Origin"] = "https://wrong.example" }, 403}, {"csrf", func(h map[string]string) { delete(h, "X-CSRF-Token") }, 403}, {"key", func(h map[string]string) { delete(h, "Idempotency-Key") }, 400}, {"cookie", func(h map[string]string) { delete(h, "Cookie") }, 401}, {"type", func(h map[string]string) { h["Content-Type"] = "text/plain" }, 400}} {
		headers := contentHeaders()
		c.Mutate(headers)
		w := privateRequest(h, "POST", "/api/v1/corrections/cases", body, headers)
		if w.Code != c.Status {
			t.Fatal(c.Name, w.Code)
		}
	}
	for _, name := range []string{"Origin", "X-CSRF-Token", "Idempotency-Key", "Content-Type"} {
		req := httptest.NewRequest("POST", "/api/v1/corrections/cases", strings.NewReader(body))
		for k, v := range contentHeaders() {
			req.Header.Set(k, v)
		}
		req.Header.Add(name, req.Header.Get(name))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != 400 && w.Code != 403 {
			t.Fatal("duplicate header", name, w.Code)
		}
	}
	if len(repo.calls) != 0 {
		t.Fatal("invalid command dispatched")
	}
	repo.fault = auth.ErrForbidden
	w := privateRequest(h, "POST", "/api/v1/corrections/cases", "invalid-private-input", contentHeaders())
	if w.Code != 403 || strings.Contains(w.Body.String(), "invalid-private-input") {
		t.Fatal("authorization before body", w.Code)
	}
}
func TestCorrectionHTTPErrorAndResponseBudget(t *testing.T) {
	for _, n := range []int{2097151, 2097152, 2097153} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/api/v1/corrections/cases", nil)
		correctionResponse(w, r, 200, strings.Repeat("a", n-2))
		if (w.Code == 200) != (n <= 2097152) || w.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatal(n, w.Code)
		}
	}
	for _, e := range []error{correction.ErrNotConfigured, correction.ErrConflict, correction.ErrSourceStale, correction.ErrAnswerOverlap, correction.ErrLeaseLost, &correction.RateError{RetryAt: time.Now().Add(time.Minute)}, question.ErrIdempotencyConflict, errors.New("private-answer-sentinel")} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/api/v1/corrections/cases", nil)
		correctionError(w, r, e)
		if w.Code < 400 || strings.Contains(w.Body.String(), "private-answer-sentinel") {
			t.Fatal("closed error")
		}
		if _, ok := e.(*correction.RateError); ok && w.Header().Get("Retry-After") == "" {
			t.Fatal("rate retry header")
		}
	}
}
func TestCorrectionHTTPDeadlineIncludesBody(t *testing.T) {
	repo := &correctionHTTPRepo{}
	h := correctionHTTPFixture(t, repo)
	server := httptest.NewServer(h)
	defer server.Close()
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	req, _ := http.NewRequest("POST", server.URL+"/api/v1/corrections/cases", reader)
	for k, v := range contentHeaders() {
		req.Header.Set(k, v)
	}
	go func() { _, _ = writer.Write([]byte("{")) }()
	begin := time.Now()
	res, e := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if e != nil {
		t.Fatal(e)
	}
	defer res.Body.Close()
	raw, e := io.ReadAll(res.Body)
	if e != nil || res.StatusCode != 503 || time.Since(begin) < 7*time.Second || time.Since(begin) > 10*time.Second || !strings.Contains(string(raw), "SERVICE_UNAVAILABLE") || repo.preflights != 1 || len(repo.calls) != 0 {
		t.Fatal("deadline failed", e, res.StatusCode)
	}
}
func TestCorrectionHTTPBodyPhysicalLimit(t *testing.T) {
	repo := &correctionHTTPRepo{}
	h := correctionHTTPFixture(t, repo)
	in := correction.CaseInput{Kind: correction.GradingRuleCase, Rule: &correction.RuleScope{RuleVersion: 1, Kind: "all"}}
	raw, _ := json.Marshal(in)
	for _, n := range []int{65536, 65537} {
		w := privateRequest(h, "POST", "/api/v1/corrections/cases", strings.Repeat(" ", n-len(raw))+string(raw), contentHeaders())
		if (w.Code == 201) != (n == 65536) {
			t.Fatal("byte limit", n, w.Code)
		}
	}
}

type correctionAssetHTTPRepo struct {
	correctionHTTPRepo
	data []byte
}

func (r *correctionAssetHTTPRepo) ReadOwnCorrectionAsset(context.Context, question.Access, string, string) ([]byte, error) {
	return r.data, nil
}
func TestCorrectionHTTPPrivateAsset(t *testing.T) {
	data := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><title>Original private correction</title></svg>`)
	digest := fmt.Sprintf("%x", sha256.Sum256(data))
	repo := &correctionAssetHTTPRepo{data: data}
	svc, e := correction.NewService(repo)
	if e != nil {
		t.Fatal(e)
	}
	h := questionHTTPFixture(t, &httpQuestionRepo{}, nil, func(o *AuthOptions) { o.Correction = &CorrectionOptions{Service: svc, PublicOrigin: privateOrigin} })
	path := "/api/v1/corrections/results/" + contentFixtureID + "/assets/" + digest
	w := privateRequest(h, "GET", path, "", contentHeaders())
	if w.Code != 200 || w.Body.String() != string(data) || w.Header().Get("Content-Type") != "image/svg+xml" || w.Header().Get("Content-Security-Policy") != "sandbox; default-src 'none'" || w.Header().Get("Cache-Control") != "private, no-store" || w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("private corrected SVG route", w.Code, w.Body.String())
	}
	for _, p := range []string{path + "?owner=" + contentFixtureID, path + "?limit=1", path + "?", strings.Replace(path, digest, strings.ToUpper(digest), 1)} {
		if v := privateRequest(h, "GET", p, "", contentHeaders()); v.Code < 400 {
			t.Fatal("invalid private asset route accepted", p)
		}
	}
	if v := privateRequest(h, "POST", path, "{}", contentHeaders()); v.Code != 405 {
		t.Fatal("asset mutation accepted", v.Code)
	}
	if v := privateRequest(h, "GET", path, "", nil); v.Code != 401 {
		t.Fatal("anonymous asset exposed", v.Code)
	}
	repo.data = []byte("unsafe mismatch")
	if v := privateRequest(h, "GET", path, "", contentHeaders()); v.Code < 400 {
		t.Fatal("bad SHA exposed")
	}
}
