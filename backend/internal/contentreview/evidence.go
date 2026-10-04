package contentreview

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/contentaudit"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
)

type evidenceReader struct {
	ctx   context.Context
	root  string
	cache map[string][]byte
	total int
}

func newEvidenceReader(ctx context.Context, root string) (*evidenceReader, error) {
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	if e := canonicalDir(root); e != nil {
		return nil, e
	}
	return &evidenceReader{ctx: ctx, root: root, cache: map[string][]byte{}}, nil
}
func (r *evidenceReader) read(ref FileRef, limit int) ([]byte, error) {
	if e := r.ctx.Err(); e != nil {
		return nil, e
	}
	if !safePath(ref.Path) || !question.ValidSHA(ref.SHA256) {
		return nil, ErrInvalid
	}
	raw, ok := r.cache[ref.Path]
	if !ok {
		var e error
		raw, e = contentaudit.ReadFileUnder(r.ctx, r.root, ref.Path, limit)
		if e != nil {
			return nil, e
		}
		if r.total+len(raw) > MaxTotalBytes {
			return nil, ErrLimit
		}
		r.total += len(raw)
		r.cache[ref.Path] = raw
	}
	if len(raw) > limit {
		return nil, ErrLimit
	}
	if digestBytes(raw) != ref.SHA256 {
		return nil, ErrInvalid
	}
	return raw, nil
}
func (r *evidenceReader) decode(ref FileRef, limit int, out any) error {
	raw, e := r.read(ref, limit)
	if e != nil {
		return e
	}
	return strict(raw, limit, out)
}
func (r *evidenceReader) signed(ref FileRef) (bool, error) {
	if ref.Path == "" && ref.SHA256 == "" {
		return false, nil
	}
	b, e := r.read(ref, MaxFileBytes)
	if e != nil {
		return false, e
	}
	return len(bytes.TrimSpace(b)) > 0, nil
}
func manifestSHA(m ReviewManifest) string { b, _ := json.Marshal(m); return digestBytes(b) }
func requiredNames(kind string, proof bool) []string {
	switch kind {
	case "knowledge":
		names := []string{"statement", "conditions", "prerequisites", "objectives"}
		if proof {
			names = append(names, "proof")
		}
		return names
	case "unit":
		return []string{"explanations", "examples", "counterexamples"}
	case "path":
		return []string{"closure", "order"}
	case "asset":
		return []string{"mathematics", "accessibility", "viewports", "rights"}
	case "template":
		return []string{"body", "domain", "generation", "answers", "explanation", "objectives"}
	case "instance":
		return []string{"body", "answers", "explanation", "objectives"}
	case "blueprint":
		return []string{"exact_binding", "core_semantics", "five_cover", "exposure_cover"}
	case "source":
		return []string{"provenance", "version_conditions", "use", "rights"}
	}
	return nil
}
func sameNames(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	aa := append([]string{}, a...)
	bb := append([]string{}, b...)
	sort.Strings(aa)
	sort.Strings(bb)
	for i := range aa {
		if aa[i] != bb[i] || i > 0 && aa[i] == aa[i-1] {
			return false
		}
	}
	return true
}
func validateManifest(m ReviewManifest) error {
	if m.SchemaVersion != 1 || len(m.CodeSHA) != 40 || strings.Trim(m.CodeSHA, "0123456789abcdef") != "" || m.CatalogueVersion < 1 || !question.ValidSHA(m.CatalogueSHA256) || m.Route.ID != "elementary-foundations" || m.Route.Version < 1 || !question.ValidSHA(m.Route.SHA256) || !question.ValidSHA(m.SnapshotID) || !question.ValidSHA(m.SourceReportSHA256) || !question.ValidSHA(m.SourceMapSHA256) || !question.ValidSHA(m.InputManifestSHA256) {
		return ErrInvalid
	}
	if !m.FixtureOnly && (m.SnapshotID != SnapshotID || m.SourceReportSHA256 != SourceReportSHA) {
		return ErrInvalid
	}
	if len(m.Objects) != 958 || len(m.Sources) != 205 || len(m.DerivedInstances) != 384 || len(m.RequiredChecks) != 1163 || len(m.Inputs) != 16 || len(m.Files) > MaxFiles {
		return ErrInvalid
	}
	objects := map[string]bool{}
	labels := map[string]bool{}
	subjects := map[string]string{}
	counts := map[string]int{}
	for _, o := range m.Objects {
		if !question.ValidSHA(o.SHA256) || !question.ValidInstanceID(o.ID) || (o.Kind == "asset") != (o.Version == nil) || o.Version != nil && *o.Version < 1 || requiredNames(o.Kind, false) == nil || labels[identityLabel(o)] {
			return ErrInvalid
		}
		labels[identityLabel(o)] = true
		key := ObjectKey(o)
		objects[key] = true
		subjects[key] = o.Kind
		counts[o.Kind]++
	}
	for kind, want := range map[string]int{"knowledge": 30, "unit": 30, "path": 1, "asset": 9, "template": 24, "instance": 834, "blueprint": 30} {
		if counts[kind] != want {
			return ErrInvalid
		}
	}
	mappings := map[string]bool{}
	for _, s := range m.Sources {
		key := SourceKey(s.SourceIdentity)
		if subjects[key] != "" || !safePath(s.Path) || !question.ValidSHA(s.FileSHA256) || s.DatasetID == "" || s.RecordID == "" || len(s.MappingIDs) != len(s.Mappings) {
			return ErrInvalid
		}
		subjects[key] = "source"
		ids := map[string]bool{}
		for _, id := range s.MappingIDs {
			if ids[id] || mappings[id] {
				return ErrInvalid
			}
			ids[id] = true
			mappings[id] = true
		}
		for _, v := range s.Mappings {
			if !ids[v.ID] || v.Path != s.Path || v.FileSHA256 != s.FileSHA256 || v.DatasetID != s.DatasetID || v.RecordID != s.RecordID {
				return ErrInvalid
			}
		}
		for _, o := range s.UsedBy {
			if !objects[ObjectKey(o)] {
				return ErrInvalid
			}
		}
	}
	derived := map[string]bool{}
	for _, d := range m.DerivedInstances {
		key := ObjectKey(instanceObject(d.Identity))
		if !objects[key] || derived[key] || !objects[ObjectKey(object("template", d.Template.ID, d.Template.Version, d.Template.SHA256))] || len(d.SourceIDs) == 0 {
			return ErrInvalid
		}
		derived[key] = true
		for _, id := range d.SourceIDs {
			if !mappings[id] {
				return ErrInvalid
			}
		}
	}
	checked := map[string]bool{}
	for _, s := range m.RequiredChecks {
		kind, ok := subjects[s.Key]
		if !ok || checked[s.Key] {
			return ErrInvalid
		}
		checked[s.Key] = true
		proof := false
		for _, name := range s.Checks {
			if name == "proof" {
				proof = true
			}
		}
		if !sameNames(s.Checks, requiredNames(kind, proof)) {
			return ErrInvalid
		}
	}
	for _, list := range [][]FileEntry{m.Inputs, m.Files} {
		seen := map[string]bool{}
		for _, f := range list {
			if !safePath(f.Path) || seen[f.Path] || f.Bytes < 0 || f.Bytes > MaxFileBytes || !question.ValidSHA(f.SHA256) {
				return ErrInvalid
			}
			seen[f.Path] = true
		}
	}
	return nil
}
func readRegister(r *evidenceReader, m ReviewManifest, ref FileRef) ([]ReviewRow, error) {
	var root ReviewRegister
	if e := r.decode(ref, MaxFileBytes, &root); e != nil {
		return nil, e
	}
	if root.SchemaVersion != 1 || root.ManifestSHA256 != manifestSHA(m) || len(root.Parts) > MaxFiles {
		return nil, ErrInvalid
	}
	expected := map[string][]string{}
	for _, s := range m.RequiredChecks {
		expected[s.Key] = s.Checks
	}
	objects := map[string]contentaudit.ObjectIdentity{}
	for _, o := range m.Objects {
		objects[ObjectKey(o)] = o
	}
	sources := map[string]SourceIdentity{}
	for _, s := range m.Sources {
		sources[SourceKey(s.SourceIdentity)] = s.SourceIdentity
	}
	rows := []ReviewRow{}
	seen := map[string]bool{}
	parts := map[string]bool{}
	for _, ref := range root.Parts {
		if parts[ref.Path] {
			return nil, ErrInvalid
		}
		parts[ref.Path] = true
		var p RegisterPart
		if e := r.decode(ref, MaxFileBytes, &p); e != nil {
			return nil, e
		}
		if p.SchemaVersion != 1 || p.ManifestSHA256 != root.ManifestSHA256 || len(p.Rows) == 0 || len(p.Rows) > 100 {
			return nil, ErrInvalid
		}
		for _, row := range p.Rows {
			want, ok := expected[row.Key]
			if !ok || seen[row.Key] || (row.Object == nil) == (row.Source == nil) {
				return nil, ErrInvalid
			}
			seen[row.Key] = true
			if row.Object != nil {
				o, ok := objects[row.Key]
				if !ok || content.Digest(o) != content.Digest(*row.Object) {
					return nil, ErrInvalid
				}
			} else {
				s, ok := sources[row.Key]
				if !ok || content.Digest(s) != content.Digest(*row.Source) {
					return nil, ErrInvalid
				}
			}
			names := []string{}
			for _, c := range row.Checks {
				if !(c.Status == "unreviewed" || c.Status == "passed" || c.Status == "returned") {
					return nil, ErrInvalid
				}
				names = append(names, c.Name)
			}
			if !sameNames(names, want) {
				return nil, ErrInvalid
			}
			rows = append(rows, row)
		}
	}
	if len(rows) != len(expected) {
		return nil, ErrInvalid
	}
	return rows, nil
}

type importEvidence struct {
	knowledge  publication.DraftInput
	questions  map[string]question.DraftInput
	references question.ReferenceSnapshot
}

func readImports(r *evidenceReader, m ReviewManifest) (importEvidence, error) {
	out := importEvidence{questions: map[string]question.DraftInput{}, references: question.ReferenceSnapshot{CatalogueVersion: m.CatalogueVersion, CatalogueSHA256: m.CatalogueSHA256, Knowledge: []question.FixedKnowledge{}, Units: []question.FixedUnit{}, Assets: []content.AssetView{}}}
	known := false
	for _, f := range m.Files {
		raw, e := r.read(FileRef{f.Path, f.SHA256}, MaxFileBytes)
		if e != nil {
			return out, e
		}
		if len(raw) != f.Bytes {
			return out, ErrInvalid
		}
		if f.Path == "imports/knowledge.json" {
			if known {
				return out, ErrInvalid
			}
			known = true
			if e = strict(raw, MaxFileBytes, &out.knowledge); e != nil {
				return out, e
			}
			if _, e = publication.DraftAssets(out.knowledge); e != nil {
				return out, e
			}
		}
		if strings.HasPrefix(f.Path, "imports/questions/") {
			q, e := question.DecodeDraft(bytes.NewReader(raw))
			if e != nil {
				return out, e
			}
			if f.Path != "imports/questions/"+q.QuestionPackage.ID+".json" {
				return out, ErrInvalid
			}
			if _, ok := out.questions[q.QuestionPackage.ID]; ok {
				return out, ErrInvalid
			}
			out.questions[q.QuestionPackage.ID] = q
		}
	}
	if !known || len(out.questions) != 5 || out.knowledge.CatalogueVersion != m.CatalogueVersion {
		return out, ErrInvalid
	}
	for _, k := range out.knowledge.Package.Knowledge {
		out.references.Knowledge = append(out.references.Knowledge, question.FixedKnowledge{Identity: question.Identity{ID: k.ID, Version: k.Version, SHA256: content.Digest(k)}, Title: k.Title, TitleZh: k.TitleZh, Objectives: k.Objectives})
	}
	for _, u := range out.knowledge.Package.Units {
		out.references.Units = append(out.references.Units, question.FixedUnit{Identity: question.Identity{ID: u.ID, Version: u.Version, SHA256: content.Digest(u)}, Knowledge: u.Knowledge, AssetIDs: u.AssetIDs})
	}
	for _, a := range out.knowledge.Package.Assets {
		out.references.Assets = append(out.references.Assets, content.AssetView{ID: a.ID, SHA256: a.SHA256, Author: a.Author, License: a.License, Attribution: a.Attribution, Knowledge: a.Knowledge})
	}
	return out, nil
}

type bindingResult struct {
	ready        bool
	returned     bool
	attestations []contentaudit.ReviewAttestation
}

func verifyReviewBindings(ctx context.Context, r *evidenceReader, m ReviewManifest, rows []ReviewRow, bindings []FrozenBinding, imports importEvidence, sourceReviewer string) (bindingResult, error) {
	out := bindingResult{ready: true, attestations: []contentaudit.ReviewAttestation{}}
	if len(bindings) > 20 {
		return out, ErrLimit
	}
	expected := map[string]contentaudit.ObjectIdentity{}
	for _, o := range m.Objects {
		expected[ObjectKey(o)] = o
	}
	reviewers := map[string]string{}
	decisions := map[string]bool{}
	submissions := map[string]bool{}
	knowledgeSeen := false
	packages := map[string]bool{}
	add := func(o contentaudit.ObjectIdentity, reviewer string) error {
		key := ObjectKey(o)
		if _, ok := expected[key]; !ok || reviewers[key] != "" {
			return ErrInvalid
		}
		reviewers[key] = reviewer
		return nil
	}
	decision := func(sub, id, reviewer, frozen, approved string, authors []string) error {
		if !publication.ValidID(sub) || !publication.ValidID(id) || !publication.ValidID(reviewer) || decisions[id] || submissions[sub] || approved != "approve" || !question.ValidSHA(frozen) || len(authors) == 0 {
			return ErrInvalid
		}
		seen := map[string]bool{}
		for _, author := range authors {
			if !publication.ValidID(author) || seen[author] || author == reviewer {
				return ErrInvalid
			}
			seen[author] = true
		}
		decisions[id] = true
		submissions[sub] = true
		return nil
	}
	for _, b := range bindings {
		if e := ctx.Err(); e != nil {
			return out, e
		}
		signed, e := r.signed(b.Attestation)
		if e != nil {
			return out, e
		}
		if !b.IndependenceVerified || !signed {
			out.ready = false
		}
		switch b.Space {
		case "knowledge":
			if b.Archive != nil || knowledgeSeen {
				return out, fmt.Errorf("%w: binding check B01", ErrInvalid)
			}
			knowledgeSeen = true
			var v publication.SubmissionView
			if e = r.decode(b.Submission, MaxFileBytes, &v); e != nil {
				return out, e
			}
			if v.Review == nil || v.Status != "approved" {
				out.ready = false
				continue
			}
			f := v.Frozen
			d := v.Review
			hash, e := publication.FrozenDigest(f)
			if e != nil {
				return out, e
			}
			if hash != f.FrozenDigest || d.SubmissionID != v.ID || d.FrozenDigest != hash || f.CatalogueVersion != m.CatalogueVersion || f.CatalogueSHA256 != m.CatalogueSHA256 || f.LegacyUnattributed || content.Digest(f.Package) != content.Digest(imports.knowledge.Package) || content.Digest(f.SourceMap) != content.Digest(imports.knowledge.SourceMap) || content.Digest(f.Assets) != content.Digest(imports.references.Assets) {
				return out, fmt.Errorf("%w: binding check B02", ErrInvalid)
			}
			if e = publication.ValidateReviewInput(publication.ReviewInput{Decision: d.Decision, Checks: d.Checks, IndependenceNote: d.IndependenceNote, Note: d.Note}); e != nil {
				return out, fmt.Errorf("%w: binding check B03", ErrInvalid)
			}
			if e = decision(v.ID, d.ID, d.ReviewerID, hash, d.Decision, f.AuthorIDs); e != nil {
				return out, e
			}
			for _, k := range f.Package.Knowledge {
				if e = add(object("knowledge", k.ID, k.Version, content.Digest(k)), d.ReviewerID); e != nil {
					return out, e
				}
				for _, s := range m.RequiredChecks {
					if s.Key == ObjectKey(object("knowledge", k.ID, k.Version, content.Digest(k))) && !sameNames(s.Checks, requiredNames("knowledge", strings.TrimSpace(k.Proof) != "")) {
						return out, fmt.Errorf("%w: binding check B04", ErrInvalid)
					}
				}
			}
			for _, u := range f.Package.Units {
				if e = add(object("unit", u.ID, u.Version, content.Digest(u)), d.ReviewerID); e != nil {
					return out, e
				}
			}
			for _, p := range f.Package.Paths {
				if p.ID != m.Route.ID || p.Version != m.Route.Version || content.Digest(p) != m.Route.SHA256 {
					return out, fmt.Errorf("%w: binding check B05", ErrInvalid)
				}
				if e = add(object("path", p.ID, p.Version, content.Digest(p)), d.ReviewerID); e != nil {
					return out, e
				}
			}
			for _, a := range f.Package.Assets {
				if e = add(contentauditObjectAsset(a), d.ReviewerID); e != nil {
					return out, e
				}
			}
			out.attestations = append(out.attestations, contentaudit.ReviewAttestation{DecisionID: d.ID, IndependenceVerified: b.IndependenceVerified && signed})
		case "questions":
			if b.Archive == nil {
				return out, fmt.Errorf("%w: binding check B06", ErrInvalid)
			}
			var v question.SubmissionView
			if e = r.decode(b.Submission, question.MaxResponseBytes, &v); e != nil {
				return out, e
			}
			raw, e := r.read(*b.Archive, question.MaxEnvelopeBytes)
			if e != nil {
				return out, e
			}
			a, e := question.DecodeArchive(bytes.NewReader(raw))
			if e != nil {
				return out, e
			}
			if v.Review == nil || v.Status != "approved" {
				out.ready = false
				continue
			}
			f := v.Frozen
			d := v.Review
			input, ok := imports.questions[f.QuestionPackage.ID]
			if !ok || packages[f.QuestionPackage.ID] {
				return out, fmt.Errorf("%w: binding check B07", ErrInvalid)
			}
			packages[f.QuestionPackage.ID] = true
			expectedSeal, expectedGate, e := question.ValidateAndSeal(ctx, input, imports.references)
			expectedSHA := expectedSeal.PackageSHA
			if !expectedGate.ReadyToSubmit {
				return out, ErrInvalid
			}
			if e != nil {
				return out, e
			}
			_, actualSHA, e := question.CanonicalPackage(f.QuestionPackage)
			if e != nil || actualSHA != expectedSHA || a.PackageSHA != expectedSHA || f.CatalogueVersion != m.CatalogueVersion || f.CatalogueSHA256 != m.CatalogueSHA256 || f.LegacyUnattributed || a.SourceResponsibility.LegacyUnattributed || !sameNames(f.AuthorIDs, a.SourceResponsibility.AuthorIDs) || content.Digest(f.SourceMap) != content.Digest(input.SourceMap) || content.Digest(a.Envelope.SourceMap) != content.Digest(input.SourceMap) {
				return out, fmt.Errorf("%w: binding check B08", ErrInvalid)
			}
			sealed, gate, e := question.ValidateArchive(ctx, a, imports.references)
			if e != nil {
				return out, e
			}
			if !gate.ReadyToSubmit || content.Digest(f.Resolved) != content.Digest(sealed.Resolved) || content.Digest(f.Objectives) != content.Digest(sealed.Objectives) || content.Digest(f.Generation) != content.Digest(sealed.Generation) || content.Digest(f.Coverage) != content.Digest(gate.Coverage) || content.Digest(f.GeneratorVersions) != content.Digest(a.GeneratorVersions) || content.Digest(f.VerifierVersions) != content.Digest(a.VerifierVersions) {
				return out, fmt.Errorf("%w: binding check B09", ErrInvalid)
			}
			byID := map[string]question.Instance{}
			for _, i := range a.Instances {
				if _, ok := byID[i.Identity.ID]; ok {
					return out, fmt.Errorf("%w: binding check B10", ErrInvalid)
				}
				byID[i.Identity.ID] = i
			}
			ordered := []question.Instance{}
			if len(f.InstanceIdentities) != len(a.Instances) {
				return out, fmt.Errorf("%w: binding check B11", ErrInvalid)
			}
			for _, id := range f.InstanceIdentities {
				i, ok := byID[id.ID]
				if !ok || i.Identity != id {
					return out, fmt.Errorf("%w: binding check B12", ErrInvalid)
				}
				delete(byID, id.ID)
				ordered = append(ordered, i)
			}
			if len(byID) != 0 {
				return out, fmt.Errorf("%w: binding check B13", ErrInvalid)
			}
			_, hash, e := question.CanonicalFrozen(f, ordered)
			if e != nil {
				return out, e
			}
			if hash != f.FrozenDigest || d.FrozenDigest != hash || d.SubmissionID != v.ID {
				return out, fmt.Errorf("%w: binding check B14", ErrInvalid)
			}
			if e = question.ValidateReviewInput(question.ReviewInput{Decision: d.Decision, Checks: d.Checks, IndependenceNote: d.IndependenceNote, GenerationNote: d.GenerationNote, Note: d.Note}, len(f.QuestionPackage.Templates) > 0); e != nil {
				return out, fmt.Errorf("%w: binding check B15", ErrInvalid)
			}
			if e = decision(v.ID, d.ID, d.ReviewerID, hash, d.Decision, f.AuthorIDs); e != nil {
				return out, e
			}
			for _, t := range sealed.Package.Templates {
				_, hash, e := question.CanonicalTemplate(t)
				if e != nil {
					return out, e
				}
				if e = add(object("template", t.ID, t.Version, hash), d.ReviewerID); e != nil {
					return out, e
				}
			}
			for _, i := range sealed.Instances {
				if e = add(instanceObject(i.Identity), d.ReviewerID); e != nil {
					return out, e
				}
				if i.Origin != "fixed" {
					match := false
					for _, index := range m.DerivedInstances {
						if index.Identity == i.Identity && i.Template != nil && index.Template == *i.Template && content.Digest(index.Parameters) == content.Digest(i.Parameters) {
							match = true
							break
						}
					}
					if !match {
						return out, fmt.Errorf("%w: binding check B16", ErrInvalid)
					}
				}
			}
			for _, n := range gate.Coverage {
				if n.Blueprint == nil {
					return out, fmt.Errorf("%w: binding check B17", ErrInvalid)
				}
				id := n.Blueprint
				if e = add(object("blueprint", id.ID, id.Version, id.SHA256), d.ReviewerID); e != nil {
					return out, e
				}
			}
			out.attestations = append(out.attestations, contentaudit.ReviewAttestation{DecisionID: d.ID, IndependenceVerified: b.IndependenceVerified && signed})
		default:
			return out, fmt.Errorf("%w: binding check B18", ErrInvalid)
		}
	}
	if len(reviewers) != len(expected) || !knowledgeSeen || len(packages) != len(imports.questions) {
		out.ready = false
	}
	for _, row := range rows {
		for _, check := range row.Checks {
			if check.Status == "returned" {
				out.returned = true
				out.ready = false
				continue
			}
			if check.Status != "passed" || strings.TrimSpace(check.Basis) == "" || strings.TrimSpace(check.ReviewerRef) == "" {
				out.ready = false
				continue
			}
			want := sourceReviewer
			if row.Object != nil {
				want = reviewers[row.Key]
			}
			if want == "" {
				out.ready = false
				continue
			}
			if check.ReviewerRef != want {
				return out, fmt.Errorf("%w: binding check B19", ErrInvalid)
			}
		}
	}
	sort.Slice(out.attestations, func(i, j int) bool { return out.attestations[i].DecisionID < out.attestations[j].DecisionID })
	return out, nil
}
func VerifyEvidence(ctx context.Context, root string, m ReviewManifest, input EvidenceInput) (Verification, error) {
	out := Verification{SchemaVersion: 1, Conclusion: "awaiting_review", FixtureOnly: m.FixtureOnly, ManifestSHA256: manifestSHA(m), Reasons: []string{}, Files: []ExportFile{}}
	if e := validateManifest(m); e != nil {
		return out, e
	}
	if input.SchemaVersion != 1 {
		return out, ErrInvalid
	}
	reader, e := newEvidenceReader(ctx, root)
	if e != nil {
		return out, e
	}
	var release ReleaseContext
	if e = reader.decode(input.ReleaseContext, MaxFileBytes, &release); e != nil {
		return out, e
	}
	if release.SchemaVersion != 1 || release.ManifestSHA256 != out.ManifestSHA256 {
		return out, ErrInvalid
	}
	imports, e := readImports(reader, m)
	if e != nil {
		return out, e
	}
	rows, e := readRegister(reader, m, input.ReviewRegister)
	if e != nil {
		return out, e
	}
	result, e := verifyReviewBindings(ctx, reader, m, rows, input.Bindings, imports, release.AttestedBy)
	if e != nil {
		return out, e
	}
	out.ReviewComplete = result.ready
	if result.returned {
		out.Conclusion = "not_ready"
		out.Reasons = append(out.Reasons, "REVIEW_RETURNED")
	} else if !result.ready {
		out.Reasons = append(out.Reasons, "REVIEW_PENDING")
	} else {
		out.Reasons = append(out.Reasons, "LEARNING_PENDING")
	}
	return out, ctx.Err()
}
