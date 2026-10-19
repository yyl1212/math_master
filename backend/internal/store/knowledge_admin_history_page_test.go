package store_test

import (
	"github.com/yyl1212/math_master/backend/internal/pagination"
	"github.com/yyl1212/math_master/backend/internal/study"
	"github.com/yyl1212/math_master/backend/internal/studyreview"
	"testing"
)

func TestNumberedManagedHistory(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	completedReviewKnowledge(t, f)
	p, e := f.repo.ListManagedHistoryPage(f.ctx, f.Access("learner_a", "history-page"), study.HistoryQuery{}, pagination.Request{Enabled: true, Page: 99, Limit: 1})
	if e != nil || p.Total != 2 || p.Offset != 1 || len(p.Items) != 1 {
		t.Fatal(p, e)
	}
}
func TestNumberedReviewHistory(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := completedReviewKnowledge(t, f)
	r := startReview(t, f, k, studyreview.Read, studyreview.Statement, "numbered-start")
	c := compareReview(t, f, r, "", false)
	_, e := f.repo.GradeReviewSession(f.ctx, f.Access("learner_a", "numbered-grade"), r.SessionID, studyreview.GradeInput{ExpectedSequence: c.Sequence, Rating: studyreview.Familiar})
	if e != nil {
		t.Fatal(e)
	}
	p, e := f.repo.ListReviewHistoryPage(f.ctx, f.Access("learner_a", "numbered-history"), studyreview.HistoryQuery{}, pagination.Request{Enabled: true, Page: 1, Limit: 20})
	if e != nil || p.Total != 1 || len(p.Items) != 1 || p.AsOf.IsZero() {
		t.Fatal(p, e)
	}
}
