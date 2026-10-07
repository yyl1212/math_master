package e2etest

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/study"
	"regexp"
	"strconv"
)

// A trusted synthetic runtime revision, never supplied by a control request.
const fixtureCutoverCodeSHA = "dddddddddddddddddddddddddddddddddddddddd"

func resetTopicCutover(ctx context.Context, db *sql.DB, s *store.Store, accounts *auth.Service, admin *auth.AdminService, root string, normal content.ValidatedPackage) error {
	if e := resetLearning(ctx, db, s, accounts, admin, root, normal, LearningBasic); e != nil {
		return e
	}
	learner, e := fixtureAccess(ctx, accounts, "auth_learner", false)
	if e != nil {
		return e
	}
	detail, e := s.ReadLearningKnowledge(ctx, learner, "learning-root", 1)
	if e != nil {
		return e
	}
	if _, e = s.StartLearning(ctx, learner, "learning-root", learning.StartInput{Knowledge: detail.State.Knowledge, ExpectedKnowledgeHead: detail.KnowledgeHead}); e != nil {
		return e
	}
	learner, e = nextFixtureAccess(learner)
	if e != nil {
		return e
	}
	if _, e = s.CompleteLearning(ctx, learner, "learning-root", learning.CompleteInput{Knowledge: detail.State.Knowledge, ExpectedKnowledgeHead: detail.KnowledgeHead}); e != nil {
		return e
	}
	other, e := fixtureAccess(ctx, accounts, "learning_other", false)
	if e != nil {
		return e
	}
	attempt, e := s.CreateAssessment(ctx, other, assessment.CreateInput{Knowledge: detail.State.Knowledge, Blueprint: detail.Blueprints[0].Blueprint, Mode: assessment.ModeDiagnostic, ExpectedKnowledgeHead: detail.KnowledgeHead, ExpectedQuestionHead: *detail.QuestionHead})
	if e != nil {
		return e
	}
	submit := assessment.SubmitInput{Answers: []assessment.PositionAnswer{}}
	arithmetic := regexp.MustCompile(`Calculate (\d+) \+ (\d+)\.`)
	for _, q := range attempt.Questions {
		match := arithmetic.FindStringSubmatch(q.Prompt)
		if len(match) != 3 {
			return fmt.Errorf("unsupported original fixture arithmetic")
		}
		left, _ := strconv.Atoi(match[1])
		right, _ := strconv.Atoi(match[2])
		answer := strconv.Itoa(left + right)
		submit.Answers = append(submit.Answers, assessment.PositionAnswer{Position: q.Position, Instance: q.Instance, Answer: assessment.Answer{Kind: "numeric", Raw: &answer}})
	}
	other, e = nextFixtureAccess(other)
	if e != nil {
		return e
	}
	result, e := s.SubmitAssessment(ctx, other, attempt.Summary.ID, submit)
	if e != nil || result.Score == nil || *result.Score != 5 {
		return fmt.Errorf("original diagnostic setup failed")
	}
	learner, e = nextFixtureAccess(learner)
	if e != nil {
		return e
	}
	if _, e = s.CreateAssessment(ctx, learner, assessment.CreateInput{Knowledge: detail.State.Knowledge, Blueprint: detail.Blueprints[0].Blueprint, Mode: assessment.ModeDiagnostic, ExpectedKnowledgeHead: detail.KnowledgeHead, ExpectedQuestionHead: *detail.QuestionHead}); e != nil {
		return e
	}
	// The same mathematical package gains a reviewed topic sidecar; old bytes stay frozen.
	if e = publishTopicCatalogueFixture(ctx, db, s, accounts, root); e != nil {
		return e
	}
	batch, e := s.MigrateLegacyStudyBatch(ctx, 50, nil)
	if e != nil || !batch.Done || batch.CreatedEvents != 2 {
		return fmt.Errorf("explicit legacy migration fixture failed")
	}
	control, e := readTaxonomyControl(ctx, db)
	if e != nil {
		return e
	}
	_, e = s.ActivateTopicExperience(ctx, study.CutoverInput{ExpectedPair: control.Pair, CodeSHA: fixtureCutoverCodeSHA, ExpectedMigrationBatchID: batch.BatchID, Reason: "Explicit original isolated business cutover fixture.", BackupRecord: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"})
	return e
}
