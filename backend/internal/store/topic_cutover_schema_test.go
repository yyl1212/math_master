package store

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"reflect"
	"testing"
)

func TestTopicCutoverSchemaGuardCatalog(t *testing.T) {
	db := testutil.Database(t)
	ctx := context.Background()
	if e := Up(ctx, db, "../../../db/migrations"); e != nil {
		t.Fatal(e)
	}
	rows, e := db.Query(`SELECT c.conrelid::regclass::text||'/'||c.conname,c.contype::text||':'||md5(pg_get_constraintdef(c.oid)) FROM pg_constraint c WHERE c.conrelid IN (SELECT oid FROM pg_class WHERE relnamespace='public'::regnamespace AND relname=ANY($1::text[])) ORDER BY 1`, append(append(append([]string{}, taxonomyTables...), studyTables...), cutoverTables...))
	if e != nil {
		t.Fatal(e)
	}
	defer rows.Close()
	items := map[string]string{}
	for rows.Next() {
		var k, v string
		if e = rows.Scan(&k, &v); e != nil {
			t.Fatal(e)
		}
		items[k] = v
	}
	if e = rows.Err(); e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(items, topicSchemaConstraintGuards) {
		t.Fatal("installed schema differs from declared integrity guards")
	}
}
