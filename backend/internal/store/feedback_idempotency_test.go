package store_test

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"github.com/yyl1212/math_master/backend/internal/question"
	"testing"
)

func TestFeedbackReplayAfterAdvance(t *testing.T) {
	f := newFeedbackFixture(t)
	a := f.Access("learner_a", false)
	in := f.feedbackInput()
	r, e := f.repo.CreateFeedback(f.ctx, a, in)
	if e != nil {
		t.Fatal(e)
	}
	id := r.Data.Ticket.ID
	key := f.Access("learner_a", false)
	reply := feedback.ReplyInput{ExpectedSequence: 1, Message: "supplement"}
	old, e := f.repo.ReplyFeedback(f.ctx, key, id, reply)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = f.repo.ReplyFeedback(f.ctx, f.Access("learner_a", false), id, feedback.ReplyInput{ExpectedSequence: 2, Message: "further"}); e != nil {
		t.Fatal(e)
	}
	// Advance the real publication head; original create replay still succeeds.
	f.publishLearningGraph()
	again, e := f.repo.CreateFeedback(f.ctx, a, in)
	if e != nil || feedbackJSON(again) != feedbackJSON(r) {
		t.Fatal("original create", again, e)
	}
	again, e = f.repo.ReplyFeedback(f.ctx, key, id, reply)
	if e != nil || feedbackJSON(again) != feedbackJSON(old) || again.Data.Ticket.Sequence != 2 {
		t.Fatal("original reply", e)
	}
	if f.count(`SELECT sequence FROM feedback_tickets WHERE id=$1`, id) != 3 || f.count(`SELECT count(*) FROM feedback_rate_limits`) != 3 {
		t.Fatal("replay mutated")
	}
	in.Title = "changed input"
	if _, e = f.repo.CreateFeedback(f.ctx, a, in); !errors.Is(e, question.ErrIdempotencyConflict) {
		t.Fatal(e)
	}
	f.exec(`UPDATE auth_sessions SET revoked_at=clock_timestamp() WHERE token_hash=$1`, a.TokenHash[:])
	if _, e = f.repo.CreateFeedback(f.ctx, a, f.feedbackInput()); !errors.Is(e, auth.ErrAuthenticationRequired) {
		t.Fatal("replay auth", e)
	}
}
