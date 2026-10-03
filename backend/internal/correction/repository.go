package correction

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/question"
)

type Repository interface {
	CorrectionPreflight(ctx context.Context, a question.Access, action Action) (auth.User, error)
	CreateCorrectionCase(ctx context.Context, a question.Access, in CaseInput) (Envelope[Receipt], error)
	CreateCorrectionPlan(ctx context.Context, a question.Access, id string, in PlanInput) (Envelope[Receipt], error)
	UpdateCorrectionPlan(ctx context.Context, a question.Access, ref PlanRef, in PlanInput) (Envelope[Receipt], error)
	SubmitCorrectionPlan(ctx context.Context, a question.Access, ref PlanRef, in SubmitInput) (Envelope[Receipt], error)
	DecideCorrectionPlan(ctx context.Context, a question.Access, ref PlanRef, in DecisionInput) (Envelope[Receipt], error)
	RetryCorrectionJob(ctx context.Context, a question.Access, id string, in RetryInput) (Envelope[Receipt], error)
	ListCorrectionCases(ctx context.Context, a question.Access, q Query) (Envelope[Page[CaseMetadata]], error)
	ReadCorrectionCase(ctx context.Context, a question.Access, id string) (Envelope[CaseMetadata], error)
	ListCorrectionPlans(ctx context.Context, a question.Access, id string, q Query) (Envelope[Page[PlanMetadata]], error)
	ReadCorrectionPlan(ctx context.Context, a question.Access, ref PlanRef, detail bool) (Envelope[PlanDetail], error)
	ListCorrectionJobs(ctx context.Context, a question.Access, id string, q Query) (Envelope[Page[JobMetadata]], error)
	ListOwnCorrections(ctx context.Context, a question.Access, ref EvidenceRef, q Query) (Envelope[Page[ResultMetadata]], error)
	ReadOwnCorrection(ctx context.Context, a question.Access, id string, detail bool) (Envelope[ResultDetail], error)
}
