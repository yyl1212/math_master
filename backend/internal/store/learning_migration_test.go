package store

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"testing"
)

func TestLearningSchemaMigrationRoundTrip(t *testing.T) {
	db := testutil.Database(t)
	ctx := context.Background()
	if err := Up(ctx, db, "../../../db/migrations"); err != nil {
		var pg *pgconn.PgError
		if errors.As(err, &pg) {
			t.Logf("migration syntax: position=%d internalPosition=%d where=%s query=%s", pg.Position, pg.InternalPosition, pg.Where, pg.InternalQuery)
		}
		t.Fatal(err)
	}
	if err := Down(ctx, db, "../../../db/migrations"); err != nil {
		t.Fatal("migration Down failed", err)
	}
	var n int
	if err := db.QueryRow(`SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_name<>'goose_db_version'`).Scan(&n); err != nil || n != 0 {
		t.Fatal("round trip left tables", n, err)
	}
}
