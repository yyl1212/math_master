package store_test

import (
	"context"
	"crypto/rand"
	"database/sql"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/store"
	"testing"
)

const accountTestPassword = "a public isolated test passphrase"
const changedTestPassword = "a changed isolated test passphrase"

type authFixture struct {
	t       *testing.T
	db      *sql.DB
	repo    *store.Store
	service *auth.Service
	ctx     context.Context
}

func newAuthFixture(t *testing.T, initialMigration ...int) *authFixture {
	t.Helper()
	db, repo, ctx := setup(t, initialMigration...)
	service, err := auth.NewService(repo, auth.NewArgon2Hasher(rand.Reader), rand.Reader)
	if err != nil {
		t.Fatal("auth constructor failed")
	}
	return &authFixture{t, db, repo, service, ctx}
}
func (f *authFixture) exec(query string, args ...any) {
	f.t.Helper()
	if _, err := f.db.ExecContext(f.ctx, query, args...); err != nil {
		f.t.Fatal("test state mutation failed")
	}
}
func (f *authFixture) count(query string, args ...any) int {
	f.t.Helper()
	var n int
	if err := f.db.QueryRowContext(f.ctx, query, args...).Scan(&n); err != nil {
		f.t.Fatal("test state query failed")
	}
	return n
}
func (f *authFixture) anonymous() (auth.Cookies, string) {
	f.t.Helper()
	view, delta, err := f.service.Context(f.ctx, auth.Cookies{})
	if err != nil || view.User != nil || delta.SetPreauth == "" {
		f.t.Fatal("anonymous context failed")
	}
	return auth.Cookies{Preauth: delta.SetPreauth}, view.CSRFToken
}
func (f *authFixture) register(username string) auth.User {
	f.t.Helper()
	cookies, csrf := f.anonymous()
	user, delta, err := f.service.Register(f.ctx, cookies, csrf, auth.RegisterInput{Username: username, Password: accountTestPassword}, "test-register")
	if err != nil || delta.SetSession != "" || !delta.ClearPreauth {
		f.t.Fatal("registration failed")
	}
	return user
}
func (f *authFixture) login(username, password string) (auth.Cookies, string) {
	f.t.Helper()
	cookies, csrf := f.anonymous()
	_, delta, err := f.service.Login(f.ctx, cookies, csrf, auth.LoginInput{Username: username, Password: password}, "test-login")
	if err != nil || delta.SetSession == "" || !delta.ClearPreauth {
		f.t.Fatal("login failed")
	}
	result := auth.Cookies{Session: delta.SetSession}
	view, _, err := f.service.Context(f.ctx, result)
	if err != nil || view.User == nil {
		f.t.Fatal("session context failed")
	}
	return result, view.CSRFToken
}
func (f *authFixture) signup(username string) (auth.User, auth.Cookies, string) {
	f.t.Helper()
	user := f.register(username)
	cookies, csrf := f.login(username, accountTestPassword)
	return user, cookies, csrf
}
func tokenHash(t *testing.T, value string) auth.Digest {
	t.Helper()
	secret, err := auth.DecodeSecret(value)
	if err != nil {
		t.Fatal("test token decode failed")
	}
	return auth.TokenDigest(secret)
}
