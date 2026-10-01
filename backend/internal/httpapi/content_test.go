package httpapi

import (
	"context"
	"crypto/rand"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const contentFixtureID = "11111111-1111-4111-8111-111111111111"
const contentOtherID = "22222222-2222-4222-8222-222222222222"
const contentTestDraft = `{"catalogueVersion":1,"package":{"schemaVersion":1,"id":"fixture","version":1,"knowledge":[],"units":[],"paths":[],"assets":[]},"assetBytes":[],"sourceMap":[]}`

type httpContentRepo struct {
	publication.Repository
	user     auth.User
	fault    error
	entered  chan struct{}
	finished chan struct{}
	asset    []byte
	boundID  string
}

func (r *httpContentRepo) Preflight(context.Context, publication.Access, publication.Action) (auth.User, error) {
	return r.user, r.fault
}
func (r *httpContentRepo) ConsumeRates(context.Context, []auth.RateKey) error { return r.fault }
func (r *httpContentRepo) CreateDraft(context.Context, publication.Access, publication.DraftInput) (publication.DraftView, error) {
	if r.entered != nil {
		r.entered <- struct{}{}
		<-r.finished
	}
	return publication.DraftView{ID: contentFixtureID}, r.fault
}
func (r *httpContentRepo) ListDrafts(context.Context, publication.Access, publication.ListQuery) (publication.Page[publication.DraftSummary], error) {
	return publication.Page[publication.DraftSummary]{Items: []publication.DraftSummary{}, Limit: 20}, r.fault
}
func (r *httpContentRepo) ReadDraftAsset(_ context.Context, _ publication.Access, id, sha string) ([]byte, error) {
	if id != r.boundID {
		return nil, auth.ErrNotFound
	}
	return r.asset, r.fault
}
func (r *httpContentRepo) ReadSubmissionAsset(ctx context.Context, a publication.Access, id, sha string) ([]byte, error) {
	return r.ReadDraftAsset(ctx, a, id, sha)
}
func contentHTTPFixture(t *testing.T, r *httpContentRepo, mutate func(*AuthOptions)) http.Handler {
	t.Helper()
	accounts, err := auth.NewService(&httpAuthRepo{preauth: map[auth.Digest]auth.Secret{}}, httpHasher{}, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	opts := AuthOptions{Accounts: accounts, PublicOrigin: privateOrigin, Content: &ContentOptions{Service: publication.NewService(r), PublicOrigin: privateOrigin, Configured: true}}
	if mutate != nil {
		mutate(&opts)
	}
	return NewApplicationHandler(&fakeReader{}, nil, opts)
}
func contentHeaders() map[string]string {
	var token auth.Secret
	for i := range token {
		token[i] = 1
	}
	return map[string]string{"Cookie": "mm_session_dev=" + auth.EncodeSecret(token), "Origin": privateOrigin, "Content-Type": "application/json", "X-CSRF-Token": auth.EncodeSecret(token), "Idempotency-Key": contentFixtureID}
}

type observedBody struct{ read bool }

func (b *observedBody) Read([]byte) (int, error) { b.read = true; return 0, io.EOF }
func (*observedBody) Close() error               { return nil }
func TestContentRoutesAndAuth(t *testing.T) {
	repo := &httpContentRepo{user: auth.User{ID: contentFixtureID, Roles: []auth.Role{auth.RoleEditor}}}
	h := contentHTTPFixture(t, repo, nil)
	for _, tc := range []struct {
		method, path string
		status       int
	}{{"GET", "/api/v1/content", 404}, {"GET", "/api/v1/content/unknown", 404}, {"HEAD", "/api/v1/content/drafts", 405}, {"OPTIONS", "/api/v1/content/drafts", 405}, {"GET", "/api/v1/content/drafts/", 404}, {"GET", "/api/v1/content/drafts?limit=20&limit=21", 400}, {"GET", "/api/v1/content/drafts?unexpected=1", 400}, {"GET", "/api/v1/content/drafts/" + contentFixtureID + "?scope=all", 400}, {"GET", "/api/v1/content/drafts/%31" + contentFixtureID[1:], 400}} {
		w := privateRequest(h, tc.method, tc.path, "", contentHeaders())
		if w.Code != tc.status || w.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatal(tc, w.Code)
		}
	}
	headers := contentHeaders()
	headers["Origin"] = "https://untrusted.example"
	w := privateRequest(h, "POST", "/api/v1/content/drafts", contentTestDraft, headers)
	if w.Code != 403 {
		t.Fatal("wrong origin accepted", w.Code)
	}
	req := httptest.NewRequest("POST", "/api/v1/content/drafts", strings.NewReader(contentTestDraft))
	for k, v := range contentHeaders() {
		req.Header.Set(k, v)
	}
	req.Header.Add("X-CSRF-Token", contentHeaders()["X-CSRF-Token"])
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 403 {
		t.Fatal("duplicate CSRF accepted")
	}
	repo.user.Roles = []auth.Role{auth.RoleLearner}
	body := &observedBody{}
	req = httptest.NewRequest("POST", "/api/v1/content/drafts", nil)
	req.Body = body
	for k, v := range contentHeaders() {
		req.Header.Set(k, v)
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != 403 || body.read {
		t.Fatal("unauthorized large body read", w.Code, body.read)
	}
	repo.user.Roles = []auth.Role{auth.RoleEditor}
	for _, tc := range []struct {
		mutate func(*AuthOptions)
		code   string
	}{{func(o *AuthOptions) { o.Content.PublicOrigin = "" }, "AUTH_NOT_CONFIGURED"}, {func(o *AuthOptions) { o.Content.Configured = false }, "CONTENT_NOT_CONFIGURED"}} {
		w = privateRequest(contentHTTPFixture(t, repo, tc.mutate), "GET", "/api/v1/content/drafts", "", contentHeaders())
		if w.Code != 503 || !strings.Contains(w.Body.String(), tc.code) {
			t.Fatal("wrong configuration failure", w.Code, w.Body.String())
		}
	}
	repo.fault = errors.New("secret database connection credential and body")
	w = privateRequest(h, "GET", "/api/v1/content/drafts", "", contentHeaders())
	if w.Code != 503 || strings.Contains(w.Body.String(), "credential") || len(w.Result().Cookies()) != 0 {
		t.Fatal("internal error or cookie leaked")
	}
}
func TestContentSlotsSurviveCancellation(t *testing.T) {
	repo := &httpContentRepo{user: auth.User{ID: contentFixtureID, Roles: []auth.Role{auth.RoleEditor}}, entered: make(chan struct{}, 2), finished: make(chan struct{})}
	h := contentHTTPFixture(t, repo, nil)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{}, 2)
	for range 2 {
		go func() {
			req := httptest.NewRequest("POST", "/api/v1/content/drafts", strings.NewReader(contentTestDraft)).WithContext(ctx)
			for k, v := range contentHeaders() {
				req.Header.Set(k, v)
			}
			h.ServeHTTP(httptest.NewRecorder(), req)
			done <- struct{}{}
		}()
	}
	for range 2 {
		select {
		case <-repo.entered:
		case <-time.After(2 * time.Second):
			t.Fatal("workers did not reach actual computation")
		}
	}
	cancel()
	w := privateRequest(h, "POST", "/api/v1/content/drafts", contentTestDraft, contentHeaders())
	if w.Code != 429 {
		t.Fatal("cancellation released a still-running validation slot", w.Code)
	}
	close(repo.finished)
	for range 2 {
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("workers did not finish")
		}
	}
	repo.entered = nil
	repo.finished = nil
	w = privateRequest(h, "POST", "/api/v1/content/drafts", contentTestDraft, contentHeaders())
	if w.Code != 201 {
		t.Fatal("completed workers did not release slots", w.Code)
	}
}

func TestContentChunkedRawLimits(t *testing.T) {
	repo := &httpContentRepo{user: auth.User{ID: contentFixtureID, Roles: []auth.Role{auth.RoleEditor}}}
	h := contentHTTPFixture(t, repo, nil)
	request := httptest.NewRequest("POST", "/api/v1/content/drafts", strings.NewReader(contentTestDraft+strings.Repeat(" ", (8<<20)+1-len(contentTestDraft))))
	request.ContentLength = -1
	request.TransferEncoding = []string{"chunked"}
	for k, v := range contentHeaders() {
		request.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, request)
	if w.Code != 413 || !strings.Contains(w.Body.String(), "PAYLOAD_TOO_LARGE") {
		t.Fatal("chunked limit bypassed", w.Code)
	}
	for _, input := range []string{`{"target":{"kind":"asset","sha256":"` + strings.Repeat("a", 64) + `","id":"","version":0}}`, `{"target":{"kind":"knowledge","id":"fixture"}}`, `{"target":{"kind":"path","id":"fixture","version":1,"sha256":""}}`} {
		var value publication.WithdrawalPreviewInput
		if decodeContentJSON(strings.NewReader(input), 8192, &value) == nil {
			t.Fatal("withdrawal target discriminator accepted an ambiguous shape")
		}
	}
}
