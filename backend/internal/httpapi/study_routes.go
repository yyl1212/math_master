package httpapi

import (
	"context"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/study"
	"github.com/yyl1212/math_master/backend/internal/taxonomy"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type StudyOptions struct {
	Service      *study.Service
	PublicOrigin string
	Production   bool
}
type studyRoute struct {
	kind, id, method string
	action           study.Action
	note             bool
	list             bool
}

func routeStudy(path, method string) (studyRoute, error) {
	r := studyRoute{method: "GET", action: study.Read}
	parts := strings.Split(strings.TrimPrefix(path, "/api/v2/study/"), "/")
	if !strings.HasPrefix(path, "/api/v2/study/") {
		return r, auth.ErrNotFound
	}
	if len(parts) == 1 {
		switch parts[0] {
		case "overview", "topics", "knowledge", "history":
			r.kind = parts[0]
			r.list = r.kind != "overview"
		default:
			return r, auth.ErrNotFound
		}
	} else if parts[0] == "knowledge" && len(parts) >= 2 {
		r.id = parts[1]
		if !study.ValidKnowledgeID(r.id) {
			return r, auth.ErrInvalidInput
		}
		r.kind = "readKnowledge"
		if len(parts) == 3 {
			switch parts[2] {
			case "note":
				r.note = true
				r.kind = "readNote"
				if method == "PUT" {
					r.kind = "saveNote"
					r.method = "PUT"
					r.action = study.SaveNote
				} else if method == "DELETE" {
					r.kind = "deleteNote"
					r.method = "DELETE"
					r.action = study.DeleteNote
				}
			case "begin":
				r.kind = "begin"
				r.method = "POST"
				r.action = study.Begin
			case "complete":
				r.kind = "complete"
				r.method = "POST"
				r.action = study.Complete
			default:
				return r, auth.ErrNotFound
			}
		} else if len(parts) == 4 && parts[2] == "review" {
			r.method = "POST"
			if parts[3] == "start" {
				r.kind = "startReview"
				r.action = study.StartReview
			} else if parts[3] == "finish" {
				r.kind = "finishReview"
				r.action = study.FinishReview
			} else {
				return r, auth.ErrNotFound
			}
		} else if len(parts) != 2 {
			return r, auth.ErrNotFound
		}
	} else {
		return r, auth.ErrNotFound
	}
	if method != r.method {
		return r, errPrivateMethod
	}
	return r, nil
}
func studyQueries(raw string, route studyRoute) (study.ListQuery, study.HistoryQuery, error) {
	var q study.ListQuery
	var h study.HistoryQuery
	v, e := url.ParseQuery(raw)
	if e != nil {
		return q, h, auth.ErrInvalidInput
	}
	for key, values := range v {
		if len(values) != 1 || values[0] == "" || !taxonomy.ValidText(values[0]) {
			return q, h, auth.ErrInvalidInput
		}
		value := values[0]
		switch key {
		case "limit", "offset":
			if key == "offset" && route.kind == "history" {
				return q, h, auth.ErrInvalidInput
			}
			for _, r := range value {
				if r < '0' || r > '9' {
					return q, h, auth.ErrInvalidInput
				}
			}
			n, e := strconv.Atoi(value)
			if e != nil || key == "limit" && (n < 1 || n > 100 || route.kind == "history" && n > 50) || key == "offset" && n > 100000 {
				return q, h, auth.ErrInvalidInput
			}
			if key == "limit" {
				q.Limit = n
				h.Limit = n
			} else {
				q.Offset = n
			}
		case "topicId":
			if !study.ValidKnowledgeID(value) || !strings.HasPrefix(value, "msc-") {
				return q, h, auth.ErrInvalidInput
			}
			q.TopicID = value
			h.TopicID = value
		case "state":
			if route.kind != "knowledge" || !study.ValidState(study.State(value)) {
				return q, h, auth.ErrInvalidInput
			}
			q.State = value
		case "q":
			if route.kind != "knowledge" || len(value) > 512 {
				return q, h, auth.ErrInvalidInput
			}
			q.Q = value
		case "reviewOnly":
			if route.kind != "knowledge" || value != "true" && value != "false" {
				return q, h, auth.ErrInvalidInput
			}
			q.ReviewOnly = value == "true"
		case "knowledgeId", "kind", "cursor", "from", "to":
			if route.kind != "history" {
				return q, h, auth.ErrInvalidInput
			}
			switch key {
			case "knowledgeId":
				if !study.ValidKnowledgeID(value) {
					return q, h, auth.ErrInvalidInput
				}
				h.KnowledgeID = value
			case "kind":
				switch value {
				case "started", "completed", "review-started", "review-finished", "note-saved", "note-deleted":
					h.Kind = value
				default:
					return q, h, auth.ErrInvalidInput
				}
			case "cursor":
				if len(value) > 512 {
					return q, h, auth.ErrInvalidInput
				}
				for _, c := range value {
					if c < 32 || c > 126 {
						return q, h, auth.ErrInvalidInput
					}
				}
				h.Cursor = value
			case "from", "to":
				at, e := time.Parse(time.RFC3339Nano, value)
				if e != nil {
					return q, h, auth.ErrInvalidInput
				}
				if key == "from" {
					h.From = &at
				} else {
					h.To = &at
				}
			}
		default:
			return q, h, auth.ErrInvalidInput
		}
	}
	if h.From != nil && h.To != nil && !h.From.Before(*h.To) {
		return q, h, auth.ErrInvalidInput
	}
	return q, h, nil
}
func studyErrorResponse(w http.ResponseWriter, r *http.Request, e error) {
	code, message, status := "", "", 0
	switch {
	case errors.Is(e, study.ErrInvalid):
		code, message, status = "STUDY_INVALID", "Study input is invalid.", 400
	case errors.Is(e, study.ErrNotConfigured):
		code, message, status = "STUDY_NOT_CONFIGURED", "Study records are temporarily unavailable.", 503
	case errors.Is(e, study.ErrStateConflict):
		code, message, status = "STUDY_STATE_CONFLICT", "Study state has changed. Refresh before continuing.", 409
	case errors.Is(e, study.ErrVersionStale):
		code, message, status = "STUDY_VERSION_STALE", "Study material has changed. Refresh before continuing.", 409
	case errors.Is(e, study.ErrIdempotencyConflict):
		code, message, status = "STUDY_IDEMPOTENCY_CONFLICT", "This request key was already used with different input.", 409
	case errors.Is(e, study.ErrNoteConflict):
		code, message, status = "STUDY_NOTE_CONFLICT", "This note has changed. Read the latest revision before saving.", 409
	case errors.Is(e, auth.ErrReauthRequired):
		code, message, status = "REAUTH_REQUIRED", "Verify your password before continuing.", 428
	}
	if code == "" {
		contentError(w, r, e)
		return
	}
	taxonomyResponse(w, r, status, map[string]any{"error": map[string]string{"code": code, "message": message, "requestId": r.Header.Get("X-Request-ID")}})
}
func serveStudy(w http.ResponseWriter, r *http.Request, o StudyOptions) {
	id := requestID()
	privateHeaders(w)
	w.Header().Set("X-Request-ID", id)
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	r = r.Clone(ctx)
	r.Header = r.Header.Clone()
	r.Header.Set("X-Request-ID", id)
	_ = http.NewResponseController(w).SetReadDeadline(time.Now().Add(8 * time.Second))
	fail := func(e error) { studyErrorResponse(w, r, e) }
	if r.URL.RawPath != "" || r.URL.ForceQuery {
		fail(auth.ErrInvalidInput)
		return
	}
	route, e := routeStudy(r.URL.Path, r.Method)
	if e != nil {
		if e == errPrivateMethod {
			w.Header().Set("Allow", route.method)
		}
		fail(e)
		return
	}
	var q study.ListQuery
	var h study.HistoryQuery
	if route.list {
		q, h, e = studyQueries(r.URL.RawQuery, route)
	} else if r.URL.RawQuery != "" {
		e = auth.ErrInvalidInput
	}
	if e != nil {
		fail(e)
		return
	}
	if o.PublicOrigin == "" {
		fail(errAuthNotConfigured)
		return
	}
	write := route.method != "GET"
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
		fail(study.ErrNotConfigured)
		return
	}
	a := study.Access{TokenHash: proof.TokenHash, CSRF: proof.CSRF, RequestID: id}
	if write {
		if len(r.Header.Values("Idempotency-Key")) != 1 || !study.ValidID(r.Header.Get("Idempotency-Key")) {
			fail(auth.ErrInvalidInput)
			return
		}
		a.IdempotencyKey = r.Header.Get("Idempotency-Key")
	}
	if _, e = o.Service.Preflight(ctx, a, route.action); e != nil {
		fail(e)
		return
	}
	limit := int64(8192)
	if route.kind == "saveNote" {
		limit = 131072
	}
	if write {
		typ, params, e := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if e != nil || typ != "application/json" || len(r.Header.Values("Content-Type")) != 1 {
			fail(auth.ErrInvalidInput)
			return
		}
		for key, value := range params {
			if key != "charset" || !strings.EqualFold(value, "utf-8") {
				fail(auth.ErrInvalidInput)
				return
			}
		}
		if r.ContentLength > limit {
			fail(errContentPayloadTooLarge)
			return
		}
	} else if r.ContentLength != 0 || len(r.TransferEncoding) > 0 || len(r.Header.Values("Idempotency-Key")) > 1 {
		fail(auth.ErrInvalidInput)
		return
	}
	value, e := dispatchStudy(ctx, o.Service, route, q, h, a, r.Body, limit)
	if ctx.Err() != nil {
		e = auth.ErrUnavailable
	}
	if e != nil {
		fail(e)
		return
	}
	raw, e := jsonMarshalTaxonomy(value)
	if e != nil {
		fail(auth.ErrUnavailable)
		return
	}
	privateHeaders(w)
	w.WriteHeader(200)
	_, _ = w.Write(raw)
}
