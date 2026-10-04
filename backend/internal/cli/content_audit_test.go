package cli

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/contentaudit"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func auditHash(b []byte) string { return fmt.Sprintf("%x", sha256.Sum256(b)) }
func auditArgs(t *testing.T, mode string) []string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	os.Mkdir(filepath.Join(dir, "files"), 0700)
	corpus := []byte(`{"schema_version":"1","dataset_id":"set","knowledge_points":[{"id":"r1"}]}`)
	os.WriteFile(filepath.Join(dir, "files", "main.json"), corpus, 0600)
	files := []map[string]any{{"path": "main.json", "sizeBytes": len(corpus), "sha256": auditHash(corpus)}}
	raw, _ := json.Marshal(files)
	snapshot := auditHash(raw)
	manifest := map[string]any{"schemaVersion": 1, "snapshotId": snapshot, "files": files, "createdAt": "2026-10-04T00:00:00Z", "sourceRoot": "/unused", "packageCount": 1, "primaryFiles": []string{"main.json"}, "changes": map[string]any{"added": []string{"main.json"}, "modified": []string{}, "missing": []string{}}, "sourceIndexMismatches": []string{}}
	save := func(name string, x any) {
		b, e := json.Marshal(x)
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(dir, name), b, 0600); e != nil {
			t.Fatal(e)
		}
	}
	save("manifest.json", manifest)
	report := contentaudit.SourceReport{SchemaVersion: 1, PolicyVersion: 1, SnapshotID: snapshot, SelectedFiles: []contentaudit.SelectedSource{{Path: "main.json", SHA256: auditHash(corpus), DatasetID: "set", RecordIDs: []string{"r1"}}}, Issues: []contentaudit.SourceIssue{}, Ready: true}
	save("report.json", report)
	rb, _ := os.ReadFile(filepath.Join(dir, "report.json"))
	mb, e := os.ReadFile("../../../content/source-maps/elementary-foundations.v1.json")
	if e != nil {
		t.Fatal(e)
	}
	var mapping contentaudit.SourceMap
	json.Unmarshal(mb, &mapping)
	mapping.SnapshotID = snapshot
	mapping.SourceReportSHA256 = auditHash(rb)
	mapping.Sources = []contentaudit.MappedSource{{ID: "s1", Path: "main.json", RecordID: "r1", DatasetID: "set", FileSHA256: auditHash(corpus), PublicSources: mapping.Sources[0].PublicSources, Use: "original_derivation", LegacyIDs: []string{}, ReviewStatus: "technical_fixture", ConditionsNote: "Original fixture source identity only."}}
	for i := range mapping.Objects {
		mapping.Objects[i].SourceIDs = []string{"s1"}
	}
	save("map.json", mapping)
	a := []string{"--mode", mode, "--route", "elementary-foundations", "--version", "1", "--snapshot", dir, "--source-report", filepath.Join(dir, "report.json"), "--source-map", filepath.Join(dir, "map.json"), "--code-sha", strings.Repeat("a", 40), "--out", filepath.Join(dir, "output")}
	if mode == "draft" {
		a = append(a, "--root", "../../..")
	} else {
		a = append(a, "--database-env", "P6A_TEST_SQL")
	}
	return a
}
func auditRun(a []string) (int, string, string) {
	var out, err bytes.Buffer
	c := RunContentAudit(context.Background(), a, &out, &err)
	return c, out.String(), err.String()
}
func auditReport(t *testing.T, a []string) contentaudit.Report {
	t.Helper()
	var r contentaudit.Report
	b, e := os.ReadFile(a[15] + "/report.json")
	if e != nil {
		t.Fatal(e)
	}
	if e = json.Unmarshal(b, &r); e != nil {
		t.Fatal(e)
	}
	return r
}
func TestContentAuditCLI(t *testing.T) {
	for _, a := range [][]string{nil, {"--unknown"}, {"--mode=draft", "--mode=published"}, {"--mode=draft", "--version=2147483648"}} {
		if c, _, _ := auditRun(a); c != 2 {
			t.Fatal("invalid args", c)
		}
	}
	t.Setenv("DATABASE_URL", "postgres://should-not-be-used/private")
	a := auditArgs(t, "draft")
	c, o, e := auditRun(a)
	if c != 0 {
		t.Fatal(c, o, e)
	}
	r := auditReport(t, a)
	if r.Conclusion != contentaudit.DraftReady || r.FormalCounts.EffectiveInstances != 0 || r.DraftCounts.EffectiveInstances != 834 {
		t.Fatal(r.Conclusion, r.DraftCounts)
	}
	for _, p := range []struct {
		name string
		mode os.FileMode
	}{{a[15], 0700}, {a[15] + "/report.json", 0600}, {a[15] + "/report.md", 0600}} {
		s, e := os.Stat(p.name)
		if e != nil || s.Mode().Perm() != p.mode {
			t.Fatal("unsafe output permissions")
		}
	}
	os.WriteFile(a[15]+"/user-file", []byte("keep"), 0600)
	if c, _, _ := auditRun(a); c != 1 {
		t.Fatal("existing output accepted", c)
	}
	if b, _ := os.ReadFile(a[15] + "/user-file"); string(b) != "keep" {
		t.Fatal("existing output changed")
	}
	a = auditArgs(t, "published")
	t.Setenv("P6A_TEST_SQL", "")
	if c, _, _ := auditRun(a); c != 1 {
		t.Fatal("explicit DSN missing", c)
	}
	t.Setenv("P6A_TEST_SQL", "postgres://user:private-test-secret@127.0.0.1:1/db?connect_timeout=1")
	c, o, e = auditRun(a)
	if c != 1 || strings.Contains(o+e, "private-test-secret") {
		t.Fatal("secret boundary")
	}
}
func auditPublishedFixture(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()
	db := testutil.Database(t)
	if e := store.Up(ctx, db, "../../../db/migrations"); e != nil {
		t.Fatal(e)
	}
	repo := store.New(db)
	accounts, e := auth.NewService(repo, auth.NewArgon2Hasher(rand.Reader), rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	access := map[string]publication.Access{}
	for _, name := range []string{"editor", "reviewer", "admin"} {
		v, d, e := accounts.Context(ctx, auth.Cookies{})
		if e != nil {
			t.Fatal(e)
		}
		u, s, e := accounts.Register(ctx, auth.Cookies{Preauth: d.SetPreauth}, v.CSRFToken, auth.RegisterInput{Username: "audit_" + name, Password: cliTestPassword}, "audit-test")
		if e != nil {
			t.Fatal(e)
		}
		if _, e = db.Exec(`INSERT INTO auth_user_roles(user_id,role) VALUES($1,$2)`, u.ID, name); e != nil {
			t.Fatal(e)
		}
		v, d, e = accounts.Context(ctx, auth.Cookies{})
		if e != nil {
			t.Fatal(e)
		}
		_, s, e = accounts.Login(ctx, auth.Cookies{Preauth: d.SetPreauth}, v.CSRFToken, auth.LoginInput{Username: "audit_" + name, Password: cliTestPassword}, "audit-test-login")
		if e != nil {
			t.Fatal(e)
		}
		cookies := auth.Cookies{Session: s.SetSession}
		v, _, e = accounts.Context(ctx, cookies)
		if e != nil {
			t.Fatal(e)
		}
		p, e := auth.DecodeContentProof(cookies, v.CSRFToken, true)
		if e != nil {
			t.Fatal(e)
		}
		access[name] = publication.Access{TokenHash: p.TokenHash, CSRF: p.CSRF, RequestID: "audit-fixture"}
		if _, e = db.Exec(`UPDATE auth_sessions SET reauthenticated_at=clock_timestamp() WHERE token_hash=$1`, p.TokenHash[:]); e != nil {
			t.Fatal(e)
		}
	}
	next := func(n string) publication.Access {
		a := access[n]
		a.IdempotencyKey, e = auth.NewID(rand.Reader)
		if e != nil {
			t.Fatal(e)
		}
		return a
	}
	draft, e := contentaudit.LoadDraft(ctx, "../../..")
	if e != nil {
		t.Fatal(e)
	}
	validated, e := contentaudit.CheckDraft(ctx, draft)
	if e != nil {
		t.Fatal(e)
	}
	_ = validated
	if _, e = repo.ImportDraft(ctx, auditValidatedContent(t, draft)); e != nil {
		t.Fatal(e)
	}
	in := publication.DraftInput{CatalogueVersion: 1, Package: draft.Content, AssetBytes: []publication.AssetInput{}, SourceMap: []publication.SourceLink{}}
	for _, a := range draft.Content.Assets {
		b, e := os.ReadFile(filepath.Join(draft.AssetsRoot, a.Path))
		if e != nil {
			t.Fatal(e)
		}
		in.AssetBytes = append(in.AssetBytes, publication.AssetInput{ID: a.ID, Base64: base64.StdEncoding.EncodeToString(b)})
	}
	d, e := repo.CreateDraft(ctx, next("editor"), in)
	if e != nil {
		t.Fatal(e)
	}
	sub, e := repo.SubmitDraft(ctx, next("editor"), d.ID, publication.SubmitInput{ExpectedRevision: d.Revision, ExpectedDigest: d.Gate.Digest})
	if e != nil {
		t.Fatal(e)
	}
	sub, e = repo.DecideReview(ctx, next("reviewer"), sub.ID, publication.ReviewInput{Decision: "approve", Checks: publication.ReviewChecks{Mathematics: true, Explanations: true, Relationships: true, Sources: true, Illustrations: true}, IndependenceNote: "Distinct technical fixture accounts; not real independent review.", Note: "Only isolated technical workflow validation."})
	if e != nil {
		t.Fatal(e)
	}
	p, e := repo.PrepareRelease(ctx, next("admin"), publication.PrepareInput{SubmissionIDs: []string{sub.ID}, Reason: "Isolated technical content acceptance fixture only."})
	if e != nil {
		t.Fatal(e)
	}
	p, e = repo.ActivateRelease(ctx, next("admin"), p.ID, publication.ActivateInput{ExpectedManifestSHA: p.ManifestSHA, Reason: "Isolated technical content acceptance fixture only."})
	if e != nil {
		t.Fatal(e)
	}
	kh := p.ID
	var qh *string
	for _, bank := range draft.Questions {
		d, e := repo.CreateQuestionDraft(ctx, next("editor"), question.DraftInput{CatalogueVersion: 1, QuestionPackage: bank, SourceMap: []question.SourceLink{}})
		if e != nil {
			t.Fatal(e)
		}
		gate, e := repo.ValidateQuestionDraft(ctx, next("editor"), d.ID, question.ValidateInput{ExpectedRevision: d.Revision})
		if e != nil || !gate.ReadyToSubmit {
			t.Fatal("question gate", e)
		}
		s, e := repo.SubmitQuestionDraft(ctx, next("editor"), d.ID, question.SubmitInput{ExpectedRevision: d.Revision, ExpectedDigest: gate.Digest})
		if e != nil {
			t.Fatal(e)
		}
		_, e = repo.DecideQuestionReview(ctx, next("reviewer"), s.ID, question.ReviewInput{Decision: "approve", Checks: question.ReviewChecks{Mathematics: true, Explanations: true, Objectives: true, Sources: true, Illustrations: true, Generation: true}, IndependenceNote: "Distinct technical fixture accounts only.", GenerationNote: "All finite generated instances checked as a technical fixture.", Note: "No real mathematical approval is attested."})
		if e != nil {
			t.Fatal(e)
		}
		p, e := repo.PrepareQuestionRelease(ctx, next("admin"), question.PrepareInput{SubmissionIDs: []string{s.ID}, ExpectedKnowledgeHead: &kh, ExpectedQuestionHead: qh, Reason: "Isolated technical content acceptance fixture only."})
		if e != nil {
			t.Fatal(e)
		}
		p, e = repo.ActivateQuestionRelease(ctx, next("admin"), p.ID, question.ActivateInput{ExpectedKnowledgeHead: &kh, ExpectedQuestionHead: qh, ExpectedManifestSHA: p.ManifestSHA, Reason: "Isolated technical content acceptance fixture only."})
		if e != nil {
			t.Fatal(e)
		}
		id := p.ID
		qh = &id
	}
	return db
}
func TestContentAuditFixtureCannotAccept(t *testing.T) {
	db := auditPublishedFixture(t)
	setURL(t, db)
	t.Setenv("P6A_TEST_SQL", os.Getenv("DATABASE_URL"))
	a := auditArgs(t, "published")
	c, o, e := auditRun(a)
	if c != 3 {
		t.Fatal(c, o, e)
	}
	r := auditReport(t, a)
	if !r.FixtureOnly || r.Conclusion != contentaudit.AwaitingReview || r.FormalCounts.EffectiveInstances != 0 || r.DraftCounts.EffectiveInstances != 834 {
		t.Fatal(r.Conclusion, r.FormalCounts, r.DraftCounts)
	}

	facts, err := store.New(db).ReadContentAudit(context.Background(), r.Context.Route)
	if err != nil {
		t.Fatal(err)
	}
	ev := contentaudit.AcceptanceEvidence{SchemaVersion: 1, CodeSHA: strings.Repeat("b", 40), RouteSHA: facts.PathSHA, KnowledgeHead: facts.KnowledgeHead, QuestionHead: facts.QuestionHead, ReviewAttestations: []contentaudit.ReviewAttestation{}, LearningChecks: []contentaudit.LearningCheck{}}
	ef := filepath.Join(filepath.Dir(a[15]), "evidence.json")
	raw, _ := json.Marshal(ev)
	os.WriteFile(ef, raw, 0600)
	bad := auditArgs(t, "published")
	bad = append(bad, "--evidence", ef)
	if c, _, _ := auditRun(bad); c != 2 {
		t.Fatal("stale code proof accepted", c)
	}
	var database string
	db.QueryRow(`SELECT current_database()`).Scan(&database)
	role := "p6a_cli_ro"
	db.Exec(`CREATE ROLE ` + role + ` NOLOGIN`)
	t.Cleanup(func() { db.Exec(`DROP OWNED BY ` + role); db.Exec(`DROP ROLE ` + role) })
	if _, e := db.Exec(`GRANT USAGE ON SCHEMA public TO ` + role + `; GRANT SELECT ON ALL TABLES IN SCHEMA public TO ` + role); e != nil {
		t.Fatal(e)
	}
	u, _ := url.Parse(os.Getenv("P6A_TEST_SQL"))
	q := u.Query()
	q.Set("role", role)
	u.RawQuery = q.Encode()
	t.Setenv("P6A_TEST_SQL", u.String())
	a = auditArgs(t, "published")
	if c, _, e := auditRun(a); c != 3 {
		t.Fatal("SELECT role", c, e)
	}
}

func auditValidatedContent(t *testing.T, d contentaudit.DraftInput) content.ValidatedPackage {
	t.Helper()
	v, r := content.ValidateAndSeal(d.Catalogue, d.Content, d.AssetsRoot)
	if len(r.Errors) > 0 {
		t.Fatal(r)
	}
	return v
}
