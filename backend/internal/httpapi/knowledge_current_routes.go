package httpapi

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"net/http"
	"strings"
	"time"
)

func serveCurrentKnowledge(w http.ResponseWriter, r *http.Request, o KnowledgeOptions, route managedRoute, q knowledgeadmin.Query) {
	if o.Current == nil {
		knowledgeHTTPError(w, r, knowledgeadmin.ErrNotConfigured)
		return
	}
	var value any
	var e error
	switch route.kind {
	case "mode":
		value, e = o.Current.ReadContentMode(r.Context())
	case "public-list":
		value, e = o.Current.ListCurrentKnowledge(r.Context(), q)
	case "public-read":
		value, e = o.Current.ReadCurrentKnowledge(r.Context(), route.id)
	case "topics":
		value, e = o.Current.ListCurrentTopics(r.Context(), q)
	case "topic":
		value, e = o.Current.ReadCurrentTopic(r.Context(), route.id, q)
	default:
		e = knowledgeadmin.ErrNotFound
	}
	if e != nil {
		knowledgeHTTPError(w, r, e)
		return
	}
	knowledgeHTTPResponse(w, r, value, false)
}
func oldKnowledgeRoute(path, method string) bool {
	for _, prefix := range []string{"/api/v1/content", "/api/v1/knowledge", "/api/v1/domains", "/api/v1/paths", "/api/v1/assets", "/api/v2/content/topic-assignments", "/api/v2/admin/publications"} {
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return true
		}
	}
	if (path == "/api/v2/topics" || strings.HasPrefix(path, "/api/v2/topics/")) && path != "/api/v2/topics/experience-mode" {
		return true
	}
	if (path == "/api/v2/study" || strings.HasPrefix(path, "/api/v2/study/")) && !(method == "GET" && (path == "/api/v2/study/history" || strings.HasPrefix(path, "/api/v2/study/knowledge/") && strings.HasSuffix(path, "/note"))) {
		return true
	}
	if method != "GET" && strings.HasPrefix(path, "/api/v1/corrections/") && strings.Contains(path, "/plans") {
		return true
	}
	return false
}
func serveManagedLegacyRetirement(w http.ResponseWriter, r *http.Request, o *KnowledgeOptions) bool {
	if o == nil || o.Current == nil || !oldKnowledgeRoute(r.URL.Path, r.Method) {
		return false
	}
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	mode, e := o.Current.ReadContentMode(ctx)
	if e == nil && mode.Mode != "managed" {
		return false
	}
	id := requestID()
	privateHeaders(w)
	w.Header().Set("X-Request-ID", id)
	r = r.Clone(r.Context())
	r.Header = r.Header.Clone()
	r.Header.Set("X-Request-ID", id)
	if e != nil {
		knowledgeHTTPError(w, r, knowledgeadmin.ErrNotConfigured)
		return true
	}
	knowledgeHTTPError(w, r, knowledgeadmin.ErrRetired)
	return true
}
