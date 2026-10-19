package correction

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/pagination"
	"github.com/yyl1212/math_master/backend/internal/question"
)

type NumberedRepository interface {
	ListCorrectionCasePage(ctx context.Context, a question.Access, q Query, p pagination.Request) (Envelope[pagination.Page[CaseMetadata]], error)
}

func (s *Service) ListCorrectionCasePage(ctx context.Context, a question.Access, q Query, p pagination.Request) (Envelope[pagination.Page[CaseMetadata]], error) {
	if r, ok := s.repo.(NumberedRepository); ok {
		return r.ListCorrectionCasePage(ctx, a, q, p)
	}
	return Envelope[pagination.Page[CaseMetadata]]{}, auth.ErrUnavailable
}
