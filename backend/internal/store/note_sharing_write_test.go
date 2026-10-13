package store_test

import (
	"errors"
	"fmt"
	"github.com/pressly/goose/v3"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/notesharing"
	"sync"
	"testing"
)

func sharingNote(t *testing.T, f *knowledgeAdminFixture, k knowledgeadmin.PublicKnowledge, body string) knowledgeadmin.ManagedNote {
	t.Helper()
	n, e := f.repo.SaveManagedNote(f.ctx, f.Access("learner_a", "note-initial"), k.ID, knowledgeadmin.ManagedNoteInput{Knowledge: k.Ref, Body: body})
	if e != nil {
		t.Fatal(e)
	}
	return n
}
func sharingSubmit(t *testing.T, f *knowledgeAdminFixture, k knowledgeadmin.PublicKnowledge, n knowledgeadmin.ManagedNote) notesharing.Receipt {
	t.Helper()
	r, e := f.repo.Submit(f.ctx, f.Access("learner_a", "share-initial"), k.ID, notesharing.SubmitInput{ExpectedRevision: n.Revision, Title: "自己的数学理解"})
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func TestSharingPrivateOwnerIsolation(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := f.PreparedManagedKnowledge()
	sharingNote(t, f, k, "本人秘密数学思路")
	for _, name := range []string{"learner_a", "learner_b", "admin_a"} {
		p, e := f.repo.ListOwnNotes(f.ctx, f.Access(name, "own-read"), notesharing.Query{})
		if e != nil {
			t.Fatal(e)
		}
		want := 0
		if name == "learner_a" {
			want = 1
		}
		if len(p.Items) != want || p.ActorID != f.ids[name] {
			t.Fatal("owner isolation", name, p)
		}
	}
	p, e := f.repo.ListAdmin(f.ctx, f.Access("admin_a", "queue"), notesharing.Query{})
	if e != nil || len(p.Items) != 0 {
		t.Fatal("private note in queue", e)
	}
}

func TestSharingNotesLiteralSearchAndPagination(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := f.PreparedManagedKnowledge()
	sharingNote(t, f, k, "含有符号 % _ **强调** $x^2$ 的私人笔记")
	old, e := f.repo.ReadManagedKnowledge(f.ctx, f.Access("admin_a", "read-clone"), k.ID)
	if e != nil {
		t.Fatal(e)
	}
	in := knowledgeadmin.CurrentInput{ExternalID: old.ExternalID + "-notes-second", Point: old.Point, Sources: old.Sources, TopicKeys: old.TopicKeys}
	in.Point.ID = in.ExternalID
	second, e := f.repo.CreateManagedKnowledge(f.ctx, f.Access("admin_a", "second-point"), in)
	if e != nil {
		t.Fatal(e)
	}
	second, e = f.repo.SetManagedKnowledgeState(f.ctx, f.Access("admin_a", "publish-second"), second.ID, second.EditToken, "publish")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = f.repo.SaveManagedNote(f.ctx, f.Access("learner_a", "second-note"), second.ID, knowledgeadmin.ManagedNoteInput{Knowledge: second.Ref, Body: "另一份记录"}); e != nil {
		t.Fatal(e)
	}
	page, e := f.repo.ListOwnNotes(f.ctx, f.Access("learner_a", "first-page"), notesharing.Query{Limit: 1})
	if e != nil || len(page.Items) != 1 || page.NextCursor == nil {
		t.Fatal(page, e)
	}
	next, e := f.repo.ListOwnNotes(f.ctx, f.Access("learner_a", "next-page"), notesharing.Query{Limit: 1, Cursor: *page.NextCursor})
	if e != nil || len(next.Items) != 1 || next.Items[0].KnowledgeID == page.Items[0].KnowledgeID || next.NextCursor != nil {
		t.Fatal(next, e)
	}
	match, e := f.repo.ListOwnNotes(f.ctx, f.Access("learner_a", "literal"), notesharing.Query{Q: "%"})
	if e != nil || len(match.Items) != 1 || match.Items[0].KnowledgeID != k.ID {
		t.Fatal("wildcard search", match, e)
	}
	if _, e = f.repo.ListOwnNotes(f.ctx, f.Access("learner_a", "wrong-filter"), notesharing.Query{Q: "%", Cursor: *page.NextCursor}); !errors.Is(e, notesharing.ErrInvalid) {
		t.Fatal("cross-filter cursor", e)
	}
	parent, e := f.repo.ListOwnNotes(f.ctx, f.Access("learner_a", "parent"), notesharing.Query{TopicKey: "97-XX"})
	if e != nil || len(parent.Items) != 2 {
		t.Fatal("parent or dedup", parent, e)
	}
}
func TestSharingSnapshotExactSavedRevision(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := f.PreparedManagedKnowledge()
	n := sharingNote(t, f, k, "这是已保存的正文 $x^2$。")
	r := sharingSubmit(t, f, k, n)
	d, e := f.repo.ReadAdmin(f.ctx, f.Access("admin_a", "snapshot"), r.SubmissionID, "")
	if e != nil || d.Body != n.Body || d.NoteRevision != n.Revision || d.NoteKnowledgeRef != *n.Knowledge {
		t.Fatal("wrong snapshot", d, e)
	}
	if _, e = f.repo.Submit(f.ctx, f.Access("learner_a", "share-again"), k.ID, notesharing.SubmitInput{ExpectedRevision: n.Revision, Title: "重复申请"}); !errors.Is(e, notesharing.ErrLocked) {
		t.Fatal("pending resubmitted", e)
	}
	if f.count("SELECT count(*) FROM note_share_submissions") != 1 {
		t.Fatal("duplicate snapshot")
	}
}
func TestSharingSaveReceiptBeforeLock(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := f.PreparedManagedKnowledge()
	n := sharingNote(t, f, k, "原保存待确认")
	sharingSubmit(t, f, k, n)
	got, e := f.repo.SaveManagedNote(f.ctx, f.Access("learner_a", "note-initial"), k.ID, knowledgeadmin.ManagedNoteInput{Knowledge: k.Ref, Body: n.Body})
	if e != nil || got.Revision != n.Revision || f.count("SELECT count(*) FROM managed_study_events WHERE kind='note-saved'") != 1 {
		t.Fatal("successful replay blocked", e)
	}
}
func TestSharingLockedDelete(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := f.PreparedManagedKnowledge()
	n := sharingNote(t, f, k, "待审内容")
	sharingSubmit(t, f, k, n)
	_, e := f.repo.DeleteManagedNote(f.ctx, f.Access("learner_a", "delete-locked"), k.ID, knowledgeadmin.ManagedNoteInput{Knowledge: k.Ref, ExpectedRevision: n.Revision})
	if !errors.Is(e, notesharing.ErrLocked) {
		t.Fatal("locked delete", e)
	}
}

func TestSharingSaveSchemaGuardAndAbsentExtension(t *testing.T) {
	for _, damaged := range []bool{false, true} {
		t.Run(fmt.Sprint(damaged), func(t *testing.T) {
			f := newKnowledgeAdminFixture(t)
			k := f.PreparedManagedKnowledge()
			n := sharingNote(t, f, k, "原笔记保留")
			if damaged {
				f.exec("ALTER TABLE note_share_heads DROP CONSTRAINT note_share_head_status")
			} else {
				if e := goose.Down(f.db, "../../../db/migrations"); e != nil {
					t.Fatal(e)
				}
			}
			_, e := f.repo.SaveManagedNote(f.ctx, f.Access("learner_a", "new-save"), k.ID, knowledgeadmin.ManagedNoteInput{Knowledge: k.Ref, ExpectedRevision: n.Revision, Body: "新笔记"})
			if damaged && !errors.Is(e, notesharing.ErrNotReady) || !damaged && e != nil {
				t.Fatal("schema write boundary", e)
			}
			got, e := f.repo.ReadManagedNote(f.ctx, f.Access("learner_a", "original-read"), k.ID)
			if e != nil {
				t.Fatal(e)
			}
			if damaged && got.Body != n.Body || !damaged && got.Body != "新笔记" {
				t.Fatal("read or original save changed")
			}
		})
	}
}
func TestSharingAdminOwnSubmissionNeedsExplicitApproval(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := f.PreparedManagedKnowledge()
	n, e := f.repo.SaveManagedNote(f.ctx, f.Access("admin_a", "own-admin-note"), k.ID, knowledgeadmin.ManagedNoteInput{Knowledge: k.Ref, Body: "管理员自己的笔记"})
	if e != nil {
		t.Fatal(e)
	}
	r, e := f.repo.Submit(f.ctx, f.Access("admin_a", "own-submit"), k.ID, notesharing.SubmitInput{ExpectedRevision: n.Revision, Title: "独立明确批准"})
	if e != nil || r.Status != notesharing.Pending {
		t.Fatal("auto-approved", r, e)
	}
	r, e = f.repo.Decide(f.ctx, f.Access("admin_a", "own-approve"), r.SubmissionID, "approve", notesharing.DecisionInput{ExpectedSequence: r.Sequence})
	if e != nil || r.Status != notesharing.Shared {
		t.Fatal(r, e)
	}
}
func TestSharingWrongKeyInput(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := f.PreparedManagedKnowledge()
	n := sharingNote(t, f, k, "同一请求保护")
	r := sharingSubmit(t, f, k, n)
	in := notesharing.SubmitInput{ExpectedRevision: n.Revision, Title: "自己的数学理解"}
	again, e := f.repo.Submit(f.ctx, f.Access("learner_a", "share-initial"), k.ID, in)
	if e != nil || again != r {
		t.Fatal("replay", e)
	}
	in.Title = "不同输入"
	_, e = f.repo.Submit(f.ctx, f.Access("learner_a", "share-initial"), k.ID, in)
	if !errors.Is(e, notesharing.ErrIdempotency) {
		t.Fatal("changed key input", e)
	}
}
func TestSharingReceiptCannotRepublish(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := f.PreparedManagedKnowledge()
	n := sharingNote(t, f, k, "先分享再撤回")
	r := sharingSubmit(t, f, k, n)
	a := f.Access("admin_a", "approve-once")
	in := notesharing.DecisionInput{ExpectedSequence: r.Sequence}
	approved, e := f.repo.Decide(f.ctx, a, r.SubmissionID, "approve", in)
	if e != nil {
		t.Fatal(e)
	}
	w, e := f.repo.Withdraw(f.ctx, f.Access("learner_a", "withdraw"), k.ID, notesharing.WithdrawInput{SubmissionID: r.SubmissionID, ExpectedSequence: approved.Sequence})
	if e != nil {
		t.Fatal(e)
	}
	got, e := f.repo.Decide(f.ctx, a, r.SubmissionID, "approve", in)
	if e != nil || got != approved {
		t.Fatal("original receipt", e)
	}
	h, e := f.repo.ReadOwnHead(f.ctx, f.Access("learner_a", "head"), k.ID)
	if e != nil || h.Status != notesharing.Withdrawn || h.Sequence != w.Sequence {
		t.Fatal("receipt republished", h, e)
	}
	newNote, e := f.repo.SaveManagedNote(f.ctx, f.Access("learner_a", "after-withdraw"), k.ID, knowledgeadmin.ManagedNoteInput{Knowledge: k.Ref, ExpectedRevision: n.Revision, Body: "撤回后修改的内容"})
	if e != nil {
		t.Fatal(e)
	}
	newShare, e := f.repo.Submit(f.ctx, f.Access("learner_a", "new-share"), k.ID, notesharing.SubmitInput{ExpectedRevision: newNote.Revision, ExpectedSequence: w.Sequence, Title: "重新分享"})
	if e != nil || newShare.SubmissionID == r.SubmissionID {
		t.Fatal("old snapshot reused", e)
	}
	d, e := f.repo.ReadAdmin(f.ctx, f.Access("admin_a", "old-snapshot"), r.SubmissionID, "")
	if e != nil || d.Body != n.Body || d.Status != notesharing.Withdrawn {
		t.Fatal("historical snapshot changed", d, e)
	}
}
func TestSharingOldSaveReplayKeepsCurrentNote(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := f.PreparedManagedKnowledge()
	n := sharingNote(t, f, k, "原正文A")
	b, e := f.repo.SaveManagedNote(f.ctx, f.Access("learner_a", "note-B"), k.ID, knowledgeadmin.ManagedNoteInput{Knowledge: k.Ref, ExpectedRevision: n.Revision, Body: "后续正文B"})
	if e != nil {
		t.Fatal(e)
	}
	a, e := f.repo.SaveManagedNote(f.ctx, f.Access("learner_a", "note-initial"), k.ID, knowledgeadmin.ManagedNoteInput{Knowledge: k.Ref, Body: n.Body})
	if e != nil || a.Body != b.Body || a.Revision != b.Revision || f.count("SELECT count(*) FROM managed_study_events WHERE kind='note-saved'") != 2 {
		t.Fatal("old save overwrote B", e)
	}
}
func TestSharingWithdrawUnavailableKnowledge(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := f.PreparedManagedKnowledge()
	n := sharingNote(t, f, k, "保留原私人笔记")
	r := sharingSubmit(t, f, k, n)
	m, e := f.repo.ReadManagedKnowledge(f.ctx, f.Access("admin_a", "read-down"), k.ID)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = f.repo.SetManagedKnowledgeState(f.ctx, f.Access("admin_a", "down"), k.ID, m.EditToken, "unpublish"); e != nil {
		t.Fatal(e)
	}
	if _, e = f.repo.Withdraw(f.ctx, f.Access("learner_a", "withdraw-down"), k.ID, notesharing.WithdrawInput{SubmissionID: r.SubmissionID, ExpectedSequence: r.Sequence}); e != nil {
		t.Fatal(e)
	}
	got, e := f.repo.ReadManagedNote(f.ctx, f.Access("learner_a", "retained"), k.ID)
	if e != nil || got.Body != n.Body {
		t.Fatal(e)
	}
}
func sharingRace(t *testing.T, f *knowledgeAdminFixture, k knowledgeadmin.PublicKnowledge, blockSQL string, jobs ...func() error) []error {
	t.Helper()
	tx, e := f.db.BeginTx(f.ctx, nil)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = tx.ExecContext(f.ctx, blockSQL, k.ID); e != nil {
		t.Fatal(e)
	}
	start := make(chan struct{})
	ready := make(chan struct{}, len(jobs))
	out := make([]error, len(jobs))
	var group sync.WaitGroup
	for i, job := range jobs {
		group.Add(1)
		go func(i int, job func() error) { defer group.Done(); ready <- struct{}{}; <-start; out[i] = job() }(i, job)
	}
	for range jobs {
		<-ready
	}
	close(start)
	if _, e = f.db.ExecContext(f.ctx, "SELECT pg_sleep(0.05)"); e != nil {
		t.Fatal(e)
	}
	if e = tx.Rollback(); e != nil {
		t.Fatal(e)
	}
	group.Wait()
	return out
}
func TestSharingSaveSubmitRace(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := f.PreparedManagedKnowledge()
	n := sharingNote(t, f, k, "原稿")
	err := sharingRace(t, f, k, "SELECT internal_id FROM managed_knowledge WHERE internal_id=$1 FOR UPDATE", func() error {
		_, e := f.repo.SaveManagedNote(f.ctx, f.Access("learner_a", "racing-save"), k.ID, knowledgeadmin.ManagedNoteInput{Knowledge: k.Ref, ExpectedRevision: n.Revision, Body: "新稿"})
		return e
	}, func() error {
		_, e := f.repo.Submit(f.ctx, f.Access("learner_a", "racing-submit"), k.ID, notesharing.SubmitInput{ExpectedRevision: n.Revision, Title: "竞态申请"})
		return e
	})
	if (err[0] == nil) == (err[1] == nil) {
		t.Fatal("both write paths succeeded or failed", err)
	}
	for _, e := range err {
		if e != nil && !errors.Is(e, notesharing.ErrLocked) && !errors.Is(e, notesharing.ErrStale) {
			t.Fatal(e)
		}
	}
}
func TestSharingWithdrawApproveRace(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := f.PreparedManagedKnowledge()
	n := sharingNote(t, f, k, "竞态正文")
	r := sharingSubmit(t, f, k, n)
	err := sharingRace(t, f, k, "SELECT knowledge_id FROM managed_study_records WHERE knowledge_id=$1 AND owner_user_id='"+f.ids["learner_a"]+"' FOR UPDATE", func() error {
		_, e := f.repo.Withdraw(f.ctx, f.Access("learner_a", "racing-withdraw"), k.ID, notesharing.WithdrawInput{SubmissionID: r.SubmissionID, ExpectedSequence: r.Sequence})
		return e
	}, func() error {
		_, e := f.repo.Decide(f.ctx, f.Access("admin_a", "racing-approve"), r.SubmissionID, "approve", notesharing.DecisionInput{ExpectedSequence: r.Sequence})
		return e
	})
	if (err[0] == nil) == (err[1] == nil) {
		t.Fatal("decisions not serialized", err)
	}
	for _, e := range err {
		if e != nil && !errors.Is(e, notesharing.ErrStale) {
			t.Fatal(e)
		}
	}
	h, e := f.repo.ReadOwnHead(f.ctx, f.Access("learner_a", "race-head"), k.ID)
	if e != nil {
		t.Fatal(e)
	}
	if h.Status == notesharing.Shared {
		if _, e = f.repo.Withdraw(f.ctx, f.Access("learner_a", "confirmed-withdraw"), k.ID, notesharing.WithdrawInput{SubmissionID: r.SubmissionID, ExpectedSequence: h.Sequence}); e != nil {
			t.Fatal(e)
		}
	}
	if _, e = f.repo.Decide(f.ctx, f.Access("admin_b", "stale-approval"), r.SubmissionID, "approve", notesharing.DecisionInput{ExpectedSequence: r.Sequence}); !errors.Is(e, notesharing.ErrStale) {
		t.Fatal("old page approved", e)
	}
}
func TestSharingTwoAdminDecisions(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := f.PreparedManagedKnowledge()
	n := sharingNote(t, f, k, "两位管理员")
	r := sharingSubmit(t, f, k, n)
	jobs := []func() error{}
	for i, name := range []string{"admin_a", "admin_b"} {
		i, name := i, name
		jobs = append(jobs, func() error {
			_, e := f.repo.Decide(f.ctx, f.Access(name, fmt.Sprintf("approve-%d", i)), r.SubmissionID, "approve", notesharing.DecisionInput{ExpectedSequence: r.Sequence})
			return e
		})
	}
	err := sharingRace(t, f, k, "SELECT internal_id FROM managed_knowledge WHERE internal_id=$1 FOR UPDATE", jobs...)
	if (err[0] == nil) == (err[1] == nil) || f.count("SELECT count(*) FROM note_share_events WHERE action='approve'") != 1 {
		t.Fatal("multiple approvals", err)
	}
}
