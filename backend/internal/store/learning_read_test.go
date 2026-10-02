package store_test

import (
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"strings"
	"testing"
)

func TestLearningReadEmptyAndNoSideEffects(t *testing.T) {
	empty := newWorkflowFixture(t)
	overview, e := empty.repo.ReadLearningOverview(empty.ctx, empty.Access("author_a", false))
	if e != nil || overview.KnowledgeHead != nil || overview.QuestionHead != nil || len(overview.AvailablePaths) != 0 || overview.StartedCount != 0 || overview.CompletedCount != 0 || overview.EffectivePassedCount != 0 || overview.HistoricalUnlockedCount != 0 {
		t.Fatal(overview, e)
	}
	f := newLearningFixture(t)
	path := f.publishLearningGraph()
	detail, e := f.repo.ReadLearningKnowledge(f.ctx, f.Access("learner_a", false), f.knowledge.ID, 1)
	if e != nil || detail.State.StartedAt != nil || !detail.State.CanEnter || len(detail.Blueprints) != 1 || !detail.Blueprints[0].Ready {
		t.Fatal(detail, e)
	}
	page, e := f.repo.ListLearningKnowledge(f.ctx, f.Access("learner_a", false), learning.ListQuery{})
	if e != nil || page.Total != 4 || page.Limit != 20 || len(page.Items) != 4 {
		t.Fatal(page, e)
	}
	paths, e := f.repo.ListLearningPaths(f.ctx, f.Access("learner_a", false), learning.ListQuery{})
	if e != nil || paths.Total != 0 {
		t.Fatal(paths, e)
	}
	overview, e = f.repo.ReadLearningOverview(f.ctx, f.Access("learner_a", false))
	if e != nil || len(overview.AvailablePaths) != 1 || overview.AvailablePaths[0].Path != path || overview.AvailablePaths[0].TotalNodes != 3 {
		t.Fatal(overview, e)
	}
	for _, v := range []any{page, detail, overview, paths} {
		raw, _ := json.Marshal(v)
		for _, bad := range []string{"correctNumeric", "explanation", "parameters", "sourceMap", "witness"} {
			if strings.Contains(string(raw), bad) {
				t.Fatal("summary leaked answer", bad)
			}
		}
	}
	if f.count(`SELECT count(*) FROM learning_events`) != 0 || f.count(`SELECT count(*) FROM learning_unlocks`) != 0 || f.count(`SELECT count(*) FROM learner_question_views`) != 0 || f.exposureSequence("learner_a") != 0 {
		t.Fatal("GET manufactured progress")
	}
}
func TestLearningReadCountsSeparateAndOwnedPath(t *testing.T) {
	f := newLearningFixture(t)
	path := f.publishLearningGraph()
	joined, e := f.repo.EnrollLearningPath(f.ctx, f.Access("learner_a", false), path.ID, learning.EnrollInput{Path: path, ExpectedKnowledgeHead: *f.KHead()})
	if e != nil {
		t.Fatal(e)
	}
	v, e := f.repo.CreateAssessment(f.ctx, f.Access("learner_a", false), f.assessmentInput(assessment.ModeNode))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = f.repo.SubmitAssessment(f.ctx, f.Access("learner_a", false), v.Summary.ID, f.answers(v, 5)); e != nil {
		t.Fatal(e)
	}
	got, e := f.repo.ReadLearningPath(f.ctx, f.Access("learner_a", false), joined.Summary.ID)
	if e != nil || got.Summary.TotalNodes != 3 || got.Summary.PassedNodes != 1 || got.Summary.CompletedNodes != 0 || got.Summary.UnlockedNodes != 1 {
		t.Fatal(got, e)
	}
	overview, e := f.repo.ReadLearningOverview(f.ctx, f.Access("learner_a", false))
	if e != nil || overview.CompletedCount != 0 || overview.EffectivePassedCount != 1 || overview.HistoricalUnlockedCount != 1 {
		t.Fatal(overview, e)
	}
	if _, e = f.repo.ReadLearningPath(f.ctx, f.Access("learner_b", false), joined.Summary.ID); !errors.Is(e, auth.ErrNotFound) {
		t.Fatal("route ownership", e)
	}
	if _, e = f.repo.ListLearningPathNodes(f.ctx, f.Access("learner_b", false), joined.Summary.ID, learning.ListQuery{}); !errors.Is(e, auth.ErrNotFound) {
		t.Fatal(e)
	}
	before := f.count(`SELECT count(*) FROM learning_unlocks`)
	nodes, e := f.repo.ListLearningPathNodes(f.ctx, f.Access("learner_a", false), joined.Summary.ID, learning.ListQuery{Limit: 2, Offset: 1})
	if e != nil || nodes.Total != 3 || len(nodes.Items) != 2 || nodes.Items[0].Position != 1 || f.count(`SELECT count(*) FROM learning_unlocks`) != before {
		t.Fatal(nodes, e)
	}
}
func TestLearningReadFixedRouteWithdrawalAndNewVersion(t *testing.T) {
	f := newLearningFixture(t)
	path := f.publishLearningGraph()
	joined, e := f.repo.EnrollLearningPath(f.ctx, f.Access("learner_a", false), path.ID, learning.EnrollInput{Path: path, ExpectedKnowledgeHead: *f.KHead()})
	if e != nil {
		t.Fatal(e)
	}
	v := learningGraphInput(f)
	v.Package.Version = 2
	v.Package.Paths[0].Version = 2
	v.Package.Paths[0].Nodes = append(v.Package.Paths[0].Nodes, questionToContentRef("workflow-related", 1))
	old := f.KHead()
	sub := f.ApprovedInput(v)
	f.Activate(f.Prepare(sub, old), old)
	got, e := f.repo.ReadLearningPath(f.ctx, f.Access("learner_a", false), joined.Summary.ID)
	if e != nil || !got.Summary.NewVersionAvailable || got.Summary.TotalNodes != 3 {
		t.Fatal(got, e)
	}
	workflowWithdraw(f.workflowFixture, contentWithdrawalKnowledge("workflow-fractions", 1), f.KHead())
	nodes, e := f.repo.ListLearningPathNodes(f.ctx, f.Access("learner_a", false), joined.Summary.ID, learning.ListQuery{})
	if e != nil || nodes.Total != 3 || len(nodes.Items) != 3 || nodes.Items[0].Available || !containsReason(nodes.Items[0].Reasons, assessment.KnowledgeWithdrawn) {
		t.Fatal("withdrawal changed denominator or omitted cause", nodes, e)
	}
	got, e = f.repo.ReadLearningPath(f.ctx, f.Access("learner_a", false), joined.Summary.ID)
	if e != nil || got.Summary.TotalNodes != 3 || got.Summary.UnlockedNodes != 1 {
		t.Fatal(got, e)
	}
}
func TestLearningReadOldKnowledgeRequiresOwnedSummary(t *testing.T) {
	f := newLearningFixture(t)
	if _, e := f.repo.StartLearning(f.ctx, f.Access("learner_a", false), f.knowledge.ID, f.startInput()); e != nil {
		t.Fatal(e)
	}
	if _, e := f.repo.CompleteLearning(f.ctx, f.Access("learner_a", false), f.knowledge.ID, f.completeInput()); e != nil {
		t.Fatal(e)
	}
	v := newMaterial(f.Input())
	old := f.KHead()
	sub := f.ApprovedInput(v)
	f.Activate(f.Prepare(sub, old), old)
	got, e := f.repo.ReadLearningKnowledge(f.ctx, f.Access("learner_a", false), f.knowledge.ID, 1)
	if e != nil || got.State.Knowledge != f.knowledge || got.State.CompletedAt == nil || got.State.CompletionValid || got.State.CanEnter || len(got.Objectives) != 0 || len(got.Blueprints) != 0 {
		t.Fatal("old version exposed lecture/options", got, e)
	}
	if _, e = f.repo.ReadLearningKnowledge(f.ctx, f.Access("learner_b", false), f.knowledge.ID, 1); !errors.Is(e, auth.ErrNotFound) {
		t.Fatal("arbitrary private old knowledge", e)
	}
	current, e := f.repo.ReadLearningKnowledge(f.ctx, f.Access("learner_a", false), f.knowledge.ID, 2)
	if e != nil || current.State.StartedAt != nil || current.State.CompletedAt != nil || current.State.State != learning.Unlearned {
		t.Fatal("copied old learning to new version", current, e)
	}
}
func TestLearningReadBlueprintChoicesAndNoBlueprint(t *testing.T) {
	f := newLearningFixture(t)
	other := f.questionInput.QuestionPackage.Blueprints[0]
	other.ID = "lf-other"
	f.questionInput.QuestionPackage.Version = 2
	f.questionInput.QuestionPackage.Blueprints = append(f.questionInput.QuestionPackage.Blueprints, other)
	sub := f.QApproved("author_a", "reviewer_a")
	f.QActivate(f.QPrepare(sub.ID))
	d, e := f.repo.ReadLearningKnowledge(f.ctx, f.Access("learner_a", false), f.knowledge.ID, 1)
	if e != nil || len(d.Blueprints) != 2 || !d.Blueprints[0].Ready || !d.Blueprints[1].Ready {
		t.Fatal(d, e)
	}
	for _, bp := range d.Blueprints {
		f.QWithdraw(question.WithdrawalTarget{Kind: "blueprint", ID: bp.Blueprint.ID, Version: bp.Blueprint.Version})
	}
	d, e = f.repo.ReadLearningKnowledge(f.ctx, f.Access("learner_a", false), f.knowledge.ID, 1)
	if e != nil || len(d.Blueprints) != 0 {
		t.Fatal(d, e)
	}
	if _, e = f.repo.CreatePractice(f.ctx, f.Access("learner_a", false), f.practiceInput()); e != nil {
		t.Fatal("no blueprint removed safe practice", e)
	}
}

func questionToContentRef(id string, v int) content.VersionRef {
	return content.VersionRef{ID: id, Version: v}
}
func contentWithdrawalKnowledge(id string, v int) publication.WithdrawalTarget {
	return publication.WithdrawalTarget{Kind: "knowledge", ID: id, Version: v}
}
