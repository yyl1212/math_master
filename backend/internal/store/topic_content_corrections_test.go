package store_test

import (
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"testing"
)

func TestTopicContentDraftWithoutRoutesCanBeReviewed(t *testing.T) {
	f := newTopicWorkflowFixture(t)
	f.input.Package.Paths = []content.Path{}
	sub := f.topicApproved("msc-00a00")
	if sub.Status != "approved" || len(sub.Frozen.Package.Paths) != 0 {
		t.Fatal("empty routes blocked normal review")
	}
}
func TestTopicContentResolvedFeedbackDoesNotPublish(t *testing.T) {
	f := newFeedbackFixture(t)
	before := *f.KHead()
	ticket := f.feedbackCreate("learner_a", f.feedbackInput())
	if _, e := f.repo.TransitionFeedback(f.ctx, f.Access("reviewer_a", false), ticket.ID, feedback.TransitionInput{ExpectedSequence: 1, Status: feedback.Processing, Message: "Original independent handling."}); e != nil {
		t.Fatal(e)
	}
	if _, e := f.repo.TransitionFeedback(f.ctx, f.Access("reviewer_a", false), ticket.ID, feedbackResolutionOnlyInput()); e != nil {
		t.Fatal(e)
	}
	if *f.KHead() != before {
		t.Fatal("resolution published content")
	}
}

// A plain clarification resolution has no publication authority.
func feedbackResolutionOnlyInput() feedback.TransitionInput {
	return feedback.TransitionInput{ExpectedSequence: 2, Status: feedback.Resolved, Message: "Original clarification without mathematical replacement.", Resolution: &feedback.Resolution{Kind: "clarified"}}
}
