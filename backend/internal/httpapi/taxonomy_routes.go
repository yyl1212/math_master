package httpapi

import (
	"context"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type TaxonomyOptions struct {
	Service      *taxonomy.Service
	PublicOrigin string
	Production   bool
}
type taxonomyRoute struct {
	kind, id, method string
	public, list     bool
	action           publication.Action
}

func routeTaxonomy(path, method string) (taxonomyRoute, error) {
	var r taxonomyRoute
	switch {
	case path == "/api/v2/topics/experience-mode":
		r = taxonomyRoute{kind: "readExperience", method: "GET", public: true}
	case path == "/api/v2/topics":
		r = taxonomyRoute{kind: "listTopics", method: "GET", public: true, list: true}
	case strings.HasPrefix(path, "/api/v2/topics/"):
		parts := strings.Split(strings.TrimPrefix(path, "/api/v2/topics/"), "/")
		if len(parts) > 2 || parts[0] == "" {
			return r, auth.ErrNotFound
		}
		r = taxonomyRoute{kind: "readTopic", id: parts[0], method: "GET", public: true}
		if !publication.ValidMathID(r.id) || !strings.HasPrefix(r.id, "msc-") {
			return r, auth.ErrInvalidInput
		}
		if len(parts) == 2 {
			if parts[1] != "knowledge" {
				return r, auth.ErrNotFound
			}
			r.kind = "listKnowledge"
			r.list = true
		}
	case strings.HasPrefix(path, "/api/v2/content/topic-assignments/"):
		p := strings.Split(strings.TrimPrefix(path, "/api/v2/content/topic-assignments/"), "/")
		if len(p) != 2 || p[0] != "drafts" && p[0] != "submissions" {
			return r, auth.ErrNotFound
		}
		r = taxonomyRoute{kind: "readDraft", id: p[1], method: "GET", action: publication.ReadDraftAction}
		if p[0] == "submissions" {
			r.kind = "readSubmission"
			r.action = publication.ReadSubmissionAction
		} else if method == "PUT" {
			r.kind = "saveDraft"
			r.method = "PUT"
			r.action = publication.SaveDraftAction
		}
	case path == "/api/v2/admin/publications":
		r = taxonomyRoute{kind: "listReleases", method: "GET", list: true, action: publication.ListPublicationsAction}
		if method == "POST" {
			r.kind = "prepareRelease"
			r.method = "POST"
			r.list = false
			r.action = publication.PrepareReleaseAction
		}
	case strings.HasPrefix(path, "/api/v2/admin/publications/"):
		p := strings.Split(strings.TrimPrefix(path, "/api/v2/admin/publications/"), "/")
		if len(p) > 2 || p[0] == "" {
			return r, auth.ErrNotFound
		}
		r = taxonomyRoute{kind: "readRelease", id: p[0], method: "GET", action: publication.ReadPublicationAction}
		if len(p) == 2 {
			if p[1] != "activate" {
				return r, auth.ErrNotFound
			}
			r.kind = "activateRelease"
			r.method = "POST"
			r.action = publication.ActivateReleaseAction
		}
	default:
		return r, auth.ErrNotFound
	}
	if !r.public && r.id != "" && !publication.ValidID(r.id) {
		return r, auth.ErrInvalidInput
	}
	if method != r.method {
		return r, errPrivateMethod
	}
	return r, nil
}
func taxonomyQuery(raw string, route taxonomyRoute) (taxonomy.Query, error) {
	var q taxonomy.Query
	values, e := url.ParseQuery(raw)
	if e != nil {
		return q, auth.ErrInvalidInput
	}
	for key, v := range values {
		if len(v) != 1 || v[0] == "" || !taxonomy.ValidText(v[0]) {
			return q, auth.ErrInvalidInput
		}
		switch key {
		case "q":
			if !route.public || len(v[0]) > 512 {
				return q, auth.ErrInvalidInput
			}
			q.Q = v[0]
		case "parentId":
			if route.kind != "listTopics" || !publication.ValidMathID(v[0]) || !strings.HasPrefix(v[0], "msc-") {
				return q, auth.ErrInvalidInput
			}
			q.ParentID = v[0]
		case "kind":
			if route.kind != "listTopics" || v[0] != "primary" && v[0] != "auxiliary" && v[0] != "other" {
				return q, auth.ErrInvalidInput
			}
			q.Kind = v[0]
		case "limit", "offset", "level":
			for _, ch := range v[0] {
				if ch < '0' || ch > '9' {
					return q, auth.ErrInvalidInput
				}
			}
			n, e := strconv.Atoi(v[0])
			if e != nil {
				return q, auth.ErrInvalidInput
			}
			if key == "limit" {
				if n < 1 || n > 100 {
					return q, auth.ErrInvalidInput
				}
				q.Limit = n
			} else if key == "offset" {
				if n > 100000 {
					return q, auth.ErrInvalidInput
				}
				q.Offset = n
			} else {
				if route.kind != "listTopics" || n < 1 || n > 3 {
					return q, auth.ErrInvalidInput
				}
				q.Level = n
			}
		default:
			return q, auth.ErrInvalidInput
		}
	}
	return q, nil
}
func taxonomyErrorResponse(w http.ResponseWriter, r *http.Request, e error) {
	code, message, status := "", "", 0
	switch {
	case errors.Is(e, taxonomy.ErrInvalid):
		code = "TAXONOMY_INVALID"
		message = "Topic classification is invalid."
		status = 400
	case errors.Is(e, taxonomy.ErrNotConfigured):
		code = "TAXONOMY_NOT_CONFIGURED"
		message = "Topic classification is temporarily unavailable."
		status = 503
	case errors.Is(e, taxonomy.ErrConflict):
		e = publication.ErrDraftConflict
	case errors.Is(e, taxonomy.ErrHeadStale):
		e = publication.ErrPublicationStale
	case errors.Is(e, taxonomy.ErrIdempotencyConflict):
		e = publication.ErrIdempotencyConflict
	case errors.Is(e, taxonomy.ErrLimit):
		e = publication.ErrContentLimitExceeded
	case errors.Is(e, store.ErrNotFound):
		e = auth.ErrNotFound
	case errors.Is(e, auth.ErrReauthRequired):
		code = "REAUTH_REQUIRED"
		message = "Verify your password before continuing."
		status = 428
	}
	if code != "" {
		taxonomyResponse(w, r, status, map[string]any{"error": map[string]string{"code": code, "message": message, "requestId": r.Header.Get("X-Request-ID")}})
		return
	}
	contentError(w, r, e)
}
func serveTaxonomy(w http.ResponseWriter, r *http.Request, o TaxonomyOptions) {
	id := requestID()
	privateHeaders(w)
	w.Header().Set("X-Request-ID", id)
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	r = r.Clone(ctx)
	r.Header = r.Header.Clone()
	r.Header.Set("X-Request-ID", id)
	_ = http.NewResponseController(w).SetReadDeadline(time.Now().Add(8 * time.Second))
	fail := func(e error) { taxonomyErrorResponse(w, r, e) }
	if r.URL.RawPath != "" {
		fail(auth.ErrInvalidInput)
		return
	}
	route, e := routeTaxonomy(r.URL.Path, r.Method)
	if e != nil {
		if e == errPrivateMethod {
			w.Header().Set("Allow", route.method)
		}
		fail(e)
		return
	}
	var q taxonomy.Query
	if r.URL.ForceQuery && r.URL.RawQuery == "" {
		fail(auth.ErrInvalidInput)
		return
	}
	if route.list {
		q, e = taxonomyQuery(r.URL.RawQuery, route)
	} else if r.URL.RawQuery != "" || r.URL.ForceQuery {
		e = auth.ErrInvalidInput
	}
	if e != nil {
		fail(e)
		return
	}
	if route.public {
		if r.ContentLength != 0 || len(r.TransferEncoding) > 0 {
			fail(auth.ErrInvalidInput)
			return
		}
		if o.Service == nil {
			fail(taxonomy.ErrNotConfigured)
			return
		}
		v, e := dispatchPublicTaxonomy(ctx, o.Service, route, q)
		if ctx.Err() != nil {
			e = auth.ErrUnavailable
		}
		if e != nil {
			fail(e)
			return
		}
		taxonomyResponse(w, r, 200, v)
		return
	}
	if o.PublicOrigin == "" {
		fail(errAuthNotConfigured)
		return
	}
	write := r.Method != "GET"
	if len(r.Header.Values("Origin")) > 1 || len(r.Header.Values("X-CSRF-Token")) > 1 || r.Header.Get("Sec-Fetch-Site") == "cross-site" || write && r.Header.Get("Origin") != o.PublicOrigin || !write && r.Header.Get("Origin") != "" && r.Header.Get("Origin") != o.PublicOrigin {
		fail(auth.ErrCSRF)
		return
	}
	cookies, _, e := privateCookies(r, o.Production)
	if e != nil {
		fail(e)
		return
	}
	proof, e := auth.DecodeContentProof(cookies, r.Header.Get("X-CSRF-Token"), write)
	if e != nil {
		fail(e)
		return
	}
	if o.Service == nil {
		fail(taxonomy.ErrNotConfigured)
		return
	}
	a := publication.Access{TokenHash: proof.TokenHash, CSRF: proof.CSRF, RequestID: id}
	if write {
		if len(r.Header.Values("Idempotency-Key")) != 1 || !publication.ValidID(r.Header.Get("Idempotency-Key")) {
			fail(auth.ErrInvalidInput)
			return
		}
		a.IdempotencyKey = r.Header.Get("Idempotency-Key")
	}
	if _, e = o.Service.Preflight(ctx, a, route.action); e != nil {
		fail(e)
		return
	}
	if write {
		typ, params, e := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if e != nil || typ != "application/json" || len(r.Header.Values("Content-Type")) != 1 {
			fail(auth.ErrInvalidInput)
			return
		}
		for k, v := range params {
			if k != "charset" || !strings.EqualFold(v, "utf-8") {
				fail(auth.ErrInvalidInput)
				return
			}
		}
		if r.ContentLength > 8192 {
			fail(errContentPayloadTooLarge)
			return
		}
		release, e := o.Service.AcquireValidation(ctx)
		if e != nil {
			fail(&auth.RateLimitError{RetryAfterSeconds: 1})
			return
		}
		defer release()
	} else if r.ContentLength != 0 || len(r.TransferEncoding) > 0 || len(r.Header.Values("Idempotency-Key")) > 1 {
		fail(auth.ErrInvalidInput)
		return
	}
	v, e := dispatchManagedTaxonomy(ctx, o.Service, route, q, a, r.Body)
	if ctx.Err() != nil {
		e = auth.ErrUnavailable
	}
	if e != nil {
		fail(e)
		return
	}
	status := 200
	if route.kind == "prepareRelease" {
		status = 201
	}
	taxonomyResponse(w, r, status, v)
}
func taxonomyResponse(w http.ResponseWriter, r *http.Request, status int, v any) {
	raw, e := jsonMarshalTaxonomy(v)
	if e != nil {
		taxonomyErrorResponse(w, r, e)
		return
	}
	privateHeaders(w)
	w.WriteHeader(status)
	_, _ = w.Write(raw)
}
