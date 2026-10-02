package httpapi

import (
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"net/http"
)

func learningError(w http.ResponseWriter, r *http.Request, e error) {
	status, code, message := 0, "", ""
	extra := map[string]any{}
	var format *question.NumericFormatError
	var active *learning.ActiveAttemptError
	var notReady *learning.NotReadyError
	switch {
	case errors.Is(e, learning.ErrNotConfigured):
		status, code, message = 503, "LEARNING_NOT_CONFIGURED", "Learning is temporarily unavailable."
	case errors.Is(e, learning.ErrVersionStale):
		status, code, message = 409, "LEARNING_VERSION_STALE", "Published learning content changed. Reload before continuing."
	case errors.Is(e, learning.ErrPrerequisitesUnmet):
		status, code, message = 409, "LEARNING_PREREQUISITES_UNMET", "Complete the prerequisites or take a diagnostic assessment."
	case errors.Is(e, learning.ErrAssessmentNotReady):
		status, code, message = 409, "ASSESSMENT_NOT_READY", "Five eligible questions are not available yet."
		if errors.As(e, &notReady) && notReady.RetryAt != nil {
			extra["retryAt"] = notReady.RetryAt
		}
	case errors.Is(e, learning.ErrAssessmentActive):
		status, code, message = 409, "ASSESSMENT_ACTIVE", "Continue or abandon the existing attempt."
		if errors.As(e, &active) {
			extra["activeAttempt"] = active.Summary
		}
	case errors.Is(e, learning.ErrAssessmentExpired):
		status, code, message = 409, "ASSESSMENT_EXPIRED", "This attempt has expired."
	case errors.Is(e, learning.ErrStateConflict):
		status, code, message = 409, "ASSESSMENT_STATE_CONFLICT", "This attempt cannot accept the action."
	case errors.As(e, &format):
		status, code, message = 400, "ANSWER_FORMAT_INVALID", "Check the requested numeric format."
		extra["formatCode"] = format.Code
	case errors.Is(e, learning.ErrAnswerFormatInvalid):
		status, code, message = 400, "ANSWER_FORMAT_INVALID", "Check the requested answer format."
	case errors.Is(e, question.ErrInvalid):
		status, code, message = 400, "INVALID_REQUEST", "The request is invalid."
	case errors.Is(e, question.ErrIdempotencyConflict):
		status, code, message = 409, "IDEMPOTENCY_CONFLICT", "This request key was used for different input."
	case errors.Is(e, errQuestionPayloadTooLarge):
		status, code, message = 413, "PAYLOAD_TOO_LARGE", "This request exceeds the size limit."
	}
	if status == 0 {
		privateError(w, r, e, auth.CookieDelta{}, false)
		return
	}
	extra["code"] = code
	extra["message"] = message
	extra["requestId"] = r.Header.Get("X-Request-ID")
	privateHeaders(w)
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": extra})
}
func learningResponse(w http.ResponseWriter, r *http.Request, status int, v any) {
	raw, e := json.Marshal(v)
	if e != nil || len(raw) > 4<<20 {
		learningError(w, r, auth.ErrUnavailable)
		return
	}
	privateHeaders(w)
	w.WriteHeader(status)
	_, _ = w.Write(raw)
}
