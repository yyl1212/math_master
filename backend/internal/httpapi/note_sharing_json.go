package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/notesharing"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

var errNoteRequest = errors.New("invalid note request")
var errNoteLarge = errors.New("note request too large")

func noteShareQuery(raw, kind string) (q notesharing.Query, events string, e error) {
	v, e := url.ParseQuery(raw)
	if e != nil {
		return q, events, errNoteRequest
	}
	for key, values := range v {
		if len(values) != 1 || values[0] == "" {
			return q, events, errNoteRequest
		}
		value := values[0]
		switch key {
		case "q":
			if kind != "own-list" && kind != "shared-list" {
				return q, events, errNoteRequest
			}
			q.Q = value
		case "topicKey":
			if kind != "own-list" && kind != "shared-list" {
				return q, events, errNoteRequest
			}
			q.TopicKey = value
		case "shareStatus":
			if kind != "own-list" {
				return q, events, errNoteRequest
			}
			q.Status = value
		case "status":
			if kind != "admin-list" {
				return q, events, errNoteRequest
			}
			q.Status = value
		case "cursor":
			if kind != "own-list" && kind != "shared-list" && kind != "admin-list" {
				return q, events, errNoteRequest
			}
			q.Cursor = value
		case "eventsCursor":
			if kind != "admin-detail" || len(value) > 512 {
				return q, events, errNoteRequest
			}
			events = value
		case "limit":
			if kind != "own-list" && kind != "shared-list" && kind != "admin-list" {
				return q, events, errNoteRequest
			}
			q.Limit, e = strconv.Atoi(value)
			if e != nil || q.Limit < 1 || q.Limit > 50 {
				return q, events, errNoteRequest
			}
		default:
			return q, events, errNoteRequest
		}
	}
	if notesharing.ValidateQuery(q) != nil {
		return q, events, errNoteRequest
	}
	return
}
func decodeNoteShare(r *http.Request, out any) error {
	if r.ContentLength > notesharing.MaxRequestBytes {
		return errNoteLarge
	}
	b, e := io.ReadAll(io.LimitReader(r.Body, notesharing.MaxRequestBytes+1))
	if e != nil {
		return e
	}
	if len(b) > notesharing.MaxRequestBytes {
		return errNoteLarge
	}
	return notesharing.Decode(bytes.NewReader(b), out)
}
func noteShareHTTPError(w http.ResponseWriter, r *http.Request, e error) {
	status, code, message := 503, "NOTE_SHARING_NOT_READY", "Note sharing is temporarily unavailable."
	switch {
	case errors.Is(e, notesharing.ErrLocked):
		status, code, message = 409, "NOTE_SHARING_LOCKED", "Withdraw the note share before editing."
	case errors.Is(e, notesharing.ErrStale):
		status, code, message = 409, "NOTE_SHARE_STALE", "The note share changed. Read its current state."
	case errors.Is(e, notesharing.ErrIdempotency):
		status, code, message = 409, "IDEMPOTENCY_CONFLICT", "The request key belongs to different input."
	case errors.Is(e, notesharing.ErrNotPublic):
		status, code, message = 404, "NOTE_SHARE_NOT_PUBLIC", "This note is not currently available for this viewer."
	case errors.Is(e, notesharing.ErrNotFound) || errors.Is(e, knowledgeadmin.ErrNotFound):
		status, code, message = 404, "NOT_FOUND", "Note was not found."
	case errors.Is(e, errNoteRequest):
		status, code, message = 400, "INVALID_REQUEST", "Check the request parameters."
	case errors.Is(e, errNoteLarge):
		status, code, message = 413, "INPUT_TOO_LARGE", "The request is too large."
	case errors.Is(e, notesharing.ErrInvalid) || errors.Is(e, knowledgeadmin.ErrInvalid):
		status, code, message = 422, "INVALID_FIELD", "Check the input fields."
	case errors.Is(e, auth.ErrAuthenticationRequired) || errors.Is(e, auth.ErrInvalidCookie):
		status, code, message = 401, "AUTHENTICATION_REQUIRED", "Sign in to continue."
	case errors.Is(e, auth.ErrForbidden):
		status, code, message = 403, "FORBIDDEN", "Administrator access is required."
	case errors.Is(e, auth.ErrPasswordChangeRequired):
		status, code, message = 403, "PASSWORD_CHANGE_REQUIRED", "Change your password before continuing."
	case errors.Is(e, auth.ErrCSRF):
		status, code, message = 403, "CSRF_FAILED", "Request verification failed."
	case errors.Is(e, errPrivateMethod):
		status, code, message = 405, "METHOD_NOT_ALLOWED", "Method not allowed."
	case errors.Is(e, context.DeadlineExceeded) || errors.Is(e, context.Canceled):
		status, code, message = 503, "REQUEST_INTERRUPTED", "The request was interrupted. Confirm the same request before trying a new action."
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code, "message": message, "requestId": r.Header.Get("X-Request-ID")}})
}
func noteShareResponse(w http.ResponseWriter, r *http.Request, value any, status int, private bool, max int) {
	b, e := json.Marshal(value)
	if e != nil || len(b) > max {
		noteShareHTTPError(w, r, notesharing.ErrNotReady)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if private {
		w.Header().Set("Cache-Control", "private, no-store")
	}
	w.WriteHeader(status)
	_, _ = w.Write(b)
}
