package httpapi

import (
	"github.com/yyl1212/math_master/backend/internal/auth"
	"net/http"
	"strings"
	"time"
)

func cookieNames(production bool) (string, string) {
	if production {
		return "__Host-mm_session", "__Host-mm_preauth"
	}
	return "mm_session_dev", "mm_preauth_dev"
}
func privateCookies(r *http.Request, production bool) (auth.Cookies, auth.CookieDelta, error) {
	sessionName, preauthName := cookieNames(production)
	var cookies auth.Cookies
	var clear auth.CookieDelta
	seen := make(map[string]bool, 2)
	bad := false
	for _, header := range r.Header.Values("Cookie") {
		for _, part := range strings.Split(header, ";") {
			part = strings.TrimLeft(part, " \t")
			name, value, hasValue := strings.Cut(part, "=")
			selected := strings.TrimSpace(name)
			if selected != sessionName && selected != preauthName {
				continue
			}
			_, err := auth.DecodeSecret(value)
			if seen[selected] || name != selected || !hasValue || err != nil {
				bad = true
				if selected == sessionName {
					clear.ClearSession = true
				} else {
					clear.ClearPreauth = true
				}
			}
			seen[selected] = true
			if selected == sessionName {
				cookies.Session = value
			} else {
				cookies.Preauth = value
			}
		}
	}
	if bad {
		return auth.Cookies{}, clear, auth.ErrInvalidCookie
	}
	return cookies, auth.CookieDelta{}, nil
}
func setPrivateCookies(w http.ResponseWriter, delta auth.CookieDelta, production bool) {
	sessionName, preauthName := cookieNames(production)
	apply := func(name, value string, clear bool, maxAge int) {
		if value == "" && !clear {
			return
		}
		cookie := &http.Cookie{Name: name, Value: value, Path: "/", HttpOnly: true, Secure: production, SameSite: http.SameSiteLaxMode, MaxAge: maxAge}
		if value == "" {
			cookie.MaxAge = -1
			cookie.Expires = time.Unix(1, 0).UTC()
		}
		http.SetCookie(w, cookie)
	}
	apply(sessionName, delta.SetSession, delta.ClearSession, 604800)
	apply(preauthName, delta.SetPreauth, delta.ClearPreauth, 600)
}
