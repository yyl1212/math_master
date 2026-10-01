package content

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"github.com/yyl1212/math_master/backend/internal/catalogue"
	"net/url"
	"regexp"
	"strings"
)

type Issue struct {
	Code    string `json:"code"`
	Path    string `json:"path"`
	Message string `json:"message"`
}
type Report struct {
	Errors      []Issue `json:"errors"`
	ReviewItems []Issue `json:"reviewItems"`
}
type ValidatedPackage struct {
	catalogue         catalogue.Catalogue
	pkg               Package
	assets            map[string][]byte
	sha, catalogueSHA string
}

func clone[T any](v T) T                                  { b, _ := json.Marshal(v); var c T; _ = json.Unmarshal(b, &c); return c }
func Digest(v any) string                                 { b, _ := json.Marshal(v); return fmt.Sprintf("%x", sha256.Sum256(b)) }
func (v ValidatedPackage) Catalogue() catalogue.Catalogue { return clone(v.catalogue) }
func (v ValidatedPackage) Package() Package               { return clone(v.pkg) }
func (v ValidatedPackage) AssetBytes(id string) ([]byte, bool) {
	b, ok := v.assets[id]
	return bytes.Clone(b), ok
}
func (v ValidatedPackage) SHA256() string          { return v.sha }
func (v ValidatedPackage) CatalogueSHA256() string { return v.catalogueSHA }
func sealDigest(c string, p Package) string {
	return Digest(struct {
		CatalogueSHA string
		Package      Package
	}{c, p})
}
func (v ValidatedPackage) Verify() bool {
	if v.sha == "" || v.catalogueSHA != Digest(v.catalogue) || v.sha != sealDigest(v.catalogueSHA, v.pkg) {
		return false
	}
	for _, a := range v.pkg.Assets {
		b, ok := v.assets[a.ID]
		if !ok || fmt.Sprintf("%x", sha256.Sum256(b)) != a.SHA256 {
			return false
		}
	}
	return true
}
func ValidateStructure(c catalogue.Catalogue, p Package, root string) Report {
	_, r := ValidateAndSeal(c, p, root)
	return r
}

var idPattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,63}$`)
var forbiddenMacro = regexp.MustCompile(`(?i)\\(?:href|url|html[a-z]*|input|include[a-z]*|def|gdef|edef|xdef|let|newcommand|renewcommand|providecommand|newenvironment|renewenvironment|usepackage|require|write|openout|read|catcode|csname)\b`)

func safeURL(s string) bool {
	if strings.ContainsAny(s, "\\\x00\r\n") {
		return false
	}
	if strings.HasPrefix(s, "/") {
		return !strings.HasPrefix(s, "//")
	}
	u, e := url.Parse(s)
	return e == nil && u.Scheme == "https" && u.Hostname() != "" && u.User == nil
}
func safeMarkdown(s string, assets map[string]bool) bool {
	if forbiddenMacro.MatchString(s) {
		return false
	}
	tree := goldmark.New().Parser().Parse(text.NewReader([]byte(s)))
	safe := true
	_ = ast.Walk(tree, func(n ast.Node, enter bool) (ast.WalkStatus, error) {
		if !enter {
			return ast.WalkContinue, nil
		}
		switch n := n.(type) {
		case *ast.RawHTML, *ast.HTMLBlock:
			safe = false
		case *ast.Link:
			if !safeURL(string(n.Destination)) {
				safe = false
			}
		case *ast.AutoLink:
			if !safeURL(string(n.URL([]byte(s)))) {
				safe = false
			}
		case *ast.Image:
			dest := string(n.Destination)
			if !strings.HasPrefix(dest, "asset:") || !assets[strings.TrimPrefix(dest, "asset:")] {
				safe = false
			}
		}
		return ast.WalkContinue, nil
	})
	return safe
}
func ValidateAndSeal(c catalogue.Catalogue, p Package, root string) (ValidatedPackage, Report) {
	return ValidateAndSealWithAssets(context.Background(), c, p, FileAssetReader(root))
}
func ValidateAndSealWithAssets(ctx context.Context, c catalogue.Catalogue, p Package, reader AssetReader) (ValidatedPackage, Report) {
	return validateAndSeal(ctx, c, p, reader, MaxPackageBytes, MaxPackageBytes, false)
}
func validateAndSeal(ctx context.Context, c catalogue.Catalogue, p Package, reader AssetReader, maxJSON, maxAssets int, uniqueAssets bool) (ValidatedPackage, Report) {
	r := Report{Errors: []Issue{}, ReviewItems: []Issue{}}
	errAt := func(code, path string) { r.Errors = append(r.Errors, Issue{code, path, "Content validation failed."}) }
	review := func(path string) {
		r.ReviewItems = append(r.ReviewItems, Issue{"REVIEW_REQUIRED", path, "Incomplete draft or independent review pending."})
	}
	cb, _ := json.Marshal(c)
	pb, _ := json.Marshal(p)
	if err := ctx.Err(); err != nil {
		errAt("CANCELLED", "/")
		return ValidatedPackage{}, r
	}
	if _, e := DecodeCatalogue(bytes.NewReader(cb)); e != nil {
		errAt("INVALID_CATALOGUE", "/")
	}
	var decoded Package
	if e := decodeLimit(bytes.NewReader(pb), "content-package.schema.json", &decoded, maxJSON); e != nil {
		errAt("INVALID_PACKAGE", "/")
	}
	if len(r.Errors) > 0 {
		return ValidatedPackage{}, r
	}
	domains := map[string]bool{}
	topics := map[string]string{}
	orders := map[int]bool{}
	for i, d := range c.Domains {
		path := fmt.Sprintf("/domains/%d", i)
		if domains[d.ID] || orders[d.Order] {
			errAt("DUPLICATE_ID", path)
		}
		domains[d.ID] = true
		orders[d.Order] = true
		for _, t := range d.Topics {
			if _, ok := topics[t.ID]; ok {
				errAt("DUPLICATE_ID", path+"/topics")
			}
			topics[t.ID] = d.ID
		}
	}
	for i, d := range c.Domains {
		seen := map[string]bool{}
		for j, id := range d.RelatedDomainIDs {
			if seen[id] {
				errAt("DUPLICATE_RELATION", fmt.Sprintf("/domains/%d/relatedDomainIds/%d", i, j))
			}
			seen[id] = true
			if !domains[id] {
				errAt("MISSING_DOMAIN", fmt.Sprintf("/domains/%d/relatedDomainIds", i))
			}
		}
	}
	checkDomains := func(ids, ts []string, path string) {
		ds := map[string]bool{}
		for _, id := range ids {
			if !domains[id] || ds[id] {
				errAt("INVALID_DOMAIN", path)
			}
			ds[id] = true
		}
		seen := map[string]bool{}
		for _, id := range ts {
			d, ok := topics[id]
			if !ok || !ds[d] || seen[id] {
				errAt("INVALID_TOPIC", path)
			}
			seen[id] = true
		}
	}
	kmap := map[string]Knowledge{}
	for i, k := range p.Knowledge {
		path := fmt.Sprintf("/knowledge/%d", i)
		if _, ok := kmap[k.ID]; ok {
			errAt("DUPLICATE_ID", path)
		}
		kmap[k.ID] = k
		checkDomains(k.DomainIDs, k.TopicIDs, path)
	}
	refOK := func(ref VersionRef, path string) bool {
		k, ok := kmap[ref.ID]
		if !ok || k.Version != ref.Version {
			errAt("INVALID_REFERENCE", path)
			return false
		}
		return true
	}
	graph := map[string][]string{}
	for i, k := range p.Knowledge {
		seen := map[string]bool{}
		for j, rel := range k.Relations {
			path := fmt.Sprintf("/knowledge/%d/relations/%d", i, j)
			refOK(rel.Target, path)
			key := rel.Kind + ":" + rel.Target.ID
			if seen[key] {
				errAt("DUPLICATE_RELATION", path)
			}
			seen[key] = true
			if rel.Kind == "prerequisite" {
				graph[k.ID] = append(graph[k.ID], rel.Target.ID)
			}
		}
	}
	// Kahn traversal bounds stack use even for long dependency chains.
	incoming := map[string]int{}
	dependents := map[string][]string{}
	for _, k := range p.Knowledge {
		incoming[k.ID] = 0
	}
	for id, refs := range graph {
		for _, ref := range refs {
			if _, ok := kmap[ref]; ok {
				incoming[id]++
				dependents[ref] = append(dependents[ref], id)
			}
		}
	}
	queue := []string{}
	for _, k := range p.Knowledge {
		if incoming[k.ID] == 0 {
			queue = append(queue, k.ID)
		}
	}
	visited := 0
	for i := 0; i < len(queue); i++ {
		if ctx.Err() != nil {
			errAt("CANCELLED", "/")
			return ValidatedPackage{}, r
		}
		visited++
		for _, id := range dependents[queue[i]] {
			incoming[id]--
			if incoming[id] == 0 {
				queue = append(queue, id)
			}
		}
	}
	if visited < len(kmap) {
		errAt("PREREQUISITE_CYCLE", "/knowledge")
	}
	assetIDs := map[string]bool{}
	assetOwners := map[string]VersionRef{}
	assetBytes := map[string][]byte{}
	total := 0
	uniqueBytes := map[string]bool{}
	for i, a := range p.Assets {
		path := fmt.Sprintf("/assets/%d", i)
		if assetIDs[a.ID] {
			errAt("DUPLICATE_ID", path)
		}
		assetIDs[a.ID] = true
		assetOwners[a.ID] = a.Knowledge
		refOK(a.Knowledge, path+"/knowledge")
		if ctx.Err() != nil {
			errAt("CANCELLED", "/")
			return ValidatedPackage{}, r
		}
		var b []byte
		var e error
		if reader == nil || !ValidAssetPath(a.Path) {
			e = fmt.Errorf("invalid asset reader or path")
		} else {
			b, e = reader(ctx, a)
		}
		if e == nil && (len(b) > 1<<20 || fmt.Sprintf("%x", sha256.Sum256(b)) != a.SHA256 || ValidateSVG(b) != nil) {
			e = fmt.Errorf("invalid stored asset")
		}

		if e != nil {
			errAt("INVALID_ASSET", path)
			continue
		}
		if !uniqueAssets || !uniqueBytes[a.SHA256] {
			total += len(b)
			uniqueBytes[a.SHA256] = true
		}
		if total > maxAssets {
			errAt("ASSETS_TOO_LARGE", "/assets")
			return ValidatedPackage{}, r
		}
		assetBytes[a.ID] = b
	}
	checkText := func(s, path string) {
		if !safeMarkdown(s, assetIDs) {
			errAt("UNSAFE_MARKUP", path)
		}
	}
	for i, k := range p.Knowledge {
		path := fmt.Sprintf("/knowledge/%d", i)
		for _, s := range append([]string{k.Title, k.TitleZh, k.Statement, k.Scope, k.System, k.Proof}, append(k.Objectives, k.Conditions...)...) {
			owned := map[string]bool{}
			for id, owner := range assetOwners {
				if owner.ID == k.ID && owner.Version == k.Version {
					owned[id] = true
				}
			}
			if !safeMarkdown(s, owned) {
				errAt("UNSAFE_MARKUP", path)
			}
		}
		if k.Statement == "" || k.Scope == "" || (k.Type == "theorem" && k.Proof == "") {
			review(path)
		}
		if len(k.Sources) == 0 {
			review(path + "/sources")
		}
		for _, s := range k.Sources {
			if s.URL != "" && !safeURL(s.URL) {
				errAt("UNSAFE_SOURCE_URL", path+"/sources")
			}
			if s.License == "" || s.Author == "" {
				review(path + "/sources")
			}
		}
		review(path)
	}
	seenUnit := map[string]bool{}
	for i, u := range p.Units {
		path := fmt.Sprintf("/units/%d", i)
		if seenUnit[u.ID] {
			errAt("DUPLICATE_ID", path)
		}
		seenUnit[u.ID] = true
		refOK(u.Knowledge, path)
		allowed := map[string]bool{}
		for _, id := range u.AssetIDs {
			if !assetIDs[id] || allowed[id] || assetOwners[id] != u.Knowledge {
				errAt("INVALID_ASSET_REFERENCE", path)
			}
			allowed[id] = true
		}
		for _, a := range u.Angles {
			if !safeMarkdown(a.Body, allowed) {
				errAt("UNSAFE_MARKUP", path)
			}
		}
		for _, s := range append(u.Examples, u.Counterexamples...) {
			if !safeMarkdown(s, allowed) {
				errAt("UNSAFE_MARKUP", path)
			}
		}
		if len(u.Angles) < 2 {
			review(path)
		}
	}
	seenPath := map[string]bool{}
	for i, path := range p.Paths {
		where := fmt.Sprintf("/paths/%d", i)
		if seenPath[path.ID] {
			errAt("DUPLICATE_ID", where)
		}
		seenPath[path.ID] = true
		checkDomains(path.DomainIDs, nil, where)
		nodes := map[string]bool{}
		for _, ref := range path.Nodes {
			refOK(ref, where)
			if nodes[ref.ID] {
				errAt("DUPLICATE_NODE", where)
			}
			nodes[ref.ID] = true
		}
		for _, ref := range path.Nodes {
			for _, pre := range graph[ref.ID] {
				if !nodes[pre] {
					errAt("INCOMPLETE_PATH", where)
				}
			}
		}
		checkText(path.Title, where)
		checkText(path.TitleZh, where)
	}
	if ctx.Err() != nil {
		errAt("CANCELLED", "/")
	}
	if len(r.Errors) > 0 {
		return ValidatedPackage{}, r
	}
	v := ValidatedPackage{catalogue: clone(c), pkg: clone(p), assets: assetBytes, catalogueSHA: Digest(c)}
	v.sha = sealDigest(v.catalogueSHA, v.pkg)
	return v, r
}
