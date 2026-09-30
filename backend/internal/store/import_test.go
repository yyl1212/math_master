package store_test

import (
	"context"
	"database/sql"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"os"
	"sync"
	"testing"
	"time"
)

func setup(t *testing.T) (*sql.DB, *store.Store, context.Context) {
	t.Helper()
	db := testutil.Database(t)
	ctx, c := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(c)
	if e := store.Up(ctx, db, "../../../db/migrations"); e != nil {
		t.Fatal(e)
	}
	var count int
	if e := db.QueryRowContext(ctx, "SELECT count(*) FROM imported_packages").Scan(&count); e != nil {
		t.Fatal("migration did not create schema")
	}
	return db, store.New(db), ctx
}
func input(t *testing.T, change func(*content.Package)) content.ValidatedPackage {
	t.Helper()
	cf, pf, ar := testutil.Seed(t)
	f, e := os.Open(cf)
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	c, e := content.DecodeCatalogue(f)
	if e != nil {
		t.Fatal(e)
	}
	f2, e := os.Open(pf)
	if e != nil {
		t.Fatal(e)
	}
	defer f2.Close()
	p, e := content.DecodePackage(f2)
	if e != nil {
		t.Fatal(e)
	}
	if change != nil {
		change(&p)
	}
	v, r := content.ValidateAndSeal(c, p, ar)
	if len(r.Errors) > 0 {
		t.Fatal(r.Errors)
	}
	return v
}
func TestImportIsIdempotent(t *testing.T) {
	db, s, ctx := setup(t)
	v := input(t, nil)
	a, e := s.ImportDraft(ctx, v)
	if e != nil || a.AlreadyImported {
		t.Fatal(a, e)
	}
	b, e := s.ImportDraft(ctx, v)
	if e != nil || !b.AlreadyImported || a.SHA256 != b.SHA256 {
		t.Fatal(b, e)
	}
	var n int
	db.QueryRowContext(ctx, "SELECT count(*) FROM imported_packages").Scan(&n)
	if n != 1 {
		t.Fatal(n)
	}
	db.QueryRowContext(ctx, "SELECT count(*) FROM publication_heads").Scan(&n)
	if n != 0 {
		t.Fatal("draft leaked to publication head")
	}
	if _, e = s.ImportDraft(ctx, content.ValidatedPackage{}); !errors.Is(e, store.ErrInvalidPackage) {
		t.Fatal("zero package accepted")
	}
}
func TestVersionCannotBeOverwritten(t *testing.T) {
	db, s, ctx := setup(t)
	if _, e := s.ImportDraft(ctx, input(t, nil)); e != nil {
		t.Fatal(e)
	}
	for _, change := range []func(*content.Package){func(p *content.Package) { p.Knowledge[0].Statement = "Changed" }, func(p *content.Package) { p.ID = "different-package"; p.Knowledge[0].Statement = "Changed" }} {
		if _, e := s.ImportDraft(ctx, input(t, change)); !errors.Is(e, store.ErrImmutableConflict) {
			t.Fatal("overwrite accepted", e)
		}
	}
	if _, e := db.ExecContext(ctx, "UPDATE knowledge_versions SET body='{}'"); e == nil {
		t.Fatal("SQL overwrite accepted")
	}
}
func TestImportRollsBackLateFailure(t *testing.T) {
	db, s, ctx := setup(t)
	if _, e := db.ExecContext(ctx, "ALTER TABLE publication_snapshots ADD CONSTRAINT test_late_failure CHECK (false)"); e != nil {
		t.Fatal(e)
	}
	if _, e := s.ImportDraft(ctx, input(t, nil)); e == nil {
		t.Fatal("late failure accepted")
	}
	for _, table := range []string{"catalogue_versions", "domains", "topics", "knowledge", "knowledge_versions", "knowledge_relations", "assets", "path_versions", "unit_versions", "imported_packages", "package_members", "publication_snapshots", "publication_members", "publication_heads"} {
		var n int
		if e := db.QueryRowContext(ctx, "SELECT count(*) FROM "+table).Scan(&n); e != nil || n != 0 {
			t.Fatalf("%s: %d %v", table, n, e)
		}
	}
}
func TestConcurrentSamePackageImport(t *testing.T) {
	db, s, ctx := setup(t)
	v := input(t, nil)
	var wg sync.WaitGroup
	results := make(chan store.ImportResult, 4)
	errs := make(chan error, 4)
	for range 4 {
		wg.Add(1)
		go func() { defer wg.Done(); r, e := s.ImportDraft(ctx, v); results <- r; errs <- e }()
	}
	wg.Wait()
	close(results)
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	fresh := 0
	for r := range results {
		if !r.AlreadyImported {
			fresh++
		}
	}
	if fresh != 1 {
		t.Fatal(fresh)
	}
	var n int
	db.QueryRowContext(ctx, "SELECT count(*) FROM knowledge_versions").Scan(&n)
	if n != 10 {
		t.Fatal(n)
	}
}
func TestMigrationRoundTrip(t *testing.T) {
	db := testutil.Database(t)
	ctx, c := context.WithTimeout(context.Background(), 30*time.Second)
	defer c()
	for _, f := range []func(context.Context, *sql.DB, string) error{store.Up, store.Up, store.Down, store.Up} {
		if e := f(ctx, db, "../../../db/migrations"); e != nil {
			t.Fatal(e)
		}
	}
	var n int
	if e := db.QueryRowContext(ctx, "SELECT count(*) FROM publication_heads").Scan(&n); e != nil || n != 0 {
		t.Fatal("migration round trip failed")
	}
}
