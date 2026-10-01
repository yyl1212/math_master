package testutil

import (
	"context"
	"crypto/rand"
	"database/sql"
	"fmt"
	_ "github.com/jackc/pgx/v5/stdlib"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"
)

func Database(t *testing.T) *sql.DB {
	t.Helper()
	raw := os.Getenv("TEST_DATABASE_URL")
	u, e := url.Parse(raw)
	if e != nil || raw == "" || !regexp.MustCompile(`^/math_master_test_[a-z0-9_]+$`).MatchString(u.Path) || (u.Scheme != "postgres" && u.Scheme != "postgresql") {
		t.Fatal("TEST_DATABASE_URL must name math_master_test_*; tests never skip")
	}
	admin := *u
	admin.Path = "/postgres"
	a, e := sql.Open("pgx", admin.String())
	if e != nil {
		t.Fatal("test database configuration failed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	b := make([]byte, 8)
	if _, e = rand.Read(b); e != nil {
		t.Fatal(e)
	}
	name := fmt.Sprintf("math_master_test_%x", b)
	if _, e = a.ExecContext(ctx, `CREATE DATABASE "`+name+`"`); e != nil {
		a.Close()
		t.Fatal("isolated test database creation failed")
	}
	u.Path = "/" + name
	db, e := sql.Open("pgx", u.String())
	if e != nil {
		t.Fatal("isolated test connection failed")
	}
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
