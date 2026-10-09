package httpapi

import (
	"testing"
)

func TestManagedReadNoStaleDisclosure(t *testing.T) {
	h, _ := knowledgeHTTPFixture()
	for _, p := range []string{"/api/v1/knowledge/fractions", "/api/v1/domains/elementary-math", "/api/v1/assets/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "/api/v2/topics/msc-97f40/knowledge"} {
		if w := privateRequest(h, "GET", p, "", nil); w.Code != 410 {
			t.Fatal("legacy content accessible", p, w.Code)
		}
	}
	if w := privateRequest(h, "GET", "/api/v3/knowledge", "", nil); w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("public read", w.Code, w.Header())
	}
	if w := privateRequest(h, "GET", "/api/v3/knowledge/k-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "", nil); w.Code != 404 {
		t.Fatal("unavailable current body", w.Code)
	}
	if w := privateRequest(h, "GET", "/api/v3/content-mode", "", nil); w.Code != 200 {
		t.Fatal("mode", w.Code)
	}
}
func TestManagedPublicStrictQueriesAndBodies(t *testing.T) {
	h, _ := knowledgeHTTPFixture()
	for _, p := range []string{"/api/v3/knowledge?limit=0", "/api/v3/knowledge?limit=101", "/api/v3/knowledge?status=deleted", "/api/v3/knowledge?limit=1&limit=2", "/api/v3/content-mode?actorId=forged"} {
		if w := privateRequest(h, "GET", p, "", nil); w.Code != 422 {
			t.Fatal("invalid query accepted", p, w.Code)
		}
	}
	if w := privateRequest(h, "GET", "/api/v3/knowledge", "{}", nil); w.Code != 422 {
		t.Fatal("GET request body accepted", w.Code)
	}
}
