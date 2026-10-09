package store_test

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/study"
	"testing"
	"time"
)

func TestManagedAdminRevocation(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	for _, name := range []string{"editor_a", "reviewer_a", "learner_a"} {
		if _, e := f.repo.PreviewManagedImport(f.ctx, f.Access(name, "forbidden"), f.doc, managedInputSHA(f.doc)); !errors.Is(e, auth.ErrForbidden) {
			t.Fatalf("%s could upload: %v", name, e)
		}
	}
	if _, e := f.repo.KnowledgePreflight(f.ctx, knowledgeadmin.Access{}); !errors.Is(e, auth.ErrAuthenticationRequired) {
		t.Fatal(e)
	}
	p, e := f.repo.PreviewManagedImport(f.ctx, f.Access("admin_a", "preview"), f.doc, managedInputSHA(f.doc))
	if e != nil {
		t.Fatal(e)
	}
	f.exec("DELETE FROM auth_user_roles WHERE user_id=$1 AND role='admin'", f.ids["admin_a"])
	_, e = f.repo.ApplyManagedImport(f.ctx, f.Access("admin_a", "apply"), p.ImportID, knowledgeadmin.ApplyInput{SelectedIndexes: []int{0, 1}, Publish: true, PreviewToken: p.PreviewToken})
	if !errors.Is(e, auth.ErrForbidden) {
		t.Fatal("revoked role accepted", e)
	}
	if f.count("SELECT count(*) FROM managed_knowledge") != 0 || f.count("SELECT count(*) FROM managed_knowledge_events") != 0 {
		t.Fatal("failed write persisted")
	}
	f.exec("UPDATE auth_users SET must_change_password=true WHERE id=$1", f.ids["admin_b"])
	if _, e = f.repo.PreviewManagedImport(f.ctx, f.Access("admin_b", "password"), f.doc, managedInputSHA(f.doc)); !errors.Is(e, auth.ErrPasswordChangeRequired) {
		t.Fatal("must-change accepted", e)
	}
}
func TestManagedAdminPreflightRequiresCSRF(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	a := f.Access("admin_a", "preflight")
	a.CSRF = auth.Secret{}
	if _, e := f.repo.KnowledgePreflight(f.ctx, a); !errors.Is(e, auth.ErrCSRF) {
		t.Fatal("large upload can read body before CSRF proof", e)
	}
}

func TestManagedActivationFencesWaitingLegacyKnowledgeWrite(t *testing.T) {
	f := newWorkflowFixture(t)
	tx, e := f.db.BeginTx(f.ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	if _, e = tx.Exec(`SELECT pg_advisory_xact_lock(1296127049)`); e != nil {
		t.Fatal(e)
	}
	before := f.count(`SELECT count(*) FROM content_workspaces`)
	events := f.count(`SELECT count(*) FROM content_workflow_events`)
	a, in := f.Access("author_a", false), f.Input()
	done := make(chan error, 1)
	go func() { _, e := f.repo.CreateDraft(f.ctx, a, in); done <- e }()
	deadline := time.Now().Add(900 * time.Millisecond)
	waiting := false
	for time.Now().Before(deadline) {
		if f.count(`SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND wait_event='advisory' AND query LIKE '%pg_advisory_xact_lock(%'`) > 0 {
			waiting = true
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !waiting {
		t.Fatal("request did not reach the cutover fence")
	}
	f.exec(`UPDATE knowledge_admin_state SET content_mode='managed',enabled_once=true,activated_at=clock_timestamp()`)
	f.exec(`UPDATE goose_db_version SET managed_knowledge_enabled=true WHERE version_id=0`)
	if e = tx.Commit(); e != nil {
		t.Fatal(e)
	}
	if e = <-done; !errors.Is(e, study.ErrModuleRetired) {
		t.Fatal("old writer passed managed activation", e)
	}
	if before != f.count(`SELECT count(*) FROM content_workspaces`) || events != f.count(`SELECT count(*) FROM content_workflow_events`) {
		t.Fatal("retired write left business rows")
	}
}
