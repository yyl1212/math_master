package store_test

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"strings"
	"testing"
)

func TestPublishCreateSelectionAndAllSnapshot(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	f.Import("admin_a", "seed", f.doc, false)
	a := f.Access("admin_a", "all-once")
	job, e := f.repo.CreatePublishJob(f.ctx, a, knowledgeadmin.PublishCreateInput{Mode: knowledgeadmin.PublishAll})
	if e != nil || job.Total != 2 || job.Pending != 2 || job.Sequence != 1 {
		t.Fatal(job, e)
	}
	f.Import("admin_a", "seed-again", f.doc, true)
	again, e := f.repo.CreatePublishJob(f.ctx, a, knowledgeadmin.PublishCreateInput{Mode: knowledgeadmin.PublishAll})
	if e != nil || again != job {
		t.Fatal(again, e)
	}
	detail, e := f.repo.ReadPublishJob(f.ctx, f.Access("admin_a", "read"), job.ID, knowledgeadmin.PublishItemQuery{})
	if e != nil || len(detail.Items) != 2 || detail.Job.Pending != 2 {
		t.Fatal(detail, e)
	}
	missing := knowledgeadmin.PublishSelection{ID: "k-" + strings.Repeat("f", 56), EditToken: "11111111-1111-4111-8111-111111111111"}
	selected, e := f.repo.CreatePublishJob(f.ctx, f.Access("admin_a", "selected"), knowledgeadmin.PublishCreateInput{Mode: knowledgeadmin.PublishSelected, Items: []knowledgeadmin.PublishSelection{missing}})
	if e != nil || selected.Failed != 1 || selected.Pending != 0 {
		t.Fatal(selected, e)
	}
}
func TestPublishCreateReplayAndOwner(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	f.Import("admin_a", "seed", f.doc, false)
	a := f.Access("admin_a", "once")
	j, e := f.repo.CreatePublishJob(f.ctx, a, knowledgeadmin.PublishCreateInput{Mode: knowledgeadmin.PublishAll})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = f.repo.ReadPublishJob(f.ctx, f.Access("admin_b", "read"), j.ID, knowledgeadmin.PublishItemQuery{}); !errors.Is(e, knowledgeadmin.ErrNotFound) {
		t.Fatal("owner leak", e)
	}
	if _, e = f.repo.CreatePublishJob(f.ctx, a, knowledgeadmin.PublishCreateInput{Mode: knowledgeadmin.PublishSelected, Items: []knowledgeadmin.PublishSelection{{ID: "k-" + strings.Repeat("f", 56), EditToken: "11111111-1111-4111-8111-111111111111"}}}); !errors.Is(e, knowledgeadmin.ErrIdempotency) {
		t.Fatal(e)
	}
	p, e := f.repo.ListPublishJobs(f.ctx, f.Access("admin_a", "lookup"), knowledgeadmin.PublishJobQuery{CreateKey: a.IdempotencyKey})
	if e != nil || p.Total != 1 || p.Items[0].ID != j.ID {
		t.Fatal(p, e)
	}
	f.exec("DELETE FROM auth_user_roles WHERE user_id=$1 AND role='admin'", f.ids["admin_a"])
	if _, e = f.repo.ReadPublishJob(f.ctx, a, j.ID, knowledgeadmin.PublishItemQuery{}); e == nil {
		t.Fatal("revoked admin read")
	}
}

func TestPublishCreateInterruptedResponseLookup(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	f.Import("admin_a", "seed", f.doc, false)
	a := f.Access("admin_a", "lost-create")
	j, e := f.repo.CreatePublishJob(f.ctx, a, knowledgeadmin.PublishCreateInput{Mode: knowledgeadmin.PublishAll})
	if e != nil {
		t.Fatal(e)
	}
	f.exec(`INSERT INTO managed_knowledge(internal_id,external_id,point,public_sources,content_sha256,created_by,updated_by) SELECT 'k-'||repeat('f',56),'later-point',jsonb_set(point,'{id}','"later-point"'),public_sources,content_sha256,created_by,updated_by FROM managed_knowledge LIMIT 1`)
	p, e := f.repo.ListPublishJobs(f.ctx, a, knowledgeadmin.PublishJobQuery{CreateKey: a.IdempotencyKey})
	if e != nil || p.Total != 1 || p.Items[0].ID != j.ID || p.Items[0].Total != 2 {
		t.Fatal(p, e)
	}
	d, e := f.repo.ReadPublishJob(f.ctx, a, j.ID, knowledgeadmin.PublishItemQuery{})
	if e != nil || d.Total != 2 {
		t.Fatal(d, e)
	}
	for _, v := range d.Items {
		if v.ExternalID == "later-point" {
			t.Fatal("snapshot absorbed new knowledge")
		}
	}
}
