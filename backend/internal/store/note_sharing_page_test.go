package store_test

import (
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/notesharing"
	"github.com/yyl1212/math_master/backend/internal/pagination"
	"testing"
)

func TestNumberedNotesFiltersAndAnchor(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := f.PreparedManagedKnowledge()
	old, e := f.repo.ReadManagedKnowledge(f.ctx, f.Access("admin_a", "read"), k.ID)
	if e != nil {
		t.Fatal(e)
	}
	for n := 0; n < 3; n++ {
		in := knowledgeadmin.CurrentInput{ExternalID: fmt.Sprintf("numbered-%d", n), Point: old.Point, Sources: old.Sources, TopicKeys: old.TopicKeys}
		in.Point.ID = in.ExternalID
		c, e := f.repo.CreateManagedKnowledge(f.ctx, f.Access("admin_a", fmt.Sprintf("create-%d", n)), in)
		if e != nil {
			t.Fatal(e)
		}
		c, e = f.repo.SetManagedKnowledgeState(f.ctx, f.Access("admin_a", fmt.Sprintf("pub-%d", n)), c.ID, c.EditToken, "publish")
		if e != nil {
			t.Fatal(e)
		}
		_, e = f.repo.SaveManagedNote(f.ctx, f.Access("learner_a", fmt.Sprintf("note-%d", n)), c.ID, knowledgeadmin.ManagedNoteInput{Knowledge: c.Ref, Body: "独立笔记"})
		if e != nil {
			t.Fatal(e)
		}
	}
	p, e := f.repo.ListOwnNotePage(f.ctx, f.Access("learner_a", "page"), notesharing.Query{}, pagination.Request{Enabled: true, Page: 2, Limit: 2})
	if e != nil || p.Total != 3 || p.Offset != 2 || len(p.Items) != 1 {
		t.Fatalf("page: %+v %v", p, e)
	}
	other, e := f.repo.ListOwnNotePage(f.ctx, f.Access("learner_b", "other"), notesharing.Query{}, pagination.Request{Enabled: true, Page: 1, Limit: 2})
	if e != nil || other.Total != 0 || len(other.Items) != 0 {
		t.Fatal(other, e)
	}
	legacy, e := f.repo.ListOwnNotes(f.ctx, f.Access("learner_a", "legacy"), notesharing.Query{Limit: 2})
	if e != nil || legacy.NextCursor == nil {
		t.Fatal(legacy, e)
	}
	anchor, e := f.repo.ListOwnNotePage(f.ctx, f.Access("learner_a", "anchor"), notesharing.Query{}, pagination.Request{Enabled: true, Page: 1, Limit: 2, Cursor: *legacy.NextCursor})
	if e != nil || anchor.Offset != 2 || len(anchor.Items) != 1 {
		t.Fatal(anchor, e)
	}
	if _, e = f.repo.ListOwnNotePage(f.ctx, f.Access("learner_a", "scope"), notesharing.Query{Q: "different"}, pagination.Request{Enabled: true, Page: 1, Limit: 2, Cursor: *legacy.NextCursor}); e == nil {
		t.Fatal("cross-filter cursor accepted")
	}
}
