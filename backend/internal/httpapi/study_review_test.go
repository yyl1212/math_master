package httpapi

import (
	"context"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/studyreview"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func studyReviewHTTP(t *testing.T) *sharingHTTPFixture {
	t.Helper()
	f := sharingHTTP(t)
	path := "/api/v3/study/knowledge/" + f.k.ID
	var begun, done knowledgeadmin.ManagedDetail
	contentHTTPCall(t, f.h, f.learner, "POST", path+"/begin", knowledgeadmin.ManagedStudyInput{Knowledge: f.k.Ref}, &begun, 200)
	contentHTTPCall(t, f.h, f.learner, "POST", path+"/complete", knowledgeadmin.ManagedStudyInput{Knowledge: f.k.Ref, ExpectedSequence: begun.Record.Sequence}, &done, 200)
	return f
}
func TestStudyReviewHTTPBoundsAndPermissions(t *testing.T) {
	f := studyReviewHTTP(t)
	base := "/api/v3/study/review"
	for _, path := range []string{"/overview", "/queue", "/history", "/sessions/11111111-1111-4111-8111-111111111111"} {
		w := privateRequest(f.h, "GET", base+path, "", nil)
		if w.Code != 401 {
			t.Fatal(path, w.Code)
		}
	}
	h := privateRequest(f.h, "GET", base+"/health", "", nil)
	if h.Code != 200 || h.Header().Get("Cache-Control") != "no-store" || strings.Contains(h.Body.String(), "actorId") {
		t.Fatal("public health", h.Code, h.Body.String())
	}
	for _, path := range []string{"/queue?limit=51", "/queue?limit=0", "/queue?q=a&q=b", "/queue?limit=01", "/queue?offset=100001", "/queue?unknown=x", "/queue?topicKey=bad", "/overview?q=x", "/history?mode=exam", "/history?cursor=bad", "/sessions?body=x"} {
		w := privateRequest(f.h, "GET", base+path, "", f.learner)
		if w.Code != 400 && path != "/sessions?body=x" {
			t.Fatal(path, w.Code, w.Body.String())
		}
	}
	if w := privateRequest(f.h, "GET", base+"/queue", "{}", f.learner); w.Code != 400 {
		t.Fatal("GET body", w.Code)
	}
	if w := privateRequest(f.h, "POST", base+"/queue", "{}", f.learner); w.Code != 405 || w.Header().Get("Allow") != "GET" {
		t.Fatal("method", w.Code)
	}
	headers := map[string]string{}
	for k, v := range f.learner {
		headers[k] = v
	}
	headers["Idempotency-Key"] = "bounds"
	for _, body := range []string{`{"knowledge":null,"mode":"read","target":"statement"}`, `{"knowledge":{},"mode":"read","target":"statement","reference":"forged"}`, `{"knowledge":{},"mode":"read","mode":"recall","target":"statement"}`} {
		if w := privateRequest(f.h, "POST", base+"/sessions", body, headers); w.Code != 422 {
			t.Fatal("strict body", w.Code, w.Body.String())
		}
	}
	if w := privateRequest(f.h, "POST", base+"/sessions", strings.Repeat("x", 131073), headers); w.Code != 413 {
		t.Fatal("request limit", w.Code)
	}
	headers["Origin"] = "https://foreign.example"
	if w := privateRequest(f.h, "POST", base+"/sessions", "{}", headers); w.Code != 403 {
		t.Fatal("origin", w.Code)
	}
	headers["Origin"] = privateOrigin
	delete(headers, "Idempotency-Key")
	if w := privateRequest(f.h, "POST", base+"/sessions", "{}", headers); w.Code != 400 {
		t.Fatal("key", w.Code)
	}
	var started studyreview.CommandReceipt
	contentHTTPCall(t, f.h, f.learner, "POST", base+"/sessions", studyreview.StartInput{Knowledge: f.k.Ref, Mode: studyreview.Recall, Target: studyreview.Statement}, &started, 200)
	idPath := base + "/sessions/" + started.SessionID
	for _, action := range []struct{ method, path, body string }{{"PUT", "/draft", `{"expectedSequence":1,"body":"missing revision"}`}, {"POST", "/compare", `{"expectedSequence":1,"body":"missing bool"}`}, {"POST", "/compare", `{"expectedSequence":1,"body":"x","unanswered":null}`}, {"POST", "/grade", `{"expectedSequence":1,"rating":"correct"}`}, {"POST", "/abandon", `{"expectedSequence":null}`}} {
		headers["Idempotency-Key"] = "invalid-fields"
		if w := privateRequest(f.h, action.method, idPath+action.path, action.body, headers); w.Code != 422 {
			t.Fatal(action.path, w.Code, w.Body.String())
		}
	}
	for _, actor := range []map[string]string{f.admin, nil} {
		if w := privateRequest(f.h, "GET", idPath, "", actor); w.Code != 404 && actor != nil || w.Code != 401 && actor == nil {
			t.Fatal("owner permission", w.Code)
		}
	}
	w := privateRequest(f.h, "GET", idPath, "", f.learner)
	if w.Code != 200 || w.Header().Get("Cache-Control") != "private, no-store" || len(w.Result().Cookies()) != 0 {
		t.Fatal("private session", w.Code)
	}
}
func TestRecallExpectedAbsentBeforeCompare(t *testing.T) {
	f := studyReviewHTTP(t)
	base := "/api/v3/study/review/sessions"
	var r, c, g studyreview.CommandReceipt
	contentHTTPCall(t, f.h, f.learner, "POST", base, studyreview.StartInput{Knowledge: f.k.Ref, Mode: studyreview.Recall, Target: studyreview.Statement}, &r, 200)
	path := base + "/" + r.SessionID
	w := privateRequest(f.h, "GET", path, "", f.learner)
	expected := f.k.Point["statement"].(string)
	var s studyreview.Session
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &s) != nil || s.Reference != nil || len(s.Prompt) != 0 || strings.Contains(w.Body.String(), expected) {
		t.Fatal("reference revealed", w.Code, w.Body.String())
	}
	contentHTTPCall(t, f.h, f.learner, "POST", path+"/compare", studyreview.CompareInput{ExpectedSequence: r.Sequence, Body: "自己写的思考"}, &c, 200)
	if c.Action != "compare" {
		t.Fatal(c)
	}
	w = privateRequest(f.h, "GET", path, "", f.learner)
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &s) != nil || s.Reference == nil || *s.Answer != "自己写的思考" {
		t.Fatal("compare current read", w.Code)
	}
	contentHTTPCall(t, f.h, f.learner, "POST", path+"/grade", studyreview.GradeInput{ExpectedSequence: c.Sequence, Rating: studyreview.Familiar}, &g, 200)
	if g.Grade == nil || g.Grade.Schedule.Stage != 1 {
		t.Fatal("grade", g)
	}
}
func TestStudyReviewResponseWireLimit(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v3/study/review/queue", nil)
	w := httptest.NewRecorder()
	studyReviewResponse(w, r, map[string]string{"answer": strings.Repeat("x", 1048577)}, 200, true)
	if w.Code != 503 || strings.Contains(w.Body.String(), strings.Repeat("x", 100)) || !json.Valid(w.Body.Bytes()) {
		t.Fatal("response oversized/leaked", w.Code)
	}
}

func TestProofPromptAllSourceRedacted(t *testing.T) {
	f := sharingHTTP(t)
	page, e := f.repo.ListCurrentKnowledge(context.Background(), knowledgeadmin.Query{Limit: 20})
	if e != nil {
		t.Fatal(e)
	}
	var proof knowledgeadmin.PublicKnowledge
	for _, summary := range page.Items {
		k, e := f.repo.ReadCurrentKnowledge(context.Background(), summary.ID)
		if e != nil {
			t.Fatal(e)
		}
		if body, ok := k.Point["proof"].(string); ok && body != "" {
			proof = k
			break
		}
	}
	if proof.ID == "" {
		t.Fatal("real proof fixture missing")
	}
	var begun, completed knowledgeadmin.ManagedDetail
	oldPath := "/api/v3/study/knowledge/" + proof.ID
	contentHTTPCall(t, f.h, f.learner, "POST", oldPath+"/begin", knowledgeadmin.ManagedStudyInput{Knowledge: proof.Ref}, &begun, 200)
	contentHTTPCall(t, f.h, f.learner, "POST", oldPath+"/complete", knowledgeadmin.ManagedStudyInput{Knowledge: proof.Ref, ExpectedSequence: begun.Record.Sequence}, &completed, 200)
	var r studyreview.CommandReceipt
	base := "/api/v3/study/review/sessions"
	contentHTTPCall(t, f.h, f.learner, "POST", base, studyreview.StartInput{Knowledge: proof.Ref, Mode: studyreview.Recall, Target: studyreview.Proof}, &r, 200)
	path := base + "/" + r.SessionID
	w := privateRequest(f.h, "GET", path, "", f.learner)
	var s studyreview.Session
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &s) != nil || len(s.Prompt) == 0 || s.Reference != nil {
		t.Fatal("proof projection", w.Code)
	}
	var admin knowledgeadmin.Knowledge
	contentHTTPCall(t, f.h, f.admin, "GET", "/api/v3/admin/knowledge/"+proof.ID, nil, &admin, 200)
	f.admin["If-Match"] = admin.EditToken
	for _, a := range []struct {
		path  string
		input any
	}{{"unpublish", map[string]string{}}} {
		var changed knowledgeadmin.Knowledge
		contentHTTPCall(t, f.h, f.admin, "POST", "/api/v3/admin/knowledge/"+proof.ID+"/"+a.path, a.input, &changed, 200)
	}
	w = privateRequest(f.h, "GET", path, "", f.learner)
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &s) != nil || s.Available || len(s.Prompt) != 0 || s.Reference != nil {
		t.Fatal("withdrawn proof prompt leaked", w.Code, w.Body.String())
	}
}

type cancelledReviewHealth struct {
	studyreview.Repository
	deadline time.Duration
}

func (s *cancelledReviewHealth) ReadReviewHealth(ctx context.Context) (studyreview.Health, error) {
	d, ok := ctx.Deadline()
	if ok {
		s.deadline = time.Until(d)
	}
	<-ctx.Done()
	return studyreview.Health{}, ctx.Err()
}
func TestStudyReviewHTTPDeadlineAndSafeHealth(t *testing.T) {
	repo := &cancelledReviewHealth{}
	h := NewApplicationHandler(&fakeReader{}, nil, AuthOptions{Knowledge: &KnowledgeOptions{Reviews: repo, PublicOrigin: privateOrigin}})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := httptest.NewRequest("GET", "/api/v3/study/review/health", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 503 || repo.deadline > 8*time.Second || repo.deadline < 7*time.Second || w.Body.String() != `{"capability":true,"schemaReady":false}` {
		t.Fatal("HTTP cancellation/deadline/health", w.Code, repo.deadline, w.Body.String())
	}
}
