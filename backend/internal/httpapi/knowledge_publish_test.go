package httpapi

import (
	"context"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/auth"
	k "github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/pagination"
	"net/http"
	"strings"
	"testing"
	"time"
)

type publishHTTPRepo struct {
	*httpKnowledgeRepo
	k.PublishRepository
}
type publishHealthHTTPRepo struct {
	*httpKnowledgeRepo
	ready bool
}

func (r *publishHealthHTTPRepo) ReadPublishHealth(context.Context) (k.PublishHealth, error) {
	return k.PublishHealth{Capability: true, SchemaReady: r.ready}, nil
}
func TestKnowledgePublishHealthPublicStrict(t *testing.T) {
	repo := &publishHealthHTTPRepo{httpKnowledgeRepo: &httpKnowledgeRepo{}}
	h := KnowledgeHandler(k.NewService(repo), KnowledgeOptions{PublicOrigin: privateOrigin})
	path := "/api/v3/knowledge-publish/health"
	for _, ready := range []bool{false, true} {
		repo.ready = ready
		w := privateRequest(h, "GET", path, "", nil)
		want := 503
		if ready {
			want = 200
		}
		if w.Code != want || strings.TrimSpace(w.Body.String()) != fmt.Sprintf(`{"capability":true,"schemaReady":%t}`, ready) {
			t.Fatal(w.Code, w.Body.String())
		}
		if w.Header().Get("Set-Cookie") != "" {
			t.Fatal("health set cookie")
		}
	}
	for _, query := range []string{"?unknown=1", "?limit=1", "?"} {
		if w := privateRequest(h, "GET", path+query, "", nil); w.Code != 422 {
			t.Fatal(query, w.Code)
		}
	}
	if w := privateRequest(h, "GET", path, "{}", nil); w.Code != 422 {
		t.Fatal(w.Code)
	}
	if w := privateRequest(h, "POST", path, "", nil); w.Code != 405 {
		t.Fatal(w.Code)
	}
	if w := privateRequest(KnowledgeHandler(nil), "GET", path, "", nil); w.Code != 503 {
		t.Fatal(w.Code)
	}
}

func (r *publishHTTPRepo) CreatePublishJob(_ context.Context, _ k.Access, in k.PublishCreateInput) (k.PublishJob, error) {
	r.calls++
	return k.PublishJob{ActorID: contentFixtureID, ID: contentFixtureID, Mode: in.Mode, Sequence: 1, CreatedAt: time.Now()}, nil
}
func (r *publishHTTPRepo) ListPublishJobs(_ context.Context, _ k.Access, _ k.PublishJobQuery) (pagination.ActorPage[k.PublishJob], error) {
	if !auth.HasRole(r.user, auth.RoleAdmin) {
		return pagination.ActorPage[k.PublishJob]{}, auth.ErrForbidden
	}
	return pagination.ActorPage[k.PublishJob]{ActorID: r.user.ID}, nil
}
func TestKnowledgePublishHTTPPermissionsAndBounds(t *testing.T) {
	base := &httpKnowledgeRepo{user: auth.User{ID: contentFixtureID, Roles: []auth.Role{auth.RoleAdmin}}, mode: "managed"}
	repo := &publishHTTPRepo{httpKnowledgeRepo: base}
	h := KnowledgeHandler(k.NewService(repo), KnowledgeOptions{PublicOrigin: privateOrigin})
	path := "/api/v3/admin/knowledge/publish-jobs"
	if w := privateRequest(h, "POST", path, `{"mode":"all"}`, contentHeaders()); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, body := range []string{`{"mode":"all","items":[]}`, `{"mode":"all","items":null}`, `{"mode":"selected","items":[]}`, `{"mode":"all","unknown":1}`, `{"mode":"all","mode":"all"}`} {
		if w := privateRequest(h, "POST", path, body, contentHeaders()); w.Code != 422 {
			t.Fatal(body, w.Code, w.Body.String())
		}
	}
	if w := privateRequest(h, "POST", path, `{"mode":"all"}`, nil); w.Code != 401 {
		t.Fatal(w.Code)
	}
	base.user.Roles = []auth.Role{auth.RoleLearner}
	if w := privateRequest(h, "POST", path, `{"mode":"all"}`, contentHeaders()); w.Code != 403 {
		t.Fatal(w.Code)
	}
	base.user.Roles = []auth.Role{auth.RoleAdmin}
	for _, q := range []string{"?limit=1&limit=2", "?actorId=bad", "?offset=9007199254740992"} {
		if w := privateRequest(h, "GET", path+q, "", contentHeaders()); w.Code != 422 {
			t.Fatal(q, w.Code)
		}
	}
	if repo.calls != 1 {
		t.Fatal("invalid input reached repository", repo.calls)
	}
	_ = http.MethodGet
}

func TestKnowledgePublishHTTPBodyLimit(t *testing.T) {
	repo := &publishHTTPRepo{httpKnowledgeRepo: &httpKnowledgeRepo{user: auth.User{ID: contentFixtureID, Roles: []auth.Role{auth.RoleAdmin}}, mode: "managed"}}
	h := KnowledgeHandler(k.NewService(repo), KnowledgeOptions{PublicOrigin: privateOrigin})
	path := "/api/v3/admin/knowledge/publish-jobs"
	body := `{"mode":"all"}`
	for _, n := range []int{262144, 262145} {
		raw := body + strings.Repeat(" ", n-len(body))
		w := privateRequest(h, "POST", path, raw, contentHeaders())
		want := 200
		if n == 262145 {
			want = 413
		}
		if w.Code != want {
			t.Fatal(n, w.Code, w.Body.String())
		}
	}
	headers := contentHeaders()
	headers["Origin"] = "https://outside.example"
	if w := privateRequest(h, "POST", path, body, headers); w.Code != 403 {
		t.Fatal(w.Code)
	}
}
