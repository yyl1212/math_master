package httpapi

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/notification"
	"github.com/yyl1212/math_master/backend/internal/question"
	"net/http"
	"strings"
	"time"
)

type NotificationOptions struct {
	Service      *notification.Service
	PublicOrigin string
	Production   bool
}
type notificationRoute struct {
	Action     notification.Action
	ID, Method string
}

func routeNotification(path, method string) (notificationRoute, error) {
	r := notificationRoute{Method: "GET"}
	switch {
	case path == "/api/v1/notifications":
		r.Action = notification.ListAction
	case path == "/api/v1/notifications/count":
		r.Action = notification.CountAction
	case strings.HasPrefix(path, "/api/v1/notifications/"):
		p := strings.Split(strings.TrimPrefix(path, "/api/v1/notifications/"), "/")
		if len(p) != 1 && len(p) != 2 {
			return r, auth.ErrNotFound
		}
		r.ID = p[0]
		if !question.ValidID(r.ID) {
			return r, auth.ErrInvalidInput
		}
		r.Action = notification.ReadAction
		if len(p) == 2 {
			if p[1] != "read" {
				return r, auth.ErrNotFound
			}
			r.Action = notification.MarkReadAction
			r.Method = "POST"
		}
	default:
		return r, auth.ErrNotFound
	}
	if method != r.Method {
		return r, errPrivateMethod
	}
	return r, nil
}
func serveNotification(w http.ResponseWriter, r *http.Request, o NotificationOptions) {
	id := requestID()
	privateHeaders(w)
	w.Header().Set("X-Request-ID", id)
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	r = r.Clone(ctx)
	r.Header = r.Header.Clone()
	r.Header.Set("X-Request-ID", id)
	_ = http.NewResponseController(w).SetReadDeadline(time.Now().Add(8 * time.Second))
	fail := func(e error) {
		if ctx.Err() != nil {
			e = auth.ErrUnavailable
		}
		notificationError(w, r, e)
	}
	if r.URL.RawPath != "" || r.URL.ForceQuery && r.URL.RawQuery == "" {
		fail(auth.ErrInvalidInput)
		return
	}
	route, e := routeNotification(r.URL.Path, r.Method)
	if e != nil {
		if e == errPrivateMethod {
			w.Header().Set("Allow", route.Method)
		}
		fail(e)
		return
	}
	q, _, e := correctionQuery(r.URL.RawQuery, route.Action == notification.ListAction, false)
	if e != nil {
		fail(e)
		return
	}
	if o.Service == nil || o.PublicOrigin == "" {
		fail(correction.ErrNotConfigured)
		return
	}
	write := route.Action == notification.MarkReadAction
	a, e := correctionPrivateAccess(r, o.PublicOrigin, o.Production, write)
	if e != nil {
		fail(e)
		return
	}
	if _, e = o.Service.Preflight(ctx, a, route.Action); e != nil {
		fail(e)
		return
	}
	if write && r.ContentLength > correction.MaxRequestBytes {
		fail(auth.ErrInvalidInput)
		return
	}
	v, status, e := dispatchNotification(ctx, r, route, a, notification.Query{Limit: q.Limit, Cursor: q.Cursor}, o.Service)
	if e != nil {
		fail(e)
		return
	}
	if ctx.Err() != nil {
		fail(auth.ErrUnavailable)
		return
	}
	correctionResponse(w, r, status, v)
}
