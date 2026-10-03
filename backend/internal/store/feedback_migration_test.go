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

func tryDownSeven(t *testing.T, db *sql.DB) error {
	t.Helper()
	p, e := goose.NewProvider(goose.DialectPostgres, db, os.DirFS("../../../db/migrations"))
	if e != nil {
		return e
	}
	_, e = p.Down(context.Background())
	return e
}
func TestFeedbackMigrationEmptyRoundTrip(t *testing.T) {
	db := testutil.Database(t)
	ctx := context.Background()
	if e := store.Up(ctx, db, "../../../db/migrations"); e != nil {
		t.Fatal(e)
	}
	if countFeedbackTables(t, db) != 4 {
		t.Fatal("four tables")
	}
	if e := tryDownSeven(t, db); e != nil {
		t.Fatal(e)
	}
	if countFeedbackTables(t, db) != 0 {
		t.Fatal("rollback")
	}
	if e := store.Up(ctx, db, "../../../db/migrations"); e != nil {
		t.Fatal(e)
	}
	if countFeedbackTables(t, db) != 4 {
		t.Fatal("roundtrip")
	}
}
func TestFeedbackMigrationNonemptyGuard(t *testing.T) {
	for _, kind := range []string{"ticket", "events", "receipt", "quota"} {
		t.Run(kind, func(t *testing.T) {
			f := newFeedbackFixture(t)
			if countFeedbackTables(t, f.db) != 4 {
				t.Fatal("four tables")
			}
			if kind == "quota" {
				f.exec(`INSERT INTO feedback_rate_limits(actor_user_id,scope,consumed_at) VALUES($1,'create',clock_timestamp())`, f.ids["learner_a"])
			} else {
				id := f.seededFeedback()
				if kind == "receipt" {
					f.seedFeedbackReceipt(id)
				}
				if kind == "events" {
					tx, _ := f.db.Begin()
					defer tx.Rollback()
					_, e := tx.Exec(`WITH e AS (INSERT INTO feedback_events(ticket_id,sequence,actor_user_id,kind,from_status,to_status,message,recorded_at,request_id) VALUES($1,2,$2,'replied','new','new','more details',clock_timestamp(),'migration-test') RETURNING recorded_at) UPDATE feedback_tickets SET sequence=2,updated_at=(SELECT recorded_at FROM e) WHERE id=$1`, id, f.ids["learner_a"])
					if e != nil {
						t.Fatal(e)
					}
					if e = tx.Commit(); e != nil {
						t.Fatal(e)
					}
				}
			}
			if e := tryDownSeven(t, f.db); e == nil {
				t.Fatal("nonempty rollback")
			}
			if countFeedbackTables(t, f.db) != 4 {
				t.Fatal("partial rollback")
			}
		})
	}
}
