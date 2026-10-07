package store

import (
	"context"
	"github.com/pressly/goose/v3"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"os"
	"testing"
)

func TestStudyMigrationSchemaEmptyRoundTripAndPermanentMarker(t *testing.T) {
	db := testutil.Database(t)
	ctx := context.Background()
	if e := Up(ctx, db, "../../../db/migrations"); e != nil {
		t.Fatal(e)
	}
	p, e := goose.NewProvider(goose.DialectPostgres, db, os.DirFS("../../../db/migrations"))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = p.DownTo(ctx, 11); e != nil {
		t.Fatal(e)
	}
	var ever bool
	if e = db.QueryRow("SELECT topic_cutover_enabled FROM goose_db_version WHERE version_id=0").Scan(&ever); e != nil || !ever {
		t.Fatal("permanent capability lost", e)
	}
	if _, e = New(db).ReadExperienceMode(ctx); e == nil {
		t.Fatal("removed capability silently restored old mode")
	}
	if _, e = db.Exec("UPDATE goose_db_version SET topic_cutover_enabled=false WHERE version_id=0"); e == nil {
		t.Fatal("marker clear accepted")
	}
	if _, e = p.Up(ctx); e != nil {
		t.Fatal(e)
	}
	tx, e := db.BeginTx(ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	if e = cutoverConfigured(ctx, tx); e != nil {
		t.Fatal("empty Up after Down unavailable", e)
	}
}
func TestStudyMigrationSchemaNonemptyDownDenied(t *testing.T) {
	db := testutil.Database(t)
	ctx := context.Background()
	if e := Up(ctx, db, "../../../db/migrations"); e != nil {
		t.Fatal(e)
	}
	if _, e := New(db).MigrateLegacyStudyBatch(ctx, 50, nil); e != nil {
		t.Fatal(e)
	}
	p, e := goose.NewProvider(goose.DialectPostgres, db, os.DirFS("../../../db/migrations"))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = p.DownTo(ctx, 11); e == nil {
		t.Fatal("audited batch removed")
	}
	var n int
	if e = db.QueryRow("SELECT count(*) FROM study_migration_batches").Scan(&n); e != nil || n != 1 {
		t.Fatal("audit lost", e)
	}
}
