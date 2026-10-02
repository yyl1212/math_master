package store_test

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"testing"
	"time"
)

func TestAssessmentExpiryAndRaceCreate(t *testing.T) {
	f := newLearningFixture(t)
	in := f.assessmentInput(assessment.ModeDiagnostic)
	done := make(chan error, 2)
	for _, a := range []question.Access{f.Access("learner_a", false), f.Access("learner_a", false)} {
		go func(a question.Access) { _, e := f.repo.CreateAssessment(f.ctx, a, in); done <- e }(a)
	}
	ok, busy := 0, 0
	for j := 0; j < 2; j++ {
		e := <-done
		if e == nil {
			ok++
		} else if errors.Is(e, learning.ErrAssessmentActive) {
			busy++
		} else {
			t.Fatal(e)
		}
	}
	if ok != 1 || busy != 1 || f.count(`SELECT count(*) FROM assessment_attempts WHERE state='active'`) != 1 {
		t.Fatal(ok, busy)
	}
}
func TestAssessmentExpiryAndRaceSubmit(t *testing.T) {
	f := newLearningFixture(t)
	v := f.createDiagnostic("learner_a")
	in := f.answers(v, 5)
	done := make(chan error, 2)
	for _, a := range []question.Access{f.Access("learner_a", false), f.Access("learner_a", false)} {
		go func(a question.Access) { _, e := f.repo.SubmitAssessment(f.ctx, a, v.Summary.ID, in); done <- e }(a)
	}
	ok, conflict := 0, 0
	for j := 0; j < 2; j++ {
		e := <-done
		if e == nil {
			ok++
		} else if errors.Is(e, learning.ErrStateConflict) {
			conflict++
		} else {
			t.Fatal(e)
		}
	}
	if ok != 1 || conflict != 1 || f.count(`SELECT count(*) FROM assessment_answers`) != 5 || f.count(`SELECT count(*) FROM assessment_results`) != 1 || f.count(`SELECT count(*) FROM learning_qualification_events`) != 1 {
		t.Fatal(ok, conflict)
	}
}
func TestAssessmentExpiryAndRaceAbandonSubmit(t *testing.T) {
	f := newLearningFixture(t)
	v := f.createDiagnostic("learner_a")
	in := f.answers(v, 5)
	done := make(chan error, 2)
	a, b := f.Access("learner_a", false), f.Access("learner_a", false)
	go func() { _, e := f.repo.SubmitAssessment(f.ctx, a, v.Summary.ID, in); done <- e }()
	go func() { _, e := f.repo.AbandonAssessment(f.ctx, b, v.Summary.ID); done <- e }()
	ok, conflict := 0, 0
	for j := 0; j < 2; j++ {
		e := <-done
		if e == nil {
			ok++
		} else if errors.Is(e, learning.ErrStateConflict) {
			conflict++
		} else {
			t.Fatal(e)
		}
	}
	var state string
	if e := f.db.QueryRow(`SELECT state FROM assessment_attempts WHERE id=$1`, v.Summary.ID).Scan(&state); e != nil {
		t.Fatal(e)
	}
	n := f.count(`SELECT count(*) FROM assessment_answers`)
	if ok != 1 || conflict != 1 || (state == "submitted" && n != 5) || (state == "abandoned" && n != 0) {
		t.Fatal(ok, conflict, state, n)
	}
}

// Timed fixtures are lawful sealed rows with approved mathematics and all dependencies.
func (f *learningFixture) timedAssessment(delta time.Duration) string {
	f.t.Helper()
	id := f.ID()
	seal := f.seal("assessment")
	raw, sha, e := assessment.CanonicalSeal(seal)
	if e != nil {
		f.t.Fatal(e)
	}
	tx, e := f.db.Begin()
	if e != nil {
		f.t.Fatal(e)
	}
	defer tx.Rollback()
	var expiry time.Time
	if e = tx.QueryRow(`SELECT clock_timestamp()+($1::bigint*interval '1 microsecond')`, delta.Microseconds()).Scan(&expiry); e != nil {
		f.t.Fatal(e)
	}
	if _, e = tx.Exec(`INSERT INTO assessment_attempts(id,owner_user_id,knowledge_id,knowledge_version,knowledge_sha256,knowledge_publication_id,question_publication_id,blueprint_id,blueprint_version,blueprint_sha256,mode,rule_version,core,seed,seal,seal_bytes,seal_sha256,created_at,expires_at,creation_exposure_sequence) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'diagnostic',1,'[0]',$11,$12,$13,$14,$15,$16,coalesce((SELECT sequence FROM learner_exposure_state WHERE owner_user_id=$2),0))`, id, f.ids["learner_a"], f.knowledge.ID, f.knowledge.Version, f.knowledge.SHA256, *f.KHead(), *f.QHead(), f.blueprint.ID, f.blueprint.Version, f.blueprint.SHA256, seal.Seed, string(raw), raw, sha, expiry.Add(-24*time.Hour), expiry); e != nil {
		f.t.Fatal(e)
	}
	for _, i := range seal.Items {
		b, _ := json.Marshal(i)
		if _, e = tx.Exec(`INSERT INTO assessment_items(attempt_id,position,instance_id,instance_version,instance_sha256,binding) VALUES($1,$2,$3,$4,$5,$6)`, id, i.Position, i.Instance.ID, i.Instance.Version, i.Instance.SHA256, string(b)); e != nil {
			f.t.Fatal(e)
		}
	}
	if e = f.insertDependencies(tx, id, f.ids["learner_a"], "assessment", seal); e != nil {
		f.t.Fatal(e)
	}
	if _, e = tx.Exec(`UPDATE assessment_attempts SET sealed=true WHERE id=$1`, id); e != nil {
		f.t.Fatal(e)
	}
	if e = tx.Commit(); e != nil {
		f.t.Fatal(e)
	}
	return id
}
func TestAssessmentExpiryDerivedAndNewCreate(t *testing.T) {
	f := newLearningFixture(t)
	id := f.timedAssessment(-time.Hour)
	v, e := f.repo.ReadAssessment(f.ctx, f.Access("learner_a", false), id)
	if e != nil || v.Summary.State != "expired" {
		t.Fatal(v, e)
	}
	var state string
	if e = f.db.QueryRow(`SELECT state FROM assessment_attempts WHERE id=$1`, id).Scan(&state); e != nil || state != "active" {
		t.Fatal("GET wrote expiry", state, e)
	}
	if _, e = f.repo.SubmitAssessment(f.ctx, f.Access("learner_a", false), id, f.answers(v, 5)); !errors.Is(e, learning.ErrAssessmentExpired) {
		t.Fatal(e)
	}
	next := f.createDiagnostic("learner_a")
	if next.Summary.ID == id {
		t.Fatal(next)
	}
	if f.count(`SELECT count(*) FROM assessment_attempts WHERE state='expired'`) != 1 {
		t.Fatal("new create did not terminalize old expiry")
	}
}
func TestAssessmentExpiryAfterAttemptLockWait(t *testing.T) {
	f := newLearningFixture(t)
	id := f.timedAssessment(650 * time.Millisecond)
	qs := make([]assessment.SafeQuestion, 0, 5)
	for _, i := range f.items {
		qs = append(qs, assessment.SafeQuestion{Instance: i.Instance})
	}
	in := f.answers(assessment.AttemptView{Questions: qs}, 5)
	block, e := f.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer block.Rollback()
	var locked string
	if e = block.QueryRow(`SELECT id::text FROM assessment_attempts WHERE id=$1 FOR UPDATE`, id).Scan(&locked); e != nil {
		t.Fatal(e)
	}
	done := make(chan error, 1)
	a := f.Access("learner_a", false)
	go func() { _, e := f.repo.SubmitAssessment(f.ctx, a, id, in); done <- e }()
	deadline := time.Now().Add(350 * time.Millisecond)
	waiting := false
	for time.Now().Before(deadline) {
		if e = f.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND pid<>pg_backend_pid() AND wait_event_type='Lock' AND query LIKE '%FROM assessment_attempts WHERE id=%')`).Scan(&waiting); e != nil {
			t.Fatal(e)
		}
		if waiting {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !waiting {
		t.Fatal("request did not reach attempt lock")
	}
	time.Sleep(700 * time.Millisecond)
	if e = block.Commit(); e != nil {
		t.Fatal(e)
	}
	if e = <-done; !errors.Is(e, learning.ErrAssessmentExpired) {
		t.Fatal("submitted using time before lock wait", e)
	}
	if f.count(`SELECT count(*) FROM assessment_answers`) != 0 || f.exposureSequence("learner_a") != 0 {
		t.Fatal("expired request committed")
	}
}
func TestAssessmentExpiryAndRaceRevocationCredentialCancellation(t *testing.T) {
	for _, kind := range []string{"session", "password", "credential", "cancel"} {
		t.Run(kind, func(t *testing.T) {
			f := newLearningFixture(t)
			v := f.createDiagnostic("learner_a")
			in := f.answers(v, 5)
			a := f.Access("learner_a", false)
			ctx := f.ctx
			switch kind {
			case "session":
				f.exec(`UPDATE auth_sessions SET revoked_at=clock_timestamp() WHERE token_hash=$1`, a.TokenHash[:])
			case "password":
				f.exec(`UPDATE auth_users SET must_change_password=true WHERE id=$1`, f.ids["learner_a"])
			case "credential":
				f.exec(`UPDATE auth_users SET credential_version=credential_version+1 WHERE id=$1`, f.ids["learner_a"])
			case "cancel":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(f.ctx)
				cancel()
			}
			r, e := f.repo.SubmitAssessment(ctx, a, v.Summary.ID, in)
			if e == nil || r.Score != nil || f.count(`SELECT count(*) FROM assessment_answers`) != 0 || f.count(`SELECT count(*) FROM learning_qualification_events`) != 0 {
				t.Fatal("invalid proof committed", r, e)
			}
			if kind == "password" && !errors.Is(e, auth.ErrPasswordChangeRequired) {
				t.Fatal(e)
			}
		})
	}
}
