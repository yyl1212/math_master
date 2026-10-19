package store_test

import (
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"github.com/yyl1212/math_master/backend/internal/notification"
	"github.com/yyl1212/math_master/backend/internal/pagination"
	"testing"
)

func TestNumberedFeedbackFilters(t *testing.T) {
	f := newFeedbackFixture(t)
	in := f.feedbackInput()
	f.feedbackCreate("learner_a", in)
	f.feedbackCreate("learner_b", in)
	p, e := f.repo.ListFeedbackTicketPage(f.ctx, f.Access("learner_a", false), false, feedback.ListQuery{}, pagination.Request{Enabled: true, Page: 1, Limit: 20})
	if e != nil || p.Data.Total != 1 || len(p.Data.Items) != 1 {
		t.Fatal(p, e)
	}
}
func TestNumberedNotificationOwner(t *testing.T) {
	f := newCorrectionFixture(t)
	f.cReadableResult()
	old, e := f.repo.ListNotifications(f.ctx, f.Access("learner_a", false), notification.Query{})
	if e != nil {
		t.Fatal(e)
	}
	p, e := f.repo.ListNotificationPage(f.ctx, f.Access("learner_a", false), notification.Query{}, pagination.Request{Enabled: true, Page: 99, Limit: 1})
	if e != nil || p.Data.Total != int64(len(old.Data.Items)) || len(p.Data.Items) != 1 {
		t.Fatal(p, e)
	}
	other, e := f.repo.ListNotificationPage(f.ctx, f.Access("learner_b", false), notification.Query{}, pagination.Request{Enabled: true, Page: 1, Limit: 20})
	if e != nil || other.Data.Total != 0 {
		t.Fatal(other, e)
	}
}
