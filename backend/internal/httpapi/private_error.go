package httpapi

import (
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"log/slog"
	"net/http"
	"strconv"
)

var errAuthNotConfigured = errors.New("accounts not configured")
var errPrivateMethod = errors.New("private method not allowed")

func privateHeaders(w http.ResponseWriter) {
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Type", "application/json")
}
func privateError(w http.ResponseWriter, r *http.Request, err error, delta auth.CookieDelta, production bool) {
	status, code, message := 503, "SERVICE_UNAVAILABLE", "Service temporarily unavailable."
	switch {
	case errors.Is(err, auth.ErrInvalidInput):
		status, code, message = 400, "INVALID_REQUEST", "Invalid request."
	case errors.Is(err, auth.ErrInvalidCookie):
		status, code, message = 400, "INVALID_COOKIE", "Invalid sign-in cookie."
	case errors.Is(err, auth.ErrInvalidCredentials):
		status, code, message = 401, "INVALID_CREDENTIALS", "Invalid username or password."
	case errors.Is(err, auth.ErrAuthenticationRequired):
		status, code, message = 401, "AUTHENTICATION_REQUIRED", "Please sign in to continue."
	case errors.Is(err, auth.ErrCSRF):
		status, code, message = 403, "CSRF_FAILED", "Request verification failed."
	case errors.Is(err, auth.ErrForbidden):
		status, code, message = 403, "FORBIDDEN", "You do not have permission."
	case errors.Is(err, auth.ErrPasswordChangeRequired):
		status, code, message = 403, "PASSWORD_CHANGE_REQUIRED", "Change your password to continue."
	case errors.Is(err, auth.ErrNotFound):
		status, code, message = 404, "NOT_FOUND", "Resource not found."
	case errors.Is(err, errPrivateMethod):
		status, code, message = 405, "METHOD_NOT_ALLOWED", "Method not allowed."
	case errors.Is(err, auth.ErrUsernameUnavailable):
		status, code, message = 409, "USERNAME_UNAVAILABLE", "This username is unavailable."
	case errors.Is(err, auth.ErrAlreadyAuthenticated):
		status, code, message = 409, "ALREADY_AUTHENTICATED", "Sign out before using another account."
	case errors.Is(err, auth.ErrLastAdminRequired):
		status, code, message = 409, "LAST_ADMIN_REQUIRED", "At least one administrator is required."
	case errors.Is(err, auth.ErrReauthRequired):
		status, code, message = 428, "REAUTHENTICATION_REQUIRED", "Verify your password before continuing."
	case errors.Is(err, errAuthNotConfigured):
		status, code, message = 503, "AUTH_NOT_CONFIGURED", "Accounts are temporarily unavailable."
	default:
		var limited *auth.RateLimitError
		if errors.As(err, &limited) {
			status, code, message = 429, "RATE_LIMITED", "Too many requests. Try again later."
			retry := limited.RetryAfterSeconds
			if retry < 1 {
				retry = 1
			}
			w.Header().Set("Retry-After", strconv.Itoa(retry))
		}
	}
	id := r.Header.Get("X-Request-ID")
	if status >= 500 {
		slog.Error("Private API request failed", "request_id", id, "error_code", code)
	}
	privateHeaders(w)
	if status == 400 && code == "INVALID_COOKIE" {
		setPrivateCookies(w, delta, production)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"error": map[string]string{"code": code, "message": message, "requestId": id}})
}
func privateResponse(w http.ResponseWriter, r *http.Request, status int, value any, delta auth.CookieDelta, production bool) {
	var raw []byte
	var err error
	if status != 204 {
		raw, err = json.Marshal(value)
		if err != nil || len(raw) > 1024*1024 {
			privateError(w, r, auth.ErrUnavailable, auth.CookieDelta{}, production)
			return
		}
	}
	privateHeaders(w)
	setPrivateCookies(w, delta, production)
	w.WriteHeader(status)
	if status != 204 {
		_, _ = w.Write(append(raw, '\n'))
	}
}
