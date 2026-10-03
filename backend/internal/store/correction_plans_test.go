package store_test

import (
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/question"
	"strings"
	"testing"
)

func correctionPlanInput() correction.PlanInput {
	return correction.PlanInput{AlgorithmVersion: 1, Mappings: []correction.Mapping{}, Reason: "Original independently reviewed exact regrading basis."}
}
func correctionReceiptJSON(t *testing.T, v any) string {
	t.Helper()
	b, e := json.Marshal(v)
	if e != nil {
		t.Fatal(e)
	}
	return string(b)
}
func (f *correctionFixture) createCorrectionPlan(caseID, actor string) correction.PlanMetadata {
	f.t.Helper()
	v, e := f.repo.CreateCorrectionPlan(f.ctx, f.Access(actor, false), caseID, correctionPlanInput())
	if e != nil || v.Data.Plan == nil {
		f.t.Fatal(v, e)
	}
	return *v.Data.Plan
}
func (f *correctionFixture) submitCorrectionPlan(p correction.PlanMetadata, actor string) correction.PlanMetadata {
	f.t.Helper()
	v, e := f.repo.SubmitCorrectionPlan(f.ctx, f.Access(actor, false), p.Ref, correction.SubmitInput{ExpectedSequence: p.Sequence})
	if e != nil || v.Data.Plan == nil {
		f.t.Fatal(v, e)
	}
	return *v.Data.Plan
}
func TestCorrectionPlanVersionFreezesAndRejectCreatesChild(t *testing.T) {
	f := newCorrectionFixture(t)
	c := f.registerRule(nil, 1)
	p := f.createCorrectionPlan(c.ID, "author_b")
	if p.Status != correction.Draft || p.Ref.Version != 1 || p.Sequence != 1 || p.Digest != nil {
		t.Fatal(p)
	}
	in := correctionPlanInput()
	in.ExpectedSequence = &p.Sequence
	in.Reason = "A revised original correction draft."
	updated, e := f.repo.UpdateCorrectionPlan(f.ctx, f.Access("author_b", false), p.Ref, in)
	if e != nil || updated.Data.Plan.Sequence != 2 {
		t.Fatal(updated, e)
	}
	p = *updated.Data.Plan
	if _, e = f.repo.UpdateCorrectionPlan(f.ctx, f.Access("author_a", false), p.Ref, in); !errors.Is(e, auth.ErrForbidden) {
		t.Fatal("other editor edited draft", e)
	}
	p = f.submitCorrectionPlan(p, "author_b")
	var before []byte
	if e = f.db.QueryRow(`SELECT frozen_bytes FROM correction_plans WHERE id=$1 AND version=1`, p.Ref.ID).Scan(&before); e != nil {
		t.Fatal(e)
	}
	in.ExpectedSequence = &p.Sequence
	if _, e = f.repo.UpdateCorrectionPlan(f.ctx, f.Access("author_b", false), p.Ref, in); !errors.Is(e, correction.ErrConflict) {
		t.Fatal("submitted body changed", e)
	}
	if _, e = f.repo.DecideCorrectionPlan(f.ctx, f.Access("reviewer_b", false), p.Ref, correction.DecisionInput{ExpectedSequence: p.Sequence, Decision: "reject", Reason: "The exact basis requires another independently reviewed version."}); e != nil {
		t.Fatal(e)
	}
	child := correctionPlanInput()
	child.Parent = &p.Ref
	next, e := f.repo.CreateCorrectionPlan(f.ctx, f.Access("author_b", false), c.ID, child)
	if e != nil || next.Data.Plan.Ref.ID != p.Ref.ID || next.Data.Plan.Ref.Version != 2 || next.Data.Plan.Sequence != 1 {
		t.Fatal("new version did not preserve lineage", next, e)
	}
	var after []byte
	if e = f.db.QueryRow(`SELECT frozen_bytes FROM correction_plans WHERE id=$1 AND version=1`, p.Ref.ID).Scan(&after); e != nil || string(before) != string(after) {
		t.Fatal("frozen rejected version changed", e)
	}
	child.Parent = &next.Data.Plan.Ref
	if _, e = f.repo.CreateCorrectionPlan(f.ctx, f.Access("author_b", false), c.ID, child); !errors.Is(e, correction.ErrConflict) {
		t.Fatal("draft parent accepted", e)
	}
}
func TestCorrectionIndependentReviewAllSourcesAndOldCutoff(t *testing.T) {
	f := newCorrectionFixture(t)
	v := f.createDiagnostic("learner_a")
	answers := f.answers(v, 5)
	c := f.registerRule(nil, 1)
	if _, e := f.repo.SubmitAssessment(f.ctx, f.Access("learner_a", false), v.Summary.ID, answers); e != nil {
		t.Fatal(e)
	}
	p := f.submitCorrectionPlan(f.createCorrectionPlan(c.ID, "author_b"), "author_b")
	decision := correction.DecisionInput{ExpectedSequence: p.Sequence, Decision: "approve", Reason: "The registered exact grading algorithm and all mathematical sources have been independently verified."}
	for _, actor := range []string{"author_b", "author_a"} {
		f.exec(`INSERT INTO auth_user_roles(user_id,role) VALUES($1,'reviewer') ON CONFLICT DO NOTHING`, f.ids[actor])
		if _, e := f.repo.DecideCorrectionPlan(f.ctx, f.Access(actor, false), p.Ref, decision); !errors.Is(e, auth.ErrForbidden) {
			t.Fatal("creator or mathematical author self-reviewed", actor, e)
		}
	}
	if f.count(`SELECT count(*) FROM correction_jobs WHERE type='approved_plan'`) != 0 {
		t.Fatal("failed decision queued plan")
	}
	key := f.Access("reviewer_b", false)
	approved, e := f.repo.DecideCorrectionPlan(f.ctx, key, p.Ref, decision)
	if e != nil || approved.Data.Plan.Status != correction.Approved {
		t.Fatal(approved, e)
	}
	raw := correctionReceiptJSON(t, approved)
	if strings.Contains(raw, decision.Reason) || strings.Contains(raw, "reason\"") {
		t.Fatal("receipt leaked free reason")
	}
	replay, e := f.repo.DecideCorrectionPlan(f.ctx, key, p.Ref, decision)
	if e != nil || raw != correctionReceiptJSON(t, replay) {
		t.Fatal("decision replay changed", e)
	}
	if f.count(`SELECT count(*) FROM correction_jobs WHERE type='approved_plan'`) != 1 || f.count(`SELECT count(*) FROM correction_events WHERE kind='plan_approved'`) != 1 {
		t.Fatal("approval outbox duplicated")
	}
	if _, e = f.repo.CreateAssessment(f.ctx, f.Access("learner_b", false), f.assessmentInput(assessment.ModeDiagnostic)); e != nil {
		t.Fatal("independent registered basis did not allow new attempt", e)
	}
	d, e := f.repo.ReadLearningKnowledge(f.ctx, f.Access("learner_a", false), f.knowledge.ID, 1)
	if e != nil || d.State.Qualification != nil {
		t.Fatal("approval alone restored old evidence", e)
	}
	f.exec(`DELETE FROM auth_user_roles WHERE user_id=$1 AND role='reviewer'`, f.ids["reviewer_b"])
	if _, e = f.repo.DecideCorrectionPlan(f.ctx, key, p.Ref, decision); !errors.Is(e, auth.ErrForbidden) {
		t.Fatal("revoked reviewer replayed", e)
	}
}
func TestCorrectionReplayAfterAdvanceOriginalReceipt(t *testing.T) {
	f := newCorrectionFixture(t)
	c := f.registerRule(nil, 1)
	key := f.Access("author_b", false)
	in := correctionPlanInput()
	created, e := f.repo.CreateCorrectionPlan(f.ctx, key, c.ID, in)
	if e != nil {
		t.Fatal(e)
	}
	p := *created.Data.Plan
	updateKey := f.Access("author_b", false)
	in.ExpectedSequence = &p.Sequence
	in.Reason = "An updated frozen draft with original wording."
	updated, e := f.repo.UpdateCorrectionPlan(f.ctx, updateKey, p.Ref, in)
	if e != nil {
		t.Fatal(e)
	}
	submitted := f.submitCorrectionPlan(*updated.Data.Plan, "author_b")
	if _, e = f.repo.DecideCorrectionPlan(f.ctx, f.Access("reviewer_b", false), p.Ref, correction.DecisionInput{ExpectedSequence: submitted.Sequence, Decision: "approve", Reason: "Independently verified exact correction basis."}); e != nil {
		t.Fatal(e)
	}
	again, e := f.repo.CreateCorrectionPlan(f.ctx, key, c.ID, correctionPlanInput())
	if e != nil || correctionReceiptJSON(t, again) != correctionReceiptJSON(t, created) {
		t.Fatal("create replay read current sequence", e)
	}
	again, e = f.repo.UpdateCorrectionPlan(f.ctx, updateKey, p.Ref, in)
	if e != nil || correctionReceiptJSON(t, again) != correctionReceiptJSON(t, updated) {
		t.Fatal("update replay changed after approval", e)
	}
	in.Reason = "Different valid body with the same original command key."
	if _, e = f.repo.UpdateCorrectionPlan(f.ctx, updateKey, p.Ref, in); !errors.Is(e, question.ErrIdempotencyConflict) {
		t.Fatal("changed input replay allowed", e)
	}
	if f.count(`SELECT count(*) FROM correction_events WHERE subject_kind='plan'`) != 4 || f.count(`SELECT count(*) FROM correction_rate_limits WHERE scope='process'`) != 3 {
		t.Fatal("failed or repeated command consumed quota")
	}
	bad := correctionPlanInput()
	bad.AlgorithmVersion = 2
	if _, e = f.repo.CreateCorrectionPlan(f.ctx, f.Access("author_b", false), c.ID, bad); !errors.Is(e, auth.ErrInvalidInput) {
		t.Fatal("unregistered algorithm accepted", e)
	}

}

func TestCorrectionPlanAdminMayEditWithoutChangingCreator(t *testing.T) {
	f := newCorrectionFixture(t)
	c := f.registerRule(nil, 1)
	p := f.createCorrectionPlan(c.ID, "author_b")
	in := correctionPlanInput()
	in.ExpectedSequence = &p.Sequence
	in.Reason = "The administrator makes an original draft edit for the existing creator."
	updated, e := f.repo.UpdateCorrectionPlan(f.ctx, f.Access("admin_a", false), p.Ref, in)
	if e != nil || updated.Data.Plan.Sequence != 2 {
		t.Fatal("authorized admin edit refused", e)
	}
	var creator string
	if e = f.db.QueryRow(`SELECT creator_user_id::text FROM correction_plans WHERE id=$1 AND version=1`, p.Ref.ID).Scan(&creator); e != nil || creator != f.ids["author_b"] {
		t.Fatal("admin changed creator", e)
	}
	submitted := f.submitCorrectionPlan(*updated.Data.Plan, "author_b")
	decision := correction.DecisionInput{ExpectedSequence: submitted.Sequence, Decision: "approve", Reason: "An administrator who edited the draft must not act as its independent reviewer."}
	if _, e = f.repo.DecideCorrectionPlan(f.ctx, f.Access("admin_a", false), p.Ref, decision); !errors.Is(e, auth.ErrForbidden) {
		t.Fatal("draft editor self reviewed", e)
	}
	if _, e = f.repo.DecideCorrectionPlan(f.ctx, f.Access("reviewer_b", false), p.Ref, decision); e != nil {
		t.Fatal("independent reviewer refused", e)
	}
}
