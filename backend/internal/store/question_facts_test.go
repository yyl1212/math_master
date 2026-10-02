package store_test

import (
	"bytes"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/question"
	"testing"
)

func TestQuestionHistoricalFactSeparation(t *testing.T) {
	f := newQuestionFixture(t)
	sub := f.QApproved("author_a", "reviewer_a")
	oldPub := f.QActivate(f.QPrepare(sub.ID))
	target := question.WithdrawalTarget{Kind: "template", ID: f.questionInput.QuestionPackage.Templates[0].ID, Version: 1}
	var oldInstance string
	var before []byte
	if err := f.db.QueryRow(`SELECT id,body_bytes FROM question_instances WHERE template_id=$1 ORDER BY id LIMIT 1`, target.ID).Scan(&oldInstance, &before); err != nil {
		t.Fatal(err)
	}
	offer, err := f.repo.TestQuestionOfferable(f.ctx, f.Access("reviewer_a", false), target)
	if err != nil || !offer {
		t.Fatal("approved current template unavailable", err)
	}
	raw, _ := json.Marshal(f.questionInput)
	var input question.DraftInput
	_ = json.Unmarshal(raw, &input)
	input.QuestionPackage.ID = "historical-replacement-bank"
	input.QuestionPackage.Templates[0].Version = 2
	input.QuestionPackage.Blueprints[0].Version = 2
	for n := range input.QuestionPackage.Blueprints[0].Sources {
		if input.QuestionPackage.Blueprints[0].Sources[n].Ref.ID == target.ID {
			input.QuestionPackage.Blueprints[0].Sources[n].Ref.Version = 2
		}
	}
	f.questionInput = input
	updated := f.QApproved("author_b", "reviewer_b")
	f.QActivate(f.QPrepare(updated.ID))
	offer, err = f.repo.TestQuestionOfferable(f.ctx, f.Access("reviewer_a", false), target)
	if err != nil || offer {
		t.Fatal("retired template still offered", err)
	}
	facts, err := f.repo.TestQuestionHistoricalFacts(f.ctx, f.Access("reviewer_a", false), target, &oldPub.ID)
	if err != nil || facts.Approval == nil || facts.Approval.SubmissionID != sub.ID || len(facts.Replacements) != 1 || len(facts.Withdrawals) != 0 {
		t.Fatal("replacement conflated with permanent withdrawal", facts, err)
	}
	result := f.QWithdraw(target) // A version outside the current head still exists and can be permanently withdrawn.
	if result.Publication.InstanceCount != 28 || result.Publication.TemplateCount != 3 {
		t.Fatal("withdrawing retired version removed its replacement")
	}
	facts, err = f.repo.TestQuestionHistoricalFacts(f.ctx, f.Access("reviewer_a", false), target, &oldPub.ID)
	if err != nil || len(facts.Withdrawals) != 1 || len(facts.Replacements) != 1 {
		t.Fatal("fixed withdrawal facts absent", facts, err)
	}
	instanceFacts, err := f.repo.TestQuestionHistoricalFacts(f.ctx, f.Access("reviewer_a", false), question.WithdrawalTarget{Kind: "instance", ID: oldInstance, Version: 1}, &oldPub.ID)
	if err != nil || instanceFacts.Approval == nil || len(instanceFacts.Replacements) != 1 || len(instanceFacts.Withdrawals) != 1 || instanceFacts.Withdrawals[0].Target.Kind != "template" {
		t.Fatal("historical generated instance lost parent withdrawal", instanceFacts, err)
	}
	var after []byte
	if err = f.db.QueryRow(`SELECT body_bytes FROM question_instances WHERE id=$1`, oldInstance).Scan(&after); err != nil || !bytes.Equal(before, after) {
		t.Fatal("historical instance bytes changed", err)
	}
}
