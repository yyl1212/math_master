package store_test

import (
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"testing"
)

func TestLearningHistoricalSourceActualPublication(t *testing.T) {
	f := newLearningFixture(t)
	seal := f.seal("assessment")
	items, e := f.repo.LearningLoadItemsForTest(f.ctx, f.Access("learner_a", false), seal)
	if e != nil || len(items) != 5 {
		t.Fatal(len(items), e)
	}
	for j, i := range items {
		if i.Identity != seal.Items[j].Instance {
			t.Fatal("historical identity lost", i.Identity)
		}
	}
	forged := seal
	forged.QuestionPublicationID = f.ID()
	if _, e = f.repo.LearningLoadItemsForTest(f.ctx, f.Access("learner_a", false), forged); e == nil {
		t.Fatal("forged publication accepted")
	}
	f.QWithdraw(question.WithdrawalTarget{Kind: "instance", ID: seal.Items[0].Instance.ID, Version: 1})
	items, e = f.repo.LearningLoadItemsForTest(f.ctx, f.Access("learner_a", false), seal)
	if e != nil || len(items) != 5 {
		t.Fatal("withdrawal removed immutable source", e)
	}
	v := 1
	deps := []learning.EvidenceDependency{{Kind: "instance", ID: seal.Items[0].Instance.ID, Version: &v, SHA256: seal.Items[0].Instance.SHA256}}
	rs, e := f.repo.LearningRestrictionsForTest(f.ctx, f.Access("learner_a", false), deps)
	if e != nil || len(rs) != 1 || rs[0] != assessment.InstanceWithdrawn {
		t.Fatal(rs, e)
	}
}
func TestLearningHistoricalSourceOrdinaryReplacement(t *testing.T) {
	f := newLearningFixture(t)
	seal := f.seal("assessment")
	f.questionInput.QuestionPackage.Version = 2
	f.questionInput.QuestionPackage.Templates[0].Version = 2
	f.questionInput.QuestionPackage.Blueprints[0].Version = 2
	f.questionInput.QuestionPackage.Blueprints[0].Sources[0].Ref.Version = 2
	s := f.QApproved("author_a", "reviewer_a")
	f.QActivate(f.QPrepare(s.ID))
	items, e := f.repo.LearningLoadItemsForTest(f.ctx, f.Access("learner_a", false), seal)
	if e != nil || len(items) != 5 {
		t.Fatal("ordinary replacement broke fixed historical questions", e)
	}
	v := 1
	rs, e := f.repo.LearningRestrictionsForTest(f.ctx, f.Access("learner_a", false), []learning.EvidenceDependency{{Kind: "template", ID: seal.Items[0].Template.ID, Version: &v, SHA256: seal.Items[0].Template.SHA256}})
	if e != nil || len(rs) != 0 {
		t.Fatal("ordinary replacement became permanent restriction", rs, e)
	}
}

func TestLearningHistoricalSourcePreparedRejected(t *testing.T) {
	f := newLearningFixture(t)
	seal := f.seal("assessment")
	f.questionInput.QuestionPackage.Version = 2
	sub := f.QApproved("author_a", "reviewer_a")
	prepared := f.QPrepare(sub.ID)
	seal.QuestionPublicationID = prepared.ID
	if _, e := f.repo.LearningLoadItemsForTest(f.ctx, f.Access("learner_a", false), seal); e == nil {
		t.Fatal("prepared publication treated as actual published source")
	}
}
