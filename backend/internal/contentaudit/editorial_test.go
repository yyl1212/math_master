package contentaudit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/content"
	"math/big"
	"os"
	"strings"
	"testing"
)

var editorialIDs = []string{"natural-numbers", "zero", "place-value", "number-line", "comparing-numbers", "rounding-estimation", "addition", "addition-properties", "subtraction", "addition-subtraction-inverse", "multiplication", "multiplication-properties", "division", "division-remainder", "order-of-operations", "factors", "divisibility", "fractions", "fraction-number-line", "equivalent-fractions", "simplifying-fractions", "comparing-fractions", "fractions-same-denominator", "fractions-unlike-denominator", "fraction-multiplication", "fraction-division", "decimal-place-value", "decimal-fraction-conversion", "ratios-proportions", "percentages"}
var editorialPrereqs = [][]int{{}, {1}, {1, 2}, {1, 2}, {3, 4}, {3, 5}, {3, 4}, {7}, {5, 7}, {7, 9}, {7, 8}, {11}, {9, 11}, {5, 13}, {7, 9, 11, 13}, {11, 14}, {3, 16}, {1, 13}, {4, 18}, {11, 18}, {16, 20}, {19, 20}, {7, 9, 18}, {16, 20, 23}, {12, 18}, {13, 25}, {3, 18}, {20, 21, 27}, {11, 26}, {28, 29}}

func editorialPackage(t *testing.T) content.Package {
	t.Helper()
	f, e := os.Open("../../../content/packages/elementary-foundations.v1.json")
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	p, e := content.DecodePackage(f)
	if e != nil {
		t.Fatal(e)
	}
	return p
}
func TestContentAuditEditorialRoute(t *testing.T) {
	p := editorialPackage(t)
	if len(p.Knowledge) != 30 || len(p.Units) != 30 || len(p.Paths) != 1 || len(p.Assets) != 9 {
		t.Fatal("locked counts")
	}
	versions2 := map[string]bool{"factors": true, "fractions": true, "equivalent-fractions": true, "simplifying-fractions": true, "comparing-fractions": true, "decimal-place-value": true, "decimal-fraction-conversion": true, "percentages": true}
	for i, k := range p.Knowledge {
		wantVersion := 1
		if versions2[k.ID] {
			wantVersion = 2
		}
		if k.ID != editorialIDs[i] || k.Version != wantVersion || len(k.Objectives) != 2 || len(k.Relations) != len(editorialPrereqs[i]) {
			t.Fatal("node identity/goals", i)
		}
		for j, num := range editorialPrereqs[i] {
			target := p.Knowledge[num-1]
			if k.Relations[j].Kind != "prerequisite" || k.Relations[j].Target != (content.VersionRef{ID: target.ID, Version: target.Version}) {
				t.Fatal("locked prerequisite", i, j)
			}
		}
		u := p.Units[i]
		if u.ID != "ef-"+k.ID+"-unit" || u.Version != 1 || u.Knowledge != (content.VersionRef{ID: k.ID, Version: k.Version}) || len(u.Angles) < 2 || u.Angles[0].Kind != "intuitive" || u.Angles[1].Kind != "formal" || len(u.Examples) < 2 || len(u.Counterexamples) == 0 {
			t.Fatal("unit", i)
		}
		if p.Paths[0].Nodes[i] != (content.VersionRef{ID: k.ID, Version: k.Version}) {
			t.Fatal("route order")
		}
	}
	if p.Paths[0].ID != "elementary-foundations" || p.Paths[0].Version != 1 {
		t.Fatal("route identity")
	}
	cfile, e := os.Open("../../../content/catalogue/domains.json")
	if e != nil {
		t.Fatal(e)
	}
	defer cfile.Close()
	c, e := content.DecodeCatalogue(cfile)
	if e != nil {
		t.Fatal(e)
	}
	sealed, r := content.ValidateWorkflow(context.Background(), c, p, content.FileAssetReader("../../../content/assets"))
	if !r.ReadyToSubmit || !sealed.Verify() {
		t.Fatal(r)
	}
	for _, a := range p.Assets {
		b, e := os.ReadFile("../../../content/assets/" + a.Path)
		if e != nil || content.ValidateSVG(b) != nil {
			t.Fatal("unsafe SVG", a.ID, e)
		}
		digest := sha256.Sum256(b)
		if hex.EncodeToString(digest[:]) != a.SHA256 || a.Author != "Math Master project" || a.License != "CC0-1.0" {
			t.Fatal("asset identity")
		}
	}
	b, e := os.ReadFile("../../../content/source-maps/elementary-foundations.v1.json")
	if e != nil {
		t.Fatal(e)
	}
	var m SourceMap
	if DecodeSourceMap(strings.NewReader(string(b)), &m) != nil {
		t.Fatal("source mapping")
	}
	counts := map[string]int{}
	for _, o := range m.Objects {
		counts[o.Kind]++
	}
	if counts["knowledge"] != 30 || counts["unit"] != 30 || counts["path"] != 1 || counts["asset"] != 9 {
		t.Fatal("mapped editorial objects")
	}
	for _, s := range m.Sources {
		if strings.Contains(strings.ToLower(s.ReviewStatus), "human_approved") {
			t.Fatal("AI became approval")
		}
	}
}
func TestContentAuditEditorialMathematicalExamples(t *testing.T) {
	p := editorialPackage(t)
	all, _ := json.Marshal(p)
	for _, fragment := range []string{"N_0", "305 = 3", "1/2 = 2/4", "1/2 + 1/3 = 5/6", "nonzero", "2.50 = 2.5", "stated whole"} {
		if !strings.Contains(string(all), fragment) {
			t.Fatal("named example missing", fragment)
		}
	}
	half := big.NewRat(1, 2)
	third := big.NewRat(1, 3)
	sum := new(big.Rat).Add(half, third)
	if sum.Cmp(big.NewRat(5, 6)) != 0 || sum.Cmp(big.NewRat(2, 5)) == 0 {
		t.Fatal("fraction golden")
	}
	if new(big.Rat).Mul(big.NewRat(1, 4), big.NewRat(200, 1)).Cmp(big.NewRat(50, 1)) != 0 {
		t.Fatal("percentage whole golden")
	}
	for _, a := range p.Assets {
		raw, _ := os.ReadFile("../../../content/assets/" + a.Path)
		if !strings.Contains(string(raw), "<title>") || !strings.Contains(string(raw), "<desc>") {
			t.Fatal("SVG labels", a.ID)
		}
	}
}
func TestContentAuditEditorialLegacyBytes(t *testing.T) {
	b, e := os.ReadFile("../../../content/packages/elementary-fractions.v1.json")
	if e != nil {
		t.Fatal(e)
	}
	sum := sha256.Sum256(b)
	if hex.EncodeToString(sum[:]) != "2f0f5bd163e036cf7523875ea1a7a0273f9e2e6b665a5532e1dfa9981b3d686c" {
		t.Fatal("legacy package bytes changed")
	}
}
