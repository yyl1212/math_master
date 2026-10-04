package store_test

import (
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/question"
	"testing"
)

func TestCorrectionLateTerminalAfterRootDone(t *testing.T) {
	f := newCorrectionFixture(t)
	v := f.createDiagnostic("learner_a")
	c := f.registerRule(nil, 1)
	root, e := f.repo.ClaimCorrectionJob(f.ctx)
	if e != nil || root == nil {
		t.Fatal(e)
	}
	if e = f.repo.FinishCorrectionJob(f.ctx, *root, correction.Succeeded, ""); e != nil {
		t.Fatal(e)
	}
	result, e := f.repo.SubmitAssessment(f.ctx, f.Access("learner_a", false), v.Summary.ID, f.answers(v, 5))
	if e != nil || result.Outcome != assessment.Affected {
		t.Fatal(result, e)
	}
	if f.count(`SELECT count(*) FROM correction_jobs WHERE case_id=$1 AND type='attempt_terminal' AND evidence_id=$2`, c.ID, v.Summary.ID) != 1 {
		t.Fatal("root success hid late terminal")
	}
	n, e := f.repo.BackfillCorrections(f.ctx, 50)
	if e != nil || n != 0 {
		t.Fatal("backfill duplicated outbox", n, e)
	}
}
func TestCorrectionLegacyTerminalBackfillAndStableLowKey(t *testing.T) {
	f := newCorrectionFixture(t)
	v := f.createDiagnostic("learner_a")
	if _, e := f.repo.SubmitAssessment(f.ctx, f.Access("learner_a", false), v.Summary.ID, f.answers(v, 5)); e != nil {
		t.Fatal(e)
	}
	c := f.registerRule(nil, 1)
	if f.count(`SELECT count(*) FROM correction_jobs WHERE type='attempt_terminal'`) != 0 {
		t.Fatal("test needs pre-registration terminal")
	}
	n, e := f.repo.BackfillCorrections(f.ctx, 1)
	if e != nil || n != 1 {
		t.Fatal("legacy terminal missed", n, e)
	}
	if f.count(`SELECT count(*) FROM correction_jobs WHERE case_id=$1 AND type='attempt_terminal' AND evidence_id=$2`, c.ID, v.Summary.ID) != 1 {
		t.Fatal("legacy individual evidence key missing")
	}
	n, e = f.repo.BackfillCorrections(f.ctx, 1)
	if e != nil || n != 0 {
		t.Fatal("terminal backfill not idempotent", n, e)
	}
}
func TestCorrectionRotatingBackfillDoesNotStarveCases(t *testing.T) {
	f := newCorrectionFixture(t)
	v := f.createDiagnostic("learner_a")
	if _, e := f.repo.SubmitAssessment(f.ctx, f.Access("learner_a", false), v.Summary.ID, f.answers(v, 5)); e != nil {
		t.Fatal(e)
	}
	ids := []string{}
	for i := 0; i < 6; i++ {
		ids = append(ids, f.registerRule(nil, 1).ID)
	}
	for i := 0; i < 6; i++ {
		n, e := f.repo.BackfillCorrections(f.ctx, 1)
		if e != nil || n != 1 {
			t.Fatal("one-case rotation did not progress", i, n, e)
		}
	}
	if f.count(`SELECT count(DISTINCT case_id) FROM correction_jobs WHERE type='attempt_terminal'`) != 6 {
		t.Fatal("old cases starved newer ones")
	}
	for _, id := range ids {
		if f.count(`SELECT count(*) FROM correction_cases WHERE id=$1 AND last_backfill_at IS NOT NULL`, id) != 1 {
			t.Fatal("case rotation not persisted")
		}
	}
	n, e := f.repo.BackfillCorrections(f.ctx, 50)
	if e != nil || n != 0 {
		t.Fatal(n, e)
	}
	if _, e = f.repo.BackfillCorrections(f.ctx, 51); e == nil {
		t.Fatal("oversized global registration batch allowed")
	}
}

// An actual immutable withdrawal made by a legacy writer, without P5b outbox.
func (f *correctionFixture) legacyCorrectionWithdrawal(i question.Identity) string {
	f.t.Helper()
	id := f.ID()
	f.exec(`INSERT INTO question_withdrawals(id,kind,target_id,target_version,sha256,actor_user_id,reason,request_id) VALUES($1,'instance',$2,$3,$4,$5,'Original legacy withdrawal with accurate approved identity.','correction-legacy')`, id, i.ID, i.Version, i.SHA256, f.ids["admin_a"])
	return id
}
func TestCorrectionLegacyTerminalBackfillRootPlanSharedBudget(t *testing.T) {
	f := newCorrectionFixture(t)
	aid, b := f.cSubmittedBasis("learner_a")
	wid := f.legacyCorrectionWithdrawal(b.OriginalItems[0].Identity)
	n, e := f.repo.BackfillCorrections(f.ctx, 1)
	if e != nil || n != 1 {
		t.Fatal(n, e)
	}
	if f.count(`SELECT count(*) FROM correction_jobs`) != 1 {
		t.Fatal("global limit did not bound orphan root")
	}
	n, e = f.repo.BackfillCorrections(f.ctx, 1)
	if e != nil || n != 1 {
		t.Fatal(n, e)
	}
	if f.count(`SELECT count(*) FROM correction_jobs WHERE type='attempt_terminal' AND evidence_id=$1`, aid) != 1 {
		t.Fatal("legacy withdrawal terminal missed")
	}
	var cid string
	if e = f.db.QueryRow(`SELECT id::text FROM correction_cases WHERE withdrawal_id=$1`, wid).Scan(&cid); e != nil {
		t.Fatal(e)
	}
	// Independent approved legacy plans also lacked their own unique task.
	cid2 := f.cCase()
	plan := f.cDraft(cid2)
	if e = f.cPending(cid2, plan); e != nil {
		t.Fatal(e)
	}
	if e = correctionApprovalFixture(f, cid2, plan, "reviewer_b"); e != nil {
		t.Fatal(e)
	}
	total := 0
	for step := 0; step < 6 && total < 3; step++ {
		n, e = f.repo.BackfillCorrections(f.ctx, 1)
		if e != nil || n < 0 || n > 1 {
			t.Fatal(step, n, e)
		}
		total += n
	}
	if total != 3 {
		t.Fatal("rotation starved incomplete root/plan/terminal", total)
	}
	if f.count(`SELECT count(*) FROM correction_jobs WHERE case_id=$1`, cid2) != 3 {
		t.Fatal("root, approved plan, terminal not sharing finite budget")
	}
	n, e = f.repo.BackfillCorrections(f.ctx, 50)
	if e != nil || n != 0 {
		t.Fatal(n, e)
	}
}
