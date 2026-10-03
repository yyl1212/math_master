package httpapi

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/notification"
	"github.com/yyl1212/math_master/backend/internal/question"
	"net/http"
)

func dispatchNotification(ctx context.Context, r *http.Request, route notificationRoute, a question.Access, q notification.Query, s *notification.Service) (any, int, error) {
	switch route.Action {
	case notification.ListAction:
		v, e := s.ListNotifications(ctx, a, q)
		return v, 200, e
	case notification.CountAction:
		v, e := s.ReadNotificationCount(ctx, a)
		return v, 200, e
	case notification.ReadAction:
		v, e := s.ReadNotification(ctx, a, route.ID)
		return v, 200, e
	case notification.MarkReadAction:
		if _, e := readNotificationInput(r.Body); e != nil {
			return nil, 0, e
		}
		v, e := s.MarkNotificationRead(ctx, a, route.ID)
		return v, v.Data.Status, e
	}
	return nil, 0, auth.ErrNotFound
}
