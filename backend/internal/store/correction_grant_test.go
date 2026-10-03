package store_test

import (
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"testing"
)

func TestCorrectionCompletionLaterGrantsAppendOnly(t *testing.T) {
	f := newCorrectionFixture(t)
	fact := f.passedFixture("learner_a", assessment.ModeNode)
	b := f.cLoadBasis(fact.ID)
	cid := f.cCase()
	p := f.cApproved(cid)
	b.HandledCaseIDs = []string{cid}
	b.PlanRefs = []correction.PlanRef{p}
	rid := f.cSealResult(fact.ID, cid, p, b)
	v, e := f.repo.LearningEvidenceForTest(f.ctx, f.Access("learner_a", false), f.knowledge)
	if e != nil || v.Qualified {
		t.Fatal("ordinary fabricated reading", v, e)
	}
	overview, e := f.repo.ReadLearningOverview(f.ctx, f.Access("learner_a", false))
	if e != nil || overview.EffectivePassedCount != 1 || overview.CompletedCount != 0 {
		t.Fatal("pass/completion conflated", overview, e)
	}
	if _, e = f.repo.StartLearning(f.ctx, f.Access("learner_a", false), f.knowledge.ID, f.startInput()); e != nil {
		t.Fatal(e)
	}
	key := f.Access("learner_a", false)
	if _, e = f.repo.CompleteLearning(f.ctx, key, f.knowledge.ID, f.completeInput()); e != nil {
		t.Fatal(e)
	}
	if _, e = f.repo.CompleteLearning(f.ctx, key, f.knowledge.ID, f.completeInput()); e != nil {
		t.Fatal(e)
	}
	v, e = f.repo.LearningEvidenceForTest(f.ctx, f.Access("learner_a", false), f.knowledge)
	if e != nil || !v.Qualified || v.Qualification.CorrectionID == nil || *v.Qualification.CorrectionID != rid || v.Qualification.CompletedEventID == nil {
		t.Fatal(v, e)
	}
	if f.count(`SELECT count(*) FROM correction_events WHERE kind='qualification_granted' AND result_id=$1`, rid) != 1 || f.count(`SELECT count(*) FROM learning_qualification_events`) != 0 {
		t.Fatal("duplicated or forged old grant")
	}
	var body []byte
	if e = f.db.QueryRow(`SELECT body->'body' FROM correction_events WHERE kind='qualification_granted' AND result_id=$1`, rid).Scan(&body); e != nil {
		t.Fatal(e)
	}
	var q learning.QualificationView
	if json.Unmarshal(body, &q) != nil || q.CompletedEventID == nil {
		t.Fatal("grant audit incomplete")
	}
}
