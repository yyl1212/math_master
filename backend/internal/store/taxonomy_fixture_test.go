package store_test

import (
	"context"

	"errors"

	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"github.com/yyl1212/math_master/backend/internal/testutil"

	"testing"
)

func taxonomyFixtureBatch() taxonomy.CapturedBatch { return testutil.TaxonomyBatch() }
func TestTaxonomyInstallRemainsPrivate(t *testing.T) {
	db := testutil.Database(t)
	ctx := context.Background()
	if e := store.Up(ctx, db, "../../../db/migrations"); e != nil {
		t.Fatal(e)
	}
	s := store.New(db)
	batch := taxonomyFixtureBatch()
	v, e := s.InstallTaxonomyBatch(ctx, batch)
	if e != nil {
		t.Fatal(e)
	}
	if v.ID != batch.Manifest.SnapshotID {
		t.Fatal("identity mismatch")
	}
	if _, e = s.ListTopics(ctx, taxonomy.Query{Level: 1}); !errors.Is(e, taxonomy.ErrNotConfigured) {
		t.Fatal("draft classification was public", e)
	}
	v2, e := s.InstallTaxonomyBatch(ctx, batch)
	if e != nil || v2 != v {
		t.Fatal("install not idempotent", e)
	}
	var count int
	if e = db.QueryRow("SELECT count(*) FROM taxonomy_nodes").Scan(&count); e != nil || count != 6603 {
		t.Fatal(count, e)
	}
}
