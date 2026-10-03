package feedback

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/question"
)

type Service struct{ repo Repository }

func NewService(repo Repository) (*Service, error) {
	if repo == nil {
		return nil, ErrNotConfigured
	}
	return &Service{repo}, nil
}
func (s *Service) Preflight(ctx context.Context, a question.Access, action Action) (auth.User, error) {
	return s.repo.FeedbackPreflight(ctx, a, action)
}
func (s *Service) ReadFeedbackContext(ctx context.Context, a question.Access, q ContextQuery) (Envelope[Context], error) {
	return s.repo.ReadFeedbackContext(ctx, a, q)
}
func (s *Service) CreateFeedback(ctx context.Context, a question.Access, in CreateInput) (Envelope[Receipt], error) {
	return s.repo.CreateFeedback(ctx, a, in)
}
func (s *Service) ReplyFeedback(ctx context.Context, a question.Access, id string, in ReplyInput) (Envelope[Receipt], error) {
	return s.repo.ReplyFeedback(ctx, a, id, in)
}
func (s *Service) TransitionFeedback(ctx context.Context, a question.Access, id string, in TransitionInput) (Envelope[Receipt], error) {
	return s.repo.TransitionFeedback(ctx, a, id, in)
}
func (s *Service) ListFeedbackTickets(ctx context.Context, a question.Access, review bool, q ListQuery) (Envelope[Page[Metadata]], error) {
	return s.repo.ListFeedbackTickets(ctx, a, review, q)
}
func (s *Service) ReadFeedbackTicket(ctx context.Context, a question.Access, id string, review bool) (Envelope[Metadata], error) {
	return s.repo.ReadFeedbackTicket(ctx, a, id, review)
}
func (s *Service) ReadFeedbackEvents(ctx context.Context, a question.Access, id string, review bool, q ListQuery) (Envelope[DiscussionPage], error) {
	return s.repo.ReadFeedbackEvents(ctx, a, id, review, q)
}
