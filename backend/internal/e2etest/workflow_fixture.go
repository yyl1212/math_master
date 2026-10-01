package e2etest

import (
	"context"
	"database/sql"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
)

// All actors live only in the harness's verified random database. Role grants
// use the account service, including real login and recent verification.
func workflowAccounts(ctx context.Context, accounts *auth.Service, admin *auth.AdminService) error {
	view, delta, err := accounts.Context(ctx, auth.Cookies{})
	if err != nil {
		return errors.New("content fixture context failed")
	}
	_, session, err := accounts.Login(ctx, auth.Cookies{Preauth: delta.SetPreauth}, view.CSRFToken, auth.LoginInput{Username: "auth_admin", Password: fixturePassword}, "content-fixture-login")
	if err != nil {
		return errors.New("content fixture admin login failed")
	}
	cookies := auth.Cookies{Session: session.SetSession}
	view, _, err = accounts.Context(ctx, cookies)
	if err != nil {
		return errors.New("content fixture admin context failed")
	}
	if _, err = accounts.Reauthenticate(ctx, cookies, view.CSRFToken, auth.ReauthInput{Password: fixturePassword}, "content-fixture-reauth"); err != nil {
		return errors.New("content fixture reauthentication failed")
	}
	for _, actor := range []struct {
		name  string
		roles []auth.Role
	}{{"content_editor", []auth.Role{auth.RoleLearner, auth.RoleEditor, auth.RoleReviewer}}, {"content_editor_two", []auth.Role{auth.RoleLearner, auth.RoleEditor, auth.RoleReviewer}}, {"content_reviewer", []auth.Role{auth.RoleLearner, auth.RoleReviewer}}, {"content_admin", []auth.Role{auth.RoleLearner, auth.RoleAdmin}}} {
		anonymous, pre, err := accounts.Context(ctx, auth.Cookies{})
		if err != nil {
			return errors.New("content fixture registration context failed")
		}
		user, _, err := accounts.Register(ctx, auth.Cookies{Preauth: pre.SetPreauth}, anonymous.CSRFToken, auth.RegisterInput{Username: actor.name, Password: fixturePassword}, "content-fixture-register")
		if err != nil {
			return errors.New("content fixture registration failed")
		}
		if _, err = admin.ReplaceRoles(ctx, cookies, view.CSRFToken, user.ID, auth.RolesInput{Roles: actor.roles, Reason: "Grant isolated browser workflow test responsibilities."}, "content-fixture-roles"); err != nil {
			return errors.New("content fixture role grant failed")
		}
	}
	return nil
}
func resetWorkflow(ctx context.Context, db *sql.DB, accounts *auth.Service, admin *auth.AdminService) error {
	if err := resetAccounts(ctx, db, accounts, admin); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, "TRUNCATE catalogue_versions, knowledge, path_versions, assets CASCADE"); err != nil {
		return errors.New("content fixture reset failed")
	}
	return workflowAccounts(ctx, accounts, admin)
}
