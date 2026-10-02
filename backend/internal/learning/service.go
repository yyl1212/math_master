package learning

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/question"
	"reflect"
)

type Service struct {
	repo    Repository
	acquire func(context.Context) (func(), error)
}

func NewService(repo Repository, acquire func(context.Context) (func(), error)) (*Service, error) {
	if repo == nil || acquire == nil {
		return nil, ErrNotConfigured
	}
	v := reflect.ValueOf(repo)
	if v.Kind() == reflect.Pointer && v.IsNil() {
		return nil, ErrNotConfigured
	}
	return &Service{repo: repo, acquire: acquire}, nil
}
func (s *Service) AcquireValidation(ctx context.Context) (func(), error) {
	if s == nil || s.acquire == nil {
		return nil, ErrNotConfigured
	}
	return s.acquire(ctx)
}
func (s *Service) Preflight(ctx context.Context, a question.Access, action Action) (auth.User, error) {
	if s == nil || s.repo == nil {
		return auth.User{}, ErrNotConfigured
	}
	u, e := s.repo.LearningPreflight(ctx, a, action)
	if e != nil {
		return auth.User{}, e
	}
	if e = Authorize(u, action); e != nil {
		return auth.User{}, e
	}
	rates, e := Rates(u.ID, action)
	if e != nil {
		return auth.User{}, e
	}
	if e = s.repo.ConsumeRates(ctx, rates); e != nil {
		return auth.User{}, e
	}
	return u, nil
}
func (s *Service) ReadLearningOverview(ctx context.Context, a question.Access) (Overview, error) {
	return s.repo.ReadLearningOverview(ctx, a)
}
func (s *Service) ListLearningKnowledge(ctx context.Context, a question.Access, q ListQuery) (question.Page[KnowledgeState], error) {
	return s.repo.ListLearningKnowledge(ctx, a, q)
}
func (s *Service) ReadLearningKnowledge(ctx context.Context, a question.Access, id string, version int) (KnowledgeDetail, error) {
	return s.repo.ReadLearningKnowledge(ctx, a, id, version)
}
func (s *Service) StartLearning(ctx context.Context, a question.Access, id string, in StartInput) (KnowledgeState, error) {
	return s.repo.StartLearning(ctx, a, id, in)
}
func (s *Service) CompleteLearning(ctx context.Context, a question.Access, id string, in CompleteInput) (KnowledgeState, error) {
	return s.repo.CompleteLearning(ctx, a, id, in)
}
func (s *Service) EnrollLearningPath(ctx context.Context, a question.Access, id string, in EnrollInput) (PathView, error) {
	return s.repo.EnrollLearningPath(ctx, a, id, in)
}
func (s *Service) ListLearningPaths(ctx context.Context, a question.Access, q ListQuery) (question.Page[PathSummary], error) {
	return s.repo.ListLearningPaths(ctx, a, q)
}
func (s *Service) ReadLearningPath(ctx context.Context, a question.Access, id string) (PathView, error) {
	return s.repo.ReadLearningPath(ctx, a, id)
}
func (s *Service) ListLearningPathNodes(ctx context.Context, a question.Access, id string, q ListQuery) (question.Page[PathNode], error) {
	return s.repo.ListLearningPathNodes(ctx, a, id, q)
}
func (s *Service) CreatePractice(ctx context.Context, a question.Access, in assessment.PracticeCreateInput) (assessment.PracticeView, error) {
	return s.repo.CreatePractice(ctx, a, in)
}
func (s *Service) ReadPractice(ctx context.Context, a question.Access, id string) (assessment.PracticeView, error) {
	return s.repo.ReadPractice(ctx, a, id)
}
func (s *Service) AnswerPractice(ctx context.Context, a question.Access, id string, in assessment.Answer) (assessment.PracticeView, error) {
	return s.repo.AnswerPractice(ctx, a, id, in)
}
func (s *Service) RevealPractice(ctx context.Context, a question.Access, id string) (assessment.PracticeView, error) {
	return s.repo.RevealPractice(ctx, a, id)
}
func (s *Service) AbandonPractice(ctx context.Context, a question.Access, id string) (assessment.PracticeView, error) {
	return s.repo.AbandonPractice(ctx, a, id)
}
func (s *Service) CreateAssessment(ctx context.Context, a question.Access, in assessment.CreateInput) (assessment.AttemptView, error) {
	return s.repo.CreateAssessment(ctx, a, in)
}
func (s *Service) ReadAssessment(ctx context.Context, a question.Access, id string) (assessment.AttemptView, error) {
	return s.repo.ReadAssessment(ctx, a, id)
}
func (s *Service) SubmitAssessment(ctx context.Context, a question.Access, id string, in assessment.SubmitInput) (assessment.ResultView, error) {
	return s.repo.SubmitAssessment(ctx, a, id, in)
}
func (s *Service) AbandonAssessment(ctx context.Context, a question.Access, id string) (assessment.AttemptView, error) {
	return s.repo.AbandonAssessment(ctx, a, id)
}
func (s *Service) ReadAssessmentResult(ctx context.Context, a question.Access, id string) (assessment.ResultView, error) {
	return s.repo.ReadAssessmentResult(ctx, a, id)
}
func (s *Service) ListLearningHistory(ctx context.Context, a question.Access, q ListQuery) (question.Page[HistoryEntry], error) {
	return s.repo.ListLearningHistory(ctx, a, q)
}
func (s *Service) ReadLearningAsset(ctx context.Context, a question.Access, attemptID, sha string) ([]byte, error) {
	return s.repo.ReadLearningAsset(ctx, a, attemptID, sha)
}
