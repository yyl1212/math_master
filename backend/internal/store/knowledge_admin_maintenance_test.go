package store_test

import (
	"errors"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/study"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"strings"
	"testing"
	"time"
)

func TestManagedCapabilityFailClosed(t *testing.T) {
	for _, mutation := range []string{"DROP TABLE managed_study_notes", "ALTER TABLE managed_knowledge DROP CONSTRAINT managed_knowledge_external_id_key", "ALTER TABLE managed_knowledge_events DISABLE TRIGGER managed_knowledge_event_immutable", "DROP TABLE knowledge_admin_state"} {
		t.Run(mutation, func(t *testing.T) {
			f := newKnowledgeAdminFixture(t)
			h, e := f.repo.ReadManagedSchemaHealth(f.ctx)
			if e != nil || !h.SchemaReady || !h.Capability {
				t.Fatal(h, e)
			}
			f.ActivateManaged()
			h, e = f.repo.ReadManagedSchemaHealth(f.ctx)
			if e != nil || !h.SchemaReady || !h.ManagedMode {
				t.Fatal(h, e)
			}
			f.exec(mutation)
			h, _ = f.repo.ReadManagedSchemaHealth(f.ctx)
			if h.SchemaReady {
				t.Fatal("partial current schema marked ready")
			}
			if _, e = f.repo.ListManagedKnowledge(f.ctx, f.Access("admin_a", "partial"), knowledgeadmin.Query{Limit: 20}); e == nil {
				t.Fatal("partial schema still serves management")
			}
		})
	}
}

func oldThirty(t *testing.T) *knowledgeAdminFixture {
	f := newKnowledgeAdminFixture(t)
	f.repo = store.NewWithTrustedCodeSHA(f.db, strings.Repeat("a", 40))
	v := input(t, func(p *content.Package) {
		base := p.Knowledge[0]
		p.ID = "cleanup-original-thirty"
		p.Knowledge = nil
		p.Units = []content.Unit{}
		p.Paths = []content.Path{}
		p.Assets = []content.Asset{}
		for i := 0; i < 30; i++ {
			k := base
			k.ID = fmt.Sprintf("old-test-%02d", i)
			k.Relations = []content.Relation{}
			p.Knowledge = append(p.Knowledge, k)
		}
	})
	r, e := f.repo.ImportDraft(f.ctx, v)
	if e != nil {
		t.Fatal(e)
	}
	f.exec(`UPDATE publication_snapshots SET status='published' WHERE id=$1`, "draft-"+r.SHA256)
	f.exec(`INSERT INTO publication_heads(singleton,snapshot_id) VALUES(true,$1)`, "draft-"+r.SHA256)
	return f
}
func cleanupInput(f *knowledgeAdminFixture) knowledgeadmin.CutoverInput {
	ids := []string{}
	for i := 0; i < 30; i++ {
		ids = append(ids, fmt.Sprintf("old-test-%02d", i))
	}
	return knowledgeadmin.CutoverInput{ActorID: f.ids["admin_a"], OldIDs: ids, BackupRecord: strings.Repeat("b", 64), BackupCreatedAt: time.Now().UTC(), RestoreVerified: true, OffsiteVerified: true, CodeSHA: strings.Repeat("a", 40)}
}
func TestExactOldThirtyCleanup(t *testing.T) {
	f := oldThirty(t)
	in := cleanupInput(f)
	f.exec(`INSERT INTO study_records(owner_user_id,knowledge_id,state,sequence,body,last_known_ref,updated_at) VALUES($1,'old-test-00','learning',1,'{"knowledgeId":"old-test-00","state":"learning","sequence":1}',jsonb_build_object('id','old-test-00','version',1,'sha256',(SELECT sha256 FROM knowledge_versions WHERE id='old-test-00')),clock_timestamp())`, f.ids["learner_a"])
	f.exec(`INSERT INTO study_notes(owner_user_id,knowledge_id,revision,body,knowledge_ref,updated_at) SELECT owner_user_id,knowledge_id,1,'Original isolated note.',last_known_ref,clock_timestamp() FROM study_records WHERE knowledge_id='old-test-00'`)
	f.exec(`CREATE FUNCTION isolated_cleanup_queue() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RETURN OLD; END $$`)
	f.exec(`CREATE CONSTRAINT TRIGGER isolated_cleanup_deferred AFTER DELETE ON study_notes DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION isolated_cleanup_queue()`)
	f.Import("admin_a", "replacement", f.doc, true)
	p, e := f.repo.PlanKnowledgeCutover(f.ctx, in)
	if e != nil || p.Counts["published"] != 30 || p.Counts["study_records"] != 1 || p.Counts["study_notes"] != 1 {
		t.Fatal(p, e)
	}
	f.ActivateManaged()
	r, e := f.repo.ApplyKnowledgeCutover(f.ctx, p)
	if e != nil || r.RemovedPublicKnowledge != 30 || r.ProtectedFingerprint != p.ProtectedFingerprint {
		t.Fatal(r, e)
	}
	again, e := f.repo.ApplyKnowledgeCutover(f.ctx, p)
	if e != nil || again.OperationID != r.OperationID || f.count(`SELECT count(*) FROM managed_knowledge_events WHERE action='clean-old'`) != 1 {
		t.Fatal("cleanup replay", e)
	}
	tampered := p
	tampered.Counts = map[string]int{"published": 31}
	if _, e = f.repo.ApplyKnowledgeCutover(f.ctx, tampered); e == nil {
		t.Fatal("different input replayed as an original receipt")
	}
	if f.count(`SELECT count(*) FROM knowledge_versions`) != 30 {
		t.Fatal("immutable content facts removed")
	}
	if _, e = f.repo.MigrateLegacyStudyBatch(f.ctx, 50, nil); e == nil {
		t.Fatal("old migration could resurrect cleared records")
	}
}
func TestExactOldThirtyRejectsChangedPlanAndMissingBackup(t *testing.T) {
	f := oldThirty(t)
	in := cleanupInput(f)
	for _, change := range []func(*knowledgeadmin.CutoverInput){func(i *knowledgeadmin.CutoverInput) { i.OldIDs = append(i.OldIDs, "old-test-31") }, func(i *knowledgeadmin.CutoverInput) { i.BackupRecord = "" }, func(i *knowledgeadmin.CutoverInput) { i.RestoreVerified = false }, func(i *knowledgeadmin.CutoverInput) { i.OffsiteVerified = false }} {
		b := in
		b.OldIDs = append([]string{}, in.OldIDs...)
		change(&b)
		if _, e := f.repo.PlanKnowledgeCutover(f.ctx, b); e == nil {
			t.Fatal("unsafe scope accepted")
		}
	}
	f.Import("admin_a", "replacement", f.doc, true)
	p, e := f.repo.PlanKnowledgeCutover(f.ctx, in)
	if e != nil {
		t.Fatal(e)
	}
	f.ActivateManaged()
	p.Input.OldIDs[0] = "unrelated"
	if _, e = f.repo.ApplyKnowledgeCutover(f.ctx, p); e == nil {
		t.Fatal("changed scope accepted")
	}
}

func TestExactOldThirtyRollbackAndNewAssociation(t *testing.T) {
	for _, stage := range []string{"new-association", "interrupted"} {
		t.Run(stage, func(t *testing.T) {
			f := oldThirty(t)
			f.Import("admin_a", "replacement", f.doc, true)
			f.exec(`INSERT INTO study_records(owner_user_id,knowledge_id,state,sequence,body,last_known_ref,updated_at) VALUES($1,'old-test-00','learning',1,'{"knowledgeId":"old-test-00","state":"learning","sequence":1}',jsonb_build_object('id','old-test-00','version',1,'sha256',(SELECT sha256 FROM knowledge_versions WHERE id='old-test-00')),clock_timestamp())`, f.ids["learner_a"])
			f.exec(`INSERT INTO study_notes(owner_user_id,knowledge_id,revision,body,knowledge_ref,updated_at) SELECT owner_user_id,knowledge_id,1,'Retained after interrupted cleanup.',last_known_ref,clock_timestamp() FROM study_records`)
			if stage == "interrupted" {
				f.exec(`CREATE TRIGGER isolated_cleanup_reject BEFORE DELETE ON study_records FOR EACH ROW EXECUTE FUNCTION reject_content_update()`)
			}
			p, e := f.repo.PlanKnowledgeCutover(f.ctx, cleanupInput(f))
			if e != nil {
				t.Fatal(e)
			}
			f.ActivateManaged()
			if stage == "new-association" {
				f.exec(`INSERT INTO study_idempotency(owner_user_id,action,knowledge_id,key,input_sha,receipt) VALUES($1,'begin','old-test-00',gen_random_uuid(),repeat('a',64),jsonb_build_object('actorId',$2::text))`, f.ids["learner_a"], f.ids["learner_a"])
			}
			if _, e = f.repo.ApplyKnowledgeCutover(f.ctx, p); e == nil {
				t.Fatal("unsafe transaction accepted")
			}
			if f.count(`SELECT count(*) FROM study_notes`) != 1 || f.count(`SELECT count(*) FROM study_records`) != 1 || f.count(`SELECT count(*) FROM managed_knowledge_events WHERE action='clean-old'`) != 0 {
				t.Fatal("partial cleanup committed")
			}
			var enabled string
			if e = f.db.QueryRow(`SELECT tgenabled FROM pg_trigger WHERE tgname='study_note_guard'`).Scan(&enabled); e != nil || enabled != "O" {
				t.Fatal("guard not restored", e)
			}
		})
	}
}

func TestManagedActivationExplicitAndPermanent(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	f.repo = store.NewWithTrustedCodeSHA(f.db, strings.Repeat("a", 40))
	in := cleanupInput(f)
	in.OldIDs = []string{}
	f.Import("admin_a", "replacement", f.doc, true)
	if _, e := f.repo.ActivateManagedKnowledge(f.ctx, in); e == nil {
		t.Fatal("activation without the complete published directory")
	}
	batch := testutil.TaxonomyBatch()
	v, e := f.repo.InstallTaxonomyBatch(f.ctx, batch)
	if e != nil {
		t.Fatal(e)
	}
	var release string
	if e = f.db.QueryRow(`SELECT gen_random_uuid()::text`).Scan(&release); e != nil {
		t.Fatal(e)
	}
	f.exec(`INSERT INTO taxonomy_releases(id,taxonomy_version_id,status,manifest_sha,assignments_sha,body,creator_user_id) VALUES($1,$2,'draft',repeat('c',64),repeat('d',64),'{}',$3)`, release, v.ID, f.ids["admin_a"])
	f.exec(`UPDATE taxonomy_releases SET status='published' WHERE id=$1`, release)
	f.exec(`INSERT INTO taxonomy_heads(singleton,release_id) VALUES(true,$1)`, release)
	r, e := f.repo.ActivateManagedKnowledge(f.ctx, in)
	if e != nil || r.Counts["taxonomyNodes"] != 6603 {
		t.Fatal(r, e)
	}
	again, e := f.repo.ActivateManagedKnowledge(f.ctx, in)
	if e != nil || r.OperationID != again.OperationID || f.count(`SELECT count(*) FROM managed_knowledge_events WHERE action='activate'`) != 1 {
		t.Fatal("activation replay", e)
	}
	if _, e = f.db.Exec(`UPDATE knowledge_admin_state SET content_mode='legacy',enabled_once=false,activated_at=NULL`); e == nil {
		t.Fatal("managed state reverted")
	}
	if _, e = f.db.Exec(`UPDATE goose_db_version SET managed_knowledge_enabled=false WHERE version_id=0`); e == nil {
		t.Fatal("permanent marker removed")
	}
	h, e := f.repo.ReadManagedSchemaHealth(f.ctx)
	if e != nil || !h.SchemaReady || !h.ManagedMode {
		t.Fatal(h, e)
	}
}

func TestManagedActivationFencesWaitingLegacyMigration(t *testing.T) {
	f, _ := newCutoverFixture(t)
	tx, e := f.db.BeginTx(f.ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	if _, e = tx.Exec(`SELECT pg_advisory_xact_lock(1296127049)`); e != nil {
		t.Fatal(e)
	}
	done := make(chan error, 1)
	go func() { _, e := f.repo.MigrateLegacyStudyBatch(f.ctx, 50, nil); done <- e }()
	deadline := time.Now().Add(3 * time.Second)
	waiting := false
	for time.Now().Before(deadline) {
		if f.count(`SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND wait_event='advisory' AND query LIKE '%pg_advisory_xact_lock_shared%'`) > 0 {
			waiting = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !waiting {
		t.Fatal("migration did not reach the activation fence")
	}
	f.exec(`UPDATE knowledge_admin_state SET content_mode='managed',enabled_once=true,activated_at=clock_timestamp()`)
	f.exec(`UPDATE goose_db_version SET managed_knowledge_enabled=true WHERE version_id=0`)
	if e = tx.Commit(); e != nil {
		t.Fatal(e)
	}
	if e = <-done; !errors.Is(e, study.ErrModuleRetired) {
		t.Fatal("waiting old migration passed the committed managed marker", e)
	}
}

func TestManagedCapabilityMissingMarkerCannotRestoreLegacyReads(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	f.Import("admin_a", "seed", f.doc, true)
	f.ActivateManaged()
	f.exec(`ALTER TABLE goose_db_version DROP COLUMN managed_knowledge_enabled`)
	if mode, e := f.repo.ReadContentMode(f.ctx); e == nil {
		t.Fatal("missing permanent structure restored a legacy mode", mode)
	}
	if _, e := f.repo.ListManagedKnowledge(f.ctx, f.Access("admin_a", "missing-marker"), knowledgeadmin.Query{Limit: 20}); e == nil {
		t.Fatal("management serves with a missing permanent marker")
	}
}
