package httpapi

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/testutil"
)

const questionHTTPDraft = `{"catalogueVersion":1,"questionPackage":{"kind":"question-bank","schemaVersion":1,"id":"http-bank","version":1,"templates":[],"fixedQuestions":[],"blueprints":[]},"sourceMap":[]}`

type httpQuestionRepo struct {
	question.Repository
	user     auth.User
	fault    error
	rates    *store.Store
	sqlDB    *sql.DB
	entered  chan struct{}
	finished chan struct{}
}

func (r *httpQuestionRepo) QuestionPreflight(context.Context, question.Access, question.Action) (auth.User, error) {
	return r.user, r.fault
}
func (r *httpQuestionRepo) ConsumeRates(ctx context.Context, v []auth.RateKey) error {
	if r.rates != nil {
		return r.rates.ConsumeRates(ctx, v)
	}
	return nil
}
func (r *httpQuestionRepo) CreateQuestionDraft(ctx context.Context, _ question.Access, _ question.DraftInput) (question.DraftView, error) {
	if r.entered != nil {
		r.entered <- struct{}{}
		<-r.finished
	}
	if r.sqlDB != nil {
		_, err := r.sqlDB.ExecContext(ctx, `SELECT pg_sleep(12)`)
		return question.DraftView{}, err
	}
	return question.DraftView{ID: contentFixtureID}, nil
}
func (r *httpQuestionRepo) SaveQuestionDraft(context.Context, question.Access, string, question.SaveDraftInput) (question.DraftView, error) {
	return question.DraftView{ID: contentFixtureID}, nil
}
func (r *httpQuestionRepo) ReadQuestionDraft(_ context.Context, _ question.Access, id string) (question.DraftView, error) {
	if id == contentOtherID {
		return question.DraftView{}, auth.ErrNotFound
	}
	return question.DraftView{ID: id}, nil
}
func (r *httpQuestionRepo) ListQuestionDrafts(context.Context, question.Access, question.ListQuery) (question.Page[question.DraftSummary], error) {
	return question.Page[question.DraftSummary]{Items: []question.DraftSummary{}, Limit: 20}, nil
}
func (r *httpQuestionRepo) AdoptQuestionDraft(context.Context, question.Access, question.AdoptInput) (question.DraftView, error) {
	return question.DraftView{ID: contentFixtureID}, nil
}
func (r *httpQuestionRepo) ValidateQuestionDraft(context.Context, question.Access, string, question.ValidateInput) (question.ValidationReport, error) {
	return question.ValidationReport{}, nil
}
func (r *httpQuestionRepo) SubmitQuestionDraft(context.Context, question.Access, string, question.SubmitInput) (question.SubmissionView, error) {
	return question.SubmissionView{ID: contentFixtureID}, nil
}
func (r *httpQuestionRepo) ListQuestionSubmissions(context.Context, question.Access, question.ListQuery) (question.Page[question.SubmissionSummary], error) {
	return question.Page[question.SubmissionSummary]{Items: []question.SubmissionSummary{}, Limit: 20}, nil
}
func (r *httpQuestionRepo) ReadQuestionSubmission(context.Context, question.Access, string) (question.SubmissionView, error) {
	return question.SubmissionView{ID: contentFixtureID}, nil
}
func (r *httpQuestionRepo) ListQuestionInstances(context.Context, question.Access, string, question.ListQuery) (question.Page[question.Instance], error) {
	return question.Page[question.Instance]{Items: []question.Instance{}, Limit: 20}, nil
}
func (r *httpQuestionRepo) ReviseQuestionSubmission(context.Context, question.Access, string) (question.DraftView, error) {
	return question.DraftView{ID: contentFixtureID}, nil
}
func (r *httpQuestionRepo) DecideQuestionReview(context.Context, question.Access, string, question.ReviewInput) (question.SubmissionView, error) {
	return question.SubmissionView{ID: contentFixtureID}, nil
}
func (r *httpQuestionRepo) ListQuestionPublications(context.Context, question.Access, question.ListQuery) (question.PublicationPage, error) {
	return question.PublicationPage{Page: question.Page[question.PublicationSummary]{Items: []question.PublicationSummary{}, Limit: 20}}, nil
}
func (r *httpQuestionRepo) ReadQuestionPublication(context.Context, question.Access, string) (question.PublicationSummary, error) {
	return question.PublicationSummary{ID: contentFixtureID}, nil
}
func (r *httpQuestionRepo) ListQuestionMembers(context.Context, question.Access, string, question.ListQuery) (question.MemberPage, error) {
	return question.MemberPage{Page: question.Page[question.ManifestMember]{Items: []question.ManifestMember{}, Limit: 20}}, nil
}
func (r *httpQuestionRepo) ListQuestionChanges(context.Context, question.Access, string, question.ListQuery) (question.ChangePage, error) {
	return question.ChangePage{Page: question.Page[question.Change]{Items: []question.Change{}, Limit: 20}}, nil
}
func (r *httpQuestionRepo) PrepareQuestionRelease(context.Context, question.Access, question.PrepareInput) (question.PublicationSummary, error) {
	return question.PublicationSummary{ID: contentFixtureID}, nil
}
func (r *httpQuestionRepo) ActivateQuestionRelease(context.Context, question.Access, string, question.ActivateInput) (question.PublicationSummary, error) {
	return question.PublicationSummary{ID: contentFixtureID}, nil
}
func (r *httpQuestionRepo) PreviewQuestionWithdrawal(context.Context, question.Access, question.WithdrawalPreviewInput, question.ListQuery) (question.WithdrawalPreview, error) {
	return question.WithdrawalPreview{}, nil
}
func (r *httpQuestionRepo) WithdrawQuestionVersion(context.Context, question.Access, question.WithdrawalInput) (question.WithdrawalResult, error) {
	return question.WithdrawalResult{}, nil
}
func (r *httpQuestionRepo) ReadQuestionCoverage(context.Context, question.Access, question.CoverageQuery) (question.CoverageReport, error) {
	return question.CoverageReport{}, nil
}
func questionHTTPFixture(t *testing.T, r *httpQuestionRepo, cr *httpContentRepo, mutate func(*AuthOptions)) http.Handler {
	t.Helper()
	accounts, e := auth.NewService(&httpAuthRepo{preauth: map[auth.Digest]auth.Secret{}}, httpHasher{}, rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	if cr == nil {
		cr = &httpContentRepo{user: r.user}
	}
	shared := publication.NewService(cr)
	service, e := question.NewService(r, shared.AcquireValidation)
	if e != nil {
		t.Fatal(e)
	}
	o := AuthOptions{Accounts: accounts, PublicOrigin: privateOrigin, Content: &ContentOptions{Service: shared, PublicOrigin: privateOrigin, Configured: true}, Question: &QuestionOptions{Service: service, PublicOrigin: privateOrigin, Configured: true}}
	if mutate != nil {
		mutate(&o)
	}
	return NewApplicationHandler(&fakeReader{}, nil, o)
}
func questionAllRoles() auth.User {
	return auth.User{ID: contentFixtureID, Roles: []auth.Role{auth.RoleEditor, auth.RoleReviewer, auth.RoleAdmin}}
}
func questionAssert(t *testing.T, w *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if w.Code != status || code != "" && !strings.Contains(w.Body.String(), `"code":"`+code+`"`) {
		t.Fatalf("want %d %s, got %d %s", status, code, w.Code, w.Body.String())
	}
	if w.Header().Get("Cache-Control") != "private, no-store" || len(w.Result().Cookies()) != 0 || w.Body.Len() > 4194304 {
		t.Fatal("private response bounds/cookies")
	}
}
func TestQuestionHTTPContract(t *testing.T) {
	h := questionHTTPFixture(t, &httpQuestionRepo{user: questionAllRoles()}, nil, nil)
	sha := strings.Repeat("a", 64)
	base := "/api/v1/question-bank"
	cases := []struct {
		method, path, body string
		status             int
	}{
		{"GET", base, "", 404}, {"GET", base + "/drafts/extra/unknown", "", 404}, {"HEAD", base + "/drafts", "", 405},
		{"GET", base + "/drafts", "", 200}, {"POST", base + "/drafts", questionHTTPDraft, 201}, {"PUT", base + "/drafts/" + contentFixtureID, strings.TrimSuffix(questionHTTPDraft, "}") + `,"expectedRevision":1}`, 200},
		{"GET", base + "/drafts/" + contentFixtureID, "", 200}, {"POST", base + "/drafts/adopt", `{"packageId":"http-bank","packageVersion":1,"reason":"Import a reviewed original draft for editing."}`, 201},
		{"POST", base + "/drafts/" + contentFixtureID + "/validate", `{"expectedRevision":1}`, 200}, {"POST", base + "/drafts/" + contentFixtureID + "/submit", `{"expectedRevision":1,"expectedDigest":"` + sha + `"}`, 201},
		{"GET", base + "/submissions", "", 200}, {"GET", base + "/submissions/" + contentFixtureID, "", 200}, {"GET", base + "/submissions/" + contentFixtureID + "/instances?limit=1&offset=2", "", 200},
		{"POST", base + "/submissions/" + contentFixtureID + "/revision", `{}`, 201}, {"POST", base + "/submissions/" + contentFixtureID + "/decision", `{"decision":"return","checks":{"mathematics":false,"explanations":false,"objectives":false,"sources":false,"illustrations":false,"generation":false},"independenceNote":"Independent technical reviewer account.","generationNote":"","note":"Please correct the objective coverage before approval."}`, 200},
		{"GET", base + "/publications", "", 200}, {"GET", base + "/publications/" + contentFixtureID, "", 200}, {"GET", base + "/publications/" + contentFixtureID + "/members", "", 200}, {"GET", base + "/publications/" + contentFixtureID + "/changes", "", 200},
		{"POST", base + "/publications/prepare", `{"submissionIds":["` + contentFixtureID + `"],"expectedKnowledgeHead":null,"expectedQuestionHead":null,"reason":"Prepare an original approved mathematics question bank."}`, 201},
		{"POST", base + "/publications/" + contentFixtureID + "/activate", `{"expectedKnowledgeHead":null,"expectedQuestionHead":null,"expectedManifestSha":"` + sha + `","reason":"Activate the reviewed fixed mathematics snapshot."}`, 200},
		{"POST", base + "/withdrawals/preview?limit=1&offset=0", `{"target":{"kind":"template","id":"fixture","version":1}}`, 200},
		{"POST", base + "/withdrawals", `{"target":{"kind":"instance","id":"fixture","version":1},"expectedKnowledgeHead":null,"expectedQuestionHead":null,"reason":"Permanently withdraw this incorrect original question."}`, 201},
		{"GET", base + "/coverage?knowledgeId=fractions&limit=1", "", 200},
	}
	for _, tc := range cases {
		t.Run(tc.method+tc.path, func(t *testing.T) {
			w := privateRequest(h, tc.method, tc.path, tc.body, contentHeaders())
			questionAssert(t, w, tc.status, "")
			if tc.status == 405 && w.Header().Get("Allow") != "GET, POST" {
				t.Fatal("missing exact Allow", w.Header())
			}
		})
	}
	raw, err := os.ReadFile("../../../api/question-boundary-cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var boundary []struct {
		Name, Method, Path, Body, BodyBase64, Code string
		Status                                     int
	}
	if json.Unmarshal(raw, &boundary) != nil {
		t.Fatal("shared cases invalid")
	}
	for _, tc := range boundary {
		t.Run(tc.Name, func(t *testing.T) {
			body := tc.Body
			if tc.BodyBase64 != "" {
				raw, err := base64.StdEncoding.DecodeString(tc.BodyBase64)
				if err != nil {
					t.Fatal(err)
				}
				body = string(raw)
			}
			questionAssert(t, privateRequest(h, tc.Method, tc.Path, body, contentHeaders()), tc.Status, tc.Code)
		})
	}
}
func TestQuestionPrivateScopeAndErrors(t *testing.T) {
	r := &httpQuestionRepo{user: auth.User{ID: contentFixtureID, Roles: []auth.Role{auth.RoleLearner}}}
	h := questionHTTPFixture(t, r, nil, nil)
	body := &observedBody{}
	req := httptest.NewRequest("POST", "/api/v1/question-bank/drafts", nil)
	req.Body = body
	for k, v := range contentHeaders() {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	questionAssert(t, w, 403, "FORBIDDEN")
	if body.read {
		t.Fatal("learner body read")
	}
	for _, path := range []string{"/drafts", "/submissions/" + contentFixtureID, "/submissions/" + contentFixtureID + "/instances", "/coverage"} {
		questionAssert(t, privateRequest(h, "GET", "/api/v1/question-bank"+path, "", contentHeaders()), 403, "FORBIDDEN")
	}
	r.user = questionAllRoles()
	questionAssert(t, privateRequest(h, "GET", "/api/v1/question-bank/drafts/"+contentOtherID, "", contentHeaders()), 404, "NOT_FOUND")
	disabled := questionHTTPFixture(t, r, nil, func(o *AuthOptions) { o.Question.Configured = false })
	questionAssert(t, privateRequest(disabled, "GET", "/api/v1/question-bank/drafts", "", contentHeaders()), 503, "QUESTION_BANK_NOT_CONFIGURED")
	if privateRequest(disabled, "GET", "/api/v1/content/drafts", "", contentHeaders()).Code != 200 {
		t.Fatal("question absence disabled old content")
	}
	for _, tc := range []struct {
		err    error
		status int
		code   string
	}{{question.ErrDraftConflict, 409, "QUESTION_DRAFT_CONFLICT"}, {question.ErrPublicationStale, 409, "QUESTION_PUBLICATION_STALE"}, {question.ErrReviewConflict, 409, "REVIEW_CONFLICT"}, {question.ErrIdempotencyConflict, 409, "IDEMPOTENCY_CONFLICT"}, {question.ErrImmutableConflict, 409, "IMMUTABLE_CONFLICT"}, {question.ErrVersionConflict, 409, "VERSION_CONFLICT"}, {question.ErrInvalid, 422, "QUESTION_INVALID"}, {question.ErrNotReady, 422, "QUESTION_NOT_READY"}, {question.ErrLimitExceeded, 422, "QUESTION_LIMIT_EXCEEDED"}, {question.ErrReviewRequired, 422, "REVIEW_REQUIRED"}, {auth.ErrReauthRequired, 428, "REAUTHENTICATION_REQUIRED"}, {errors.New("secret SQL answer /private/source/path"), 503, "SERVICE_UNAVAILABLE"}} {
		r.fault = tc.err
		w = privateRequest(h, "GET", "/api/v1/question-bank/drafts", "", contentHeaders())
		questionAssert(t, w, tc.status, tc.code)
		if strings.Contains(w.Body.String(), "secret") || strings.Contains(w.Body.String(), "/private/") {
			t.Fatal("secret leaked")
		}
	}
}
func TestSharedQuestionValidationSlots(t *testing.T) {
	r := &httpQuestionRepo{user: questionAllRoles(), entered: make(chan struct{}, 1), finished: make(chan struct{})}
	cr := &httpContentRepo{user: questionAllRoles(), entered: make(chan struct{}, 1), finished: make(chan struct{})}
	h := questionHTTPFixture(t, r, cr, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{}, 2)
	for _, path := range []string{"/api/v1/content/drafts", "/api/v1/question-bank/drafts"} {
		go func(path string) {
			b := questionHTTPDraft
			if strings.Contains(path, "/content/") {
				b = contentTestDraft
			}
			req := httptest.NewRequest("POST", path, strings.NewReader(b)).WithContext(ctx)
			for k, v := range contentHeaders() {
				req.Header.Set(k, v)
			}
			h.ServeHTTP(httptest.NewRecorder(), req)
			done <- struct{}{}
		}(path)
	}
	for _, entered := range []chan struct{}{r.entered, cr.entered} {
		select {
		case <-entered:
		case <-time.After(2 * time.Second):
			t.Fatal("worker did not acquire actual slot")
		}
	}
	cancel()
	questionAssert(t, privateRequest(h, "POST", "/api/v1/question-bank/drafts", questionHTTPDraft, contentHeaders()), 503, "SERVICE_UNAVAILABLE")
	close(r.finished)
	close(cr.finished)
	for range 2 {
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("worker leak")
		}
	}
	r.entered = nil
	questionAssert(t, privateRequest(h, "POST", "/api/v1/question-bank/drafts", "{", contentHeaders()), 400, "INVALID_REQUEST")
	questionAssert(t, privateRequest(h, "POST", "/api/v1/question-bank/drafts", questionHTTPDraft, contentHeaders()), 201, "")
}
func TestSharedContentQuestionRateWindow(t *testing.T) {
	db := testutil.Database(t)
	if err := store.Up(context.Background(), db, "../../../db/migrations"); err != nil {
		t.Fatal(err)
	}
	repo := store.New(db)
	r := &httpQuestionRepo{user: questionAllRoles(), rates: repo}
	h := questionHTTPFixture(t, r, nil, nil)
	// Seed 29 ordinary writes through the existing content service's real rate keys.
	for range 29 {
		rates, _ := publication.Rates(contentFixtureID, publication.CreateDraftAction)
		if err := repo.ConsumeRates(context.Background(), rates); err != nil {
			t.Fatal(err)
		}
	}
	questionAssert(t, privateRequest(h, "POST", "/api/v1/question-bank/drafts", questionHTTPDraft, contentHeaders()), 201, "")
	w := privateRequest(h, "POST", "/api/v1/question-bank/drafts", questionHTTPDraft, contentHeaders())
	questionAssert(t, w, 429, "RATE_LIMITED")
	if w.Header().Get("Retry-After") == "" {
		t.Fatal("missing rate retry delay")
	}
	for range 9 {
		rates, _ := publication.Rates(contentFixtureID, publication.SubmitDraftAction)
		if err := repo.ConsumeRates(context.Background(), rates); err != nil {
			t.Fatal(err)
		}
	}
	questionAssert(t, privateRequest(h, "GET", "/api/v1/question-bank/coverage", "", contentHeaders()), 200, "")
	questionAssert(t, privateRequest(h, "GET", "/api/v1/question-bank/coverage", "", contentHeaders()), 429, "RATE_LIMITED")
}
func TestQuestionBodyDeadline(t *testing.T) {
	r := &httpQuestionRepo{user: questionAllRoles()}
	h := questionHTTPFixture(t, r, nil, nil)
	req := httptest.NewRequest("POST", "/api/v1/question-bank/drafts", strings.NewReader(questionHTTPDraft+strings.Repeat(" ", 4194305-len(questionHTTPDraft))))
	req.ContentLength = -1
	req.TransferEncoding = []string{"chunked"}
	for k, v := range contentHeaders() {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	questionAssert(t, w, 413, "PAYLOAD_TOO_LARGE")
	srv := httptest.NewServer(h)
	defer srv.Close()
	for _, tc := range []struct {
		path    string
		seconds float64
	}{{"/api/v1/question-bank/drafts", 8}, {"/api/v1/auth/login", 4}} {
		c, err := net.Dial("tcp", strings.TrimPrefix(srv.URL, "http://"))
		if err != nil {
			t.Fatal(err)
		}
		_ = c.SetDeadline(time.Now().Add(11 * time.Second))
		start := time.Now()
		fmt.Fprintf(c, "POST %s HTTP/1.1\r\nHost: fixture\r\nConnection: close\r\nContent-Type: application/json\r\nContent-Length: 100\r\nOrigin: %s\r\nCookie: %s\r\nX-CSRF-Token: %s\r\nIdempotency-Key: %s\r\n\r\n{", tc.path, privateOrigin, contentHeaders()["Cookie"], contentHeaders()["X-CSRF-Token"], contentFixtureID)
		_, err = io.ReadAll(c)
		c.Close()
		elapsed := time.Since(start).Seconds()
		if err != nil || elapsed < tc.seconds-0.5 || elapsed > tc.seconds+1.5 {
			t.Fatalf("actual body deadline %s %.2fs %v", tc.path, elapsed, err)
		}
	}
	db := testutil.Database(t)
	r.sqlDB = db
	start := time.Now()
	questionAssert(t, privateRequest(h, "POST", "/api/v1/question-bank/drafts", questionHTTPDraft, contentHeaders()), 503, "SERVICE_UNAVAILABLE")
	if elapsed := time.Since(start); elapsed < 7*time.Second || elapsed > 10*time.Second {
		t.Fatal("SQL excluded from total deadline", elapsed)
	}
	r.sqlDB = nil
	questionAssert(t, privateRequest(h, "POST", "/api/v1/question-bank/drafts", questionHTTPDraft, contentHeaders()), 201, "")
}
func TestQuestionReadinessDoesNotMigrate(t *testing.T) {
	db := testutil.Database(t)
	ready, err := QuestionReady(context.Background(), db)
	if err != nil || ready {
		t.Fatal("absent schema ready", err)
	}
	var exists bool
	if err = db.QueryRow(`SELECT to_regclass('public.question_workspaces') IS NOT NULL`).Scan(&exists); err != nil || exists {
		t.Fatal("readiness migrated schema")
	}
	if err = store.Up(context.Background(), db, "../../../db/migrations"); err != nil {
		t.Fatal(err)
	}
	ready, err = QuestionReady(context.Background(), db)
	if err != nil || !ready {
		t.Fatal("complete schema not ready", err)
	}
}
