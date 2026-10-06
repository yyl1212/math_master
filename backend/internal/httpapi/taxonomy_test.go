package httpapi

import (
	"context"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"net/url"
	"strings"
	"testing"
)

type httpTaxonomyRepo struct {
	taxonomy.Repository
	query taxonomy.Query
}

func (r *httpTaxonomyRepo) ListTopics(_ context.Context, q taxonomy.Query) (taxonomy.Page[taxonomy.TopicSummary], error) {
	r.query = q
	return taxonomy.Page[taxonomy.TopicSummary]{Items: []taxonomy.TopicSummary{}, Total: 63, Limit: 20, Pair: taxonomy.PairRef{TaxonomyVersionID: strings.Repeat("a", 64)}}, nil
}
func TestTaxonomyHTTPQueryBoundaries(t *testing.T) {
	repo := &httpTaxonomyRepo{}
	h := NewApplicationHandler(&fakeReader{}, nil, AuthOptions{Taxonomy: &TaxonomyOptions{Service: taxonomy.NewService(repo)}})
	allowed := url.QueryEscape(strings.Repeat("汉", 170) + "aa")
	over := url.QueryEscape(strings.Repeat("汉", 171))
	cases := []struct {
		path   string
		status int
	}{{"/api/v2/topics", 200}, {"/api/v2/topics?q=" + allowed, 200}, {"/api/v2/topics?q=" + over, 400}, {"/api/v2/topics?q=a&q=b", 400}, {"/api/v2/topics?limit=101", 400}, {"/api/v2/topics?offset=-1", 400}, {"/api/v2/topics?level=4", 400}, {"/api/v2/topics?q=%FF", 400}, {"/api/v2/topics?q=%00", 400}, {"/api/v2/topics?kind=auxiliary", 200}, {"/api/v2/topics?kind=other", 200}, {"/api/v2/topics/msc-13c60/arbitrary", 404}, {"/api/v2/topics/", 404}, {"/api/v2/topics?unexpected=1", 400}}
	for _, tc := range cases {
		w := privateRequest(h, "GET", tc.path, "", nil)
		if w.Header().Get("Cache-Control") != "private, no-store" {
			t.Fatal("topic response cache contract", w.Header().Get("Cache-Control"))
		}
		if w.Code != tc.status {
			t.Fatalf("%s: %d", tc.path, w.Code)
		}
		if len(w.Result().Cookies()) != 0 || strings.Contains(w.Body.String(), "sourceRefs") {
			t.Fatal("public identity or source leak")
		}
	}
	w := privateRequest(h, "GET", "/api/v2/topics?kind=auxiliary", "", map[string]string{"Cookie": "unrelated=bad"})
	if w.Code != 200 || repo.query.Kind != "auxiliary" {
		t.Fatal("public query read identity or lost kind")
	}
	w = privateRequest(h, "HEAD", "/api/v2/topics", "", nil)
	if w.Code != 405 {
		t.Fatal("unexpected method", w.Code)
	}
}
func TestTaxonomyPrivateAnonymousAndJSON(t *testing.T) {
	h := NewApplicationHandler(&fakeReader{}, nil, AuthOptions{Taxonomy: &TaxonomyOptions{Service: taxonomy.NewService(&httpTaxonomyRepo{}), PublicOrigin: privateOrigin}})
	for _, path := range []string{"/api/v2/content/topic-assignments/drafts/" + contentFixtureID, "/api/v2/content/topic-assignments/submissions/" + contentFixtureID, "/api/v2/admin/publications/" + contentFixtureID} {
		w := privateRequest(h, "GET", path, "", nil)
		if w.Code != 401 {
			t.Fatal(path, w.Code)
		}
	}
	in := taxonomy.DraftTopicInput{ExpectedDraftRevision: 1, ExpectedAssignmentRevision: 0, TaxonomyVersionID: strings.Repeat("a", 64), Member: taxonomy.AssignmentInput{Knowledge: content.VersionRef{ID: "fixture", Version: 1}, TopicIDs: []string{"msc-00a00"}, SourceRefs: []taxonomy.SourceRecordRef{{SourceID: "source", WorkFamilyID: "work", RecordID: "original", Path: "original.json", SHA256: strings.Repeat("b", 64)}}, SourceBatchSHA: strings.Repeat("c", 64)}}
	raw, _ := json.Marshal(in)
	if e := decodeTaxonomyJSON(strings.NewReader(string(raw)), &taxonomy.DraftTopicInput{}); e != nil {
		t.Fatal("valid topic input rejected", e)
	}
	for _, bad := range []string{strings.Replace(string(raw), `"expectedDraftRevision":1`, `"expectedDraftRevision":1,"ExpectedDraftRevision":1`, 1), strings.Replace(string(raw), `"member":`, `"actorId":"forged","member":`, 1), `{"expectedDraftRevision":1}`} {
		if e := decodeTaxonomyJSON(strings.NewReader(bad), &taxonomy.DraftTopicInput{}); e == nil {
			t.Fatal("unsafe input accepted")
		}
	}
}
