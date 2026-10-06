package taxonomy

import "context"

type Service struct{ repo Repository }

func NewService(r Repository) *Service { return &Service{repo: r} }
func (s *Service) ListTopics(ctx context.Context, q Query) (Page[TopicSummary], error) {
	return s.repo.ListTopics(ctx, q)
}
func (s *Service) ReadTopic(ctx context.Context, id string) (TopicDetail, error) {
	return s.repo.ReadTopic(ctx, id)
}
func (s *Service) ListTopicKnowledge(ctx context.Context, id string, q Query) (Page[KnowledgeSummary], error) {
	return s.repo.ListTopicKnowledge(ctx, id, q)
}
