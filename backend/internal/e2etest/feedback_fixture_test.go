package e2etest

import (
	"context"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"net/http"
	"testing"
	"time"
)

func TestFeedbackRealFixtureAndReset(t *testing.T) {
	db := testutil.Database(t)
	ctx, stop := context.WithTimeout(context.Background(), 40*time.Second)
	defer stop()
	root, e := rootDir()
	if e != nil {
		t.Fatal(e)
	}
	if e = store.Up(ctx, db, root+"/db/migrations"); e != nil {
		t.Fatal(e)
	}
	s := store.New(db)
	a, admin, e := fixtureAccounts(s)
	if e != nil {
		t.Fatal(e)
	}
	normal, e := loadFixture(root, false)
	if e != nil {
		t.Fatal(e)
	}
	if e = resetLearning(ctx, db, s, a, admin, root, normal, LearningBasic); e != nil {
		t.Fatal(e)
	}
	f, e := setupFeedback(ctx, s)
	if e != nil {
		t.Fatal(e)
	}
	owner, e := fixtureAccess(ctx, a, "learning_other", false)
	if e != nil {
		t.Fatal(e)
	}
	v, e := s.ReadFeedbackTicket(ctx, owner, f.TicketID, false)
	if e != nil || v.Data.Target.Identity.ID != f.Instance.Target.Identity.ID {
		t.Fatal(v, e)
	}
	reviewer, e := fixtureAccess(ctx, a, "content_reviewer", false)
	if e != nil {
		t.Fatal(e)
	}
	page, e := s.ListFeedbackTickets(ctx, reviewer, true, feedback.ListQuery{})
	if e != nil || len(page.Data.Items) != 1 {
		t.Fatal(page, e)
	}

	handler, e := fixtureAccess(ctx, a, "feedback_handler", false)
	if e != nil {
		t.Fatal("fresh unexposed handler", e)
	}
	detail, e := s.ReadLearningKnowledge(ctx, handler, "learning-root", 1)
	if e != nil || !detail.Blueprints[0].Ready {
		t.Fatal("fresh handler eligibility", e)
	}
	attempt, e := s.CreateAssessment(ctx, handler, assessment.CreateInput{Knowledge: detail.State.Knowledge, Blueprint: detail.Blueprints[0].Blueprint, Mode: assessment.ModeDiagnostic, ExpectedKnowledgeHead: detail.KnowledgeHead, ExpectedQuestionHead: *detail.QuestionHead})
	if e != nil {
		t.Fatal(e)
	}
	if out, err := s.ReadFeedbackEvents(ctx, handler, f.TicketID, true, feedback.ListQuery{}); !errors.Is(err, feedback.ErrAnswerOverlap) || out.Data.Title != "" {
		t.Fatal("active handling exposure", err)
	}
	handler, _ = nextFixtureAccess(handler)
	if _, e = s.AbandonAssessment(ctx, handler, attempt.Summary.ID); e != nil {
		t.Fatal(e)
	}
	for _, scene := range []string{"feedback-withdraw-instance", "feedback-new-instance"} {
		if handled, err := feedbackChange(ctx, db, s, a, root, scene); !handled || err != nil {
			t.Fatal(scene, handled, err)
		}
	}
	state, e := feedbackState(ctx, db)
	if e != nil || state.WithdrawalID == nil || state.Replacement == nil || state.Replacement.Identity.ID == f.Instance.Target.Identity.ID {
		t.Fatal("actual withdrawal/new identity", state, e)
	}
	original, e := s.ReadFeedbackTicket(ctx, owner, f.TicketID, false)
	if e != nil || original.Data.Target.Identity.ID != f.Instance.Target.Identity.ID || original.Data.TargetValidity != "withdrawn" {
		t.Fatal("original binding", original, e)
	}
	if e = resetAccounts(ctx, db, a, admin); e != nil {
		t.Fatal(e)
	}
	var n int
	if e = db.QueryRowContext(ctx, "SELECT count(*) FROM feedback_tickets").Scan(&n); e != nil || n != 0 {
		t.Fatal(n, e)
	}
}
func TestFeedbackHarnessControlsPrivate(t *testing.T) {
	s, _, _, _ := startHarness(t)
	client := &http.Client{Timeout: 40 * time.Second}
	for _, route := range []string{s.ControlURL + "/feedback/state", s.APIURL + "/feedback/state"} {
		r, e := client.Get(route)
		if e != nil {
			t.Fatal(e)
		}
		r.Body.Close()
		if r.StatusCode != 401 && r.StatusCode != 404 {
			t.Fatal(r.StatusCode)
		}
	}
	req, _ := http.NewRequest("POST", s.ControlURL+"/scene/feedback", nil)
	req.Header.Set("Authorization", "Bearer "+s.Token)
	r, e := client.Do(req)
	if e != nil {
		t.Fatal(e)
	}
	r.Body.Close()
	if r.StatusCode != 204 {
		t.Fatal("feedback scene", r.StatusCode)
	}
}
