package httpapi

import (
	"github.com/yyl1212/math_master/backend/internal/auth"
	"net/http"
)

func serveAuth(w http.ResponseWriter, r *http.Request, kind string, cookies auth.Cookies, options AuthOptions) {
	service := options.Accounts
	ctx := r.Context()
	csrf := r.Header.Get("X-CSRF-Token")
	requestID := r.Header.Get("X-Request-ID")
	fail := func(err error) { privateError(w, r, err, auth.CookieDelta{}, options.Production) }
	switch kind {
	case "session":
		user, err := service.Session(ctx, cookies)
		if err != nil {
			fail(err)
			return
		}
		privateResponse(w, r, 200, map[string]any{"data": map[string]any{"user": user}}, auth.CookieDelta{}, options.Production)
	case "context":
		view, delta, err := service.Context(ctx, cookies)
		if err != nil {
			fail(err)
			return
		}
		privateResponse(w, r, 200, map[string]any{"data": view}, delta, options.Production)
	case "register":
		var input auth.RegisterInput
		if err := decodePrivateJSON(r.Body, &input); err != nil {
			fail(err)
			return
		}
		user, delta, err := service.Register(ctx, cookies, csrf, input, requestID)
		if err != nil {
			fail(err)
			return
		}
		privateResponse(w, r, 201, map[string]any{"data": map[string]any{"user": user}}, delta, options.Production)
	case "login":
		var input auth.LoginInput
		if err := decodePrivateJSON(r.Body, &input); err != nil {
			fail(err)
			return
		}
		user, delta, err := service.Login(ctx, cookies, csrf, input, requestID)
		if err != nil {
			fail(err)
			return
		}
		privateResponse(w, r, 200, map[string]any{"data": map[string]any{"user": user}}, delta, options.Production)
	case "logout", "logout-all":
		var input struct{}
		if err := decodePrivateJSON(r.Body, &input); err != nil {
			fail(err)
			return
		}
		delta, err := service.Logout(ctx, cookies, csrf, kind == "logout-all", requestID)
		if err != nil {
			fail(err)
			return
		}
		privateResponse(w, r, 204, nil, delta, options.Production)
	case "password":
		var input auth.PasswordInput
		if err := decodePrivateJSON(r.Body, &input); err != nil {
			fail(err)
			return
		}
		delta, err := service.ChangePassword(ctx, cookies, csrf, input, requestID)
		if err != nil {
			fail(err)
			return
		}
		privateResponse(w, r, 204, nil, delta, options.Production)
	case "reauth":
		var input auth.ReauthInput
		if err := decodePrivateJSON(r.Body, &input); err != nil {
			fail(err)
			return
		}
		until, err := service.Reauthenticate(ctx, cookies, csrf, input, requestID)
		if err != nil {
			fail(err)
			return
		}
		privateResponse(w, r, 200, map[string]any{"data": map[string]string{"validUntil": until.UTC().Format("2006-01-02T15:04:05.999999999Z07:00")}}, auth.CookieDelta{}, options.Production)
	}
}
