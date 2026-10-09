package store_test

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"testing"
)

func TestManagedPrivateNotesDuringUnpublish(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := f.PreparedManagedKnowledge()
	in := knowledgeadmin.ManagedNoteInput{Knowledge: k.Ref, ExpectedRevision: 0, Body: "只有本人能够读取的内容。"}
	a := f.Access("learner_a", "save-once")
	note, e := f.repo.SaveManagedNote(f.ctx, a, k.ID, in)
	if e != nil {
		t.Fatal(e)
	}
	again, e := f.repo.SaveManagedNote(f.ctx, a, k.ID, in)
	if e != nil || again.Revision != note.Revision || f.count("SELECT count(*) FROM managed_study_events WHERE kind='note-saved'") != 1 {
		t.Fatal("duplicate note", e)
	}
	for _, name := range []string{"learner_b", "admin_a"} {
		other, e := f.repo.ReadManagedNote(f.ctx, f.Access(name, "read-own"), k.ID)
		if e != nil || other.Body != "" {
			t.Fatal("other owner's note disclosed", name, other, e)
		}
	}
	_, e = f.repo.SaveManagedNote(f.ctx, f.Access("learner_a", "stale-revision"), k.ID, in)
	if !errors.Is(e, knowledgeadmin.ErrStale) {
		t.Fatal("stale note overwritten", e)
	}
	in.ExpectedRevision = note.Revision
	in.Body = "<script>alert(1)</script>"
	if _, e = f.repo.SaveManagedNote(f.ctx, f.Access("learner_a", "unsafe"), k.ID, in); !errors.Is(e, knowledgeadmin.ErrInvalid) {
		t.Fatal("dangerous note", e)
	}
	admin, e := f.repo.ReadManagedKnowledge(f.ctx, f.Access("admin_a", "get"), k.ID)
	if e != nil {
		t.Fatal(e)
	}
	_, e = f.repo.SetManagedKnowledgeState(f.ctx, f.Access("admin_a", "unpublish"), k.ID, admin.EditToken, "unpublish")
	if e != nil {
		t.Fatal(e)
	}
	in.Body = "不应写入的新笔记内容。"
	if _, e = f.repo.SaveManagedNote(f.ctx, f.Access("learner_a", "down-save"), k.ID, in); !errors.Is(e, knowledgeadmin.ErrNotFound) {
		t.Fatal("down content saved", e)
	}
	own, e := f.repo.ReadManagedNote(f.ctx, f.Access("learner_a", "own-down"), k.ID)
	if e != nil || own.Body != note.Body {
		t.Fatal("own note unavailable", own, e)
	}
	detail, e := f.repo.ReadManagedStudy(f.ctx, f.Access("learner_a", "down-detail"), k.ID)
	if e != nil || detail.Available || detail.CurrentKnowledge != nil {
		t.Fatal("unpublished content disclosed", detail, e)
	}
}
