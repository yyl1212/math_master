package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type KnowledgeOptions struct {
	Service      *knowledgeadmin.Service
	Current      knowledgeadmin.CurrentRepository
	PublicOrigin string
	Production   bool
}

var managedKnowledgeID = regexp.MustCompile(`^k-[0-9a-f]{56}$`)
var managedOperationID = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

type managedRoute struct {
	kind, id, method    string
	admin, list, upload bool
}

func routeManaged(path, method string) (out managedRoute, e error) {
	out.method = "GET"
	parts := strings.Split(strings.TrimPrefix(path, "/api/v3/"), "/")
	if !strings.HasPrefix(path, "/api/v3/") {
		return out, auth.ErrNotFound
	}
	for _, p := range parts {
		if p == "" {
			return out, auth.ErrNotFound
		}
	}
	switch {
	case len(parts) == 1 && parts[0] == "content-mode":
		out.kind = "mode"
	case len(parts) >= 2 && parts[0] == "admin" && parts[1] == "knowledge":
		out.admin = true
		switch {
		case len(parts) == 2:
			out.kind = "list"
			out.list = true
			if method == "POST" {
				out.kind = "create"
				out.method = "POST"
				out.list = false
			}
		case len(parts) == 3 && parts[2] == "imports":
			out.kind = "preview"
			out.method = "POST"
			out.upload = true
		case len(parts) == 4 && parts[2] == "imports":
			out.kind = "import-status"
			out.id = parts[3]
			if !managedOperationID.MatchString(out.id) {
				return out, knowledgeadmin.ErrInvalid
			}
		case len(parts) == 5 && parts[2] == "imports" && parts[4] == "apply":
			out.kind = "apply"
			out.id = parts[3]
			out.method = "POST"
			if !managedOperationID.MatchString(out.id) {
				return out, knowledgeadmin.ErrInvalid
			}
		case len(parts) == 3:
			out.kind = "read"
			out.id = parts[2]
			if method == "PUT" {
				out.kind = "update"
				out.method = "PUT"
			} else if method == "DELETE" {
				out.kind = "delete"
				out.method = "DELETE"
			}
		case len(parts) == 4 && (parts[3] == "publish" || parts[3] == "unpublish" || parts[3] == "restore"):
			out.kind = parts[3]
			out.method = "POST"
			out.id = parts[2]
		default:
			return out, auth.ErrNotFound
		}
		if out.id != "" && out.kind != "import-status" && out.kind != "apply" && !managedKnowledgeID.MatchString(out.id) {
			return out, knowledgeadmin.ErrInvalid
		}
	case len(parts) == 1 && parts[0] == "knowledge":
		out.kind = "public-list"
		out.list = true
	case len(parts) == 2 && parts[0] == "knowledge":
		out.kind = "public-read"
		out.id = parts[1]
		if !managedKnowledgeID.MatchString(out.id) {
			return out, knowledgeadmin.ErrInvalid
		}
	case len(parts) == 1 && parts[0] == "topics":
		out.kind = "topics"
		out.list = true
	case len(parts) == 2 && parts[0] == "topics":
		out.kind = "topic"
		out.id = parts[1]
		out.list = true
	default:
		return out, auth.ErrNotFound
	}
	if method != out.method {
		return out, errPrivateMethod
	}
	return out, nil
}
func managedQuery(raw string, route managedRoute) (out knowledgeadmin.Query, e error) {
	v, e := url.ParseQuery(raw)
	if e != nil {
		return out, knowledgeadmin.ErrInvalid
	}
	for key, values := range v {
		if len(values) != 1 || values[0] == "" {
			return out, knowledgeadmin.ErrInvalid
		}
		s := values[0]
		switch key {
		case "topicKey":
			out.TopicKey = s
		case "q":
			out.Q = s
		case "type":
			out.Type = s
		case "status":
			if !route.admin {
				return out, knowledgeadmin.ErrInvalid
			}
			out.Status = s
		case "difficulty", "limit", "offset":
			if !regexp.MustCompile(`^[0-9]+$`).MatchString(s) {
				return out, knowledgeadmin.ErrInvalid
			}
			n, e := strconv.Atoi(s)
			if e != nil {
				return out, knowledgeadmin.ErrInvalid
			}
			switch key {
			case "difficulty":
				out.Difficulty = n
			case "limit":
				if n < 1 || n > 100 {
					return out, knowledgeadmin.ErrInvalid
				}
				out.Limit = n
			case "offset":
				out.Offset = n
			}
		default:
			return out, knowledgeadmin.ErrInvalid
		}
	}
	if out.Limit == 0 {
		out.Limit = 20
	}
	if out.Limit < 1 || out.Limit > 100 || out.Offset < 0 || out.Offset > 1000000 || out.Difficulty < 0 || out.Difficulty > 5 || len(out.Q) > 256 || len(out.TopicKey) > 64 {
		return out, knowledgeadmin.ErrInvalid
	}
	return out, nil
}
func knowledgeHTTPError(w http.ResponseWriter, r *http.Request, e error) {
	status, code, message := 503, "CONTENT_NOT_READY", "Knowledge is temporarily unavailable."
	var de *knowledgeadmin.DecodeError
	switch {
	case errors.Is(e, errPrivateMethod):
		status, code, message = 405, "METHOD_NOT_ALLOWED", "Method not allowed."
	case errors.Is(e, auth.ErrNotFound) || errors.Is(e, knowledgeadmin.ErrNotFound):
		status, code, message = 404, "NOT_FOUND", "Knowledge was not found."
	case errors.Is(e, auth.ErrAuthenticationRequired) || errors.Is(e, auth.ErrInvalidCookie):
		status, code, message = 401, "AUTHENTICATION_REQUIRED", "Sign in to continue."
	case errors.Is(e, auth.ErrForbidden):
		status, code, message = 403, "FORBIDDEN", "Administrator access is required."
	case errors.Is(e, auth.ErrCSRF):
		status, code, message = 403, "CSRF_FAILED", "Request verification failed."
	case errors.Is(e, auth.ErrPasswordChangeRequired):
		status, code, message = 403, "PASSWORD_CHANGE_REQUIRED", "Change your password before continuing."
	case errors.Is(e, knowledgeadmin.ErrStale):
		status, code, message = 409, "CONTENT_STALE", "Knowledge changed. Reload before saving."
	case errors.Is(e, knowledgeadmin.ErrConflict):
		status, code, message = 409, "ID_CONTENT_CONFLICT", "This data ID has conflicting content. Edit the existing knowledge."
	case errors.Is(e, knowledgeadmin.ErrIdempotency):
		status, code, message = 409, "IDEMPOTENCY_CONFLICT", "The request key belongs to different input."
	case errors.Is(e, knowledgeadmin.ErrRetired):
		status, code, message = 410, "KNOWLEDGE_WORKFLOW_RETIRED", "Use current knowledge management."
	case errors.Is(e, knowledgeadmin.ErrBusy):
		status, code, message = 429, "UPLOAD_BUSY", "Another upload is being validated. Retry shortly."
		w.Header().Set("Retry-After", "1")
	case errors.Is(e, knowledgeadmin.ErrInvalid) || errors.Is(e, auth.ErrInvalidInput):
		status, code, message = 422, "INVALID_FIELD", "Check the input fields."
	case errors.As(e, &de):
		status, code, message = 422, de.Code, "Source validation failed."
		if de.Code == "INPUT_TOO_LARGE" {
			status = 413
		}
	case errors.Is(e, context.DeadlineExceeded) || errors.Is(e, context.Canceled):
		status, code, message = 503, "REQUEST_INTERRUPTED", "The request was interrupted. Check the operation receipt before retrying."
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	problem := map[string]string{"code": code, "message": message, "requestId": r.Header.Get("X-Request-ID")}
	if de != nil {
		problem["path"] = de.Path
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"error": problem})
}
func knowledgeHTTPResponse(w http.ResponseWriter, r *http.Request, value any, private bool) {
	b, e := json.Marshal(value)
	if e != nil {
		knowledgeHTTPError(w, r, auth.ErrUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-store")
	if private {
		w.Header().Set("Cache-Control", "private, no-store")
	}
	w.WriteHeader(200)
	_, _ = w.Write(b)
}

// Options carry the established origin and cookie policy; omitted options fail closed for management.
func KnowledgeHandler(service *knowledgeadmin.Service, options ...KnowledgeOptions) http.Handler {
	o := KnowledgeOptions{Service: service}
	if len(options) > 0 {
		o = options[0]
		o.Service = service
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { serveKnowledge(w, r, o) })
}
func serveKnowledge(w http.ResponseWriter, r *http.Request, o KnowledgeOptions) {
	id := requestID()
	w.Header().Set("X-Request-ID", id)
	r = r.Clone(r.Context())
	r.Header = r.Header.Clone()
	r.Header.Set("X-Request-ID", id)
	route, e := routeManaged(r.URL.Path, r.Method)
	if e != nil {
		if errors.Is(e, errPrivateMethod) {
			w.Header().Set("Allow", route.method)
		}
		knowledgeHTTPError(w, r, e)
		return
	}
	if r.URL.RawPath != "" || r.URL.ForceQuery {
		knowledgeHTTPError(w, r, knowledgeadmin.ErrInvalid)
		return
	}
	var query knowledgeadmin.Query
	if route.list {
		query, e = managedQuery(r.URL.RawQuery, route)
	} else if r.URL.RawQuery != "" {
		e = knowledgeadmin.ErrInvalid
	}
	if e != nil {
		knowledgeHTTPError(w, r, e)
		return
	}
	timeout := 8 * time.Second
	if route.upload {
		timeout = 60 * time.Second
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()
	r = r.WithContext(ctx)
	controller := http.NewResponseController(w)
	_ = controller.SetReadDeadline(time.Now().Add(timeout))
	if route.upload {
		_ = controller.SetWriteDeadline(time.Now().Add(timeout))
	}
	if !route.admin {
		if r.ContentLength != 0 || len(r.TransferEncoding) > 0 {
			knowledgeHTTPError(w, r, knowledgeadmin.ErrInvalid)
			return
		}
		serveCurrentKnowledge(w, r, o, route, query)
		return
	}
	if o.Service == nil || o.PublicOrigin == "" {
		knowledgeHTTPError(w, r, knowledgeadmin.ErrNotConfigured)
		return
	}
	write := r.Method != "GET"
	cookies, _, e := privateCookies(r, o.Production)
	if e != nil {
		knowledgeHTTPError(w, r, e)
		return
	}
	proof, e := auth.DecodeContentProof(cookies, r.Header.Get("X-CSRF-Token"), write)
	if e != nil {
		knowledgeHTTPError(w, r, e)
		return
	}
	if len(r.Header.Values("Origin")) > 1 || len(r.Header.Values("X-CSRF-Token")) > 1 || r.Header.Get("Sec-Fetch-Site") == "cross-site" || write && r.Header.Get("Origin") != o.PublicOrigin || !write && r.Header.Get("Origin") != "" && r.Header.Get("Origin") != o.PublicOrigin {
		knowledgeHTTPError(w, r, auth.ErrCSRF)
		return
	}
	if write {
		typ, params, e := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if e != nil || typ != "application/json" || len(r.Header.Values("Content-Type")) != 1 {
			knowledgeHTTPError(w, r, knowledgeadmin.ErrInvalid)
			return
		}
		for k, v := range params {
			if k != "charset" || !strings.EqualFold(v, "utf-8") {
				knowledgeHTTPError(w, r, knowledgeadmin.ErrInvalid)
				return
			}
		}
	} else if r.ContentLength != 0 || len(r.TransferEncoding) > 0 {
		knowledgeHTTPError(w, r, knowledgeadmin.ErrInvalid)
		return
	}
	a := knowledgeadmin.Access{TokenHash: proof.TokenHash, CSRF: proof.CSRF, RequestID: id}
	if write {
		if len(r.Header.Values("Idempotency-Key")) != 1 || strings.TrimSpace(r.Header.Get("Idempotency-Key")) == "" || len(r.Header.Get("Idempotency-Key")) > 128 {
			knowledgeHTTPError(w, r, knowledgeadmin.ErrInvalid)
			return
		}
		a.IdempotencyKey = r.Header.Get("Idempotency-Key")
		if _, e = o.Service.Repository.KnowledgePreflight(ctx, a); e != nil {
			knowledgeHTTPError(w, r, e)
			return
		}
	}
	var value any
	token := r.Header.Get("If-Match")
	if route.kind == "update" || route.kind == "publish" || route.kind == "unpublish" || route.kind == "restore" || route.kind == "delete" {
		if len(r.Header.Values("If-Match")) != 1 || !managedOperationID.MatchString(token) {
			knowledgeHTTPError(w, r, knowledgeadmin.ErrInvalid)
			return
		}
	}
	switch route.kind {
	case "preview":
		value, e = o.Service.Preview(ctx, a, r.Body)
	case "list":
		var list knowledgeadmin.Page[knowledgeadmin.Knowledge]
		list, e = o.Service.Repository.ListManagedKnowledge(ctx, a, query)
		summary := knowledgeadmin.Page[knowledgeadmin.KnowledgeSummary]{Items: []knowledgeadmin.KnowledgeSummary{}, Total: list.Total, Limit: list.Limit, Offset: list.Offset}
		for _, k := range list.Items {
			summary.Items = append(summary.Items, knowledgeadmin.KnowledgeSummary{ID: k.ID, ExternalID: k.ExternalID, TopicKeys: k.TopicKeys, Title: k.Point.Title, TitleZH: k.Point.TitleZH, Type: k.Point.Type, Difficulty: k.Point.LearningDifficulty.Level, Ref: k.Ref, Published: k.Published, Deleted: k.Deleted, EditToken: k.EditToken, UpdatedAt: k.UpdatedAt})
		}
		value = summary
	case "read":
		value, e = o.Service.Repository.ReadManagedKnowledge(ctx, a, route.id)
	case "import-status":
		var p knowledgeadmin.Preview
		var receipt *knowledgeadmin.Receipt
		p, receipt, e = o.Service.Repository.ReadManagedImport(ctx, a, route.id)
		value = knowledgeadmin.ImportStatus{Preview: p, Receipt: receipt}
	case "apply":
		var in knowledgeadmin.ApplyInput
		if e = decodeKnowledgeJSON(r.Body, 8192, &in); e == nil {
			value, e = o.Service.Apply(ctx, a, route.id, in)
		}
	case "create", "update":
		var in knowledgeadmin.CurrentInput
		if e = decodeKnowledgeJSON(r.Body, knowledgeadmin.MaxSourceBytes, &in); e == nil {
			if route.kind == "create" {
				value, e = o.Service.Repository.CreateManagedKnowledge(ctx, a, in)
			} else {
				value, e = o.Service.Repository.UpdateManagedKnowledge(ctx, a, route.id, token, in)
			}
		}
	case "publish", "unpublish", "delete", "restore":
		var empty struct{}
		if e = decodeKnowledgeJSON(r.Body, 8192, &empty); e == nil {
			value, e = o.Service.Repository.SetManagedKnowledgeState(ctx, a, route.id, token, route.kind)
		}
	default:
		e = auth.ErrNotFound
	}
	if e != nil {
		knowledgeHTTPError(w, r, e)
		return
	}
	knowledgeHTTPResponse(w, r, value, true)
}
