package notification

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/question"
)

type Service struct{ repo Repository }

func NewService(repo Repository) (*Service, error) {
	if repo == nil {
		return nil, correction.ErrNotConfigured
	}
	return &Service{repo: repo}, nil
}
func (s *Service) Preflight(ctx context.Context, a question.Access, action Action) (auth.User, error) {
	if action != ListAction && action != CountAction && action != ReadAction && action != MarkReadAction {
		return auth.User{}, auth.ErrInvalidInput
	}
	u, e := s.repo.CorrectionPreflight(ctx, a, correction.ReadOwnAction)
	if e != nil {
		return u, e
	}
	return u, Authorize(u, action)
}
func (s *Service) ListNotifications(ctx context.Context, a question.Access, q Query) (Envelope[Page[Metadata]], error) {
	return s.repo.ListNotifications(ctx, a, q)
}
func (s *Service) ReadNotificationCount(ctx context.Context, a question.Access) (Envelope[UnreadCount], error) {
	return s.repo.ReadNotificationCount(ctx, a)
}
func (s *Service) ReadNotification(ctx context.Context, a question.Access, id string) (Envelope[Metadata], error) {
	return s.repo.ReadNotification(ctx, a, id)
}
func (s *Service) MarkNotificationRead(ctx context.Context, a question.Access, id string) (Envelope[ReadReceipt], error) {
	return s.repo.MarkNotificationRead(ctx, a, id)
}
