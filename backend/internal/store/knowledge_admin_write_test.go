package store_test

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"testing"
)

func TestManagedEditCAS(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	f.Import("admin_a", "seed", f.doc, true)
	id := knowledgeadmin.KnowledgeID(f.doc.KnowledgePoints[0].ID)
	k, e := f.repo.ReadManagedKnowledge(f.ctx, f.Access("admin_a", "read"), id)
	if e != nil {
		t.Fatal(e)
	}
	in := knowledgeadmin.CurrentInput{ExternalID: k.ExternalID, Point: k.Point, Sources: k.Sources}
	in.Point.Statement += " 修正。"
	a := f.Access("admin_a", "edit-once")
	saved, e := f.repo.UpdateManagedKnowledge(f.ctx, a, id, k.EditToken, in)
	if e != nil {
		t.Fatal(e)
	}
	again, e := f.repo.UpdateManagedKnowledge(f.ctx, a, id, k.EditToken, in)
	if e != nil || saved.EditToken != again.EditToken {
		t.Fatal("write replay not deterministic", e)
	}
	_, e = f.repo.UpdateManagedKnowledge(f.ctx, f.Access("admin_b", "stale"), id, k.EditToken, in)
	if !errors.Is(e, knowledgeadmin.ErrStale) {
		t.Fatal("stale overwrite", e)
	}
	in.Point.Statement += " changed request"
	_, e = f.repo.UpdateManagedKnowledge(f.ctx, a, id, k.EditToken, in)
	if !errors.Is(e, knowledgeadmin.ErrIdempotency) {
		t.Fatal("same key different input", e)
	}
	if f.count("SELECT count(*) FROM managed_knowledge_events WHERE action='update'") != 1 {
		t.Fatal("duplicate edit event")
	}
	unpublished, e := f.repo.SetManagedKnowledgeState(f.ctx, f.Access("admin_a", "unpublish"), id, saved.EditToken, "unpublish")
	if e != nil || unpublished.Published {
		t.Fatal(e)
	}
	deleted, e := f.repo.SetManagedKnowledgeState(f.ctx, f.Access("admin_a", "delete"), id, unpublished.EditToken, "delete")
	if e != nil || !deleted.Deleted {
		t.Fatal(e)
	}
	r := f.Import("admin_a", "deleted-replay", f.doc, true)
	if r.Counts.CreatedKnowledge != 0 {
		t.Fatal("deleted replay recreated")
	}
	k, e = f.repo.ReadManagedKnowledge(f.ctx, f.Access("admin_a", "deleted-read"), id)
	if e != nil || !k.Deleted || k.Published {
		t.Fatal("deleted replay resurrected", e)
	}
}
