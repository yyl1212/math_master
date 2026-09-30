package content

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAssetsRejectTraversalAndActiveSVG(t *testing.T) {
	cases := []string{"../x.svg", "/x.svg", "script", "onload", "foreignObject", "href", "style", "doctype", "oversize", "digest", "symlink"}
	for _, kind := range cases {
		t.Run(kind, func(t *testing.T) {
			c, p, old := seed(t)
			root := t.TempDir()
			b, e := os.ReadFile(filepath.Join(old, p.Assets[0].Path))
			if e != nil {
				t.Fatal(e)
			}
			switch kind {
			case "../x.svg", "/x.svg":
				p.Assets[0].Path = kind
			case "script", "foreignObject":
				b = []byte(`<svg xmlns="http://www.w3.org/2000/svg"><` + kind + `/></svg>`)
			case "onload", "href", "style":
				b = []byte(`<svg xmlns="http://www.w3.org/2000/svg" ` + kind + `="x"/>`)
			case "doctype":
				b = []byte(`<!DOCTYPE svg><svg/>`)
			case "oversize":
				b = []byte(strings.Repeat(" ", 1024*1024+1))
			}
			p.Assets[0].SHA256 = fmt.Sprintf("%x", sha256.Sum256(b))
			if kind == "digest" {
				p.Assets[0].SHA256 = strings.Repeat("0", 64)
			}
			path := filepath.Join(root, "equivalent-fractions.v1.svg")
			if kind == "symlink" {
				outside := filepath.Join(t.TempDir(), "x.svg")
				os.WriteFile(outside, b, 0600)
				if e = os.Symlink(outside, path); e != nil {
					t.Fatal(e)
				}
			} else {
				os.WriteFile(path, b, 0600)
			}
			blocked(t, c, p, root)
		})
	}
}
func TestMarkdownRejectsExecutableContent(t *testing.T) {
	for _, text := range []string{`<script>alert(1)</script>`, `[x](javascript:alert)`, `[x](//evil.test)`, `![x](https://evil.test/image)`, `$\href{https://evil.test}{x}$`, `$\input{file}$`, `$\newcommand{\x}{x}$`, `$\htmlClass{x}{x}$`} {
		t.Run(text, func(t *testing.T) { c, p, r := seed(t); p.Knowledge[0].Statement = text; blocked(t, c, p, r) })
	}
	c, p, r := seed(t)
	p.Knowledge[0].Statement = `For $a^2+b^2=c^2$, see [guide](https://example.com).`
	if report := ValidateStructure(c, p, r); len(report.Errors) > 0 {
		t.Fatal(report.Errors)
	}
}
func TestSealedPackageIncludesImmutableAssetBytes(t *testing.T) {
	c, p, old := seed(t)
	r := t.TempDir()
	bytes, err := os.ReadFile(filepath.Join(old, p.Assets[0].Path))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(r, p.Assets[0].Path), bytes, 0600); err != nil {
		t.Fatal(err)
	}
	v, report := ValidateAndSeal(c, p, r)
	if len(report.Errors) > 0 || v.SHA256() == "" {
		t.Fatal("valid package not sealed", report)
	}
	id := p.Assets[0].ID
	b, ok := v.AssetBytes(id)
	if !ok {
		t.Fatal("asset missing")
	}
	original := string(b)
	if err := os.WriteFile(filepath.Join(r, p.Assets[0].Path), []byte("changed after sealing"), 0600); err != nil {
		t.Fatal(err)
	}
	b[0] = 'x'
	copyP := v.Package()
	copyP.Knowledge[0].Title = "changed"
	copyC := v.Catalogue()
	copyC.Domains[0].Name = "changed"
	b2, _ := v.AssetBytes(id)
	if string(b2) != original || v.Package().Knowledge[0].Title == "changed" || v.Catalogue().Domains[0].Name == "changed" {
		t.Fatal("seal mutated")
	}
}
