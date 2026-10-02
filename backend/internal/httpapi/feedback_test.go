package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"github.com/yyl1212/math_master/backend/internal/question"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFeedbackHTTPRoutes(t *testing.T) {
	id := "11111111-1111-4111-8111-111111111111"
	paths := []struct{ path, method string }{{"contexts/site", "GET"}, {"contexts/knowledge/math-root", "GET"}, {"contexts/path/math-route", "GET"}, {"contexts/practice/" + id, "GET"}, {"contexts/assessment/" + id, "GET"}, {"tickets", "GET"}, {"tickets", "POST"}, {"tickets/" + id, "GET"}, {"tickets/" + id + "/events", "GET"}, {"tickets/" + id + "/reply", "POST"}, {"review/tickets", "GET"}, {"review/tickets/" + id, "GET"}, {"review/tickets/" + id + "/events", "GET"}, {"review/tickets/" + id + "/transition", "POST"}}
	for _, p := range paths {
		r, e := routeFeedback("/api/v1/feedback/"+p.path, p.method)
		if e != nil {
			t.Fatal(p, e)
		}
		if _, e = routeFeedback("/api/v1/feedback/"+p.path, "PUT"); e != errPrivateMethod {
			t.Fatal(r, e)
		}
	}
}
func TestFeedbackHTTPResponseBudget(t *testing.T) {
	for _, n := range []int{2097151, 2097152, 2097153} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/api/v1/feedback/tickets", nil)
		feedbackResponse(w, r, 200, strings.Repeat("a", n-2))
		if (w.Code == 200) != (n <= 2097152) || w.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatal(n, w.Code)
		}
	}
}

type feedbackHTTPRepo struct {
	feedback.Repository
	preflights, creates int
	fault               error
	deadline            time.Duration
}

func (r *feedbackHTTPRepo) FeedbackPreflight(ctx context.Context, _ question.Access, _ feedback.Action) (auth.User, error) {
	r.preflights++
	d, _ := ctx.Deadline()
	r.deadline = time.Until(d)
	return auth.User{ID: contentFixtureID, Roles: []auth.Role{auth.RoleLearner}}, r.fault
}
func (r *feedbackHTTPRepo) CreateFeedback(context.Context, question.Access, feedback.CreateInput) (feedback.Envelope[feedback.Receipt], error) {
	r.creates++
	return feedback.Envelope[feedback.Receipt]{ActorID: contentFixtureID, Data: feedback.Receipt{Status: 201}}, nil
}
func TestFeedbackHTTPPrivateBoundary(t *testing.T) {
	repo := &feedbackHTTPRepo{}
	svc, e := feedback.NewService(repo)
	if e != nil {
		t.Fatal(e)
	}
	h := questionHTTPFixture(t, &httpQuestionRepo{}, nil, func(o *AuthOptions) { o.Feedback = &FeedbackOptions{Service: svc, PublicOrigin: privateOrigin} })
	in := feedback.CreateInput{Target: feedback.Target{Kind: "site", Area: feedbackArea("home")}, Source: feedback.Source{Kind: "site"}, Category: "technical_issue", Title: "Title", Message: strings.Repeat("中", 4000), Location: ""}
	b, _ := json.Marshal(in)
	for _, c := range []struct {
		name   string
		body   string
		mutate func(map[string]string)
		status int
	}{{"legal max", string(b), func(map[string]string) {}, 201}, {"oversize", strings.Repeat(" ", 65537), func(map[string]string) {}, 400}, {"origin", string(b), func(h map[string]string) { h["Origin"] = "https://wrong.example" }, 403}, {"csrf", string(b), func(h map[string]string) { delete(h, "X-CSRF-Token") }, 403}, {"key", string(b), func(h map[string]string) { delete(h, "Idempotency-Key") }, 400}, {"cookie", string(b), func(h map[string]string) { delete(h, "Cookie") }, 401}, {"type", string(b), func(h map[string]string) { h["Content-Type"] = "text/plain" }, 400}} {
		headers := contentHeaders()
		c.mutate(headers)
		w := privateRequest(h, "POST", "/api/v1/feedback/tickets", c.body, headers)
		if w.Code != c.status || w.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatal(c.name, w.Code, w.Body.String())
		}
	}
	if repo.creates != 1 || repo.deadline <= 7*time.Second || repo.deadline > 8*time.Second {
		t.Fatal("deadline or boundary", repo.creates, repo.deadline)
	}
	for _, err := range []error{feedback.ErrNotConfigured, feedback.ErrConflict, feedback.ErrTargetStale, feedback.ErrAnswerOverlap, &feedback.RateError{RetryAt: time.Now().UTC().Add(time.Minute)}, question.ErrIdempotencyConflict, errors.New("answer-sentinel")} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("GET", "/api/v1/feedback/tickets", nil)
		feedbackError(w, r, err)
		if w.Code < 400 || strings.Contains(w.Body.String(), "answer-sentinel") {
			t.Fatal("error input leaked")
		}
	}
	w := privateRequest(h, "GET", "/api/v1/feedback/tickets?limit=51", "", contentHeaders())
	if w.Code != 400 {
		t.Fatal("limit")
	}
}
func feedbackArea(v feedback.Area) *feedback.Area { return &v }

func TestFeedbackHTTPDeadlineIncludesBody(t *testing.T) {
	repo := &feedbackHTTPRepo{}
	svc, _ := feedback.NewService(repo)
	h := questionHTTPFixture(t, &httpQuestionRepo{}, nil, func(o *AuthOptions) { o.Feedback = &FeedbackOptions{Service: svc, PublicOrigin: privateOrigin} })
	server := httptest.NewServer(h)
	defer server.Close()
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	req, _ := http.NewRequest("POST", server.URL+"/api/v1/feedback/tickets", reader)
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
	if e != nil || res.StatusCode != 503 || time.Since(begin) < 7*time.Second || time.Since(begin) > 10*time.Second || !strings.Contains(string(raw), "SERVICE_UNAVAILABLE") || repo.preflights != 1 || repo.creates != 0 {
		t.Fatal("deadline failed", e, res.StatusCode, repo.preflights, repo.creates)
	}
}
func TestFeedbackHTTPDuplicateHeaders(t *testing.T) {
	repo := &feedbackHTTPRepo{}
	svc, _ := feedback.NewService(repo)
	h := questionHTTPFixture(t, &httpQuestionRepo{}, nil, func(o *AuthOptions) { o.Feedback = &FeedbackOptions{Service: svc, PublicOrigin: privateOrigin} })
	for _, name := range []string{"Origin", "X-CSRF-Token", "Idempotency-Key", "Content-Type"} {
		r := httptest.NewRequest("POST", "/api/v1/feedback/tickets", strings.NewReader("{}"))
		for k, v := range contentHeaders() {
			r.Header.Set(k, v)
		}
		r.Header.Add(name, r.Header.Get(name))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 400 && w.Code != 403 {
			t.Fatal("duplicate header", name, w.Code)
		}
	}
	if repo.creates != 0 {
		t.Fatal("invalid header entered command")
	}
}
