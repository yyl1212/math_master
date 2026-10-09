package httpapi

import (
	"bufio"
	"context"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

type httpKnowledgeRepo struct {
	knowledgeadmin.Repository
	user  auth.User
	calls int
	mode  string
}

func (r *httpKnowledgeRepo) KnowledgePreflight(context.Context, knowledgeadmin.Access) (auth.User, error) {
	if !auth.HasRole(r.user, auth.RoleAdmin) {
		return r.user, auth.ErrForbidden
	}
	return r.user, nil
}
func (r *httpKnowledgeRepo) PreviewManagedImport(_ context.Context, _ knowledgeadmin.Access, d knowledgeadmin.SourceDocument, sha string) (knowledgeadmin.Preview, error) {
	r.calls++
	return knowledgeadmin.Preview{ImportID: contentFixtureID, InputSHA256: sha, PreviewToken: contentFixtureID, Items: []knowledgeadmin.PreviewItem{}}, nil
}
func (r *httpKnowledgeRepo) ReadContentMode(context.Context) (knowledgeadmin.ContentMode, error) {
	return knowledgeadmin.ContentMode{Mode: r.mode, Capability: r.mode == "managed"}, nil
}
func (r *httpKnowledgeRepo) ListCurrentKnowledge(context.Context, knowledgeadmin.Query) (knowledgeadmin.Page[knowledgeadmin.PublicKnowledge], error) {
	return knowledgeadmin.Page[knowledgeadmin.PublicKnowledge]{Items: []knowledgeadmin.PublicKnowledge{}, Limit: 20}, nil
}
func (r *httpKnowledgeRepo) ReadCurrentKnowledge(context.Context, string) (knowledgeadmin.PublicKnowledge, error) {
	return knowledgeadmin.PublicKnowledge{}, knowledgeadmin.ErrNotFound
}
func (r *httpKnowledgeRepo) ListCurrentTopics(context.Context, knowledgeadmin.Query) (knowledgeadmin.Page[knowledgeadmin.CurrentTopic], error) {
	return knowledgeadmin.Page[knowledgeadmin.CurrentTopic]{Items: []knowledgeadmin.CurrentTopic{}, Limit: 20}, nil
}
func (r *httpKnowledgeRepo) ReadCurrentTopic(context.Context, string, knowledgeadmin.Query) (knowledgeadmin.CurrentTopic, error) {
	return knowledgeadmin.CurrentTopic{}, knowledgeadmin.ErrNotFound
}
func knowledgeHTTPFixture() (http.Handler, *httpKnowledgeRepo) {
	repo := &httpKnowledgeRepo{user: auth.User{ID: contentFixtureID, Roles: []auth.Role{auth.RoleAdmin}}, mode: "managed"}
	return NewApplicationHandler(&fakeReader{}, nil, AuthOptions{Knowledge: &KnowledgeOptions{Service: knowledgeadmin.NewService(repo), Current: repo, PublicOrigin: privateOrigin}, PublicOrigin: privateOrigin}), repo
}
func TestManagedRoutesPermissionsAndRetirement(t *testing.T) {
	h, r := knowledgeHTTPFixture()
	b, e := os.ReadFile("../knowledgeadmin/testdata/valid-source.json")
	if e != nil {
		t.Fatal(e)
	}
	if w := privateRequest(h, "POST", "/api/v3/admin/knowledge/imports", string(b), nil); w.Code != 401 {
		t.Fatal("anonymous upload", w.Code)
	}
	for _, role := range []auth.Role{auth.RoleLearner, auth.RoleEditor, auth.RoleReviewer} {
		r.user.Roles = []auth.Role{role}
		if w := privateRequest(h, "POST", "/api/v3/admin/knowledge/imports", string(b), contentHeaders()); w.Code != 403 {
			t.Fatal(role, w.Code)
		}
	}
	r.user.Roles = []auth.Role{auth.RoleAdmin}
	if w := privateRequest(h, "POST", "/api/v3/admin/knowledge/imports", string(b), contentHeaders()); w.Code != 200 {
		t.Fatal("admin upload", w.Code, w.Body.String())
	}
	for _, p := range []string{"/api/v1/content/drafts", "/api/v1/content/submissions/" + contentFixtureID + "/decision", "/api/v1/content/publications/" + contentFixtureID + "/activate", "/api/v2/content/topic-assignments/" + contentFixtureID, "/api/v2/admin/publications"} {
		if w := privateRequest(h, "POST", p, "{}", contentHeaders()); w.Code != 410 {
			t.Fatal("old writer live", p, w.Code)
		}
	}
	for _, p := range []string{"/api/v1/feedback", "/api/v1/admin/users", "/api/v1/corrections"} {
		if w := privateRequest(h, "GET", p, "", contentHeaders()); w.Code == 410 {
			t.Fatal("unrelated module retired", p)
		}
	}
}

type managedDeadlineRecorder struct {
	*httptest.ResponseRecorder
	read, write time.Time
}

func (w *managedDeadlineRecorder) SetReadDeadline(t time.Time) error  { w.read = t; return nil }
func (w *managedDeadlineRecorder) SetWriteDeadline(t time.Time) error { w.write = t; return nil }
func TestManagedUploadDeadlineIsolation(t *testing.T) {
	h, _ := knowledgeHTTPFixture()
	raw, e := os.ReadFile("../knowledgeadmin/testdata/valid-source.json")
	if e != nil {
		t.Fatal(e)
	}
	w := &managedDeadlineRecorder{ResponseRecorder: httptest.NewRecorder()}
	r := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader("{}"))
	h.ServeHTTP(w, r)
	if left := time.Until(w.read); left < 3*time.Second || left > 5*time.Second {
		t.Fatal("ordinary account deadline changed", left)
	}
	srv := httptest.NewUnstartedServer(h)
	srv.Config.ReadTimeout = 15 * time.Second
	srv.Config.WriteTimeout = 15 * time.Second
	srv.Start()
	defer srv.Close()
	slow := func(delay time.Duration) (*http.Response, error) {
		conn, e := net.DialTimeout("tcp", srv.Listener.Addr().String(), time.Second)
		if e != nil {
			return nil, e
		}
		defer conn.Close()
		_ = conn.SetDeadline(time.Now().Add(70 * time.Second))
		headers := contentHeaders()
		_, e = fmt.Fprintf(conn, "POST /api/v3/admin/knowledge/imports HTTP/1.1\r\nHost: %s\r\nContent-Length: %d\r\nConnection: close\r\n", srv.Listener.Addr().String(), len(raw))
		if e != nil {
			return nil, e
		}
		for k, v := range headers {
			if _, e = fmt.Fprintf(conn, "%s: %s\r\n", k, v); e != nil {
				return nil, e
			}
		}
		if _, e = conn.Write([]byte("\r\n")); e != nil {
			return nil, e
		}
		if _, e = conn.Write(raw[:32]); e != nil {
			return nil, e
		}
		time.Sleep(delay)
		if _, e = conn.Write(raw[32:]); e != nil {
			return nil, e
		}
		return http.ReadResponse(bufio.NewReader(conn), nil)
	}
	response, e := slow(16 * time.Second)
	if e != nil || response.StatusCode != 200 {
		t.Fatal("dedicated upload failed beyond original 15 seconds", e)
	}
	response.Body.Close()
	response, e = slow(61 * time.Second)
	if e == nil {
		defer response.Body.Close()
		if response.StatusCode == 200 {
			t.Fatal("upload exceeded 60 seconds")
		}
	}
	if srv.Config.ReadTimeout != 15*time.Second || srv.Config.WriteTimeout != 15*time.Second {
		t.Fatal("global timeouts changed")
	}
}
