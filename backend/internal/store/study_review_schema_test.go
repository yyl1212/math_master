package store

import (
	"context"
	"database/sql"
	"github.com/pressly/goose/v3"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"testing"
)

func reviewSchemaDB(t *testing.T) *sql.DB {
	t.Helper()
	db := testutil.Database(t)
	if e := Up(context.Background(), db, "../../../db/migrations"); e != nil {
		t.Fatal(e)
	}
	return db
}
func TestReviewSchemaConstraintsAndDigests(t *testing.T) {
	db := reviewSchemaDB(t)
	var count int
	if e := db.QueryRow(`SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name IN ('study_review_sessions','study_review_snapshots','study_review_schedules','study_review_events','study_review_receipts')`).Scan(&count); e != nil || count != 5 {
		t.Fatalf("five review tables missing: %d %v", count, e)
	}
	for _, entry := range []struct {
		tables []string
		want   string
	}{{managedTables, "f81857f4af9a3be336d85f279578df8b"}, {noteSharingTables, "2179de9c9ef5b78ec31debb6d5f4fb6c"}} {
		var got string
		if e := db.QueryRow(managedSchemaDigestSQL, entry.tables).Scan(&got); e != nil || got != entry.want {
			t.Fatalf("old schema changed: %s %v", got, e)
		}
	}
	sharingSchemaSeed(t, db)
	seed := `INSERT INTO study_review_sessions(session_id,owner_user_id,knowledge_id,review_id,knowledge_ref,mode,target,answer) VALUES('22222222-2222-4222-8222-222222222222','11111111-1111-4111-8111-111111111111','k-'||repeat('a',56),'33333333-3333-4333-8333-333333333333',jsonb_build_object('id','k-'||repeat('a',56),'contentSha256',repeat('a',64),'sourceKind','managed'),'recall','statement','draft')`
	if _, e := db.Exec(seed); e != nil {
		t.Fatal(e)
	}
	for name, q := range map[string]string{
		"active uniqueness": `INSERT INTO study_review_sessions(session_id,owner_user_id,knowledge_id,review_id,knowledge_ref,mode,target,answer) SELECT gen_random_uuid(),owner_user_id,knowledge_id,review_id,knowledge_ref,mode,target,answer FROM study_review_sessions`,
		"owner FK":          `INSERT INTO study_review_sessions(session_id,owner_user_id,knowledge_id,review_id,knowledge_ref,mode,target) SELECT gen_random_uuid(),'44444444-4444-4444-8444-444444444444',knowledge_id,review_id,knowledge_ref,'read','statement' FROM study_review_sessions`,
		"invalid status":    `UPDATE study_review_sessions SET status='graded'`,
		"sequence guard":    `UPDATE study_review_sessions SET answer='other'`,
		"mode guard":        `UPDATE study_review_sessions SET mode='read',sequence=2`,
	} {
		if _, e := db.Exec(q); e == nil {
			t.Fatalf("accepted %s", name)
		}
	}
	if _, e := db.Exec(`INSERT INTO study_review_snapshots(session_id,owner_user_id,knowledge_id,title,title_zh,knowledge_type,prompt,expected,sha256) SELECT session_id,owner_user_id,knowledge_id,'Title','标题','theorem','[]','[{"kind":"statement","items":["A"]}]',repeat('b',64) FROM study_review_sessions;
 INSERT INTO study_review_events(session_id,owner_user_id,knowledge_id,review_id,sequence,action,knowledge_ref,mode,target) SELECT session_id,owner_user_id,knowledge_id,review_id,1,'opened',knowledge_ref,mode,target FROM study_review_sessions;
 INSERT INTO study_review_receipts(actor_user_id,action,key,resource,input_sha256,session_id,knowledge_id,receipt) SELECT owner_user_id,'start','seed',knowledge_id,repeat('b',64),session_id,knowledge_id,'{}' FROM study_review_sessions;`); e != nil {
		t.Fatal(e)
	}
	for _, table := range []string{"study_review_snapshots", "study_review_events", "study_review_receipts"} {
		for _, action := range []string{"UPDATE " + table + " SET knowledge_id=knowledge_id", "DELETE FROM " + table} {
			if _, e := db.Exec(action); e == nil {
				t.Fatalf("mutable %s", table)
			}
		}
	}
	if _, e := db.Exec(`UPDATE study_review_sessions SET status='compared',sequence=2,draft_revision=1,compared_at=clock_timestamp(),answer='frozen'`); e != nil {
		t.Fatal(e)
	}
	if _, e := db.Exec(`UPDATE study_review_sessions SET answer='copied reference',sequence=3`); e == nil {
		t.Fatal("compared answer overwritten")
	}
	if e := goose.Down(db, "../../../db/migrations"); e == nil {
		t.Fatal("used migration can Down")
	}
}
func TestReviewEmptyMigrationDown(t *testing.T) {
	db := reviewSchemaDB(t)
	if e := goose.Down(db, "../../../db/migrations"); e != nil {
		t.Fatal(e)
	}
	var v int
	if e := db.QueryRow(`SELECT max(version_id) FROM goose_db_version WHERE is_applied`).Scan(&v); e != nil || v != 14 {
		t.Fatalf("empty down target %d %v", v, e)
	}
}

func TestReviewSchemaDigestRecorded(t *testing.T) {
	db := reviewSchemaDB(t)
	tx, e := db.BeginTx(context.Background(), nil)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	if _, e = tx.Exec("SET LOCAL search_path=pg_catalog,public"); e != nil {
		t.Fatal(e)
	}
	var digest string
	e = tx.QueryRow(studyReviewSchemaDigestSQL, []string{"study_review_sessions", "study_review_snapshots", "study_review_schedules", "study_review_events", "study_review_receipts"}).Scan(&digest)
	if e != nil {
		t.Fatal(e)
	}
	t.Log("review schema digest:", digest)
}

func TestReviewIncompleteSchemaFailClosed(t *testing.T) {
	for name, sqlText := range map[string]string{
		"table":      `ALTER TABLE study_review_snapshots RENAME TO damaged_study_review_snapshots`,
		"constraint": `ALTER TABLE study_review_sessions DROP CONSTRAINT study_review_sessions_mode_check`,
		"trigger":    `ALTER TABLE study_review_snapshots DISABLE TRIGGER study_review_snapshot_immutable`,
		"function":   `CREATE OR REPLACE FUNCTION study_review_blank(body text) RETURNS boolean LANGUAGE sql IMMUTABLE STRICT AS 'SELECT false'`,
		"index":      `DROP INDEX study_review_one_active`,
	} {
		t.Run(name, func(t *testing.T) {
			db := reviewSchemaDB(t)
			repo := New(db)
			h, e := repo.ReadReviewHealth(context.Background())
			if e != nil || !h.Capability || !h.SchemaReady {
				t.Fatal(h, e)
			}
			if _, e = db.Exec(sqlText); e != nil {
				t.Fatal(e)
			}
			h, e = repo.ReadReviewHealth(context.Background())
			if e != nil || h.SchemaReady {
				t.Fatal("damaged schema ready", h, e)
			}
		})
	}
}
func TestReviewCompletedGradeCannotBeNull(t *testing.T) {
	db := reviewSchemaDB(t)
	sharingSchemaSeed(t, db)
	_, e := db.Exec(`INSERT INTO study_review_sessions(session_id,owner_user_id,knowledge_id,review_id,knowledge_ref,mode,target,status,sequence,draft_revision,compared_at,graded_at) SELECT gen_random_uuid(),owner_user_id,knowledge_id,gen_random_uuid(),knowledge_ref,'read','statement','completed',3,1,clock_timestamp(),clock_timestamp() FROM managed_study_notes`)
	if e == nil {
		t.Fatal("completed without grade accepted")
	}
}

func TestReviewEventCannotChangeCycle(t *testing.T) {
	db := reviewSchemaDB(t)
	sharingSchemaSeed(t, db)
	_, e := db.Exec(`INSERT INTO study_review_sessions(session_id,owner_user_id,knowledge_id,review_id,knowledge_ref,mode,target) SELECT gen_random_uuid(),owner_user_id,knowledge_id,gen_random_uuid(),knowledge_ref,'read','statement' FROM managed_study_notes`)
	if e != nil {
		t.Fatal(e)
	}
	_, e = db.Exec(`INSERT INTO study_review_events(session_id,owner_user_id,knowledge_id,review_id,sequence,action,knowledge_ref,mode,target) SELECT session_id,owner_user_id,knowledge_id,gen_random_uuid(),1,'opened',knowledge_ref,mode,target FROM study_review_sessions`)
	if e == nil {
		t.Fatal("event bound to wrong original cycle")
	}
}

func TestReviewMissingExtensionKeepsOriginalCapabilities(t *testing.T) {
	db := sharingSchemaDB(t)
	repo := New(db)
	h, e := repo.ReadReviewHealth(context.Background())
	if e != nil || !h.Capability || h.SchemaReady {
		t.Fatal("missing15 false ready", h, e)
	}
	original, e := repo.ReadManagedSchemaHealth(context.Background())
	if e != nil || !original.SchemaReady {
		t.Fatal("missing15 broke original", original, e)
	}
	sharing, e := repo.ReadSchemaHealth(context.Background())
	if e != nil || !sharing.SchemaReady {
		t.Fatal("missing15 broke sharing", sharing, e)
	}
}
