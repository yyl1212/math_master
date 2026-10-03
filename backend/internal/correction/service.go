package correction

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
	return &Service{repo: repo}, nil
}
func (s *Service) Preflight(ctx context.Context, a question.Access, action Action) (auth.User, error) {
	return s.repo.CorrectionPreflight(ctx, a, action)
}
func (s *Service) CreateCorrectionCase(ctx context.Context, a question.Access, in CaseInput) (Envelope[Receipt], error) {
	return s.repo.CreateCorrectionCase(ctx, a, in)
}
func (s *Service) CreateCorrectionPlan(ctx context.Context, a question.Access, id string, in PlanInput) (Envelope[Receipt], error) {
	return s.repo.CreateCorrectionPlan(ctx, a, id, in)
}
func (s *Service) UpdateCorrectionPlan(ctx context.Context, a question.Access, ref PlanRef, in PlanInput) (Envelope[Receipt], error) {
	return s.repo.UpdateCorrectionPlan(ctx, a, ref, in)
}
func (s *Service) SubmitCorrectionPlan(ctx context.Context, a question.Access, ref PlanRef, in SubmitInput) (Envelope[Receipt], error) {
	return s.repo.SubmitCorrectionPlan(ctx, a, ref, in)
}
func (s *Service) DecideCorrectionPlan(ctx context.Context, a question.Access, ref PlanRef, in DecisionInput) (Envelope[Receipt], error) {
	return s.repo.DecideCorrectionPlan(ctx, a, ref, in)
}
func (s *Service) RetryCorrectionJob(ctx context.Context, a question.Access, id string, in RetryInput) (Envelope[Receipt], error) {
	return s.repo.RetryCorrectionJob(ctx, a, id, in)
}
func (s *Service) ListCorrectionCases(ctx context.Context, a question.Access, q Query) (Envelope[Page[CaseMetadata]], error) {
	return s.repo.ListCorrectionCases(ctx, a, q)
}
func (s *Service) ReadCorrectionCase(ctx context.Context, a question.Access, id string) (Envelope[CaseMetadata], error) {
	return s.repo.ReadCorrectionCase(ctx, a, id)
}
func (s *Service) ListCorrectionPlans(ctx context.Context, a question.Access, id string, q Query) (Envelope[Page[PlanMetadata]], error) {
	return s.repo.ListCorrectionPlans(ctx, a, id, q)
}
func (s *Service) ReadCorrectionPlan(ctx context.Context, a question.Access, ref PlanRef, detail bool) (Envelope[PlanDetail], error) {
	return s.repo.ReadCorrectionPlan(ctx, a, ref, detail)
}
func (s *Service) ListCorrectionJobs(ctx context.Context, a question.Access, id string, q Query) (Envelope[Page[JobMetadata]], error) {
	return s.repo.ListCorrectionJobs(ctx, a, id, q)
}
func (s *Service) ListOwnCorrections(ctx context.Context, a question.Access, ref EvidenceRef, q Query) (Envelope[Page[ResultMetadata]], error) {
	return s.repo.ListOwnCorrections(ctx, a, ref, q)
}
func (s *Service) ReadOwnCorrection(ctx context.Context, a question.Access, id string, detail bool) (Envelope[ResultDetail], error) {
	return s.repo.ReadOwnCorrection(ctx, a, id, detail)
}
