package store_test

import (
	"errors"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/question"
	"testing"
)

func TestQuestionReviewFinality(t *testing.T) {
	f := newQuestionFixture(t)
	sub := f.QSubmitted("author_a")
	for field := 0; field < 6; field++ {
		input := approvedQuestionInput()
		checks := []*bool{&input.Checks.Mathematics, &input.Checks.Explanations, &input.Checks.Objectives, &input.Checks.Sources, &input.Checks.Illustrations, &input.Checks.Generation}
		*checks[field] = false
		if _, err := f.repo.DecideQuestionReview(f.ctx, f.Access("reviewer_a", false), sub.ID, input); !errors.Is(err, question.ErrNotReady) {
			t.Fatal("missing review check accepted", field, err)
		}
	}
	done := make(chan error, 2)
	for _, actor := range []string{"reviewer_a", "reviewer_b"} {
		a := f.Access(actor, false)
		go func() { _, err := f.repo.DecideQuestionReview(f.ctx, a, sub.ID, approvedQuestionInput()); done <- err }()
	}
	success, conflict := 0, 0
	for i := 0; i < 2; i++ {
		err := <-done
		if err == nil {
			success++
		} else if errors.Is(err, question.ErrReviewConflict) {
			conflict++
		} else {
			t.Fatal(err)
		}
	}
	if success != 1 || conflict != 1 || f.count(`SELECT count(*) FROM question_review_decisions WHERE submission_id=$1`, sub.ID) != 1 {
		t.Fatal("not one final decision")
	}
	revision, err := f.repo.ReviseQuestionSubmission(f.ctx, f.Access("author_a", false), sub.ID)
	if err != nil || revision.ID == sub.WorkspaceID || revision.Status != "editing" || revision.Revision != 1 || revision.Gate.ReadyToSubmit {
		t.Fatal("approved revision inherited approval", err)
	}
	returned := f.QSubmitted("author_b")
	if _, err = f.repo.DecideQuestionReview(f.ctx, f.Access("reviewer_a", false), returned.ID, question.ReviewInput{Decision: "return", Note: "Refine this complete technical fixture before re-review."}); err != nil {
		t.Fatal(err)
	}
	draft, err := f.repo.ReadQuestionDraft(f.ctx, f.Access("author_b", false), returned.WorkspaceID)
	if err != nil || draft.Revision != 2 || draft.Status != "editing" {
		t.Fatal("return did not reopen revision", err)
	}
}

func TestQuestionReviewNoTemplatesAndAtomicity(t *testing.T) {
	f := newQuestionFixture(t)
	input := f.questionInput
	template := input.QuestionPackage.Templates[0]
	instances, _, err := question.Generate(f.ctx, template)
	if err != nil {
		t.Fatal(err)
	}
	input.QuestionPackage.ID = "fixed-review-bank"
	input.QuestionPackage.Templates = []question.Template{}
	input.QuestionPackage.FixedQuestions = []question.FixedQuestion{}
	input.QuestionPackage.Blueprints = input.QuestionPackage.Blueprints[:1]
	input.QuestionPackage.Blueprints[0].Sources = []question.BlueprintSource{}
	for n, i := range instances[:5] {
		id := fmt.Sprintf("fixed-review-%d", n)
		b := i.Body
		b.Witness = &question.VerificationWitness{Engine: template.Engine, Parameters: i.Parameters}
		input.QuestionPackage.FixedQuestions = append(input.QuestionPackage.FixedQuestions, question.FixedQuestion{ID: id, Version: 1, Body: b})
		input.QuestionPackage.Blueprints[0].Sources = append(input.QuestionPackage.Blueprints[0].Sources, question.BlueprintSource{Kind: "instance", Ref: question.Ref{ID: id, Version: 1}})
	}
	input.QuestionPackage.Blueprints[0].ID = "fixed-review-blueprint"
	d, err := f.repo.CreateQuestionDraft(f.ctx, f.Access("author_a", false), input)
	if err != nil {
		t.Fatal(err)
	}
	gate, err := f.repo.ValidateQuestionDraft(f.ctx, f.Access("author_a", false), d.ID, question.ValidateInput{ExpectedRevision: 1})
	if err != nil || !gate.ReadyToSubmit {
		t.Fatal("fixed bank incomplete", gate, err)
	}
	a := f.Access("author_a", false)
	command := question.SubmitInput{ExpectedRevision: 1, ExpectedDigest: gate.Digest}
	sub, err := f.repo.SubmitQuestionDraft(f.ctx, a, d.ID, command)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := f.repo.SubmitQuestionDraft(f.ctx, a, d.ID, command)
	if err != nil || replay.ID != sub.ID {
		t.Fatal("submit replay failed", err)
	}
	if len(sub.Frozen.GeneratorVersions) != 0 || len(sub.Frozen.VerifierVersions) != 1 {
		t.Fatal("invented engine provenance")
	}
	review := approvedQuestionInput()
	review.GenerationNote = ""
	if _, err = f.repo.DecideQuestionReview(f.ctx, f.Access("reviewer_a", false), sub.ID, review); !errors.Is(err, question.ErrNotReady) {
		t.Fatal("no-template approval lacked reason", err)
	}
	f.exec(`CREATE FUNCTION question_test_audit_failure() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'fixture audit failure'; END $$`)
	f.exec(`CREATE TRIGGER question_test_audit_failure BEFORE INSERT ON question_events FOR EACH ROW EXECUTE FUNCTION question_test_audit_failure()`)
	a = f.Access("reviewer_a", false)
	if _, err = f.repo.DecideQuestionReview(f.ctx, a, sub.ID, question.ReviewInput{Decision: "return", Note: "This return must be fully rolled back if auditing fails."}); !errors.Is(err, auth.ErrUnavailable) {
		t.Fatal("audit failure not returned", err)
	}
	if f.count(`SELECT count(*) FROM question_review_decisions WHERE submission_id=$1`, sub.ID) != 0 || f.count(`SELECT count(*) FROM question_idempotency WHERE actor_user_id=$1 AND route='decideReview'`, f.ids["reviewer_a"]) != 0 || f.count(`SELECT count(*) FROM question_submissions WHERE id=$1 AND status='pending'`, sub.ID) != 1 || f.count(`SELECT count(*) FROM question_workspaces WHERE id=$1 AND status='submitted' AND revision=1`, d.ID) != 1 {
		t.Fatal("audit failure leaked changes")
	}
	f.exec(`DROP TRIGGER question_test_audit_failure ON question_events`)
	if _, err = f.repo.DecideQuestionReview(f.ctx, f.Access("reviewer_a", false), sub.ID, approvedQuestionInput()); err != nil {
		t.Fatal(err)
	}
}
