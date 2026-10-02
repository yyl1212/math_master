package httpapi

import (
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/question"
	"net/http"
)

var errQuestionPayloadTooLarge = errors.New("question request too large")

func questionError(w http.ResponseWriter, r *http.Request, e error) {
	status, code, message := 0, "", ""
	switch {
	case errors.Is(e, question.ErrDraftConflict):
		status, code, message = 409, "QUESTION_DRAFT_CONFLICT", "This draft has changed. Reload before continuing."
	case errors.Is(e, question.ErrPublicationStale):
		status, code, message = 409, "QUESTION_PUBLICATION_STALE", "Published snapshots changed. Prepare again."
	case errors.Is(e, question.ErrReviewConflict):
		status, code, message = 409, "REVIEW_CONFLICT", "This submission already has a final decision."
	case errors.Is(e, question.ErrIdempotencyConflict):
		status, code, message = 409, "IDEMPOTENCY_CONFLICT", "This request key was used for different input."
	case errors.Is(e, question.ErrImmutableConflict):
		status, code, message = 409, "IMMUTABLE_CONFLICT", "This fixed version conflicts with saved questions."
	case errors.Is(e, question.ErrVersionConflict):
		status, code, message = 409, "VERSION_CONFLICT", "Create a new version for changed questions."
	case errors.Is(e, question.ErrInvalid):
		status, code, message = 422, "QUESTION_INVALID", "Question validation failed."
	case errors.Is(e, question.ErrNotReady):
		status, code, message = 422, "QUESTION_NOT_READY", "Complete the required questions before submitting."
	case errors.Is(e, question.ErrLimitExceeded):
		status, code, message = 422, "QUESTION_LIMIT_EXCEEDED", "Split this question bank into smaller reviewed batches."
	case errors.Is(e, question.ErrReviewRequired):
		status, code, message = 422, "REVIEW_REQUIRED", "Independent review is required."
	case errors.Is(e, errQuestionPayloadTooLarge):
		status, code, message = 413, "PAYLOAD_TOO_LARGE", "This request exceeds the size limit."
	case errors.Is(e, question.ErrNotConfigured):
		status, code, message = 503, "QUESTION_BANK_NOT_CONFIGURED", "Question management is temporarily unavailable."
	}
	if status == 0 {
		privateError(w, r, e, auth.CookieDelta{}, false)
		return
	}
	privateHeaders(w)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code, "message": message, "requestId": r.Header.Get("X-Request-ID")}})
}
func questionResponse(w http.ResponseWriter, r *http.Request, status int, v any) {
	raw, e := json.Marshal(v)
	if e != nil {
		questionError(w, r, auth.ErrUnavailable)
		return
	}
	if len(raw) > question.MaxResponseBytes {
		questionError(w, r, question.ErrLimitExceeded)
		return
	}
	privateHeaders(w)
	w.WriteHeader(status)
	_, _ = w.Write(raw)
}
