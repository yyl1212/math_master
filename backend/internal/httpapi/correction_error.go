package httpapi

import (
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/question"
	"math"
	"net/http"
	"strconv"
	"time"
)

func correctionError(w http.ResponseWriter, r *http.Request, e error) {
	status, code, message := 0, "", ""
	extra := map[string]any{}
	var rate *correction.RateError
	switch {
	case errors.Is(e, correction.ErrNotConfigured):
		status, code, message = 503, "CORRECTION_NOT_CONFIGURED", "Corrections are temporarily unavailable."
	case errors.Is(e, correction.ErrConflict):
		status, code, message = 409, "CORRECTION_CONFLICT", "This correction changed. Reload before continuing."
	case errors.Is(e, correction.ErrSourceStale):
		status, code, message = 409, "CORRECTION_SOURCE_STALE", "This source changed. Reload before continuing."
	case errors.Is(e, correction.ErrAnswerOverlap):
		status, code, message = 409, "CORRECTION_ANSWER_OVERLAP", "Finish or leave the overlapping assessment before reading this correction."
	case errors.Is(e, correction.ErrLeaseLost):
		status, code, message = 409, "CORRECTION_LEASE_LOST", "This job lease changed. Reload before continuing."
	case errors.Is(e, question.ErrIdempotencyConflict):
		status, code, message = 409, "IDEMPOTENCY_CONFLICT", "This request key was used for different input."
	case errors.As(e, &rate):
		status, code, message = 429, "RATE_LIMITED", "Too many requests. Try again later."
		extra["retryAt"] = rate.RetryAt.UTC()
		seconds := int(math.Ceil(time.Until(rate.RetryAt).Seconds()))
		if seconds < 1 {
			seconds = 1
		}
		w.Header().Set("Retry-After", strconv.Itoa(seconds))
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
func correctionResponse(w http.ResponseWriter, r *http.Request, status int, v any) {
	raw, e := json.Marshal(v)
	if e != nil || len(raw) > correction.MaxResponseBytes {
		correctionError(w, r, auth.ErrUnavailable)
		return
	}
	privateHeaders(w)
	w.WriteHeader(status)
	_, _ = w.Write(raw)
}
