package e2etest

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"io"
	"strings"

	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/cli"
	"github.com/yyl1212/math_master/backend/internal/store"
)

// Public test-only credentials. Never use this binary with a real database.
const fixturePassword = "Test-only 中文数学密码 with spaces"
const fixtureOrigin = "http://127.0.0.1:18080"

func resetAccounts(ctx context.Context, db *sql.DB, accounts *auth.Service, admin *auth.AdminService) error {
	// OpenVerified checks this harness's random database on every physical connection.
	// Explicitly name all seven tables, including both sides of the deferred learner FK.
	if _, err := db.ExecContext(ctx, `TRUNCATE auth_bootstrap, auth_audit_events, auth_rate_limits, auth_preauth, auth_sessions, auth_user_roles, auth_users RESTART IDENTITY`); err != nil {
		return errors.New("account fixture reset failed")
	}
	if cli.RunAdminInit(ctx, []string{"--username", "auth_admin", "--password-stdin"}, strings.NewReader(fixturePassword+"\n"), io.Discard, io.Discard, admin) != 0 {
		return errors.New("account fixture initialization failed")
	}
	view, delta, err := accounts.Context(ctx, auth.Cookies{})
	if err != nil {
		return errors.New("account fixture context failed")
	}
	_, _, err = accounts.Register(ctx, auth.Cookies{Preauth: delta.SetPreauth}, view.CSRFToken, auth.RegisterInput{Username: "auth_learner", Password: fixturePassword}, "fixture-register")
	if err != nil {
		return errors.New("account fixture registration failed")
	}
	return nil
}

func fixtureAccounts(repo *store.Store) (*auth.Service, *auth.AdminService, error) {
	hasher := auth.NewArgon2Hasher(rand.Reader)
	accounts, err := auth.NewService(repo, hasher, rand.Reader)
	if err != nil {
		return nil, nil, errors.New("account fixture service failed")
	}
	return accounts, auth.NewAdminService(repo, hasher, rand.Reader), nil
}
