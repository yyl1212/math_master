package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pressly/goose/v3"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/question"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"os"
	"strings"
	"testing"
)

func TestQuestionSchema(t *testing.T) {
	s, db, a, user := workflowGuardFixture(t)
	ctx := context.Background()
	tables := []string{"question_workspaces", "question_workspace_authors", "question_packages", "question_templates", "question_instances", "question_blueprints", "question_instance_coverage", "question_blueprint_sources", "question_submissions", "question_submission_authors", "question_submission_members", "question_review_decisions", "question_publications", "question_publication_members", "question_heads", "question_withdrawals", "question_events", "question_idempotency"}
	for _, table := range tables {
		var n int
		if err := db.QueryRow(`SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name=$1`, table).Scan(&n); err != nil || n != 1 {
			t.Fatalf("missing table %s", table)
		}
	}
	if err := s.questionReadTx(ctx, a, question.ListDraftsAction, func(ctx context.Context, tx *sql.Tx, _ auth.User) error { return questionConfigured(ctx, tx) }); err != nil {
		t.Fatal(err)
	}
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, q, args...); err != nil {
			t.Fatal(err)
		}
	}
	reject := func(q string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, q, args...); err == nil {
			t.Fatalf("constraint accepted %s", q)
		}
	}
	exec(`INSERT INTO catalogue_versions(version,sha256,body) VALUES(900,$1,'{}')`, fmt.Sprintf("%064x", 900))
	workspace := "44444444-4444-4444-8444-444444444444"
	reject(`INSERT INTO question_workspaces(id,owner_user_id,catalogue_version,package,revision) VALUES($1,$2,900,'{}',0)`, workspace, user)
	exec(`INSERT INTO question_workspaces(id,owner_user_id,catalogue_version,package,revision) VALUES($1,$2,900,'{}',1)`, workspace, user)
	exec(`INSERT INTO question_workspace_authors VALUES($1,$2)`, workspace, user)
	reject(`UPDATE question_workspaces SET status='submitted' WHERE id=$1`, workspace)
	reject(`UPDATE question_workspaces SET owner_user_id='99999999-9999-4999-8999-999999999999' WHERE id=$1`, workspace)
	p := question.QuestionPackage{Kind: "question-bank", SchemaVersion: 1, ID: "schema-test", Version: 1, Templates: []question.Template{}, FixedQuestions: []question.FixedQuestion{}, Blueprints: []question.Blueprint{}}
	bytes, sha, err := question.CanonicalPackage(p)
	if err != nil {
		t.Fatal(err)
	}
	reject(`INSERT INTO question_packages(id,version,sha256,catalogue_version,catalogue_sha256,body,body_bytes,instance_identities) VALUES('schema-test',1,$1,900,$2,$3,$4,'[]')`, fmt.Sprintf("%064x", 0), fmt.Sprintf("%064x", 900), string(bytes), bytes)
	pkgTx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer pkgTx.Rollback()
	if _, err = pkgTx.Exec(`INSERT INTO question_packages(id,version,sha256,catalogue_version,catalogue_sha256,body,body_bytes,instance_identities,legacy_unattributed) VALUES('schema-test',1,$1,900,$2,$3,$4,'[]',false)`, sha, fmt.Sprintf("%064x", 900), string(bytes), bytes); err != nil {
		t.Fatal(err)
	}
	if _, err = pkgTx.Exec(`UPDATE question_packages SET sealed=true WHERE id='schema-test'`); err != nil {
		t.Fatal(err)
	}
	if err = pkgTx.Commit(); err != nil {
		t.Fatal(err)
	}
	reject(`UPDATE question_packages SET source_map='[{}]' WHERE id='schema-test'`)
	reject(`DELETE FROM question_packages WHERE id='schema-test'`)
	frozen := question.FrozenBody{CatalogueVersion: 900, CatalogueSHA256: fmt.Sprintf("%064x", 900), QuestionPackage: p, SourceMap: []question.SourceLink{}, AuthorIDs: []string{user}, Resolved: []question.KnownObject{}, Objectives: []question.ResolvedObjective{}, Generation: []question.GenerationReport{}, InstanceIdentities: []question.Identity{}, Coverage: []question.CoverageNode{}, GeneratorVersions: []int{}, VerifierVersions: []int{}}
	raw, hash, err := question.CanonicalFrozen(frozen, []question.Instance{})
	if err != nil {
		t.Fatal(err)
	}
	sub := "55555555-5555-4555-8555-555555555555"
	insert := func(tx *sql.Tx) error {
		_, err := tx.Exec(`INSERT INTO question_submissions(id,workspace_id,owner_user_id,revision,package_id,package_version,package_sha256,catalogue_version,catalogue_sha256,frozen_body,frozen_bytes,frozen_digest,gate) VALUES($1,$2,$3,1,'schema-test',1,$4,900,$5,$6,$7,$8,'{}')`, sub, workspace, user, sha, fmt.Sprintf("%064x", 900), string(raw), raw, hash)
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err = insert(tx); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err == nil {
		t.Fatal("partial submission committed")
	}
	tx, err = db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err = insert(tx); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`INSERT INTO question_submission_authors VALUES($1,$2)`, sub, user); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`UPDATE question_submissions SET sealed=true WHERE id=$1`, sub); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`UPDATE question_workspaces SET status='submitted' WHERE id=$1`, workspace); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	reject(`UPDATE question_submissions SET frozen_digest=$2 WHERE id=$1`, sub, fmt.Sprintf("%064x", 1))
	reject(`DELETE FROM question_submissions WHERE id=$1`, sub)
	questionRejectFrozen(t, db, `INSERT INTO question_submission_authors VALUES($1,$2)`, sub, user)
	questionRejectFrozen(t, db, `DELETE FROM question_submission_authors WHERE submission_id=$1`, sub)
	questionRejectFrozen(t, db, `INSERT INTO question_submission_members(submission_id,kind,id,version,sha256) VALUES($1,'instance','other',1,$2)`, sub, sha)
	exec(`INSERT INTO auth_users(id,username,password_phc) VALUES('66666666-6666-4666-8666-666666666666','schema_reviewer','isolated-test-only'); INSERT INTO auth_user_roles VALUES('66666666-6666-4666-8666-666666666666','learner'),('66666666-6666-4666-8666-666666666666','reviewer')`)
	review := `INSERT INTO question_review_decisions(id,submission_id,reviewer_user_id,frozen_digest,decision,checks,independence_note,generation_note,note) VALUES('77777777-7777-4777-8777-777777777777',$1,'66666666-6666-4666-8666-666666666666',$2,'approve','{"mathematics":true,"explanations":true,"objectives":true,"sources":true,"illustrations":true,"generation":true}','Independent technical fixture.','No templates in this fixture.','This is a technical fixture review.')`
	reject(review, sub, hash)
	tx, err = db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err = tx.Exec(review, sub, hash); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`UPDATE question_submissions SET status='approved' WHERE id=$1`, sub); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	reject(`UPDATE question_review_decisions SET note='Changed technical review.'`)
	reject(`DELETE FROM question_review_decisions`)
	testQuestionFixedRelations(t, db)
	testQuestionUpgrade(t)
}

func testQuestionUpgrade(t *testing.T) {
	db := testutil.Database(t)
	ctx := context.Background()
	provider, err := goose.NewProvider(goose.DialectPostgres, db, os.DirFS("../../../db/migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = provider.UpTo(ctx, 4); err != nil {
		t.Fatal(err)
	}
	if _, err = New(db).QuestionPreflight(ctx, question.Access{}, question.ListDraftsAction); !errors.Is(err, question.ErrNotConfigured) {
		t.Fatalf("missing schema not distinguished: %v", err)
	}
	before := map[string][32]byte{}
	for _, file := range []string{"00001_content_foundation.sql", "00002_unit_asset_bindings.sql", "00003_accounts.sql", "00004_content_workflow.sql"} {
		raw, err := os.ReadFile("../../../db/migrations/" + file)
		if err != nil {
			t.Fatal(err)
		}
		before[file] = sha256.Sum256(raw)
	}
	state := func() string {
		out := map[string]json.RawMessage{}
		for _, table := range []string{"catalogue_versions", "knowledge_versions", "unit_versions", "path_versions", "assets", "imported_packages", "package_members", "publication_snapshots", "publication_members", "publication_heads"} {
			var raw string
			if err := db.QueryRow(`SELECT coalesce(jsonb_agg(to_jsonb(t) ORDER BY to_jsonb(t)::text),'[]')::text FROM ` + table + ` t`).Scan(&raw); err != nil {
				t.Fatal(err)
			}
			out[table] = json.RawMessage(raw)
		}
		raw, _ := json.Marshal(out)
		return string(raw)
	}
	// Seed old fixed data before the upgrade to prove preservation, not just empty tables.
	if _, err = db.Exec(`INSERT INTO catalogue_versions VALUES(901,repeat('a',64),'{"fixture":"old bytes"}');INSERT INTO knowledge VALUES('legacy');INSERT INTO knowledge_versions VALUES('legacy',1,repeat('b',64),'{"title":"Legacy fixed knowledge"}')`); err != nil {
		t.Fatal(err)
	}
	snapshot := state()
	if _, err = provider.Up(ctx); err != nil {
		t.Fatal(err)
	}
	if snapshot != state() {
		t.Fatal("old data changed")
	}
	for file, want := range before {
		raw, err := os.ReadFile("../../../db/migrations/" + file)
		if err != nil || sha256.Sum256(raw) != want {
			t.Fatal("old migration changed")
		}
	}
}

func testQuestionFixedRelations(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	exec := func(q string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, q, args...); err != nil {
			t.Fatal(err)
		}
	}
	reject := func(q string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, q, args...); err == nil {
			t.Fatalf("fixed constraint accepted %s", q)
		}
	}
	exec(`INSERT INTO knowledge VALUES('schema-knowledge')`)
	exec(`INSERT INTO knowledge_versions VALUES('schema-knowledge',1,repeat('c',64),'{"objectives":["Add exact fractions"]}')`)
	ref := question.Ref{ID: "schema-knowledge", Version: 1}
	format := "rational"
	tmpl := question.Template{ID: "schema-template", Version: 1, Knowledge: ref, Coverage: []question.ObjectiveCoverage{{Knowledge: ref, ObjectiveIndices: []int{0}}}, Units: []question.Ref{}, Type: "numeric", AnswerFormat: &format, PromptTemplate: "Add {{left}} and {{right}}.", ExplanationTemplate: "Adding both operands gives {{answer}}.", Engine: question.EngineSpec{Family: "rational_arithmetic", Operation: "add", GeneratorVersion: 1, VerifierVersion: 1}, Parameters: []question.Parameter{{Name: "left", Values: []string{"1/1"}}, {Name: "right", Values: []string{"2/1"}}}, Constraints: []question.Constraint{}, Distractors: []question.Distractor{}, Assets: []question.AssetRef{}, Sources: []content.Source{{Title: strings.Repeat("A", 35000), URL: "https://example.com/technical-source", License: "Original technical fixture"}}}
	instances, report, err := question.Generate(ctx, tmpl)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(struct {
		Purpose string            `json:"purpose"`
		Body    question.Template `json:"body"`
	}{"question-template-v1", tmpl})
	sha := fmt.Sprintf("%x", sha256.Sum256(raw))
	if sha != report.Template.SHA256 {
		t.Fatal("test template canonical mismatch")
	}
	badTemplate := []byte(`{"purpose":"question-template-v1","body":{}}`)
	reject(`INSERT INTO question_templates(id,version,sha256,body,body_bytes,knowledge_id,knowledge_version) VALUES('schema-malformed',1,$1,$2,$3,'schema-knowledge',1)`, fmt.Sprintf("%x", sha256.Sum256(badTemplate)), string(badTemplate), badTemplate)
	exec(`INSERT INTO question_templates(id,version,sha256,body,body_bytes,knowledge_id,knowledge_version) VALUES('schema-template',1,$1,$2,$3,'schema-knowledge',1)`, sha, string(raw), raw)
	reject(`UPDATE question_templates SET body='{}'`)
	reject(`DELETE FROM question_templates`)
	instance := instances[0]
	insert := func(tx *sql.Tx, i question.Instance, parameterSHA string) error {
		body, hash, err := question.CanonicalInstance(i)
		if err != nil {
			return err
		}
		params, _ := json.Marshal(i.Parameters)
		_, err = tx.Exec(`INSERT INTO question_instances(id,version,sha256,body,body_bytes,origin,knowledge_id,knowledge_version,template_id,template_version,template_sha256,generator_version,verifier_version,parameters,parameter_bytes,parameter_sha256) VALUES($1,1,$2,$3,$4,'template','schema-knowledge',1,'schema-template',1,$5,1,1,$6,$7,$8)`, i.Identity.ID, hash, string(body), body, sha, string(params), params, parameterSHA)
		return err
	}
	params, _ := json.Marshal(instance.Parameters)
	parameterSHA := fmt.Sprintf("%x", sha256.Sum256(params))
	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err = insert(tx, instance, strings.Repeat("d", 64)); err == nil {
		t.Fatal("bad parameter digest accepted")
	}
	tx.Rollback()
	tx, err = db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err = insert(tx, instance, parameterSHA); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`UPDATE question_instances SET sealed=true`); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err == nil {
		t.Fatal("half coverage committed")
	}
	tx, err = db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err = insert(tx, instance, parameterSHA); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`INSERT INTO question_instance_coverage VALUES($1,1,'schema-knowledge',1,0)`, instance.Identity.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`UPDATE question_instances SET sealed=true`); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	questionRejectFrozen(t, db, `INSERT INTO question_instance_coverage VALUES($1,1,'schema-knowledge',1,1)`, instance.Identity.ID)
	questionRejectFrozen(t, db, `DELETE FROM question_instance_coverage`)
	reject(`UPDATE question_instances SET body='{}'`)
	reject(`DELETE FROM question_instances`)
	duplicate := instance
	duplicate.Identity.ID = "qi-" + strings.Repeat("e", 64)
	tx, err = db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err = insert(tx, duplicate, parameterSHA); err == nil {
		t.Fatal("same template parameters counted twice")
	}
	tx.Rollback()
	blueprint := question.Blueprint{ID: "schema-blueprint", Version: 1, Knowledge: ref, CoreObjectiveIndices: []int{0}, Sources: []question.BlueprintSource{{Kind: "template", Ref: question.Ref{ID: tmpl.ID, Version: 1}}}, CoverageNote: "Technical fixed source fixture.", RuleVersion: 1, QuestionCount: 5, PassCount: 4}
	bpRaw, _ := json.Marshal(struct {
		Purpose string             `json:"purpose"`
		Body    question.Blueprint `json:"body"`
	}{"question-blueprint-v1", blueprint})
	bpSHA := fmt.Sprintf("%x", sha256.Sum256(bpRaw))
	tx, err = db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err = tx.Exec(`INSERT INTO question_blueprints(id,version,sha256,body,body_bytes,knowledge_id,knowledge_version) VALUES('schema-blueprint',1,$1,$2,$3,'schema-knowledge',1)`, bpSHA, string(bpRaw), bpRaw); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`INSERT INTO question_blueprint_sources(blueprint_id,blueprint_version,kind,id,version,template_id) VALUES('schema-blueprint',1,'template','schema-template',1,'schema-template')`); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`UPDATE question_blueprints SET sealed=true`); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	questionRejectFrozen(t, db, `INSERT INTO question_blueprint_sources(blueprint_id,blueprint_version,kind,id,version,template_id) VALUES('schema-blueprint',1,'template','schema-template',1,'schema-template')`)
	reject(`DELETE FROM question_blueprint_sources`)
	reject(`UPDATE question_blueprint_sources SET version=2`)
}

func questionRejectFrozen(t *testing.T, db *sql.DB, q string, args ...any) {
	t.Helper()
	_, err := db.Exec(q, args...)
	var pg *pgconn.PgError
	if !errors.As(err, &pg) || pg.Code != "P0001" {
		t.Fatalf("expected immutable trigger, got %v", err)
	}
}
