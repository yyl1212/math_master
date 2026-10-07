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
	// Reset account-owned workflow records and both sides of the deferred learner FK.
	if _, err := db.ExecContext(ctx, `TRUNCATE topic_cutovers,study_legacy_event_links,study_migration_batches,study_records,study_events,study_notes,study_idempotency,study_content_changes,`+strings.Join(append(append([]string{}, correctionResetTableNames...), taxonomyResetTableNames...), ",")+`, feedback_idempotency, feedback_events, feedback_rate_limits, feedback_tickets, learning_idempotency, learning_evidence_dependencies, learning_unlocks, learning_qualification_events, assessment_results, assessment_answers, assessment_items, assessment_attempts, practice_attempts, learner_question_views, learner_answer_exposures, learner_exposure_state, learning_path_nodes, learning_path_enrollments, learning_records, learning_events, question_idempotency, question_events, question_withdrawals, question_heads, question_publication_members, question_publications, question_review_decisions, question_submission_members, question_submission_authors, question_submissions, question_blueprint_sources, question_instance_coverage, question_blueprints, question_instances, question_templates, question_packages, question_workspace_authors, question_workspaces, content_idempotency, content_workflow_events, content_withdrawals, content_publication_manifests, content_review_decisions, content_submission_members, content_submission_authors, content_submissions, content_workspace_assets, content_workspaces, auth_bootstrap, auth_audit_events, auth_rate_limits, auth_preauth, auth_sessions, auth_user_roles, auth_users RESTART IDENTITY`); err != nil {
		return errors.New("account fixture reset failed")
	}
	// Experience is fixture state only; every scene starts from the legacy baseline.
	if _, err := db.ExecContext(ctx, "UPDATE topic_learning_state SET experience_mode='legacy',retired_at=NULL WHERE singleton"); err != nil {
		return errors.New("fixture experience reset failed")
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
