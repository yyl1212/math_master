package store_test

import (
	"context"
	"database/sql"
	"github.com/pressly/goose/v3"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"os"
	"testing"
)

func downCorrection(t *testing.T, db *sql.DB) error {
	t.Helper()
	p, e := goose.NewProvider(goose.DialectPostgres, db, os.DirFS("../../../db/migrations"))
	if e != nil {
		return e
	}
	// Exercise correction migration 00008 even when later migrations exist.
	_, e = p.DownTo(context.Background(), 7)
	return e
}
func TestCorrectionMigrationEmptyDownAndMarker(t *testing.T) {
	db := testutil.Database(t)
	if e := store.Up(context.Background(), db, "../../../db/migrations"); e != nil {
		t.Fatal(e)
	}
	if correctionTableCount(t, db) != 10 {
		t.Fatal("ten tables")
	}
	if e := downCorrection(t, db); e != nil {
		t.Fatal(e)
	}
	if correctionTableCount(t, db) != 0 {
		t.Fatal("empty Down left business tables")
	}
	var marker bool
	if e := db.QueryRow(`SELECT correction_enabled FROM goose_db_version WHERE version_id=0`).Scan(&marker); e != nil || !marker {
		t.Fatal("permanent correction marker missing", e)
	}
	for _, q := range []string{`UPDATE goose_db_version SET correction_enabled=false WHERE version_id=0`, `DELETE FROM goose_db_version WHERE version_id=0`, `UPDATE goose_db_version SET version_id=99 WHERE version_id=0`} {
		if _, e := db.Exec(q); e == nil {
			t.Fatal("marker bypass", q)
		}
	}
	if e := store.Up(context.Background(), db, "../../../db/migrations"); e != nil {
		t.Fatal(e)
	}
	if correctionTableCount(t, db) != 10 {
		t.Fatal("roundtrip failed")
	}
}
func TestCorrectionMigrationNonemptyDown(t *testing.T) {
	f := newCorrectionFixture(t)
	f.cCase()
	if e := downCorrection(t, f.db); e == nil {
		t.Fatal("nonempty correction facts erased")
	}
	if correctionTableCount(t, f.db) != 10 {
		t.Fatal("partial Down")
	}
}
func TestCorrectionMigrationQuotaAloneProtectsDown(t *testing.T) {
	f := newCorrectionFixture(t)
	f.exec(`INSERT INTO correction_rate_limits(owner_user_id,scope,command_key) VALUES($1,'create',$2)`, f.ids["admin_a"], f.ID())
	if e := downCorrection(t, f.db); e == nil {
		t.Fatal("quota-only Down allowed")
	}
}
