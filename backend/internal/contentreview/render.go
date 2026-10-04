package contentreview

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"sort"
	"strings"

	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/contentaudit"
	"github.com/yyl1212/math_master/backend/internal/question"
)

const MaxFileBytes = 8 << 20
const MaxFiles = 256
const MaxTotalBytes = 256 << 20

// Preformatted escaped text preserves mathematics while keeping Markdown, links and HTML inert.
func materialBlock(text string) string { return "<pre>" + html.EscapeString(text) + "</pre>\n" }
func encodeFile(path string, v any) (ExportFile, error) {
	b, e := json.Marshal(v)
	if e != nil {
		return ExportFile{}, ErrInvalid
	}
	if len(b) > MaxFileBytes {
		return ExportFile{}, ErrLimit
	}
	return ExportFile{path, b}, nil
}
func entry(f ExportFile) FileEntry {
	return FileEntry{Path: f.Path, Bytes: len(f.Bytes), SHA256: digestBytes(f.Bytes)}
}
func Prepare(ctx context.Context, in PrepareInput) (Bundle, error) {
	out := Bundle{Files: []ExportFile{}}
	scope, e := BuildScope(ctx, in)
	if e != nil {
		return out, e
	}
	imports, e := BuildImports(ctx, in, scope)
	if e != nil {
		return out, e
	}
	files := []ExportFile{}
	addJSON := func(p string, v any) error {
		f, e := encodeFile(p, v)
		if e != nil {
			return e
		}
		files = append(files, f)
		return nil
	}
	if e = addJSON("sources.json", scope.Sources); e != nil {
		return out, e
	}
	if e = addJSON("imports/knowledge.json", imports.Knowledge); e != nil {
		return out, e
	}
	for _, q := range imports.Questions {
		if !question.ValidMathID(q.QuestionPackage.ID) {
			return out, ErrInvalid
		}
		if e = addJSON("imports/questions/"+q.QuestionPackage.ID+".json", q); e != nil {
			return out, e
		}
	}
	for _, a := range scope.facts.Content.Assets {
		if !question.ValidMathID(a.ID) {
			return out, ErrInvalid
		}
		files = append(files, ExportFile{"images/" + a.ID + ".svg", append([]byte{}, in.Selected.Assets[a.ID]...)})
	}
	materials, e := renderMaterials(ctx, scope)
	if e != nil {
		return out, e
	}
	files = append(files, materials...)
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	manifest := ReviewManifest{SchemaVersion: 1, CodeSHA: in.CodeSHA, FixtureOnly: in.FixtureOnly, CatalogueVersion: scope.facts.References.CatalogueVersion, CatalogueSHA256: scope.facts.References.CatalogueSHA256, Route: RouteIdentity{ID: scope.facts.Path.ID, Version: scope.facts.Path.Version, SHA256: content.Digest(scope.facts.Path)}, SnapshotID: in.Sources.SnapshotID, SourceReportSHA256: in.Sources.ReportSHA, SourceMapSHA256: digestBytes(in.SourceMapRaw), InputManifestSHA256: digestBytes(in.ManifestRaw), Inputs: []FileEntry{}, Objects: scope.Objects, DerivedInstances: scope.DerivedInstances, Sources: scope.Sources, RequiredChecks: []SubjectChecks{}, Files: []FileEntry{}}
	for _, f := range in.Selected.Files {
		manifest.Inputs = append(manifest.Inputs, entry(ExportFile{f.Path, f.Bytes}))
	}
	sort.Slice(manifest.Inputs, func(i, j int) bool { return manifest.Inputs[i].Path < manifest.Inputs[j].Path })
	for key, checks := range scope.RequiredChecks {
		manifest.RequiredChecks = append(manifest.RequiredChecks, SubjectChecks{key, checks})
	}
	sort.Slice(manifest.RequiredChecks, func(i, j int) bool { return manifest.RequiredChecks[i].Key < manifest.RequiredChecks[j].Key })
	// Registers bind the completed manifest. Their hashes are in the register root,
	// outside manifest.files, avoiding a manifest/register digest cycle.
	for _, f := range files {
		manifest.Files = append(manifest.Files, entry(f))
	}
	manifestRaw, e := json.Marshal(manifest)
	if e != nil {
		return out, e
	}
	if len(manifestRaw) > MaxFileBytes {
		return out, ErrLimit
	}
	manifestSHA := digestBytes(manifestRaw)
	rows := []ReviewRow{}
	row := func(key string) ReviewRow {
		r := ReviewRow{Key: key, Checks: []ReviewCheck{}}
		for _, name := range scope.RequiredChecks[key] {
			r.Checks = append(r.Checks, ReviewCheck{Name: name, Status: "unreviewed"})
		}
		return r
	}
	for _, o := range scope.Objects {
		r := row(ObjectKey(o))
		value := o
		r.Object = &value
		rows = append(rows, r)
	}
	for _, s := range scope.Sources {
		r := row(SourceKey(s.SourceIdentity))
		value := s.SourceIdentity
		r.Source = &value
		rows = append(rows, r)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Key < rows[j].Key })
	register := ReviewRegister{SchemaVersion: 1, ManifestSHA256: manifestSHA, Parts: []FileRef{}}
	for offset := 0; offset < len(rows); {
		if e = ctx.Err(); e != nil {
			return out, e
		}
		end := min(offset+100, len(rows))
		var f ExportFile
		for {
			f, e = encodeFile(fmt.Sprintf("register/%03d.json", len(register.Parts)+1), RegisterPart{1, manifestSHA, rows[offset:end]})
			if e == nil {
				break
			}
			if e != ErrLimit || end == offset+1 {
				return out, e
			}
			end--
		}
		files = append(files, f)
		register.Parts = append(register.Parts, FileRef{f.Path, digestBytes(f.Bytes)})
		offset = end
	}
	f, e := encodeFile("review-register.json", register)
	if e != nil {
		return out, e
	}
	files = append(files, f)
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	total := len(manifestRaw)
	seen := map[string]bool{}
	for _, f := range files {
		if e = ctx.Err(); e != nil {
			return out, e
		}
		if !safePath(f.Path) || seen[f.Path] {
			return out, ErrInvalid
		}
		seen[f.Path] = true
		if len(f.Bytes) > MaxFileBytes {
			return out, ErrLimit
		}
		total += len(f.Bytes)
	}
	if len(files)+1 > MaxFiles || total > MaxTotalBytes {
		return out, ErrLimit
	}
	out.Manifest = manifest
	out.Files = files
	return out, ctx.Err()
}
func renderMaterials(ctx context.Context, scope ReviewScope) ([]ExportFile, error) {
	type material struct {
		identity contentaudit.ObjectIdentity
		value    any
		image    string
	}
	groups := map[string][]material{}
	themes := map[content.VersionRef]string{}
	for _, s := range scope.facts.Sealed {
		theme := strings.TrimPrefix(s.Package.ID, "elementary-foundations-")
		switch theme {
		case "numbers", "operations", "fractions", "decimals", "ratios":
		default:
			return nil, ErrInvalid
		}
		for _, b := range s.Package.Blueprints {
			themes[b.Knowledge] = theme
		}
	}
	put := func(kind string, ref content.VersionRef, id contentaudit.ObjectIdentity, value any, image string) error {
		theme := themes[ref]
		if kind == "path" {
			theme = "numbers"
		}
		if theme == "" {
			return ErrInvalid
		}
		key := theme + "/" + kind
		groups[key] = append(groups[key], material{id, value, image})
		return nil
	}
	for _, k := range scope.facts.Content.Knowledge {
		if e := put("mathematics", content.VersionRef{ID: k.ID, Version: k.Version}, object("knowledge", k.ID, k.Version, content.Digest(k)), k, ""); e != nil {
			return nil, e
		}
	}
	for _, u := range scope.facts.Content.Units {
		if e := put("mathematics", u.Knowledge, object("unit", u.ID, u.Version, content.Digest(u)), u, ""); e != nil {
			return nil, e
		}
	}
	for _, a := range scope.facts.Content.Assets {
		if e := put("mathematics", a.Knowledge, contentauditObjectAsset(a), a, a.ID); e != nil {
			return nil, e
		}
	}
	p := scope.facts.Path
	if e := put("path", content.VersionRef{}, object("path", p.ID, p.Version, content.Digest(p)), p, ""); e != nil {
		return nil, e
	}
	for _, s := range scope.facts.Sealed {
		for _, t := range s.Package.Templates {
			_, hash, e := question.CanonicalTemplate(t)
			if e != nil {
				return nil, e
			}
			if e = put("templates", t.Knowledge, object("template", t.ID, t.Version, hash), t, ""); e != nil {
				return nil, e
			}
		}
		for _, i := range s.Instances {
			if e := put("instances", i.Body.Knowledge, instanceObject(i.Identity), i, ""); e != nil {
				return nil, e
			}
		}
	}
	for p, r := range scope.facts.QuestionReports {
		for _, n := range r.Coverage {
			for _, b := range scope.facts.Sealed[p].Package.Blueprints {
				if n.Blueprint != nil && b.ID == n.Blueprint.ID && b.Version == n.Blueprint.Version {
					if e := put("blueprints", b.Knowledge, object("blueprint", b.ID, b.Version, n.Blueprint.SHA256), struct {
						Blueprint question.Blueprint    `json:"blueprint"`
						Coverage  question.CoverageNode `json:"coverage"`
					}{b, n}, ""); e != nil {
						return nil, e
					}
				}
			}
		}
	}
	keys := []string{}
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	files := []ExportFile{}
	seen := map[string]bool{}
	for _, key := range keys {
		items := groups[key]
		sort.Slice(items, func(i, j int) bool { return objectLess(items[i].identity, items[j].identity) })
		page := 1
		count := 0
		body := "# 全量复核材料\n\n每个准确身份逐项核对；本材料未作数学或许可批准。\n\n"
		flush := func() {
			files = append(files, ExportFile{fmt.Sprintf("materials/%s-%03d.md", key, page), []byte(body)})
			page++
			count = 0
			body = "# 全量复核材料（续）\n\n"
		}
		for _, m := range items {
			if e := ctx.Err(); e != nil {
				return nil, e
			}
			id := ObjectKey(m.identity)
			if seen[id] {
				return nil, ErrInvalid
			}
			seen[id] = true
			raw, e := json.MarshalIndent(struct {
				Identity contentaudit.ObjectIdentity `json:"identity"`
				Value    any                         `json:"value"`
			}{m.identity, m.value}, "", "  ")
			if e != nil {
				return nil, e
			}
			block := materialBlock(string(raw))
			if m.image != "" {
				block += "\n![原创图示，需核对桌面和移动视口](../../images/" + m.image + ".svg)\n\n"
			}
			if len(block) > MaxFileBytes {
				return nil, ErrLimit
			}
			if count == 100 || len(body)+len(block) > MaxFileBytes {
				flush()
			}
			body += block
			count++
		}
		if count > 0 {
			flush()
		}
	}
	if len(seen) != len(scope.Objects) {
		return nil, ErrInvalid
	}
	return files, nil
}
