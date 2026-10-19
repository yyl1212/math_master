package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	k "github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"testing"
	"time"
)

func TestPublishCapacity10001SnapshotAndSteps(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	f.Import("admin_a", "seed", f.doc, false)
	var cancel context.CancelFunc
	f.ctx, cancel = context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	f.exec(`WITH source AS(SELECT * FROM managed_knowledge ORDER BY internal_id LIMIT 1),fixtures AS(SELECT source.*,'publish-capacity-'||n AS external FROM source CROSS JOIN generate_series(1,9999) n) INSERT INTO managed_knowledge(internal_id,external_id,point,public_sources,content_sha256,created_by,updated_by) SELECT 'k-'||substr(encode(sha256(convert_to(external,'UTF8')),'hex'),1,56),external,jsonb_set(point,'{id}',to_jsonb(external)),public_sources,content_sha256,created_by,updated_by FROM fixtures`)
	start := time.Now()
	job, e := f.repo.CreatePublishJob(f.ctx, f.Access("admin_a", "capacity-create"), k.PublishCreateInput{Mode: k.PublishAll})
	if e != nil || job.Total != 10001 {
		t.Fatal(job, e)
	}
	t.Logf("create snapshot duration=%s", time.Since(start))
	f.exec(`INSERT INTO managed_knowledge(internal_id,external_id,point,public_sources,content_sha256,created_by,updated_by) SELECT 'k-'||repeat('f',56),'after-snapshot',jsonb_set(point,'{id}','"after-snapshot"'),public_sources,content_sha256,created_by,updated_by FROM managed_knowledge LIMIT 1`)
	f.exec(`UPDATE managed_knowledge SET updated_at=updated_at+interval '1 hour' WHERE NOT published`)
	seq := int64(1)
	steps := 0
	published := int64(0)
	start = time.Now()
	for published < 10001 {
		r, e := f.repo.StepPublishJob(f.ctx, f.Access("admin_a", fmt.Sprintf("capacity-step-%d", seq)), job.ID, k.PublishStepInput{ExpectedSequence: seq})
		if e != nil {
			t.Fatal("batch", seq, e)
		}
		if r.Total != r.Pending+r.Published+r.Skipped+r.Failed || r.Failed != 0 || r.Skipped != 0 || r.Sequence != seq+1 {
			t.Fatal(r)
		}
		b, _ := json.Marshal(r)
		if len(b) > 1048576 {
			t.Fatal("unbounded receipt")
		}
		seq = r.Sequence
		published = r.Published
		steps++
	}
	t.Logf("steps=%d publish duration=%s", steps, time.Since(start))
	if steps != 501 {
		t.Fatal(steps)
	}
	if f.count("SELECT count(*) FROM managed_knowledge WHERE published") != 10001 || f.count("SELECT count(*) FROM managed_knowledge_events WHERE action='publish'") != 10001 {
		t.Fatal("wrong publication/audit counts")
	}
	if f.count("SELECT count(*) FROM managed_knowledge WHERE external_id='after-snapshot' AND NOT published") != 1 {
		t.Fatal("later item published")
	}
}

func TestPublishCapacityLargeBodyAndInterrupt(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	f.Import("admin_a", "seed", f.doc, false)
	f.exec(`WITH source AS(SELECT * FROM managed_knowledge ORDER BY internal_id LIMIT 1),fixtures AS(SELECT source.*,'publish-large-'||n AS external FROM source CROSS JOIN generate_series(1,39) n) INSERT INTO managed_knowledge(internal_id,external_id,point,public_sources,content_sha256,created_by,updated_by) SELECT 'k-'||substr(encode(sha256(convert_to(external,'UTF8')),'hex'),1,56),external,jsonb_set(point,'{id}',to_jsonb(external)),public_sources,content_sha256,created_by,updated_by FROM fixtures`)
	f.exec(`UPDATE managed_knowledge SET point=jsonb_set(point,'{statement}',to_jsonb(repeat('x',65536)))`)
	j, e := f.repo.CreatePublishJob(f.ctx, f.Access("admin_a", "large-create"), k.PublishCreateInput{Mode: k.PublishAll})
	if e != nil || j.Total != 41 {
		t.Fatal(j, e)
	}
	lost := publishFaultStore(t, f, "commit")
	a := f.Access("admin_a", "large-lost-step")
	if _, e = lost.StepPublishJob(f.ctx, a, j.ID, k.PublishStepInput{ExpectedSequence: 1}); e == nil || errors.Is(e, k.ErrPublishRolledBack) {
		t.Fatal(e)
	}
	r, e := f.repo.ReadPublishJobReceipt(f.ctx, a, j.ID, a.IdempotencyKey)
	if e != nil || r == nil || r.Published != 20 || r.Pending != 21 {
		t.Fatal(r, e)
	}
	b, _ := json.Marshal(r)
	if len(b) > 1048576 {
		t.Fatal("oversize receipt")
	}
	fault := publishFaultStore(t, f, "statement")
	if _, e = fault.StepPublishJob(f.ctx, f.Access("admin_a", "large-rollback"), j.ID, k.PublishStepInput{ExpectedSequence: 2}); !errors.Is(e, k.ErrPublishRolledBack) {
		t.Fatal(e)
	}
	one := 1
	next, e := f.repo.StepPublishJob(f.ctx, f.Access("admin_a", "large-smaller"), j.ID, k.PublishStepInput{ExpectedSequence: 2, BatchSize: &one})
	if e != nil || next.Published != 21 || next.Pending != 20 {
		t.Fatal(next, e)
	}
}
