package store_test

import (
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"testing"
)

func ruleCaseInput(k *question.Identity, version int) correction.CaseInput {
	scope := "all"
	if k != nil {
		scope = "knowledge"
	}
	return correction.CaseInput{Kind: correction.GradingRuleCase, Rule: &correction.RuleScope{RuleVersion: version, Kind: scope, Knowledge: k}}
}
func (f *correctionFixture) registerRule(k *question.Identity, version int) correction.CaseMetadata {
	f.t.Helper()
	v, e := f.repo.CreateCorrectionCase(f.ctx, f.Access("admin_a", false), ruleCaseInput(k, version))
	if e != nil || v.Data.Case == nil {
		f.t.Fatal(v, e)
	}
	return *v.Data.Case
}
func TestCorrectionCasesImmediateRestrictionAndReplay(t *testing.T) {
	f := newCorrectionFixture(t)
	v := f.createDiagnostic("learner_a")
	submitted, e := f.repo.SubmitAssessment(f.ctx, f.Access("learner_a", false), v.Summary.ID, f.answers(v, 5))
	if e != nil || !submitted.Progress.QualificationGranted {
		t.Fatal(submitted, e)
	}
	var before []byte
	if e = f.db.QueryRow(`SELECT receipt_bytes FROM learning_idempotency WHERE action='submitAssessment' AND target=$1`, v.Summary.ID).Scan(&before); e != nil {
		t.Fatal(e)
	}
	a := f.Access("admin_a", false)
	input := ruleCaseInput(&f.knowledge, 1)
	first, e := f.repo.CreateCorrectionCase(f.ctx, a, input)
	if e != nil || first.ActorID != f.ids["admin_a"] || first.Data.Status != 201 || first.Data.Case == nil {
		t.Fatal(first, e)
	}
	second, e := f.repo.CreateCorrectionCase(f.ctx, a, input)
	b1, _ := json.Marshal(first)
	b2, _ := json.Marshal(second)
	if e != nil || string(b1) != string(b2) {
		t.Fatal("replay changed receipt", e)
	}
	if f.count(`SELECT count(*) FROM correction_cases`) != 1 || f.count(`SELECT count(*) FROM correction_jobs`) != 1 || f.count(`SELECT count(*) FROM correction_rate_limits`) != 1 {
		t.Fatal("non-atomic or duplicate registration")
	}
	var cutoff string
	if e = f.db.QueryRow(`SELECT cutoff::text FROM correction_cases WHERE id=$1`, first.Data.Case.ID).Scan(&cutoff); e != nil || first.Data.Case.Cutoff == nil || !first.Data.Case.Cutoff.Equal(first.Data.Case.CreatedAt) {
		t.Fatal("database cutoff not frozen", e)
	}
	d, e := f.repo.ReadLearningKnowledge(f.ctx, f.Access("learner_a", false), f.knowledge.ID, 1)
	if e != nil || d.State.Qualification != nil || d.Blueprints[0].Ready {
		t.Fatal("worker stopped: qualification still valid", d, e)
	}
	o, e := f.repo.ReadLearningOverview(f.ctx, f.Access("learner_a", false))
	if e != nil || o.EffectivePassedCount != 0 || o.HistoricalUnlockedCount < 1 {
		t.Fatal("projection or historical unlock lost", o, e)
	}
	r, e := f.repo.ReadAssessmentResult(f.ctx, f.Access("learner_a", false), v.Summary.ID)
	if e != nil || r.Outcome != assessment.Passed || r.Score == nil || *r.Score != 5 || r.Validity != assessment.Restricted || r.Items[0].Correct != nil || r.Items[0].Explanation != nil {
		t.Fatal("original fact changed or answer leaked", r, e)
	}
	h, e := f.repo.ListLearningHistory(f.ctx, f.Access("learner_a", false), learning.ListQuery{})
	if e != nil || len(h.Items) != 1 || h.Items[0].Validity != assessment.Restricted {
		t.Fatal("history ignores rule", h, e)
	}
	var after []byte
	if e = f.db.QueryRow(`SELECT receipt_bytes FROM learning_idempotency WHERE action='submitAssessment' AND target=$1`, v.Summary.ID).Scan(&after); e != nil || string(before) != string(after) {
		t.Fatal("original receipt changed", e)
	}
	changed := ruleCaseInput(nil, 1)
	if _, e = f.repo.CreateCorrectionCase(f.ctx, a, changed); !errors.Is(e, question.ErrIdempotencyConflict) {
		t.Fatal("same key changed input", e)
	}
	if _, e = f.repo.CreateCorrectionCase(f.ctx, f.Access("author_a", false), input); !errors.Is(e, auth.ErrForbidden) {
		t.Fatal("editor registered rule", e)
	}
}
func TestCorrectionCutoffLateSubmissionAndNewAttempt(t *testing.T) {
	f := newCorrectionFixture(t)
	v := f.createDiagnostic("learner_a")
	answers := f.answers(v, 5)
	c := f.registerRule(nil, 1)
	if !v.Summary.CreatedAt.Before(*c.Cutoff) && !v.Summary.CreatedAt.Equal(*c.Cutoff) {
		t.Fatal("invalid test chronology")
	}
	r, e := f.repo.SubmitAssessment(f.ctx, f.Access("learner_a", false), v.Summary.ID, answers)
	if e != nil || r.Outcome != assessment.Affected || r.Score != nil || r.Passed != nil || r.Progress.QualificationGranted || f.count(`SELECT count(*) FROM assessment_answers`) != 5 {
		t.Fatal("late submit escaped cutoff", r, e)
	}
	if _, e = f.repo.CreateAssessment(f.ctx, f.Access("learner_b", false), f.assessmentInput(assessment.ModeDiagnostic)); !errors.Is(e, learning.ErrAssessmentNotReady) {
		t.Fatal("post-cutoff attempt opened", e)
	}
}
func TestCorrectionCasesScopeAndUnknownRule(t *testing.T) {
	f := newCorrectionFixture(t)
	f.publishLearningGraph()
	original := f.knowledge
	f.registerRule(&original, 2)
	v := f.createDiagnostic("learner_a")
	r, e := f.repo.SubmitAssessment(f.ctx, f.Access("learner_a", false), v.Summary.ID, f.answers(v, 5))
	if e != nil || r.Outcome != assessment.Passed {
		t.Fatal("unknown rule damaged rule one", e)
	}
	f.registerRule(&original, 1)
	f.setLearningTarget("workflow-dependent", "lf-workflow-dependent-five", 0)
	if _, e = f.repo.CreateAssessment(f.ctx, f.Access("learner_b", false), f.assessmentInput(assessment.ModeDiagnostic)); e != nil {
		t.Fatal("knowledge scope damaged other knowledge", e)
	}
	if _, e = f.repo.CreateCorrectionCase(f.ctx, f.Access("admin_a", false), ruleCaseInput(&question.Identity{ID: original.ID, Version: 1, SHA256: string(make([]byte, 64))}, 1)); !errors.Is(e, auth.ErrInvalidInput) {
		t.Fatal(e)
	}
}
