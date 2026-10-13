package httpapi

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/notesharing"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

type sharingHTTPFixture struct {
	db             *sql.DB
	repo           *store.Store
	h              http.Handler
	admin, learner map[string]string
	k              knowledgeadmin.PublicKnowledge
}

func sharingHTTP(t *testing.T) *sharingHTTPFixture {
	t.Helper()
	ctx := context.Background()
	db := testutil.Database(t)
	if e := store.Up(ctx, db, "../../../db/migrations"); e != nil {
		t.Fatal(e)
	}
	repo := store.New(db)
	hasher := auth.NewArgon2Hasher(rand.Reader)
	accounts, e := auth.NewService(repo, hasher, rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	h := NewApplicationHandler(repo, db, AuthOptions{Accounts: accounts, PublicOrigin: privateOrigin, Knowledge: &KnowledgeOptions{Service: knowledgeadmin.NewService(repo), Current: repo, Study: repo, Notes: repo, PublicOrigin: privateOrigin}})
	admin, u := contentRealActor(t, h, "sharing_http_admin")
	learner, _ := contentRealActor(t, h, "sharing_http_learner")
	if _, e = db.Exec("INSERT INTO auth_user_roles(user_id,role) VALUES($1,'admin')", u.ID); e != nil {
		t.Fatal(e)
	}
	c := admin["Cookie"]
	secret := strings.TrimPrefix(c, "mm_session_dev=")
	proof, e := auth.DecodeContentProof(auth.Cookies{Session: secret}, admin["X-CSRF-Token"], true)
	if e != nil {
		t.Fatal(e)
	}
	a := knowledgeadmin.Access{TokenHash: proof.TokenHash, CSRF: proof.CSRF, IdempotencyKey: "http-preview", RequestID: "sharing-http-fixture"}
	b, e := os.ReadFile("../knowledgeadmin/testdata/valid-source.json")
	if e != nil {
		t.Fatal(e)
	}
	doc, e := knowledgeadmin.DecodeSource(bytes.NewReader(b))
	if e != nil {
		t.Fatal(e)
	}
	p, e := repo.PreviewManagedImport(ctx, a, doc, knowledgeadmin.DigestBytes(b))
	if e != nil {
		t.Fatal(e)
	}
	indexes := []int{}
	for _, item := range p.Items {
		if item.Action == "create" || item.Action == "link" {
			indexes = append(indexes, item.Index)
		}
	}
	a.IdempotencyKey = "http-apply"
	if _, e = repo.ApplyManagedImport(ctx, a, p.ImportID, knowledgeadmin.ApplyInput{SelectedIndexes: indexes, Publish: true, PreviewToken: p.PreviewToken}); e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec("UPDATE knowledge_admin_state SET content_mode='managed',enabled_once=true,activated_at=clock_timestamp() WHERE singleton; UPDATE goose_db_version SET managed_knowledge_enabled=true WHERE version_id=0"); e != nil {
		t.Fatal(e)
	}
	k, e := repo.ReadCurrentKnowledge(ctx, knowledgeadmin.KnowledgeID(doc.KnowledgePoints[0].ID))
	if e != nil {
		t.Fatal(e)
	}
	return &sharingHTTPFixture{db, repo, h, admin, learner, k}
}
func TestSharingRoutePermissions(t *testing.T) {
	f := sharingHTTP(t)
	for _, path := range []string{"/api/v3/study/notes", "/api/v3/shared-notes", "/api/v3/admin/note-shares"} {
		if w := privateRequest(f.h, "GET", path, "", nil); w.Code != 401 {
			t.Fatal(path, w.Code)
		}
	}
	if w := privateRequest(f.h, "GET", "/api/v3/admin/note-shares", "", f.learner); w.Code != 403 {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, role := range []string{"editor", "reviewer"} {
		headers, u := contentRealActor(t, f.h, "sharing_http_"+role)
		if _, e := f.db.Exec("INSERT INTO auth_user_roles(user_id,role) VALUES($1,$2)", u.ID, role); e != nil {
			t.Fatal(e)
		}
		if w := privateRequest(f.h, "GET", "/api/v3/admin/note-shares", "", headers); w.Code != 403 {
			t.Fatal("old role approved notes", role, w.Code)
		}
	}
	if w := privateRequest(f.h, "GET", "/api/v3/shared-notes/latest", "", nil); w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal(w.Code, w.Body.String())
	}
}
func TestSharingRequestBounds(t *testing.T) {
	f := sharingHTTP(t)
	for _, path := range []string{"/api/v3/shared-notes/latest?limit=50", "/api/v3/shared-notes/latest?q=private", "/api/v3/study/notes?limit=51", "/api/v3/study/notes?q=a&q=b", "/api/v3/study/notes?unknown=a"} {
		if w := privateRequest(f.h, "GET", path, "", f.learner); w.Code != 400 {
			t.Fatal(path, w.Code, w.Body.String())
		}
	}
	if w := privateRequest(f.h, "GET", "/api/v3/shared-notes/latest", "{}", nil); w.Code != 400 {
		t.Fatal("GET body", w.Code)
	}
	if w := privateRequest(f.h, "GET", "/api/v3/study/notes?cursor=not-a-cursor", "", f.learner); w.Code != 400 {
		t.Fatal("invalid cursor status", w.Code, w.Body.String())
	}
	path := "/api/v3/study/knowledge/" + f.k.ID + "/note-sharing/submit"
	headers := map[string]string{}
	for k, v := range f.learner {
		headers[k] = v
	}
	headers["Idempotency-Key"] = "invalid-input"
	if w := privateRequest(f.h, "POST", path, `{"expectedRevision":1,"expectedSequence":0,"title":"笔记","body":"forged"}`, headers); w.Code != 422 {
		t.Fatal("caller supplied snapshot", w.Code)
	}
	if w := privateRequest(f.h, "POST", path, strings.Repeat("x", 131073), headers); w.Code != 413 {
		t.Fatal("oversized request", w.Code)
	}
	headers["Origin"] = "https://foreign.example"
	if w := privateRequest(f.h, "POST", path, "{}", headers); w.Code != 403 {
		t.Fatal("foreign origin", w.Code)
	}
}

func TestSharingResponseBounds(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/v3/shared-notes/latest", nil)
	r.Header.Set("X-Request-ID", "response-limit-test")
	w := httptest.NewRecorder()
	noteShareResponse(w, r, map[string]string{"body": strings.Repeat("x", notesharing.MaxDetailBytes+1)}, 200, false, notesharing.MaxDetailBytes)
	if w.Code != 503 || !json.Valid(w.Body.Bytes()) || strings.Contains(w.Body.String(), strings.Repeat("x", 100)) {
		t.Fatal("oversized response leaked or truncated", w.Code)
	}
}
func TestSharingStoreHTTPExactErrors(t *testing.T) {
	f := sharingHTTP(t)
	path := "/api/v3/study/knowledge/" + f.k.ID
	var n knowledgeadmin.ManagedNote
	contentHTTPCall(t, f.h, f.learner, "PUT", path+"/note", knowledgeadmin.ManagedNoteInput{Knowledge: f.k.Ref, Body: "真实HTTP保存的笔记"}, &n, 200)
	f.learner["Idempotency-Key"] = "share-exact-once"
	raw, _ := json.Marshal(notesharing.SubmitInput{ExpectedRevision: n.Revision, Title: "真实分享"})
	w := privateRequest(f.h, "POST", path+"/note-sharing/submit", string(raw), f.learner)
	var r notesharing.Receipt
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &r) != nil {
		t.Fatal(w.Code, w.Body.String())
	}
	assertCode := func(method, url, body string, headers map[string]string, status int, code string) {
		t.Helper()
		w := privateRequest(f.h, method, url, body, headers)
		if w.Code != status || !strings.Contains(w.Body.String(), code) {
			t.Fatal(url, w.Code, w.Body.String())
		}
	}
	f.learner["Idempotency-Key"] = "share-exact-once"
	assertCode("POST", path+"/note-sharing/submit", strings.Replace(string(raw), "真实分享", "不同分享", 1), f.learner, 409, "IDEMPOTENCY_CONFLICT")
	f.learner["Idempotency-Key"] = "locked-save"
	b, _ := json.Marshal(knowledgeadmin.ManagedNoteInput{Knowledge: f.k.Ref, ExpectedRevision: n.Revision, Body: "未经批准的修改"})
	assertCode("PUT", path+"/note", string(b), f.learner, 409, "NOTE_SHARING_LOCKED")
	contentHTTPCall(t, f.h, f.learner, "POST", path+"/note-sharing/withdraw", notesharing.WithdrawInput{SubmissionID: r.SubmissionID, ExpectedSequence: r.Sequence}, nil, 200)
	f.admin["Idempotency-Key"] = "stale-approval"
	b, _ = json.Marshal(notesharing.DecisionInput{ExpectedSequence: r.Sequence})
	assertCode("POST", "/api/v3/admin/note-shares/"+r.SubmissionID+"/approve", string(b), f.admin, 409, "NOTE_SHARE_STALE")
	f.learner["Idempotency-Key"] = "invalid-title"
	assertCode("POST", path+"/note-sharing/submit", `{"expectedRevision":1,"expectedSequence":2,"title":""}`, f.learner, 422, "INVALID_FIELD")
	if _, e := f.db.Exec("ALTER TABLE note_share_heads DROP CONSTRAINT note_share_head_status"); e != nil {
		t.Fatal(e)
	}
	assertCode("GET", "/api/v3/study/notes", "", f.learner, 503, "NOTE_SHARING_NOT_READY")
}
func TestSharingHealthRealSchema(t *testing.T) {
	f := sharingHTTP(t)
	w := privateRequest(f.h, "GET", "/api/v3/note-sharing/health", "", nil)
	var health map[string]bool
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &health) != nil || len(health) != 2 || !health["capability"] || !health["schemaReady"] {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, path := range []string{"/api/v3/note-sharing/health?unknown=true", "/api/v3/note-sharing/health?limit=10"} {
		if w = privateRequest(f.h, "GET", path, "", nil); w.Code != 400 {
			t.Fatal(w.Code)
		}
	}
	if w = privateRequest(f.h, "GET", "/api/v3/note-sharing/health", "{}", nil); w.Code != 400 {
		t.Fatal(w.Code)
	}
	if _, e := f.db.Exec("ALTER TABLE note_share_heads DROP CONSTRAINT note_share_head_status"); e != nil {
		t.Fatal(e)
	}
	w = privateRequest(f.h, "GET", "/api/v3/note-sharing/health", "", nil)
	if w.Code != 503 || json.Unmarshal(w.Body.Bytes(), &health) != nil || health["schemaReady"] || len(health) != 2 {
		t.Fatal(w.Code, w.Body.String())
	}
}
