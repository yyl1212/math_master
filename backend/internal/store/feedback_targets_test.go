package store_test

import (
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"github.com/yyl1212/math_master/backend/internal/question"
	"strings"
	"testing"
)

func TestFeedbackTargetsContexts(t *testing.T) {
	f := newFeedbackFixture(t)
	p := f.createPractice("learner_a")
	aid := f.ID()
	tx, _ := f.db.Begin()
	if e := f.insertAssessment(tx, aid, f.ids["learner_a"], 5); e != nil {
		t.Fatal(e)
	}
	if e := tx.Commit(); e != nil {
		t.Fatal(e)
	}
	asset := f.Input().Package.Assets[0].ID
	for _, q := range []feedback.ContextQuery{{Kind: "site", Area: "home"}, {Kind: "knowledge", ID: f.knowledge.ID}, {Kind: "knowledge", ID: f.knowledge.ID, PartKind: "unit", PartID: "workflow-fractions-unit"}, {Kind: "knowledge", ID: f.knowledge.ID, PartKind: "asset", PartID: asset}, {Kind: "practice", ID: p.Summary.ID}, {Kind: "assessment", ID: aid, Position: 5}} {
		got, e := f.repo.ReadFeedbackContext(f.ctx, f.Access("learner_a", false), q)
		if e != nil {
			t.Fatal(q, e)
		}
		if got.ActorID != f.ids["learner_a"] || got.Data.Label == "" {
			t.Fatal(got)
		}
		if q.Kind == "assessment" && (got.Data.Target.Identity.ID != f.items[4].Instance.ID || *got.Data.Source.Position != 5) {
			t.Fatal("original fifth item", got)
		}
		if q.Kind == "practice" && got.Data.Target.Identity.ID != p.Question.Instance.ID {
			t.Fatal("original practice")
		}
		raw, _ := json.Marshal(got)
		for _, bad := range []string{"correctNumeric", "approval", "template", "Original mathematical"} {
			if strings.Contains(string(raw), bad) {
				t.Fatal("context leak", bad)
			}
		}
	}
	for _, q := range []feedback.ContextQuery{{Kind: "assessment", ID: aid, Position: 5}, {Kind: "practice", ID: p.Summary.ID}} {
		if _, e := f.repo.ReadFeedbackContext(f.ctx, f.Access("learner_b", false), q); !errors.Is(e, auth.ErrNotFound) {
			t.Fatal("other owner", e)
		}
	}
	if _, e := f.repo.ReadFeedbackContext(f.ctx, f.Access("learner_a", false), feedback.ContextQuery{Kind: "knowledge", ID: f.knowledge.ID, PartKind: "unit", PartID: "fake-unit"}); e == nil {
		t.Fatal("fake part")
	}
	f.QWithdraw(question.WithdrawalTarget{Kind: "instance", ID: p.Question.Instance.ID, Version: 1})
	got, e := f.repo.ReadFeedbackContext(f.ctx, f.Access("learner_a", false), feedback.ContextQuery{Kind: "practice", ID: p.Summary.ID})
	if e != nil || got.Data.Target.Identity != nil && *got.Data.Target.Identity != p.Question.Instance {
		t.Fatal("historical instance", e)
	}
	pathFixture := newFeedbackFixture(t)
	path := pathFixture.publishLearningGraph()
	got, e = pathFixture.repo.ReadFeedbackContext(pathFixture.ctx, pathFixture.Access("learner_a", false), feedback.ContextQuery{Kind: "path", ID: path.ID})
	if e != nil || *got.Data.Target.Identity != path {
		t.Fatal("path", e)
	}
	for _, actor := range []string{"author_a", "reviewer_a", "admin_a", "learner_a"} {
		if _, e = f.repo.FeedbackPreflight(f.ctx, f.Access(actor, false), feedback.ReadContextAction); e != nil {
			t.Fatal(actor, e)
		}
	}
	if _, e = f.repo.FeedbackPreflight(f.ctx, f.Access("author_a", false), feedback.ListReviewAction); !errors.Is(e, auth.ErrForbidden) {
		t.Fatal("editor review", e)
	}
}
