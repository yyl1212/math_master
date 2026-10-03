package store_test

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"github.com/yyl1212/math_master/backend/internal/question"
	"testing"
	"time"
)

func (f *feedbackFixture) instanceFeedback() feedback.Metadata {
	f.t.Helper()
	p := f.createPractice("learner_a")
	context, e := f.repo.ReadFeedbackContext(f.ctx, f.Access("learner_a", false), feedback.ContextQuery{Kind: "practice", ID: p.Summary.ID})
	if e != nil {
		f.t.Fatal(e)
	}
	in := f.feedbackInput()
	in.Target = context.Data.Target
	in.Source = context.Data.Source
	return f.feedbackCreate("learner_a", in)
}
func TestFeedbackOriginalTemplateExposure(t *testing.T) {
	f := newFeedbackFixture(t)
	m := f.instanceFeedback()
	for _, actor := range []string{"learner_a", "reviewer_a"} {
		tx, _ := f.db.Begin()
		id := f.ID()
		if e := f.insertAssessment(tx, id, f.ids[actor], 5); e != nil {
			t.Fatal(e)
		}
		if e := tx.Commit(); e != nil {
			t.Fatal(e)
		}
		before := f.exposureSequence(actor)
		out, e := f.repo.ReadFeedbackEvents(f.ctx, f.Access(actor, false), m.ID, actor == "reviewer_a", feedback.ListQuery{})
		if !errors.Is(e, feedback.ErrAnswerOverlap) || out.Data.Title != "" || len(out.Data.Items) != 0 || f.exposureSequence(actor) != before {
			t.Fatal("overlap delivery", e)
		}
		f.exec(`UPDATE assessment_attempts SET state='abandoned',terminal_at=clock_timestamp() WHERE id=$1`, id)
	}
	f.QWithdraw(question.WithdrawalTarget{Kind: "template", ID: "lf-addition", Version: 1})
	page, e := f.repo.ReadFeedbackEvents(f.ctx, f.Access("reviewer_a", false), m.ID, true, feedback.ListQuery{})
	if e != nil || page.Data.Title == "" {
		t.Fatal("original historical discussion", e)
	}
	if f.count(`SELECT count(*) FROM learner_answer_exposures WHERE owner_user_id=$1 AND kind='instance' AND id=$2 AND version=$3 AND sha256=$4`, f.ids["reviewer_a"], m.Target.Identity.ID, m.Target.Identity.Version, m.Target.Identity.SHA256) != 1 {
		t.Fatal("instance exposure")
	}
	template := f.items[0].Template
	if f.count(`SELECT count(*) FROM learner_answer_exposures WHERE owner_user_id=$1 AND kind='template' AND id=$2 AND version=$3 AND sha256=$4`, f.ids["reviewer_a"], template.ID, template.Version, template.SHA256) != 1 {
		t.Fatal("original template")
	}
	first := f.exposureSequence("reviewer_a")
	if _, e = f.repo.ReadFeedbackEvents(f.ctx, f.Access("reviewer_a", false), m.ID, true, feedback.ListQuery{}); e != nil || f.exposureSequence("reviewer_a") <= first {
		t.Fatal("every delivery", e)
	}
}
func TestFeedbackExposureExpiredActive(t *testing.T) {
	f := newFeedbackFixture(t)
	m := f.instanceFeedback()
	id := f.timedAssessment(-time.Minute)
	page, e := f.repo.ReadFeedbackEvents(f.ctx, f.Access("learner_a", false), m.ID, false, feedback.ListQuery{})
	if e != nil || len(page.Data.Items) != 1 {
		t.Fatal("expired active blocked", e)
	}
	if f.count(`SELECT count(*) FROM assessment_attempts WHERE id=$1 AND state='active'`, id) != 1 {
		t.Fatal("test did not cover uncleaned row")
	}
}
func TestFeedbackExposureFailureClosed(t *testing.T) {
	f := newFeedbackFixture(t)
	m := f.instanceFeedback()
	before := f.exposureSequence("learner_a")
	f.exec(`CREATE FUNCTION fail_feedback_exposure() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'isolated exposure failure'; END $$`)
	f.exec(`CREATE TRIGGER fail_feedback_exposure BEFORE INSERT OR UPDATE ON learner_answer_exposures FOR EACH ROW EXECUTE FUNCTION fail_feedback_exposure()`)
	page, e := f.repo.ReadFeedbackEvents(f.ctx, f.Access("learner_a", false), m.ID, false, feedback.ListQuery{})
	if e == nil || page.Data.Title != "" || len(page.Data.Items) != 0 || f.exposureSequence("learner_a") != before {
		t.Fatal("failed accounting delivered text", e)
	}
}

func TestFeedbackOriginalTemplateNewVersion(t *testing.T) {
	f := newFeedbackFixture(t)
	m := f.instanceFeedback()
	pkg := &f.questionInput.QuestionPackage
	pkg.Version = 2
	pkg.Templates[0].Version = 2
	pkg.Blueprints[0].Version = 2
	pkg.Blueprints[0].Sources[0].Ref.Version = 2
	sub := f.QApproved("author_b", "reviewer_b")
	f.QActivate(f.QPrepare(sub.ID))
	f.setLearningTarget("workflow-fractions", "lf-five", 0)
	f.blueprint.Version = 2
	if e := f.db.QueryRow(`SELECT sha256 FROM question_blueprints WHERE id=$1 AND version=2`, f.blueprint.ID).Scan(&f.blueprint.SHA256); e != nil {
		t.Fatal(e)
	}
	if f.items[0].Template.ID != "lf-addition" || f.items[0].Template.Version != 2 {
		t.Fatal("new version fixture")
	}
	tx, _ := f.db.Begin()
	if e := f.insertAssessment(tx, f.ID(), f.ids["learner_a"], 5); e != nil {
		t.Fatal(e)
	}
	if e := tx.Commit(); e != nil {
		t.Fatal(e)
	}
	page, e := f.repo.ReadFeedbackEvents(f.ctx, f.Access("learner_a", false), m.ID, false, feedback.ListQuery{})
	if e != nil || page.Data.Title == "" {
		t.Fatal("new template blocked original", e)
	}
	if f.count(`SELECT count(*) FROM learner_answer_exposures WHERE owner_user_id=$1 AND kind='template' AND id='lf-addition' AND version=1`, f.ids["learner_a"]) != 1 {
		t.Fatal("original template version not recorded")
	}
}
func TestFeedbackExposureExpiryWhileWaiting(t *testing.T) {
	f := newFeedbackFixture(t)
	m := f.instanceFeedback()
	f.timedAssessment(250 * time.Millisecond)
	block, _ := f.db.Begin()
	defer block.Rollback()
	if _, e := block.Exec(`SELECT pg_advisory_xact_lock(1296127048)`); e != nil {
		t.Fatal(e)
	}
	a := f.Access("learner_a", false)
	done := make(chan error, 1)
	go func() { _, e := f.repo.ReadFeedbackEvents(f.ctx, a, m.ID, false, feedback.ListQuery{}); done <- e }()
	deadline := time.Now().Add(750 * time.Millisecond)
	waiting := false
	for time.Now().Before(deadline) {
		if f.count(`SELECT count(*) FROM pg_locks WHERE database=(SELECT oid FROM pg_database WHERE datname=current_database()) AND locktype='advisory' AND objid=1296127048 AND NOT granted`) > 0 {
			waiting = true
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if !waiting {
		t.Fatal("no real lock wait")
	}
	time.Sleep(280 * time.Millisecond)
	if e := block.Commit(); e != nil {
		t.Fatal(e)
	}
	if e := <-done; e != nil {
		t.Fatal("stale overlap clock", e)
	}
}
