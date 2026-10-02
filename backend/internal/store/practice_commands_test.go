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

func (f *learningFixture) practiceInput() assessment.PracticeCreateInput {
	return assessment.PracticeCreateInput{Knowledge: f.knowledge, ExpectedKnowledgeHead: *f.KHead(), ExpectedQuestionHead: *f.QHead()}
}
func (f *learningFixture) createPractice(actor string) assessment.PracticeView {
	f.t.Helper()
	p, e := f.repo.CreatePractice(f.ctx, f.Access(actor, false), f.practiceInput())
	if e != nil {
		f.t.Fatal(e)
	}
	return p
}
func TestLearningPracticeRevealNoQualification(t *testing.T) {
	f := newLearningFixture(t)
	f.QWithdraw(question.WithdrawalTarget{Kind: "blueprint", ID: f.blueprint.ID, Version: 1})
	p := f.createPractice("learner_a")
	if p.Summary.State != "active" || p.Result != nil || p.Question.Knowledge != f.knowledge {
		t.Fatal(p)
	}
	raw, _ := json.Marshal(p)
	for _, forbidden := range []string{"correctNumeric", "correctChoiceId", "explanation", "witness", "parameters", "template", "sourceMap"} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatal("active practice leaked", forbidden)
		}
	}
	key := f.Access("learner_a", false)
	got, e := f.repo.RevealPractice(f.ctx, key, p.Summary.ID)
	if e != nil || got.Summary.State != "revealed" || got.Result == nil || got.Result.Item.Correct != nil || got.Result.Item.CorrectNumeric == nil || f.count(`SELECT count(*) FROM learning_qualification_events`) != 0 || f.count(`SELECT count(*) FROM learning_records`) != 0 {
		t.Fatal(got, e)
	}
	seq := f.exposureSequence("learner_a")
	replay, e := f.repo.RevealPractice(f.ctx, key, p.Summary.ID)
	if e != nil || replay.Summary.ID != got.Summary.ID || f.exposureSequence("learner_a") <= seq {
		t.Fatal("same-key terminal delivery omitted exposure", replay, e)
	}
	next := f.createPractice("learner_a")
	if next.Question.Instance == p.Question.Instance {
		t.Fatal("practice ignored unseen priority")
	}
}
func TestLearningPracticeFormatAndOneAnswer(t *testing.T) {
	f := newLearningFixture(t)
	p := f.createPractice("learner_a")
	for _, in := range []assessment.Answer{{Kind: "skipped"}, {Kind: "numeric", Raw: stringPointer("1/0")}, {Kind: "numeric", Raw: stringPointer(strings.Repeat(" ", 129))}} {
		if _, e := f.repo.AnswerPractice(f.ctx, f.Access("learner_a", false), p.Summary.ID, in); e == nil {
			t.Fatal("invalid input accepted")
		}
		got, e := f.repo.ReadPractice(f.ctx, f.Access("learner_a", false), p.Summary.ID)
		if e != nil || got.Result != nil || got.Summary.State != "active" {
			t.Fatal("invalid answer revealed partial correctness", got, e)
		}
	}
	key := f.Access("learner_a", false)
	answer := assessment.Answer{Kind: "numeric", Raw: stringPointer(" 0 ")}
	got, e := f.repo.AnswerPractice(f.ctx, key, p.Summary.ID, answer)
	if e != nil || got.Summary.State != "answered" || got.Result == nil || got.Result.Item.Correct == nil || *got.Result.Item.Correct || got.Result.Item.Answer == nil || *got.Result.Item.Answer.Raw != " 0 " {
		t.Fatal("valid wrong answer did not terminate exactly once", got, e)
	}
	if _, e = f.repo.AnswerPractice(f.ctx, key, p.Summary.ID, answer); e != nil {
		t.Fatal("exact same-key answer replay failed", e)
	}
	if _, e = f.repo.AnswerPractice(f.ctx, f.Access("learner_a", false), p.Summary.ID, assessment.Answer{Kind: "numeric", Raw: stringPointer("2")}); !errors.Is(e, learning.ErrStateConflict) {
		t.Fatal("second answer rewrote result", e)
	}
	if _, e = f.repo.ReadPractice(f.ctx, f.Access("learner_b", false), p.Summary.ID); !errors.Is(e, auth.ErrNotFound) {
		t.Fatal("practice leaked across owners", e)
	}
}
func stringPointer(v string) *string { return &v }
func TestLearningPracticeActiveAndAbandonReplay(t *testing.T) {
	f := newLearningFixture(t)
	key := f.Access("learner_a", false)
	in := f.practiceInput()
	p, e := f.repo.CreatePractice(f.ctx, key, in)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = f.repo.CreatePractice(f.ctx, f.Access("learner_a", false), in); !errors.Is(e, learning.ErrAssessmentActive) {
		t.Fatal("second active practice allowed", e)
	} else {
		var active *learning.ActiveAttemptError
		if !errors.As(e, &active) || active.Summary.ID != p.Summary.ID || active.Summary.Kind != "practice" {
			t.Fatal("conflict omitted safe active summary", e)
		}
	}
	got, e := f.repo.AbandonPractice(f.ctx, f.Access("learner_a", false), p.Summary.ID)
	if e != nil || got.Summary.State != "abandoned" || got.Result != nil {
		t.Fatal(got, e)
	}
	replay, e := f.repo.CreatePractice(f.ctx, key, in)
	if e != nil || replay.Summary.ID != p.Summary.ID || replay.Summary.State != "abandoned" {
		t.Fatal("old create key created another attempt", replay, e)
	}
	next := f.createPractice("learner_a")
	if next.Summary.ID == p.Summary.ID {
		t.Fatal("new key did not create new attempt")
	}
}
func TestLearningPracticeConcurrentSingleActive(t *testing.T) {
	f := newLearningFixture(t)
	done := make(chan error, 2)
	for _, a := range []question.Access{f.Access("learner_a", false), f.Access("learner_a", false)} {
		go func(a question.Access) { _, e := f.repo.CreatePractice(f.ctx, a, f.practiceInput()); done <- e }(a)
	}
	success, conflict := 0, 0
	for j := 0; j < 2; j++ {
		e := <-done
		if e == nil {
			success++
		} else if errors.Is(e, learning.ErrAssessmentActive) {
			conflict++
		} else {
			t.Fatal(e)
		}
	}
	if success != 1 || conflict != 1 || f.count(`SELECT count(*) FROM practice_attempts WHERE state='active'`) != 1 {
		t.Fatal(success, conflict)
	}
}
func TestLearningPracticeFormalOverlapProtected(t *testing.T) {
	f := newLearningFixture(t)
	p := f.createPractice("learner_a")
	pool, e := f.repo.LearningSourcePoolForTest(f.ctx, f.Access("learner_a", false), f.knowledge, &f.blueprint)
	if e != nil {
		t.Fatal(e)
	}
	ids := []question.Identity{p.Question.Instance}
	for _, c := range pool.Candidates {
		if c.Identity != p.Question.Instance && len(ids) < 5 {
			ids = append(ids, c.Identity)
		}
	}
	for j := range f.items {
		f.items[j].Instance = ids[j]
	}
	tx, e := f.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	if e = f.insertAssessment(tx, f.ID(), f.ids["learner_a"], 5); e != nil {
		t.Fatal(e)
	}
	if e = tx.Commit(); e != nil {
		t.Fatal(e)
	}
	for _, reveal := range []bool{false, true} {
		var e error
		if reveal {
			_, e = f.repo.RevealPractice(f.ctx, f.Access("learner_a", false), p.Summary.ID)
		} else {
			_, e = f.repo.AnswerPractice(f.ctx, f.Access("learner_a", false), p.Summary.ID, assessment.Answer{Kind: "numeric", Raw: stringPointer("0")})
		}
		if !errors.Is(e, learning.ErrStateConflict) {
			t.Fatal("old practice delivered answer for active formal item", e)
		}
	}
	if f.exposureSequence("learner_a") != 0 {
		t.Fatal("blocked overlap exposed an answer")
	}
}
func TestLearningPracticeCreateExcludesActiveFive(t *testing.T) {
	f := newLearningFixture(t)
	tx, e := f.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	if e = f.insertAssessment(tx, f.ID(), f.ids["learner_a"], 5); e != nil {
		t.Fatal(e)
	}
	if e = tx.Commit(); e != nil {
		t.Fatal(e)
	}
	p := f.createPractice("learner_a")
	for _, i := range f.items {
		if i.Instance == p.Question.Instance {
			t.Fatal("new practice overlaps active formal five")
		}
	}
}

func TestLearningPracticeLockedNodeCorrectWithoutQualification(t *testing.T) {
	f := newLearningFixture(t)
	f.publishLearningGraph()
	f.setLearningTarget("workflow-dependent", "lf-workflow-dependent-five", 0)
	if _, e := f.repo.StartLearning(f.ctx, f.Access("learner_a", false), f.knowledge.ID, f.startInput()); !errors.Is(e, learning.ErrPrerequisitesUnmet) {
		t.Fatal("fixture was not actually locked", e)
	}
	p := f.createPractice("learner_a")
	var raw []byte
	var instance question.Instance
	if e := f.db.QueryRow(`SELECT body->'body' FROM question_instances WHERE id=$1 AND version=$2 AND sha256=$3`, p.Question.Instance.ID, p.Question.Instance.Version, p.Question.Instance.SHA256).Scan(&raw); e != nil {
		t.Fatal(e)
	}
	if json.Unmarshal(raw, &instance) != nil {
		t.Fatal("fixed synthetic instance decode")
	}
	answer := instance.Body.CorrectNumeric.Numerator + "/" + instance.Body.CorrectNumeric.Denominator
	got, e := f.repo.AnswerPractice(f.ctx, f.Access("learner_a", false), p.Summary.ID, assessment.Answer{Kind: "numeric", Raw: &answer})
	if e != nil || got.Result == nil || got.Result.Item.Correct == nil || !*got.Result.Item.Correct || f.count(`SELECT count(*) FROM learning_qualification_events`) != 0 || f.count(`SELECT count(*) FROM learning_unlocks`) != 0 || f.count(`SELECT count(*) FROM learning_records`) != 0 {
		t.Fatal("correct locked-node practice granted formal progress", got, e)
	}
}
