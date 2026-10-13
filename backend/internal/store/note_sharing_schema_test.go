package store

import (
	"context"
	"database/sql"
	"errors"
	"github.com/pressly/goose/v3"
	"github.com/yyl1212/math_master/backend/internal/notesharing"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"testing"
)

func sharingSchemaDB(t *testing.T) *sql.DB {
	t.Helper()
	db := testutil.Database(t)
	if e := Up(context.Background(), db, "../../../db/migrations"); e != nil {
		t.Fatal(e)
	}
	return db
}
func TestSharingMigrationPreservesManagedDigest(t *testing.T) {
	db := sharingSchemaDB(t)
	var digest string
	if e := db.QueryRow(managedSchemaDigestSQL, managedTables).Scan(&digest); e != nil || digest != managedSchemaDigest {
		t.Fatal("original schema changed", digest, e)
	}
	var version int
	if e := db.QueryRow("SELECT max(version_id) FROM goose_db_version WHERE is_applied").Scan(&version); e != nil || version != 14 {
		t.Fatal("sharing migration missing", version, e)
	}
	tx, e := db.BeginTx(context.Background(), nil)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	var sharingDigest string
	if e = tx.QueryRow(managedSchemaDigestSQL, noteSharingTables).Scan(&sharingDigest); e != nil {
		t.Fatal(e)
	}
	t.Log("sharing schema digest:", sharingDigest)
	if e = noteSharingSchemaIntegrity(context.Background(), tx); e != nil {
		t.Fatal(e)
	}
	if e = tx.Rollback(); e != nil {
		t.Fatal(e)
	}
	if e = goose.Down(db, "../../../db/migrations"); e != nil {
		t.Fatal("empty extension down", e)
	}
	tx, e = db.BeginTx(context.Background(), nil)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	installed, e := noteSharingInstalled(context.Background(), tx)
	if e != nil || installed {
		t.Fatal("extension absent", installed, e)
	}
}

func sharingSchemaSeed(t *testing.T, db *sql.DB) {
	t.Helper()
	_, e := db.Exec(`BEGIN;
INSERT INTO auth_users(id,username,password_phc) VALUES('11111111-1111-4111-8111-111111111111','sharing_schema','isolated-schema-fixture');
INSERT INTO auth_user_roles(user_id,role) VALUES('11111111-1111-4111-8111-111111111111','learner');
INSERT INTO managed_knowledge(internal_id,external_id,point,public_sources,content_sha256,created_by,updated_by) VALUES('k-'||repeat('a',56),'schema-point','{"id":"schema-point"}','[]',repeat('a',64),'11111111-1111-4111-8111-111111111111','11111111-1111-4111-8111-111111111111');
INSERT INTO managed_study_records(owner_user_id,knowledge_id,state) VALUES('11111111-1111-4111-8111-111111111111','k-'||repeat('a',56),'unlearned');
INSERT INTO managed_study_notes(owner_user_id,knowledge_id,body,revision,knowledge_ref) VALUES('11111111-1111-4111-8111-111111111111','k-'||repeat('a',56),'自己整理的数学笔记',1,jsonb_build_object('id','k-'||repeat('a',56),'contentSha256',repeat('a',64),'sourceKind','managed'));
COMMIT;`)
	if e != nil {
		t.Fatal(e)
	}
}
func TestSharingMigrationUsedDataCannotDown(t *testing.T) {
	db := sharingSchemaDB(t)
	sharingSchemaSeed(t, db)
	_, e := db.Exec(`INSERT INTO note_share_submissions(submission_id,owner_user_id,knowledge_id,note_revision,note_knowledge_ref,title,body,body_sha256,knowledge_title,knowledge_title_zh,topic_keys) SELECT '22222222-2222-4222-8222-222222222222',owner_user_id,knowledge_id,revision,knowledge_ref,'数学笔记',body,repeat('b',64),'Mathematics','数学','[]'::jsonb FROM managed_study_notes;
INSERT INTO note_share_heads(owner_user_id,knowledge_id,status,sequence,current_submission_id) SELECT owner_user_id,knowledge_id,'pending',1,submission_id FROM note_share_submissions;
INSERT INTO note_share_events(event_id,owner_user_id,knowledge_id,submission_id,actor_user_id,action,sequence) SELECT '33333333-3333-4333-8333-333333333333',owner_user_id,knowledge_id,submission_id,owner_user_id,'submit',1 FROM note_share_submissions;
INSERT INTO note_share_receipts(actor_user_id,action,key,resource,input_sha256,receipt) VALUES('11111111-1111-4111-8111-111111111111','submit','test','schema-point',repeat('c',64),'{}');`)
	if e != nil {
		t.Fatal(e)
	}
	for _, table := range []string{"note_share_submissions", "note_share_events", "note_share_receipts"} {
		if _, e = db.Exec("UPDATE " + table + " SET " + map[string]string{"note_share_submissions": "title=title", "note_share_events": "reason=reason", "note_share_receipts": "receipt=receipt"}[table]); e == nil {
			t.Fatal("immutable update accepted", table)
		}
		if _, e = db.Exec("DELETE FROM " + table); e == nil {
			t.Fatal("immutable delete accepted", table)
		}
	}
	if e = goose.Down(db, "../../../db/migrations"); e == nil {
		t.Fatal("used sharing migration rolled back")
	}
}
func TestSharingIncompleteSchemaFailClosed(t *testing.T) {
	db := sharingSchemaDB(t)
	if _, e := db.Exec("ALTER TABLE note_share_heads DROP CONSTRAINT note_share_head_status"); e != nil {
		t.Fatal(e)
	}
	tx, e := db.BeginTx(context.Background(), nil)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	if e = noteSharingSchemaIntegrity(context.Background(), tx); !errors.Is(e, notesharing.ErrNotReady) {
		t.Fatal("damaged schema accepted", e)
	}
}
