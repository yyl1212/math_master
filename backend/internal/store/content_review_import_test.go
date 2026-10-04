package store_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/contentaudit"
	"github.com/yyl1212/math_master/backend/internal/contentreview"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

func crsSha(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func crsFixtureWrite(t *testing.T, p string, b []byte) {
	t.Helper()
	if e := os.MkdirAll(filepath.Dir(p), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(p, b, 0600); e != nil {
		t.Fatal(e)
	}
}
func crsFixtureJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, e := json.Marshal(v)
	if e != nil {
		t.Fatal(e)
	}
	return b
}

// Uses original application mathematics and a synthetic provenance corpus, never book prose.
func crsReviewFixture(t *testing.T) contentreview.PrepareInput {
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
	corpus := crsFixtureJSON(t, map[string]any{"dataset_id": mapping.Sources[0].DatasetID, "knowledge_points": records})
	sourcePath := mapping.Sources[0].Path
	files := []map[string]any{{"path": sourcePath, "sizeBytes": len(corpus), "sha256": crsSha(corpus)}}
	snapshotSHA := crsSha(crsFixtureJSON(t, files))
	snapshot, e := filepath.EvalSymlinks(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	crsFixtureWrite(t, filepath.Join(snapshot, "files", sourcePath), corpus)
	crsFixtureWrite(t, filepath.Join(snapshot, "manifest.json"), crsFixtureJSON(t, map[string]any{"schemaVersion": 1, "snapshotId": snapshotSHA, "files": files, "createdAt": nil, "sourceRoot": nil, "packageCount": nil, "primaryFiles": nil, "changes": nil, "sourceIndexMismatches": nil}))
	report := contentaudit.SourceReport{SchemaVersion: 1, PolicyVersion: 1, SnapshotID: snapshotSHA, SelectedFiles: []contentaudit.SelectedSource{{Path: sourcePath, SHA256: crsSha(corpus), DatasetID: mapping.Sources[0].DatasetID, RecordIDs: recordIDs}}, Issues: []contentaudit.SourceIssue{}, Ready: true}
	reportRaw := crsFixtureJSON(t, report)
	mapping.SnapshotID = snapshotSHA
	mapping.SourceReportSHA256 = crsSha(reportRaw)
	for i := range mapping.Sources {
		mapping.Sources[i].FileSHA256 = crsSha(corpus)
	}
	mapRaw := crsFixtureJSON(t, mapping)
	sources, e := contentaudit.LoadSourcesFromBytes(context.Background(), snapshot, reportRaw, mapRaw)
	if e != nil {
		t.Fatal("sources", e)
	}
	return contentreview.PrepareInput{CodeSHA: strings.Repeat("a", 40), Route: content.VersionRef{ID: "elementary-foundations", Version: 1}, Manifest: manifest, ManifestRaw: raw, Selected: selected, Sources: sources, SourceReportRaw: reportRaw, SourceMapRaw: mapRaw, FixtureOnly: true}
}

func TestContentReviewWorkflowRoundTrip(t *testing.T) {
	in := crsReviewFixture(t)
	scope, e := contentreview.BuildScope(context.Background(), in)
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
	f := crsWorkflowFixture(t)
	whole, stop := context.WithTimeout(context.Background(), 4*time.Minute)
	defer stop()
	f.ctx = whole
	ksvc := publication.NewService(f.repo)
	qsvc, e := question.NewService(f.repo, ksvc.AcquireValidation)
	if e != nil {
		t.Fatal(e)
	}
	// Technical sessions and role grants are confined to testutil's random database.
	f.exec(`INSERT INTO auth_user_roles(user_id,role) VALUES($1,'reviewer')`, f.ids["author_a"])
	d, e := ksvc.CreateDraft(f.ctx, f.Access("author_a", false), imports.Knowledge)
	if e != nil {
		t.Fatal("knowledge import", e)
	}
	read, e := ksvc.ReadDraft(f.ctx, f.Access("author_a", false), d.ID)
	if e != nil || content.Digest(read.Package) != content.Digest(imports.Knowledge.Package) || content.Digest(read.SourceMap) != content.Digest(imports.Knowledge.SourceMap) {
		t.Fatal("knowledge editable export differs", e)
	}
	for _, a := range read.Assets {
		raw, e := ksvc.ReadDraftAsset(f.ctx, f.Access("author_a", false), d.ID, a.SHA256)
		if e != nil || !bytes.Equal(raw, in.Selected.Assets[a.ID]) {
			t.Fatal("SVG export differs", e)
		}
	}
	if _, e = ksvc.SubmitDraft(f.ctx, f.Access("author_a", false), d.ID, publication.SubmitInput{ExpectedRevision: d.Revision + 1, ExpectedDigest: d.Gate.Digest}); e == nil {
		t.Fatal("stale revision accepted")
	}
	ksub, e := ksvc.SubmitDraft(f.ctx, f.Access("author_a", false), d.ID, publication.SubmitInput{ExpectedRevision: d.Revision, ExpectedDigest: d.Gate.Digest})
	if e != nil {
		t.Fatal("knowledge submit", e)
	}
	if _, e = ksvc.DecideReview(f.ctx, f.Access("author_a", false), ksub.ID, approvedReviewInput()); e == nil {
		t.Fatal("author approved own knowledge")
	}
	ksub, e = ksvc.DecideReview(f.ctx, f.Access("reviewer_a", false), ksub.ID, approvedReviewInput())
	if e != nil {
		t.Fatal("knowledge review", e)
	}
	kpub := f.Activate(f.Prepare(ksub, nil), nil)
	knowledgeExport, e := f.repo.ExportPackage(f.ctx, ksub.Frozen.Package.ID, ksub.Frozen.Package.Version)
	if e != nil || content.Digest(knowledgeExport) != content.Digest(ksub.Frozen.Package) {
		t.Fatal("sealed knowledge export differs", e)
	}
	qf := &questionFixture{workflowFixture: f}
	subs := []question.SubmissionView{}
	ids := []string{}
	archives := []question.Archive{}
	for _, q := range imports.Questions {
		d, e := qsvc.CreateDraft(f.ctx, f.Access("author_a", false), q)
		if e != nil {
			t.Fatal("question import", e)
		}
		read, e := qsvc.ReadDraft(f.ctx, f.Access("author_a", false), d.ID)
		if e != nil || content.Digest(read.SourceMap) != content.Digest(q.SourceMap) {
			t.Fatal("question editable export differs", e)
		}
		gate, e := qsvc.ValidateDraft(f.ctx, f.Access("author_a", false), d.ID, question.ValidateInput{ExpectedRevision: d.Revision})
		if e != nil || !gate.ReadyToSubmit {
			t.Fatal("question gate", e)
		}
		sub, e := qsvc.SubmitDraft(f.ctx, f.Access("author_a", false), d.ID, question.SubmitInput{ExpectedRevision: d.Revision, ExpectedDigest: gate.Digest})
		if e != nil {
			t.Fatal("question submit", e)
		}
		if _, e = qsvc.DecideReview(f.ctx, f.Access("author_a", false), sub.ID, approvedQuestionInput()); e == nil {
			t.Fatal("author approved own question")
		}
		sub, e = qsvc.DecideReview(f.ctx, f.Access("reviewer_a", false), sub.ID, approvedQuestionInput())
		if e != nil {
			t.Fatal("question review", e)
		}
		if len(sub.Frozen.AuthorIDs) != 1 || sub.Frozen.AuthorIDs[0] != f.ids["author_a"] {
			t.Fatal("server author responsibility omitted")
		}
		ids = append(ids, sub.ID)
		subs = append(subs, sub)
	}
	qp := qf.QPrepare(ids...)
	wrong := f.ID()
	if _, e = f.repo.ActivateQuestionRelease(f.ctx, f.Access("admin_a", true), qp.ID, question.ActivateInput{ExpectedKnowledgeHead: &wrong, ExpectedQuestionHead: qp.BaseQuestionHead, ExpectedManifestSHA: qp.ManifestSHA, Reason: "Isolated wrong-head rejection check."}); e == nil {
		t.Fatal("wrong knowledge head accepted")
	}
	qpub := qf.QActivate(qp)
	for _, sub := range subs {
		a, e := f.repo.ExportQuestionArchive(f.ctx, sub.Frozen.QuestionPackage.ID, sub.Frozen.QuestionPackage.Version)
		if e != nil {
			t.Fatal("archive export", e)
		}
		byID := map[string]question.Instance{}
		for _, i := range a.Instances {
			byID[i.Identity.ID] = i
		}
		ordered := []question.Instance{}
		for _, id := range sub.Frozen.InstanceIdentities {
			i, ok := byID[id.ID]
			if !ok || i.Identity != id {
				t.Fatal("archive identity omitted")
			}
			ordered = append(ordered, i)
		}
		_, hash, e := question.CanonicalFrozen(sub.Frozen, ordered)
		if e != nil || hash != sub.Frozen.FrozenDigest {
			t.Fatal("normal export frozen mismatch", e)
		}
		archives = append(archives, a)
	}
	root, e := filepath.EvalSymlinks(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	for _, file := range bundle.Files {
		crsFixtureWrite(t, filepath.Join(root, file.Path), file.Bytes)
	}
	save := func(p string, v any) contentreview.FileRef {
		raw := crsFixtureJSON(t, v)
		crsFixtureWrite(t, filepath.Join(root, p), raw)
		return contentreview.FileRef{Path: p, SHA256: crsSha(raw)}
	}
	manifestSHA := crsSha(crsFixtureJSON(t, bundle.Manifest))
	release := contentreview.ReleaseContext{SchemaVersion: 1, ReleaseIdentity: contentreview.ReleaseIdentity{CodeSHA: in.CodeSHA, CatalogueVersion: bundle.Manifest.CatalogueVersion, CatalogueSHA256: bundle.Manifest.CatalogueSHA256, Route: bundle.Manifest.Route, KnowledgeHead: contentaudit.Head{ID: kpub.ID, SHA256: kpub.ManifestSHA}, QuestionHead: contentaudit.Head{ID: qpub.ID, SHA256: qpub.ManifestSHA}, ManifestSHA256: manifestSHA, FixtureOnly: true}, AttestedBy: f.ids["reviewer_a"], BuildRecord: save("build.json", map[string]string{"codeSHA": in.CodeSHA, "note": "isolated technical test only"}), Attestation: save("attestation.json", map[string]string{"note": "Technical independence and exact bindings only, no human approval."})}
	var reg contentreview.ReviewRegister
	if e = json.Unmarshal(bundleFileStore(t, bundle, "review-register.json"), &reg); e != nil {
		t.Fatal(e)
	}
	final := contentreview.ReviewRegister{SchemaVersion: 1, ManifestSHA256: manifestSHA, Parts: []contentreview.FileRef{}}
	for index, ref := range reg.Parts {
		var part contentreview.RegisterPart
		if e = json.Unmarshal(bundleFileStore(t, bundle, ref.Path), &part); e != nil {
			t.Fatal(e)
		}
		for row := range part.Rows {
			for c := range part.Rows[row].Checks {
				part.Rows[row].Checks[c].Status = "passed"
				part.Rows[row].Checks[c].Basis = "Complete technical round-trip identity check in a random isolated database."
				part.Rows[row].Checks[c].ReviewerRef = f.ids["reviewer_a"]
			}
		}
		final.Parts = append(final.Parts, save(fmt.Sprintf("register-final/%03d.json", index+1), part))
	}
	evidence := contentreview.EvidenceInput{SchemaVersion: 1, ReleaseContext: save("release-context.json", release), ReviewRegister: save("review-register.final.json", final), Bindings: []contentreview.FrozenBinding{{Space: "knowledge", Submission: save("knowledge-submission.json", ksub), IndependenceVerified: true, Attestation: release.Attestation}}, LearningChecks: []contentreview.LearningFile{}}
	for i, sub := range subs {
		archive := save(fmt.Sprintf("archive-%d.json", i), archives[i])
		evidence.Bindings = append(evidence.Bindings, contentreview.FrozenBinding{Space: "questions", Submission: save(fmt.Sprintf("submission-%d.json", i), sub), Archive: &archive, IndependenceVerified: true, Attestation: release.Attestation})
	}
	verified, e := contentreview.VerifyEvidence(f.ctx, root, bundle.Manifest, evidence)
	if e != nil || !verified.ReviewComplete || verified.Conclusion != "awaiting_review" || verified.Evidence != nil {
		t.Fatal("actual normal exports did not bind or missing learning evidence became ready", e)
	}
	before := auditRows(t, f.db)
	facts, e := f.repo.ReadContentAudit(f.ctx, in.Route)
	if e != nil {
		t.Fatal(e)
	}
	report, e := contentaudit.EvaluatePublished(f.ctx, contentaudit.Request{Mode: contentaudit.Published, Route: in.Route, FixtureOnly: true, CodeSHA: in.CodeSHA, At: time.Now()}, facts, in.Sources, contentaudit.AcceptanceEvidence{})
	if e != nil || !report.FixtureOnly || report.FormalCounts.Knowledge != 0 || report.Conclusion == contentaudit.Accepted {
		t.Fatal("technical fixture upgraded", e)
	}
	if content.Digest(before) != content.Digest(auditRows(t, f.db)) {
		t.Fatal("read-only audit mutated database")
	}
	bad := imports.Knowledge
	bad.AssetBytes = append([]publication.AssetInput{}, bad.AssetBytes...)
	bad.AssetBytes[0].Base64 = base64.StdEncoding.EncodeToString([]byte("<svg>altered</svg>"))
	invalid, err := ksvc.CreateDraft(f.ctx, f.Access("author_a", false), bad)
	if err == nil {
		if invalid.Gate.ReadyToSubmit {
			t.Fatal("changed SVG ready as unchanged identity")
		}
		if _, err = ksvc.SubmitDraft(f.ctx, f.Access("author_a", false), invalid.ID, publication.SubmitInput{ExpectedRevision: invalid.Revision, ExpectedDigest: invalid.Gate.Digest}); err == nil {
			t.Fatal("changed SVG submitted")
		}
	}
}
func bundleFileStore(t *testing.T, b contentreview.Bundle, path string) []byte {
	t.Helper()
	for _, f := range b.Files {
		if f.Path == path {
			return f.Bytes
		}
	}
	t.Fatal("missing bundle file", path)
	return nil
}

func crsWorkflowFixture(t *testing.T) *workflowFixture {
	t.Helper()
	f := &workflowFixture{authFixture: newAuthFixture(t), ids: map[string]string{}, access: map[string]publication.Access{}}
	// Preserve the catalogue and legacy knowledge, but do not pre-publish a conflicting route.
	if _, e := f.repo.ImportDraft(f.ctx, input(t, func(p *content.Package) { p.Paths = []content.Path{} })); e != nil {
		t.Fatal(e)
	}
	for _, name := range []string{"author_a", "reviewer_a", "admin_a"} {
		u, cookies, csrf := f.signup(name)
		f.ids[name] = u.ID
		role := "editor"
		if strings.HasPrefix(name, "reviewer") {
			role = "reviewer"
		}
		if strings.HasPrefix(name, "admin") {
			role = "admin"
		}
		f.exec(`INSERT INTO auth_user_roles(user_id,role) VALUES($1,$2)`, u.ID, role)
		proof, e := auth.DecodeContentProof(cookies, csrf, true)
		if e != nil {
			t.Fatal(e)
		}
		f.access[name] = publication.Access{TokenHash: proof.TokenHash, CSRF: proof.CSRF, RequestID: "content-review-fixture"}
	}
	return f
}
func TestContentReviewExistingRouteConflict(t *testing.T) {
	in := crsReviewFixture(t)
	scope, e := contentreview.BuildScope(context.Background(), in)
	if e != nil {
		t.Fatal(e)
	}
	imports, e := contentreview.BuildImports(context.Background(), in, scope)
	if e != nil {
		t.Fatal(e)
	}
	f := newWorkflowFixture(t)
	d, e := f.repo.CreateDraft(f.ctx, f.Access("author_a", false), imports.Knowledge)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = f.repo.SubmitDraft(f.ctx, f.Access("author_a", false), d.ID, publication.SubmitInput{ExpectedRevision: d.Revision, ExpectedDigest: d.Gate.Digest}); e == nil {
		t.Fatal("existing different route overwritten at same version")
	}
}
