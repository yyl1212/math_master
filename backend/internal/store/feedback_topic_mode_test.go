package store_test

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"testing"
	"time"
)

func TestFeedbackTopicModeWithoutQuestionBank(t *testing.T) {
	f := newFeedbackFixture(t)
	f.exec(`UPDATE topic_learning_state SET experience_mode='topics' WHERE singleton`)
	f.exec(`ALTER TABLE question_heads RENAME TO retired_question_heads`)
	f.exec(`ALTER TABLE question_withdrawals RENAME TO retired_question_withdrawals`)
	f.exec(`ALTER TABLE learning_records RENAME TO retired_learning_records`)
	for _, kind := range []string{"site", "knowledge"} {
		q := feedback.ContextQuery{Kind: kind, ID: f.knowledge.ID}
		if kind == "site" {
			q.ID = ""
			q.Area = "other"
		}
		ctx, e := f.repo.ReadFeedbackContext(f.ctx, f.Access("learner_a", false), q)
		if e != nil {
			t.Fatalf("%s context needs retired modules: %v", kind, e)
		}
		in := f.feedbackInput()
		if kind == "site" {
			in.Category = "technical_issue"
		}
		in.Target = ctx.Data.Target
		in.Source = ctx.Data.Source
		r, e := f.repo.CreateFeedback(f.ctx, f.Access("learner_a", false), in)
		if e != nil {
			t.Fatalf("%s create needs retired modules: %v", kind, e)
		}
		page, e := f.repo.ReadFeedbackEvents(f.ctx, f.Access("learner_a", false), r.Data.Ticket.ID, false, feedback.ListQuery{})
		if e != nil || len(page.Data.Items) != 1 || page.Data.Items[0].Message != in.Message {
			t.Fatal("discussion unavailable", e)
		}
		if kind == "knowledge" && (*ctx.Data.Target.Identity != f.knowledge || *ctx.Data.Source.PublicationID != *f.KHead()) {
			t.Fatal("exact knowledge binding lost")
		}
	}
}
func TestFeedbackTopicModeLegacyOverlap(t *testing.T) {
	f := newFeedbackFixture(t)
	ticket := f.instanceFeedback()
	f.timedAssessment(5 * time.Minute)
	f.exec(`UPDATE topic_learning_state SET experience_mode='topics' WHERE singleton`)
	out, e := f.repo.ReadFeedbackEvents(f.ctx, f.Access("learner_a", false), ticket.ID, false, feedback.ListQuery{})
	if !errors.Is(e, feedback.ErrAnswerOverlap) || out.Data.Title != "" || len(out.Data.Items) != 0 {
		t.Fatal("legacy overlap bypassed", e)
	}
	f.exec(`ALTER TABLE question_heads RENAME TO unavailable_question_heads`)
	out, e = f.repo.ReadFeedbackEvents(f.ctx, f.Access("learner_a", false), ticket.ID, false, feedback.ListQuery{})
	if !errors.Is(e, feedback.ErrNotConfigured) || out.Data.Title != "" || len(out.Data.Items) != 0 {
		t.Fatal("missing legacy protection delivered discussion", e)
	}
}
func TestFeedbackTopicModeOwnerCannotHandle(t *testing.T) {
	f := newFeedbackFixture(t)
	f.exec(`ALTER TABLE question_heads RENAME TO retired_question_heads`)
	r, e := f.repo.CreateFeedback(f.ctx, f.Access("reviewer_a", false), f.feedbackInput())
	if e != nil {
		t.Fatal(e)
	}
	_, e = f.repo.TransitionFeedback(f.ctx, f.Access("reviewer_a", false), r.Data.Ticket.ID, feedback.TransitionInput{ExpectedSequence: 1, Status: feedback.Processing, Message: "Own report"})
	if !errors.Is(e, auth.ErrForbidden) {
		t.Fatal("self handling allowed", e)
	}
}
