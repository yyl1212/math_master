package taxonomy

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/publication"
)

type Service struct {
	repo  Repository
	guard *publication.Service
}

func NewService(r Repository, shared ...*publication.Service) *Service {
	gate := publication.NewService(nil)
	if len(shared) > 0 && shared[0] != nil {
		gate = shared[0]
	}
	return &Service{repo: r, guard: gate}
}
func (s *Service) ListTopics(ctx context.Context, q Query) (Page[TopicSummary], error) {
	return s.repo.ListTopics(ctx, q)
}
func (s *Service) ReadTopic(ctx context.Context, id string) (TopicDetail, error) {
	return s.repo.ReadTopic(ctx, id)
}
func (s *Service) ListTopicKnowledge(ctx context.Context, id string, q Query) (Page[KnowledgeSummary], error) {
	return s.repo.ListTopicKnowledge(ctx, id, q)
}
