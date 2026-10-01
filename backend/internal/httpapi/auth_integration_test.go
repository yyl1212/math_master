package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"net/http"
	"testing"
	"time"
)

func TestAuthIntegration(t *testing.T) {
	db := testutil.Database(t)
	ctx, c := context.WithTimeout(context.Background(), 30*time.Second)
	defer c()
	if store.Up(ctx, db, "../../../db/migrations") != nil {
		t.Fatal("HTTP test migration failed")
	}
	repo := store.New(db)
	hasher := auth.NewArgon2Hasher(rand.Reader)
	service, err := auth.NewService(repo, hasher, rand.Reader)
	if err != nil {
		t.Fatal("HTTP service failed")
	}
	admin := auth.NewAdminService(repo, hasher, rand.Reader)
	h := NewApplicationHandler(repo, db, AuthOptions{Accounts: service, Admin: admin, PublicOrigin: privateOrigin})
	cookie, csrf := httpContext(t, h)
	headers := map[string]string{"Origin": privateOrigin, "Cookie": cookie, "X-CSRF-Token": csrf, "Content-Type": "application/json"}
	w := privateRequest(h, "POST", "/api/v1/auth/register", `{"username":"http_learner","password":"`+httpTestPassword+`"}`, headers)
	if w.Code != 201 {
		t.Fatal("real register failed")
	}
	cookie, csrf = httpContext(t, h)
	headers["Cookie"] = cookie
	headers["X-CSRF-Token"] = csrf
	w = privateRequest(h, "POST", "/api/v1/auth/login", `{"username":"http_learner","password":"`+httpTestPassword+`"}`, headers)
	if w.Code != 200 {
		t.Fatal("real login failed")
	}
	var sessionCookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == "mm_session_dev" {
			sessionCookie = c
		}
	}
	if sessionCookie == nil {
		t.Fatal("login session cookie missing")
	}
	headers["Cookie"] = sessionCookie.Name + "=" + sessionCookie.Value
	w = privateRequest(h, "GET", "/api/v1/auth/context", "", map[string]string{"Cookie": headers["Cookie"], "X-Requested-With": "MathMaster"})
	var body struct{ Data auth.ContextView }
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &body) != nil || body.Data.User == nil {
		t.Fatal("real authenticated context failed")
	}
	headers["X-CSRF-Token"] = body.Data.CSRFToken
	w = privateRequest(h, "GET", "/api/v1/admin/users", "", headers)
	if w.Code != 403 {
		t.Fatal("learner read admin data")
	}
	w = privateRequest(h, "POST", "/api/v1/auth/reauth", `{"password":"`+httpTestPassword+`"}`, headers)
	if w.Code != 200 {
		t.Fatal("real reauth failed")
	}
	w = privateRequest(h, "POST", "/api/v1/auth/logout", `{}`, headers)
	if w.Code != 204 || w.Body.Len() != 0 {
		t.Fatal("logout 204 failed")
	}
	w = privateRequest(h, "GET", "/api/v1/auth/session", "", headers)
	if w.Code != 200 || len(w.Result().Cookies()) != 0 {
		t.Fatal("SSR session read changed cookies")
	}
	actor, err := admin.Initialize(ctx, "http_admin", httpTestPassword, "http-init")
	if err != nil {
		t.Fatal("real admin init failed")
	}
	anon, csrf := httpContext(t, h)
	headers["Cookie"] = anon
	headers["X-CSRF-Token"] = csrf
	w = privateRequest(h, "POST", "/api/v1/auth/login", `{"username":"http_admin","password":"`+httpTestPassword+`"}`, headers)
	if w.Code != 200 {
		t.Fatal("admin login failed")
	}
	for _, c := range w.Result().Cookies() {
		if c.Name == "mm_session_dev" {
			headers["Cookie"] = c.Name + "=" + c.Value
		}
	}
	w = privateRequest(h, "GET", "/api/v1/auth/context", "", map[string]string{"Cookie": headers["Cookie"], "X-Requested-With": "MathMaster"})
	if json.Unmarshal(w.Body.Bytes(), &body) != nil {
		t.Fatal("admin context decode failed")
	}
	headers["X-CSRF-Token"] = body.Data.CSRFToken
	w = privateRequest(h, "PUT", "/api/v1/admin/users/"+actor.ID+"/roles", `{"roles":["learner","admin"],"reason":"Reviewed independent staff identity."}`, headers)
	if w.Code != 428 {
		t.Fatal("login bypassed explicit reauth")
	}
	w = privateRequest(h, "POST", "/api/v1/auth/reauth", `{"password":"`+httpTestPassword+`"}`, headers)
	if w.Code != 200 {
		t.Fatal("admin reauth failed")
	}
	w = privateRequest(h, "PUT", "/api/v1/admin/users/"+actor.ID+"/roles", `{"roles":["learner","admin"],"reason":"Reviewed independent staff identity."}`, headers)
	if w.Code != 204 || w.Body.Len() != 0 || len(w.Result().Cookies()) != 0 {
		t.Fatal("same roles mutation protocol failed")
	}
	w = privateRequest(h, "POST", "/api/v1/admin/users/"+actor.ID+"/password-reset", `{"temporaryPassword":"a temporary HTTP test password", "reason":"Requested manual account recovery.", "ownershipNote":"Identity confirmed with trusted records."}`, headers)
	if w.Code != 204 || w.Body.Len() != 0 {
		t.Fatal("reset 204 protocol failed")
	}
	var n int
	if db.QueryRowContext(ctx, "SELECT count(*) FROM publication_heads").Scan(&n) != nil || n != 0 {
		t.Fatal("account HTTP published content")
	}
}
