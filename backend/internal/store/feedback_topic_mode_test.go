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

func TestFeedbackTopicModeMixedListsPagePastRestrictedLegacy(t *testing.T) {
	for _, missing := range []string{"question_heads", "learning_records"} {
		t.Run(missing, func(t *testing.T) {
			f := newFeedbackFixture(t)
			knowledge, e := f.repo.CreateFeedback(f.ctx, f.Access("learner_a", false), f.feedbackInput())
			if e != nil {
				t.Fatal(e)
			}
			legacy := f.instanceFeedback()
			context, e := f.repo.ReadFeedbackContext(f.ctx, f.Access("learner_a", false), feedback.ContextQuery{Kind: "site", Area: "other"})
			if e != nil {
				t.Fatal(e)
			}
			in := f.feedbackInput()
			in.Target = context.Data.Target
			in.Source = context.Data.Source
			in.Category = "technical_issue"
			site, e := f.repo.CreateFeedback(f.ctx, f.Access("learner_a", false), in)
			if e != nil {
				t.Fatal(e)
			}
			f.exec(`UPDATE topic_learning_state SET experience_mode='topics' WHERE singleton`)
			f.exec("ALTER TABLE " + missing + " RENAME TO restricted_legacy_table")
			for _, review := range []bool{false, true} {
				actor := "learner_a"
				if review {
					actor = "reviewer_a"
				}
				first, e := f.repo.ListFeedbackTickets(f.ctx, f.Access(actor, false), review, feedback.ListQuery{Limit: 1})
				if e != nil || len(first.Data.Items) != 1 || first.Data.Items[0].ID != site.Data.Ticket.ID || first.Data.NextCursor == nil {
					t.Fatal("legacy entry blocked site page", review, e)
				}
				next, e := f.repo.ListFeedbackTickets(f.ctx, f.Access(actor, false), review, feedback.ListQuery{Limit: 1, Cursor: *first.Data.NextCursor})
				if e != nil || len(next.Data.Items) != 1 || next.Data.Items[0].ID != knowledge.Data.Ticket.ID || next.Data.NextCursor != nil {
					t.Fatal("filtered legacy entry broke knowledge cursor", review, e)
				}
				discussion, e := f.repo.ReadFeedbackEvents(f.ctx, f.Access(actor, false), legacy.ID, review, feedback.ListQuery{})
				if !errors.Is(e, feedback.ErrNotConfigured) || discussion.Data.Title != "" || len(discussion.Data.Items) != 0 {
					t.Fatal("restricted legacy discussion exposed", review, e)
				}
			}
			other, e := f.repo.ListFeedbackTickets(f.ctx, f.Access("learner_b", false), false, feedback.ListQuery{})
			if e != nil || len(other.Data.Items) != 0 {
				t.Fatal("mixed list crossed owner", e)
			}
		})
	}
}
