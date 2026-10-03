package store_test

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"strings"
	"testing"
)

func TestFeedbackCommands(t *testing.T) {
	f := newFeedbackFixture(t)
	a := f.Access("learner_a", false)
	in := f.feedbackInput()
	start := make(chan struct{})
	done := make(chan feedback.Envelope[feedback.Receipt], 2)
	errs := make(chan error, 2)
	for j := 0; j < 2; j++ {
		go func() { <-start; r, e := f.repo.CreateFeedback(f.ctx, a, in); done <- r; errs <- e }()
	}
	close(start)
	first, second := <-done, <-done
	for j := 0; j < 2; j++ {
		if e := <-errs; e != nil {
			t.Fatal(e)
		}
	}
	if first.Data.Status != 201 || first.Data.Ticket.ID != second.Data.Ticket.ID || f.count(`SELECT count(*) FROM feedback_events`) != 1 || f.count(`SELECT count(*) FROM feedback_rate_limits`) != 1 {
		t.Fatal("duplicate success")
	}
	raw := feedbackJSON(first)
	if strings.Contains(raw, "answer-sentinel") || strings.Contains(raw, in.Message) || strings.Contains(raw, in.Location) {
		t.Fatal("receipt text")
	}
	id := first.Data.Ticket.ID
	if _, e := f.repo.ReplyFeedback(f.ctx, f.Access("learner_b", false), id, feedback.ReplyInput{ExpectedSequence: 1, Message: "foreign"}); !errors.Is(e, auth.ErrNotFound) {
		t.Fatal(e)
	}
	if _, e := f.repo.ReplyFeedback(f.ctx, f.Access("learner_a", false), id, feedback.ReplyInput{ExpectedSequence: 2, Message: "stale"}); !errors.Is(e, feedback.ErrConflict) {
		t.Fatal(e)
	}
	if f.count(`SELECT count(*) FROM feedback_rate_limits`) != 1 {
		t.Fatal("failure consumed")
	}
}
func TestFeedbackRatesCommands(t *testing.T) {
	f := newFeedbackFixture(t)
	var id string
	for n := 0; n < 5; n++ {
		r, e := f.repo.CreateFeedback(f.ctx, f.Access("learner_a", false), f.feedbackInput())
		if e != nil {
			t.Fatal(n, e)
		}
		id = r.Data.Ticket.ID
	}
	_, e := f.repo.CreateFeedback(f.ctx, f.Access("learner_a", false), f.feedbackInput())
	var rate *feedback.RateError
	if !errors.As(e, &rate) || rate.RetryAt.IsZero() {
		t.Fatal("sixth create", e)
	}
	for n := 0; n < 30; n++ {
		_, e = f.repo.ReplyFeedback(f.ctx, f.Access("learner_a", false), id, feedback.ReplyInput{ExpectedSequence: int64(n + 1), Message: "More details"})
		if e != nil {
			t.Fatal(n, e)
		}
	}
	_, e = f.repo.ReplyFeedback(f.ctx, f.Access("learner_a", false), id, feedback.ReplyInput{ExpectedSequence: 31, Message: "More details"})
	if !errors.As(e, &rate) {
		t.Fatal("31st reply", e)
	}
	if f.count(`SELECT count(*) FROM feedback_events WHERE ticket_id=$1`, id) != 31 {
		t.Fatal("failure appended")
	}
}
