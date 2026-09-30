package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/store"
	"log/slog"
	"net/http"
)

func requestID() string {
	b := make([]byte, 16)
	if _, e := rand.Read(b); e != nil {
		return "unavailable"
	}
	return hex.EncodeToString(b)
}
func response(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func apiError(w http.ResponseWriter, id string, e error) {
	status, code, msg := 500, "INTERNAL_ERROR", "Internal server error."
	switch {
	case errors.Is(e, store.ErrNotFound):
		status, code, msg = 404, "NOT_FOUND", "Resource not found."
	case errors.Is(e, store.ErrUnavailable):
		status, code, msg = 503, "SERVICE_UNAVAILABLE", "Service temporarily unavailable."
	}
	if status >= 500 {
		slog.Error("API request failed", "request_id", id, "error_code", code)
	}
	errorResponse(w, id, status, code, msg)
}
func errorResponse(w http.ResponseWriter, id string, status int, code, msg string) {
	response(w, status, map[string]any{"error": map[string]string{"code": code, "message": msg, "requestId": id}})
}
