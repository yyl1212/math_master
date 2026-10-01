package cli

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func args() []string {
	return []string{"--catalogue", "../../../content/catalogue/domains.json", "--package", "../../../content/packages/elementary-fractions.v1.json", "--assets", "../../../content/assets"}
}
func TestCheckDoesNotRequireDatabase(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	var out, err bytes.Buffer
	if code := Run(context.Background(), "check", args(), &out, &err); code != 0 || !strings.Contains(out.String(), "reviewItems") {
		t.Fatal(code, out.String(), err.String())
	}
}
func TestImportRejectsClaimedPublication(t *testing.T) {
	b, e := os.ReadFile(args()[3])
	if e != nil {
		t.Fatal(e)
	}
	var obj map[string]any
	json.Unmarshal(b, &obj)
	obj["reviewedBy"] = "author"
	b, _ = json.Marshal(obj)
	p := filepath.Join(t.TempDir(), "bad.json")
	os.WriteFile(p, b, 0600)
	a := args()
	a[3] = p
	var out, err bytes.Buffer
	if code := Run(context.Background(), "import", a, &out, &err); code != 2 {
		t.Fatal(code, err.String())
	}
}
func TestCLIReportsRedactedFailure(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:private-test-secret@127.0.0.1:1/db?sslmode=disable&connect_timeout=1")
	var out, err bytes.Buffer
	if code := Run(context.Background(), "import", args(), &out, &err); code != 1 || strings.Contains(out.String()+err.String(), "private-test-secret") {
		t.Fatal("failure was not redacted")
	}
}
func setURL(t *testing.T, db *sql.DB) {
	t.Helper()
	var name string
	if e := db.QueryRow("SELECT current_database()").Scan(&name); e != nil {
		t.Fatal(e)
	}
	u, e := url.Parse(os.Getenv("TEST_DATABASE_URL"))
	if e != nil {
		t.Fatal("invalid test URL")
	}
	u.Path = "/" + name
	t.Setenv("DATABASE_URL", u.String())
}
func TestExportRoundTripPreservesVersionsAndAssets(t *testing.T) {
	db := testutil.Database(t)
	ctx, c := context.WithTimeout(context.Background(), 30*time.Second)
	defer c()
	if e := store.Up(ctx, db, "../../../db/migrations"); e != nil {
		t.Fatal(e)
	}
	setURL(t, db)
	var out, err bytes.Buffer
	if code := Run(ctx, "import", args(), &out, &err); code != 0 {
		t.Fatal(code, err.String())
	}
	var imported store.ImportResult
	if e := json.Unmarshal(out.Bytes(), &imported); e != nil {
		t.Fatal(e)
	}
	dir := filepath.Join(t.TempDir(), "export")
	a := []string{"--id", imported.PackageID, "--version", "1", "--catalogue-version", "1", "--out", dir}
	out.Reset()
	if code := Run(ctx, "export", a, &out, &err); code != 0 {
		t.Fatal(code, err.String())
	}
	f, e := os.Open(filepath.Join(dir, "catalogue.json"))
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	cat, e := content.DecodeCatalogue(f)
	if e != nil {
		t.Fatal(e)
	}
	f2, e := os.Open(filepath.Join(dir, "package.json"))
	if e != nil {
		t.Fatal(e)
	}
	defer f2.Close()
	p, e := content.DecodePackage(f2)
	if e != nil {
		t.Fatal(e)
	}
	v, r := content.ValidateAndSeal(cat, p, filepath.Join(dir, "assets"))
	if len(r.Errors) > 0 || v.SHA256() != imported.SHA256 {
		t.Fatal("round trip changed versions/assets", r)
	}
	out.Reset()
	restore := []string{"--catalogue", filepath.Join(dir, "catalogue.json"), "--package", filepath.Join(dir, "package.json"), "--assets", filepath.Join(dir, "assets")}
	if code := Run(ctx, "import", restore, &out, &err); code != 0 {
		t.Fatal(code, err.String())
	}
	var repeat store.ImportResult
	if e := json.Unmarshal(out.Bytes(), &repeat); e != nil || !repeat.AlreadyImported || repeat.SHA256 != imported.SHA256 {
		t.Fatal("restored import changed content", e)
	}

	if code := Run(ctx, "export", a, &out, &err); code != 1 {
		t.Fatal("existing output overwritten")
	}
	a[5] = "2"
	a[7] = filepath.Join(t.TempDir(), "wrong-catalogue")
	if code := Run(ctx, "export", a, &out, &err); code != 1 {
		t.Fatal("wrong catalogue accepted")
	}
}
