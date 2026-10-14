package store_test

import (
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/notesharing"
	"github.com/yyl1212/math_master/backend/internal/study"
	"github.com/yyl1212/math_master/backend/internal/studyreview"
	"testing"
)

func completedReviewKnowledge(t *testing.T, f *knowledgeAdminFixture) knowledgeadmin.PublicKnowledge {
	t.Helper()
	k := f.PreparedManagedKnowledge()
	begun, e := f.repo.ApplyManagedStudy(f.ctx, f.Access("learner_a", "review-seed-begin"), k.ID, "begin", knowledgeadmin.ManagedStudyInput{Knowledge: k.Ref})
	if e != nil {
		t.Fatal(e)
	}
	r, e := f.repo.ApplyManagedStudy(f.ctx, f.Access("learner_a", "review-seed-complete"), k.ID, "complete", knowledgeadmin.ManagedStudyInput{Knowledge: k.Ref, ExpectedSequence: begun.Record.Sequence})
	if e != nil || r.Record.FirstCompletedAt == nil {
		t.Fatal(r, e)
	}
	return k
}
func startReview(t *testing.T, f *knowledgeAdminFixture, k knowledgeadmin.PublicKnowledge, mode studyreview.Mode, target studyreview.Target, key string) studyreview.CommandReceipt {
	t.Helper()
	r, e := f.repo.StartReviewSession(f.ctx, f.Access("learner_a", key), studyreview.StartInput{Knowledge: k.Ref, Mode: mode, Target: target})
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func compareReview(t *testing.T, f *knowledgeAdminFixture, r studyreview.CommandReceipt, body string, unanswered bool) studyreview.CommandReceipt {
	t.Helper()
	c, e := f.repo.CompareReviewSession(f.ctx, f.Access("learner_a", "compare-"+r.SessionID), r.SessionID, studyreview.CompareInput{ExpectedSequence: r.Sequence, Body: body, Unanswered: unanswered})
	if e != nil {
		t.Fatal(e)
	}
	return c
}
func TestReviewLifecycleAndOldCycle(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := completedReviewKnowledge(t, f)
	original, e := f.repo.ReadManagedStudy(f.ctx, f.Access("learner_a", "initial"), k.ID)
	if e != nil {
		t.Fatal(e)
	}
	r := startReview(t, f, k, studyreview.Read, studyreview.Statement, "start")
	if r.Sequence != 1 || r.DraftRevision != 0 || r.Status != studyreview.Working || r.Action != "start" || r.Grade != nil {
		t.Fatalf("receipt %+v", r)
	}
	s, e := f.repo.ReadReviewSession(f.ctx, f.Access("learner_a", "read"), r.SessionID)
	if e != nil || s.Reference == nil || s.Answer != nil {
		t.Fatalf("read %+v %v", s, e)
	}
	c := compareReview(t, f, r, "", false)
	// Old finish arrives first; new self-rating must preserve exactly one original finish fact.
	old, e := f.repo.ReadManagedStudy(f.ctx, f.Access("learner_a", "old-state"), k.ID)
	if e != nil || old.Record.ActiveReviewID == nil {
		t.Fatal(e)
	}
	_, e = f.repo.ApplyManagedStudy(f.ctx, f.Access("learner_a", "old-finish"), k.ID, "finish-review", knowledgeadmin.ManagedStudyInput{Knowledge: k.Ref, ExpectedSequence: old.Record.Sequence, ReviewID: old.Record.ActiveReviewID})
	if e != nil {
		t.Fatal(e)
	}
	g, e := f.repo.GradeReviewSession(f.ctx, f.Access("learner_a", "grade"), r.SessionID, studyreview.GradeInput{ExpectedSequence: c.Sequence, Rating: studyreview.Familiar})
	if e != nil || g.Status != studyreview.Completed || g.Grade == nil || g.Grade.Schedule.Stage != 1 || g.Grade.Schedule.DueAt.Sub(g.AppliedAt).Seconds() != 86400 {
		t.Fatalf("grade %+v %v", g, e)
	}
	if f.count(`SELECT count(*) FROM managed_study_events WHERE kind='review-started'`) != 1 || f.count(`SELECT count(*) FROM managed_study_events WHERE kind='review-finished'`) != 1 {
		t.Fatal("duplicated original cycle")
	}
	old, e = f.repo.ReadManagedStudy(f.ctx, f.Access("learner_a", "facts"), k.ID)
	if e != nil || old.Record.State != study.Completed || !old.Record.FirstCompletedAt.Equal(*original.Record.FirstCompletedAt) || *old.Record.CompletedRef != *original.Record.CompletedRef {
		t.Fatal("completion changed", e)
	}
	next := startReview(t, f, k, studyreview.Recall, studyreview.Statement, "next-start")
	a, e := f.repo.AbandonReviewSession(f.ctx, f.Access("learner_a", "abandon"), next.SessionID, studyreview.AbandonInput{ExpectedSequence: next.Sequence})
	if e != nil || a.Status != studyreview.Abandoned {
		t.Fatal(a, e)
	}
	continued := startReview(t, f, k, studyreview.Read, studyreview.Statement, "continue-cycle")
	if continued.ReviewID != next.ReviewID || f.count(`SELECT count(*) FROM managed_study_events WHERE kind='review-started'`) != 2 || f.count(`SELECT count(*) FROM managed_study_events WHERE kind='review-finished'`) != 1 {
		t.Fatal("abandon faked finish")
	}
}
func TestReviewOldNextCycleCannotGrade(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := completedReviewKnowledge(t, f)
	r := startReview(t, f, k, studyreview.Recall, studyreview.Statement, "start")
	c := compareReview(t, f, r, "自己写的内容", false)
	d, e := f.repo.ReadManagedStudy(f.ctx, f.Access("learner_a", "old"), k.ID)
	if e != nil {
		t.Fatal(e)
	}
	d, e = f.repo.ApplyManagedStudy(f.ctx, f.Access("learner_a", "old-finish"), k.ID, "finish-review", knowledgeadmin.ManagedStudyInput{Knowledge: k.Ref, ExpectedSequence: d.Record.Sequence, ReviewID: d.Record.ActiveReviewID})
	if e != nil {
		t.Fatal(e)
	}
	d, e = f.repo.ApplyManagedStudy(f.ctx, f.Access("learner_a", "old-next"), k.ID, "start-review", knowledgeadmin.ManagedStudyInput{Knowledge: k.Ref, ExpectedSequence: d.Record.Sequence})
	if e != nil {
		t.Fatal(e)
	}
	_, e = f.repo.GradeReviewSession(f.ctx, f.Access("learner_a", "grade"), r.SessionID, studyreview.GradeInput{ExpectedSequence: c.Sequence, Rating: studyreview.Familiar})
	if !errors.Is(e, studyreview.ErrConflict) || f.count(`SELECT count(*) FROM study_review_schedules`) != 0 {
		t.Fatal("wrong cycle ended", e)
	}
	s, e := f.repo.ReadReviewSession(f.ctx, f.Access("learner_a", "answer"), r.SessionID)
	if e != nil || s.Answer == nil || *s.Answer != "自己写的内容" || s.Status != studyreview.Compared {
		t.Fatal("answer lost", e)
	}
}
func TestReviewReceiptsNeverResurrect(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := completedReviewKnowledge(t, f)
	in := studyreview.StartInput{Knowledge: k.Ref, Mode: studyreview.Recall, Target: studyreview.Statement}
	a := f.Access("learner_a", "start")
	r := startReview(t, f, k, in.Mode, in.Target, "start")
	di := studyreview.DraftInput{ExpectedSequence: 1, ExpectedDraftRevision: 0, Body: "我的私人草稿"}
	da := f.Access("learner_a", "draft")
	d, e := f.repo.SaveReviewDraft(f.ctx, da, r.SessionID, di)
	if e != nil {
		t.Fatal(e)
	}
	ci := studyreview.CompareInput{ExpectedSequence: d.Sequence, Body: "最终私人作答"}
	ca := f.Access("learner_a", "compare")
	c, e := f.repo.CompareReviewSession(f.ctx, ca, r.SessionID, ci)
	if e != nil {
		t.Fatal(e)
	}
	gi := studyreview.GradeInput{ExpectedSequence: c.Sequence, Rating: studyreview.Uncertain}
	ga := f.Access("learner_a", "grade")
	g, e := f.repo.GradeReviewSession(f.ctx, ga, r.SessionID, gi)
	if e != nil {
		t.Fatal(e)
	}
	again, e := f.repo.StartReviewSession(f.ctx, a, in)
	if e != nil || !equalReceipt(again, r) {
		t.Fatal("start replay", e)
	}
	again, e = f.repo.SaveReviewDraft(f.ctx, da, r.SessionID, di)
	if e != nil || !equalReceipt(again, d) {
		t.Fatal("draft replay", e)
	}
	again, e = f.repo.CompareReviewSession(f.ctx, ca, r.SessionID, ci)
	if e != nil || !equalReceipt(again, c) {
		t.Fatal("compare replay", e)
	}
	again, e = f.repo.GradeReviewSession(f.ctx, ga, r.SessionID, gi)
	if e != nil || !equalReceipt(again, g) || f.count(`SELECT count(*) FROM study_review_events WHERE action='graded'`) != 1 {
		t.Fatal("grade replay", e)
	}
	di.Body = "different"
	if _, e = f.repo.SaveReviewDraft(f.ctx, da, r.SessionID, di); !errors.Is(e, studyreview.ErrIdempotency) {
		t.Fatal("wrong input accepted", e)
	}
	raw, _ := json.Marshal(c)
	var receipt map[string]any
	_ = json.Unmarshal(raw, &receipt)
	for _, key := range []string{"answer", "body", "prompt", "reference"} {
		if _, ok := receipt[key]; ok {
			t.Fatal("receipt leaked body")
		}
	}
	s, e := f.repo.ReadReviewSession(f.ctx, f.Access("learner_a", "final"), r.SessionID)
	if e != nil || s.Status != studyreview.Completed || s.Sequence != g.Sequence || *s.Answer != "最终私人作答" {
		t.Fatal("replay revived", e)
	}
	admin, e := f.repo.ReadManagedKnowledge(f.ctx, f.Access("admin_a", "hide-read"), k.ID)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = f.repo.SetManagedKnowledgeState(f.ctx, f.Access("admin_a", "hide"), k.ID, admin.EditToken, "unpublish"); e != nil {
		t.Fatal(e)
	}
	again, e = f.repo.GradeReviewSession(f.ctx, ga, r.SessionID, gi)
	if e != nil || !equalReceipt(again, g) || f.count(`SELECT count(*) FROM study_review_events WHERE action='graded'`) != 1 {
		t.Fatal("down grade replay changed schedule", e)
	}
	hidden, e := f.repo.ReadReviewSession(f.ctx, f.Access("learner_a", "hidden"), r.SessionID)
	if e != nil || hidden.Reference != nil || len(hidden.Prompt) != 0 || hidden.Schedule.Sequence != 1 {
		t.Fatal("grade receipt leaked unavailable body", e)
	}

}
func equalReceipt(a, b studyreview.CommandReceipt) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}
func TestReviewMigration15KeepsSharing(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := f.PreparedManagedKnowledge()
	n := sharingNote(t, f, k, "十五迁移后原笔记")
	r := sharingSubmit(t, f, k, n)
	got, e := f.repo.Decide(f.ctx, f.Access("admin_a", "approve15"), r.SubmissionID, "approve", notesharing.DecisionInput{ExpectedSequence: r.Sequence})
	if e != nil || got.Status != notesharing.Shared {
		t.Fatal(got, e)
	}
}
func TestReviewOwnerIsolation(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := completedReviewKnowledge(t, f)
	r := startReview(t, f, k, studyreview.Recall, studyreview.Statement, "start")
	for _, actor := range []string{"learner_b", "admin_a"} {
		s, e := f.repo.ReadReviewSession(f.ctx, f.Access(actor, "private"), r.SessionID)
		if !errors.Is(e, studyreview.ErrNotFound) || s.Answer != nil {
			t.Fatal("other answer", actor, e)
		}
	}
	s, e := f.repo.ReadReviewSession(f.ctx, knowledgeadmin.Access{}, r.SessionID)
	if !errors.Is(e, auth.ErrAuthenticationRequired) || s.Answer != nil {
		t.Fatal("anonymous", e)
	}
	f.exec(`UPDATE auth_users SET must_change_password=true WHERE id=$1`, f.ids["learner_a"])
	if _, e = f.repo.ReadReviewSession(f.ctx, f.Access("learner_a", "forced"), r.SessionID); !errors.Is(e, auth.ErrPasswordChangeRequired) {
		t.Fatal("forced password", e)
	}
}

func TestReviewCrossTabAndUnanswered(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := completedReviewKnowledge(t, f)
	jobs := []func() error{}
	receipts := make([]studyreview.CommandReceipt, 2)
	for i := 0; i < 2; i++ {
		i := i
		jobs = append(jobs, func() error {
			r, e := f.repo.StartReviewSession(f.ctx, f.Access("learner_a", []string{"tab-a", "tab-b"}[i]), studyreview.StartInput{Knowledge: k.Ref, Mode: studyreview.Recall, Target: studyreview.Statement})
			receipts[i] = r
			return e
		})
	}
	result := sharingRace(t, f, k, `SELECT internal_id FROM managed_knowledge WHERE internal_id=$1 FOR UPDATE`, jobs...)
	winner := 0
	if result[0] != nil {
		winner = 1
	}
	if result[winner] != nil {
		t.Fatal(result)
	}
	var conflict *studyreview.ActiveSessionError
	if !errors.As(result[1-winner], &conflict) || conflict.SessionID != receipts[winner].SessionID {
		t.Fatal("active session identity", result)
	}
	if f.count(`SELECT count(*) FROM study_review_sessions`) != 1 || f.count(`SELECT count(*) FROM managed_study_events WHERE kind='review-started'`) != 1 {
		t.Fatal("two active cycles")
	}
	r := compareReview(t, f, receipts[winner], " \n\u3000", true)
	for _, rating := range []studyreview.Rating{studyreview.Uncertain, studyreview.Familiar} {
		if _, e := f.repo.GradeReviewSession(f.ctx, f.Access("learner_a", string(rating)), r.SessionID, studyreview.GradeInput{ExpectedSequence: r.Sequence, Rating: rating}); !errors.Is(e, studyreview.ErrInvalid) {
			t.Fatal("unanswered grade", e)
		}
	}
	g, e := f.repo.GradeReviewSession(f.ctx, f.Access("learner_a", "forgotten"), r.SessionID, studyreview.GradeInput{ExpectedSequence: r.Sequence, Rating: studyreview.Forgotten})
	if e != nil || g.Grade.Schedule.Stage != 0 || g.Grade.Schedule.DueAt.Sub(g.AppliedAt).Seconds() != 600 {
		t.Fatal(g, e)
	}
}
func TestReviewReceiptsAfterUnavailableAndAbandon(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := completedReviewKnowledge(t, f)
	in := studyreview.StartInput{Knowledge: k.Ref, Mode: studyreview.Recall, Target: studyreview.Statement}
	r := startReview(t, f, k, in.Mode, in.Target, "start")
	dIn := studyreview.DraftInput{ExpectedSequence: 1, Body: "旧版本的私人草稿"}
	d, e := f.repo.SaveReviewDraft(f.ctx, f.Access("learner_a", "draft"), r.SessionID, dIn)
	if e != nil {
		t.Fatal(e)
	}
	cIn := studyreview.CompareInput{ExpectedSequence: d.Sequence, Body: dIn.Body}
	c, e := f.repo.CompareReviewSession(f.ctx, f.Access("learner_a", "compare"), r.SessionID, cIn)
	if e != nil {
		t.Fatal(e)
	}
	m, e := f.repo.ReadManagedKnowledge(f.ctx, f.Access("admin_a", "admin"), k.ID)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = f.repo.SetManagedKnowledgeState(f.ctx, f.Access("admin_a", "down"), k.ID, m.EditToken, "unpublish"); e != nil {
		t.Fatal(e)
	}
	abIn := studyreview.AbandonInput{ExpectedSequence: c.Sequence}
	ab, e := f.repo.AbandonReviewSession(f.ctx, f.Access("learner_a", "abandon"), r.SessionID, abIn)
	if e != nil {
		t.Fatal(e)
	}
	replay, e := f.repo.StartReviewSession(f.ctx, f.Access("learner_a", "start"), in)
	if e != nil || !equalReceipt(r, replay) {
		t.Fatal("down start receipt", e)
	}
	replay, e = f.repo.SaveReviewDraft(f.ctx, f.Access("learner_a", "draft"), r.SessionID, dIn)
	if e != nil || !equalReceipt(d, replay) {
		t.Fatal("down draft receipt", e)
	}
	replay, e = f.repo.CompareReviewSession(f.ctx, f.Access("learner_a", "compare"), r.SessionID, cIn)
	if e != nil || !equalReceipt(c, replay) {
		t.Fatal("down compare receipt", e)
	}
	replay, e = f.repo.AbandonReviewSession(f.ctx, f.Access("learner_a", "abandon"), r.SessionID, abIn)
	if e != nil || !equalReceipt(ab, replay) {
		t.Fatal("abandon receipt", e)
	}
	s, e := f.repo.ReadReviewSession(f.ctx, f.Access("learner_a", "read"), r.SessionID)
	if e != nil || s.Status != studyreview.Abandoned || s.Reference != nil || len(s.Prompt) != 0 || *s.Answer != dIn.Body {
		t.Fatal("receipt revived old answer", e)
	}
}

func TestReviewCASModesAndConcurrentCompareGrade(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := completedReviewKnowledge(t, f)
	r := startReview(t, f, k, studyreview.Recall, studyreview.Statement, "start")
	if _, e := f.repo.GradeReviewSession(f.ctx, f.Access("learner_a", "early-grade"), r.SessionID, studyreview.GradeInput{ExpectedSequence: 1, Rating: studyreview.Familiar}); !errors.Is(e, studyreview.ErrConflict) {
		t.Fatal("working graded", e)
	}
	if _, e := f.repo.SaveReviewDraft(f.ctx, f.Access("learner_a", "bad-rev"), r.SessionID, studyreview.DraftInput{ExpectedSequence: 1, ExpectedDraftRevision: 1, Body: "incorrect"}); !errors.Is(e, studyreview.ErrConflict) {
		t.Fatal("draft CAS", e)
	}
	jobs := []func() error{}
	for i := 0; i < 2; i++ {
		i := i
		jobs = append(jobs, func() error {
			_, e := f.repo.CompareReviewSession(f.ctx, f.Access("learner_a", []string{"compare-a", "compare-b"}[i]), r.SessionID, studyreview.CompareInput{ExpectedSequence: 1, Body: []string{"答案A", "答案B"}[i]})
			return e
		})
	}
	result := sharingRace(t, f, k, `SELECT internal_id FROM managed_knowledge WHERE internal_id=$1 FOR UPDATE`, jobs...)
	if (result[0] == nil) == (result[1] == nil) {
		t.Fatal("both compare succeeded/failed", result)
	}
	s, e := f.repo.ReadReviewSession(f.ctx, f.Access("learner_a", "compared"), r.SessionID)
	if e != nil || s.Status != studyreview.Compared || s.DraftRevision != 1 || s.Sequence != 2 {
		t.Fatal(s, e)
	}
	if _, e = f.repo.SaveReviewDraft(f.ctx, f.Access("learner_a", "copy-reference"), r.SessionID, studyreview.DraftInput{ExpectedSequence: 2, ExpectedDraftRevision: 1, Body: "覆盖答案"}); !errors.Is(e, studyreview.ErrConflict) {
		t.Fatal("compared overwritten", e)
	}
	jobs = nil
	for i := 0; i < 2; i++ {
		i := i
		jobs = append(jobs, func() error {
			_, e := f.repo.GradeReviewSession(f.ctx, f.Access("learner_a", []string{"grade-a", "grade-b"}[i]), r.SessionID, studyreview.GradeInput{ExpectedSequence: 2, Rating: studyreview.Familiar})
			return e
		})
	}
	result = sharingRace(t, f, k, `SELECT internal_id FROM managed_knowledge WHERE internal_id=$1 FOR UPDATE`, jobs...)
	if (result[0] == nil) == (result[1] == nil) || f.count(`SELECT count(*) FROM study_review_events WHERE action='graded'`) != 1 || f.count(`SELECT count(*) FROM managed_study_events WHERE kind='review-finished'`) != 1 {
		t.Fatal("duplicated grade", result)
	}
	read := startReview(t, f, k, studyreview.Read, studyreview.Statement, "read-start")
	if _, e = f.repo.SaveReviewDraft(f.ctx, f.Access("learner_a", "read-draft"), read.SessionID, studyreview.DraftInput{ExpectedSequence: 1, Body: "x"}); !errors.Is(e, studyreview.ErrConflict) {
		t.Fatal("read has draft", e)
	}
}
