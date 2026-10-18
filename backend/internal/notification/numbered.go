package notification

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/pagination"
	"github.com/yyl1212/math_master/backend/internal/question"
)

type NumberedRepository interface {
	ListNotificationPage(ctx context.Context, a question.Access, q Query, p pagination.Request) (Envelope[pagination.Page[Metadata]], error)
}

func (s *Service) ListNotificationPage(ctx context.Context, a question.Access, q Query, p pagination.Request) (Envelope[pagination.Page[Metadata]], error) {
	if r, ok := s.repo.(NumberedRepository); ok {
		return r.ListNotificationPage(ctx, a, q, p)
	}
	return Envelope[pagination.Page[Metadata]]{}, auth.ErrUnavailable
}
