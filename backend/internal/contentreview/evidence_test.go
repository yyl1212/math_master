package contentreview

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/contentaudit"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const fixtureAuthor = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
const fixtureReviewer = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"

type evidenceFixtureData struct {
	Root     string
	Manifest ReviewManifest
	Input    EvidenceInput
	Release  ReleaseContext
	Rows     []ReviewRow
}

func evidenceSave(t *testing.T, root, p string, v any) FileRef {
	t.Helper()
	b := fixtureJSON(t, v)
	fixtureWrite(t, filepath.Join(root, p), b)
	return FileRef{p, sha(b)}
}
func evidenceRead(t *testing.T, f evidenceFixtureData, ref FileRef, v any) {
	t.Helper()
	b, e := os.ReadFile(filepath.Join(f.Root, ref.Path))
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(b, v); e != nil {
		t.Fatal(e)
	}
}
func evidenceFixture(t *testing.T) evidenceFixtureData {
	t.Helper()
	in := reviewFixture(t)
	scope, e := BuildScope(context.Background(), in)
	if e != nil {
		t.Fatal(e)
	}
	imports, e := BuildImports(context.Background(), in, scope)
	if e != nil {
		t.Fatal(e)
	}
	bundle, e := Prepare(context.Background(), in)
	if e != nil {
		t.Fatal(e)
	}
	f := evidenceFixtureData{Root: outputRoot(t), Manifest: bundle.Manifest, Rows: registerRows(t, bundle)}
	for _, file := range bundle.Files {
		fixtureWrite(t, filepath.Join(f.Root, file.Path), file.Bytes)
	}
	sig := evidenceSave(t, f.Root, "attestation.json", map[string]string{"attestedBy": fixtureReviewer, "note": "Technical fixture only: independent author and reviewer binding checked."})
	build := evidenceSave(t, f.Root, "build.json", map[string]string{"codeSHA": in.CodeSHA, "note": "Isolated technical test build, not deployment."})
	f.Release = ReleaseContext{SchemaVersion: 1, ReleaseIdentity: ReleaseIdentity{CodeSHA: in.CodeSHA, CatalogueVersion: f.Manifest.CatalogueVersion, CatalogueSHA256: f.Manifest.CatalogueSHA256, Route: f.Manifest.Route, KnowledgeHead: contentaudit.Head{ID: "11111111-1111-4111-8111-111111111111", SHA256: strings.Repeat("1", 64)}, QuestionHead: contentaudit.Head{ID: "22222222-2222-4222-8222-222222222222", SHA256: strings.Repeat("2", 64)}, ManifestSHA256: sha(fixtureJSON(t, f.Manifest)), FixtureOnly: true}, BuildRecord: build, AttestedBy: fixtureReviewer, Attestation: sig}
	f.Input = EvidenceInput{SchemaVersion: 1, ReleaseContext: evidenceSave(t, f.Root, "release-context.json", f.Release), Bindings: []FrozenBinding{}, LearningChecks: []LearningFile{}}
	k := publication.FrozenBody{CatalogueVersion: f.Manifest.CatalogueVersion, CatalogueSHA256: f.Manifest.CatalogueSHA256, Package: imports.Knowledge.Package, SourceMap: imports.Knowledge.SourceMap, AuthorIDs: []string{fixtureAuthor}, Assets: scope.facts.References.Assets}
	k.FrozenDigest, e = publication.FrozenDigest(k)
	if e != nil {
		t.Fatal(e)
	}
	kv := publication.SubmissionView{ID: "33333333-3333-4333-8333-333333333333", Status: "approved", OwnerID: fixtureAuthor, WorkspaceID: "77777777-7777-4777-8777-777777777777", Revision: 1, CreatedAt: "2026-10-04T00:00:00Z", Gate: publication.GateReport{StructuralErrors: []content.Issue{}, CompletenessErrors: []content.Issue{}, HumanReviewRequirements: []content.Issue{}, ReadyToSubmit: true}, Frozen: k, Review: &publication.ReviewDecision{ID: "44444444-4444-4444-8444-444444444444", SubmissionID: "33333333-3333-4333-8333-333333333333", ReviewerID: fixtureReviewer, FrozenDigest: k.FrozenDigest, Decision: "approve", Checks: publication.ReviewChecks{Mathematics: true, Explanations: true, Relationships: true, Sources: true, Illustrations: true}, IndependenceNote: "Isolated independent technical reviewer account.", Note: "The exact complete technical fixture was checked."}}
	f.Input.Bindings = append(f.Input.Bindings, FrozenBinding{Space: "knowledge", Submission: evidenceSave(t, f.Root, "knowledge-submission.json", kv), IndependenceVerified: true, Attestation: sig})
	for p, s := range scope.facts.Sealed {
		var input question.DraftInput
		for _, q := range imports.Questions {
			if q.QuestionPackage.ID == s.Package.ID {
				input = q
			}
		}
		ids := []question.Identity{}
		for _, i := range s.Instances {
			ids = append(ids, i.Identity)
		}
		g, v := question.UsedEngineVersions(s.Package, s.Instances)
		frozen := question.FrozenBody{CatalogueVersion: f.Manifest.CatalogueVersion, CatalogueSHA256: f.Manifest.CatalogueSHA256, QuestionPackage: s.Package, SourceMap: input.SourceMap, AuthorIDs: []string{fixtureAuthor}, Resolved: s.Resolved, Objectives: s.Objectives, Generation: s.Generation, InstanceIdentities: ids, Coverage: scope.facts.QuestionReports[p].Coverage, GeneratorVersions: g, VerifierVersions: v}
		_, frozen.FrozenDigest, e = question.CanonicalFrozen(frozen, s.Instances)
		if e != nil {
			t.Fatal(e)
		}
		id := fmt.Sprintf("50000000-0000-4000-8000-%012d", p+1)
		decision := fmt.Sprintf("60000000-0000-4000-8000-%012d", p+1)
		qv := question.SubmissionView{ID: id, Status: "approved", OwnerID: fixtureAuthor, WorkspaceID: "88888888-8888-4888-8888-888888888888", Revision: 1, CreatedAt: "2026-10-04T00:00:00Z", Gate: scope.facts.QuestionReports[p], Frozen: frozen, Review: &question.ReviewDecision{ID: decision, SubmissionID: id, ReviewerID: fixtureReviewer, FrozenDigest: frozen.FrozenDigest, Decision: "approve", Checks: question.ReviewChecks{Mathematics: true, Explanations: true, Objectives: true, Sources: true, Illustrations: true, Generation: true}, IndependenceNote: "Isolated independent technical reviewer account.", GenerationNote: "All finite instances checked in this technical fixture.", Note: "The complete exact technical question fixture was checked."}}
		archive := question.Archive{Envelope: input, PackageSHA: s.PackageSHA, Instances: s.Instances, GeneratorVersions: g, VerifierVersions: v, SourceResponsibility: question.SourceResponsibility{AuthorIDs: []string{fixtureAuthor}}}
		ar := evidenceSave(t, f.Root, fmt.Sprintf("archive-%d.json", p), archive)
		f.Input.Bindings = append(f.Input.Bindings, FrozenBinding{Space: "questions", Submission: evidenceSave(t, f.Root, fmt.Sprintf("submission-%d.json", p), qv), Archive: &ar, IndependenceVerified: true, Attestation: sig})
	}
	for i := range f.Rows {
		for c := range f.Rows[i].Checks {
			f.Rows[i].Checks[c].Status = "passed"
			f.Rows[i].Checks[c].Basis = "Full independent technical fixture check, not a real approval."
			f.Rows[i].Checks[c].ReviewerRef = fixtureReviewer
		}
	}
	saveFinalRegister(t, &f)
	return f
}
func saveFinalRegister(t *testing.T, f *evidenceFixtureData) {
	t.Helper()
	reg := ReviewRegister{SchemaVersion: 1, ManifestSHA256: sha(fixtureJSON(t, f.Manifest)), Parts: []FileRef{}}
	for start := 0; start < len(f.Rows); start += 100 {
		end := min(start+100, len(f.Rows))
		ref := evidenceSave(t, f.Root, fmt.Sprintf("register-final/%03d.json", len(reg.Parts)+1), RegisterPart{1, reg.ManifestSHA256, f.Rows[start:end]})
		reg.Parts = append(reg.Parts, ref)
	}
	f.Input.ReviewRegister = evidenceSave(t, f.Root, "review-register.final.json", reg)
}
func TestReviewFrozenBinding(t *testing.T) {
	f := evidenceFixture(t)
	for _, binding := range f.Input.Bindings {
		if binding.Archive != nil {
			var a question.Archive
			evidenceRead(t, f, *binding.Archive, &a)
			slices.Reverse(a.Instances)
			*binding.Archive = evidenceSave(t, f.Root, binding.Archive.Path, a)
		}
	}
	got, e := VerifyEvidence(context.Background(), f.Root, f.Manifest, f.Input)
	if e != nil {
		t.Fatal("valid frozen/export ordering", e)
	}
	if !got.ReviewComplete {
		t.Fatal("complete frozen set not verified")
	}
	if !got.FixtureOnly {
		t.Fatal("fixture flag lost")
	}
	cases := []struct {
		name   string
		mutate func(*evidenceFixtureData)
	}{
		{"wrong-frozen", func(f *evidenceFixtureData) {
			var v publication.SubmissionView
			evidenceRead(t, *f, f.Input.Bindings[0].Submission, &v)
			v.Frozen.FrozenDigest = sha(fixtureJSON(t, v.Frozen))
			f.Input.Bindings[0].Submission = evidenceSave(t, f.Root, "knowledge-submission.json", v)
		}},
		{"wrong-catalogue", func(f *evidenceFixtureData) {
			var v question.SubmissionView
			evidenceRead(t, *f, f.Input.Bindings[1].Submission, &v)
			v.Frozen.CatalogueVersion++
			f.Input.Bindings[1].Submission = evidenceSave(t, f.Root, "submission-0.json", v)
		}},
		{"reviewer-author", func(f *evidenceFixtureData) {
			var v publication.SubmissionView
			evidenceRead(t, *f, f.Input.Bindings[0].Submission, &v)
			v.Review.ReviewerID = fixtureAuthor
			f.Input.Bindings[0].Submission = evidenceSave(t, f.Root, "knowledge-submission.json", v)
		}},
		{"wrong-check", func(f *evidenceFixtureData) {
			var v question.SubmissionView
			evidenceRead(t, *f, f.Input.Bindings[1].Submission, &v)
			v.Review.Checks.Generation = false
			f.Input.Bindings[1].Submission = evidenceSave(t, f.Root, "submission-0.json", v)
		}},
		{"source-link-change", func(f *evidenceFixtureData) {
			var a question.Archive
			evidenceRead(t, *f, *f.Input.Bindings[1].Archive, &a)
			a.Envelope.SourceMap[0].Note += " altered"
			*f.Input.Bindings[1].Archive = evidenceSave(t, f.Root, f.Input.Bindings[1].Archive.Path, a)
		}},
		{"author-change", func(f *evidenceFixtureData) {
			var a question.Archive
			evidenceRead(t, *f, *f.Input.Bindings[1].Archive, &a)
			a.SourceResponsibility.AuthorIDs = []string{fixtureReviewer}
			*f.Input.Bindings[1].Archive = evidenceSave(t, f.Root, f.Input.Bindings[1].Archive.Path, a)
		}},
		{"missing-instance", func(f *evidenceFixtureData) {
			var a question.Archive
			evidenceRead(t, *f, *f.Input.Bindings[1].Archive, &a)
			a.Instances = a.Instances[:len(a.Instances)-1]
			*f.Input.Bindings[1].Archive = evidenceSave(t, f.Root, f.Input.Bindings[1].Archive.Path, a)
		}},
		{"duplicate-instance", func(f *evidenceFixtureData) {
			var a question.Archive
			evidenceRead(t, *f, *f.Input.Bindings[1].Archive, &a)
			a.Instances = append(a.Instances, a.Instances[0])
			*f.Input.Bindings[1].Archive = evidenceSave(t, f.Root, f.Input.Bindings[1].Archive.Path, a)
		}},
		{"body-change", func(f *evidenceFixtureData) {
			var a question.Archive
			evidenceRead(t, *f, *f.Input.Bindings[1].Archive, &a)
			a.Instances[0].Body.Prompt += " changed"
			*f.Input.Bindings[1].Archive = evidenceSave(t, f.Root, f.Input.Bindings[1].Archive.Path, a)
		}},
		{"duplicate-binding", func(f *evidenceFixtureData) { f.Input.Bindings = append(f.Input.Bindings, f.Input.Bindings[0]) }},
		{"missing-row", func(f *evidenceFixtureData) { f.Rows = f.Rows[:len(f.Rows)-1]; saveFinalRegister(t, f) }},
		{"missing-check", func(f *evidenceFixtureData) {
			f.Rows[0].Checks = f.Rows[0].Checks[:len(f.Rows[0].Checks)-1]
			saveFinalRegister(t, f)
		}},
		{"wrong-row-reviewer", func(f *evidenceFixtureData) { f.Rows[0].Checks[0].ReviewerRef = fixtureAuthor; saveFinalRegister(t, f) }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := evidenceFixture(t)
			c.mutate(&f)
			got, e := VerifyEvidence(context.Background(), f.Root, f.Manifest, f.Input)
			if e == nil && got.ReviewComplete {
				t.Fatal("invalid frozen binding accepted")
			}
			if got.Evidence != nil {
				t.Fatal("invalid evidence produced")
			}
		})
	}
}
func TestReviewIndependencePending(t *testing.T) {
	for _, name := range []string{"unsigned", "unverified", "unreviewed", "returned", "empty-basis", "empty-reviewer", "missing-binding"} {
		t.Run(name, func(t *testing.T) {
			f := evidenceFixture(t)
			switch name {
			case "unsigned":
				f.Input.Bindings[0].Attestation = FileRef{}
			case "unverified":
				f.Input.Bindings[0].IndependenceVerified = false
			case "unreviewed":
				f.Rows[0].Checks[0] = ReviewCheck{Name: f.Rows[0].Checks[0].Name, Status: "unreviewed"}
			case "returned":
				f.Rows[0].Checks[0].Status = "returned"
				f.Rows[0].Checks[0].Issue = "Exact object fixture issue location"
			case "empty-basis":
				f.Rows[0].Checks[0].Basis = ""
			case "empty-reviewer":
				f.Rows[0].Checks[0].ReviewerRef = ""
			case "missing-binding":
				f.Input.Bindings = f.Input.Bindings[1:]
			}
			saveFinalRegister(t, &f)
			got, e := VerifyEvidence(context.Background(), f.Root, f.Manifest, f.Input)
			if e != nil {
				t.Fatal(e)
			}
			if got.Conclusion == "evidence_ready" || got.Evidence != nil {
				t.Fatal("未复核被当作正式证据")
			}
		})
	}
}

func TestReviewRegisterEvidenceJSONBudget(t *testing.T) {
	type metadata struct {
		SchemaVersion int `json:"schemaVersion"`
	}
	raw := []byte(`{"schemaVersion":1}`)
	raw = append(raw, bytes.Repeat([]byte(" "), MaxFileBytes-len(raw))...)
	var value metadata
	if e := question.DecodeOperationalJSON(bytes.NewReader(raw), MaxFileBytes, &value); e != nil {
		t.Fatal("private 8MiB boundary", e)
	}
	if e := question.DecodeStrictJSON(bytes.NewReader(raw), MaxFileBytes, &value); e == nil {
		t.Fatal("existing 4MiB envelope changed")
	}
	if e := question.DecodeOperationalJSON(bytes.NewReader(append(raw, ' ')), MaxFileBytes, &value); e == nil {
		t.Fatal("limit+1 accepted")
	}
	for _, s := range []string{`{"schemaVersion":2147483648}`, `{"schemaVersion":1,"schemaVersion":2}`, `{"schemaVersion":1,"extra":true}`, `{"schemaVersion":"\u0000"}`} {
		if question.DecodeOperationalJSON(strings.NewReader(s), MaxFileBytes, &value) == nil {
			t.Fatal("invalid private metadata accepted")
		}
	}
}
