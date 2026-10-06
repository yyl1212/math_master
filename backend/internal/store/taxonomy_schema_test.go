package store_test

import (
	"context"
	"github.com/pressly/goose/v3"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"os"
	"testing"
)

func TestTaxonomySchemaEmptyRoundTrip(t *testing.T) {
	db := testutil.Database(t)
	ctx := context.Background()
	if e := store.Up(ctx, db, "../../../db/migrations"); e != nil {
		t.Fatal(e)
	}
	p, e := goose.NewProvider(goose.DialectPostgres, db, os.DirFS("../../../db/migrations"))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = p.DownTo(ctx, 9); e != nil {
		t.Fatal(e)
	}
	if e = store.Up(ctx, db, "../../../db/migrations"); e != nil {
		t.Fatal(e)
	}
}
func TestTaxonomySchemaNonemptyDownDenied(t *testing.T) {
	db := testutil.Database(t)
	ctx := context.Background()
	if e := store.Up(ctx, db, "../../../db/migrations"); e != nil {
		t.Fatal(e)
	}
	if _, e := store.New(db).InstallTaxonomyBatch(ctx, taxonomyFixtureBatch()); e != nil {
		t.Fatal(e)
	}
	p, e := goose.NewProvider(goose.DialectPostgres, db, os.DirFS("../../../db/migrations"))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = p.DownTo(ctx, 9); e == nil {
		t.Fatal("nonempty taxonomy dropped")
	}
	var count int
	if e = db.QueryRow("SELECT count(*) FROM taxonomy_nodes").Scan(&count); e != nil || count != 6603 {
		t.Fatal(count, e)
	}
}
