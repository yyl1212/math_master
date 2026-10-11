package store_test

import (
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/study"
	"testing"
)

func TestManagedChangedNoteDoesNotAcknowledgeContent(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := f.PreparedManagedKnowledge()
	d, e := f.repo.ApplyManagedStudy(f.ctx, f.Access("learner_a", "begin-fixed"), k.ID, "begin", knowledgeadmin.ManagedStudyInput{Knowledge: k.Ref})
	if e != nil {
		t.Fatal(e)
	}
	d, e = f.repo.ApplyManagedStudy(f.ctx, f.Access("learner_a", "complete-fixed"), k.ID, "complete", knowledgeadmin.ManagedStudyInput{Knowledge: k.Ref, ExpectedSequence: d.Record.Sequence})
	if e != nil {
		t.Fatal(e)
	}
	originalSequence := d.Record.Sequence
	admin, e := f.repo.ReadManagedKnowledge(f.ctx, f.Access("admin_a", "read-fixed"), k.ID)
	if e != nil {
		t.Fatal(e)
	}
	input := knowledgeadmin.CurrentInput{ExternalID: admin.ExternalID, Point: admin.Point, Sources: admin.Sources}
	input.Point.Statement += " 更正后的数学正文。"
	updated, e := f.repo.UpdateManagedKnowledge(f.ctx, f.Access("admin_a", "update-fixed"), k.ID, admin.EditToken, input)
	if e != nil {
		t.Fatal(e)
	}
	_, e = f.repo.SaveManagedNote(f.ctx, f.Access("learner_a", "note-fixed"), k.ID, knowledgeadmin.ManagedNoteInput{Knowledge: updated.Ref, Body: "笔记不确认已复习", ExpectedRevision: 0})
	if e != nil {
		t.Fatal(e)
	}
	overview, e := f.repo.ReadManagedOverview(f.ctx, f.Access("learner_a", "overview-fixed"))
	if e != nil || overview.MaterialChanged != 1 {
		t.Fatalf("note cleared overview: %+v %v", overview, e)
	}
	page, e := f.repo.ListManagedStudyKnowledge(f.ctx, f.Access("learner_a", "list-fixed"), knowledgeadmin.Query{State: "completed"})
	if e != nil || len(page.Items) != 1 || !page.Items[0].MaterialChanged || page.Items[0].Record.Sequence != originalSequence {
		t.Fatal(page, e)
	}
	d, e = f.repo.ReadManagedStudy(f.ctx, f.Access("learner_a", "detail-fixed"), k.ID)
	if e != nil || !d.MaterialChanged {
		t.Fatal(d, e)
	}
	d, e = f.repo.ApplyManagedStudy(f.ctx, f.Access("learner_a", "review-fixed"), k.ID, "start-review", knowledgeadmin.ManagedStudyInput{Knowledge: updated.Ref, ExpectedSequence: d.Record.Sequence})
	if e != nil {
		t.Fatal(e)
	}
	d, e = f.repo.ApplyManagedStudy(f.ctx, f.Access("learner_a", "finish-fixed"), k.ID, "finish-review", knowledgeadmin.ManagedStudyInput{Knowledge: updated.Ref, ExpectedSequence: d.Record.Sequence, ReviewID: d.Record.ActiveReviewID})
	if e != nil || d.MaterialChanged {
		t.Fatal(d, e)
	}
	overview, e = f.repo.ReadManagedOverview(f.ctx, f.Access("learner_a", "review-overview-fixed"))
	if e != nil || overview.MaterialChanged != 0 {
		t.Fatal(overview, e)
	}
}

func TestManagedRetiresCorrectionWorkerWithoutTouchingState(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	f.PreparedManagedKnowledge()
	before := f.count("SELECT count(*) FROM correction_jobs")
	_, e := f.repo.ClaimCorrectionJob(f.ctx)
	if !errors.Is(e, correction.ErrRetired) {
		t.Fatal("worker remains active", e)
	}
	if f.count("SELECT count(*) FROM correction_jobs") != before {
		t.Fatal("old jobs mutated")
	}
}

func TestManagedMissingStateCannotResumeOldCorrectionJob(t *testing.T) {
	f := newCorrectionFixture(t)
	f.registerRule(nil, 1)
	var before, after string
	if e := f.db.QueryRow(`SELECT jsonb_agg(to_jsonb(j) ORDER BY id)::text FROM correction_jobs j`).Scan(&before); e != nil {
		t.Fatal(e)
	}
	f.exec(`DROP TABLE knowledge_admin_state CASCADE`)
	_, e := f.repo.ClaimCorrectionJob(f.ctx)
	if e == nil {
		t.Fatal("partial managed schema resumed an old job")
	}
	if e = f.db.QueryRow(`SELECT jsonb_agg(to_jsonb(j) ORDER BY id)::text FROM correction_jobs j`).Scan(&after); e != nil {
		t.Fatal(e)
	}
	if before != after {
		t.Fatal("old job state, lease or sequence changed")
	}
}

func TestManagedReviewerCannotReadCurrentFeedbackQueue(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	f.PreparedManagedKnowledge()
	_, e := f.repo.ListFeedbackTickets(f.ctx, f.FeedbackAccess("reviewer_a"), true, feedback.ListQuery{})
	if !errors.Is(e, auth.ErrForbidden) {
		t.Fatal("old reviewer received managed queue", e)
	}
	_, e = f.repo.ListFeedbackTickets(f.ctx, f.FeedbackAccess("admin_a"), true, feedback.ListQuery{})
	if e != nil {
		t.Fatal(e)
	}
}

func TestManagedRoleAssignmentRejectsRetiredRoles(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	f.PreparedManagedKnowledge()
	cookies, csrf := f.login("admin_a", accountTestPassword)
	f.reauth(cookies, csrf)
	_, e := f.admin().ReplaceRoles(f.ctx, cookies, csrf, f.ids["learner_a"], auth.RolesInput{Roles: []auth.Role{auth.RoleLearner, auth.RoleReviewer}, Reason: adminReason}, "managed-retired-role")
	if !errors.Is(e, auth.ErrInvalidInput) {
		t.Fatal("retired role assigned", e)
	}
	if f.count("SELECT count(*) FROM auth_user_roles WHERE user_id=$1 AND role='reviewer'", f.ids["learner_a"]) != 0 {
		t.Fatal("role changed")
	}
}

func TestManagedStreamlinedHistoryKeepsDeletedNameAndIdentity(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := f.PreparedManagedKnowledge()
	_, e := f.repo.ApplyManagedStudy(f.ctx, f.Access("learner_a", "history-begin"), k.ID, "begin", knowledgeadmin.ManagedStudyInput{Knowledge: k.Ref})
	if e != nil {
		t.Fatal(e)
	}
	before, e := f.repo.ListManagedHistory(f.ctx, f.Access("learner_a", "history-before"), study.HistoryQuery{})
	if e != nil || len(before.Items) != 1 {
		t.Fatal(before, e)
	}
	admin, e := f.repo.ReadManagedKnowledge(f.ctx, f.Access("admin_a", "history-admin"), k.ID)
	if e != nil {
		t.Fatal(e)
	}
	_, e = f.repo.SetManagedKnowledgeState(f.ctx, f.Access("admin_a", "history-delete"), k.ID, admin.EditToken, "delete")
	if e != nil {
		t.Fatal(e)
	}
	after, e := f.repo.ListManagedHistory(f.ctx, f.Access("learner_a", "history-after"), study.HistoryQuery{})
	if e != nil || len(after.Items) != 1 {
		t.Fatal(after, e)
	}
	item := after.Items[0]
	raw, _ := json.Marshal(item)
	var display map[string]any
	_ = json.Unmarshal(raw, &display)
	if item.ID != before.Items[0].ID || item.Knowledge != before.Items[0].Knowledge || display["titleZh"] != admin.Point.TitleZH || display["available"] != false {
		t.Fatal("history identity/name unavailable", item)
	}
	other, e := f.repo.ListManagedHistory(f.ctx, f.Access("learner_b", "other-history"), study.HistoryQuery{})
	if e != nil || len(other.Items) != 0 {
		t.Fatal(other, e)
	}
}

func TestManagedLatestCompletionWinsOlderReview(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := f.PreparedManagedKnowledge()
	d, e := f.repo.ApplyManagedStudy(f.ctx, f.Access("learner_a", "latest-begin"), k.ID, "begin", knowledgeadmin.ManagedStudyInput{Knowledge: k.Ref})
	if e != nil {
		t.Fatal(e)
	}
	for _, action := range []string{"complete", "start-review", "finish-review"} {
		d, e = f.repo.ApplyManagedStudy(f.ctx, f.Access("learner_a", "latest-"+action), k.ID, action, knowledgeadmin.ManagedStudyInput{Knowledge: k.Ref, ExpectedSequence: d.Record.Sequence, ReviewID: d.Record.ActiveReviewID})
		if e != nil {
			t.Fatal(e)
		}
	}
	admin, e := f.repo.ReadManagedKnowledge(f.ctx, f.Access("admin_a", "latest-admin"), k.ID)
	if e != nil {
		t.Fatal(e)
	}
	input := knowledgeadmin.CurrentInput{ExternalID: admin.ExternalID, Point: admin.Point, Sources: admin.Sources}
	input.Point.Statement += " 新确认内容。"
	updated, e := f.repo.UpdateManagedKnowledge(f.ctx, f.Access("admin_a", "latest-update"), k.ID, admin.EditToken, input)
	if e != nil {
		t.Fatal(e)
	}
	d, e = f.repo.ApplyManagedStudy(f.ctx, f.Access("learner_a", "latest-complete-new"), k.ID, "complete", knowledgeadmin.ManagedStudyInput{Knowledge: updated.Ref, ExpectedSequence: d.Record.Sequence})
	if e != nil || d.MaterialChanged {
		t.Fatal(d, e)
	}
	overview, e := f.repo.ReadManagedOverview(f.ctx, f.Access("learner_a", "latest-overview"))
	if e != nil || overview.MaterialChanged != 0 {
		t.Fatal(overview, e)
	}
}
