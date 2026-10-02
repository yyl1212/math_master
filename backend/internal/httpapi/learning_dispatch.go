package httpapi

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"io"
	"net/http"
)

func dispatchLearning(ctx context.Context, r *http.Request, route learningRoute, a question.Access, q learning.ListQuery, version int, s *learning.Service) (any, error) {
	var input any
	if !learning.IsRead(route.Action) {
		raw, e := io.ReadAll(io.LimitReader(r.Body, learningBodyLimit+1))
		if e != nil {
			return nil, auth.ErrInvalidInput
		}
		input, e = DecodeLearningInput(raw, route.Action)
		if e != nil {
			return nil, e
		}
	}
	switch route.Action {
	case learning.ReadOverviewAction:
		return s.ReadLearningOverview(ctx, a)
	case learning.ListKnowledgeAction:
		return s.ListLearningKnowledge(ctx, a, q)
	case learning.ReadKnowledgeAction:
		return s.ReadLearningKnowledge(ctx, a, route.ID, version)
	case learning.StartKnowledgeAction:
		return s.StartLearning(ctx, a, route.ID, input.(learning.StartInput))
	case learning.CompleteKnowledgeAction:
		return s.CompleteLearning(ctx, a, route.ID, input.(learning.CompleteInput))
	case learning.EnrollPathAction:
		return s.EnrollLearningPath(ctx, a, route.ID, input.(learning.EnrollInput))
	case learning.ListPathsAction:
		return s.ListLearningPaths(ctx, a, q)
	case learning.ReadPathAction:
		return s.ReadLearningPath(ctx, a, route.ID)
	case learning.ListPathNodesAction:
		return s.ListLearningPathNodes(ctx, a, route.ID, q)
	case learning.CreatePracticeAction:
		return s.CreatePractice(ctx, a, input.(assessment.PracticeCreateInput))
	case learning.ReadPracticeAction:
		return s.ReadPractice(ctx, a, route.ID)
	case learning.AnswerPracticeAction:
		return s.AnswerPractice(ctx, a, route.ID, input.(assessment.Answer))
	case learning.RevealPracticeAction:
		return s.RevealPractice(ctx, a, route.ID)
	case learning.AbandonPracticeAction:
		return s.AbandonPractice(ctx, a, route.ID)
	case learning.CreateAssessmentAction:
		return s.CreateAssessment(ctx, a, input.(assessment.CreateInput))
	case learning.ReadAssessmentAction:
		return s.ReadAssessment(ctx, a, route.ID)
	case learning.SubmitAssessmentAction:
		return s.SubmitAssessment(ctx, a, route.ID, input.(assessment.SubmitInput))
	case learning.AbandonAssessmentAction:
		return s.AbandonAssessment(ctx, a, route.ID)
	case learning.ReadAssessmentResultAction:
		return s.ReadAssessmentResult(ctx, a, route.ID)
	case learning.ListHistoryAction:
		return s.ListLearningHistory(ctx, a, q)
	case learning.ReadAssetAction:
		return s.ReadLearningAsset(ctx, a, route.ID, route.SHA)
	}
	return nil, auth.ErrNotFound
}
