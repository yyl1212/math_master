package store

import (
	"context"
	"github.com/pressly/goose/v3"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"strings"
	"testing"
)

func TestPublishHealth16IntegrityAndMissing(t *testing.T) {
	db := reviewSchemaDB(t)
	repo := New(db)
	ctx := context.Background()
	health, e := repo.ReadPublishHealth(ctx)
	if e != nil || !health.Capability || health.SchemaReady {
		t.Fatal(health, e)
	}
	if e = goose.UpContext(ctx, db, "../../../db/migrations"); e != nil {
		t.Fatal(e)
	}
	health, e = repo.ReadPublishHealth(ctx)
	if e != nil || !health.Capability || !health.SchemaReady {
		t.Fatal(health, e)
	}
	if _, e = db.Exec("ALTER TABLE managed_knowledge_publish_items DISABLE TRIGGER managed_publish_item_guard"); e != nil {
		t.Fatal(e)
	}
	health, e = repo.ReadPublishHealth(ctx)
	if e != nil || !health.Capability || health.SchemaReady {
		t.Fatal(health, e)
	}
	original, e := repo.ReadReviewHealth(ctx)
	if e != nil || !original.SchemaReady {
		t.Fatal("publish damage affected review", original, e)
	}
}

func TestPublishSchema16KeepsOriginal(t *testing.T) {
	db := testutil.Database(t)
	if e := Up(context.Background(), db, "../../../db/migrations"); e != nil {
		t.Fatal(e)
	}
	for _, v := range []struct {
		tables []string
		want   string
	}{{managedTables, managedSchemaDigest}, {noteSharingTables, noteSharingDigest}, {studyReviewTables, studyReviewDigest}} {
		var got string
		q := managedSchemaDigestSQL
		if len(v.tables) == 5 {
			q = studyReviewSchemaDigestSQL
		}
		if e := db.QueryRow(q, v.tables).Scan(&got); e != nil || got != v.want {
			t.Fatal(got, v.want, e)
		}
	}
	var got string
	if e := db.QueryRow(managedSchemaDigestSQL, publishTables).Scan(&got); e != nil {
		t.Fatal(e)
	}
	if got != publishSchemaDigest {
		t.Fatalf("publish schema capture %s", got)
	}
	tx, e := db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	if e = publishSchemaIntegrity(context.Background(), tx); e != nil {
		t.Fatal(e)
	}
}
func TestPublishMigrationEmptyDownAndUsedGuard(t *testing.T) {
	db := testutil.Database(t)
	ctx := context.Background()
	if e := Up(ctx, db, "../../../db/migrations"); e != nil {
		t.Fatal(e)
	}
	if e := goose.DownContext(ctx, db, "../../../db/migrations"); e != nil {
		t.Fatal(e)
	}
	v, e := goose.GetDBVersionContext(ctx, db)
	if e != nil || v != 15 {
		t.Fatal(v, e)
	}
	if e = goose.UpContext(ctx, db, "../../../db/migrations"); e != nil {
		t.Fatal(e)
	}
}

func TestPublishSchema16Constraints(t *testing.T) {
	db := testutil.Database(t)
	ctx := context.Background()
	if e := Up(ctx, db, "../../../db/migrations"); e != nil {
		t.Fatal(e)
	}
	sharingSchemaSeed(t, db)
	const job = "22222222-2222-4222-8222-222222222222"
	const owner = "11111111-1111-4111-8111-111111111111"
	if _, e := db.Exec(strings.NewReplacer("$1", "'"+job+"'", "$2", "'"+owner+"'").Replace(`INSERT INTO managed_knowledge_publish_jobs(job_id,owner_user_id,mode,key,input_sha256,total,pending) VALUES($1,$2,'all','seed',repeat('a',64),1,1);INSERT INTO managed_knowledge_publish_items(job_id,knowledge_id,expected_token,ordinal,external_id,title,title_zh) VALUES($1,'k-'||repeat('a',56),gen_random_uuid(),0,'a','Title','标题')`)); e != nil {
		t.Fatal(e)
	}
	for name, q := range map[string]string{"snapshot": "UPDATE managed_knowledge_publish_items SET expected_token=gen_random_uuid()", "fake counts": "UPDATE managed_knowledge_publish_jobs SET sequence=2,pending=0,published=1", "fake terminal": "UPDATE managed_knowledge_publish_items SET state='published'", "wrong owner": `INSERT INTO managed_knowledge_publish_receipts(owner_user_id,action,key,job_id,input_sha256,receipt) VALUES('44444444-4444-4444-8444-444444444444','step','bad','22222222-2222-4222-8222-222222222222',repeat('a',64),'{}')`} {
		if _, e := db.Exec(q); e == nil {
			t.Fatal("accepted " + name)
		}
	}
	if _, e := db.Exec(strings.NewReplacer("$1", "'"+owner+"'", "$2", "'"+job+"'").Replace(`UPDATE managed_knowledge_publish_items SET state='published';UPDATE managed_knowledge_publish_jobs SET sequence=2,pending=0,published=1;INSERT INTO managed_knowledge_publish_receipts(owner_user_id,action,key,job_id,input_sha256,receipt) VALUES($1,'step','done',$2,repeat('a',64),'{}')`)); e != nil {
		t.Fatal(e)
	}
	for _, q := range []string{"UPDATE managed_knowledge_publish_items SET state='pending'", "DELETE FROM managed_knowledge_publish_receipts"} {
		if _, e := db.Exec(q); e == nil {
			t.Fatal("accepted immutable change")
		}
	}
	if e := goose.DownContext(ctx, db, "../../../db/migrations"); e == nil {
		t.Fatal("used down allowed")
	}
	if _, e := db.Exec("ALTER TABLE managed_knowledge_publish_items DISABLE TRIGGER managed_publish_item_guard"); e != nil {
		t.Fatal(e)
	}
	tx, e := db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	if e = publishSchemaIntegrity(ctx, tx); e == nil {
		t.Fatal("disabled trigger accepted")
	}
}
func TestPublishMigrationMissingKeepsOriginal(t *testing.T) {
	db := reviewSchemaDB(t)
	tx, e := db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	if publishSchemaIntegrity(context.Background(), tx) == nil {
		t.Fatal("missing migration accepted")
	}
	if e = managedSchemaIntegrity(context.Background(), tx); e != nil {
		t.Fatal(e)
	}
	if e = studyReviewSchemaIntegrity(context.Background(), tx); e != nil {
		t.Fatal(e)
	}
}

func TestPublishMigration16AfterReviewRestore(t *testing.T) {
	db := reviewSchemaDB(t)
	sharingSchemaSeed(t, db)
	if e := goose.UpContext(context.Background(), db, "../../../db/migrations"); e != nil {
		t.Fatal(e)
	}
	var body string
	if e := db.QueryRow("SELECT body FROM managed_study_notes").Scan(&body); e != nil || body != "自己整理的数学笔记" {
		t.Fatal(body, e)
	}
	tx, e := db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	if e = studyReviewSchemaIntegrity(context.Background(), tx); e != nil {
		t.Fatal(e)
	}
	if e = publishSchemaIntegrity(context.Background(), tx); e != nil {
		t.Fatal(e)
	}
}
