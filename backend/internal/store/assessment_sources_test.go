package store_test

import (
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"strings"
	"testing"
	"time"
)

func TestLearningSourcePoolExactVersion(t *testing.T) {
	f := newLearningFixture(t)
	for _, bp := range []*question.Identity{&f.blueprint, nil} {
		p, e := f.repo.LearningSourcePoolForTest(f.ctx, f.Access("learner_a", false), f.knowledge, bp)
		if e != nil || len(p.Candidates) != 16 || p.Knowledge != f.knowledge || p.KnowledgeHead != *f.KHead() || p.QuestionHead != *f.QHead() {
			t.Fatal(p, e)
		}
		for _, c := range p.Candidates {
			if len(c.Coverage) != 1 || c.Coverage[0] != 0 || c.Template == nil || c.Seen || c.RecentSubmitted || c.ExposedAt != nil {
				t.Fatal(c)
			}
		}
		raw, _ := json.Marshal(p)
		for _, s := range []string{"correctNumeric", "witness", "explanation", "parameters", "prompt"} {
			if strings.Contains(string(raw), s) {
				t.Fatal("candidate query loaded private question data", s)
			}
		}
	}
	for _, k := range []question.Identity{{ID: f.knowledge.ID, Version: 2, SHA256: f.knowledge.SHA256}, {ID: f.knowledge.ID, Version: 1, SHA256: strings.Repeat("0", 64)}} {
		if _, e := f.repo.LearningSourcePoolForTest(f.ctx, f.Access("learner_a", false), k, nil); !errors.Is(e, learning.ErrVersionStale) {
			t.Fatal("wrong version accepted", e)
		}
	}
}
func TestLearningSourcePoolTemplateExposureIsExact(t *testing.T) {
	f := newLearningFixture(t)
	owner := f.ids["learner_a"]
	id := *f.items[0].Template
	f.exec(`INSERT INTO learner_exposure_state(owner_user_id,sequence) VALUES($1,1)`, owner)
	f.exec(`INSERT INTO learner_answer_exposures(owner_user_id,kind,id,version,sha256,sequence,exposed_at) VALUES($1,'template',$2,$3,$4,1,clock_timestamp())`, owner, id.ID, id.Version, strings.Repeat("0", 64))
	p, e := f.repo.LearningSourcePoolForTest(f.ctx, f.Access("learner_a", false), f.knowledge, &f.blueprint)
	if e != nil {
		t.Fatal(e)
	}
	for _, c := range p.Candidates {
		if c.ExposedAt != nil {
			t.Fatal("same name with different body poisoned pool")
		}
	}
	f.exec(`UPDATE learner_exposure_state SET sequence=2 WHERE owner_user_id=$1`, owner)
	f.exec(`INSERT INTO learner_answer_exposures(owner_user_id,kind,id,version,sha256,sequence,exposed_at) VALUES($1,'template',$2,$3,$4,2,clock_timestamp())`, owner, id.ID, id.Version, id.SHA256)
	p, e = f.repo.LearningSourcePoolForTest(f.ctx, f.Access("learner_a", false), f.knowledge, &f.blueprint)
	if e != nil {
		t.Fatal(e)
	}
	if len(assessment.EligibleCandidates(p.Candidates, time.Now())) != 0 {
		t.Fatal("one exact template answer must exclude all sixteen instances")
	}
	b, e := f.repo.LearningSourcePoolForTest(f.ctx, f.Access("learner_b", false), f.knowledge, &f.blueprint)
	if e != nil || len(assessment.EligibleCandidates(b.Candidates, time.Now())) != 16 {
		t.Fatal("exposure crossed accounts", e)
	}
}
func (f *learningFixture) submitSkippedFixture(owner string) string {
	f.t.Helper()
	tx, e := f.db.Begin()
	if e != nil {
		f.t.Fatal(e)
	}
	defer tx.Rollback()
	id := f.ID()
	if e = f.insertAssessment(tx, id, f.ids[owner], 5); e != nil {
		f.t.Fatal(e)
	}
	for _, i := range f.items {
		if _, e = tx.Exec(`INSERT INTO assessment_answers(attempt_id,owner_user_id,position,instance_id,instance_version,instance_sha256,answer) VALUES($1,$2,$3,$4,$5,$6,'{"kind":"skipped"}')`, id, f.ids[owner], i.Position, i.Instance.ID, i.Instance.Version, i.Instance.SHA256); e != nil {
			f.t.Fatal(e)
		}
	}
	var now time.Time
	if e = tx.QueryRow(`SELECT clock_timestamp()`).Scan(&now); e != nil {
		f.t.Fatal(e)
	}
	progress := assessment.ProgressUpdate{Knowledge: f.knowledge, NewlyUnlocked: []question.Identity{}}
	raw, _ := json.Marshal(progress)
	if _, e = tx.Exec(`INSERT INTO assessment_results(attempt_id,owner_user_id,rule_version,outcome,score,passed,original_reasons,progress,submitted_at) VALUES($1,$2,1,'failed',0,false,'[]',$3,$4)`, id, f.ids[owner], string(raw), now); e != nil {
		f.t.Fatal(e)
	}
	if _, e = tx.Exec(`UPDATE assessment_results SET sealed=true WHERE attempt_id=$1`, id); e != nil {
		f.t.Fatal(e)
	}
	if _, e = tx.Exec(`UPDATE assessment_attempts SET state='submitted',terminal_at=$2,submission_exposure_sequence=0 WHERE id=$1`, id, now); e != nil {
		f.t.Fatal(e)
	}
	if e = tx.Commit(); e != nil {
		f.t.Fatal(e)
	}
	return id
}
func TestLearningSourcePoolRecentAssessmentIsGlobal(t *testing.T) {
	f := newLearningFixture(t)
	f.submitSkippedFixture("learner_a")
	p, e := f.repo.LearningSourcePoolForTest(f.ctx, f.Access("learner_a", false), f.knowledge, &f.blueprint)
	if e != nil {
		t.Fatal(e)
	}
	n := 0
	for _, c := range p.Candidates {
		if c.RecentSubmitted {
			n++
		}
	}
	if n != 5 || len(assessment.EligibleCandidates(p.Candidates, time.Now().Add(48*time.Hour))) != 11 {
		t.Fatal("latest diagnostic exclusion expired or missing", n)
	}
	p, e = f.repo.LearningSourcePoolForTest(f.ctx, f.Access("learner_b", false), f.knowledge, nil)
	if e != nil {
		t.Fatal(e)
	}
	for _, c := range p.Candidates {
		if c.RecentSubmitted {
			t.Fatal("another account's latest submission excluded a question")
		}
	}
}

func TestLearningSourcePoolExplicitBlueprints(t *testing.T) {
	f := newLearningFixture(t)
	other := f.questionInput.QuestionPackage.Blueprints[0]
	other.ID = "lf-other"
	f.questionInput.QuestionPackage.Version = 2
	f.questionInput.QuestionPackage.Blueprints = append(f.questionInput.QuestionPackage.Blueprints, other)
	sub := f.QApproved("author_a", "reviewer_a")
	f.QActivate(f.QPrepare(sub.ID))
	var id question.Identity
	id.ID = other.ID
	id.Version = 1
	if e := f.db.QueryRow(`SELECT sha256 FROM question_blueprints WHERE id=$1 AND version=1`, id.ID).Scan(&id.SHA256); e != nil {
		t.Fatal(e)
	}
	for _, bp := range []question.Identity{f.blueprint, id} {
		p, e := f.repo.LearningSourcePoolForTest(f.ctx, f.Access("learner_a", false), f.knowledge, &bp)
		if e != nil || p.BlueprintIdentity == nil || *p.BlueprintIdentity != bp || len(p.Candidates) != 16 {
			t.Fatal("explicit blueprint choice lost", p, e)
		}
	}
}
