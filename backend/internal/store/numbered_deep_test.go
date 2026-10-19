package store_test

import (
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/pagination"
	"github.com/yyl1212/math_master/backend/internal/studyreview"
	"testing"
)

func TestNumberedDeepOffsetKeepsLegacyLimit(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	f.PreparedManagedKnowledge()
	p := pagination.Request{Enabled: true, Page: 50002, Limit: 20}
	v, e := f.repo.ListManagedKnowledge(f.ctx, f.Access("admin_a", "deep"), knowledgeadmin.Query{Numbered: &p, Limit: 20})
	if e != nil || v.Offset != 0 || len(v.Items) != 2 {
		t.Fatal(v, e)
	}
	if _, e = f.repo.ListManagedKnowledge(f.ctx, f.Access("admin_a", "old"), knowledgeadmin.Query{Offset: 1000020, Limit: 20}); e == nil {
		t.Fatal("old offset cap removed")
	}
	q := pagination.Request{Enabled: true, Page: 5002, Limit: 20}
	r, e := f.repo.ListReviewQueue(f.ctx, f.Access("learner_a", "review-deep"), studyreview.QueueQuery{Numbered: &q, Limit: 20, Bucket: "all"})
	if e != nil || r.Offset != 0 {
		t.Fatal(r, e)
	}
}
