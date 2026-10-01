package httpapi

import (
	"github.com/yyl1212/math_master/backend/internal/auth"
	"net/http"
	"net/url"
	"strconv"
)

func privateUserQuery(raw string) (auth.UserQuery, error) {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return auth.UserQuery{}, auth.ErrInvalidInput
	}
	query := auth.UserQuery{Limit: 20}
	for key, items := range values {
		if len(items) != 1 {
			return query, auth.ErrInvalidInput
		}
		value := items[0]
		switch key {
		case "q":
			query.Q = value
		case "limit", "offset":
			if value == "" {
				return query, auth.ErrInvalidInput
			}
			for _, r := range value {
				if r < '0' || r > '9' {
					return query, auth.ErrInvalidInput
				}
			}
			n, err := strconv.Atoi(value)
			if err != nil {
				return query, auth.ErrInvalidInput
			}
			if key == "limit" {
				if n < 1 || n > 100 {
					return query, auth.ErrInvalidInput
				}
				query.Limit = n
			} else {
				query.Offset = n
			}
		default:
			return query, auth.ErrInvalidInput
		}
	}
	return auth.NormalizeUserQuery(query)
}
func serveAdmin(w http.ResponseWriter, r *http.Request, route privateRoute, cookies auth.Cookies, options AuthOptions) {
	fail := func(err error) { privateError(w, r, err, auth.CookieDelta{}, options.Production) }
	if options.Admin == nil {
		fail(errAuthNotConfigured)
		return
	}
	csrf, requestID := r.Header.Get("X-CSRF-Token"), r.Header.Get("X-Request-ID")
	switch route.kind {
	case "users":
		query, err := privateUserQuery(r.URL.RawQuery)
		if err != nil {
			fail(err)
			return
		}
		page, err := options.Admin.ListUsers(r.Context(), cookies, query)
		if err != nil {
			fail(err)
			return
		}
		privateResponse(w, r, 200, page, auth.CookieDelta{}, options.Production)
	case "roles":
		var input auth.RolesInput
		if err := decodePrivateJSON(r.Body, &input); err != nil {
			fail(err)
			return
		}
		delta, err := options.Admin.ReplaceRoles(r.Context(), cookies, csrf, route.target, input, requestID)
		if err != nil {
			fail(err)
			return
		}
		privateResponse(w, r, 204, nil, delta, options.Production)
	case "reset":
		var input auth.ResetInput
		if err := decodePrivateJSON(r.Body, &input); err != nil {
			fail(err)
			return
		}
		delta, err := options.Admin.ResetPassword(r.Context(), cookies, csrf, route.target, input, requestID)
		if err != nil {
			fail(err)
			return
		}
		privateResponse(w, r, 204, nil, delta, options.Production)
	}
}
