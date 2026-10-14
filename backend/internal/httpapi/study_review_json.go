package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/studyreview"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"unicode/utf8"
)

var errReviewRequest = errors.New("invalid review request")
var errReviewLarge = errors.New("review request too large")
var reviewInteger = regexp.MustCompile(`^(0|[1-9][0-9]*)$`)

func reviewQuery(raw, kind string) (q studyreview.QueueQuery, h studyreview.HistoryQuery, e error) {
	q.Limit = 20
	h.Limit = 20
	values, e := url.ParseQuery(raw)
	if e != nil {
		return q, h, errReviewRequest
	}
	for key, items := range values {
		if len(items) != 1 || items[0] == "" || (kind != "queue" && kind != "history") {
			return q, h, errReviewRequest
		}
		v := items[0]
		if key == "limit" || key == "offset" {
			if !reviewInteger.MatchString(v) {
				return q, h, errReviewRequest
			}
			n, err := strconv.Atoi(v)
			if err != nil {
				return q, h, errReviewRequest
			}
			if key == "limit" {
				q.Limit = n
				h.Limit = n
				continue
			}
			if kind != "queue" {
				return q, h, errReviewRequest
			}
			q.Offset = n
			continue
		}
		switch key {
		case "topicKey":
			q.TopicKey = v
			h.TopicKey = v
		case "q":
			if kind != "queue" {
				return q, h, errReviewRequest
			}
			q.Q = v
		case "bucket":
			if kind != "queue" {
				return q, h, errReviewRequest
			}
			q.Bucket = v
		case "knowledgeId":
			if kind != "history" {
				return q, h, errReviewRequest
			}
			h.KnowledgeID = v
		case "mode":
			if kind != "history" {
				return q, h, errReviewRequest
			}
			h.Mode = studyreview.Mode(v)
		case "rating":
			if kind != "history" {
				return q, h, errReviewRequest
			}
			h.Rating = studyreview.Rating(v)
		case "cursor":
			if kind != "history" {
				return q, h, errReviewRequest
			}
			h.Cursor = v
		default:
			return q, h, errReviewRequest
		}
	}
	if kind == "queue" && studyreview.ValidateQueue(q) != nil || kind == "history" && studyreview.ValidateHistory(h) != nil {
		return q, h, errReviewRequest
	}
	return
}
func decodeStudyReview(r *http.Request, out any) error {
	if r.ContentLength > 131072 {
		return errReviewLarge
	}
	raw, e := io.ReadAll(io.LimitReader(r.Body, 131073))
	if e != nil {
		return e
	}
	if len(raw) > 131072 {
		return errReviewLarge
	}
	if !utf8.Valid(raw) {
		return studyreview.ErrInvalid
	}
	if _, e = knowledgeadmin.CanonicalJSON(raw); e != nil {
		return studyreview.ErrInvalid
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || fields == nil {
		return studyreview.ErrInvalid
	}
	typ := reflect.TypeOf(out)
	if typ.Kind() != reflect.Pointer || typ.Elem().Kind() != reflect.Struct {
		return studyreview.ErrInvalid
	}
	typ = typ.Elem()
	for i := 0; i < typ.NumField(); i++ {
		name := typ.Field(i).Tag.Get("json")
		value, ok := fields[name]
		if !ok || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return studyreview.ErrInvalid
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(out) != nil {
		return studyreview.ErrInvalid
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return studyreview.ErrInvalid
	}
	return nil
}
func studyReviewHTTPError(w http.ResponseWriter, r *http.Request, e error) {
	status, code, message := 503, "STUDY_REVIEW_NOT_READY", "Review is temporarily unavailable."
	var active *studyreview.ActiveSessionError
	switch {
	case errors.As(e, &active):
		status, code, message = 409, "REVIEW_SESSION_ACTIVE", "Continue the existing session or explicitly abandon it."
	case errors.Is(e, studyreview.ErrReferenceTooLarge):
		status, code, message = 422, "REVIEW_REFERENCE_TOO_LARGE", "This material exceeds the review budget. Continue with ordinary reading."
	case errors.Is(e, studyreview.ErrStale) || errors.Is(e, knowledgeadmin.ErrStale):
		status, code, message = 409, "REVIEW_MATERIAL_CHANGED", "Knowledge changed. Read the current state before restarting."
	case errors.Is(e, studyreview.ErrConflict) || errors.Is(e, knowledgeadmin.ErrConflict):
		status, code, message = 409, "REVIEW_STATE_CONFLICT", "Review changed. Read its current state."
	case errors.Is(e, studyreview.ErrIdempotency) || errors.Is(e, knowledgeadmin.ErrIdempotency):
		status, code, message = 409, "IDEMPOTENCY_CONFLICT", "The request key belongs to different input."
	case errors.Is(e, studyreview.ErrUnavailable):
		status, code, message = 404, "REVIEW_KNOWLEDGE_UNAVAILABLE", "Knowledge is unavailable. Your own answer is retained."
	case errors.Is(e, studyreview.ErrNotFound) || errors.Is(e, knowledgeadmin.ErrNotFound):
		status, code, message = 404, "NOT_FOUND", "Review was not found."
	case errors.Is(e, errReviewRequest):
		status, code, message = 400, "INVALID_REQUEST", "Check the request parameters."
	case errors.Is(e, errReviewLarge):
		status, code, message = 413, "INPUT_TOO_LARGE", "The request is too large."
	case errors.Is(e, studyreview.ErrInvalid) || errors.Is(e, knowledgeadmin.ErrInvalid):
		status, code, message = 422, "INVALID_FIELD", "Check the input fields."
	case errors.Is(e, auth.ErrAuthenticationRequired) || errors.Is(e, auth.ErrInvalidCookie):
		status, code, message = 401, "AUTHENTICATION_REQUIRED", "Sign in to continue."
	case errors.Is(e, auth.ErrPasswordChangeRequired):
		status, code, message = 403, "PASSWORD_CHANGE_REQUIRED", "Change your password before continuing."
	case errors.Is(e, auth.ErrForbidden):
		status, code, message = 403, "FORBIDDEN", "This action is unavailable."
	case errors.Is(e, auth.ErrCSRF):
		status, code, message = 403, "CSRF_FAILED", "Request verification failed."
	case errors.Is(e, errPrivateMethod):
		status, code, message = 405, "METHOD_NOT_ALLOWED", "Method not allowed."
	case errors.Is(e, context.Canceled) || errors.Is(e, context.DeadlineExceeded):
		status, code, message = 503, "REQUEST_INTERRUPTED", "Confirm the same request before trying a new action."
	}
	problem := map[string]string{"code": code, "message": message, "requestId": r.Header.Get("X-Request-ID")}
	if active != nil {
		problem["sessionId"] = active.SessionID
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": problem})
}
func studyReviewResponse(w http.ResponseWriter, r *http.Request, value any, status int, private bool) {
	b, e := json.Marshal(value)
	if e != nil || len(b) > studyreview.MaxResponseBytes {
		studyReviewHTTPError(w, r, studyreview.ErrNotConfigured)
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
