package contentreview

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/contentaudit"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
)

const SnapshotID = "eebca0e9f8cbe8ce9afbed5f1f872f382f54e42b8004fcb57d23d05b99ff879a"
const SourceReportSHA = "7bd7c12118b915bc0a0ae4398b8018f1ba76e305736703a0617802677c7874ad"

var firstIDs = strings.Fields("natural-numbers zero place-value number-line comparing-numbers rounding-estimation addition addition-properties subtraction addition-subtraction-inverse multiplication multiplication-properties division division-remainder order-of-operations factors divisibility fractions fraction-number-line equivalent-fractions simplifying-fractions comparing-fractions fractions-same-denominator fractions-unlike-denominator fraction-multiplication fraction-division decimal-place-value decimal-fraction-conversion ratios-proportions percentages")

func digestBytes(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func safePath(p string) bool {
	return publication.RelativeSourcePath(p) && path.Clean(p) == p && !strings.Contains(p, "\\")
}
func strict(raw []byte, limit int, out any) error {
	if len(raw) > limit {
		return ErrLimit
	}
	if e := question.DecodeOperationalJSON(bytes.NewReader(raw), limit, out); e != nil {
		return ErrInvalid
	}
	return nil
}
func object(kind, id string, v int, sha string) contentaudit.ObjectIdentity {
	return contentaudit.ObjectIdentity{Kind: kind, ID: id, Version: &v, SHA256: sha}
}
func instanceObject(i question.Identity) contentaudit.ObjectIdentity {
	return object("instance", i.ID, i.Version, i.SHA256)
}
func identityLabel(o contentaudit.ObjectIdentity) string { o.SHA256 = ""; return ObjectKey(o) }
func objectLess(a, b contentaudit.ObjectIdentity) bool {
	if a.Kind != b.Kind {
		return a.Kind < b.Kind
	}
	if a.ID != b.ID {
		return a.ID < b.ID
	}
	return ObjectKey(a) < ObjectKey(b)
}
func sortObjects(s []contentaudit.ObjectIdentity) {
	sort.Slice(s, func(i, j int) bool { return objectLess(s[i], s[j]) })
}
func (s ReviewScope) checksFor(k string) []string { return s.RequiredChecks[k] }

func BuildScope(ctx context.Context, in PrepareInput) (ReviewScope, error) {
	out := ReviewScope{Objects: []contentaudit.ObjectIdentity{}, MappedObjects: []contentaudit.ObjectIdentity{}, DerivedInstances: []DerivedInstance{}, Sources: []SourceRecord{}, RequiredChecks: map[string][]string{}}
	if e := ctx.Err(); e != nil {
		return out, e
	}
	if len(in.CodeSHA) != 40 || strings.Trim(in.CodeSHA, "0123456789abcdef") != "" {
		return out, ErrInvalid
	}
	var manifest InputManifest
	if e := strict(in.ManifestRaw, MaxManifestBytes, &manifest); e != nil {
		return out, e
	}
	if manifest.SchemaVersion != 1 || content.Digest(manifest) != content.Digest(in.Manifest) || len(manifest.QuestionPaths) != 5 || !safePath(manifest.AssetsRoot) {
		return out, ErrInvalid
	}
	paths := append([]string{manifest.CataloguePath, manifest.ContentPath}, manifest.QuestionPaths...)
	if len(in.Selected.Files) != 7+len(in.Selected.Input.Content.Assets) {
		return out, ErrInvalid
	}
	seenPath := map[string]bool{}
	for i, p := range paths {
		if !safePath(p) || seenPath[p] || in.Selected.Files[i].Path != p {
			return out, ErrInvalid
		}
		seenPath[p] = true
	}
	for i, a := range in.Selected.Input.Content.Assets {
		if !safePath(a.Path) || in.Selected.Files[7+i].Path != path.Join(manifest.AssetsRoot, a.Path) {
			return out, ErrInvalid
		}
	}
	var r contentaudit.SourceReport
	var m contentaudit.SourceMap
	if len(in.SourceReportRaw) > contentaudit.MaxMetadataBytes || len(in.SourceMapRaw) > contentaudit.MaxSourceMapBytes {
		return out, ErrLimit
	}
	if contentaudit.DecodeSourceReport(bytes.NewReader(in.SourceReportRaw), &r) != nil || contentaudit.DecodeSourceMap(bytes.NewReader(in.SourceMapRaw), &m) != nil {
		return out, ErrInvalid
	}
	if content.Digest(r) != content.Digest(in.Sources.Report) || content.Digest(m) != content.Digest(in.Sources.Mapping) || digestBytes(in.SourceReportRaw) != in.Sources.ReportSHA || m.SourceReportSHA256 != in.Sources.ReportSHA || r.SnapshotID != in.Sources.SnapshotID || m.SnapshotID != r.SnapshotID || r.SchemaVersion != 1 || r.PolicyVersion != 1 || m.SchemaVersion != 1 || m.PolicyVersion != 1 || r.PublicationApproved || !r.Ready || !question.ValidSHA(r.SnapshotID) {
		return out, ErrInvalid
	}
	if !in.FixtureOnly && (r.SnapshotID != SnapshotID || in.Sources.ReportSHA != SourceReportSHA) {
		return out, ErrInvalid
	}
	for _, i := range r.Issues {
		if i.BlocksSelected {
			return out, ErrInvalid
		}
	}
	sourceIndex := map[string]int{}
	selectedPaths := map[string]bool{}
	for _, f := range r.SelectedFiles {
		if !safePath(f.Path) || !question.ValidSHA(f.SHA256) || f.DatasetID == "" || selectedPaths[f.Path] {
			return out, ErrInvalid
		}
		selectedPaths[f.Path] = true
		records := map[string]bool{}
		for _, id := range f.RecordIDs {
			if id == "" || records[id] {
				return out, ErrInvalid
			}
			records[id] = true
			s := SourceRecord{SourceIdentity: SourceIdentity{Path: f.Path, FileSHA256: f.SHA256, DatasetID: f.DatasetID, RecordID: id}, MappingIDs: []string{}, Mappings: []contentaudit.MappedSource{}, UsedBy: []contentaudit.ObjectIdentity{}}
			key := SourceKey(s.SourceIdentity)
			if _, ok := sourceIndex[key]; ok {
				return out, ErrInvalid
			}
			sourceIndex[key] = len(out.Sources)
			out.Sources = append(out.Sources, s)
			out.RequiredChecks[key] = []string{"provenance", "version_conditions", "use", "rights"}
		}
	}
	// The initial corpus has 205 records, including 154 that are not referenced by the draft.
	if len(out.Sources) != 205 {
		return out, ErrInvalid
	}
	mappedSources := map[string]int{}
	for _, s := range m.Sources {
		key := SourceKey(SourceIdentity{Path: s.Path, FileSHA256: s.FileSHA256, DatasetID: s.DatasetID, RecordID: s.RecordID})
		idx, ok := sourceIndex[key]
		if !ok || !question.ValidMathID(s.ID) || !(s.Use == "fact_check" || s.Use == "background" || s.Use == "original_derivation") {
			return out, ErrInvalid
		}
		if _, exists := mappedSources[s.ID]; exists {
			return out, ErrInvalid
		}
		mappedSources[s.ID] = idx
		out.Sources[idx].MappingIDs = append(out.Sources[idx].MappingIDs, s.ID)
		out.Sources[idx].Mappings = append(out.Sources[idx].Mappings, s)
	}
	facts, e := contentaudit.CheckSelectedDraft(ctx, in.Selected)
	if e != nil {
		return out, e
	}
	if facts.Path.ID != in.Route.ID || facts.Path.Version != in.Route.Version || in.Route.ID != "elementary-foundations" {
		return out, ErrInvalid
	}
	wantedIDs := map[string]bool{}
	for _, id := range firstIDs {
		wantedIDs[id] = true
	}
	if len(facts.Content.Knowledge) != 30 || len(facts.Content.Units) != 30 || len(facts.Content.Assets) != 9 {
		return out, ErrInvalid
	}
	for _, k := range facts.Content.Knowledge {
		if !wantedIDs[k.ID] {
			return out, ErrInvalid
		}
		delete(wantedIDs, k.ID)
	}
	if len(wantedIDs) != 0 {
		return out, ErrInvalid
	}
	report, e := contentaudit.EvaluateDraft(ctx, contentaudit.Request{Mode: contentaudit.Draft, Route: in.Route, FixtureOnly: in.FixtureOnly, CodeSHA: in.CodeSHA, At: time.Unix(1, 0)}, facts, in.Sources)
	if e != nil {
		return out, e
	}
	if report.Conclusion != contentaudit.DraftReady {
		return out, ErrInvalid
	}
	out.facts = facts
	original := map[string]contentaudit.ObjectIdentity{}
	allLabels := map[string]bool{}
	add := func(o contentaudit.ObjectIdentity, checks []string, mapped bool) error {
		label := identityLabel(o)
		if allLabels[label] || !question.ValidSHA(o.SHA256) {
			return ErrInvalid
		}
		allLabels[label] = true
		out.Objects = append(out.Objects, o)
		out.RequiredChecks[ObjectKey(o)] = checks
		if mapped {
			original[ObjectKey(o)] = o
			out.MappedObjects = append(out.MappedObjects, o)
		}
		return nil
	}
	for _, k := range facts.Content.Knowledge {
		checks := []string{"statement", "conditions", "prerequisites", "objectives"}
		if strings.TrimSpace(k.Proof) != "" {
			checks = append(checks, "proof")
		}
		if e = add(object("knowledge", k.ID, k.Version, content.Digest(k)), checks, true); e != nil {
			return out, e
		}
	}
	for _, u := range facts.Content.Units {
		if e = add(object("unit", u.ID, u.Version, content.Digest(u)), []string{"explanations", "examples", "counterexamples"}, true); e != nil {
			return out, e
		}
	}
	if e = add(object("path", facts.Path.ID, facts.Path.Version, content.Digest(facts.Path)), []string{"closure", "order"}, true); e != nil {
		return out, e
	}
	for _, a := range facts.Content.Assets {
		if e = add(contentaudit.ObjectIdentity{Kind: "asset", ID: a.ID, SHA256: a.SHA256}, []string{"mathematics", "accessibility", "viewports", "rights"}, true); e != nil {
			return out, e
		}
	}
	for _, s := range facts.Sealed {
		for _, t := range s.Package.Templates {
			_, hash, e := question.CanonicalTemplate(t)
			if e != nil {
				return out, e
			}
			if e = add(object("template", t.ID, t.Version, hash), []string{"body", "domain", "generation", "answers", "explanation", "objectives"}, true); e != nil {
				return out, e
			}
		}
		for _, i := range s.Instances {
			if e = add(instanceObject(i.Identity), []string{"body", "answers", "explanation", "objectives"}, i.Origin == "fixed"); e != nil {
				return out, e
			}
			if i.Origin != "fixed" {
				if i.Template == nil {
					return out, ErrInvalid
				}
				out.DerivedInstances = append(out.DerivedInstances, DerivedInstance{Identity: i.Identity, Template: *i.Template, Parameters: i.Parameters, SourceIDs: []string{}})
			}
		}
	}
	for _, r := range facts.QuestionReports {
		for _, n := range r.Coverage {
			if n.Blueprint == nil {
				return out, ErrInvalid
			}
			b := n.Blueprint
			if e = add(object("blueprint", b.ID, b.Version, b.SHA256), []string{"exact_binding", "core_semantics", "five_cover", "exposure_cover"}, true); e != nil {
				return out, e
			}
		}
	}
	// Exact membership, not just minimum acceptance counts, prevents omitted tails or unrelated extras.
	if len(out.MappedObjects) != 574 || len(out.DerivedInstances) != 384 || len(out.Objects) != 958 || len(m.Objects) != len(original) {
		return out, ErrInvalid
	}
	provenance := map[string][]string{}
	bindSources := func(o contentaudit.ObjectIdentity, ids []string) error {
		seen := map[int]bool{}
		for _, id := range ids {
			idx, ok := mappedSources[id]
			if !ok {
				return ErrInvalid
			}
			if !seen[idx] {
				out.Sources[idx].UsedBy = append(out.Sources[idx].UsedBy, o)
				seen[idx] = true
			}
		}
		return nil
	}
	for _, o := range m.Objects {
		id := contentaudit.ObjectIdentity{Kind: o.Kind, ID: o.ID, Version: o.Version, SHA256: o.SHA256}
		key := ObjectKey(id)
		if _, ok := original[key]; !ok || o.Origin != "original" || len(o.SourceIDs) == 0 {
			return out, ErrInvalid
		}
		if _, duplicate := provenance[key]; duplicate {
			return out, ErrInvalid
		}
		ids := append([]string{}, o.SourceIDs...)
		sort.Strings(ids)
		for i := 1; i < len(ids); i++ {
			if ids[i-1] == ids[i] {
				return out, ErrInvalid
			}
		}
		provenance[key] = ids
		if e = bindSources(id, ids); e != nil {
			return out, e
		}
	}
	for i := range out.DerivedInstances {
		d := &out.DerivedInstances[i]
		ids, ok := provenance[ObjectKey(object("template", d.Template.ID, d.Template.Version, d.Template.SHA256))]
		if !ok {
			return out, ErrInvalid
		}
		d.SourceIDs = append([]string{}, ids...)
		if e = bindSources(instanceObject(d.Identity), ids); e != nil {
			return out, e
		}
	}
	sortObjects(out.Objects)
	sortObjects(out.MappedObjects)
	sort.Slice(out.DerivedInstances, func(i, j int) bool {
		return ObjectKey(instanceObject(out.DerivedInstances[i].Identity)) < ObjectKey(instanceObject(out.DerivedInstances[j].Identity))
	})
	for i := range out.Sources {
		sort.Strings(out.Sources[i].MappingIDs)
		sort.Slice(out.Sources[i].Mappings, func(a, b int) bool { return out.Sources[i].Mappings[a].ID < out.Sources[i].Mappings[b].ID })
		sortObjects(out.Sources[i].UsedBy)
	}
	sort.Slice(out.Sources, func(i, j int) bool {
		return SourceKey(out.Sources[i].SourceIdentity) < SourceKey(out.Sources[j].SourceIdentity)
	})
	return out, ctx.Err()
}
