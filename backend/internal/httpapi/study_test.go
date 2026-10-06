package httpapi

import (
	"context"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/study"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"net/http"
	"strings"
	"testing"
)

type httpStudyRepo struct {
	study.Repository
	user  auth.User
	calls int
}

func (r *httpStudyRepo) StudyPreflight(context.Context, study.Access, study.Action) (auth.User, error) {
	return r.user, nil
}
func (r *httpStudyRepo) ConsumeRates(context.Context, []auth.RateKey) error { return nil }
func (r *httpStudyRepo) BeginStudy(_ context.Context, _ study.Access, id string, in study.CommandInput) (study.StudyDetail, error) {
	r.calls++
	return study.StudyDetail{ActorID: r.user.ID, Record: study.StudyRecord{KnowledgeID: id, State: study.Learning, Sequence: 1}, Pair: taxonomy.PairRef{TaxonomyVersionID: strings.Repeat("a", 64)}}, nil
}
func (r *httpStudyRepo) SaveStudyNote(_ context.Context, _ study.Access, id string, in study.NoteInput) (study.NoteReceipt, error) {
	r.calls++
	return study.NoteReceipt{ActorID: r.user.ID, KnowledgeID: id, Revision: 1}, nil
}
func (r *httpStudyRepo) ReadStudyOverview(context.Context, study.Access) (study.Overview, error) {
	return study.Overview{ActorID: r.user.ID, Mode: taxonomy.ModeTopics, Reminders: []study.ContentReminder{}}, nil
}
func studyHTTPFixture(t *testing.T) (http.Handler, *httpStudyRepo) {
	t.Helper()
	repo := &httpStudyRepo{user: auth.User{ID: contentFixtureID, Roles: []auth.Role{auth.RoleLearner}}}
	service, e := study.NewService(repo)
	if e != nil {
		t.Fatal(e)
	}
	h := NewApplicationHandler(&fakeReader{}, nil, AuthOptions{Study: &StudyOptions{Service: service, PublicOrigin: privateOrigin}, PublicOrigin: privateOrigin})
	return h, repo
}
func TestStudyHTTPBodyBoundaries(t *testing.T) {
	h, repo := studyHTTPFixture(t)
	command := study.CommandInput{Knowledge: study.KnowledgeRef{ID: "fractions", Version: 1, SHA256: strings.Repeat("a", 64)}, ExpectedKnowledgeHead: contentFixtureID}
	raw, _ := json.Marshal(command)
	for _, n := range []int{8192, 8193} {
		body := string(raw) + strings.Repeat(" ", n-len(raw))
		w := privateRequest(h, "POST", "/api/v2/study/knowledge/fractions/begin", body, contentHeaders())
		want := 200
		if n > 8192 {
			want = 413
		}
		if w.Code != want {
			t.Fatal("control bytes", n, w.Code)
		}
	}
	note := study.NoteInput{Knowledge: command.Knowledge, Body: "Safe private note."}
	raw, _ = json.Marshal(note)
	for _, n := range []int{131072, 131073} {
		body := string(raw) + strings.Repeat(" ", n-len(raw))
		w := privateRequest(h, "PUT", "/api/v2/study/knowledge/fractions/note", body, contentHeaders())
		want := 200
		if n > 131072 {
			want = 413
		}
		if w.Code != want {
			t.Fatal("note bytes", n, w.Code)
		}
	}
	before := repo.calls
	raw, _ = json.Marshal(command)
	for _, body := range []string{string(raw) + "{}", strings.Replace(string(raw), `"expectedSequence":0`, `"expectedSequence":0,"ExpectedSequence":0`, 1), strings.Replace(string(raw), `"knowledge":`, `"actorId":"forged","knowledge":`, 1), `{"knowledge":null}`} {
		w := privateRequest(h, "POST", "/api/v2/study/knowledge/fractions/begin", body, contentHeaders())
		if w.Code != 400 {
			t.Fatal("unsafe JSON", w.Code)
		}
	}
	if repo.calls != before {
		t.Fatal("unsafe input reached storage")
	}
}
func TestStudyHTTPAnonymousOriginAndQueries(t *testing.T) {
	h, _ := studyHTTPFixture(t)
	for _, path := range []string{"/api/v2/study/overview", "/api/v2/study/topics", "/api/v2/study/knowledge", "/api/v2/study/history", "/api/v2/study/knowledge/fractions/note"} {
		if w := privateRequest(h, "GET", path, "", nil); w.Code != 401 {
			t.Fatal("anonymous private route", path, w.Code)
		}
	}
	if w := privateRequest(h, "GET", "/api/v2/study/overview", "", contentHeaders()); w.Code != 200 || w.Header().Get("Cache-Control") != "private, no-store" || len(w.Result().Cookies()) != 0 {
		t.Fatal("private response contract", w.Code, w.Header())
	}
	for _, path := range []string{"/api/v2/study/history?limit=51", "/api/v2/study/knowledge?q=%FF", "/api/v2/study/knowledge?reviewOnly=yes", "/api/v2/study/knowledge?actorId=" + contentFixtureID, "/api/v2/study/overview?unexpected=1", "/api/v2/study/knowledge/fractions?limit=1"} {
		if w := privateRequest(h, "GET", path, "", contentHeaders()); w.Code != 400 {
			t.Fatal("unsafe query", path, w.Code)
		}
	}
	headers := contentHeaders()
	headers["Origin"] = "https://foreign.example"
	if w := privateRequest(h, "POST", "/api/v2/study/knowledge/fractions/begin", "{}", headers); w.Code != 403 {
		t.Fatal("foreign origin", w.Code)
	}
}
