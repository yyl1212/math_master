package store_test

import (
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"testing"
)

func TestPublishStepMixedResultsAndOriginalAPI(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	f.Import("admin_a", "seed", f.doc, false)
	a := f.Access("admin_a", "job")
	j, e := f.repo.CreatePublishJob(f.ctx, a, knowledgeadmin.PublishCreateInput{Mode: knowledgeadmin.PublishAll})
	if e != nil {
		t.Fatal(e)
	}
	id := knowledgeadmin.KnowledgeID(f.doc.KnowledgePoints[0].ID)
	k, e := f.repo.ReadManagedKnowledge(f.ctx, a, id)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = f.repo.SetManagedKnowledgeState(f.ctx, f.Access("admin_a", "single"), id, k.EditToken, "publish"); e != nil {
		t.Fatal(e)
	}
	r, e := f.repo.StepPublishJob(f.ctx, f.Access("admin_a", "step"), j.ID, knowledgeadmin.PublishStepInput{ExpectedSequence: 1})
	if e != nil || r.Published != 1 || r.Skipped != 1 || r.Pending != 0 || r.Sequence != 2 {
		t.Fatal(r, e)
	}
	if f.count("SELECT count(*) FROM managed_knowledge_events WHERE action='publish'") != 2 {
		t.Fatal("duplicate or missing audit")
	}
	b, _ := json.Marshal(r)
	if len(b) > 1048576 {
		t.Fatal("receipt exceeded limit")
	}
}
func TestPublishStepReplayAndTabs(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	f.Import("admin_a", "seed", f.doc, false)
	a := f.Access("admin_a", "job")
	j, e := f.repo.CreatePublishJob(f.ctx, a, knowledgeadmin.PublishCreateInput{Mode: knowledgeadmin.PublishAll})
	if e != nil {
		t.Fatal(e)
	}
	one := 1
	input := knowledgeadmin.PublishStepInput{ExpectedSequence: 1, BatchSize: &one}
	a = f.Access("admin_a", "step-1")
	r, e := f.repo.StepPublishJob(f.ctx, a, j.ID, input)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = f.repo.StepPublishJob(f.ctx, f.Access("admin_a", "other-tab"), j.ID, input); !errors.Is(e, knowledgeadmin.ErrStale) {
		t.Fatal(e)
	}
	if _, e = f.repo.StepPublishJob(f.ctx, f.Access("admin_a", "step-2"), j.ID, knowledgeadmin.PublishStepInput{ExpectedSequence: 2}); e != nil {
		t.Fatal(e)
	}
	again, e := f.repo.StepPublishJob(f.ctx, a, j.ID, input)
	if e != nil || again.Sequence != r.Sequence || again.Pending != 1 {
		t.Fatal(again, e)
	}
	if _, e = f.repo.StepPublishJob(f.ctx, a, j.ID, knowledgeadmin.PublishStepInput{ExpectedSequence: 1}); !errors.Is(e, knowledgeadmin.ErrIdempotency) {
		t.Fatal(e)
	}
	receipt, e := f.repo.ReadPublishJobReceipt(f.ctx, a, j.ID, "step-1")
	if e != nil || receipt == nil || receipt.Sequence != 2 {
		t.Fatal(receipt, e)
	}
}

func TestPublishStepRevocationAndRollback(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	f.Import("admin_a", "seed", f.doc, false)
	a := f.Access("admin_a", "job")
	j, e := f.repo.CreatePublishJob(f.ctx, a, knowledgeadmin.PublishCreateInput{Mode: knowledgeadmin.PublishAll})
	if e != nil {
		t.Fatal(e)
	}

	fault := publishFaultStore(t, f, "statement")
	_, e = fault.StepPublishJob(f.ctx, f.Access("admin_a", "failed-step"), j.ID, knowledgeadmin.PublishStepInput{ExpectedSequence: 1})
	if !errors.Is(e, knowledgeadmin.ErrPublishRolledBack) {
		t.Fatal("rollback must be confirmed", e)
	}
	if f.count("SELECT count(*) FROM managed_knowledge WHERE published") != 0 || f.count("SELECT count(*) FROM managed_knowledge_publish_receipts WHERE action='step'") != 0 {
		t.Fatal("partial commit")
	}
	d, e := f.repo.ReadPublishJob(f.ctx, a, j.ID, knowledgeadmin.PublishItemQuery{})
	if e != nil || d.Job.Sequence != 1 || d.Job.Pending != 2 {
		t.Fatal(d, e)
	}

	f.exec("DELETE FROM auth_user_roles WHERE user_id=$1 AND role='admin'", f.ids["admin_a"])
	if _, e = f.repo.StepPublishJob(f.ctx, f.Access("admin_a", "revoked-step"), j.ID, knowledgeadmin.PublishStepInput{ExpectedSequence: 1}); e == nil || errors.Is(e, knowledgeadmin.ErrPublishRolledBack) {
		t.Fatal("revocation rejected incorrectly", e)
	}
}
func TestPublishStepMixedStaleDeletedInvalid(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	f.Import("admin_a", "seed", f.doc, false)
	a := f.Access("admin_a", "job")
	j, e := f.repo.CreatePublishJob(f.ctx, a, knowledgeadmin.PublishCreateInput{Mode: knowledgeadmin.PublishAll})
	if e != nil {
		t.Fatal(e)
	}
	f.exec("UPDATE managed_knowledge SET edit_token=gen_random_uuid() WHERE external_id=$1", f.doc.KnowledgePoints[0].ID)
	f.exec("UPDATE managed_knowledge SET point=jsonb_set(point,'{statement}','\"\"') WHERE external_id=$1", f.doc.KnowledgePoints[1].ID)
	r, e := f.repo.StepPublishJob(f.ctx, f.Access("admin_a", "step"), j.ID, knowledgeadmin.PublishStepInput{ExpectedSequence: 1})
	if e != nil || r.Failed != 2 || r.Pending != 0 || r.Published != 0 {
		t.Fatal(r, e)
	}
}

func TestPublishStepReplayOverlappingJobs(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	f.Import("admin_a", "seed", f.doc, false)
	j, e := f.repo.CreatePublishJob(f.ctx, f.Access("admin_a", "one"), knowledgeadmin.PublishCreateInput{Mode: knowledgeadmin.PublishAll})
	if e != nil {
		t.Fatal(e)
	}
	other, e := f.repo.CreatePublishJob(f.ctx, f.Access("admin_b", "two"), knowledgeadmin.PublishCreateInput{Mode: knowledgeadmin.PublishAll})
	if e != nil {
		t.Fatal(e)
	}
	results := make(chan error, 2)
	for _, key := range []string{"tab-one", "tab-two"} {
		go func(key string) {
			_, e := f.repo.StepPublishJob(f.ctx, f.Access("admin_a", key), j.ID, knowledgeadmin.PublishStepInput{ExpectedSequence: 1})
			results <- e
		}(key)
	}
	success, stale := 0, 0
	for n := 0; n < 2; n++ {
		e := <-results
		if e == nil {
			success++
		} else if errors.Is(e, knowledgeadmin.ErrStale) {
			stale++
		} else {
			t.Fatal(e)
		}
	}
	if success != 1 || stale != 1 {
		t.Fatal(success, stale)
	}
	r, e := f.repo.StepPublishJob(f.ctx, f.Access("admin_b", "other-step"), other.ID, knowledgeadmin.PublishStepInput{ExpectedSequence: 1})
	if e != nil || r.Skipped != 2 || r.Published != 0 {
		t.Fatal(r, e)
	}
}

func TestPublishStepLargeContent(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	f.Import("admin_a", "seed", f.doc, false)
	f.exec(`UPDATE managed_knowledge SET point=jsonb_set(point,'{statement}',to_jsonb(repeat('x',65536)))`)
	j, e := f.repo.CreatePublishJob(f.ctx, f.Access("admin_a", "large"), knowledgeadmin.PublishCreateInput{Mode: knowledgeadmin.PublishAll})
	if e != nil {
		t.Fatal(e)
	}
	r, e := f.repo.StepPublishJob(f.ctx, f.Access("admin_a", "large-step"), j.ID, knowledgeadmin.PublishStepInput{ExpectedSequence: 1})
	if e != nil || r.Published != 2 {
		t.Fatal(r, e)
	}
	b, _ := json.Marshal(r)
	if len(b) > 1048576 {
		t.Fatal("unbounded metadata")
	}
}
