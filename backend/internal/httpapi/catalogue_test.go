package httpapi

import (
	"context"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/catalogue"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/store"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeReader struct {
	err           error
	q             string
	limit, offset int
}

func (f *fakeReader) ListDomains(ctx context.Context, q string, l, o int) ([]catalogue.DomainSummary, int, error) {
	f.q = q
	f.limit = l
	f.offset = o
	return []catalogue.DomainSummary{}, 0, f.err
}
func (f *fakeReader) GetDomain(context.Context, string) (catalogue.DomainDetail, error) {
	return catalogue.DomainDetail{}, f.err
}
func (f *fakeReader) GetPublishedPath(context.Context, string) (content.PathView, error) {
	return content.PathView{}, f.err
}
func (f *fakeReader) GetPublishedKnowledge(context.Context, string) (content.KnowledgeView, error) {
	return content.KnowledgeView{}, f.err
}
func request(h http.Handler, path string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
	return w
}
func TestCatalogueSearchAndPagination(t *testing.T) {
	f := &fakeReader{}
	h := NewHandler(f, nil)
	for _, path := range []string{"/api/v1/domains?limit=0", "/api/v1/domains?limit=101", "/api/v1/domains?offset=-1", "/api/v1/domains?limit=x", "/api/v1/domains?limit=1&limit=2"} {
		w := request(h, path)
		if w.Code != 400 || !strings.Contains(w.Body.String(), "INVALID_QUERY") {
			t.Fatal(path, w.Code, w.Body.String())
		}
	}
	w := request(h, "/api/v1/domains?q=Markov&limit=5&offset=3")
	if w.Code != 200 || f.q != "Markov" || f.limit != 5 || f.offset != 3 || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal(w.Code, w.Body.String())
	}
}
func TestErrorsAreEnglishAndRedacted(t *testing.T) {
	for _, c := range []struct {
		err       error
		status    int
		code, msg string
	}{{store.ErrNotFound, 404, "NOT_FOUND", "Resource not found."}, {store.ErrUnavailable, 503, "SERVICE_UNAVAILABLE", "Service temporarily unavailable."}, {errors.New("private-secret connection detail"), 500, "INTERNAL_ERROR", "Internal server error."}} {
		f := &fakeReader{err: c.err}
		w := request(NewHandler(f, nil), "/api/v1/knowledge/fractions")
		if w.Code != c.status || !strings.Contains(w.Body.String(), c.code) || !strings.Contains(w.Body.String(), c.msg) || !strings.Contains(w.Body.String(), "requestId") || strings.Contains(w.Body.String(), "private-secret") {
			t.Fatal(w.Code, w.Body.String())
		}
	}
}
