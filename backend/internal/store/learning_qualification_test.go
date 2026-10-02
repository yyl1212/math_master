package store_test

import (
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"testing"
	"time"
)

func (f *learningFixture) passedFixture(actor string, mode assessment.Mode) assessment.AttemptFact {
	f.t.Helper()
	prior := f.mode
	f.mode = &mode
	defer func() { f.mode = prior }()
	seal := f.seal("assessment")
	items, e := f.repo.LearningLoadItemsForTest(f.ctx, f.Access(actor, false), seal)
	if e != nil {
		f.t.Fatal(e)
	}
	answers := assessment.SubmitInput{Answers: []assessment.PositionAnswer{}}
	for j, i := range items {
		raw := i.Body.CorrectNumeric.Numerator + "/" + i.Body.CorrectNumeric.Denominator
		answers.Answers = append(answers.Answers, assessment.PositionAnswer{Position: j + 1, Instance: i.Identity, Answer: assessment.Answer{Kind: "numeric", Raw: &raw}})
	}
	score, passed, e := assessment.GradeFive(items, answers)
	if e != nil || score != 5 || !passed {
		f.t.Fatal("synthetic passed fixture was not mathematically valid", score, passed, e)
	}
	tx, e := f.db.Begin()
	if e != nil {
		f.t.Fatal(e)
	}
	defer tx.Rollback()
	id := f.ID()
	if e = f.insertAssessment(tx, id, f.ids[actor], 5); e != nil {
		f.t.Fatal(e)
	}
	for _, a := range answers.Answers {
		raw, _ := json.Marshal(a.Answer)
		if _, e = tx.Exec(`INSERT INTO assessment_answers(attempt_id,owner_user_id,position,instance_id,instance_version,instance_sha256,answer) VALUES($1,$2,$3,$4,$5,$6,$7)`, id, f.ids[actor], a.Position, a.Instance.ID, a.Instance.Version, a.Instance.SHA256, string(raw)); e != nil {
			f.t.Fatal(e)
		}
	}
	var now time.Time
	if e = tx.QueryRow(`SELECT clock_timestamp()`).Scan(&now); e != nil {
		f.t.Fatal(e)
	}
	raw, _ := json.Marshal(assessment.ProgressUpdate{Knowledge: f.knowledge, NewlyUnlocked: []question.Identity{}})
	if _, e = tx.Exec(`INSERT INTO assessment_results(attempt_id,owner_user_id,rule_version,outcome,score,passed,original_reasons,progress,submitted_at) VALUES($1,$2,1,'passed',5,true,'[]',$3,$4)`, id, f.ids[actor], string(raw), now); e != nil {
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
	return assessment.AttemptFact{ID: id, Knowledge: f.knowledge, Mode: mode, Outcome: assessment.Passed, Score: &score, Passed: &passed, SubmittedAt: now, Validity: assessment.Effective}
}
func TestLearningQualificationBothActionOrders(t *testing.T) {
	for _, first := range []string{"reading", "assessment"} {
		t.Run(first, func(t *testing.T) {
			f := newLearningFixture(t)
			var fact assessment.AttemptFact
			if first == "assessment" {
				fact = f.passedFixture("learner_a", assessment.ModeNode)
				v, e := f.repo.LearningEvidenceForTest(f.ctx, f.Access("learner_a", false), f.knowledge)
				if e != nil || v.Qualified {
					t.Fatal("ordinary pass fabricated explicit completion", v, e)
				}
			}
			if _, e := f.repo.StartLearning(f.ctx, f.Access("learner_a", false), f.knowledge.ID, f.startInput()); e != nil {
				t.Fatal(e)
			}
			if _, e := f.repo.CompleteLearning(f.ctx, f.Access("learner_a", false), f.knowledge.ID, f.completeInput()); e != nil {
				t.Fatal(e)
			}
			if first == "reading" {
				fact = f.passedFixture("learner_a", assessment.ModeNode)
			}
			p, e := f.repo.LearningApplyForTest(f.ctx, f.Access("learner_a", false), fact)
			if e != nil || !p.QualificationGranted {
				t.Fatal(p, e)
			}
			v, e := f.repo.LearningEvidenceForTest(f.ctx, f.Access("learner_a", false), f.knowledge)
			if e != nil || !v.Qualified || v.Qualification.Kind != "normal" || v.Qualification.CompletedEventID == nil {
				t.Fatal(v, e)
			}
			bad := fact
			n := 4
			bad.Score = &n
			if _, e = f.repo.LearningApplyForTest(f.ctx, f.Access("learner_a", false), bad); e == nil {
				t.Fatal("unpersisted score accepted")
			}
		})
	}
}
func TestLearningAlternativeEvidenceRetainsValidPass(t *testing.T) {
	f := newLearningFixture(t)
	old := f.passedFixture("learner_a", assessment.ModeDiagnostic)
	p, e := f.repo.LearningApplyForTest(f.ctx, f.Access("learner_a", false), old)
	if e != nil || !p.QualificationGranted || f.count(`SELECT count(*) FROM learning_records`) != 0 {
		t.Fatal("diagnostic fabricated reading", p, e)
	}
	pool, e := f.repo.LearningSourcePoolForTest(f.ctx, f.Access("learner_a", false), f.knowledge, &f.blueprint)
	if e != nil {
		t.Fatal(e)
	}
	for j := range f.items {
		f.items[j].Instance = pool.Candidates[j+5].Identity
	}
	latest := f.passedFixture("learner_a", assessment.ModeDiagnostic)
	if _, e = f.repo.LearningApplyForTest(f.ctx, f.Access("learner_a", false), latest); e != nil {
		t.Fatal(e)
	}
	f.QWithdraw(question.WithdrawalTarget{Kind: "instance", ID: f.items[0].Instance.ID, Version: 1})
	v, e := f.repo.LearningEvidenceForTest(f.ctx, f.Access("learner_a", false), f.knowledge)
	if e != nil || !v.Qualified || v.Qualification.EvidenceAttemptID != old.ID {
		t.Fatal("latest invalidation discarded older effective pass", v, e)
	}
	f.setLearningTarget("workflow-fractions", "lf-five", 0)
	mode := assessment.ModeReview
	f.mode = &mode
	f.submitSkippedFixture("learner_a")
	got, e := f.repo.StartLearning(f.ctx, f.Access("learner_a", false), f.knowledge.ID, f.startInput())
	if e != nil || got.State != learning.NeedsReview || got.Qualification == nil || !got.EverUnlocked {
		t.Fatal("failed review erased historical unlock or valid alternative", got, e)
	}
}
