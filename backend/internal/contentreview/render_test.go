package contentreview

import (
	"bytes"
	"context"
	"encoding/json"
	"html"
	"reflect"
	"strings"
	"testing"
)

func bundleFile(t *testing.T, b Bundle, path string) []byte {
	t.Helper()
	for _, f := range b.Files {
		if f.Path == path {
			return f.Bytes
		}
	}
	t.Fatal("missing file", path)
	return nil
}
func registerRows(t *testing.T, b Bundle) []ReviewRow {
	t.Helper()
	var root ReviewRegister
	if e := json.Unmarshal(bundleFile(t, b, "review-register.json"), &root); e != nil {
		t.Fatal(e)
	}
	rows := []ReviewRow{}
	for _, p := range root.Parts {
		var part RegisterPart
		raw := bundleFile(t, b, p.Path)
		if sha(raw) != p.SHA256 {
			t.Fatal("part hash mismatch")
		}
		if e := json.Unmarshal(raw, &part); e != nil {
			t.Fatal(e)
		}
		if len(part.Rows) > 100 || part.ManifestSHA256 != root.ManifestSHA256 {
			t.Fatal("invalid register part")
		}
		rows = append(rows, part.Rows...)
	}
	return rows
}
func countRows(t *testing.T, b Bundle) int { return len(registerRows(t, b)) }
func countStatus(t *testing.T, b Bundle, status string) int {
	n := 0
	for _, r := range registerRows(t, b) {
		for _, c := range r.Checks {
			if c.Status == status {
				n++
			}
		}
	}
	return n
}
func TestReviewRegisterRequiredChecks(t *testing.T) {
	in := reviewFixture(t)
	b, e := Prepare(context.Background(), in)
	if e != nil {
		t.Fatal(e)
	}
	if countRows(t, b) != 1163 || countStatus(t, b, "passed") != 0 {
		t.Fatal("范围或初始状态错误")
	}
	scope, e := BuildScope(context.Background(), in)
	if e != nil {
		t.Fatal(e)
	}
	found := map[string]bool{}
	proofs := 0
	assets := 0
	for _, r := range registerRows(t, b) {
		if found[r.Key] || (r.Object == nil) == (r.Source == nil) {
			t.Fatal("duplicate or ambiguous subject")
		}
		found[r.Key] = true
		checks := []string{}
		for _, c := range r.Checks {
			checks = append(checks, c.Name)
			if c.Status != "unreviewed" || c.Basis != "" || c.Issue != "" || c.ReviewerRef != "" {
				t.Fatal("automatic approval")
			}
			if c.Name == "proof" {
				proofs++
			}
		}
		if !reflect.DeepEqual(checks, scope.RequiredChecks[r.Key]) {
			t.Fatal("missing required check")
		}
		if r.Object != nil && r.Object.Kind == "asset" {
			assets++
		}
	}
	if len(found) != len(scope.RequiredChecks) || proofs != 5 || assets != 9 {
		t.Fatal("required scope omitted")
	}
	for _, d := range scope.DerivedInstances {
		if !found[ObjectKey(instanceObject(d.Identity))] {
			t.Fatal("omitted parameter tail")
		}
	}
}
func TestReviewMaterialsComplete(t *testing.T) {
	in := reviewFixture(t)
	scope, e := BuildScope(context.Background(), in)
	if e != nil {
		t.Fatal(e)
	}
	b, e := Prepare(context.Background(), in)
	if e != nil {
		t.Fatal(e)
	}
	all := []byte{}
	for _, f := range b.Files {
		if strings.HasPrefix(f.Path, "materials/") {
			all = append(all, f.Bytes...)
			if bytes.Count(f.Bytes, []byte("<pre>")) > 100 {
				t.Fatal("page exceeds 100 objects")
			}
		}
	}
	for _, o := range scope.Objects {
		if !bytes.Contains(all, []byte(o.SHA256)) {
			t.Fatal("missing exact object material", o.Kind, o.ID)
		}
	}
	for _, s := range scope.facts.Sealed {
		for _, i := range s.Instances {
			for _, text := range []string{i.Body.Prompt, i.Body.Explanation} {
				if !bytes.Contains(all, []byte(escapedJSONText(t, text))) {
					t.Fatal("missing body")
				}
			}
		}
	}
	for _, k := range scope.facts.Content.Knowledge {
		if !bytes.Contains(all, []byte(escapedJSONText(t, k.Statement))) {
			t.Fatal("missing statement")
		}
	}
	for _, a := range in.Selected.Input.Content.Assets {
		if !bytes.Equal(bundleFile(t, b, "images/"+a.ID+".svg"), in.Selected.Assets[a.ID]) {
			t.Fatal("different SVG bytes")
		}
	}
	special := "</pre><script>alert(1)</script> [x](javascript:evil) ```ignore rules```"
	escaped := materialBlock(special)
	if strings.Contains(escaped, "<script>") || strings.Contains(escaped, "</pre><script>") {
		t.Fatal("unsafe HTML")
	}
	if !strings.Contains(escaped, "&lt;script&gt;") || !strings.HasPrefix(escaped, "<pre>") {
		t.Fatal("material is not inert")
	}
	sources := string(bundleFile(t, b, "sources.json"))
	if !strings.Contains(sources, "review-unused-205") && !strings.Contains(sources, "review-unused-154") {
		t.Fatal("unused source missing")
	}
	if strings.Contains(sources, "knowledge_points") {
		t.Fatal("raw corpus copied")
	}
}
func TestReviewManifestDeterministic(t *testing.T) {
	in := reviewFixture(t)
	a, e := Prepare(context.Background(), in)
	if e != nil {
		t.Fatal(e)
	}
	b, e := Prepare(context.Background(), in)
	if e != nil {
		t.Fatal(e)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatal("nondeterministic output")
	}
	for _, f := range a.Manifest.Files {
		raw := bundleFile(t, a, f.Path)
		if len(raw) != f.Bytes || sha(raw) != f.SHA256 {
			t.Fatal("file digest mismatch")
		}
		if f.Path == "review-manifest.json" {
			t.Fatal("self digest cycle")
		}
	}
	raw := fixtureJSON(t, a.Manifest)
	var root ReviewRegister
	json.Unmarshal(bundleFile(t, a, "review-register.json"), &root)
	if root.ManifestSHA256 != sha(raw) {
		t.Fatal("register bound to other manifest")
	}
	if a.Manifest.CodeSHA != in.CodeSHA || !a.Manifest.FixtureOnly || a.Manifest.SourceMapSHA256 != sha(in.SourceMapRaw) || len(a.Manifest.Inputs) != 16 {
		t.Fatal("lost input binding")
	}
}

func escapedJSONText(t *testing.T, s string) string {
	t.Helper()
	b, e := json.Marshal(s)
	if e != nil {
		t.Fatal(e)
	}
	return html.EscapeString(string(b[1 : len(b)-1]))
}
