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
	"time"
)

func (f *learningFixture) assessmentInput(mode assessment.Mode) assessment.CreateInput {
	return assessment.CreateInput{Knowledge: f.knowledge, Blueprint: f.blueprint, Mode: mode, ExpectedKnowledgeHead: *f.KHead(), ExpectedQuestionHead: *f.QHead()}
}
func (f *learningFixture) createDiagnostic(actor string) assessment.AttemptView {
	f.t.Helper()
	out, e := f.repo.CreateAssessment(f.ctx, f.Access(actor, false), f.assessmentInput(assessment.ModeDiagnostic))
	if e != nil {
		f.t.Fatal(e)
	}
	return out
}

// Answers are derived only from this test's original, actually approved fixed data.
func (f *learningFixture) answers(v assessment.AttemptView, n int) assessment.SubmitInput {
	f.t.Helper()
	out := assessment.SubmitInput{Answers: []assessment.PositionAnswer{}}
	for j, q := range v.Questions {
		a := assessment.Answer{Kind: "skipped"}
		if j < n {
			var raw []byte
			var i question.Instance
			if e := f.db.QueryRow(`SELECT body->'body' FROM question_instances WHERE id=$1 AND version=$2 AND sha256=$3`, q.Instance.ID, q.Instance.Version, q.Instance.SHA256).Scan(&raw); e != nil {
				f.t.Fatal(e)
			}
			if json.Unmarshal(raw, &i) != nil {
				f.t.Fatal("decode approved test question")
			}
			if i.Body.Type == "numeric" {
				x := i.Body.CorrectNumeric.Numerator + "/" + i.Body.CorrectNumeric.Denominator
				a = assessment.Answer{Kind: "numeric", Raw: &x}
			} else {
				a = assessment.Answer{Kind: "choice", ChoiceID: i.Body.CorrectChoiceID}
			}
		}
		out.Answers = append(out.Answers, assessment.PositionAnswer{Position: j + 1, Instance: q.Instance, Answer: a})
	}
	return out
}
func TestAssessmentFourOfFive(t *testing.T) {
	for _, n := range []int{0, 3, 4, 5} {
		t.Run(string(rune('0'+n)), func(t *testing.T) {
			f := newLearningFixture(t)
			v := f.createDiagnostic("learner_a")
			raw, _ := json.Marshal(v)
			for _, bad := range []string{"correctNumeric", "explanation", "template", "parameters", "witness"} {
				if strings.Contains(string(raw), bad) {
					t.Fatal("active data leaked", bad)
				}
			}
			if len(v.Questions) != 5 {
				t.Fatal(v)
			}
			got, e := f.repo.SubmitAssessment(f.ctx, f.Access("learner_a", false), v.Summary.ID, f.answers(v, n))
			if e != nil || got.Score == nil || *got.Score != n || got.Passed == nil || *got.Passed != (n >= 4) || got.Summary.State != "submitted" {
				t.Fatal(got, e)
			}
			if f.count(`SELECT count(*) FROM assessment_answers`) != 5 || f.count(`SELECT count(*) FROM assessment_results`) != 1 {
				t.Fatal("partial result")
			}
			if got.Progress.QualificationGranted != (n >= 4) || f.count(`SELECT count(*) FROM learning_records`) != 0 {
				t.Fatal("diagnostic proof or fictitious reading", got)
			}
		})
	}
}
func TestAssessmentWholeFormatRejection(t *testing.T) {
	f := newLearningFixture(t)
	v := f.createDiagnostic("learner_a")
	for _, kind := range []string{"short", "duplicate-position", "wrong-identity", "numeric", "choice", "mixed"} {
		in := f.answers(v, 5)
		switch kind {
		case "short":
			in.Answers = in.Answers[:4]
		case "duplicate-position":
			in.Answers[4].Position = 1
		case "wrong-identity":
			in.Answers[4].Instance.SHA256 = strings.Repeat("0", 64)
		case "numeric":
			in.Answers[4].Answer.Raw = stringPointer("1/0")
		case "choice":
			in.Answers[4].Answer = assessment.Answer{Kind: "choice", ChoiceID: stringPointer("missing")}
		case "mixed":
			in.Answers[4].Answer.ChoiceID = stringPointer("x")
		}
		got, e := f.repo.SubmitAssessment(f.ctx, f.Access("learner_a", false), v.Summary.ID, in)
		if e == nil || got.Score != nil || len(got.Items) > 0 {
			t.Fatal("whole invalid input revealed partial scoring", kind, got, e)
		}
		if f.count(`SELECT count(*) FROM assessment_answers`) != 0 || f.count(`SELECT count(*) FROM assessment_results`) != 0 || f.exposureSequence("learner_a") != 0 {
			t.Fatal("invalid submission persisted", kind)
		}
	}
	if _, e := f.repo.SubmitAssessment(f.ctx, f.Access("learner_a", false), v.Summary.ID, f.answers(v, 5)); e != nil {
		t.Fatal(e)
	}
}
func TestAssessmentIdempotencyAndOwnership(t *testing.T) {
	f := newLearningFixture(t)
	create := f.Access("learner_a", false)
	in := f.assessmentInput(assessment.ModeDiagnostic)
	v, e := f.repo.CreateAssessment(f.ctx, create, in)
	if e != nil {
		t.Fatal(e)
	}
	again, e := f.repo.CreateAssessment(f.ctx, create, in)
	if e != nil || again.Summary.ID != v.Summary.ID || f.count(`SELECT count(*) FROM learner_question_views WHERE owner_user_id=$1`, f.ids["learner_a"]) != 5 {
		t.Fatal(again, e)
	}
	_, e = f.repo.CreateAssessment(f.ctx, f.Access("learner_a", false), in)
	var active *learning.ActiveAttemptError
	if !errors.As(e, &active) || active.Summary.ID != v.Summary.ID || active.Summary.Kind != "assessment" {
		t.Fatal("active summary", e)
	}
	for _, act := range []string{"read", "submit", "abandon"} {
		switch act {
		case "read":
			_, e = f.repo.ReadAssessment(f.ctx, f.Access("learner_b", false), v.Summary.ID)
		case "submit":
			_, e = f.repo.SubmitAssessment(f.ctx, f.Access("learner_b", false), v.Summary.ID, f.answers(v, 5))
		case "abandon":
			_, e = f.repo.AbandonAssessment(f.ctx, f.Access("learner_b", false), v.Summary.ID)
		}
		if !errors.Is(e, auth.ErrNotFound) {
			t.Fatal("cross-owner", act, e)
		}
	}
	key := f.Access("learner_a", false)
	answers := f.answers(v, 4)
	r, e := f.repo.SubmitAssessment(f.ctx, key, v.Summary.ID, answers)
	if e != nil {
		t.Fatal(e)
	}
	seq := f.exposureSequence("learner_a")
	replay, e := f.repo.SubmitAssessment(f.ctx, key, v.Summary.ID, answers)
	if e != nil || replay.Score == nil || *replay.Score != 4 || replay.Progress.QualificationGranted != r.Progress.QualificationGranted || f.exposureSequence("learner_a") <= seq {
		t.Fatal("result replay", replay, e)
	}
	_, e = f.repo.SubmitAssessment(f.ctx, key, v.Summary.ID, f.answers(v, 5))
	if !errors.Is(e, question.ErrIdempotencyConflict) {
		t.Fatal("changed input reused key", e)
	}
	_, e = f.repo.SubmitAssessment(f.ctx, f.Access("learner_a", false), v.Summary.ID, answers)
	if !errors.Is(e, learning.ErrStateConflict) {
		t.Fatal(e)
	}
	old, e := f.repo.CreateAssessment(f.ctx, create, in)
	if e != nil || old.Summary.ID != v.Summary.ID || old.Summary.State != "submitted" {
		t.Fatal("create replay changed business ID", old, e)
	}
}
func TestAssessmentAbandonAndReadDoNotGrant(t *testing.T) {
	f := newLearningFixture(t)
	v := f.createDiagnostic("learner_a")
	for j := 0; j < 2; j++ {
		if _, e := f.repo.ReadAssessment(f.ctx, f.Access("learner_a", false), v.Summary.ID); e != nil {
			t.Fatal(e)
		}
	}
	key := f.Access("learner_a", false)
	got, e := f.repo.AbandonAssessment(f.ctx, key, v.Summary.ID)
	if e != nil || got.Summary.State != "abandoned" {
		t.Fatal(got, e)
	}
	if _, e = f.repo.AbandonAssessment(f.ctx, key, v.Summary.ID); e != nil {
		t.Fatal(e)
	}
	if _, e = f.repo.SubmitAssessment(f.ctx, f.Access("learner_a", false), v.Summary.ID, f.answers(v, 5)); !errors.Is(e, learning.ErrStateConflict) {
		t.Fatal(e)
	}
	if f.count(`SELECT count(*) FROM assessment_results`) != 0 || f.count(`SELECT count(*) FROM learning_events`) != 0 || f.exposureSequence("learner_a") != 0 {
		t.Fatal("read/abandon granted or exposed")
	}
	if f.createDiagnostic("learner_a").Summary.ID == v.Summary.ID {
		t.Fatal("no new attempt")
	}
}
func TestAssessmentExposureAfterCreation(t *testing.T) {
	f := newLearningFixture(t)
	f.exec(`INSERT INTO auth_user_roles(user_id,role) VALUES($1,'reviewer')`, f.ids["learner_a"])
	v := f.createDiagnostic("learner_a")
	if _, e := f.repo.ReadQuestionSubmission(f.ctx, f.Access("learner_a", false), f.approvedSubmissionID); e != nil {
		t.Fatal(e)
	}
	got, e := f.repo.SubmitAssessment(f.ctx, f.Access("learner_a", false), v.Summary.ID, f.answers(v, 5))
	if e != nil || got.Outcome != assessment.Affected || got.Score != nil || got.Passed != nil || got.Progress.QualificationGranted || !containsReason(got.Reasons, assessment.ExposedAfterCreation) {
		t.Fatal(got, e)
	}
	if f.count(`SELECT count(*) FROM learning_qualification_events`) != 0 || f.count(`SELECT count(*) FROM assessment_answers`) != 5 {
		t.Fatal("affected input discarded or granted")
	}
}
func TestAssessmentExposureQuestionReplay(t *testing.T) {
	f := newLearningFixture(t)
	f.exec(`INSERT INTO auth_user_roles(user_id,role) VALUES($1,'editor')`, f.ids["learner_a"])
	v := f.createDiagnostic("learner_a")
	key := f.Access("learner_a", false)
	d, e := f.repo.CreateQuestionDraft(f.ctx, key, f.questionInput)
	if e != nil {
		t.Fatal(e)
	}
	seq := f.exposureSequence("learner_a")
	replay, e := f.repo.CreateQuestionDraft(f.ctx, key, f.questionInput)
	if e != nil || replay.ID != d.ID || f.exposureSequence("learner_a") <= seq {
		t.Fatal(replay, e)
	}
	got, e := f.repo.SubmitAssessment(f.ctx, f.Access("learner_a", false), v.Summary.ID, f.answers(v, 5))
	if e != nil || got.Outcome != assessment.Affected || got.Score != nil {
		t.Fatal(got, e)
	}
}
func containsReason(rs []assessment.RestrictionReason, r assessment.RestrictionReason) bool {
	for _, x := range rs {
		if x == r {
			return true
		}
	}
	return false
}
func TestAssessmentNotReadyTemplateCooldown(t *testing.T) {
	f := newLearningFixture(t)
	f.exec(`INSERT INTO auth_user_roles(user_id,role) VALUES($1,'reviewer')`, f.ids["learner_a"])
	if _, e := f.repo.ReadQuestionSubmission(f.ctx, f.Access("learner_a", false), f.approvedSubmissionID); e != nil {
		t.Fatal(e)
	}
	_, e := f.repo.CreateAssessment(f.ctx, f.Access("learner_a", false), f.assessmentInput(assessment.ModeDiagnostic))
	var nr *learning.NotReadyError
	if !errors.As(e, &nr) || nr.RetryAt == nil || !nr.RetryAt.After(time.Now()) || nr.RetryAt.After(time.Now().Add(31*time.Minute)) {
		t.Fatal("real template cooldown omitted feasible retry", e)
	}
	if f.count(`SELECT count(*) FROM assessment_attempts WHERE owner_user_id=$1`, f.ids["learner_a"]) != 0 {
		t.Fatal("not ready created")
	}
	if len(f.createDiagnostic("learner_b").Questions) != 5 {
		t.Fatal("cooldown crossed users")
	}
}
func TestAssessmentNotReadyRecentFiveNeverExpire(t *testing.T) {
	f := newLearningFixture(t)
	// Five original fixed questions give a finite pool with no timed recovery.
	f.questionInput.QuestionPackage.Version = 2
	f.questionInput.QuestionPackage.Templates = []question.Template{}
	f.questionInput.QuestionPackage.FixedQuestions = []question.FixedQuestion{}
	bp := f.questionInput.QuestionPackage.Blueprints[0]
	bp.ID = "lf-exact-five"
	bp.Sources = []question.BlueprintSource{}
	for j, i := range f.items {
		var raw []byte
		var instance question.Instance
		if e := f.db.QueryRow(`SELECT body->'body' FROM question_instances WHERE id=$1 AND version=$2`, i.Instance.ID, i.Instance.Version).Scan(&raw); e != nil {
			t.Fatal(e)
		}
		if json.Unmarshal(raw, &instance) != nil {
			t.Fatal("decode fixed source")
		}
		id := "lf-fixed-" + string(rune('a'+j))
		f.questionInput.QuestionPackage.FixedQuestions = append(f.questionInput.QuestionPackage.FixedQuestions, question.FixedQuestion{ID: id, Version: 1, Body: instance.Body})
		bp.Sources = append(bp.Sources, question.BlueprintSource{Kind: "instance", Ref: question.Ref{ID: id, Version: 1}})
	}
	f.questionInput.QuestionPackage.Blueprints = []question.Blueprint{bp}
	sub := f.QApproved("author_a", "reviewer_a")
	f.QActivate(f.QPrepare(sub.ID))
	f.blueprint = question.Identity{ID: bp.ID, Version: 1}
	if e := f.db.QueryRow(`SELECT sha256 FROM question_blueprints WHERE id=$1 AND version=1`, bp.ID).Scan(&f.blueprint.SHA256); e != nil {
		t.Fatal(e)
	}
	v := f.createDiagnostic("learner_a")
	if _, e := f.repo.SubmitAssessment(f.ctx, f.Access("learner_a", false), v.Summary.ID, f.answers(v, 0)); e != nil {
		t.Fatal(e)
	}
	_, e := f.repo.CreateAssessment(f.ctx, f.Access("learner_a", false), f.assessmentInput(assessment.ModeDiagnostic))
	var nr *learning.NotReadyError
	if !errors.As(e, &nr) || nr.RetryAt != nil {
		t.Fatal("recent five must not promise a timed recovery", e)
	}
}
func TestAssessmentEvidenceBothOrders(t *testing.T) {
	for _, order := range []string{"read-first", "pass-first"} {
		t.Run(order, func(t *testing.T) {
			f := newLearningFixture(t)
			complete := func() {
				if _, e := f.repo.StartLearning(f.ctx, f.Access("learner_a", false), f.knowledge.ID, f.startInput()); e != nil {
					t.Fatal(e)
				}
				if _, e := f.repo.CompleteLearning(f.ctx, f.Access("learner_a", false), f.knowledge.ID, f.completeInput()); e != nil {
					t.Fatal(e)
				}
			}
			if order == "read-first" {
				complete()
			}
			v, e := f.repo.CreateAssessment(f.ctx, f.Access("learner_a", false), f.assessmentInput(assessment.ModeNode))
			if e != nil {
				t.Fatal(e)
			}
			r, e := f.repo.SubmitAssessment(f.ctx, f.Access("learner_a", false), v.Summary.ID, f.answers(v, 5))
			if e != nil || r.Outcome != assessment.Passed {
				t.Fatal(r, e)
			}
			if r.Progress.QualificationGranted != (order == "read-first") {
				t.Fatal("ordinary pass skipped reading", r)
			}
			if order == "pass-first" {
				complete()
			}
			state, e := f.repo.StartLearning(f.ctx, f.Access("learner_a", false), f.knowledge.ID, f.startInput())
			if e != nil || state.Qualification == nil || state.Qualification.Kind != "normal" || state.Qualification.CompletedEventID == nil {
				t.Fatal(state, e)
			}
		})
	}
}
func TestAssessmentEvidenceLockedDiagnosticAndReview(t *testing.T) {
	f := newLearningFixture(t)
	f.publishLearningGraph()
	f.setLearningTarget("workflow-dependent", "lf-workflow-dependent-five", 0)
	for _, mode := range []assessment.Mode{assessment.ModeNode, assessment.ModeReview} {
		_, e := f.repo.CreateAssessment(f.ctx, f.Access("learner_a", false), f.assessmentInput(mode))
		if e == nil {
			t.Fatal("locked/no-history mode allowed", mode)
		}
	}
	v := f.createDiagnostic("learner_a")
	r, e := f.repo.SubmitAssessment(f.ctx, f.Access("learner_a", false), v.Summary.ID, f.answers(v, 5))
	if e != nil || !r.Progress.QualificationGranted || f.count(`SELECT count(*) FROM learning_records`) != 0 || f.count(`SELECT count(*) FROM learning_unlocks WHERE knowledge_id='workflow-fractions'`) != 0 {
		t.Fatal("diagnostic fabricated ancestor/reading", r, e)
	}
	review, e := f.repo.CreateAssessment(f.ctx, f.Access("learner_a", false), f.assessmentInput(assessment.ModeReview))
	if e != nil {
		t.Fatal(e)
	}
	failed, e := f.repo.SubmitAssessment(f.ctx, f.Access("learner_a", false), review.Summary.ID, f.answers(review, 0))
	if e != nil || failed.Outcome != assessment.Failed {
		t.Fatal(failed, e)
	}
	state, e := f.repo.StartLearning(f.ctx, f.Access("learner_a", false), f.knowledge.ID, f.startInput())
	if e != nil || state.State != learning.NeedsReview || state.Qualification == nil || !state.EverUnlocked {
		t.Fatal("failed review erased alternative", state, e)
	}
}
func TestAssessmentEvidenceOrdinaryReplacement(t *testing.T) {
	f := newLearningFixture(t)
	v := f.createDiagnostic("learner_a")
	f.questionInput.QuestionPackage.Version = 2
	f.questionInput.QuestionPackage.Templates[0].Version = 2
	f.questionInput.QuestionPackage.Templates[0].PromptTemplate += " Additional original wording."
	f.questionInput.QuestionPackage.Blueprints[0].Version = 2
	f.questionInput.QuestionPackage.Blueprints[0].Sources[0].Ref.Version = 2
	sub := f.QApproved("author_a", "reviewer_a")
	f.QActivate(f.QPrepare(sub.ID))
	r, e := f.repo.SubmitAssessment(f.ctx, f.Access("learner_a", false), v.Summary.ID, f.answers(v, 5))
	if e != nil || r.Outcome != assessment.Passed || !r.Progress.QualificationGranted {
		t.Fatal("ordinary replacement invalidated fixed approved exam", r, e)
	}
}
func TestAssessmentEvidenceKnowledgeUpdateAffected(t *testing.T) {
	f := newLearningFixture(t)
	v := f.createDiagnostic("learner_a")
	in := newMaterial(f.Input())
	old := f.KHead()
	sub := f.ApprovedInput(in)
	f.Activate(f.Prepare(sub, old), old)
	r, e := f.repo.SubmitAssessment(f.ctx, f.Access("learner_a", false), v.Summary.ID, f.answers(v, 5))
	if e != nil || r.Outcome != assessment.Affected || r.Score != nil || r.Passed != nil || !containsReason(r.Reasons, assessment.KnowledgeUpdated) || r.Progress.QualificationGranted {
		t.Fatal(r, e)
	}
}
func TestAssessmentEvidenceWithdrawalBeforeAndAfterGrant(t *testing.T) {
	for _, order := range []string{"before", "after"} {
		t.Run(order, func(t *testing.T) {
			f := newLearningFixture(t)
			v := f.createDiagnostic("learner_a")
			withdraw := func() {
				f.QWithdraw(question.WithdrawalTarget{Kind: "instance", ID: v.Questions[0].Instance.ID, Version: v.Questions[0].Instance.Version})
			}
			if order == "before" {
				withdraw()
			}
			r, e := f.repo.SubmitAssessment(f.ctx, f.Access("learner_a", false), v.Summary.ID, f.answers(v, 5))
			if e != nil {
				t.Fatal(e)
			}
			if order == "before" && (r.Outcome != assessment.Affected || r.Score != nil || r.Progress.QualificationGranted) {
				t.Fatal(r)
			}
			if order == "after" {
				if r.Score == nil || *r.Score != 5 || !r.Progress.QualificationGranted {
					t.Fatal(r)
				}
				withdraw()
				state, e := f.repo.StartLearning(f.ctx, f.Access("learner_a", false), f.knowledge.ID, f.startInput())
				if e != nil || state.Qualification != nil || !state.EverUnlocked {
					t.Fatal(state, e)
				}
			}
		})
	}
}
