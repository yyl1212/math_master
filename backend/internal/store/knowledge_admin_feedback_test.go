package store_test

import (
	"crypto/rand"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/question"
	"testing"
)

func (f *knowledgeAdminFixture) FeedbackAccess(name string) question.Access {
	a := f.Access(name, "unused")
	id, e := auth.NewID(rand.Reader)
	if e != nil {
		f.t.Fatal(e)
	}
	return question.Access{TokenHash: a.TokenHash, CSRF: a.CSRF, IdempotencyKey: id, RequestID: "managed-feedback-test"}
}
func TestManagedFeedbackWithoutApproval(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := f.PreparedManagedKnowledge()
	context, e := f.repo.ReadFeedbackContext(f.ctx, f.FeedbackAccess("learner_a"), feedback.ContextQuery{Kind: "managed-knowledge", ID: k.ID})
	if e != nil || context.Data.Target.ManagedRef == nil || context.Data.Source.Kind != "managed" {
		t.Fatal(context, e)
	}
	in := feedback.CreateInput{Target: context.Data.Target, Source: context.Data.Source, Category: "math_error", Title: "条件说明需要修正", Message: "请检查知识点当前的条件说明。", Location: "知识点正文"}
	receipt, e := f.repo.CreateFeedback(f.ctx, f.FeedbackAccess("learner_a"), in)
	if e != nil || receipt.Data.Ticket.TargetValidity != "current" {
		t.Fatal(receipt, e)
	}
	if f.count("SELECT count(*) FROM content_review_decisions") != 0 {
		t.Fatal("fake approval created")
	}
	admin, e := f.repo.ReadManagedKnowledge(f.ctx, f.Access("admin_a", "get"), k.ID)
	if e != nil {
		t.Fatal(e)
	}
	edit := knowledgeadmin.CurrentInput{ExternalID: admin.ExternalID, Point: admin.Point, Sources: admin.Sources}
	edit.Point.Statement += " 管理员修正后的条件。"
	admin, e = f.repo.UpdateManagedKnowledge(f.ctx, f.Access("admin_a", "fix"), k.ID, admin.EditToken, edit)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = f.repo.CreateFeedback(f.ctx, f.FeedbackAccess("learner_a"), in); !errors.Is(e, feedback.ErrTargetStale) {
		t.Fatal("stale feedback accepted", e)
	}
	_, e = f.repo.SetManagedKnowledgeState(f.ctx, f.Access("admin_a", "hide"), k.ID, admin.EditToken, "unpublish")
	if e != nil {
		t.Fatal(e)
	}
	own, e := f.repo.ReadFeedbackTicket(f.ctx, f.FeedbackAccess("learner_a"), receipt.Data.Ticket.ID, false)
	if e != nil || own.Data.Target.ManagedRef.ContentSHA256 != k.Ref.ContentSHA256 || own.Data.TargetValidity != "withdrawn" {
		t.Fatal("historical ticket lost", own, e)
	}
	_, e = f.repo.ReplyFeedback(f.ctx, f.FeedbackAccess("learner_a"), receipt.Data.Ticket.ID, feedback.ReplyInput{ExpectedSequence: own.Data.Sequence, Message: "保留这条反馈讨论。"})
	if e != nil {
		t.Fatal("discussion blocked", e)
	}
}
