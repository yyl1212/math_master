package httpapi

import (
	"context"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/study"
	"strings"
	"testing"
)

type httpManagedStudy struct {
	knowledgeadmin.StudyRepository
	calls int
}

func (r *httpManagedStudy) ReadManagedOverview(context.Context, knowledgeadmin.Access) (knowledgeadmin.ManagedOverview, error) {
	return knowledgeadmin.ManagedOverview{ActorID: contentFixtureID, Reminders: []knowledgeadmin.ManagedReminder{}}, nil
}
func (r *httpManagedStudy) ApplyManagedStudy(_ context.Context, _ knowledgeadmin.Access, id, action string, in knowledgeadmin.ManagedStudyInput) (knowledgeadmin.ManagedDetail, error) {
	r.calls++
	return knowledgeadmin.ManagedDetail{ActorID: contentFixtureID, Record: knowledgeadmin.ManagedRecord{KnowledgeID: id, State: study.Learning, Sequence: 1}, Available: true}, nil
}
func TestManagedStudyHTTPPrivateBoundary(t *testing.T) {
	repo := &httpManagedStudy{}
	h := NewApplicationHandler(&fakeReader{}, nil, AuthOptions{Knowledge: &KnowledgeOptions{Study: repo, PublicOrigin: privateOrigin}, PublicOrigin: privateOrigin})
	if w := privateRequest(h, "GET", "/api/v3/study/overview", "", nil); w.Code != 401 {
		t.Fatal("anonymous private study", w.Code)
	}
	if w := privateRequest(h, "GET", "/api/v3/study/overview", "", contentHeaders()); w.Code != 200 || w.Header().Get("Cache-Control") != "private, no-store" {
		t.Fatal(w.Code)
	}
	id := knowledgeadmin.KnowledgeID("raw-point")
	in := knowledgeadmin.ManagedStudyInput{Knowledge: knowledgeadmin.Ref{ID: id, ContentSHA256: strings.Repeat("a", 64), SourceKind: "managed"}, ExpectedSequence: 0}
	b, _ := json.Marshal(in)
	if w := privateRequest(h, "POST", "/api/v3/study/knowledge/"+id+"/begin", string(b), contentHeaders()); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	forged := strings.Replace(string(b), `"knowledge":`, `"actorId":"forged","knowledge":`, 1)
	if w := privateRequest(h, "POST", "/api/v3/study/knowledge/"+id+"/begin", forged, contentHeaders()); w.Code != 422 {
		t.Fatal("forged actor accepted", w.Code)
	}
	headers := contentHeaders()
	headers["Origin"] = "https://foreign.example"
	if w := privateRequest(h, "POST", "/api/v3/study/knowledge/"+id+"/begin", string(b), headers); w.Code != 403 {
		t.Fatal("foreign origin", w.Code)
	}
	if repo.calls != 1 {
		t.Fatal("invalid input reached persistence")
	}
}
