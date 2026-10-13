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
	if err := resetFixtureStorage(ctx, db, false); err != nil {
		return err
	}
	return initializeFixtureAccounts(ctx, accounts, admin)
}

func resetFixtureStorage(ctx context.Context, db *sql.DB, clearCatalogue bool) error {
	// This control exists only in the random, verified e2e database, never in server.
	var database string
	if err := db.QueryRowContext(ctx, "SELECT current_database()").Scan(&database); err != nil || !strings.HasPrefix(database, "math_master_test_") {
		return errors.New("unsafe fixture reset")
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return errors.New("fixture reset transaction failed")
	}
	defer tx.Rollback()
	query := "ALTER TABLE goose_db_version DISABLE TRIGGER managed_knowledge_marker_guard; TRUNCATE knowledge_admin_state; INSERT INTO knowledge_admin_state(singleton) VALUES(true); UPDATE goose_db_version SET managed_knowledge_enabled=false WHERE version_id=0; ALTER TABLE goose_db_version ENABLE TRIGGER managed_knowledge_marker_guard;"

	// OpenVerified checks this harness's random database on every physical connection.
	// Reset account-owned workflow records and both sides of the deferred learner FK.
	query += `TRUNCATE study_review_receipts,study_review_events,study_review_schedules,study_review_snapshots,study_review_sessions,note_share_receipts,note_share_events,note_share_heads,note_share_submissions,managed_study_content_changes,managed_study_idempotency,managed_study_events,managed_study_notes,managed_study_records,managed_knowledge_events,managed_knowledge_imports,managed_knowledge_sources,managed_knowledge_topics,managed_knowledge,topic_cutovers,study_legacy_event_links,study_migration_batches,study_records,study_events,study_notes,study_idempotency,study_content_changes,` + strings.Join(append(append([]string{}, correctionResetTableNames...), taxonomyResetTableNames...), ",") + `, feedback_idempotency, feedback_events, feedback_rate_limits, feedback_tickets, learning_idempotency, learning_evidence_dependencies, learning_unlocks, learning_qualification_events, assessment_results, assessment_answers, assessment_items, assessment_attempts, practice_attempts, learner_question_views, learner_answer_exposures, learner_exposure_state, learning_path_nodes, learning_path_enrollments, learning_records, learning_events, question_idempotency, question_events, question_withdrawals, question_heads, question_publication_members, question_publications, question_review_decisions, question_submission_members, question_submission_authors, question_submissions, question_blueprint_sources, question_instance_coverage, question_blueprints, question_instances, question_templates, question_packages, question_workspace_authors, question_workspaces, content_idempotency, content_workflow_events, content_withdrawals, content_publication_manifests, content_review_decisions, content_submission_members, content_submission_authors, content_submissions, content_workspace_assets, content_workspaces, auth_bootstrap, auth_audit_events, auth_rate_limits, auth_preauth, auth_sessions, auth_user_roles, auth_users RESTART IDENTITY;`
	// Experience is fixture state only; every scene starts from the legacy baseline.
	query += "UPDATE topic_learning_state SET experience_mode='legacy',retired_at=NULL WHERE singleton;"
	if clearCatalogue {
		query += "TRUNCATE catalogue_versions, knowledge, path_versions, assets CASCADE;"
	}
	if _, err = tx.ExecContext(ctx, query); err != nil {
		return errors.New("fixture storage reset failed")
	}
	if err = tx.Commit(); err != nil {
		return errors.New("fixture reset commit failed")
	}
	return nil
}

func initializeFixtureAccounts(ctx context.Context, accounts *auth.Service, admin *auth.AdminService) error {
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
