package store_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/notification"
	"github.com/yyl1212/math_master/backend/internal/question"
	"github.com/yyl1212/math_master/backend/internal/store"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

func correctionCapacityContext(t *testing.T, f *correctionFixture) {
	t.Helper()
	deadline, ok := t.Deadline()
	if !ok {
		t.Fatal("capacity needs a bounded Go test")
	}
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	f.ctx = ctx
	t.Cleanup(cancel)
}
func correctionCapacityMetadata(t *testing.T, value any) {
	t.Helper()
	b, e := json.Marshal(value)
	if e != nil || len(b) > correction.MaxResponseBytes {
		t.Fatal("response bound", len(b), e)
	}
	for _, bad := range []string{"correctNumeric", "originalAnswers", "frozenBody", "explanation", "Original independently reviewed"} {
		if strings.Contains(string(b), bad) {
			t.Fatal("metadata body leak", bad)
		}
	}
}
func correctionCapacitySet(t *testing.T, db *sql.DB, query string, args ...any) map[string]bool {
	t.Helper()
	rows, e := db.Query(query, args...)
	if e != nil {
		t.Fatal(e)
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var id string
		if e = rows.Scan(&id); e != nil {
			t.Fatal(e)
		}
		if out[id] {
			t.Fatal("duplicate", id)
		}
		out[id] = true
	}
	if e = rows.Err(); e != nil {
		t.Fatal(e)
	}
	return out
}

// A real approved original practice supplies its immutable seal, exact answer,
// dependencies and publication facts. SQL creates historical volume while all
// FK, hash, ownership, immutable and deferred guards stay enabled.
func (f *correctionFixture) cCapacityPractices() string {
	f.t.Helper()
	p := f.createPractice("learner_a")
	// Safe question views intentionally omit answers; use the actual frozen item.
	var value string
	if e := f.db.QueryRow(`SELECT (i.body#>>'{body,body,correctNumeric,numerator}')||'/'||(i.body#>>'{body,body,correctNumeric,denominator}') FROM practice_attempts a JOIN question_instances i ON i.id=a.seal#>>'{body,items,0,instance,id}' AND i.version=(a.seal#>>'{body,items,0,instance,version}')::int WHERE a.id=$1`, p.Summary.ID).Scan(&value); e != nil {
		f.t.Fatal(e)
	}
	if _, e := f.repo.AnswerPractice(f.ctx, f.Access("learner_a", false), p.Summary.ID, assessment.Answer{Kind: "numeric", Raw: &value}); e != nil {
		f.t.Fatal(e)
	}
	tx, e := f.db.BeginTx(f.ctx, nil)
	if e != nil {
		f.t.Fatal(e)
	}
	defer tx.Rollback()
	_, e = tx.ExecContext(f.ctx, `CREATE TEMP TABLE correction_volume ON COMMIT DROP AS SELECT gen_random_uuid() id,CASE WHEN n%2=0 THEN $2::uuid ELSE $1::uuid END owner FROM generate_series(2,10000) n`, f.ids["learner_a"], f.ids["learner_b"])
	if e != nil {
		f.t.Fatal(e)
	}
	_, e = tx.ExecContext(f.ctx, `INSERT INTO practice_attempts(id,owner_user_id,knowledge_id,knowledge_version,knowledge_sha256,knowledge_publication_id,question_publication_id,seal,seal_bytes,seal_sha256,state,created_at,expires_at,terminal_at,answer,correct)
 SELECT v.id,v.owner,a.knowledge_id,a.knowledge_version,a.knowledge_sha256,a.knowledge_publication_id,a.question_publication_id,a.seal,a.seal_bytes,a.seal_sha256,a.state,a.created_at,a.expires_at,a.terminal_at,a.answer,a.correct FROM correction_volume v CROSS JOIN practice_attempts a WHERE a.id=$1`, p.Summary.ID)
	if e != nil {
		f.t.Fatal(e)
	}
	_, e = tx.ExecContext(f.ctx, `INSERT INTO learning_evidence_dependencies(evidence_kind,evidence_id,owner_user_id,kind,id,version,sha256) SELECT 'practice',v.id,v.owner,d.kind,d.id,d.version,d.sha256 FROM correction_volume v CROSS JOIN learning_evidence_dependencies d WHERE d.evidence_kind='practice' AND d.evidence_id=$1`, p.Summary.ID)
	if e != nil {
		f.t.Fatal(e)
	}
	if e = tx.Commit(); e != nil {
		f.t.Fatal(e)
	}
	return p.Summary.ID
}
func (f *correctionFixture) cCapacityClonePlans(seed correction.PlanRef, n int) {
	f.t.Helper()
	var raw []byte
	var env struct {
		Body struct {
			Input correction.PlanInput `json:"input"`
			Proof correction.PlanProof `json:"proof"`
		} `json:"body"`
	}
	if e := f.db.QueryRow(`SELECT frozen_body FROM correction_plans WHERE id=$1 AND version=$2`, seed.ID, seed.Version).Scan(&raw); e != nil {
		f.t.Fatal(e)
	}
	if e := json.Unmarshal(raw, &env); e != nil || len(env.Body.Proof.QuestionApprovalIDs) == 0 || len(env.Body.Proof.Authors) == 0 {
		f.t.Fatal("real independent proof required", e)
	}
	tx, e := f.db.BeginTx(f.ctx, nil)
	if e != nil {
		f.t.Fatal(e)
	}
	defer tx.Rollback()
	for i := 0; i < n; i++ {
		cid, pid := f.ID(), f.ID()
		_, e = tx.ExecContext(f.ctx, `INSERT INTO correction_cases(id,kind,rule_version,scope_kind,cutoff,creator_user_id) VALUES($1,'grading_rule',1,'all',clock_timestamp(),$2)`, cid, f.ids["admin_a"])
		if e != nil {
			f.t.Fatal(e)
		}
		if e = f.cEvent(tx, "case", cid, "case_registered", f.ids["admin_a"], cid, nil, 1, map[string]any{"caseId": cid}); e != nil {
			f.t.Fatal(e)
		}
		if _, e = tx.ExecContext(f.ctx, `UPDATE correction_cases SET sealed=true WHERE id=$1`, cid); e != nil {
			f.t.Fatal(e)
		}
		in, proof := env.Body.Input, env.Body.Proof
		proof.CaseID = cid
		frozen, h, e := correction.Canonical("correction-plan-v1", struct {
			Input correction.PlanInput `json:"input"`
			Proof correction.PlanProof `json:"proof"`
		}{in, proof})
		if e != nil {
			f.t.Fatal(e)
		}
		if _, e = tx.ExecContext(f.ctx, `INSERT INTO correction_plans(id,version,case_id,creator_user_id,body) VALUES($1,1,$2,$3,$4)`, pid, cid, f.ids["author_b"], corrJSON(in)); e != nil {
			f.t.Fatal(e)
		}
		ver := 1
		if e = f.cEvent(tx, "plan", pid, "plan_created", f.ids["author_b"], cid, &ver, 1, in); e != nil {
			f.t.Fatal(e)
		}
		if e = f.cEvent(tx, "plan", pid, "plan_submitted", f.ids["author_b"], cid, &ver, 2, map[string]any{"digest": h}); e != nil {
			f.t.Fatal(e)
		}
		if _, e = tx.ExecContext(f.ctx, `UPDATE correction_plans SET status='pending',sequence=2,sealed=true,frozen_body=$2,frozen_bytes=$3,frozen_digest=$4 WHERE id=$1`, pid, string(frozen), frozen, h); e != nil {
			f.t.Fatal(e)
		}
		if e = f.cEvent(tx, "plan", pid, "plan_approved", f.ids["reviewer_b"], cid, &ver, 3, map[string]any{"decision": "approve", "digest": h}); e != nil {
			f.t.Fatal(e)
		}
		if _, e = tx.ExecContext(f.ctx, `UPDATE correction_plans SET status='approved',sequence=3 WHERE id=$1`, pid); e != nil {
			f.t.Fatal(e)
		}
	}
	if e = tx.Commit(); e != nil {
		f.t.Fatal(e)
	}
}
func correctionCapacityOriginalHash(t *testing.T, f *correctionFixture) string {
	t.Helper()
	var h string
	if e := f.db.QueryRow(`SELECT encode(sha256(convert_to(string_agg(id::text||encode(seal_bytes,'hex')||answer::text||correct::text,'' ORDER BY id),'UTF8')),'hex') FROM practice_attempts`).Scan(&h); e != nil {
		t.Fatal(e)
	}
	return h
}
func TestCorrectionCapacityImpact(t *testing.T) {
	started := time.Now()
	f := newCorrectionFixture(t)
	correctionCapacityContext(t, f)
	// 999 earlier cutoffs do not incorrectly capture the later 10,000 attempts.
	c := f.registerRule(nil, 1)
	seed := f.cApproveAPI(c.ID)
	f.cFinishRoots()
	f.cCapacityClonePlans(seed, 998)
	first := f.cCapacityPractices()
	expected := correctionCapacitySet(t, f.db, `SELECT id::text FROM practice_attempts`)
	if len(expected) != 10000 {
		t.Fatal("must retain 10000", len(expected))
	}
	original := correctionCapacityOriginalHash(t, f)
	var instance question.Identity
	if e := f.db.QueryRow(`SELECT seal#>>'{body,items,0,instance,id}',(seal#>>'{body,items,0,instance,version}')::int,seal#>>'{body,items,0,instance,sha256}' FROM practice_attempts WHERE id=$1`, first).Scan(&instance.ID, &instance.Version, &instance.SHA256); e != nil {
		t.Fatal(e)
	}
	wid := f.legacyCorrectionWithdrawal(instance)
	created, e := f.repo.CreateCorrectionCase(f.ctx, f.Access("admin_a", false), correction.CaseInput{Kind: correction.WithdrawalCase, Withdrawal: &correction.WithdrawalRef{Space: "question", ID: wid}})
	if e != nil || created.Data.Case == nil {
		t.Fatal("exact withdrawal case", e)
	}
	cid := created.Data.Case.ID
	// Finish only this fixture's impact root, then the actual approved-plan worker
	// must scan every affected original evidence, not a representative subset.
	f.cFinishRoots()
	f.cApproveAPI(cid)
	if f.count(`SELECT count(*) FROM correction_cases`) != 1000 || f.count(`SELECT count(*) FROM correction_plans WHERE status='approved'`) != 1000 {
		t.Fatal("must retain 1000 cases and 1000 approved plans")
	}
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	total, batches := 0, 0
	var previous string
	var largestBatchBytes uint64
	for total < 10000 {
		l, e := f.repo.ClaimCorrectionJob(f.ctx)
		if e != nil || l == nil {
			t.Fatal("unfinished evidence lost", total, e)
		}
		var batchBefore, batchAfter runtime.MemStats
		runtime.ReadMemStats(&batchBefore)
		began := time.Now()
		v, e := f.repo.ProcessCorrectionJob(f.ctx, *l, 50)
		if e != nil || v.Processed < 1 || v.Processed > 50 || time.Since(began) >= 30*time.Second {
			t.Fatal("finite body", total, v, e)
		}
		runtime.ReadMemStats(&batchAfter)
		allocated := batchAfter.TotalAlloc - batchBefore.TotalAlloc
		if allocated > largestBatchBytes {
			largestBatchBytes = allocated
		}
		if allocated > 32<<20 {
			t.Fatal("this one-item fixture loaded excessive bodies in one bounded batch", allocated)
		}
		total += v.Processed
		batches++
		if batches%20 == 0 {
			t.Logf("capacity worker processed=%d elapsed=%s", total, time.Since(started))
		}
		var cursor string
		if e = f.db.QueryRow(`SELECT cursor_id::text FROM correction_jobs WHERE id=$1`, l.JobID).Scan(&cursor); e != nil || cursor <= previous {
			t.Fatal("cursor failed to advance", e)
		}
		previous = cursor
		if total < 10000 && !v.Remaining || total == 10000 && v.Remaining {
			t.Fatal("remaining count", total, v)
		}
	}
	runtime.GC()
	runtime.ReadMemStats(&after)
	if int64(after.HeapAlloc)-int64(before.HeapAlloc) > 32<<20 {
		t.Fatal("bounded processing retained full historical bodies")
	}
	actual := correctionCapacitySet(t, f.db, `SELECT evidence_id::text FROM correction_results WHERE case_id=$1 AND evidence_kind='practice' AND status='retake_required' AND sealed`, cid)
	if !reflect.DeepEqual(actual, expected) || total != 10000 || batches != 200 {
		t.Fatal("complete worker expected set", len(actual), total, batches)
	}
	if f.count(`SELECT count(*) FROM correction_results WHERE score IS NOT NULL OR passed IS NOT NULL`) != 0 || f.count(`SELECT count(*) FROM correction_events WHERE kind='qualification_granted'`) != 0 || f.count(`SELECT count(*) FROM learning_qualification_events`) != 0 {
		t.Fatal("practice/withdrawal fabricated score or grant")
	}
	if correctionCapacityOriginalHash(t, f) != original {
		t.Fatal("original seal or answer changed")
	}
	t.Logf("capacity worker complete elapsed=%s", time.Since(started))
	cases := correctionCapacitySet(t, f.db, `SELECT id::text FROM correction_cases`)
	seen := map[string]bool{}
	q := correction.Query{Limit: 50}
	for {
		page, e := f.repo.ListCorrectionCases(f.ctx, f.Access("reviewer_b", false), q)
		if e != nil {
			t.Fatal(e)
		}
		correctionCapacityMetadata(t, page)
		for _, m := range page.Data.Items {
			if seen[m.ID] {
				t.Fatal("case repeat")
			}
			seen[m.ID] = true
			plans, e := f.repo.ListCorrectionPlans(f.ctx, f.Access("reviewer_b", false), m.ID, correction.Query{Limit: 50})
			if e != nil || len(plans.Data.Items) != 1 || plans.Data.NextCursor != nil || plans.Data.Items[0].Status != correction.Approved {
				t.Fatal("complete per-case approved set", e)
			}
			correctionCapacityMetadata(t, plans)
		}
		if page.Data.NextCursor == nil {
			break
		}
		q.Cursor = *page.Data.NextCursor
	}
	if !reflect.DeepEqual(cases, seen) {
		t.Fatal("complete case set")
	}
	t.Logf("capacity metadata complete elapsed=%s", time.Since(started))
lines, explainErr := store.CorrectionBackfillDiagnosticForTest(f.ctx,f.db,c.ID);if explainErr!=nil{t.Fatal(explainErr)};for _,line:=range lines{t.Log("BACKFILL_POST_WORKER_PLAN",line)}
backfillStarted:=time.Now()

	// Rotate until every case has both required root and approved-plan outboxes.
	for i := 0; i < 100 && (f.count(`SELECT count(DISTINCT case_id) FROM correction_jobs WHERE source_key LIKE 'case:%'`) != 1000 || f.count(`SELECT count(*) FROM correction_cases WHERE last_backfill_at IS NULL`) > 0); i++ {
		backfillStepStarted:=time.Now();n, e := f.repo.BackfillCorrections(f.ctx, 50);t.Logf("BACKFILL_POST_WORKER step=%d count=%d duration=%s total=%s",i,n,time.Since(backfillStepStarted),time.Since(backfillStarted))
		if e != nil || n > 50 {
			t.Fatal("global registration bound", n, e)
		}
	}
	roots := correctionCapacitySet(t, f.db, `SELECT case_id::text FROM correction_jobs WHERE source_key LIKE 'case:%'`)
	plansExpected := correctionCapacitySet(t, f.db, `SELECT id::text FROM correction_plans WHERE status='approved'`)
	plansActual := correctionCapacitySet(t, f.db, `SELECT plan_id::text FROM correction_jobs WHERE type='approved_plan'`)
	if !reflect.DeepEqual(plansExpected, plansActual) {
		t.Fatal("complete approved-plan outbox set", len(plansActual))
	}
	if !reflect.DeepEqual(roots, cases) || f.count(`SELECT count(*) FROM correction_cases WHERE last_backfill_at IS NULL`) != 0 {
		t.Fatal("rotation starved cases", len(roots))
	}
	// Full owner paging also scans all 10,000 delivered result notifications.
	correctionCapacityNoticePages(t, f, expected, 10000)
	t.Logf("CORRECTION_CAPACITY cases=1000 plans=1000 affected=10000 workerBatches=200 completeSets=true heapDelta=%d allocatedBytes=%d maxBatchBytes=%d elapsed=%s", int64(after.HeapAlloc)-int64(before.HeapAlloc), after.TotalAlloc-before.TotalAlloc, largestBatchBytes, time.Since(started))
}
func correctionCapacityNoticePages(t *testing.T, f *correctionFixture, expected map[string]bool, want int) {
	t.Helper()
	seen := map[string]bool{}
	var foreignCursor string
	for _, actor := range []string{"learner_a", "learner_b"} {
		q := notification.Query{Limit: 50}
		for {
			page, e := f.repo.ListNotifications(f.ctx, f.Access(actor, false), q)
			if e != nil {
				t.Fatal(e)
			}
			correctionCapacityMetadata(t, page)
			if len(page.Data.Items) > 50 {
				t.Fatal("notification page unbounded")
			}
			for _, m := range page.Data.Items {
				if seen[m.Evidence.ID] || !expected[m.Evidence.ID] {
					t.Fatal("foreign/repeat evidence notice")
				}
				seen[m.Evidence.ID] = true
				if m.ReadAt != nil {
					t.Fatal("paging changed read state")
				}
			}
			if page.Data.NextCursor == nil {
				break
			}
			q.Cursor = *page.Data.NextCursor
			if actor == "learner_a" && foreignCursor == "" {
				foreignCursor = q.Cursor
			}
		}
	}
	if len(seen) != want || !reflect.DeepEqual(seen, expected) {
		t.Fatal("complete notification expected set", len(seen), want)
	}
	page, e := f.repo.ListNotifications(f.ctx, f.Access("learner_b", false), notification.Query{Limit: 50, Cursor: foreignCursor})
	if e != nil {
		t.Fatal(e)
	}
	for _, m := range page.Data.Items {
		var owner string
		if e = f.db.QueryRow(`SELECT owner_user_id::text FROM notifications WHERE id=$1`, m.ID).Scan(&owner); e != nil || owner != f.ids["learner_b"] {
			t.Fatal("cursor expanded owner", e)
		}
	}
}
func TestCorrectionCapacityNotifications(t *testing.T) {
	began := time.Now()
	f := newCorrectionFixture(t)
	correctionCapacityContext(t, f)
	f.cCapacityPractices()
	c := f.registerRule(nil, 1)
	expected := correctionCapacitySet(t, f.db, `SELECT id::text FROM practice_attempts`)
	if len(expected) != 10000 {
		t.Fatal("must retain 10000 source evidence")
	}
	for repeat := 0; repeat < 2; repeat++ {
		q := ""
		count := 0
		for count < 10000 {
			rows, e := f.db.QueryContext(f.ctx, `SELECT id::text,owner_user_id::text FROM practice_attempts WHERE ($1::text='' OR id>$1::uuid) ORDER BY id LIMIT 50`, q)
			if e != nil {
				t.Fatal(e)
			}
			type source struct{ id, owner string }
			batch := []source{}
			for rows.Next() {
				var s source
				if e = rows.Scan(&s.id, &s.owner); e != nil {
					t.Fatal(e)
				}
				batch = append(batch, s)
			}
			e = rows.Err()
			rows.Close()
			if e != nil || len(batch) == 0 {
				t.Fatal("source scan stalled", e)
			}
			ctx, stop := context.WithTimeout(f.ctx, 8*time.Second)
			tx, e := f.db.BeginTx(ctx, nil)
			if e != nil {
				stop()
				t.Fatal(e)
			}
			for _, s := range batch {
				e = store.CorrectionAppendNotificationForTest(ctx, tx, s.owner, notification.Source{DedupKey: "checking:" + c.ID + ":practice:" + s.id, Type: notification.Checking, Evidence: correction.EvidenceRef{Kind: correction.PracticeEvidence, ID: s.id}, CaseID: c.ID})
				if e != nil {
					_ = tx.Rollback()
					stop()
					t.Fatal(e)
				}
			}
			e = tx.Commit()
			stop()
			if e != nil {
				t.Fatal(e)
			}
			count += len(batch)
			q = batch[len(batch)-1].id
		}
		if count != 10000 || f.count(`SELECT count(*) FROM notifications`) != 10000 {
			t.Fatal("same-source rerun duplicated/missed notices", repeat, count)
		}
	}
	correctionCapacityNoticePages(t, f, expected, 10000)
	for _, actor := range []string{"learner_a", "learner_b"} {
		count, e := f.repo.ReadNotificationCount(f.ctx, f.Access(actor, false))
		if e != nil || count.Data.Count != 5000 {
			t.Fatal("unread expected set", actor, count, e)
		}
		var id string
		if e = f.db.QueryRow(`SELECT id::text FROM notifications WHERE owner_user_id=$1 ORDER BY id LIMIT 1`, f.ids[actor]).Scan(&id); e != nil {
			t.Fatal(e)
		}
		key := f.Access(actor, false)
		first, e := f.repo.MarkNotificationRead(f.ctx, key, id)
		if e != nil {
			t.Fatal(e)
		}
		again, e := f.repo.MarkNotificationRead(f.ctx, key, id)
		if e != nil || !reflect.DeepEqual(first, again) {
			t.Fatal("first read/replay changed", e)
		}
		other := "learner_b"
		if actor == other {
			other = "learner_a"
		}
		if _, e = f.repo.ReadNotification(f.ctx, f.Access(other, false), id); !errors.Is(e, auth.ErrNotFound) {
			t.Fatal("cross owner", e)
		}
	}
	if f.count(`SELECT count(*) FROM notification_reads`) != 2 || f.count(`SELECT count(*) FROM correction_rate_limits WHERE scope='notification-read'`) != 2 || f.count(`SELECT count(*) FROM correction_events WHERE kind='qualification_granted'`) != 0 {
		t.Fatal("repeat granted/read/quota duplicated")
	}
	t.Logf("CORRECTION_CAPACITY notifications=10000 sameSourceReruns=10000 completeOwnerSets=true immutableFirstReads=2 elapsed=%s", time.Since(began))
}
