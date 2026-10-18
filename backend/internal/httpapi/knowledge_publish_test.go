package httpapi

import (
	"context"
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
