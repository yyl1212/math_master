package feedback

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/pagination"
	"github.com/yyl1212/math_master/backend/internal/question"
)

type NumberedRepository interface {
	ListFeedbackTicketPage(ctx context.Context, a question.Access, review bool, q ListQuery, p pagination.Request) (Envelope[pagination.Page[Metadata]], error)
}

func (s *Service) ListFeedbackTicketPage(ctx context.Context, a question.Access, review bool, q ListQuery, p pagination.Request) (Envelope[pagination.Page[Metadata]], error) {
	if r, ok := s.repo.(NumberedRepository); ok {
		return r.ListFeedbackTicketPage(ctx, a, review, q, p)
	}
	return Envelope[pagination.Page[Metadata]]{}, auth.ErrUnavailable
}
