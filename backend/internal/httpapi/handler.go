package httpapi

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/catalogue"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/store"
	"net/http"
)

type Reader interface {
	ListDomains(context.Context, string, int, int) ([]catalogue.DomainSummary, int, error)
	GetDomain(context.Context, string) (catalogue.DomainDetail, error)
	GetPublishedPath(context.Context, string) (content.PathView, error)
	GetPublishedKnowledge(context.Context, string) (content.KnowledgeView, error)
}

func NewHandler(reader Reader, pinger Pinger) http.Handler {
	mux := http.NewServeMux()
	health := NewHealthHandler(pinger)
	mux.Handle("GET /healthz", health)
	mux.Handle("GET /readyz", health)
	mux.HandleFunc("GET /api/v1/domains", func(w http.ResponseWriter, r *http.Request) { listDomains(w, r, reader) })
	mux.HandleFunc("GET /api/v1/domains/{id}", func(w http.ResponseWriter, r *http.Request) {
		v, e := reader.GetDomain(r.Context(), r.PathValue("id"))
		detail(w, r, v, e)
	})
	mux.HandleFunc("GET /api/v1/knowledge/{id}", func(w http.ResponseWriter, r *http.Request) {
		v, e := reader.GetPublishedKnowledge(r.Context(), r.PathValue("id"))
		detail(w, r, v, e)
	})
	mux.HandleFunc("GET /api/v1/paths/{id}", func(w http.ResponseWriter, r *http.Request) {
		v, e := reader.GetPublishedPath(r.Context(), r.PathValue("id"))
		detail(w, r, v, e)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		apiError(w, r.Header.Get("X-Request-ID"), store.ErrNotFound)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := requestID()
		w.Header().Set("X-Request-ID", id)
		r = r.Clone(r.Context())
		r.Header = r.Header.Clone()
		r.Header.Set("X-Request-ID", id)
		mux.ServeHTTP(w, r)
	})
}
func detail(w http.ResponseWriter, r *http.Request, data any, e error) {
	if e != nil {
		apiError(w, r.Header.Get("X-Request-ID"), e)
		return
	}
	response(w, 200, map[string]any{"data": data})
}
