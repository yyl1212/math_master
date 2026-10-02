package store_test

import (
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"github.com/yyl1212/math_master/backend/internal/store"
	"testing"
	"time"
)

func TestLearningPracticeExpiryDerivedAndOldKey(t *testing.T) {
	f := newLearningFixture(t)
	in := f.practiceInput()
	key := f.Access("learner_a", false)
	id := f.ID()
	seal := f.seal("practice")
	raw, sha, e := assessment.CanonicalSeal(seal)
	if e != nil {
		t.Fatal(e)
	}
	tx, e := f.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	if _, e = tx.Exec(`WITH n AS MATERIALIZED(SELECT clock_timestamp() t) INSERT INTO practice_attempts(id,owner_user_id,knowledge_id,knowledge_version,knowledge_sha256,knowledge_publication_id,question_publication_id,seal,seal_bytes,seal_sha256,created_at,expires_at) SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,t-interval '25 hours',t-interval '1 hour' FROM n`, id, f.ids["learner_a"], f.knowledge.ID, f.knowledge.Version, f.knowledge.SHA256, *f.KHead(), *f.QHead(), string(raw), raw, sha); e != nil {
		t.Fatal(e)
	}
	if e = f.insertDependencies(tx, id, f.ids["learner_a"], "practice", seal); e != nil {
		t.Fatal(e)
	}
	receipt, _ := json.Marshal(learning.Receipt{ResourceKind: "practice", ResourceID: id, Status: 201})
	if _, e = tx.Exec(`INSERT INTO learning_idempotency(owner_user_id,action,target,key,request_sha256,receipt,receipt_bytes) VALUES($1,'createPractice',$2,$3,$4,$5,$6)`, f.ids["learner_a"], f.knowledge.ID, key.IdempotencyKey, store.LearningRequestSHAForTest(f.knowledge.ID, in), string(receipt), receipt); e != nil {
		t.Fatal(e)
	}
	if e = tx.Commit(); e != nil {
		t.Fatal(e)
	}
	got, e := f.repo.ReadPractice(f.ctx, f.Access("learner_a", false), id)
	if e != nil || got.Summary.State != "expired" || got.Result != nil {
		t.Fatal(got, e)
	}
	var actual string
	if e = f.db.QueryRow(`SELECT state FROM practice_attempts WHERE id=$1`, id).Scan(&actual); e != nil || actual != "active" {
		t.Fatal("GET changed stored state", actual, e)
	}
	if _, e = f.repo.RevealPractice(f.ctx, f.Access("learner_a", false), id); !errors.Is(e, learning.ErrAssessmentExpired) {
		t.Fatal("expired answer reveal allowed", e)
	}
	replay, e := f.repo.CreatePractice(f.ctx, key, in)
	if e != nil || replay.Summary.ID != id || replay.Summary.State != "expired" {
		t.Fatal("expired key silently created another attempt", replay, e)
	}
	next := f.createPractice("learner_a")
	if next.Summary.ID == id {
		t.Fatal("new create key did not replace expired active")
	}
	if e = f.db.QueryRow(`SELECT state FROM practice_attempts WHERE id=$1`, id).Scan(&actual); e != nil || actual != "expired" {
		t.Fatal("new create failed to terminalize old expired record", actual, e)
	}
}
func TestLearningPracticeReadDoesNotAddViewsAndExposesResults(t *testing.T) {
	f := newLearningFixture(t)
	p := f.createPractice("learner_a")
	for j := 0; j < 3; j++ {
		if _, e := f.repo.ReadPractice(f.ctx, f.Access("learner_a", false), p.Summary.ID); e != nil {
			t.Fatal(e)
		}
	}
	if f.count(`SELECT count(*) FROM learner_question_views`) != 1 || f.exposureSequence("learner_a") != 0 {
		t.Fatal("safe refresh counted answer delivery or more questions")
	}
	if _, e := f.repo.RevealPractice(f.ctx, f.Access("learner_a", false), p.Summary.ID); e != nil {
		t.Fatal(e)
	}
	n := f.exposureSequence("learner_a")
	if _, e := f.repo.ReadPractice(f.ctx, f.Access("learner_a", false), p.Summary.ID); e != nil || f.exposureSequence("learner_a") <= n {
		t.Fatal("result read omitted exposure", e)
	}
	f.QWithdraw(question.WithdrawalTarget{Kind: "instance", ID: p.Question.Instance.ID, Version: 1})
	got, e := f.repo.ReadPractice(f.ctx, f.Access("learner_a", false), p.Summary.ID)
	if e != nil || got.Result == nil || got.Result.Item.Validity != assessment.Restricted || got.Result.Item.CorrectNumeric != nil || got.Result.Item.Explanation != nil {
		t.Fatal("withdrawn practice answer remained visible", got, e)
	}
}

func TestLearningPracticeHistoricalResultOverlapBlocked(t *testing.T) {
	f := newLearningFixture(t)
	id := f.ID()
	key := f.Access("learner_a", false)
	seal := f.seal("practice")
	raw, sha, e := assessment.CanonicalSeal(seal)
	if e != nil {
		t.Fatal(e)
	}
	tx, e := f.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	// This accurate fixed practice was revealed 31 minutes ago, so a new formal
	// attempt may legally select its instance after the exposure cooldown.
	if _, e = tx.Exec(`WITH n AS MATERIALIZED(SELECT clock_timestamp() t) INSERT INTO practice_attempts(id,owner_user_id,knowledge_id,knowledge_version,knowledge_sha256,knowledge_publication_id,question_publication_id,seal,seal_bytes,seal_sha256,state,created_at,expires_at,terminal_at) SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,'revealed',t-interval '1 hour',t+interval '23 hours',t-interval '31 minutes' FROM n`, id, f.ids["learner_a"], f.knowledge.ID, f.knowledge.Version, f.knowledge.SHA256, *f.KHead(), *f.QHead(), string(raw), raw, sha); e != nil {
		t.Fatal(e)
	}
	if e = f.insertDependencies(tx, id, f.ids["learner_a"], "practice", seal); e != nil {
		t.Fatal(e)
	}
	receipt, _ := json.Marshal(learning.Receipt{ResourceKind: "practice", ResourceID: id, Status: 200})
	if _, e = tx.Exec(`INSERT INTO learning_idempotency(owner_user_id,action,target,key,request_sha256,receipt,receipt_bytes) VALUES($1,'revealPractice',$2,$3,$4,$5,$6)`, f.ids["learner_a"], id, key.IdempotencyKey, store.LearningRequestSHAForTest(id, struct{}{}), string(receipt), receipt); e != nil {
		t.Fatal(e)
	}
	if _, e = tx.Exec(`INSERT INTO learner_exposure_state(owner_user_id,sequence,updated_at) VALUES($1,1,clock_timestamp()-interval '31 minutes')`, f.ids["learner_a"]); e != nil {
		t.Fatal(e)
	}
	if _, e = tx.Exec(`INSERT INTO learner_answer_exposures(owner_user_id,kind,id,version,sha256,sequence,exposed_at) SELECT $1,'instance',$2,$3,$4,1,terminal_at FROM practice_attempts WHERE id=$5`, f.ids["learner_a"], seal.Items[0].Instance.ID, seal.Items[0].Instance.Version, seal.Items[0].Instance.SHA256, id); e != nil {
		t.Fatal(e)
	}
	if e = f.insertAssessment(tx, f.ID(), f.ids["learner_a"], 5); e != nil {
		t.Fatal(e)
	}
	if e = tx.Commit(); e != nil {
		t.Fatal(e)
	}
	if _, e = f.repo.ReadPractice(f.ctx, f.Access("learner_a", false), id); !errors.Is(e, learning.ErrStateConflict) {
		t.Fatal("historical practice GET exposed active formal item", e)
	}
	if _, e = f.repo.RevealPractice(f.ctx, key, id); !errors.Is(e, learning.ErrStateConflict) {
		t.Fatal("historical same-key reveal exposed active formal item", e)
	}
	if f.exposureSequence("learner_a") != 1 {
		t.Fatal("blocked answer retrieval changed exposure")
	}
}

func TestLearningPracticeExpiryAfterAttemptLockWait(t *testing.T) {
	f := newLearningFixture(t)
	id := f.ID()
	seal := f.seal("practice")
	raw, sha, e := assessment.CanonicalSeal(seal)
	if e != nil {
		t.Fatal(e)
	}
	seed, e := f.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer seed.Rollback()
	if _, e = seed.Exec(`WITH n AS MATERIALIZED(SELECT clock_timestamp()+interval '350 milliseconds' t) INSERT INTO practice_attempts(id,owner_user_id,knowledge_id,knowledge_version,knowledge_sha256,knowledge_publication_id,question_publication_id,seal,seal_bytes,seal_sha256,created_at,expires_at) SELECT $1,$2,$3,$4,$5,$6,$7,$8,$9,$10,t-interval '24 hours',t FROM n`, id, f.ids["learner_a"], f.knowledge.ID, f.knowledge.Version, f.knowledge.SHA256, *f.KHead(), *f.QHead(), string(raw), raw, sha); e != nil {
		t.Fatal(e)
	}
	if e = f.insertDependencies(seed, id, f.ids["learner_a"], "practice", seal); e != nil {
		t.Fatal(e)
	}
	if e = seed.Commit(); e != nil {
		t.Fatal(e)
	}
	block, e := f.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer block.Rollback()
	var locked string
	if e = block.QueryRow(`SELECT id::text FROM practice_attempts WHERE id=$1 FOR UPDATE`, id).Scan(&locked); e != nil {
		t.Fatal(e)
	}
	done := make(chan error, 1)
	a := f.Access("learner_a", false)
	go func() { _, e := f.repo.RevealPractice(f.ctx, a, id); done <- e }()
	deadline := time.Now().Add(300 * time.Millisecond)
	waiting := false
	for time.Now().Before(deadline) {
		if e = f.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM pg_stat_activity WHERE datname=current_database() AND pid<>pg_backend_pid() AND wait_event_type='Lock' AND query LIKE '%FROM practice_attempts WHERE id=%')`).Scan(&waiting); e != nil {
			t.Fatal(e)
		}
		if waiting {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !waiting {
		t.Fatal("request never reached owned attempt lock")
	}
	time.Sleep(400 * time.Millisecond)
	if e = block.Commit(); e != nil {
		t.Fatal(e)
	}
	if e = <-done; !errors.Is(e, learning.ErrAssessmentExpired) {
		t.Fatal("request used pre-wait time to reveal expired answer", e)
	}
	if f.exposureSequence("learner_a") != 0 {
		t.Fatal("expired command committed answer exposure")
	}
}
