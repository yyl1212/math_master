package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const privateOrigin = "http://127.0.0.1:3000"
const httpTestPassword = "a public HTTP test passphrase"

type httpAuthRepo struct {
	auth.AccountRepository
	preauth    map[auth.Digest]auth.Secret
	fault      bool
	registered auth.User
}

func (r *httpAuthRepo) ConsumeRates(context.Context, []auth.RateKey) error { return nil }
func (r *httpAuthRepo) ReadSession(context.Context, auth.Digest, bool) (auth.SessionRecord, error) {
	if r.fault {
		return auth.SessionRecord{}, errors.New("private test fault detail")
	}
	return auth.SessionRecord{}, auth.ErrAuthenticationRequired
}
func (r *httpAuthRepo) ReadPreauth(ctx context.Context, hash auth.Digest) (auth.PreauthRecord, error) {
	if r.fault {
		return auth.PreauthRecord{}, auth.ErrUnavailable
	}
	if value, ok := r.preauth[hash]; ok {
		return auth.PreauthRecord{CSRF: value}, nil
	}
	return auth.PreauthRecord{}, auth.ErrCSRF
}
func (r *httpAuthRepo) CreatePreauth(ctx context.Context, input auth.NewPreauth) error {
	if r.fault {
		return auth.ErrUnavailable
	}
	r.preauth[input.TokenHash] = input.CSRF
	return nil
}
func (r *httpAuthRepo) RegisterLearner(ctx context.Context, proof auth.PreauthProof, id, username, phc, request string) (auth.User, error) {
	secret, ok := r.preauth[proof.TokenHash]
	if !ok || !auth.EqualSecret(secret, proof.CSRF) {
		return auth.User{}, auth.ErrCSRF
	}
	delete(r.preauth, proof.TokenHash)
	r.registered = auth.User{ID: id, Username: username, Roles: []auth.Role{auth.RoleLearner}}
	return r.registered, nil
}
func (r *httpAuthRepo) ReadCredential(context.Context, string) (auth.Credential, error) {
	return auth.Credential{UserID: "10000000-0000-4000-8000-000000000001", PHC: "fixturePHC", Version: 1}, nil
}
func (r *httpAuthRepo) LoginSession(ctx context.Context, proof auth.PreauthProof, id string, version int64, next auth.NewSession, request string) (auth.User, error) {
	secret, ok := r.preauth[proof.TokenHash]
	if !ok || !auth.EqualSecret(secret, proof.CSRF) {
		return auth.User{}, auth.ErrCSRF
	}
	delete(r.preauth, proof.TokenHash)
	return auth.User{ID: id, Username: "http_user", Roles: []auth.Role{auth.RoleLearner}}, nil
}

type httpHasher struct{}

func (httpHasher) Hash(context.Context, string) (string, error)         { return "fixturePHC", nil }
func (httpHasher) Verify(context.Context, string, string) (bool, error) { return true, nil }
func authHTTPFixture(t *testing.T) (http.Handler, *httpAuthRepo) {
	t.Helper()
	repo := &httpAuthRepo{preauth: make(map[auth.Digest]auth.Secret)}
	service, err := auth.NewService(repo, httpHasher{}, rand.Reader)
	if err != nil {
		t.Fatal("HTTP fixture failed")
	}
	return NewApplicationHandler(&fakeReader{}, nil, AuthOptions{Accounts: service, PublicOrigin: privateOrigin}), repo
}
func privateRequest(h http.Handler, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	h.ServeHTTP(w, r)
	return w
}
func httpContext(t *testing.T, h http.Handler) (string, string) {
	t.Helper()
	w := privateRequest(h, "GET", "/api/v1/auth/context", "", map[string]string{"X-Requested-With": "MathMaster"})
	if w.Code != 200 {
		t.Fatal("context failed")
	}
	var body struct {
		Data struct {
			CSRF string `json:"csrfToken"`
		}
	}
	if json.Unmarshal(w.Body.Bytes(), &body) != nil || len(w.Result().Cookies()) != 1 {
		t.Fatal("context contract failed")
	}
	cookie := w.Result().Cookies()[0]
	return cookie.Name + "=" + cookie.Value, body.Data.CSRF
}
func TestOriginAndLoginCSRF(t *testing.T) {
	h, _ := authHTTPFixture(t)
	cookie, csrf := httpContext(t, h)
	otherCookie, otherCSRF := httpContext(t, h)
	_ = otherCookie
	for _, tt := range []struct{ origin, csrf, site string }{{"", csrf, ""}, {"null", csrf, ""}, {"https://untrusted.invalid", csrf, ""}, {privateOrigin, "", ""}, {privateOrigin, otherCSRF, ""}, {privateOrigin, csrf, "cross-site"}} {
		w := privateRequest(h, "POST", "/api/v1/auth/login", `{"username":"http_user","password":"`+httpTestPassword+`"}`, map[string]string{"Origin": tt.origin, "Cookie": cookie, "X-CSRF-Token": tt.csrf, "Content-Type": "application/json", "Sec-Fetch-Site": tt.site})
		if w.Code != 403 {
			t.Fatal("origin or CSRF forgery accepted")
		}
	}
	for _, headers := range []map[string]string{{}, {"X-Requested-With": "MathMaster", "Origin": "null"}, {"X-Requested-With": "MathMaster", "Sec-Fetch-Site": "cross-site"}} {
		w := privateRequest(h, "GET", "/api/v1/auth/context", "", headers)
		if w.Code != 403 {
			t.Fatal("unsafe context accepted")
		}
	}
	// Configured origin stays authoritative when client-supplied forwarding headers lie.
	w := privateRequest(h, "GET", "/api/v1/auth/context", "", map[string]string{"X-Requested-With": "MathMaster", "Origin": "https://untrusted.invalid", "Host": "untrusted.invalid", "X-Forwarded-Host": "untrusted.invalid"})
	if w.Code != 403 {
		t.Fatal("forwarding headers overrode configured origin")
	}
}
func TestPrivateCookieBoundary(t *testing.T) {
	h, repo := authHTTPFixture(t)
	cookie, csrf := httpContext(t, h)
	parts := strings.SplitN(cookie, "=", 2)
	for _, bad := range []string{cookie + "; " + cookie, parts[0] + `="` + parts[1] + `"`, cookie + "=", parts[0] + "=short", cookie + " "} {
		w := privateRequest(h, "GET", "/api/v1/auth/context", "", map[string]string{"X-Requested-With": "MathMaster", "Cookie": bad})
		if w.Code != 400 || len(w.Result().Cookies()) != 1 || w.Result().Cookies()[0].Name != parts[0] || w.Result().Cookies()[0].MaxAge != -1 {
			t.Fatal("malformed cookie not rejected and cleared")
		}
	}
	repo.fault = true
	w := privateRequest(h, "GET", "/api/v1/auth/context", "", map[string]string{"X-Requested-With": "MathMaster", "Cookie": "mm_session_dev=" + parts[1]})
	if w.Code != 503 || len(w.Header().Values("Set-Cookie")) != 0 || strings.Contains(w.Body.String(), "private test fault") {
		t.Fatal("DB fault changed cookie or exposed details")
	}
	repo.fault = false
	w = privateRequest(h, "POST", "/api/v1/auth/login", `{"username":"http_user","password":"`+httpTestPassword+`"}`, map[string]string{"Origin": privateOrigin, "X-CSRF-Token": csrf, "Cookie": cookie, "Content-Type": "application/json"})
	if w.Code != 200 || len(w.Header().Values("Set-Cookie")) != 2 {
		t.Fatal("login merged separate cookies")
	}
	service, err := auth.NewService(repo, httpHasher{}, rand.Reader)
	if err != nil {
		t.Fatal("fixture failed")
	}
	prod := NewApplicationHandler(&fakeReader{}, nil, AuthOptions{Accounts: service, PublicOrigin: "https://math.example", Production: true})
	w = privateRequest(prod, "GET", "/api/v1/auth/session", "", map[string]string{"Cookie": "mm_session_dev=" + parts[1]})
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"user":null`) {
		t.Fatal("production accepted development cookie")
	}
}
func TestPrivateProtocol(t *testing.T) {
	h, _ := authHTTPFixture(t)
	for _, tt := range []struct {
		method, path string
		want         int
	}{{"GET", "/api/v1/auth/unknown", 404}, {"HEAD", "/api/v1/auth/context", 405}, {"OPTIONS", "/api/v1/auth/session", 405}, {"GET", "/api/v1/auth/login", 405}, {"GET", "/api/v1/auth/session?extra=1", 400}, {"GET", "/api/v1/auth/", 404}, {"GET", "/api/v1/admin/unknown", 404}} {
		w := privateRequest(h, tt.method, tt.path, "", nil)
		if w.Code != tt.want || w.Header().Get("Cache-Control") != "private, no-store" || w.Header().Get("X-Content-Type-Options") != "nosniff" || w.Header().Get("Content-Type") != "application/json" || strings.Contains(w.Body.String(), "<html") {
			t.Fatal("private method/path/error contract failed")
		}
	}
	disabled := NewApplicationHandler(&fakeReader{}, nil, AuthOptions{})
	w := privateRequest(disabled, "GET", "/api/v1/auth/session", "", nil)
	if w.Code != 503 || !strings.Contains(w.Body.String(), "AUTH_NOT_CONFIGURED") {
		t.Fatal("readonly compatibility failed")
	}
	cookie, csrf := httpContext(t, h)
	w = privateRequest(h, "POST", "/api/v1/auth/register", `{"username":"http_user","password":"`+httpTestPassword+`"}`, map[string]string{"Origin": privateOrigin, "Cookie": cookie, "X-CSRF-Token": csrf, "Content-Type": "application/json", "X-Request-ID": "user-provided"})
	if w.Code != 201 || w.Header().Get("X-Request-ID") == "user-provided" {
		t.Fatal("registration protocol failed")
	}
}

func TestPrivateSessionHasNoCookieSideEffects(t *testing.T) {
	h, _ := authHTTPFixture(t)
	w := privateRequest(h, "GET", "/api/v1/auth/session", "", map[string]string{"Cookie": "mm_session_dev=short"})
	if w.Code != 400 || len(w.Header().Values("Set-Cookie")) != 0 {
		t.Fatal("SSR session response mutated browser cookies")
	}
}
func TestOriginAndLoginCSRFDuplicateHeaders(t *testing.T) {
	h, _ := authHTTPFixture(t)
	cookie, csrf := httpContext(t, h)
	for _, header := range []string{"Origin", "X-CSRF-Token"} {
		r := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{"username":"http_user","password":"`+httpTestPassword+`"}`))
		r.Header.Set("Origin", privateOrigin)
		r.Header.Set("Cookie", cookie)
		r.Header.Set("X-CSRF-Token", csrf)
		r.Header.Set("Content-Type", "application/json")
		r.Header.Add(header, "untrusted-extra-value")
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 403 {
			t.Fatal("ambiguous request verification header accepted")
		}
	}
}
