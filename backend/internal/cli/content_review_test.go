package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/contentaudit"
	"github.com/yyl1212/math_master/backend/internal/contentreview"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

func crSha(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func crFixtureWrite(t *testing.T, p string, b []byte) {
	t.Helper()
	if e := os.MkdirAll(filepath.Dir(p), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(p, b, 0600); e != nil {
		t.Fatal(e)
	}
}
func crFixtureJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, e := json.Marshal(v)
	if e != nil {
		t.Fatal(e)
	}
	return b
}

// Uses original application mathematics and a synthetic provenance corpus, never book prose.
func crReviewFixture(t *testing.T) (contentreview.PrepareInput, string) {
	t.Helper()
	root, e := filepath.Abs("../../..")
	if e != nil {
		t.Fatal(e)
	}
	root, e = filepath.EvalSymlinks(root)
	if e != nil {
		t.Fatal(e)
	}
	raw, e := os.ReadFile(filepath.Join(root, "content/review/elementary-foundations.input.v1.json"))
	if e != nil {
		t.Fatal(e)
	}
	var manifest contentreview.InputManifest
	if e = json.Unmarshal(raw, &manifest); e != nil {
		t.Fatal(e)
	}
	selected, e := contentaudit.LoadSelectedDraft(context.Background(), root, manifest.Selection())
	if e != nil {
		t.Fatal("selected", e)
	}
	mapping, e := contentaudit.ReadSourceMap(filepath.Join(root, "content/source-maps/elementary-foundations.v1.json"))
	if e != nil {
		t.Fatal("map", e)
	}
	ids := map[string]bool{}
	for _, s := range mapping.Sources {
		ids[s.RecordID] = true
	}
	for i := 1; len(ids) < 205; i++ {
		ids[fmt.Sprintf("review-unused-%03d", i)] = true
	}
	recordIDs := []string{}
	for id := range ids {
		recordIDs = append(recordIDs, id)
	}
	sort.Strings(recordIDs)
	records := []map[string]string{}
	for _, id := range recordIDs {
		records = append(records, map[string]string{"id": id})
	}
	corpus := crFixtureJSON(t, map[string]any{"dataset_id": mapping.Sources[0].DatasetID, "knowledge_points": records})
	sourcePath := mapping.Sources[0].Path
	files := []map[string]any{{"path": sourcePath, "sizeBytes": len(corpus), "sha256": crSha(corpus)}}
	snapshotSHA := crSha(crFixtureJSON(t, files))
	snapshot, e := filepath.EvalSymlinks(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	crFixtureWrite(t, filepath.Join(snapshot, "files", sourcePath), corpus)
	crFixtureWrite(t, filepath.Join(snapshot, "manifest.json"), crFixtureJSON(t, map[string]any{"schemaVersion": 1, "snapshotId": snapshotSHA, "files": files, "createdAt": nil, "sourceRoot": nil, "packageCount": nil, "primaryFiles": nil, "changes": nil, "sourceIndexMismatches": nil}))
	report := contentaudit.SourceReport{SchemaVersion: 1, PolicyVersion: 1, SnapshotID: snapshotSHA, SelectedFiles: []contentaudit.SelectedSource{{Path: sourcePath, SHA256: crSha(corpus), DatasetID: mapping.Sources[0].DatasetID, RecordIDs: recordIDs}}, Issues: []contentaudit.SourceIssue{}, Ready: true}
	reportRaw := crFixtureJSON(t, report)
	mapping.SnapshotID = snapshotSHA
	mapping.SourceReportSHA256 = crSha(reportRaw)
	for i := range mapping.Sources {
		mapping.Sources[i].FileSHA256 = crSha(corpus)
	}
	mapRaw := crFixtureJSON(t, mapping)
	sources, e := contentaudit.LoadSourcesFromBytes(context.Background(), snapshot, reportRaw, mapRaw)
	if e != nil {
		t.Fatal("sources", e)
	}
	return contentreview.PrepareInput{CodeSHA: strings.Repeat("a", 40), Route: content.VersionRef{ID: "elementary-foundations", Version: 1}, Manifest: manifest, ManifestRaw: raw, Selected: selected, Sources: sources, SourceReportRaw: reportRaw, SourceMapRaw: mapRaw, FixtureOnly: true}, snapshot
}

const crFixtureAuthor = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
const crFixtureReviewer = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"

type crEvidenceFixtureData struct {
	Root     string
	Manifest contentreview.ReviewManifest
	Input    contentreview.EvidenceInput
	Release  contentreview.ReleaseContext
	Rows     []contentreview.ReviewRow
}

func crEvidenceSave(t *testing.T, root, p string, v any) contentreview.FileRef {
	t.Helper()
	b := crFixtureJSON(t, v)
	crFixtureWrite(t, filepath.Join(root, p), b)
	return contentreview.FileRef{Path: p, SHA256: crSha(b)}
}
func crEvidenceRead(t *testing.T, f crEvidenceFixtureData, ref contentreview.FileRef, v any) {
	t.Helper()
	b, e := os.ReadFile(filepath.Join(f.Root, ref.Path))
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(b, v); e != nil {
		t.Fatal(e)
	}
}
func crEvidenceFixture(t *testing.T) crEvidenceFixtureData {
	t.Helper()
	in, _ := crReviewFixture(t)
	scope, e := contentreview.BuildScope(context.Background(), in)
	if e != nil {
		t.Fatal(e)
	}
	facts, e := contentaudit.CheckSelectedDraft(context.Background(), in.Selected)
	if e != nil {
		t.Fatal(e)
	}
	imports, e := contentreview.BuildImports(context.Background(), in, scope)
	if e != nil {
		t.Fatal(e)
	}
	bundle, e := contentreview.Prepare(context.Background(), in)
	if e != nil {
		t.Fatal(e)
	}
	f := crEvidenceFixtureData{Root: crOutputRoot(t), Manifest: bundle.Manifest, Rows: crRegisterRows(t, bundle)}
	for _, file := range bundle.Files {
		crFixtureWrite(t, filepath.Join(f.Root, file.Path), file.Bytes)
	}
	sig := crEvidenceSave(t, f.Root, "attestation.json", map[string]string{"attestedBy": crFixtureReviewer, "note": "Technical fixture only: independent author and reviewer binding checked."})
	build := crEvidenceSave(t, f.Root, "build.json", map[string]string{"codeSHA": in.CodeSHA, "note": "Isolated technical test build, not deployment."})
	f.Release = contentreview.ReleaseContext{SchemaVersion: 1, ReleaseIdentity: contentreview.ReleaseIdentity{CodeSHA: in.CodeSHA, CatalogueVersion: f.Manifest.CatalogueVersion, CatalogueSHA256: f.Manifest.CatalogueSHA256, Route: f.Manifest.Route, KnowledgeHead: contentaudit.Head{ID: "11111111-1111-4111-8111-111111111111", SHA256: strings.Repeat("1", 64)}, QuestionHead: contentaudit.Head{ID: "22222222-2222-4222-8222-222222222222", SHA256: strings.Repeat("2", 64)}, ManifestSHA256: crSha(crFixtureJSON(t, f.Manifest)), FixtureOnly: true}, BuildRecord: build, AttestedBy: crFixtureReviewer, Attestation: sig}
	f.Input = contentreview.EvidenceInput{SchemaVersion: 1, ReleaseContext: crEvidenceSave(t, f.Root, "release-context.json", f.Release), Bindings: []contentreview.FrozenBinding{}, LearningChecks: []contentreview.LearningFile{}}
	k := publication.FrozenBody{CatalogueVersion: f.Manifest.CatalogueVersion, CatalogueSHA256: f.Manifest.CatalogueSHA256, Package: imports.Knowledge.Package, SourceMap: imports.Knowledge.SourceMap, AuthorIDs: []string{crFixtureAuthor}, Assets: facts.References.Assets}
	k.FrozenDigest, e = publication.FrozenDigest(k)
	if e != nil {
		t.Fatal(e)
	}
	kv := publication.SubmissionView{ID: "33333333-3333-4333-8333-333333333333", Status: "approved", OwnerID: crFixtureAuthor, WorkspaceID: "77777777-7777-4777-8777-777777777777", Revision: 1, CreatedAt: "2026-10-04T00:00:00Z", Gate: publication.GateReport{StructuralErrors: []content.Issue{}, CompletenessErrors: []content.Issue{}, HumanReviewRequirements: []content.Issue{}, ReadyToSubmit: true}, Frozen: k, Review: &publication.ReviewDecision{ID: "44444444-4444-4444-8444-444444444444", SubmissionID: "33333333-3333-4333-8333-333333333333", ReviewerID: crFixtureReviewer, FrozenDigest: k.FrozenDigest, Decision: "approve", Checks: publication.ReviewChecks{Mathematics: true, Explanations: true, Relationships: true, Sources: true, Illustrations: true}, IndependenceNote: "Isolated independent technical reviewer account.", Note: "The exact complete technical fixture was checked."}}
	f.Input.Bindings = append(f.Input.Bindings, contentreview.FrozenBinding{Space: "knowledge", Submission: crEvidenceSave(t, f.Root, "knowledge-submission.json", kv), IndependenceVerified: true, Attestation: sig})
	for p, s := range facts.Sealed {
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
		frozen := question.FrozenBody{CatalogueVersion: f.Manifest.CatalogueVersion, CatalogueSHA256: f.Manifest.CatalogueSHA256, QuestionPackage: s.Package, SourceMap: input.SourceMap, AuthorIDs: []string{crFixtureAuthor}, Resolved: s.Resolved, Objectives: s.Objectives, Generation: s.Generation, InstanceIdentities: ids, Coverage: facts.QuestionReports[p].Coverage, GeneratorVersions: g, VerifierVersions: v}
		_, frozen.FrozenDigest, e = question.CanonicalFrozen(frozen, s.Instances)
		if e != nil {
			t.Fatal(e)
		}
		id := fmt.Sprintf("50000000-0000-4000-8000-%012d", p+1)
		decision := fmt.Sprintf("60000000-0000-4000-8000-%012d", p+1)
		qv := question.SubmissionView{ID: id, Status: "approved", OwnerID: crFixtureAuthor, WorkspaceID: "88888888-8888-4888-8888-888888888888", Revision: 1, CreatedAt: "2026-10-04T00:00:00Z", Gate: facts.QuestionReports[p], Frozen: frozen, Review: &question.ReviewDecision{ID: decision, SubmissionID: id, ReviewerID: crFixtureReviewer, FrozenDigest: frozen.FrozenDigest, Decision: "approve", Checks: question.ReviewChecks{Mathematics: true, Explanations: true, Objectives: true, Sources: true, Illustrations: true, Generation: true}, IndependenceNote: "Isolated independent technical reviewer account.", GenerationNote: "All finite instances checked in this technical fixture.", Note: "The complete exact technical question fixture was checked."}}
		archive := question.Archive{Envelope: input, PackageSHA: s.PackageSHA, Instances: s.Instances, GeneratorVersions: g, VerifierVersions: v, SourceResponsibility: question.SourceResponsibility{AuthorIDs: []string{crFixtureAuthor}}}
		ar := crEvidenceSave(t, f.Root, fmt.Sprintf("archive-%d.json", p), archive)
		f.Input.Bindings = append(f.Input.Bindings, contentreview.FrozenBinding{Space: "questions", Submission: crEvidenceSave(t, f.Root, fmt.Sprintf("submission-%d.json", p), qv), Archive: &ar, IndependenceVerified: true, Attestation: sig})
	}
	for i := range f.Rows {
		for c := range f.Rows[i].Checks {
			f.Rows[i].Checks[c].Status = "passed"
			f.Rows[i].Checks[c].Basis = "Full independent technical fixture check, not a real approval."
			f.Rows[i].Checks[c].ReviewerRef = crFixtureReviewer
		}
	}
	crSaveFinalRegister(t, &f)
	return f
}
func crSaveFinalRegister(t *testing.T, f *crEvidenceFixtureData) {
	t.Helper()
	reg := contentreview.ReviewRegister{SchemaVersion: 1, ManifestSHA256: crSha(crFixtureJSON(t, f.Manifest)), Parts: []contentreview.FileRef{}}
	for start := 0; start < len(f.Rows); start += 100 {
		end := min(start+100, len(f.Rows))
		ref := crEvidenceSave(t, f.Root, fmt.Sprintf("register-final/%03d.json", len(reg.Parts)+1), contentreview.RegisterPart{SchemaVersion: 1, ManifestSHA256: reg.ManifestSHA256, Rows: f.Rows[start:end]})
		reg.Parts = append(reg.Parts, ref)
	}
	f.Input.ReviewRegister = crEvidenceSave(t, f.Root, "review-register.final.json", reg)
}
func crBundleFile(t *testing.T, b contentreview.Bundle, path string) []byte {
	t.Helper()
	for _, f := range b.Files {
		if f.Path == path {
			return f.Bytes
		}
	}
	t.Fatal("missing file", path)
	return nil
}
func crRegisterRows(t *testing.T, b contentreview.Bundle) []contentreview.ReviewRow {
	t.Helper()
	var root contentreview.ReviewRegister
	if e := json.Unmarshal(crBundleFile(t, b, "review-register.json"), &root); e != nil {
		t.Fatal(e)
	}
	rows := []contentreview.ReviewRow{}
	for _, p := range root.Parts {
		var part contentreview.RegisterPart
		raw := crBundleFile(t, b, p.Path)
		if crSha(raw) != p.SHA256 {
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
func crOutputRoot(t *testing.T) string {
	t.Helper()
	p, e := filepath.EvalSymlinks(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	return p
}

var crLearningNames = []string{"reading", "pass", "fail", "prerequisites", "review", "practice_exposure", "retake", "feedback_correction"}

func crWithLearning(t *testing.T, f *crEvidenceFixtureData) {
	t.Helper()
	for _, name := range crLearningNames {
		a := crEvidenceSave(t, f.Root, "learning/"+name+"-observation.json", map[string]string{"observation": "Isolated technical fixture observation, no real learner."})
		record := contentreview.LearningRecord{SchemaVersion: 1, Name: name, Result: "passed", Context: f.Release.ReleaseIdentity, ExecutedAt: "2026-10-04T00:00:00Z", Steps: []contentreview.LearningStep{{Action: "Execute the exact technical scenario", Expected: "Observe the specified acceptance or refusal", Observed: "The expected behavior was observed in this fixture"}}, Attachments: []contentreview.FileRef{a}, AttestedBy: crFixtureReviewer, Attestation: f.Release.Attestation}
		f.Input.LearningChecks = append(f.Input.LearningChecks, contentreview.LearningFile{Name: name, File: crEvidenceSave(t, f.Root, "learning/"+name+".json", record)})
	}
}
func crMutateLearning(t *testing.T, f *crEvidenceFixtureData, index int, mutate func(*contentreview.LearningRecord)) {
	t.Helper()
	ref := f.Input.LearningChecks[index].File
	var record contentreview.LearningRecord
	crEvidenceRead(t, *f, ref, &record)
	mutate(&record)
	f.Input.LearningChecks[index].File = crEvidenceSave(t, f.Root, ref.Path, record)
}

func crPrepareArgs(t *testing.T, mixed bool) ([]string, string) {
	t.Helper()
	in, snapshot := crReviewFixture(t)
	root := crOutputRoot(t)
	if mixed {
		in.Selected.Input.Questions[0].Version = 2
		in.Manifest.QuestionPaths[0] = "content/questions/elementary-foundations-numbers.v2.json"
		in.Selected.Files[2].Path = in.Manifest.QuestionPaths[0]
		in.Selected.Files[2].Bytes = crFixtureJSON(t, in.Selected.Input.Questions[0])
		in.ManifestRaw = crFixtureJSON(t, in.Manifest)
	}
	for _, f := range in.Selected.Files {
		crFixtureWrite(t, filepath.Join(root, f.Path), f.Bytes)
	}
	manifest := filepath.Join(root, "input.json")
	crFixtureWrite(t, manifest, in.ManifestRaw)
	report := filepath.Join(root, "report.json")
	crFixtureWrite(t, report, in.SourceReportRaw)
	sourceMap := filepath.Join(root, "map.json")
	crFixtureWrite(t, sourceMap, in.SourceMapRaw)
	out := filepath.Join(root, "prepared")
	return []string{"prepare", "--root", root, "--input-manifest", manifest, "--snapshot", snapshot, "--source-report", report, "--source-map", sourceMap, "--route", "elementary-foundations", "--version", "1", "--code-sha", in.CodeSHA, "--out", out, "--fixture-only"}, out
}
func crRun(ctx context.Context, args []string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := RunContentReview(ctx, args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}
func crFlag(args []string, flag, value string) []string {
	out := append([]string{}, args...)
	for i := range out {
		if out[i] == flag && i+1 < len(out) {
			out[i+1] = value
			return out
		}
	}
	return append(out, flag, value)
}
func TestContentReviewPrepare(t *testing.T) {
	for _, mixed := range []bool{false, true} {
		args, out := crPrepareArgs(t, mixed)
		code, stdout, stderr := crRun(context.Background(), args)
		if code != 0 || stderr != "" {
			t.Fatal("prepare failed", code, stderr)
		}
		if strings.Contains(stdout, out) || !strings.Contains(stdout, `"objects":958`) {
			t.Fatal("private or incomplete summary", stdout)
		}
		var manifest contentreview.ReviewManifest
		b, e := os.ReadFile(filepath.Join(out, "review-manifest.json"))
		if e != nil {
			t.Fatal(e)
		}
		if e = json.Unmarshal(b, &manifest); e != nil {
			t.Fatal(e)
		}
		if len(manifest.Objects) != 958 || len(manifest.Sources) != 205 || !manifest.FixtureOnly {
			t.Fatal("incomplete export")
		}
		if mixed {
			var q question.DraftInput
			b, e = os.ReadFile(filepath.Join(out, "imports/questions/elementary-foundations-numbers.json"))
			if e != nil {
				t.Fatal(e)
			}
			if e = json.Unmarshal(b, &q); e != nil || q.QuestionPackage.Version != 2 {
				t.Fatal("selected v2 not exported", e)
			}
		}
	}
}
func crVerifyArgs(t *testing.T, f crEvidenceFixtureData) ([]string, string) {
	t.Helper()
	manifest := filepath.Join(f.Root, "review-manifest.json")
	crFixtureWrite(t, manifest, crFixtureJSON(t, f.Manifest))
	input := filepath.Join(f.Root, "evidence-input.json")
	crFixtureWrite(t, input, crFixtureJSON(t, f.Input))
	out := filepath.Join(crOutputRoot(t), "verification")
	return []string{"verify-evidence", "--evidence-root", f.Root, "--review-manifest", manifest, "--input", input, "--out", out}, out
}
func TestContentReviewVerify(t *testing.T) {
	for _, state := range []string{"pending", "ready", "failed"} {
		t.Run(state, func(t *testing.T) {
			f := crEvidenceFixture(t)
			want := 3
			if state != "pending" {
				crWithLearning(t, &f)
				want = 0
			}
			if state == "failed" {
				crMutateLearning(t, &f, 0, func(r *contentreview.LearningRecord) { r.Result = "failed" })
				want = 2
			}
			args, out := crVerifyArgs(t, f)
			code, stdout, stderr := crRun(context.Background(), args)
			if code != want || stderr != "" {
				t.Fatal("wrong verification exit", code, stderr)
			}
			_, e := os.Stat(filepath.Join(out, "acceptance-evidence.json"))
			if state == "pending" && !os.IsNotExist(e) || state != "pending" && e != nil {
				t.Fatal("wrong evidence file presence", e)
			}
			if strings.Contains(stdout, f.Root) || strings.Contains(stdout, crFixtureReviewer) {
				t.Fatal("private material logged")
			}
		})
	}
}
func TestContentReviewOfflineOnly(t *testing.T) {
	t.Setenv("DATABASE_URL", "unreachable://private-database-sentinel")
	t.Setenv("TEST_DATABASE_URL", "unreachable://private-database-sentinel")
	args, _ := crPrepareArgs(t, false)
	code, stdout, stderr := crRun(context.Background(), args)
	if code != 0 {
		t.Fatal(code, stderr)
	}
	f := crEvidenceFixture(t)
	args, _ = crVerifyArgs(t, f)
	code, stdout, stderr = crRun(context.Background(), args)
	if code != 3 {
		t.Fatal(code, stderr)
	}
	if strings.Contains(stdout+stderr, "private-database-sentinel") {
		t.Fatal("configuration read or logged")
	}
}
func TestContentReviewArguments(t *testing.T) {
	args, _ := crPrepareArgs(t, false)
	for _, bad := range [][]string{append(append([]string{}, args...), "--publish"), append(append([]string{}, args...), "--database-env", "DATABASE_URL"), append(append([]string{}, args...), "--root", args[2]), append(append([]string{}, args...), "extra"), crFlag(args, "--version", "2147483648"), crFlag(args, "--version", "0"), crFlag(args, "--code-sha", "old"), crFlag(args, "--route", "bad/route"), crFlag(args, "--root", args[2]+"/."), {"prepare"}, {"unknown"}} {
		if code, _, _ := crRun(context.Background(), bad); code != 2 {
			t.Fatal("invalid argument accepted", code)
		}
	}
	f := crEvidenceFixture(t)
	verify, _ := crVerifyArgs(t, f)
	verify = append(verify, "--root", f.Root)
	if code, _, _ := crRun(context.Background(), verify); code != 2 {
		t.Fatal("foreign subcommand flag accepted")
	}
}
func TestContentReviewIOBudgetCancel(t *testing.T) {
	args, out := crPrepareArgs(t, false)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if code, _, _ := crRun(ctx, args); code != 1 {
		t.Fatal("canceled command", code)
	}
	if _, e := os.Stat(out); !os.IsNotExist(e) {
		t.Fatal("canceled output exists")
	}
	missing := crFlag(args, "--input-manifest", filepath.Join(filepath.Dir(out), "missing.json"))
	if code, _, _ := crRun(context.Background(), missing); code != 1 {
		t.Fatal("missing input", code)
	}
	large := filepath.Join(filepath.Dir(out), "large.json")
	crFixtureWrite(t, large, bytes.Repeat([]byte(" "), contentreview.MaxManifestBytes+1))
	if code, _, _ := crRun(context.Background(), crFlag(args, "--input-manifest", large)); code != 1 {
		t.Fatal("oversized manifest", code)
	}
	fifo := filepath.Join(filepath.Dir(out), "fifo")
	if e := exec.Command("mkfifo", fifo).Run(); e != nil {
		t.Fatal(e)
	}
	if code, _, _ := crRun(context.Background(), crFlag(args, "--input-manifest", fifo)); code != 2 {
		t.Fatal("FIFO input", code)
	}
	crFixtureWrite(t, filepath.Join(out, "old"), []byte("old"))
	if code, _, _ := crRun(context.Background(), args); code != 1 {
		t.Fatal("existing output", code)
	}
	b, e := os.ReadFile(filepath.Join(out, "old"))
	if e != nil || string(b) != "old" {
		t.Fatal("old output changed")
	}
}
