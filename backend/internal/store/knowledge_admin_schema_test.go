package store_test

import (
	"testing"
)

func TestManagedKnowledgeSchema(t *testing.T) {
	db, _, ctx := setup(t)
	for _, name := range []string{"knowledge_admin_state", "managed_knowledge", "managed_knowledge_topics", "managed_knowledge_sources", "managed_knowledge_imports", "managed_knowledge_events", "managed_study_records", "managed_study_notes", "managed_study_events", "managed_study_idempotency", "managed_study_content_changes"} {
		var exists bool
		if e := db.QueryRowContext(ctx, "SELECT to_regclass($1) IS NOT NULL", name).Scan(&exists); e != nil || !exists {
			t.Fatalf("missing %s: %v", name, e)
		}
	}
	var mode string
	var enabled bool
	if e := db.QueryRowContext(ctx, "SELECT content_mode,enabled_once FROM knowledge_admin_state WHERE singleton").Scan(&mode, &enabled); e != nil || mode != "legacy" || enabled {
		t.Fatalf("startup activation: %s %v %v", mode, enabled, e)
	}
	var marker bool
	if e := db.QueryRowContext(ctx, "SELECT managed_knowledge_enabled FROM goose_db_version WHERE version_id=0").Scan(&marker); e != nil || marker {
		t.Fatal("capability not initially false", e)
	}
}

func TestManagedKnowledgeSchemaUniqueAndProof(t *testing.T) {
	f := newAuthFixture(t)
	u, _, _ := f.signup("schema_admin")
	const id = "k-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	f.exec(`INSERT INTO managed_knowledge(internal_id,external_id,point,public_sources,content_sha256,created_by,updated_by) VALUES($1,'raw-id','{"id":"raw-id"}','[]',repeat('a',64),$2,$2)`, id, u.ID)
	if _, e := f.db.Exec(`INSERT INTO managed_knowledge(internal_id,external_id,point,public_sources,content_sha256,created_by,updated_by) VALUES('k-bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb','raw-id','{"id":"raw-id"}','[]',repeat('a',64),$1,$1)`, u.ID); e == nil {
		t.Fatal("duplicate external id")
	}
	f.exec(`INSERT INTO managed_knowledge_topics(internal_id,external_id,topic_key) VALUES($1,'raw-id','97F40')`, id)
	if _, e := f.db.Exec(`INSERT INTO managed_knowledge_topics(internal_id,external_id,topic_key) VALUES($1,'raw-id','97F40')`, id); e == nil {
		t.Fatal("duplicate association")
	}
	if _, e := f.db.Exec(`INSERT INTO managed_knowledge_topics(internal_id,external_id,topic_key) VALUES($1,'another-id','97F50')`, id); e == nil {
		t.Fatal("wrong entity pairing")
	}
	var n int
	if e := f.db.QueryRow(`SELECT count(*) FROM pg_constraint WHERE conrelid='feedback_tickets'::regclass AND conname='feedback_managed_target_shape' AND pg_get_constraintdef(oid) LIKE '%feedback_target_shape(%'`).Scan(&n); e != nil || n != 1 {
		t.Fatal("feedback shape not rebound", e)
	}
}
