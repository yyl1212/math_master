package testutil

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func Database(t *testing.T) *sql.DB {
	t.Helper()
	raw := os.Getenv("TEST_DATABASE_URL")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	b := make([]byte, 8)
	if _, e := rand.Read(b); e != nil {
		t.Fatal(e)
	}
	name := fmt.Sprintf("math_master_test_%x", b)
	admin, work, e := IsolatedConfigs(raw, name)
	if e != nil {
		t.Fatal("unsafe isolated test database configuration")
	}
	a := OpenVerified(admin)
	if _, e = a.ExecContext(ctx, `CREATE DATABASE "`+name+`"`); e != nil {
		a.Close()
		t.Fatal("isolated test database creation failed")
	}
	db := OpenVerified(work)
	t.Cleanup(func() {
		db.Close()
		ctx, c := context.WithTimeout(context.Background(), 10*time.Second)
		defer c()
		if _, e := a.ExecContext(ctx, `DROP DATABASE "`+name+`" WITH (FORCE)`); e != nil {
			t.Error("isolated database cleanup failed")
		}
		a.Close()
	})
	return db
}
func Seed(t *testing.T) (string, string, string) {
	t.Helper()
	return "../../../content/catalogue/domains.json", "../../../content/packages/elementary-fractions.v1.json", "../../../content/assets"
}
func Guard(t *testing.T) {
	t.Helper()
	if strings.Contains(os.Getenv("TEST_DATABASE_URL"), "production") {
		t.Fatal("unsafe test database")
	}
}
