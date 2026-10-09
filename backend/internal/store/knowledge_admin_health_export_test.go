package store

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"testing"
)

func TestManagedSchemaInstalledDigest(t *testing.T) {
	db := testutil.Database(t)
	if e := Up(context.Background(), db, "../../../db/migrations"); e != nil {
		t.Fatal(e)
	}
	var digest string
	if e := db.QueryRow(managedSchemaDigestSQL, managedTables).Scan(&digest); e != nil {
		t.Fatal(e)
	}
	t.Log(digest)
	if digest != managedSchemaDigest {
		t.Fatal("installed schema differs from declared guards")
	}
}
