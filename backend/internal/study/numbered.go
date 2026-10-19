package study

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/pagination"
)

type NumberedRepository interface {
	ListStudyHistoryPage(ctx context.Context, a Access, q HistoryQuery, p pagination.Request) (pagination.ActorPage[HistoryEntry], error)
}

func (s *Service) ListStudyHistoryPage(ctx context.Context, a Access, q HistoryQuery, p pagination.Request) (pagination.ActorPage[HistoryEntry], error) {
	if r, ok := s.repo.(NumberedRepository); ok {
		return r.ListStudyHistoryPage(ctx, a, q, p)
	}
	return pagination.ActorPage[HistoryEntry]{}, auth.ErrUnavailable
}
