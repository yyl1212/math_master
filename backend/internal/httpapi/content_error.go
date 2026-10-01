package httpapi

import (
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"net/http"
)

var errContentPayloadTooLarge = errors.New("content request too large")

func contentError(w http.ResponseWriter, r *http.Request, err error) {
	status, code, message := 0, "", ""
	switch {
	case errors.Is(err, publication.ErrDraftConflict):
		status, code, message = 409, "DRAFT_CONFLICT", "This draft has changed. Reload it before continuing."
	case errors.Is(err, publication.ErrReviewConflict):
		status, code, message = 409, "REVIEW_CONFLICT", "This submission already has a review decision."
	case errors.Is(err, publication.ErrImmutableConflict):
		status, code, message = 409, "IMMUTABLE_CONFLICT", "This version conflicts with saved content."
	case errors.Is(err, publication.ErrVersionConflict):
		status, code, message = 409, "VERSION_CONFLICT", "This version conflicts with saved content."
	case errors.Is(err, publication.ErrIdempotencyConflict):
		status, code, message = 409, "IDEMPOTENCY_CONFLICT", "This request key was already used for different input."
	case errors.Is(err, publication.ErrPublicationStale):
		status, code, message = 409, "PUBLICATION_STALE", "Published content has changed. Prepare a new snapshot."
	case errors.Is(err, publication.ErrContentNotReady):
		status, code, message = 422, "CONTENT_NOT_READY", "Complete the required content before submitting."
	case errors.Is(err, publication.ErrContentInvalid):
		status, code, message = 422, "CONTENT_INVALID", "Content validation failed."
	case errors.Is(err, publication.ErrReviewRequired):
		status, code, message = 422, "REVIEW_REQUIRED", "Independent review is required before publication."
	case errors.Is(err, publication.ErrContentLimitExceeded):
		status, code, message = 422, "CONTENT_LIMIT_EXCEEDED", "Split this content into smaller reviewed batches."
	case errors.Is(err, errContentPayloadTooLarge):
		status, code, message = 413, "PAYLOAD_TOO_LARGE", "This content exceeds the request size limit."
	case errors.Is(err, publication.ErrContentNotConfigured):
		status, code, message = 503, "CONTENT_NOT_CONFIGURED", "Content management is temporarily unavailable."
	}
	if status == 0 {
		privateError(w, r, err, auth.CookieDelta{}, false)
		return
	}
	privateHeaders(w)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code, "message": message, "requestId": r.Header.Get("X-Request-ID")}})
}
func contentResponse(w http.ResponseWriter, r *http.Request, status int, v any) {
	raw, err := json.Marshal(v)
	if err != nil {
		contentError(w, r, auth.ErrUnavailable)
		return
	}
	if len(raw) > 4<<20 {
		contentError(w, r, publication.ErrContentLimitExceeded)
		return
	}
	privateHeaders(w)
	w.WriteHeader(status)
	_, _ = w.Write(raw)
}
