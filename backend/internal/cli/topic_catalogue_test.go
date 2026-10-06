package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTopicCatalogueCLIRejectsWithoutLeakingConfiguration(t *testing.T) {
	t.Setenv("DATABASE_URL", "do-not-print-secret")
	var out, err bytes.Buffer
	if RunTopicCatalogue(context.Background(), []string{"--password", "secret"}, &out, &err) != 2 {
		t.Fatal("password argument accepted")
	}
	if strings.Contains(err.String(), "secret") {
		t.Fatal("argument leaked")
	}
	p := filepath.Join(t.TempDir(), "source.json")
	if e := os.WriteFile(p, []byte(`{}`), 0600); e != nil {
		t.Fatal(e)
	}
	out.Reset()
	err.Reset()
	if RunTopicCatalogue(context.Background(), []string{"--archive", p}, &out, &err) == 0 {
		t.Fatal("invalid archive succeeded")
	}
	if strings.Contains(err.String(), "do-not-print-secret") {
		t.Fatal("credentials leaked")
	}
}
func TestTopicCatalogueCLIImportsOnlyDraft(t *testing.T) {
	db := testutil.Database(t)
	ctx := context.Background()
	if e := store.Up(ctx, db, "../../../db/migrations"); e != nil {
		t.Fatal(e)
	}
	setURL(t, db)
	raw, _ := json.Marshal(testutil.TaxonomyBatch())
	p := filepath.Join(t.TempDir(), "taxonomy.json")
	if e := os.WriteFile(p, raw, 0600); e != nil {
		t.Fatal(e)
	}
	var out, err bytes.Buffer
	if code := RunTopicCatalogue(ctx, []string{"--archive", p}, &out, &err); code != 0 {
		t.Fatal(code, err.String())
	}
	var reply struct {
		Status    string
		Published bool
	}
	if e := json.Unmarshal(out.Bytes(), &reply); e != nil || reply.Status != "draft" || reply.Published {
		t.Fatal(reply, e)
	}
	var count int
	if e := db.QueryRow("SELECT count(*) FROM taxonomy_heads").Scan(&count); e != nil || count != 0 {
		t.Fatal("CLI published", count, e)
	}
}
