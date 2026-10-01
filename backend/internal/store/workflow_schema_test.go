package store_test

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
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"os"
	"strings"
	"testing"
	"time"
)

func TestWorkflowSchema(t *testing.T) {
	f := newAuthFixture(t)
	tables := []string{"content_workspaces", "content_workspace_assets", "content_submissions", "content_submission_authors", "content_submission_members", "content_review_decisions", "content_publication_manifests", "content_withdrawals", "content_workflow_events", "content_idempotency"}
	for _, name := range tables {
		if f.count("SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name=$1", name) != 1 {
			t.Fatalf("workflow table missing: %s", name)
		}
	}
	user := f.register("schema_editor")
	f.exec("INSERT INTO catalogue_versions(version,sha256,body) VALUES(900,$1,'{}')", string(makeHex('a', 64)))
	id := "11111111-1111-4111-8111-111111111111"

	rejected := func(code, query string, args ...any) {
		t.Helper()
		_, err := f.db.ExecContext(f.ctx, query, args...)
		var pg *pgconn.PgError
		if !errors.As(err, &pg) || pg.Code != code {
			t.Fatalf("expected SQL constraint %s, got %v", code, err)
		}
	}
	rejected("23503", `INSERT INTO content_workspaces(id,owner_user_id,catalogue_version,package,revision) VALUES($1,'22222222-2222-4222-8222-222222222222',900,'{}',1)`, id)
	rejected("23514", `INSERT INTO content_workspaces(id,owner_user_id,catalogue_version,package,revision) VALUES($1,$2,900,'{}',0)`, id, user.ID)
	f.exec(`INSERT INTO content_workspaces(id,owner_user_id,catalogue_version,package,revision) VALUES($1,$2,900,'{}',1)`, id, user.ID)
	large := make([]byte, 1048577)
	hash := fmt.Sprintf("%x", sha256.Sum256(large))
	rejected("23514", `INSERT INTO content_workspace_assets(workspace_id,asset_id,sha256,bytes) VALUES($1,'asset-test',$2,$3)`, id, hash, large)
	reviewer := f.register("schema_reviewer")
	v := input(t, nil)
	if _, err := f.repo.ImportDraft(f.ctx, v); err != nil {
		t.Fatal(err)
	}
	pkg := v.Package()
	frozen := publication.FrozenBody{CatalogueVersion: v.Catalogue().Version, CatalogueSHA256: v.CatalogueSHA256(), Package: pkg, SourceMap: []publication.SourceLink{}, AuthorIDs: []string{user.ID}, Assets: []content.AssetView{}}
	payload, err := json.Marshal(frozen)
	if err != nil {
		t.Fatal(err)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(payload))
	submission := "33333333-3333-4333-8333-333333333333"
	insert := func(tx *sql.Tx, subID string, revision int64) error {
		_, err := tx.ExecContext(f.ctx, `INSERT INTO content_submissions(id,workspace_id,owner_user_id,revision,package_id,package_version,package_sha256,catalogue_version,catalogue_sha256,frozen_body,frozen_bytes,frozen_digest,gate) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,'{}')`, subID, id, user.ID, revision, pkg.ID, pkg.Version, v.SHA256(), v.Catalogue().Version, v.CatalogueSHA256(), string(payload), payload, digest)
		return err
	}
	tx, err := f.db.BeginTx(f.ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = insert(tx, submission, 1); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err == nil {
		t.Fatal("unsealed submission committed")
	}
	tx, err = f.db.BeginTx(f.ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err = insert(tx, submission, 1); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.ExecContext(f.ctx, `INSERT INTO content_submission_authors VALUES($1,$2)`, submission, user.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.ExecContext(f.ctx, `INSERT INTO content_submission_members SELECT $1,m.package_id,m.package_version,m.kind,m.id,m.version,CASE m.kind WHEN 'knowledge' THEN k.sha256 WHEN 'unit' THEN u.sha256 WHEN 'path' THEN p.sha256 ELSE m.asset_sha256 END FROM package_members m LEFT JOIN knowledge_versions k ON m.kind='knowledge' AND k.id=m.id AND k.version=m.version LEFT JOIN unit_versions u ON m.kind='unit' AND u.id=m.id AND u.version=m.version LEFT JOIN path_versions p ON m.kind='path' AND p.id=m.id AND p.version=m.version WHERE m.package_id=$2 AND m.package_version=$3`, submission, pkg.ID, pkg.Version); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.ExecContext(f.ctx, `UPDATE content_submissions SET sealed=true WHERE id=$1`, submission); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	rejected("P0001", `UPDATE content_submissions SET frozen_body=jsonb_set(frozen_body,'{sourceMap}','[{}]') WHERE id=$1`, submission)
	rejected("P0001", `INSERT INTO content_submission_authors VALUES($1,$2)`, submission, reviewer.ID)
	rejected("P0001", `DELETE FROM content_submission_authors WHERE submission_id=$1`, submission)
	rejected("P0001", `UPDATE content_submission_members SET sha256=$2 WHERE submission_id=$1`, submission, strings.Repeat("b", 64))
	rejected("P0001", `INSERT INTO content_submission_members SELECT * FROM content_submission_members WHERE submission_id=$1`, submission)
	reviewQuery := `INSERT INTO content_review_decisions(id,submission_id,reviewer_user_id,frozen_digest,decision,checks,independence_note,note) VALUES($1,$2,$3,$4,'approve','{"mathematics":true,"explanations":true,"relationships":true,"sources":true,"illustrations":true}','An independent technical test review.','All technical fixture conditions verified.')`
	// A decision and its terminal parent state must commit together.
	tx, err = f.db.BeginTx(f.ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.ExecContext(f.ctx, reviewQuery, "44444444-4444-4444-8444-444444444444", submission, reviewer.ID, digest); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err == nil {
		t.Fatal("review without terminal parent state committed")
	}
	tx, err = f.db.BeginTx(f.ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(f.ctx, reviewQuery, "44444444-4444-4444-8444-444444444444", submission, reviewer.ID, digest); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.ExecContext(f.ctx, `SAVEPOINT duplicate_review`); err != nil {
		t.Fatal(err)
	}
	_, err = tx.ExecContext(f.ctx, reviewQuery, "55555555-5555-4555-8555-555555555555", submission, reviewer.ID, digest)
	var pg *pgconn.PgError
	if !errors.As(err, &pg) || pg.Code != "23505" {
		t.Fatalf("second review not uniquely rejected: %v", err)
	}
	if _, err = tx.ExecContext(f.ctx, `ROLLBACK TO SAVEPOINT duplicate_review`); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.ExecContext(f.ctx, `UPDATE content_submissions SET status='approved' WHERE id=$1`, submission); err != nil {
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	rejected("P0001", `UPDATE content_submissions SET status='returned' WHERE id=$1`, submission)
	f.exec(`INSERT INTO publication_snapshots VALUES('schema-manifest', $1,'draft')`, v.Catalogue().Version)
	f.exec(`INSERT INTO publication_members SELECT 'schema-manifest',package_id,package_version,kind,id,version,'active' FROM package_members WHERE package_id=$1 AND package_version=$2`, pkg.ID, pkg.Version)
	manifest := []byte(`{}`)
	manifestSHA := fmt.Sprintf("%x", sha256.Sum256(manifest))
	f.exec(`INSERT INTO content_publication_manifests(snapshot_id,manifest,manifest_bytes,sha256,diff,creator_user_id) VALUES('schema-manifest','{}',$1,$2,'{}',$3)`, manifest, manifestSHA, user.ID)
	rejected("P0001", `INSERT INTO publication_members SELECT 'schema-manifest',package_id,package_version,kind,id,version,'active' FROM package_members WHERE package_id=$1 AND package_version=$2`, pkg.ID, pkg.Version)
	rejected("P0001", `UPDATE publication_members SET availability='withdrawn' WHERE snapshot_id='schema-manifest'`)
	rejected("P0001", `DELETE FROM publication_members WHERE snapshot_id='schema-manifest'`)
	t.Run("upgrade preserves content", testWorkflowUpgrade)

}
func makeHex(c byte, n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = c
	}
	return b
}

func TestContentRateBudget(t *testing.T) {
	for _, tc := range []struct {
		action publication.Action
		limit  int
	}{
		{publication.ReadDraftAction, 120}, {publication.SaveDraftAction, 30}, {publication.SubmitDraftAction, 10},
	} {
		f, _ := fixedRateFixture(t, time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC))
		keys, err := publication.Rates("11111111-1111-4111-8111-111111111111", tc.action)
		if err != nil {
			t.Fatal("invalid content rate policy")
		}
		for i := 0; i < tc.limit; i++ {
			if f.repo.ConsumeRates(f.ctx, keys) != nil {
				t.Fatal("content quota rejected early")
			}
		}
		var limited *auth.RateLimitError
		if !errors.As(f.repo.ConsumeRates(f.ctx, keys), &limited) {
			t.Fatal("content actor quota exceeded")
		}
	}
	for _, tc := range []struct {
		action         publication.Action
		limit, perUser int
	}{
		{publication.SaveDraftAction, 120, 30}, {publication.SubmitDraftAction, 40, 10},
	} {
		f, _ := fixedRateFixture(t, time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC))
		for i := 0; i < tc.limit; i++ {
			actor := []string{"11111111-1111-4111-8111-111111111111", "22222222-2222-4222-8222-222222222222", "33333333-3333-4333-8333-333333333333", "44444444-4444-4444-8444-444444444444"}[i/tc.perUser]
			keys, _ := publication.Rates(actor, tc.action)
			if f.repo.ConsumeRates(f.ctx, keys) != nil {
				t.Fatal("global content quota rejected early")
			}
		}
		keys, _ := publication.Rates("55555555-5555-4555-8555-555555555555", tc.action)
		var limited *auth.RateLimitError
		if !errors.As(f.repo.ConsumeRates(f.ctx, keys), &limited) {
			t.Fatal("global content quota exceeded")
		}
	}
}

func testWorkflowUpgrade(t *testing.T) {
	db := testutil.Database(t)
	ctx := context.Background()
	provider, err := goose.NewProvider(goose.DialectPostgres, db, os.DirFS("../../../db/migrations"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err = provider.UpTo(ctx, 3); err != nil {
		t.Fatal(err)
	}
	if _, err = store.New(db).ImportDraft(ctx, input(t, nil)); err != nil {
		t.Fatal(err)
	}
	state := func() string {
		out := ""
		for _, table := range []string{"catalogue_versions", "knowledge_versions", "unit_versions", "path_versions", "assets", "imported_packages", "package_members", "unit_asset_bindings", "publication_snapshots", "publication_members", "publication_heads"} {
			var body string
			err := db.QueryRowContext(ctx, `SELECT COALESCE(jsonb_agg(row_body ORDER BY row_body::text),'[]'::jsonb)::text FROM (SELECT to_jsonb(t) AS row_body FROM `+table+` t) rows`).Scan(&body)
			if err != nil {
				t.Fatal(err)
			}
			out += table + body
		}
		return out
	}
	before := state()
	if _, err = provider.Up(ctx); err != nil {
		t.Fatal(err)
	}
	if before != state() {
		t.Fatal("legacy content counts, identities, or bytes changed during upgrade")
	}
	if f := store.Down(ctx, db, "../../../db/migrations"); f != nil {
		t.Fatal(f)
	}
}
