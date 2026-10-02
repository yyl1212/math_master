package store_test

import (
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"strings"
	"testing"
)

func TestLearningReadHistory101Pagination(t *testing.T) {
	f := newLearningFixture(t)
	for j := 0; j < 101; j++ {
		p := f.createPractice("learner_a")
		if _, e := f.repo.AbandonPractice(f.ctx, f.Access("learner_a", false), p.Summary.ID); e != nil {
			t.Fatal(e)
		}
	}
	a := f.Access("learner_a", false)
	first, e := f.repo.ListLearningHistory(f.ctx, a, learning.ListQuery{})
	if e != nil || first.Total != 101 || first.Limit != 20 || len(first.Items) != 20 {
		t.Fatal(first, e)
	}
	large, e := f.repo.ListLearningHistory(f.ctx, a, learning.ListQuery{Limit: 100})
	if e != nil || len(large.Items) != 100 || large.Items[0].ID != first.Items[0].ID || large.Items[19].ID != first.Items[19].ID {
		t.Fatal(large, e)
	}
	last, e := f.repo.ListLearningHistory(f.ctx, a, learning.ListQuery{Limit: 100, Offset: 100})
	if e != nil || last.Total != 101 || len(last.Items) != 1 {
		t.Fatal(last, e)
	}
	max, e := f.repo.ListLearningHistory(f.ctx, a, learning.ListQuery{Offset: 100000})
	if e != nil || max.Total != 101 || len(max.Items) != 0 {
		t.Fatal(max, e)
	}
	for _, q := range []learning.ListQuery{{Limit: 101}, {Limit: -1}, {Offset: -1}, {Offset: 100001}} {
		if _, e = f.repo.ListLearningHistory(f.ctx, a, q); !errors.Is(e, auth.ErrInvalidInput) {
			t.Fatal(q, e)
		}
	}
	other, e := f.repo.ListLearningHistory(f.ctx, f.Access("learner_b", false), learning.ListQuery{})
	if e != nil || other.Total != 0 {
		t.Fatal("history crossed accounts", other, e)
	}
	raw, _ := json.Marshal(large)
	if strings.Contains(string(raw), "correctNumeric") || strings.Contains(string(raw), "questions") || f.exposureSequence("learner_a") != 0 {
		t.Fatal("summary exposed answers")
	}
}
func TestLearningReplayProjectionRestricted(t *testing.T) {
	f := newLearningFixture(t)
	v := f.createDiagnostic("learner_a")
	key := f.Access("learner_a", false)
	in := f.answers(v, 5)
	original, e := f.repo.SubmitAssessment(f.ctx, key, v.Summary.ID, in)
	if e != nil {
		t.Fatal(e)
	}
	f.QWithdraw(question.WithdrawalTarget{Kind: "instance", ID: v.Questions[0].Instance.ID, Version: 1})
	before := f.exposureSequence("learner_a")
	got, e := f.repo.ReadAssessmentResult(f.ctx, f.Access("learner_a", false), v.Summary.ID)
	if e != nil || got.Score == nil || *got.Score != 5 || got.Passed == nil || !*got.Passed || got.Validity != assessment.Restricted || got.Items[0].CorrectNumeric != nil || got.Items[0].Explanation != nil || got.Items[0].Answer == nil || got.Items[1].CorrectNumeric == nil || f.exposureSequence("learner_a") <= before {
		t.Fatal("original score or current redaction", got, e)
	}
	for j := 0; j < 2; j++ {
		before = f.exposureSequence("learner_a")
		replay, e := f.repo.SubmitAssessment(f.ctx, key, v.Summary.ID, in)
		if e != nil || replay.Items[0].Explanation != nil || replay.Items[0].CorrectNumeric != nil || replay.Progress.QualificationGranted != original.Progress.QualificationGranted || f.exposureSequence("learner_a") <= before {
			t.Fatal("same-key replay bypassed withdrawal", replay, e)
		}
	}
	raw, _ := json.Marshal(got.Items[0])
	if strings.Contains(string(raw), "Adding these") {
		t.Fatal("withdrawn explanation leaked")
	}
	if _, e = f.repo.ReadAssessmentResult(f.ctx, f.Access("learner_b", false), v.Summary.ID); !errors.Is(e, auth.ErrNotFound) {
		t.Fatal(e)
	}
}
func TestLearningHistoricalProjectionAlternativeAndCurrentCounts(t *testing.T) {
	f := newLearningFixture(t)
	first := f.createDiagnostic("learner_a")
	r, e := f.repo.SubmitAssessment(f.ctx, f.Access("learner_a", false), first.Summary.ID, f.answers(first, 5))
	if e != nil {
		t.Fatal(e)
	}
	second := f.createDiagnostic("learner_a")
	if _, e = f.repo.SubmitAssessment(f.ctx, f.Access("learner_a", false), second.Summary.ID, f.answers(second, 5)); e != nil {
		t.Fatal(e)
	}
	f.QWithdraw(question.WithdrawalTarget{Kind: "instance", ID: second.Questions[0].Instance.ID, Version: 1})
	d, e := f.repo.ReadLearningKnowledge(f.ctx, f.Access("learner_a", false), f.knowledge.ID, 1)
	if e != nil || d.State.Qualification == nil || d.State.Qualification.EvidenceAttemptID != r.Summary.ID || d.State.State != learning.Mastered {
		t.Fatal("valid alternative ignored", d, e)
	}
	o, e := f.repo.ReadLearningOverview(f.ctx, f.Access("learner_a", false))
	if e != nil || o.CompletedCount != 0 || o.StartedCount != 0 || o.EffectivePassedCount != 1 || o.HistoricalUnlockedCount != 1 {
		t.Fatal(o, e)
	}
	for j := 0; j < 2; j++ {
		before := f.exposureSequence("learner_a")
		old, e := f.repo.ReadAssessmentResult(f.ctx, f.Access("learner_a", false), first.Summary.ID)
		if e != nil || old.Validity != assessment.Effective || f.exposureSequence("learner_a") <= before {
			t.Fatal(old, e)
		}
	}
}
func TestLearningHistoricalProjectionStaleAndAffected(t *testing.T) {
	f := newLearningFixture(t)
	p := f.createPractice("learner_a")
	key := f.Access("learner_a", false)
	if _, e := f.repo.RevealPractice(f.ctx, key, p.Summary.ID); e != nil {
		t.Fatal(e)
	}
	v := f.createDiagnostic("learner_a")
	next := newMaterial(f.Input())
	head := f.KHead()
	sub := f.ApprovedInput(next)
	f.Activate(f.Prepare(sub, head), head)
	got, e := f.repo.SubmitAssessment(f.ctx, f.Access("learner_a", false), v.Summary.ID, f.answers(v, 5))
	if e != nil || got.Outcome != assessment.Affected || got.Score != nil || got.Passed != nil {
		t.Fatal(got, e)
	}
	got, e = f.repo.ReadAssessmentResult(f.ctx, f.Access("learner_a", false), v.Summary.ID)
	if e != nil || got.Outcome != assessment.Affected || got.Score != nil || got.Passed != nil {
		t.Fatal(got, e)
	}
	old, e := f.repo.ReadPractice(f.ctx, f.Access("learner_a", false), p.Summary.ID)
	if e != nil || old.Result == nil || old.Result.Item.Validity != assessment.Stale || !containsReason(old.Result.Item.Reasons, assessment.KnowledgeUpdated) {
		t.Fatal("practice did not project current knowledge version", old, e)
	}
	replay, e := f.repo.RevealPractice(f.ctx, key, p.Summary.ID)
	if e != nil || replay.Result.Item.Validity != assessment.Stale {
		t.Fatal(replay, e)
	}
}
