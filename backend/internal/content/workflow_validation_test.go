package content

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/catalogue"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func workflowSeed(t *testing.T) (catalogue.Catalogue, Package, AssetReader) {
	t.Helper()
	c, _, _ := seed(t)
	raw, err := os.ReadFile("testdata/workflow-ready.json")
	if err != nil {
		t.Fatal(err)
	}
	var p Package
	if err = json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile("testdata/workflow-ready.svg")
	if err != nil {
		t.Fatal(err)
	}
	return c, p, func(ctx context.Context, a Asset) ([]byte, error) { return append([]byte{}, b...), ctx.Err() }
}
func TestWorkflowAssetReaderParity(t *testing.T) {
	c, p, db := workflowSeed(t)
	ctx := context.Background()
	file := FileAssetReader("testdata")
	a, ra := ValidateAndSealWithAssets(ctx, c, p, file)
	b, rb := ValidateAndSealWithAssets(ctx, c, p, db)
	if !a.Verify() || !b.Verify() || a.SHA256() != b.SHA256() || !reflect.DeepEqual(ra, rb) {
		t.Fatal("file and stored bytes changed digest/report")
	}
	old, legacy := ValidateAndSeal(c, p, "testdata")
	if old.SHA256() != a.SHA256() || !reflect.DeepEqual(legacy, ra) {
		t.Fatal("legacy seal semantics changed")
	}
	p.Assets[0].Path = "does-not-exist.svg"
	if v, r := ValidateWorkflow(ctx, c, p, db); !v.Verify() || !r.ReadyToSubmit {
		t.Fatal("database reader opened a disk path")
	}
	for _, path := range []string{"../workflow-ready.svg", "/workflow-ready.svg", "a\\b.svg", "a//b.svg"} {
		q := clone(p)
		q.Assets[0].Path = path
		if _, r := ValidateWorkflow(ctx, c, q, db); r.StructuralTotal == 0 {
			t.Fatal("bad path accepted")
		}
	}
	dir := t.TempDir()
	if err := os.Symlink(filepath.Join(mustAbs(t, "testdata"), "workflow-ready.svg"), filepath.Join(dir, "link.svg")); err != nil {
		t.Fatal(err)
	}
	q := clone(p)
	q.Assets[0].Path = "link.svg"
	if _, r := ValidateWorkflow(ctx, c, q, FileAssetReader(dir)); r.StructuralTotal == 0 {
		t.Fatal("symlink accepted")
	}
	q = clone(p)
	q.Assets[0].SHA256 = strings.Repeat("a", 64)
	if _, r := ValidateWorkflow(ctx, c, q, db); r.StructuralTotal == 0 {
		t.Fatal("wrong SHA accepted")
	}
	unsafe := []byte(`<svg xmlns="http://www.w3.org/2000/svg"><script/></svg>`)
	q = clone(p)
	q.Assets[0].SHA256 = fmt.Sprintf("%x", sha256.Sum256(unsafe))
	if _, r := ValidateWorkflow(ctx, c, q, func(context.Context, Asset) ([]byte, error) { return unsafe, nil }); r.StructuralTotal == 0 {
		t.Fatal("unsafe SVG with a matching SHA accepted")
	}

}
func mustAbs(t *testing.T, p string) string {
	t.Helper()
	v, err := filepath.Abs(p)
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func TestWorkflowCompleteness(t *testing.T) {
	ctx := context.Background()
	c, p, reader := workflowSeed(t)
	if v, r := ValidateWorkflow(ctx, c, p, reader); !v.Verify() || !r.ReadyToSubmit || r.HumanReviewTotal == 0 {
		t.Fatal("original fixture not ready for independent review")
	}
	for _, change := range []func(*Package){
		func(p *Package) { p.Knowledge[0].Type = "theorem"; p.Knowledge[0].Proof = " " }, func(p *Package) { p.Knowledge[0].System = "" }, func(p *Package) { p.Knowledge[0].Objectives = []string{} }, func(p *Package) { p.Knowledge[0].Sources = []Source{} }, func(p *Package) { p.Units[0].Angles[1].Kind = p.Units[0].Angles[0].Kind }, func(p *Package) { p.Units[0].Examples = []string{" "} }, func(p *Package) {
			p.Knowledge[0].Sources[0].Kind = "external"
			p.Knowledge[0].Sources[0].URL = "http://example.test/source"
			p.Knowledge[0].Sources[0].AccessedAt = "2026-02-30"
		}, func(p *Package) {
			p.Knowledge[0].Sources[0].Kind = "external"
			p.Knowledge[0].Sources[0].URL = "https://example.test/source"
			p.Knowledge[0].Sources[0].AccessedAt = "2026-02-30"
		},
	} {
		q := clone(p)
		change(&q)
		if _, r := ValidateWorkflow(ctx, c, q, reader); r.ReadyToSubmit || r.CompletenessTotal+r.StructuralTotal == 0 {
			t.Fatal("incomplete fixture ready")
		}
	}
	c, old, root := seed(t)
	_, r := ValidateWorkflow(ctx, c, old, FileAssetReader(root))
	if r.ReadyToSubmit || r.CompletenessTotal < 9 {
		t.Fatal("P1 skeletons became ready")
	}
}
func TestEditableDraftAndReportTruncation(t *testing.T) {
	c, p, reader := workflowSeed(t)
	ctx := context.Background()
	p.Knowledge[0].Relations = []Relation{{Kind: "prerequisite", Target: VersionRef{ID: "missing", Version: 1}}}
	r, err := ValidateEditable(ctx, c, p, reader)
	if err != nil || r.StructuralTotal != 1 || r.ReadyToSubmit {
		t.Fatal("safe incomplete draft rejected or marked ready")
	}
	p.Knowledge[0].Statement = "<script>alert(1)</script>"
	if _, err = ValidateEditable(ctx, c, p, reader); err == nil {
		t.Fatal("unsafe editable draft accepted")
	}
	c, p, reader = workflowSeed(t)
	for i := 0; i < 101; i++ {
		p.Knowledge[0].Relations = append(p.Knowledge[0].Relations, Relation{Kind: "prerequisite", Target: VersionRef{ID: fmt.Sprintf("missing-%d", i), Version: 1}})
	}
	full, err := ValidateEditable(ctx, c, p, reader)
	if err != nil || full.StructuralTotal != 101 {
		t.Fatal("actual issue totals lost")
	}
	report := full.Display()

	if len(report.StructuralErrors) != 100 || report.StructuralTotal != 101 || !report.Truncated || report.ReadyToSubmit {
		t.Fatal("truncation lost true totals or readiness")
	}
	ctx, cancel := context.WithCancel(ctx)
	cancel()
	if _, err = ValidateEditable(ctx, c, p, reader); err == nil {
		t.Fatal("canceled validation kept working")
	}
}

func TestWorkflowPrechecksBeforeReadingAssets(t *testing.T) {
	c, p, _ := workflowSeed(t)
	reader := func(context.Context, Asset) ([]byte, error) {
		t.Fatal("oversized package read assets")
		return nil, nil
	}
	for len(p.Knowledge) < 101 {
		p.Knowledge = append(p.Knowledge, p.Knowledge[0])
	}
	if _, err := ValidateEditable(context.Background(), c, p, reader); err != ErrLimit {
		t.Fatal("count limit missing")
	}
	c, p, _ = workflowSeed(t)
	p.Knowledge[0].Statement = strings.Repeat("x", 2<<20)
	if _, err := ValidateEditable(context.Background(), c, p, reader); err != ErrLimit {
		t.Fatal("byte limit missing")
	}
}
