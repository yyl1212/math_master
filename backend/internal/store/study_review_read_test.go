package store_test

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/knowledgeadmin"
	"github.com/yyl1212/math_master/backend/internal/studyreview"
	"strings"
	"testing"
)

func TestReviewQueueAndHistoryOwnership(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := completedReviewKnowledge(t, f)
	a := f.Access("learner_a", "queue")
	o, e := f.repo.ReadReviewOverview(f.ctx, a)
	if e != nil || o.Total != 1 || o.Unplanned != 1 || o.ActorID != f.ids["learner_a"] {
		t.Fatalf("unplanned %+v %v", o, e)
	}
	p, e := f.repo.ListReviewQueue(f.ctx, a, studyreview.QueueQuery{Limit: 20, Bucket: "unplanned", TopicKey: "97-XX"})
	if e != nil || p.Total != 1 || len(p.Items) != 1 || p.Items[0].Knowledge != k.Ref {
		t.Fatalf("queue %+v %v", p, e)
	}
	p, e = f.repo.ListReviewQueue(f.ctx, a, studyreview.QueueQuery{Limit: 20, Q: "%"})
	if e != nil || p.Total != 0 {
		t.Fatal("wildcard expanded", e)
	}
	r := startReview(t, f, k, studyreview.Read, studyreview.Statement, "start")
	o, e = f.repo.ReadReviewOverview(f.ctx, a)
	if e != nil || o.Unfinished != 1 || o.Unplanned != 0 {
		t.Fatal("exclusive bucket", o, e)
	}
	c := compareReview(t, f, r, "", false)
	g, e := f.repo.GradeReviewSession(f.ctx, f.Access("learner_a", "grade"), r.SessionID, studyreview.GradeInput{ExpectedSequence: c.Sequence, Rating: studyreview.Familiar})
	if e != nil {
		t.Fatal(e)
	}
	o, e = f.repo.ReadReviewOverview(f.ctx, a)
	if e != nil || o.Future != 1 || o.Unfinished != 0 {
		t.Fatal(o, e)
	}
	h, e := f.repo.ListReviewHistory(f.ctx, a, studyreview.HistoryQuery{Limit: 20})
	if e != nil || len(h.Items) != 1 || h.Items[0].Grade.Schedule.Stage != 1 || h.Items[0].SessionID != g.SessionID {
		t.Fatal(h, e)
	}
	h, e = f.repo.ListReviewHistory(f.ctx, f.Access("admin_a", "history"), studyreview.HistoryQuery{Limit: 20})
	if e != nil || len(h.Items) != 0 {
		t.Fatal("admin history leak", e)
	}
	// A later early familiar grade keeps dueAt; the first immutable Grade remains at sequence 1.
	next := startReview(t, f, k, studyreview.Read, studyreview.Statement, "next")
	nc := compareReview(t, f, next, "", false)
	ng, e := f.repo.GradeReviewSession(f.ctx, f.Access("learner_a", "grade2"), next.SessionID, studyreview.GradeInput{ExpectedSequence: nc.Sequence, Rating: studyreview.Familiar})
	if e != nil || !ng.Grade.Schedule.DueAt.Equal(g.Grade.Schedule.DueAt) || ng.Grade.Schedule.Stage != 1 {
		t.Fatal("early advanced", ng, e)
	}
	old, e := f.repo.ReadReviewSession(f.ctx, a, r.SessionID)
	if e != nil || old.Grade.Schedule.Sequence != 1 || old.Schedule.Sequence != 2 {
		t.Fatal("history rewritten", old, e)
	}
}
func TestReviewCurrentRefAndAvailability(t *testing.T) {
	for _, status := range []string{"working", "compared", "completed"} {
		t.Run(status, func(t *testing.T) {
			f := newKnowledgeAdminFixture(t)
			k := completedReviewKnowledge(t, f)
			r := startReview(t, f, k, studyreview.Recall, studyreview.Statement, "start")
			d, e := f.repo.SaveReviewDraft(f.ctx, f.Access("learner_a", "draft"), r.SessionID, studyreview.DraftInput{ExpectedSequence: r.Sequence, Body: "保留作答"})
			if e != nil {
				t.Fatal(e)
			}
			r = d
			if status != "working" {
				r = compareReview(t, f, r, "保留作答", false)
			}
			if status == "completed" {
				r, e = f.repo.GradeReviewSession(f.ctx, f.Access("learner_a", "grade"), r.SessionID, studyreview.GradeInput{ExpectedSequence: r.Sequence, Rating: studyreview.Familiar})
				if e != nil {
					t.Fatal(e)
				}
			}
			m, e := f.repo.ReadManagedKnowledge(f.ctx, f.Access("admin_a", "admin"), k.ID)
			if e != nil {
				t.Fatal(e)
			}
			down, e := f.repo.SetManagedKnowledgeState(f.ctx, f.Access("admin_a", "down"), k.ID, m.EditToken, "unpublish")
			if e != nil {
				t.Fatal(e)
			}
			s, e := f.repo.ReadReviewSession(f.ctx, f.Access("learner_a", "read-down"), r.SessionID)
			if e != nil || s.Available || len(s.Prompt) != 0 || s.Reference != nil || s.Answer == nil || *s.Answer != "保留作答" {
				t.Fatal("down text exposed", s, e)
			}
			up, e := f.repo.SetManagedKnowledgeState(f.ctx, f.Access("admin_a", "up"), k.ID, down.EditToken, "publish")
			if e != nil {
				t.Fatal(e)
			}
			s, e = f.repo.ReadReviewSession(f.ctx, f.Access("learner_a", "read-up"), r.SessionID)
			if e != nil || !s.Available || (status != "working" && s.Reference == nil) {
				t.Fatal("same ref restore", e)
			}
			in := knowledgeadmin.CurrentInput{ExternalID: up.ExternalID, Point: up.Point, Sources: up.Sources, TopicKeys: up.TopicKeys}
			in.Point.Statement += " 新版本条件。"
			_, e = f.repo.UpdateManagedKnowledge(f.ctx, f.Access("admin_a", "edit"), k.ID, up.EditToken, in)
			if e != nil {
				t.Fatal(e)
			}
			s, e = f.repo.ReadReviewSession(f.ctx, f.Access("learner_a", "read-changed"), r.SessionID)
			if e != nil || !s.MaterialChanged || len(s.Prompt) != 0 || s.Reference != nil || *s.Answer != "保留作答" {
				t.Fatal("changed text exposed", e)
			}
			if status != "completed" {
				if _, e = f.repo.CompareReviewSession(f.ctx, f.Access("learner_a", "bad-compare"), r.SessionID, studyreview.CompareInput{ExpectedSequence: r.Sequence, Body: "new"}); !errors.Is(e, studyreview.ErrStale) {
					t.Fatal("stale compare", e)
				}
				if _, e = f.repo.AbandonReviewSession(f.ctx, f.Access("learner_a", "abandon"), r.SessionID, studyreview.AbandonInput{ExpectedSequence: r.Sequence}); e != nil {
					t.Fatal(e)
				}
			}
		})
	}
}
func TestReviewProofPromptRedaction(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	f.doc.KnowledgePoints[0], f.doc.KnowledgePoints[1] = f.doc.KnowledgePoints[1], f.doc.KnowledgePoints[0]
	f.doc.KnowledgePoints[0].Equations = []string{"x^2>0"}
	k := completedReviewKnowledge(t, f)
	r := startReview(t, f, k, studyreview.Recall, studyreview.Proof, "proof")
	s, e := f.repo.ReadReviewSession(f.ctx, f.Access("learner_a", "proof-read"), r.SessionID)
	if e != nil || s.Reference != nil || len(s.Prompt) == 0 {
		t.Fatal(s, e)
	}
	found := false
	for _, v := range s.Prompt {
		if v.Kind == "equations" && v.Items[0] == "x^2>0" {
			found = true
		}
	}
	if !found {
		t.Fatal("equation missing")
	}
	m, e := f.repo.ReadManagedKnowledge(f.ctx, f.Access("admin_a", "admin"), k.ID)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = f.repo.SetManagedKnowledgeState(f.ctx, f.Access("admin_a", "down"), k.ID, m.EditToken, "unpublish"); e != nil {
		t.Fatal(e)
	}
	s, e = f.repo.ReadReviewSession(f.ctx, f.Access("learner_a", "after-down"), r.SessionID)
	if e != nil || len(s.Prompt) != 0 || s.Reference != nil {
		t.Fatal("proof prompt leaked", e)
	}
}
func TestReviewReferenceTooLargeHasNoBusinessWrites(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := completedReviewKnowledge(t, f)
	// Published content is allowed to exceed the recall wire budget; existing reading remains usable.
	f.exec(`UPDATE managed_knowledge SET point=jsonb_set(point,'{statement}',to_jsonb($2::text)),content_sha256=repeat('c',64) WHERE internal_id=$1`, k.ID, strings.Repeat("<", 100000))
	current, e := f.repo.ReadCurrentKnowledge(f.ctx, k.ID)
	if e != nil {
		t.Fatal(e)
	}
	_, e = f.repo.StartReviewSession(f.ctx, f.Access("learner_a", "huge"), studyreview.StartInput{Knowledge: current.Ref, Mode: studyreview.Recall, Target: studyreview.Statement})
	if !errors.Is(e, studyreview.ErrReferenceTooLarge) {
		t.Fatal("oversize", e)
	}
	if f.count(`SELECT count(*) FROM study_review_sessions`) != 0 || f.count(`SELECT count(*) FROM managed_study_events WHERE kind='review-started'`) != 0 {
		t.Fatal("422 partial business write")
	}
}

func TestReviewQueueDedupChangedManualCycleAndHistoryPages(t *testing.T) {
	f := newKnowledgeAdminFixture(t)
	k := completedReviewKnowledge(t, f)
	m, e := f.repo.ReadManagedKnowledge(f.ctx, f.Access("admin_a", "first"), k.ID)
	if e != nil {
		t.Fatal(e)
	}
	in := knowledgeadmin.CurrentInput{ExternalID: m.ExternalID, Point: m.Point, Sources: m.Sources, TopicKeys: []string{"97F40", "97F50"}}
	in.Point.Title += " % _"
	in.Point.Statement += " 已修订。"
	updated, e := f.repo.UpdateManagedKnowledge(f.ctx, f.Access("admin_a", "changed"), k.ID, m.EditToken, in)
	if e != nil {
		t.Fatal(e)
	}
	k.Ref = updated.Ref
	a := f.Access("learner_a", "queue")
	o, e := f.repo.ReadReviewOverview(f.ctx, a)
	if e != nil || o.Changed != 1 || o.Unplanned != 0 {
		t.Fatal("old unscheduled change hidden", o, e)
	}
	for _, topic := range []string{"97-XX", "97Fxx", "97F40", "97F50"} {
		p, e := f.repo.ListReviewQueue(f.ctx, a, studyreview.QueueQuery{Limit: 1, TopicKey: topic, Q: "%", Bucket: "due"})
		if e != nil || p.Total != 1 || len(p.Items) != 1 || p.Items[0].Bucket != "changed" {
			t.Fatal("topic dedup/literal/changed due", topic, p, e)
		}
	}
	old, e := f.repo.ReadManagedStudy(f.ctx, a, k.ID)
	if e != nil {
		t.Fatal(e)
	}
	old, e = f.repo.ApplyManagedStudy(f.ctx, f.Access("learner_a", "manual-start"), k.ID, "start-review", knowledgeadmin.ManagedStudyInput{Knowledge: k.Ref, ExpectedSequence: old.Record.Sequence})
	if e != nil {
		t.Fatal(e)
	}
	o, e = f.repo.ReadReviewOverview(f.ctx, a)
	if e != nil || o.Unfinished != 1 || o.Changed != 0 {
		t.Fatal("old reviewing absent", o, e)
	}
	r := startReview(t, f, k, studyreview.Read, studyreview.Statement, "new-reuses-manual")
	if r.ReviewID != *old.Record.ActiveReviewID {
		t.Fatal("manual cycle not reused")
	}
	c := compareReview(t, f, r, "", false)
	if _, e = f.repo.GradeReviewSession(f.ctx, f.Access("learner_a", "grade-a"), r.SessionID, studyreview.GradeInput{ExpectedSequence: c.Sequence, Rating: studyreview.Uncertain}); e != nil {
		t.Fatal(e)
	}
	r2 := startReview(t, f, k, studyreview.Read, studyreview.Statement, "next")
	c2 := compareReview(t, f, r2, "", false)
	if _, e = f.repo.GradeReviewSession(f.ctx, f.Access("learner_a", "grade-b"), r2.SessionID, studyreview.GradeInput{ExpectedSequence: c2.Sequence, Rating: studyreview.Forgotten}); e != nil {
		t.Fatal(e)
	}
	h, e := f.repo.ListReviewHistory(f.ctx, a, studyreview.HistoryQuery{Limit: 1, TopicKey: "97-XX"})
	if e != nil || len(h.Items) != 1 || h.Items[0].SessionID != r2.SessionID || h.NextCursor == nil {
		t.Fatal("first history page", h, e)
	}
	next, e := f.repo.ListReviewHistory(f.ctx, a, studyreview.HistoryQuery{Limit: 1, TopicKey: "97-XX", Cursor: *h.NextCursor})
	if e != nil || len(next.Items) != 1 || next.Items[0].SessionID != r.SessionID || next.NextCursor != nil {
		t.Fatal("second history page", next, e)
	}
	if _, e = f.repo.ListReviewHistory(f.ctx, a, studyreview.HistoryQuery{Limit: 1, Rating: studyreview.Familiar, Cursor: *h.NextCursor}); !errors.Is(e, studyreview.ErrInvalid) {
		t.Fatal("cross-filter cursor accepted", e)
	}
	page, e := f.repo.ListReviewQueue(f.ctx, a, studyreview.QueueQuery{Limit: 1, Offset: 1, TopicKey: "97-XX"})
	if e != nil || page.Total != 1 || len(page.Items) != 0 {
		t.Fatal("empty offset page", page, e)
	}
}
