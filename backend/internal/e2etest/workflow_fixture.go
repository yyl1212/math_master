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

// Add volume using a real approved and activated snapshot. Copies are test-only
// prepared history; immutable bytes, members and evidence stay exact. This does
// not manufacture a review or change the actual public head.
func expandWorkflowHistory(ctx context.Context, db *sql.DB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return errors.New("content history fixture unavailable")
	}
	defer tx.Rollback()
	var head string
	if err = tx.QueryRowContext(ctx, `SELECT h.snapshot_id FROM publication_heads h JOIN content_publication_manifests m ON m.snapshot_id=h.snapshot_id JOIN publication_snapshots s ON s.id=h.snapshot_id AND s.status='published'`).Scan(&head); err != nil {
		return errors.New("content history fixture requires reviewed head")
	}
	queries := []string{
		`CREATE TEMP TABLE content_history_fixture_ids ON COMMIT DROP AS SELECT gen_random_uuid()::text AS id FROM generate_series(1,100)`,
		`INSERT INTO publication_snapshots SELECT ids.id,s.catalogue_version,'draft' FROM content_history_fixture_ids ids CROSS JOIN publication_snapshots s WHERE s.id=$1`,
		`INSERT INTO publication_members SELECT ids.id,m.package_id,m.package_version,m.kind,m.id,m.version,m.availability FROM content_history_fixture_ids ids CROSS JOIN publication_members m WHERE m.snapshot_id=$1`,
		`INSERT INTO content_publication_manifests SELECT ids.id,m.base_head,m.manifest,m.manifest_bytes,m.sha256,m.diff,m.creator_user_id,clock_timestamp() FROM content_history_fixture_ids ids CROSS JOIN content_publication_manifests m WHERE m.snapshot_id=$1`,
	}
	for i, q := range queries {
		var err error
		if i == 0 {
			_, err = tx.ExecContext(ctx, q)
		} else {
			_, err = tx.ExecContext(ctx, q, head)
		}
		if err != nil {
			return errors.New("content history fixture unavailable")
		}
	}
	if tx.Commit() != nil {
		return errors.New("content history fixture unavailable")
	}
	return nil
}
