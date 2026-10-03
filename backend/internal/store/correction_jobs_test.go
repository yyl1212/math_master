package store_test

import (
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"testing"
	"time"
)

func TestCorrectionEnqueueWithdrawalAtomicReplayAndRollback(t *testing.T) {
	for _, space := range []string{"question", "content"} {
		t.Run(space, func(t *testing.T) {
			f := newCorrectionFixture(t)
			key := f.Access("admin_a", true)
			head := *f.QHead()
			var wid string
			var e error
			if space == "question" {
				in := question.WithdrawalInput{Target: question.WithdrawalTarget{Kind: "instance", ID: f.items[0].Instance.ID, Version: 1}, ExpectedKnowledgeHead: f.KHead(), ExpectedQuestionHead: &head, Reason: "Original exact withdrawal and transactional outbox test."}
				a, x := f.repo.WithdrawQuestionVersion(f.ctx, key, in)
				e = x
				wid = a.EventID
				if e == nil {
					b, x := f.repo.WithdrawQuestionVersion(f.ctx, key, in)
					e = x
					if b.EventID != wid {
						t.Fatal("replay changed withdrawal")
					}
				}
			} else {
				in := publication.WithdrawalInput{Target: publication.WithdrawalTarget{Kind: "unit", ID: f.questionInput.QuestionPackage.Templates[0].Units[0].ID, Version: 1}, ExpectedHead: f.KHead(), Reason: "Original exact content withdrawal and transactional outbox test."}
				a, x := f.repo.WithdrawVersion(f.ctx, key, in)
				e = x
				wid = a.EventID
				if e == nil {
					b, x := f.repo.WithdrawVersion(f.ctx, key, in)
					e = x
					if b.EventID != wid {
						t.Fatal("replay changed withdrawal")
					}
				}
			}
			if e != nil {
				t.Fatal(e)
			}
			if f.count(`SELECT count(*) FROM correction_cases WHERE withdrawal_space=$1 AND withdrawal_id=$2`, space, wid) != 1 || f.count(`SELECT count(*) FROM correction_jobs`) != 1 || f.count(`SELECT count(*) FROM correction_events WHERE kind='job_created'`) != 1 {
				t.Fatal("withdrawal outbox absent or duplicated")
			}
		})
	}
	f := newCorrectionFixture(t)
	oldHead := *f.QHead()
	f.exec(`CREATE FUNCTION isolated_job_failure() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'isolated atomic rollback fixture'; END $$`)
	f.exec(`CREATE TRIGGER isolated_job_failure BEFORE INSERT ON correction_jobs FOR EACH ROW EXECUTE FUNCTION isolated_job_failure()`)
	if _, e := f.repo.WithdrawQuestionVersion(f.ctx, f.Access("admin_a", true), question.WithdrawalInput{Target: question.WithdrawalTarget{Kind: "instance", ID: f.items[0].Instance.ID, Version: 1}, ExpectedKnowledgeHead: f.KHead(), ExpectedQuestionHead: &oldHead, Reason: "Failure of the task must roll back withdrawal and publication together."}); e == nil {
		t.Fatal("outbox failure did not fail command")
	}
	if *f.QHead() != oldHead || f.count(`SELECT count(*) FROM question_withdrawals`) != 0 || f.count(`SELECT count(*) FROM correction_cases`) != 0 || f.count(`SELECT count(*) FROM correction_jobs`) != 0 {
		t.Fatal("outbox failure left withdrawal or publication")
	}
}
func TestCorrectionEnqueueTerminalRollbackAndReplay(t *testing.T) {
	f := newCorrectionFixture(t)
	v := f.createDiagnostic("learner_a")
	f.registerRule(nil, 1)
	f.exec(`CREATE FUNCTION isolated_job_failure() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'isolated terminal rollback fixture'; END $$`)
	f.exec(`CREATE TRIGGER isolated_job_failure BEFORE INSERT ON correction_jobs FOR EACH ROW WHEN (NEW.type='attempt_terminal') EXECUTE FUNCTION isolated_job_failure()`)
	key := f.Access("learner_a", false)
	in := f.answers(v, 5)
	if _, e := f.repo.SubmitAssessment(f.ctx, key, v.Summary.ID, in); e == nil {
		t.Fatal("terminal outbox failure did not roll back")
	}
	if f.count(`SELECT count(*) FROM assessment_answers`) != 0 || f.count(`SELECT count(*) FROM assessment_results`) != 0 || f.count(`SELECT count(*) FROM assessment_attempts WHERE state='active'`) != 1 {
		t.Fatal("terminal failure left original answers or result")
	}
	f.exec(`DROP TRIGGER isolated_job_failure ON correction_jobs`)
	if _, e := f.repo.SubmitAssessment(f.ctx, key, v.Summary.ID, in); e != nil {
		t.Fatal(e)
	}
	if _, e := f.repo.SubmitAssessment(f.ctx, key, v.Summary.ID, in); e != nil {
		t.Fatal(e)
	}
	if f.count(`SELECT count(*) FROM correction_jobs WHERE type='attempt_terminal'`) != 1 || f.count(`SELECT count(*) FROM correction_events e JOIN correction_jobs j ON j.id=e.subject_id WHERE e.kind='job_created' AND j.type='attempt_terminal'`) != 1 {
		t.Fatal("terminal replay duplicated task")
	}
}
func (f *correctionFixture) jobState(id string) (state string, sequence int64, attempt, epoch int, token int64, next *time.Time) {
	f.t.Helper()
	if e := f.db.QueryRow(`SELECT state,sequence,attempt,epoch,lease_token,next_run_at FROM correction_jobs WHERE id=$1`, id).Scan(&state, &sequence, &attempt, &epoch, &token, &next); e != nil {
		f.t.Fatal(e)
	}
	return
}
func (f *correctionFixture) moveJobClock(id string, expired bool) {
	f.t.Helper()
	tx, e := f.db.Begin()
	if e != nil {
		f.t.Fatal(e)
	}
	defer tx.Rollback()
	var seq int64
	var cid string
	if e = tx.QueryRow(`SELECT sequence,case_id::text FROM correction_jobs WHERE id=$1 FOR UPDATE`, id).Scan(&seq, &cid); e != nil {
		f.t.Fatal(e)
	}
	col := "next_run_at"
	kind := "job_continued"
	if expired {
		col = "lease_until"
		kind = "job_renewed"
	}
	if _, e = tx.Exec(`UPDATE correction_jobs SET `+col+`=clock_timestamp()-interval '1 second',sequence=$2 WHERE id=$1`, id, seq+1); e != nil {
		f.t.Fatal(e)
	}
	if e = f.cEvent(tx, "job", id, kind, "", cid, nil, seq+1, map[string]any{"fixtureClock": true}); e != nil {
		f.t.Fatal(e)
	}
	if e = tx.Commit(); e != nil {
		f.t.Fatal(e)
	}
}
func TestCorrectionLeaseClaimRenewAndExpiredFence(t *testing.T) {
	f := newCorrectionFixture(t)
	f.registerRule(nil, 1)
	leases := make(chan *correction.Lease, 2)
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() { lease, e := f.repo.ClaimCorrectionJob(f.ctx); leases <- lease; errs <- e }()
	}
	var lease *correction.Lease
	claims := 0
	for i := 0; i < 2; i++ {
		l := <-leases
		if e := <-errs; e != nil {
			t.Fatal(e)
		}
		if l != nil {
			lease = l
			claims++
		}
	}
	if claims != 1 || lease.Attempt != 1 || lease.Epoch != 1 || lease.Token < 1 {
		t.Fatal("parallel claim not exclusive", claims, lease)
	}
	renewed, e := f.repo.RenewCorrectionLease(f.ctx, *lease)
	if e != nil || renewed.Token != lease.Token || renewed.Until.Before(lease.Until) {
		t.Fatal("renew did not preserve fencing identity", e)
	}
	f.moveJobClock(lease.JobID, true)
	if _, e = f.repo.RenewCorrectionLease(f.ctx, *lease); !errors.Is(e, correction.ErrLeaseLost) {
		t.Fatal("expired lease renewed", e)
	}
	if e = f.repo.FinishCorrectionJob(f.ctx, *lease, correction.Succeeded, ""); !errors.Is(e, correction.ErrLeaseLost) {
		t.Fatal("expired lease committed", e)
	}
	next, e := f.repo.ClaimCorrectionJob(f.ctx)
	if e != nil || next != nil {
		t.Fatal("expired claim skipped retry backoff", next, e)
	}
	state, _, attempt, _, token, at := f.jobState(lease.JobID)
	if state != "retry_wait" || attempt != 1 || token <= lease.Token || at == nil {
		t.Fatal("expired token not fenced or audited")
	}
	if f.count(`SELECT count(*) FROM correction_events WHERE kind='job_attempt_failed'`) != 1 {
		t.Fatal("expired attempt lost audit")
	}
	f.moveJobClock(lease.JobID, false)
	fresh, e := f.repo.ClaimCorrectionJob(f.ctx)
	if e != nil || fresh == nil || fresh.Token <= token || fresh.Attempt != 2 {
		t.Fatal("retry claim did not fence previous epoch", fresh, e)
	}
	if e = f.repo.FinishCorrectionJob(f.ctx, *lease, correction.Succeeded, ""); !errors.Is(e, correction.ErrLeaseLost) {
		t.Fatal("stale token succeeded", e)
	}
}
func TestCorrectionRetryEpochEightAttemptsAndContinuation(t *testing.T) {
	f := newCorrectionFixture(t)
	f.registerRule(nil, 1)
	lease, e := f.repo.ClaimCorrectionJob(f.ctx)
	if e != nil || lease == nil {
		t.Fatal(e)
	}
	id := lease.JobID
	if e = f.repo.FinishCorrectionJob(f.ctx, *lease, correction.Queued, ""); e != nil {
		t.Fatal(e)
	}
	lease, e = f.repo.ClaimCorrectionJob(f.ctx)
	if e != nil || lease == nil || lease.Attempt != 1 {
		t.Fatal("normal continuation consumed retry", lease, e)
	}
	for attempt, seconds := range []int{5, 10, 20, 40, 80, 160, 300, 0} {
		if lease.Attempt != attempt+1 {
			t.Fatal("wrong total attempt", lease)
		}
		if e = f.repo.FinishCorrectionJob(f.ctx, *lease, correction.RetryWait, "database"); e != nil {
			t.Fatal(e)
		}
		state, _, n, epoch, _, next := f.jobState(id)
		if n != attempt+1 || epoch != 1 {
			t.Fatal(n, epoch)
		}
		if seconds > 0 {
			if state != "retry_wait" || next == nil {
				t.Fatal("retry state", state)
			}
			var recorded time.Time
			if e = f.db.QueryRow(`SELECT recorded_at FROM correction_events WHERE subject_id=$1 AND kind='job_attempt_failed' ORDER BY sequence DESC LIMIT 1`, id).Scan(&recorded); e != nil {
				t.Fatal(e)
			}
			delta := next.Sub(recorded)
			if delta < time.Duration(seconds)*time.Second-100*time.Millisecond || delta > time.Duration(seconds)*time.Second+100*time.Millisecond {
				t.Fatal("wrong exact backoff", seconds, delta)
			}
			f.moveJobClock(id, false)
			lease, e = f.repo.ClaimCorrectionJob(f.ctx)
			if e != nil || lease == nil {
				t.Fatal(e)
			}
		} else if state != "failed" || next != nil {
			t.Fatal("ninth attempt made available", state)
		}
	}
	_, sequence, _, _, _, _ := f.jobState(id)
	key := f.Access("admin_a", false)
	retried, e := f.repo.RetryCorrectionJob(f.ctx, key, id, correction.RetryInput{ExpectedSequence: sequence})
	if e != nil || retried.Data.Job == nil || retried.Data.Job.Epoch != 2 || retried.Data.Job.Attempt != 0 || retried.Data.Job.State != correction.Queued {
		t.Fatal("manual epoch did not reset bounded attempts", retried, e)
	}
	again, e := f.repo.RetryCorrectionJob(f.ctx, key, id, correction.RetryInput{ExpectedSequence: sequence})
	if e != nil || correctionReceiptJSON(t, again) != correctionReceiptJSON(t, retried) {
		t.Fatal("manual retry receipt not stable", e)
	}
	if f.count(`SELECT count(*) FROM correction_events WHERE kind='job_attempt_failed'`) != 8 || f.count(`SELECT count(*) FROM correction_events WHERE kind='job_retry'`) != 1 {
		t.Fatal("manual epoch lost error history")
	}
}

func TestCorrectionRetryEpochPreservesCursorAndCurrentAuthority(t *testing.T) {
	f := newCorrectionFixture(t)
	f.registerRule(nil, 1)
	l, e := f.repo.ClaimCorrectionJob(f.ctx)
	if e != nil || l == nil {
		t.Fatal(e)
	}
	cursor := f.ID()
	tx, e := f.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	_, seq, _, _, _, _ := f.jobState(l.JobID)
	if _, e = tx.Exec(`UPDATE correction_jobs SET cursor_kind='assessment',cursor_id=$2,processed_count=17,sequence=sequence+1 WHERE id=$1`, l.JobID, cursor); e != nil {
		t.Fatal(e)
	}
	var cid string
	if e = tx.QueryRow(`SELECT case_id::text FROM correction_jobs WHERE id=$1`, l.JobID).Scan(&cid); e != nil {
		t.Fatal(e)
	}
	if e = f.cEvent(tx, "job", l.JobID, "job_continued", "", cid, nil, seq+1, map[string]any{"processedCount": 17}); e != nil {
		t.Fatal(e)
	}
	if e = tx.Commit(); e != nil {
		t.Fatal(e)
	}
	if e = f.repo.FinishCorrectionJob(f.ctx, *l, correction.RetryWait, "source"); e != nil {
		t.Fatal(e)
	}
	_, seq, _, _, _, _ = f.jobState(l.JobID)
	if _, e = f.repo.RetryCorrectionJob(f.ctx, f.Access("reviewer_b", false), l.JobID, correction.RetryInput{ExpectedSequence: seq}); !errors.Is(e, auth.ErrForbidden) {
		t.Fatal("non-admin retry", e)
	}
	if _, e = f.repo.RetryCorrectionJob(f.ctx, f.Access("admin_a", false), l.JobID, correction.RetryInput{ExpectedSequence: seq}); e != nil {
		t.Fatal(e)
	}
	var ck string
	var count int
	if e = f.db.QueryRow(`SELECT cursor_id::text,processed_count FROM correction_jobs WHERE id=$1`, l.JobID).Scan(&ck, &count); e != nil || ck != cursor || count != 17 {
		t.Fatal("retry reset cursor", ck, count, e)
	}
	if e = f.repo.FinishCorrectionJob(f.ctx, *l, correction.Succeeded, ""); !errors.Is(e, correction.ErrLeaseLost) {
		t.Fatal("previous epoch committed", e)
	}
}
func TestCorrectionLeaseClaimSkipsLockedOtherJob(t *testing.T) {
	f := newCorrectionFixture(t)
	f.registerRule(nil, 1)
	f.registerRule(nil, 1)
	tx, e := f.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	var locked string
	if e = tx.QueryRow(`SELECT id::text FROM correction_jobs ORDER BY created_at,id LIMIT 1 FOR UPDATE`).Scan(&locked); e != nil {
		t.Fatal(e)
	}
	lease, e := f.repo.ClaimCorrectionJob(f.ctx)
	if e != nil || lease == nil || lease.JobID == locked {
		t.Fatal("locked job blocked another claim", lease, e)
	}
}
