package store_test

import (
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"testing"
	"time"
)

func (f *correctionFixture) cApproved(caseID string) correction.PlanRef {
	f.t.Helper()
	p := f.cDraft(caseID)
	if e := f.cPending(caseID, p); e != nil {
		f.t.Fatal(e)
	}
	if e := correctionApprovalFixture(f, caseID, p, "reviewer_b"); e != nil {
		f.t.Fatal(e)
	}
	return p
}
func (f *correctionFixture) cSealResult(aid, cid string, p correction.PlanRef, b correction.Basis) string {
	f.t.Helper()
	id, e := f.cResult(aid, f.ids["learner_a"], cid, p, b, true)
	if e != nil {
		f.t.Fatal(e)
	}
	return id
}
func TestCorrectionProjectionDiagnosticAllViewsOriginalBytes(t *testing.T) {
	f := newCorrectionFixture(t)
	aid, b := f.cSubmittedBasis("learner_a")
	var old string
	if e := f.db.QueryRow(`SELECT jsonb_build_object('attempt',a.seal,'answers',(SELECT jsonb_agg(answer ORDER BY position) FROM assessment_answers WHERE attempt_id=a.id),'result',to_jsonb(r),'receipt',(SELECT jsonb_agg(encode(receipt_bytes,'hex')) FROM learning_idempotency WHERE receipt->>'resourceId'=a.id::text))::text FROM assessment_attempts a JOIN assessment_results r ON r.attempt_id=a.id WHERE a.id=$1`, aid).Scan(&old); e != nil {
		t.Fatal(e)
	}
	cid := f.cCase()
	p := f.cApproved(cid)
	b.HandledCaseIDs = []string{cid}
	b.PlanRefs = []correction.PlanRef{p}
	rid := f.cSealResult(aid, cid, p, b)
	v, e := f.repo.LearningEvidenceForTest(f.ctx, f.Access("learner_a", false), f.knowledge)
	if e != nil || !v.Qualified || v.Qualification.CorrectionID == nil || *v.Qualification.CorrectionID != rid || v.Qualification.CompletedEventID != nil {
		t.Fatal("correction not in current qualification", v, e)
	}
	overview, e := f.repo.ReadLearningOverview(f.ctx, f.Access("learner_a", false))
	if e != nil || overview.EffectivePassedCount != 1 {
		t.Fatal("overview omitted correction", overview, e)
	}
	state, e := f.repo.ReadLearningKnowledge(f.ctx, f.Access("learner_a", false), f.knowledge.ID, 1)
	if e != nil || state.State.State != learning.Mastered {
		t.Fatal("knowledge not passed", state, e)
	}
	if f.count(`SELECT count(*) FROM learning_records`) != 0 {
		t.Fatal("diagnostic fabricated reading")
	}
	var fresh string
	if e = f.db.QueryRow(`SELECT jsonb_build_object('attempt',a.seal,'answers',(SELECT jsonb_agg(answer ORDER BY position) FROM assessment_answers WHERE attempt_id=a.id),'result',to_jsonb(r),'receipt',(SELECT jsonb_agg(encode(receipt_bytes,'hex')) FROM learning_idempotency WHERE receipt->>'resourceId'=a.id::text))::text FROM assessment_attempts a JOIN assessment_results r ON r.attempt_id=a.id WHERE a.id=$1`, aid).Scan(&fresh); e != nil || old != fresh {
		t.Fatal("original changed", e)
	}
}
func TestCorrectionOtherCaseStillRestricts(t *testing.T) {
	f := newCorrectionFixture(t)
	aid, b := f.cSubmittedBasis("learner_a")
	cid := f.cCase()
	p := f.cApproved(cid)
	b.HandledCaseIDs = []string{cid}
	b.PlanRefs = []correction.PlanRef{p}
	f.cSealResult(aid, cid, p, b)
	other := f.cCase()
	v, e := f.repo.LearningEvidenceForTest(f.ctx, f.Access("learner_a", false), f.knowledge)
	if e != nil || v.Qualified {
		t.Fatal("unhandled case hidden", v, e)
	}
	p2 := f.cApproved(other)
	b.HandledCaseIDs = append(b.HandledCaseIDs, other)
	b.PlanRefs = append(b.PlanRefs, p2)
	f.cSealResult(aid, other, p2, b)
	v, e = f.repo.LearningEvidenceForTest(f.ctx, f.Access("learner_a", false), f.knowledge)
	if e != nil || !v.Qualified {
		t.Fatal("cumulative cases not restored", v, e)
	}
	f.legacyCorrectionWithdrawal(b.EffectiveItems[0].Identity)
	v, e = f.repo.LearningEvidenceForTest(f.ctx, f.Access("learner_a", false), f.knowledge)
	if e != nil || v.Qualified {
		t.Fatal("withdrawn effective instance passed", v, e)
	}
}
func TestCorrectionProjectionSupersededPassNotFallback(t *testing.T) {
	f := newCorrectionFixture(t)
	aid, b := f.cSubmittedBasis("learner_a")
	cid := f.cCase()
	p := f.cApproved(cid)
	b.HandledCaseIDs = []string{cid}
	b.PlanRefs = []correction.PlanRef{p}
	parent := f.cSealResult(aid, cid, p, b)
	child := b
	child.ParentResultID = &parent
	child.ParentResultIDs = []string{parent}
	child.EffectiveItems = child.EffectiveItems[:4]
	f.cSealResult(aid, cid, p, child)
	v, e := f.repo.LearningEvidenceForTest(f.ctx, f.Access("learner_a", false), f.knowledge)
	if e != nil || v.Qualified {
		t.Fatal("superseded ancestor became fallback", v, e)
	}
	// A separate current original attempt remains valid after the cutoff.
	fresh := f.createDiagnostic("learner_a")
	if _, e = f.repo.SubmitAssessment(f.ctx, f.Access("learner_a", false), fresh.Summary.ID, f.answers(fresh, 5)); e != nil {
		t.Fatal(e)
	}
	v, e = f.repo.LearningEvidenceForTest(f.ctx, f.Access("learner_a", false), f.knowledge)
	if e != nil || !v.Qualified || v.Qualification.EvidenceAttemptID != fresh.Summary.ID || v.Qualification.CorrectionID != nil {
		t.Fatal("independent attempt lost", v, e)
	}
}

// This models an immutable historical grading defect: four mathematically
// correct original answers, but the legacy writer recorded three, hence failed.
func (f *correctionFixture) cLegacyFailedFour() (string, correction.Basis) {
	f.t.Helper()
	seal := f.seal("assessment")
	items, e := f.repo.LearningLoadItemsForTest(f.ctx, f.Access("learner_a", false), seal)
	if e != nil {
		f.t.Fatal(e)
	}
	tx, e := f.db.Begin()
	if e != nil {
		f.t.Fatal(e)
	}
	defer tx.Rollback()
	id := f.ID()
	if e = f.insertAssessment(tx, id, f.ids["learner_a"], 5); e != nil {
		f.t.Fatal(e)
	}
	for pos, i := range items {
		raw := i.Body.CorrectNumeric.Numerator + "/" + i.Body.CorrectNumeric.Denominator
		a := assessment.Answer{Kind: "numeric", Raw: &raw}
		if pos == 4 {
			a = assessment.Answer{Kind: "skipped"}
		}
		if _, e = tx.Exec(`INSERT INTO assessment_answers(attempt_id,owner_user_id,position,instance_id,instance_version,instance_sha256,answer) VALUES($1,$2,$3,$4,$5,$6,$7)`, id, f.ids["learner_a"], pos+1, i.Identity.ID, i.Identity.Version, i.Identity.SHA256, corrJSON(a)); e != nil {
			f.t.Fatal(e)
		}
	}
	var now time.Time
	if e = tx.QueryRow(`SELECT clock_timestamp()`).Scan(&now); e != nil {
		f.t.Fatal(e)
	}
	progress := assessment.ProgressUpdate{Knowledge: f.knowledge, NewlyUnlocked: []question.Identity{}}
	if _, e = tx.Exec(`INSERT INTO assessment_results(attempt_id,owner_user_id,rule_version,outcome,score,passed,original_reasons,progress,submitted_at) VALUES($1,$2,1,'failed',3,false,'[]',$3,$4)`, id, f.ids["learner_a"], corrJSON(progress), now); e != nil {
		f.t.Fatal(e)
	}
	if _, e = tx.Exec(`UPDATE assessment_results SET sealed=true WHERE attempt_id=$1`, id); e != nil {
		f.t.Fatal(e)
	}
	if _, e = tx.Exec(`UPDATE assessment_attempts SET state='submitted',terminal_at=$2,submission_exposure_sequence=creation_exposure_sequence WHERE id=$1`, id, now); e != nil {
		f.t.Fatal(e)
	}
	if e = tx.Commit(); e != nil {
		f.t.Fatal(e)
	}
	return id, f.cLoadBasis(id)
}
func TestCorrectionProjectionOriginalFailedFourDoesNotForgeOldQualification(t *testing.T) {
	f := newCorrectionFixture(t)
	aid, b := f.cLegacyFailedFour()
	cid := f.cCase()
	p := f.cApproved(cid)
	b.HandledCaseIDs = []string{cid}
	b.PlanRefs = []correction.PlanRef{p}
	rid := f.cSealResult(aid, cid, p, b)
	v, e := f.repo.LearningEvidenceForTest(f.ctx, f.Access("learner_a", false), f.knowledge)
	if e != nil || !v.Qualified || v.Qualification.CorrectionID == nil || *v.Qualification.CorrectionID != rid {
		t.Fatal(v, e)
	}
	if f.count(`SELECT count(*) FROM correction_results WHERE id=$1 AND score=4 AND passed`, rid) != 1 || f.count(`SELECT count(*) FROM assessment_results WHERE attempt_id=$1 AND score=3 AND NOT passed AND outcome='failed'`, aid) != 1 {
		t.Fatal("original/new score conflated")
	}
	if _, e = f.db.Exec(`INSERT INTO learning_qualification_events(id,owner_user_id,knowledge_id,knowledge_version,knowledge_sha256,kind,attempt_id,created_at) VALUES($1,$2,$3,$4,$5,'diagnostic',$6,clock_timestamp())`, f.ID(), f.ids["learner_a"], f.knowledge.ID, f.knowledge.Version, f.knowledge.SHA256, aid); e == nil {
		t.Fatal("old failed qualification guard replaced")
	}
}
func TestCorrectionProjectionIncompatibleApprovedLeavesRestrict(t *testing.T) {
	f := newCorrectionFixture(t)
	aid, b := f.cSubmittedBasis("learner_a")
	cid := f.cCase()
	p := f.cApproved(cid)
	b.HandledCaseIDs = []string{cid}
	b.PlanRefs = []correction.PlanRef{p}
	f.cSealResult(aid, cid, p, b)
	sibling := b
	sibling.EffectiveItems = sibling.EffectiveItems[:4]
	f.cSealResult(aid, cid, p, sibling)
	v, e := f.repo.LearningEvidenceForTest(f.ctx, f.Access("learner_a", false), f.knowledge)
	if e != nil || v.Qualified {
		t.Fatal("latest timestamp selected incompatible branch", v, e)
	}
}
