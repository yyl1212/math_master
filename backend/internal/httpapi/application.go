package httpapi

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"mime"
	"net/http"
	"strings"
	"time"
)

type AuthOptions struct {
	Taxonomy     *TaxonomyOptions
	Correction   *CorrectionOptions
	Notification *NotificationOptions
	Feedback     *FeedbackOptions
	Learning     *LearningOptions
	Question     *QuestionOptions
	Content      *ContentOptions
	Accounts     *auth.Service
	Admin        *auth.AdminService
	PublicOrigin string
	Production   bool
}
type privateRoute struct{ kind, method, target string }

func routePrivate(path string) (privateRoute, bool) {
	for _, kind := range []string{"session", "context", "register", "login", "logout", "logout-all", "password", "reauth"} {
		if path == "/api/v1/auth/"+kind {
			method := "POST"
			if kind == "session" || kind == "context" {
				method = "GET"
			}
			return privateRoute{kind: kind, method: method}, true
		}
	}
	if path == "/api/v1/admin/users" {
		return privateRoute{kind: "users", method: "GET"}, true
	}
	parts := strings.Split(path, "/")
	if len(parts) == 7 && parts[1] == "api" && parts[2] == "v1" && parts[3] == "admin" && parts[4] == "users" {
		switch parts[6] {
		case "roles":
			return privateRoute{kind: "roles", method: "PUT", target: parts[5]}, true
		case "password-reset":
			return privateRoute{kind: "reset", method: "POST", target: parts[5]}, true
		}
	}
	return privateRoute{}, false
}
func NewApplicationHandler(reader Reader, pinger Pinger, options AuthOptions) http.Handler {
	public := NewHandler(reader, pinger)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/api/v2/topics" || strings.HasPrefix(path, "/api/v2/topics/") || path == "/api/v2/admin/publications" || strings.HasPrefix(path, "/api/v2/admin/publications/") || path == "/api/v2/content/topic-assignments" || strings.HasPrefix(path, "/api/v2/content/topic-assignments/") {
			o := TaxonomyOptions{PublicOrigin: options.PublicOrigin, Production: options.Production}
			if options.Taxonomy != nil {
				o = *options.Taxonomy
			}
			serveTaxonomy(w, r, o)
			return
		}
		if path == "/api/v1/corrections" || strings.HasPrefix(path, "/api/v1/corrections/") {
			o := CorrectionOptions{PublicOrigin: options.PublicOrigin, Production: options.Production}
			if options.Correction != nil {
				o = *options.Correction
			}
			if options.Accounts == nil {
				o.PublicOrigin = ""
			}
			serveCorrection(w, r, o)
			return
		}
		if path == "/api/v1/notifications" || strings.HasPrefix(path, "/api/v1/notifications/") {
			o := NotificationOptions{PublicOrigin: options.PublicOrigin, Production: options.Production}
			if options.Notification != nil {
				o = *options.Notification
			}
			if options.Accounts == nil {
				o.PublicOrigin = ""
			}
			serveNotification(w, r, o)
			return
		}

		if path == "/api/v1/feedback" || strings.HasPrefix(path, "/api/v1/feedback/") {
			o := FeedbackOptions{PublicOrigin: options.PublicOrigin, Production: options.Production}
			if options.Feedback != nil {
				o = *options.Feedback
			}
			if options.Accounts == nil {
				o.PublicOrigin = ""
			}
			serveFeedback(w, r, o)
			return
		}
		if path == "/api/v1/learning" || strings.HasPrefix(path, "/api/v1/learning/") {
			o := LearningOptions{PublicOrigin: options.PublicOrigin, Production: options.Production}
			if options.Learning != nil {
				o = *options.Learning
			}
			if options.Accounts == nil {
				o.PublicOrigin = ""
			}
			serveLearning(w, r, o)
			return
		}
		if path == "/api/v1/question-bank" || strings.HasPrefix(path, "/api/v1/question-bank/") {
			q := QuestionOptions{PublicOrigin: options.PublicOrigin, Production: options.Production}
			if options.Question != nil {
				q = *options.Question
			}
			if options.Accounts == nil {
				q.PublicOrigin = ""
			}
			serveQuestion(w, r, q)
			return
		}
		if path == "/api/v1/content" || strings.HasPrefix(path, "/api/v1/content/") {
			content := ContentOptions{PublicOrigin: options.PublicOrigin, Production: options.Production}
			if options.Content != nil {
				content = *options.Content
			}
			if options.Accounts == nil {
				content.PublicOrigin = ""
			}
			serveContent(w, r, content)
			return
		}
		if !(path == "/api/v1/auth" || strings.HasPrefix(path, "/api/v1/auth/") || path == "/api/v1/admin" || strings.HasPrefix(path, "/api/v1/admin/")) {
			public.ServeHTTP(w, r)
			return
		}
		id := requestID()
		w.Header().Set("X-Request-ID", id)
		privateHeaders(w)
		ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
		defer cancel()
		r = r.Clone(ctx)
		r.Header = r.Header.Clone()
		r.Header.Set("X-Request-ID", id)
		controller := http.NewResponseController(w)
		_ = controller.SetReadDeadline(time.Now().Add(4 * time.Second))
		route, ok := routePrivate(path)
		if !ok {
			privateError(w, r, auth.ErrNotFound, auth.CookieDelta{}, options.Production)
			return
		}
		if r.Method != route.method {
			w.Header().Set("Allow", route.method)
			privateError(w, r, errPrivateMethod, auth.CookieDelta{}, options.Production)
			return
		}
		if route.kind != "users" && (r.URL.RawQuery != "" || r.URL.ForceQuery) {
			privateError(w, r, auth.ErrInvalidInput, auth.CookieDelta{}, options.Production)
			return
		}
		if route.target != "" && auth.ValidateUserID(route.target) != nil {
			privateError(w, r, auth.ErrInvalidInput, auth.CookieDelta{}, options.Production)
			return
		}
		if options.Accounts == nil || options.PublicOrigin == "" {
			privateError(w, r, errAuthNotConfigured, auth.CookieDelta{}, options.Production)
			return
		}
		isWrite := r.Method != "GET"
		if (isWrite || route.kind == "context") && (len(r.Header.Values("Origin")) > 1 || len(r.Header.Values("X-CSRF-Token")) > 1 || len(r.Header.Values("X-Requested-With")) > 1) {
			privateError(w, r, auth.ErrCSRF, auth.CookieDelta{}, options.Production)
			return
		}
		if (isWrite || route.kind == "context") && (r.Header.Get("Sec-Fetch-Site") == "cross-site" || (isWrite && r.Header.Get("Origin") != options.PublicOrigin) || (!isWrite && r.Header.Get("Origin") != "" && r.Header.Get("Origin") != options.PublicOrigin) || (route.kind == "context" && r.Header.Get("X-Requested-With") != "MathMaster")) {
			privateError(w, r, auth.ErrCSRF, auth.CookieDelta{}, options.Production)
			return
		}
		if isWrite {
			typ, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil || typ != "application/json" || r.ContentLength > privateBodyLimit {
				privateError(w, r, auth.ErrInvalidInput, auth.CookieDelta{}, options.Production)
				return
			}
			for key, value := range params {
				if key != "charset" || !strings.EqualFold(value, "utf-8") {
					privateError(w, r, auth.ErrInvalidInput, auth.CookieDelta{}, options.Production)
					return
				}
			}
		}
		cookies, clear, err := privateCookies(r, options.Production)
		if err != nil {
			if route.kind == "session" {
				clear = auth.CookieDelta{}
			}
			privateError(w, r, err, clear, options.Production)
			return
		}
		if route.kind == "users" || route.kind == "roles" || route.kind == "reset" {
			serveAdmin(w, r, route, cookies, options)
			return
		}
		serveAuth(w, r, route.kind, cookies, options)
	})
}
