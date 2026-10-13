package httpapi

import (
	"context"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/notesharing"
	"github.com/yyl1212/math_master/backend/internal/study"
	"mime"
	"net/http"
	"strings"
	"time"
)

type noteShareRoute struct {
	kind, id, method, action string
	private, list            bool
}

func isNoteSharingPath(path string) bool {
	return path == "/api/v3/study/notes" || strings.HasPrefix(path, "/api/v3/shared-notes") || strings.HasPrefix(path, "/api/v3/admin/note-shares") || strings.HasPrefix(path, "/api/v3/note-sharing") || strings.HasPrefix(path, "/api/v3/study/knowledge/") && strings.Contains(path, "/note-sharing")
}
func routeNoteSharing(path string) (out noteShareRoute, e error) {
	out.method = "GET"
	parts := strings.Split(strings.TrimPrefix(path, "/api/v3/"), "/")
	switch {
	case path == "/api/v3/note-sharing/health":
		out.kind = "health"
	case path == "/api/v3/study/notes":
		out.kind = "own-list"
		out.private = true
		out.list = true
	case len(parts) >= 4 && len(parts) <= 5 && parts[0] == "study" && parts[1] == "knowledge" && parts[3] == "note-sharing":
		out.private = true
		out.id = parts[2]
		if !managedKnowledgeID.MatchString(out.id) {
			return out, errNoteRequest
		}
		out.kind = "head"
		if len(parts) == 5 {
			out.kind = parts[4]
			if out.kind != "submit" && out.kind != "withdraw" {
				return out, notesharing.ErrNotFound
			}
			out.method = "POST"
		}
	case path == "/api/v3/shared-notes/latest":
		out.kind = "latest"
	case path == "/api/v3/shared-notes":
		out.kind = "shared-list"
		out.private = true
		out.list = true
	case len(parts) == 3 && parts[0] == "shared-notes" && parts[1] == "public":
		out.kind = "public-detail"
		out.id = parts[2]
	case len(parts) == 2 && parts[0] == "shared-notes":
		out.kind = "shared-detail"
		out.private = true
		out.id = parts[1]
	case path == "/api/v3/admin/note-shares":
		out.kind = "admin-list"
		out.private = true
		out.list = true
	case len(parts) >= 3 && len(parts) <= 4 && parts[0] == "admin" && parts[1] == "note-shares":
		out.private = true
		out.kind = "admin-detail"
		out.id = parts[2]
		if len(parts) == 4 {
			out.action = parts[3]
			if out.action != "approve" && out.action != "return" && out.action != "hide" {
				return out, notesharing.ErrNotFound
			}
			out.kind = "decision"
			out.method = "POST"
		}
	default:
		return out, notesharing.ErrNotFound
	}
	if out.id != "" && out.kind != "head" && out.kind != "submit" && out.kind != "withdraw" && !study.ValidID(out.id) {
		return out, errNoteRequest
	}
	return
}
func serveNoteSharing(w http.ResponseWriter, r *http.Request, o KnowledgeOptions) {
	id := requestID()
	w.Header().Set("X-Request-ID", id)
	r = r.Clone(r.Context())
	r.Header = r.Header.Clone()
	r.Header.Set("X-Request-ID", id)
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	r = r.WithContext(ctx)
	_ = http.NewResponseController(w).SetReadDeadline(time.Now().Add(8 * time.Second))
	if r.URL.RawPath != "" || r.URL.ForceQuery || r.URL.Fragment != "" {
		noteShareHTTPError(w, r, errNoteRequest)
		return
	}
	route, e := routeNoteSharing(r.URL.Path)
	if e != nil {
		noteShareHTTPError(w, r, e)
		return
	}
	if r.Method != route.method {
		w.Header().Set("Allow", route.method)
		noteShareHTTPError(w, r, errPrivateMethod)
		return
	}
	if r.Method == "GET" && (r.ContentLength != 0 || len(r.TransferEncoding) > 0) {
		noteShareHTTPError(w, r, errNoteRequest)
		return
	}
	repo := o.Notes
	if repo == nil && o.Service != nil {
		repo, _ = o.Service.Repository.(notesharing.Repository)
	}
	if repo == nil {
		if route.kind == "health" {
			noteShareResponse(w, r, notesharing.NoteSharingHealth{Capability: true}, 503, false, notesharing.MaxDetailBytes)
		} else {
			noteShareHTTPError(w, r, notesharing.ErrNotReady)
		}
		return
	}
	var a knowledgeadmin.Access
	write := r.Method != "GET"
	if route.private {
		if o.PublicOrigin == "" {
			noteShareHTTPError(w, r, notesharing.ErrNotReady)
			return
		}
		cookies, _, e := privateCookies(r, o.Production)
		if e != nil {
			noteShareHTTPError(w, r, e)
			return
		}
		proof, e := auth.DecodeContentProof(cookies, r.Header.Get("X-CSRF-Token"), write)
		if e != nil {
			noteShareHTTPError(w, r, e)
			return
		}
		if len(r.Header.Values("Origin")) > 1 || len(r.Header.Values("X-CSRF-Token")) > 1 || r.Header.Get("Sec-Fetch-Site") == "cross-site" || write && r.Header.Get("Origin") != o.PublicOrigin || !write && r.Header.Get("Origin") != "" && r.Header.Get("Origin") != o.PublicOrigin {
			noteShareHTTPError(w, r, auth.ErrCSRF)
			return
		}
		a = knowledgeadmin.Access{TokenHash: proof.TokenHash, CSRF: proof.CSRF, RequestID: id}
		if write {
			if len(r.Header.Values("Idempotency-Key")) != 1 || strings.TrimSpace(r.Header.Get("Idempotency-Key")) == "" || len(r.Header.Get("Idempotency-Key")) > 128 {
				noteShareHTTPError(w, r, errNoteRequest)
				return
			}
			a.IdempotencyKey = r.Header.Get("Idempotency-Key")
			typ, params, e := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if e != nil || typ != "application/json" || len(r.Header.Values("Content-Type")) != 1 {
				noteShareHTTPError(w, r, errNoteRequest)
				return
			}
			for k, v := range params {
				if k != "charset" || !strings.EqualFold(v, "utf-8") {
					noteShareHTTPError(w, r, errNoteRequest)
					return
				}
			}
		}
	}
	q, events, e := noteShareQuery(r.URL.RawQuery, route.kind)
	if e != nil {
		noteShareHTTPError(w, r, e)
		return
	}
	var value any
	switch route.kind {
	case "health":
		var h notesharing.NoteSharingHealth
		h, e = repo.ReadSchemaHealth(ctx)
		status := 200
		if e != nil || !h.SchemaReady {
			status = 503
			h.Capability = true
			h.SchemaReady = false
		}
		noteShareResponse(w, r, h, status, false, notesharing.MaxDetailBytes)
		return
	case "own-list":
		value, e = repo.ListOwnNotes(ctx, a, q)
	case "head":
		value, e = repo.ReadOwnHead(ctx, a, route.id)
	case "submit":
		var in notesharing.SubmitInput
		if e = decodeNoteShare(r, &in); e == nil {
			value, e = repo.Submit(ctx, a, route.id, in)
		}
	case "withdraw":
		var in notesharing.WithdrawInput
		if e = decodeNoteShare(r, &in); e == nil {
			value, e = repo.Withdraw(ctx, a, route.id, in)
		}
	case "latest":
		value, e = repo.Latest(ctx)
	case "public-detail":
		value, e = repo.ReadPublic(ctx, route.id)
	case "shared-list":
		value, e = repo.ListShared(ctx, a, q)
	case "shared-detail":
		value, e = repo.ReadShared(ctx, a, route.id)
	case "admin-list":
		value, e = repo.ListAdmin(ctx, a, q)
	case "admin-detail":
		value, e = repo.ReadAdmin(ctx, a, route.id, events)
	case "decision":
		var in notesharing.DecisionInput
		if e = decodeNoteShare(r, &in); e == nil {
			value, e = repo.Decide(ctx, a, route.id, route.action, in)
		}
	default:
		e = notesharing.ErrNotFound
	}
	if e != nil {
		if r.Method == "GET" && errors.Is(e, notesharing.ErrInvalid) {
			e = errNoteRequest
		}
		noteShareHTTPError(w, r, e)
		return
	}
	max := notesharing.MaxDetailBytes
	if route.list || route.kind == "latest" {
		max = notesharing.MaxPageBytes
	}
	noteShareResponse(w, r, value, 200, route.private, max)
}
