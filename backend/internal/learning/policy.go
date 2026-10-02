package learning

import (
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/question"
	"time"
)

type Action string

const (
	ReadOverviewAction         Action = "readOverview"
	ListKnowledgeAction        Action = "listKnowledge"
	ReadKnowledgeAction        Action = "readKnowledge"
	StartKnowledgeAction       Action = "startKnowledge"
	CompleteKnowledgeAction    Action = "completeKnowledge"
	EnrollPathAction           Action = "enrollPath"
	ListPathsAction            Action = "listPaths"
	ReadPathAction             Action = "readPath"
	ListPathNodesAction        Action = "listPathNodes"
	CreatePracticeAction       Action = "createPractice"
	ReadPracticeAction         Action = "readPractice"
	AnswerPracticeAction       Action = "answerPractice"
	RevealPracticeAction       Action = "revealPractice"
	AbandonPracticeAction      Action = "abandonPractice"
	CreateAssessmentAction     Action = "createAssessment"
	ReadAssessmentAction       Action = "readAssessment"
	SubmitAssessmentAction     Action = "submitAssessment"
	AbandonAssessmentAction    Action = "abandonAssessment"
	ReadAssessmentResultAction Action = "readAssessmentResult"
	ListHistoryAction          Action = "listHistory"
	ReadAssetAction            Action = "readAsset"
)

func IsRead(a Action) bool {
	switch a {
	case ReadOverviewAction, ListKnowledgeAction, ReadKnowledgeAction, ListPathsAction, ReadPathAction, ListPathNodesAction, ReadPracticeAction, ReadAssessmentAction, ReadAssessmentResultAction, ListHistoryAction, ReadAssetAction:
		return true
	}
	return false
}
func IsHeavy(a Action) bool { return a == CreatePracticeAction || a == CreateAssessmentAction }
func validAction(a Action) bool {
	if IsRead(a) {
		return true
	}
	switch a {
	case StartKnowledgeAction, CompleteKnowledgeAction, EnrollPathAction, CreatePracticeAction, AnswerPracticeAction, RevealPracticeAction, AbandonPracticeAction, CreateAssessmentAction, SubmitAssessmentAction, AbandonAssessmentAction:
		return true
	}
	return false
}
func IsIdempotent(a Action) bool { return validAction(a) && !IsRead(a) }
func Authorize(u auth.User, a Action) error {
	if u.ID == "" {
		return auth.ErrAuthenticationRequired
	}
	if !validAction(a) {
		return auth.ErrInvalidInput
	}
	if u.MustChangePassword {
		return auth.ErrPasswordChangeRequired
	}
	for _, r := range u.Roles {
		if r == auth.RoleLearner {
			return nil
		}
	}
	return auth.ErrForbidden
}
func Rates(actor string, a Action) ([]auth.RateKey, error) {
	if !question.ValidID(actor) || !validAction(a) {
		return nil, auth.ErrInvalidInput
	}
	scope, userScope, global, user := "content_write", "content_write_user", 120, 30
	if IsHeavy(a) {
		scope, userScope, global, user = "content_heavy", "content_heavy_user", 40, 10
	} else if IsRead(a) {
		scope, userScope, global, user = "auth_read", "content_read_user", 600, 120
	}
	return []auth.RateKey{{Scope: scope, Limit: global, Window: time.Minute}, {Scope: userScope, Key: actor, Limit: user, Window: time.Minute}}, nil
}
