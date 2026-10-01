package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"net/http"
	"os"
	"strings"
	"testing"
)

func contentRealActor(t *testing.T, h http.Handler, name string) (map[string]string, auth.User) {
	t.Helper()
	cookie, csrf := httpContext(t, h)
	headers := map[string]string{"Cookie": cookie, "Origin": privateOrigin, "X-CSRF-Token": csrf, "Content-Type": "application/json"}
	w := privateRequest(h, "POST", "/api/v1/auth/register", `{"username":"`+name+`","password":"`+httpTestPassword+`"}`, headers)
	if w.Code != 201 {
		t.Fatal("isolated registration failed", w.Code)
	}
	var registered struct{ Data struct{ User auth.User } }
	if json.Unmarshal(w.Body.Bytes(), &registered) != nil {
		t.Fatal("registration DTO failed")
	}
	cookie, csrf = httpContext(t, h)
	headers["Cookie"] = cookie
	headers["X-CSRF-Token"] = csrf
	w = privateRequest(h, "POST", "/api/v1/auth/login", `{"username":"`+name+`","password":"`+httpTestPassword+`"}`, headers)
	if w.Code != 200 {
		t.Fatal("isolated login failed", w.Code)
	}
	for _, c := range w.Result().Cookies() {
		if c.Name == "mm_session_dev" {
			headers["Cookie"] = c.Name + "=" + c.Value
		}
	}
	w = privateRequest(h, "GET", "/api/v1/auth/context", "", map[string]string{"Cookie": headers["Cookie"], "X-Requested-With": "MathMaster"})
	var contextual struct{ Data auth.ContextView }
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &contextual) != nil {
		t.Fatal("isolated auth context failed")
	}
	headers["X-CSRF-Token"] = contextual.Data.CSRFToken
	return headers, registered.Data.User
}
func contentHTTPCall(t *testing.T, h http.Handler, headers map[string]string, method, path string, input, out any, status int) {
	t.Helper()
	id, err := auth.NewID(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	headers["Idempotency-Key"] = id
	raw := ""
	if input != nil {
		b, e := json.Marshal(input)
		if e != nil {
			t.Fatal(e)
		}
		raw = string(b)
	}
	w := privateRequest(h, method, path, raw, headers)
	if w.Code != status {
		t.Fatalf("%s %s: %d %s", method, path, w.Code, w.Body.String())
	}
	if len(w.Result().Cookies()) != 0 {
		t.Fatal("content endpoint changed sign-in cookies")
	}
	if out != nil && json.Unmarshal(w.Body.Bytes(), out) != nil {
		t.Fatal("content DTO decoding failed")
	}
}
func TestContentHTTPWorkflow(t *testing.T) {
	db := testutil.Database(t)
	ctx := context.Background()
	if err := store.Up(ctx, db, "../../../db/migrations"); err != nil {
		t.Fatal(err)
	}
	repo := store.New(db)
	cf, pf, assetRoot := testutil.Seed(t)
	f, err := os.Open(cf)
	if err != nil {
		t.Fatal(err)
	}
	catalogue, err := content.DecodeCatalogue(f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	f, err = os.Open(pf)
	if err != nil {
		t.Fatal(err)
	}
	p, err := content.DecodePackage(f)
	f.Close()
	if err != nil {
		t.Fatal(err)
	}
	v, report := content.ValidateAndSeal(catalogue, p, assetRoot)
	if len(report.Errors) > 0 {
		t.Fatal(report.Errors)
	}
	if _, err = repo.ImportDraft(ctx, v); err != nil {
		t.Fatal(err)
	}
	ready, err := contentReady(ctx, db)
	if err != nil || !ready {
		t.Fatal("content table readiness failed", err)
	}
	hasher := auth.NewArgon2Hasher(rand.Reader)
	accounts, err := auth.NewService(repo, hasher, rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	h := NewApplicationHandler(repo, db, AuthOptions{Accounts: accounts, Admin: auth.NewAdminService(repo, hasher, rand.Reader), PublicOrigin: privateOrigin, Content: &ContentOptions{Service: publication.NewService(repo), PublicOrigin: privateOrigin, Configured: ready}})
	editor, e := contentRealActor(t, h, "content_http_editor")
	reviewer, r := contentRealActor(t, h, "content_http_reviewer")
	admin, a := contentRealActor(t, h, "content_http_admin")
	for _, pair := range []struct{ id, role string }{{e.ID, "editor"}, {r.ID, "reviewer"}, {a.ID, "admin"}} {
		if _, err = db.ExecContext(ctx, `INSERT INTO auth_user_roles(user_id,role) VALUES($1,$2)`, pair.id, pair.role); err != nil {
			t.Fatal(err)
		}
	}
	raw, err := os.ReadFile("../content/testdata/workflow-ready.json")
	if err != nil {
		t.Fatal(err)
	}
	if json.Unmarshal(raw, &p) != nil {
		t.Fatal("fixture package failed")
	}
	p.Knowledge[0].ID = "http-workflow-fractions"
	p.Units[0].Knowledge.ID = p.Knowledge[0].ID
	p.Assets[0].Knowledge.ID = p.Knowledge[0].ID
	p.Units[0].ID = "http-workflow-unit"
	svg, err := os.ReadFile("../content/testdata/workflow-ready.svg")
	if err != nil {
		t.Fatal(err)
	}
	input := publication.DraftInput{CatalogueVersion: 1, Package: p, AssetBytes: []publication.AssetInput{{ID: p.Assets[0].ID, Base64: base64.StdEncoding.EncodeToString(svg)}}, SourceMap: []publication.SourceLink{}}
	var draft publication.DraftView
	contentHTTPCall(t, h, editor, "POST", "/api/v1/content/drafts", input, &draft, 201)
	var gate publication.GateReport
	contentHTTPCall(t, h, editor, "POST", "/api/v1/content/drafts/"+draft.ID+"/validate", publication.ValidateInput{ExpectedRevision: 1}, &gate, 200)
	if !gate.ReadyToSubmit {
		t.Fatal("ready original fixture blocked", gate)
	}
	var sub publication.SubmissionView
	contentHTTPCall(t, h, editor, "POST", "/api/v1/content/drafts/"+draft.ID+"/submit", publication.SubmitInput{ExpectedRevision: 1, ExpectedDigest: gate.Digest}, &sub, 201)
	privateSVG := privateRequest(h, "GET", "/api/v1/content/submissions/"+sub.ID+"/assets/"+p.Assets[0].SHA256, "", reviewer)
	if privateSVG.Code != 200 || privateSVG.Body.String() != string(svg) {
		t.Fatal("scoped frozen SVG missing")
	}
	var queue publication.Page[publication.SubmissionSummary]
	contentHTTPCall(t, h, reviewer, "GET", "/api/v1/content/submissions?scope=review", nil, &queue, 200)
	if len(queue.Items) != 1 {
		t.Fatal("review queue missing pending independent submission")
	}
	contentHTTPCall(t, h, reviewer, "POST", "/api/v1/content/submissions/"+sub.ID+"/decision", publication.ReviewInput{Decision: "approve", Checks: publication.ReviewChecks{Mathematics: true, Explanations: true, Relationships: true, Sources: true, Illustrations: true}, IndependenceNote: "Separate isolated author and reviewer technical accounts.", Note: "All technical fixture review checks were explicitly completed."}, &sub, 200)
	var published publication.PublicationView
	contentHTTPCall(t, h, admin, "POST", "/api/v1/content/publications/prepare", publication.PrepareInput{SubmissionIDs: []string{sub.ID}, Reason: "Prepare this technically approved isolated HTTP fixture."}, &published, 201)
	if w := privateRequest(h, "GET", "/api/v1/knowledge/"+p.Knowledge[0].ID, "", nil); w.Code != 404 {
		t.Fatal("prepared HTTP content public")
	}
	contentHTTPCall(t, h, admin, "POST", "/api/v1/auth/reauth", auth.ReauthInput{Password: httpTestPassword}, nil, 200)
	contentHTTPCall(t, h, admin, "POST", "/api/v1/content/publications/"+published.ID+"/activate", publication.ActivateInput{ExpectedManifestSHA: published.ManifestSHA, Reason: "Activate this isolated HTTP technical fixture only."}, &published, 200)
	if w := privateRequest(h, "GET", "/api/v1/knowledge/"+p.Knowledge[0].ID, "", nil); w.Code != 200 {
		t.Fatal("activated HTTP content unavailable")
	}
	var withdrawal publication.WithdrawalResult
	contentHTTPCall(t, h, admin, "POST", "/api/v1/content/withdrawals", publication.WithdrawalInput{Target: publication.WithdrawalTarget{Kind: "knowledge", ID: p.Knowledge[0].ID, Version: 1}, ExpectedHead: &published.ID, Reason: "Withdraw the isolated HTTP fixture and preserve its history."}, &withdrawal, 201)
	if w := privateRequest(h, "GET", "/api/v1/knowledge/"+p.Knowledge[0].ID, "", nil); w.Code != 404 {
		t.Fatal("withdrawn HTTP content public")
	}
	largeLogin := `{"username":"content_http_editor","password":"` + strings.Repeat("x", 8193) + `"}`
	if w := privateRequest(h, "POST", "/api/v1/auth/login", largeLogin, editor); w.Code != 400 {
		t.Fatal("old account 8 KiB boundary changed")
	}
}
func TestContentReadinessDoesNotMigrate(t *testing.T) {
	db := testutil.Database(t)
	ready, err := contentReady(context.Background(), db)
	if err != nil || ready {
		t.Fatal("absent schema incorrectly configured")
	}
	var exists bool
	if err = db.QueryRow(`SELECT to_regclass('public.content_workspaces') IS NOT NULL`).Scan(&exists); err != nil || exists {
		t.Fatal("readiness check created schema")
	}
}
