package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/pressly/goose/v3"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"os"
	"strings"
	"testing"
)

func TestCorrectionCLIArgumentsAndSafeErrors(t *testing.T) {
	for _, a := range [][]string{{"--batches=0"}, {"--batches=11"}, {"--limit=0"}, {"--limit=51"}, {"--loop"}, {"extra"}, {"--batches=secret"}} {
		var out, err bytes.Buffer
		if n := RunCorrection(context.Background(), a, &out, &err); n != 2 || out.Len() != 0 || !json.Valid(err.Bytes()) || strings.Contains(err.String(), "secret") {
			t.Fatal("unsafe arguments", n)
		}
	}
	t.Setenv("DATABASE_URL", "postgres://user:dsn-secret@broken.invalid/db")
	t.Setenv("CORRECTION_WORKER_ENABLED", "invalid")
	var out, err bytes.Buffer
	if n := RunCorrection(context.Background(), nil, &out, &err); n != 1 || strings.Contains(err.String(), "dsn-secret") {
		t.Fatal(n)
	}
}
func TestCorrectionCLIRandomDatabaseFiniteBatch(t *testing.T) {
	db := testutil.Database(t)
	ctx := context.Background()
	if e := store.Up(ctx, db, "../../../db/migrations"); e != nil {
		t.Fatal(e)
	}
	setURL(t, db)
	t.Setenv("APP_ENV", "development")
	t.Setenv("HTTP_ADDR", "")
	t.Setenv("AUTH_PUBLIC_ORIGIN", "")
	t.Setenv("SHUTDOWN_TIMEOUT", "")
	t.Setenv("CORRECTION_WORKER_ENABLED", "false")
	var out, err bytes.Buffer
	if n := RunCorrection(ctx, nil, &out, &err); n != 0 {
		t.Fatal(n, err.String())
	}
	var v struct {
		Batches   int `json:"batches"`
		Claimed   int `json:"claimed"`
		Processed int `json:"processed"`
	}
	if json.Unmarshal(out.Bytes(), &v) != nil || v.Batches != 1 || v.Claimed != 0 || v.Processed != 0 {
		t.Fatal(out.String())
	}
}
func TestCorrectionCLINeverEnabledIsExplicit(t *testing.T) {
	db := testutil.Database(t)
	ctx := context.Background()
	p, e := goose.NewProvider(goose.DialectPostgres, db, os.DirFS("../../../db/migrations"))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = p.UpTo(ctx, 7); e != nil {
		t.Fatal(e)
	}
	setURL(t, db)
	t.Setenv("APP_ENV", "development")
	t.Setenv("AUTH_PUBLIC_ORIGIN", "")
	t.Setenv("CORRECTION_WORKER_ENABLED", "false")
	var out, err bytes.Buffer
	if n := RunCorrection(ctx, nil, &out, &err); n != 1 || !strings.Contains(err.String(), "CORRECTION_NOT_CONFIGURED") {
		t.Fatal(n, err.String())
	}
}
