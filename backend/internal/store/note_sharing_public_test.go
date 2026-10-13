package store_test

import (
	"errors"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/notesharing"
	"sort"
	"testing"
)

func sharingPublished(t *testing.T, f *knowledgeAdminFixture, k knowledgeadmin.PublicKnowledge, count int) ([]notesharing.Receipt, []knowledgeadmin.Access) {
	t.Helper()
	out := []notesharing.Receipt{}
	owners := []knowledgeadmin.Access{}
	for i := 0; i < count; i++ {
		_, cookies, csrf := f.signup(fmt.Sprintf("note_public_%02d", i))
		proof, e := auth.DecodeContentProof(cookies, csrf, true)
		if e != nil {
			t.Fatal(e)
		}
		a := knowledgeadmin.Access{TokenHash: proof.TokenHash, CSRF: proof.CSRF, IdempotencyKey: "save", RequestID: "public-test"}
		n, e := f.repo.SaveManagedNote(f.ctx, a, k.ID, knowledgeadmin.ManagedNoteInput{Knowledge: k.Ref, Body: fmt.Sprintf("自己的数学思路%d $x^2$", i)})
		if e != nil {
			t.Fatal(e)
		}
		a.IdempotencyKey = "submit"
		r, e := f.repo.Submit(f.ctx, a, k.ID, notesharing.SubmitInput{ExpectedRevision: n.Revision, Title: fmt.Sprintf("笔记%02d", i)})
		if e != nil {
			t.Fatal(e)
		}
		r, e = f.repo.Decide(f.ctx, f.Access("admin_a", fmt.Sprintf("approve-public-%d", i)), r.SubmissionID, "approve", notesharing.DecisionInput{ExpectedSequence: r.Sequence})
		if e != nil {
			t.Fatal(e)
		}
		out = append(out, r)
		owners = append(owners, a)
	}
	return out, owners
}
func TestSharingGuestLatestTenAndDirectDetail(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := f.PreparedManagedKnowledge()
	all, owners := sharingPublished(t, f, k, 12)
	latest, e := f.repo.Latest(f.ctx)
	if e != nil || len(latest.Items) != 10 {
		t.Fatal(latest, e)
	}
	if latest.Items[0].SubmissionID != all[11].SubmissionID || latest.Items[9].SubmissionID != all[2].SubmissionID {
		t.Fatal("not latest ten")
	}
	if _, e = f.repo.ReadPublic(f.ctx, all[1].SubmissionID); !errors.Is(e, notesharing.ErrNotPublic) {
		t.Fatal("eleventh direct detail", e)
	}
	logged, e := f.repo.ReadShared(f.ctx, f.Access("learner_b", "member-detail"), all[1].SubmissionID)
	if e != nil || logged.ActorID != f.ids["learner_b"] || logged.Note.Body == "" {
		t.Fatal(logged, e)
	}
	page, e := f.repo.ListShared(f.ctx, f.Access("learner_b", "member-list"), notesharing.Query{Limit: 5})
	if e != nil || len(page.Items) != 5 || page.NextCursor == nil {
		t.Fatal(page, e)
	}
	seen := map[string]bool{}
	for {
		for _, n := range page.Items {
			if seen[n.SubmissionID] {
				t.Fatal("duplicate")
			}
			seen[n.SubmissionID] = true
		}
		if page.NextCursor == nil {
			break
		}
		page, e = f.repo.ListShared(f.ctx, f.Access("learner_b", "member-next"), notesharing.Query{Limit: 5, Cursor: *page.NextCursor})
		if e != nil {
			t.Fatal(e)
		}
	}
	if len(seen) != 12 {
		t.Fatal("missing member items", len(seen))
	}
	a := owners[11]
	a.IdempotencyKey = "withdraw-public"
	if _, e = f.repo.Withdraw(f.ctx, a, k.ID, notesharing.WithdrawInput{SubmissionID: all[11].SubmissionID, ExpectedSequence: all[11].Sequence}); e != nil {
		t.Fatal(e)
	}
	if _, e = f.repo.ReadPublic(f.ctx, all[1].SubmissionID); e != nil {
		t.Fatal("window not replenished", e)
	}
}
func TestSharingPublicReadWindowBeforeID(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := f.PreparedManagedKnowledge()
	all, _ := sharingPublished(t, f, k, 12)
	f.exec("UPDATE note_share_heads SET approved_at='2026-01-01T00:00:00Z'")
	ids := []string{}
	for _, r := range all {
		ids = append(ids, r.SubmissionID)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(ids)))
	latest, e := f.repo.Latest(f.ctx)
	if e != nil {
		t.Fatal(e)
	}
	for i := range latest.Items {
		if latest.Items[i].SubmissionID != ids[i] {
			t.Fatal("unstable tied order")
		}
	}
	if _, e = f.repo.ReadPublic(f.ctx, ids[10]); !errors.Is(e, notesharing.ErrNotPublic) {
		t.Fatal("ID filtered before window", e)
	}
}
func TestSharingReadAfterWithdrawal(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := f.PreparedManagedKnowledge()
	all, owners := sharingPublished(t, f, k, 1)
	a := owners[0]
	a.IdempotencyKey = "withdraw"
	if _, e := f.repo.Withdraw(f.ctx, a, k.ID, notesharing.WithdrawInput{SubmissionID: all[0].SubmissionID, ExpectedSequence: all[0].Sequence}); e != nil {
		t.Fatal(e)
	}
	if _, e := f.repo.ReadPublic(f.ctx, all[0].SubmissionID); !errors.Is(e, notesharing.ErrNotPublic) {
		t.Fatal("public body after withdrawal", e)
	}
	if _, e := f.repo.ReadShared(f.ctx, f.Access("learner_a", "logged-withdrawn"), all[0].SubmissionID); !errors.Is(e, notesharing.ErrNotPublic) {
		t.Fatal("member body after withdrawal", e)
	}
}
func TestSharingKnowledgeRestoreVisibility(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := f.PreparedManagedKnowledge()
	all, owners := sharingPublished(t, f, k, 3)
	a := owners[1]
	a.IdempotencyKey = "withdraw"
	if _, e := f.repo.Withdraw(f.ctx, a, k.ID, notesharing.WithdrawInput{SubmissionID: all[1].SubmissionID, ExpectedSequence: all[1].Sequence}); e != nil {
		t.Fatal(e)
	}
	if _, e := f.repo.Decide(f.ctx, f.Access("admin_a", "hide"), all[2].SubmissionID, "hide", notesharing.DecisionInput{ExpectedSequence: all[2].Sequence, Reason: "这一份学习笔记暂不公开。"}); e != nil {
		t.Fatal(e)
	}
	m, e := f.repo.ReadManagedKnowledge(f.ctx, f.Access("admin_a", "k-read"), k.ID)
	if e != nil {
		t.Fatal(e)
	}
	m, e = f.repo.SetManagedKnowledgeState(f.ctx, f.Access("admin_a", "k-down"), k.ID, m.EditToken, "unpublish")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = f.repo.ReadPublic(f.ctx, all[0].SubmissionID); !errors.Is(e, notesharing.ErrNotPublic) {
		t.Fatal("down knowledge share", e)
	}
	if _, e = f.repo.SetManagedKnowledgeState(f.ctx, f.Access("admin_a", "k-up"), k.ID, m.EditToken, "publish"); e != nil {
		t.Fatal(e)
	}
	p, e := f.repo.Latest(f.ctx)
	if e != nil || len(p.Items) != 1 || p.Items[0].SubmissionID != all[0].SubmissionID {
		t.Fatal("withdrawn or hidden restored", p, e)
	}
}
