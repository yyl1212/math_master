package store_test

import (
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/study"
	"testing"
)

func (f *knowledgeAdminFixture) PreparedManagedKnowledge() knowledgeadmin.PublicKnowledge {
	f.t.Helper()
	f.Import("admin_a", "study-seed", f.doc, true)
	f.ActivateManaged()
	k, e := f.repo.ReadCurrentKnowledge(f.ctx, knowledgeadmin.KnowledgeID(f.doc.KnowledgePoints[0].ID))
	if e != nil {
		f.t.Fatal(e)
	}
	return k
}
func TestManagedStudyEditAndMove(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := f.PreparedManagedKnowledge()
	d, e := f.repo.ApplyManagedStudy(f.ctx, f.Access("learner_a", "begin"), k.ID, "begin", knowledgeadmin.ManagedStudyInput{Knowledge: k.Ref, ExpectedSequence: 0})
	if e != nil || d.Record.State != study.Learning {
		t.Fatal(d, e)
	}
	d, e = f.repo.ApplyManagedStudy(f.ctx, f.Access("learner_a", "complete"), k.ID, "complete", knowledgeadmin.ManagedStudyInput{Knowledge: k.Ref, ExpectedSequence: d.Record.Sequence})
	if e != nil || d.Record.State != study.Completed {
		t.Fatal(d, e)
	}
	completed := d.Record.FirstCompletedAt
	n, e := f.repo.SaveManagedNote(f.ctx, f.Access("learner_a", "note"), k.ID, knowledgeadmin.ManagedNoteInput{Knowledge: k.Ref, ExpectedRevision: 0, Body: "这是自己的学习笔记。"})
	if e != nil || n.Revision != 1 {
		t.Fatal(n, e)
	}
	admin, e := f.repo.ReadManagedKnowledge(f.ctx, f.Access("admin_a", "read"), k.ID)
	if e != nil {
		t.Fatal(e)
	}
	in := knowledgeadmin.CurrentInput{ExternalID: admin.ExternalID, Point: admin.Point, Sources: admin.Sources}
	in.Point.Statement += " 修正条件说明。"
	in.Point.MSCCodes = []string{"97F50"}
	in.Point.ClassificationEvidence[0].MSCCode = "97F50"
	_, e = f.repo.UpdateManagedKnowledge(f.ctx, f.Access("admin_a", "edit"), k.ID, admin.EditToken, in)
	if e != nil {
		t.Fatal(e)
	}
	d, e = f.repo.ReadManagedStudy(f.ctx, f.Access("learner_a", "detail"), k.ID)
	if e != nil || d.Record.State != study.Completed || !d.MaterialChanged || !d.Record.FirstCompletedAt.Equal(*completed) {
		t.Fatal("state or time reset", d, e)
	}
	n, e = f.repo.ReadManagedNote(f.ctx, f.Access("learner_a", "note-read"), k.ID)
	if e != nil || n.Body != "这是自己的学习笔记。" || n.Revision != 1 {
		t.Fatal("note lost", n, e)
	}
	history, e := f.repo.ListManagedHistory(f.ctx, f.Access("learner_a", "history"), study.HistoryQuery{KnowledgeID: k.ID})
	if e != nil || len(history.Items) != 3 {
		t.Fatal(history, e)
	}
	for _, v := range history.Items {
		if v.Knowledge.ContentSHA256 != k.Ref.ContentSHA256 || v.TopicKeys[0] != "97F40" {
			t.Fatal("old event context rewritten", v)
		}
	}
	progress, e := f.repo.ListManagedStudyTopics(f.ctx, f.Access("learner_a", "progress"), knowledgeadmin.Query{TopicKey: "97F50"})
	if e != nil || len(progress.Items) != 1 || progress.Items[0].Completed != 1 {
		t.Fatal(progress, e)
	}
	current, e := f.repo.ReadCurrentKnowledge(f.ctx, k.ID)
	if e != nil {
		t.Fatal(e)
	}
	d, e = f.repo.ApplyManagedStudy(f.ctx, f.Access("learner_a", "read-again"), k.ID, "begin", knowledgeadmin.ManagedStudyInput{Knowledge: current.Ref, ExpectedSequence: d.Record.Sequence})
	if e != nil || d.Record.State != study.Completed || f.count("SELECT count(*) FROM managed_study_events WHERE kind='completed'") != 1 {
		t.Fatal("read fakes completion", d, e)
	}
	d, e = f.repo.ApplyManagedStudy(f.ctx, f.Access("learner_a", "review-start"), k.ID, "start-review", knowledgeadmin.ManagedStudyInput{Knowledge: current.Ref, ExpectedSequence: d.Record.Sequence})
	if e != nil || d.Record.State != study.Reviewing || d.Record.ActiveReviewID == nil {
		t.Fatal(d, e)
	}
	d, e = f.repo.ApplyManagedStudy(f.ctx, f.Access("learner_a", "review-finish"), k.ID, "finish-review", knowledgeadmin.ManagedStudyInput{Knowledge: current.Ref, ExpectedSequence: d.Record.Sequence, ReviewID: d.Record.ActiveReviewID})
	if e != nil || d.Record.State != study.Completed || d.MaterialChanged {
		t.Fatal(d, e)
	}
}
func TestManagedStudyKnowledgeStateFilter(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := f.PreparedManagedKnowledge()
	_, e := f.repo.ApplyManagedStudy(f.ctx, f.Access("learner_a", "start"), k.ID, "begin", knowledgeadmin.ManagedStudyInput{Knowledge: k.Ref})
	if e != nil {
		t.Fatal(e)
	}
	page, e := f.repo.ListManagedStudyKnowledge(f.ctx, f.Access("learner_a", "list"), knowledgeadmin.Query{State: "learning"})
	if e != nil || page.Total != 1 || len(page.Items) != 1 {
		t.Fatal(page, e)
	}
	page, e = f.repo.ListManagedStudyKnowledge(f.ctx, f.Access("learner_a", "unlearned"), knowledgeadmin.Query{State: "unlearned"})
	if e != nil || page.Total != 1 || page.Items[0].Record.State != study.Unlearned {
		t.Fatal(page, e)
	}
}
