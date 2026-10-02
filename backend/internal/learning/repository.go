package learning

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/question"
)

type Repository interface {
	LearningPreflight(context.Context, question.Access, Action) (auth.User, error)
	ConsumeRates(context.Context, []auth.RateKey) error
	ReadLearningOverview(ctx context.Context, a question.Access) (Overview, error)
	ListLearningKnowledge(ctx context.Context, a question.Access, q ListQuery) (question.Page[KnowledgeState], error)
	ReadLearningKnowledge(ctx context.Context, a question.Access, id string, version int) (KnowledgeDetail, error)
	StartLearning(ctx context.Context, a question.Access, id string, in StartInput) (KnowledgeState, error)
	CompleteLearning(ctx context.Context, a question.Access, id string, in CompleteInput) (KnowledgeState, error)
	EnrollLearningPath(ctx context.Context, a question.Access, id string, in EnrollInput) (PathView, error)
	ListLearningPaths(ctx context.Context, a question.Access, q ListQuery) (question.Page[PathSummary], error)
	ReadLearningPath(ctx context.Context, a question.Access, id string) (PathView, error)
	ListLearningPathNodes(ctx context.Context, a question.Access, id string, q ListQuery) (question.Page[PathNode], error)
	CreatePractice(ctx context.Context, a question.Access, in assessment.PracticeCreateInput) (assessment.PracticeView, error)
	ReadPractice(ctx context.Context, a question.Access, id string) (assessment.PracticeView, error)
	AnswerPractice(ctx context.Context, a question.Access, id string, in assessment.Answer) (assessment.PracticeView, error)
	RevealPractice(ctx context.Context, a question.Access, id string) (assessment.PracticeView, error)
	AbandonPractice(ctx context.Context, a question.Access, id string) (assessment.PracticeView, error)
	CreateAssessment(ctx context.Context, a question.Access, in assessment.CreateInput) (assessment.AttemptView, error)
	ReadAssessment(ctx context.Context, a question.Access, id string) (assessment.AttemptView, error)
	SubmitAssessment(ctx context.Context, a question.Access, id string, in assessment.SubmitInput) (assessment.ResultView, error)
	AbandonAssessment(ctx context.Context, a question.Access, id string) (assessment.AttemptView, error)
	ReadAssessmentResult(ctx context.Context, a question.Access, id string) (assessment.ResultView, error)
	ListLearningHistory(ctx context.Context, a question.Access, q ListQuery) (question.Page[HistoryEntry], error)
	ReadLearningAsset(ctx context.Context, a question.Access, attemptID, sha string) ([]byte, error)
}
