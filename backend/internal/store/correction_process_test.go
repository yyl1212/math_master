package store_test

import (
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/question"
	"testing"
)

func (f *correctionFixture) cApproveAPI(cid string) correction.PlanRef {
	f.t.Helper()
	p := f.submitCorrectionPlan(f.createCorrectionPlan(cid, "author_b"), "author_b")
	v, e := f.repo.DecideCorrectionPlan(f.ctx, f.Access("reviewer_b", false), p.Ref, correction.DecisionInput{ExpectedSequence: p.Sequence, Decision: "approve", Reason: "Original answers, five positions and exact rational algorithm verified independently."})
	if e != nil {
		f.t.Fatal(e)
	}
	return v.Data.Plan.Ref
}
func (f *correctionFixture) cFinishRoots() {
	f.t.Helper()
	for {
		l, e := f.repo.ClaimCorrectionJob(f.ctx)
		if e != nil {
			f.t.Fatal(e)
		}
		if l == nil {
			return
		}
		if e = f.repo.FinishCorrectionJob(f.ctx, *l, correction.Succeeded, ""); e != nil {
			f.t.Fatal(e)
		}
	}
}
func (f *correctionFixture) cRunAll() {
	f.t.Helper()
	for step := 0; step < 100; step++ {
		l, e := f.repo.ClaimCorrectionJob(f.ctx)
		if e != nil {
			f.t.Fatal(e)
		}
		if l == nil {
			return
		}
		if _, e = f.repo.ProcessCorrectionJob(f.ctx, *l, 50); e != nil {
			f.t.Fatal(e)
		}
	}
	f.t.Fatal("bounded fixture did not drain")
}
func TestCorrectionProcessAtomic(t *testing.T) {
	for _, point := range []string{"result", "dependency", "qualification", "notification", "cursor"} {
		t.Run(point, func(t *testing.T) {
			f := newCorrectionFixture(t)
			aid, _ := f.cLegacyFailedFour()
			c := f.registerRule(nil, 1)
			f.cFinishRoots()
			f.cApproveAPI(c.ID)
			l, e := f.repo.ClaimCorrectionJob(f.ctx)
			if e != nil || l == nil {
				t.Fatal(e)
			}
			f.exec(`CREATE FUNCTION isolated_process_failure() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'isolated atomic process fault'; END $$`)
			table := "correction_results"
			event := "BEFORE INSERT"
			condition := ""
			switch point {
			case "dependency":
				table = "correction_dependencies"
			case "qualification":
				table = "correction_events"
				condition = "WHEN (NEW.kind='qualification_granted')"
			case "notification":
				table = "notifications"
			case "cursor":
				table = "correction_jobs"
				event = "BEFORE UPDATE"
				condition = "WHEN (NEW.cursor_id IS NOT NULL)"
			}
			f.exec(`CREATE TRIGGER isolated_process_failure ` + event + ` ON ` + table + ` FOR EACH ROW ` + condition + ` EXECUTE FUNCTION isolated_process_failure()`)
			if _, e = f.repo.ProcessCorrectionJob(f.ctx, *l, 50); e == nil {
				t.Fatal("fault did not reject transaction", point)
			}
			for _, q := range []string{`SELECT count(*) FROM correction_results`, `SELECT count(*) FROM correction_dependencies`, `SELECT count(*) FROM correction_events WHERE kind='qualification_granted'`, `SELECT count(*) FROM notifications`, `SELECT count(*) FROM correction_jobs WHERE cursor_id IS NOT NULL`} {
				if f.count(q) != 0 {
					t.Fatal("partial commit", point, q)
				}
			}
			f.exec(`DROP TRIGGER isolated_process_failure ON ` + table)
			batch, e := f.repo.ProcessCorrectionJob(f.ctx, *l, 50)
			if e != nil || batch.Processed != 1 || batch.State != correction.Succeeded {
				t.Fatal("same lease could not resume rolled-back item", batch, e)
			}
			if f.count(`SELECT count(*) FROM correction_results WHERE evidence_id=$1 AND status='corrected_passed' AND score=4 AND passed`, aid) != 1 || f.count(`SELECT count(*) FROM assessment_results WHERE attempt_id=$1 AND outcome='failed' AND score=3`, aid) != 1 || f.count(`SELECT count(*) FROM correction_events WHERE kind='qualification_granted'`) != 1 || f.count(`SELECT count(*) FROM notifications WHERE type='corrected'`) != 1 {
				t.Fatal("original/result/grant/notice inconsistent")
			}
		})
	}
}
func TestCorrectionStaleLeaseNoEvidenceWrites(t *testing.T) {
	f := newCorrectionFixture(t)
	f.cLegacyFailedFour()
	c := f.registerRule(nil, 1)
	f.cFinishRoots()
	f.cApproveAPI(c.ID)
	l, e := f.repo.ClaimCorrectionJob(f.ctx)
	if e != nil || l == nil {
		t.Fatal(e)
	}
	f.moveJobClock(l.JobID, true)
	if _, e = f.repo.ProcessCorrectionJob(f.ctx, *l, 50); !errors.Is(e, correction.ErrLeaseLost) {
		t.Fatal(e)
	}
	if f.count(`SELECT count(*) FROM correction_results`) != 0 || f.count(`SELECT count(*) FROM notifications`) != 0 {
		t.Fatal("expired lease committed")
	}
}
func TestCorrectionCrashResumeCursorAndNotificationDedup(t *testing.T) {
	f := newCorrectionFixture(t)
	a, _ := f.cLegacyFailedFour()
	b, _ := f.cSubmittedBasis("learner_b")
	c := f.registerRule(nil, 1)
	f.cFinishRoots()
	f.cApproveAPI(c.ID)
	high := a
	if b > high {
		high = b
	}
	f.exec(`CREATE FUNCTION isolated_process_failure() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'isolated second evidence interruption'; END $$`)
	f.exec(`CREATE TRIGGER isolated_process_failure BEFORE INSERT ON correction_results FOR EACH ROW WHEN (NEW.evidence_id='` + high + `'::uuid) EXECUTE FUNCTION isolated_process_failure()`)
	l, e := f.repo.ClaimCorrectionJob(f.ctx)
	if e != nil || l == nil {
		t.Fatal(e)
	}
	if _, e = f.repo.ProcessCorrectionJob(f.ctx, *l, 50); e == nil {
		t.Fatal("interruption missing")
	}
	if f.count(`SELECT count(*) FROM correction_results`) != 1 || f.count(`SELECT count(*) FROM correction_jobs WHERE processed_count=1 AND cursor_id IS NOT NULL`) != 1 {
		t.Fatal("first committed item not durable")
	}
	f.exec(`DROP TRIGGER isolated_process_failure ON correction_results`)
	if e = f.repo.FinishCorrectionJob(f.ctx, *l, correction.RetryWait, "database"); e != nil {
		t.Fatal(e)
	}
	f.moveJobClock(l.JobID, false)
	f.cRunAll()
	if f.count(`SELECT count(*) FROM correction_results`) != 2 || f.count(`SELECT count(*) FROM notifications WHERE type='corrected'`) != 2 {
		t.Fatal("resume duplicated or missed evidence")
	}
	for n := 0; n < 100; n++ {
		if _, e = f.repo.BackfillCorrections(f.ctx, 50); e != nil {
			t.Fatal(e)
		}
		f.cRunAll()
	}
	if f.count(`SELECT count(*) FROM correction_results`) != 2 || f.count(`SELECT count(*) FROM notifications WHERE type='corrected'`) != 2 || f.count(`SELECT count(*) FROM correction_events WHERE kind='qualification_granted'`) != 2 {
		t.Fatal("terminal replay duplicated results, grants or notices")
	}
}
func TestCorrectionProcessRootActiveLateTerminalAfterDone(t *testing.T) {
	f := newCorrectionFixture(t)
	v := f.createDiagnostic("learner_a")
	c := f.registerRule(nil, 1)
	f.cRunAll()
	if f.count(`SELECT count(*) FROM assessment_answers`) != 0 || f.count(`SELECT count(*) FROM correction_results`) != 0 || f.count(`SELECT count(*) FROM notifications WHERE type='checking'`) != 1 {
		t.Fatal("active attempt read/graded")
	}
	r, e := f.repo.SubmitAssessment(f.ctx, f.Access("learner_a", false), v.Summary.ID, f.answers(v, 5))
	if e != nil || r.Outcome != assessment.Affected {
		t.Fatal(e)
	}
	f.cApproveAPI(c.ID)
	f.cRunAll()
	if f.count(`SELECT count(*) FROM correction_results WHERE status='corrected_passed'`) != 1 {
		t.Fatal("late original answers not corrected")
	}
	q, e := f.repo.LearningEvidenceForTest(f.ctx, f.Access("learner_a", false), f.knowledge)
	if e != nil || !q.Qualified {
		t.Fatal(q, e)
	}
}
func TestCorrectionCumulativeResultsOtherCaseStillRestricts(t *testing.T) {
	f := newCorrectionFixture(t)
	aid, _ := f.cLegacyFailedFour()
	a := f.registerRule(nil, 1)
	b := f.registerRule(nil, 1)
	f.cFinishRoots()
	f.cApproveAPI(a.ID)
	f.cRunAll()
	v, e := f.repo.LearningEvidenceForTest(f.ctx, f.Access("learner_a", false), f.knowledge)
	if e != nil || v.Qualified {
		t.Fatal("only one case hid other", v, e)
	}
	f.cApproveAPI(b.ID)
	f.cRunAll()
	v, e = f.repo.LearningEvidenceForTest(f.ctx, f.Access("learner_a", false), f.knowledge)
	if e != nil || !v.Qualified {
		t.Fatal("cumulative basis not valid", v, e)
	}
	if f.count(`SELECT count(*) FROM correction_results WHERE evidence_id=$1 AND jsonb_array_length(handled_case_ids)=2`, aid) != 1 {
		t.Fatal("full cumulative cases lost")
	}
}
func TestCorrectionProcessWithdrawalWithoutMappingRetake(t *testing.T) {
	f := newCorrectionFixture(t)
	aid, b := f.cSubmittedBasis("learner_a")
	w := f.QWithdraw(question.WithdrawalTarget{Kind: "instance", ID: b.OriginalItems[0].Identity.ID, Version: 1})
	f.cFinishRoots()
	var cid string
	if e := f.db.QueryRow(`SELECT id::text FROM correction_cases WHERE withdrawal_id=$1`, w.EventID).Scan(&cid); e != nil {
		t.Fatal(e)
	}
	f.cApproveAPI(cid)
	f.cRunAll()
	if f.count(`SELECT count(*) FROM correction_results WHERE evidence_id=$1 AND status='retake_required' AND score IS NULL AND passed IS NULL`, aid) != 1 || f.count(`SELECT count(*) FROM notifications WHERE type='retake'`) != 1 {
		t.Fatal("unmapped withdrawal invented approved replacement")
	}
}

func TestCorrectionProcessThreeOfFiveStaysFailed(t *testing.T) {
	f := newCorrectionFixture(t)
	aid, _ := f.cLegacyFailedFour(3)
	c := f.registerRule(nil, 1)
	f.cFinishRoots()
	f.cApproveAPI(c.ID)
	f.cRunAll()
	if f.count(`SELECT count(*) FROM correction_results WHERE evidence_id=$1 AND status='corrected_failed' AND score=3 AND NOT passed`, aid) != 1 || f.count(`SELECT count(*) FROM correction_events WHERE kind='qualification_granted'`) != 0 {
		t.Fatal("three of five granted")
	}
}
func TestCorrectionStaleLeaseAtFinalCommitFence(t *testing.T) {
	f := newCorrectionFixture(t)
	f.cLegacyFailedFour()
	c := f.registerRule(nil, 1)
	f.cFinishRoots()
	f.cApproveAPI(c.ID)
	l, e := f.repo.ClaimCorrectionJob(f.ctx)
	if e != nil || l == nil {
		t.Fatal(e)
	}
	block, e := f.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer block.Rollback()
	var id string
	if e = block.QueryRow(`SELECT id::text FROM auth_users WHERE id=$1 FOR UPDATE`, f.ids["learner_a"]).Scan(&id); e != nil {
		t.Fatal(e)
	}
	done := make(chan error, 1)
	go func() { _, e := f.repo.ProcessCorrectionJob(f.ctx, *l, 50); done <- e }()
	waitCorrectionSQL(t, f, "%FROM auth_users WHERE id=%FOR UPDATE%")
	f.moveJobClock(l.JobID, true)
	if e = block.Commit(); e != nil {
		t.Fatal(e)
	}
	if e = <-done; !errors.Is(e, correction.ErrLeaseLost) {
		t.Fatal("old lease passed final fence", e)
	}
	if f.count(`SELECT count(*) FROM correction_results`) != 0 || f.count(`SELECT count(*) FROM correction_events WHERE kind='qualification_granted'`) != 0 || f.count(`SELECT count(*) FROM notifications`) != 0 {
		t.Fatal("late fence did not roll back all writes")
	}
}
func (f *correctionFixture) cReplacementPlan(cid string, b correction.Basis, oldPub string, version int, parent ...correction.PlanRef) correction.PlanRef {
	f.t.Helper()
	q := &f.questionInput.QuestionPackage
	q.Version = version
	q.Templates[0].Version = version
	q.Templates[0].ExplanationTemplate = "Original replacement explanation with unchanged actual question and parameters."
	q.Blueprints[0].Version = version
	q.Blueprints[0].Sources[0].Ref.Version = version
	sub := f.QApproved("author_b", "reviewer_a")
	f.QActivate(f.QPrepare(sub.ID))
	in := correctionPlanInput()
	if len(parent) > 0 {
		in.Parent = &parent[0]
	}
	for _, o := range b.EffectiveItems {
		var raw []byte
		var replacement question.Instance
		if e := f.db.QueryRow(`SELECT jsonb_set(i.body->'body','{identity,sha256}',to_jsonb(i.sha256)) FROM question_instances i JOIN question_publication_members m ON m.publication_id=$1 AND m.kind='instance' AND m.id=i.id AND m.version=i.version AND m.sha256=i.sha256 WHERE i.template_version=$2 AND i.body#>'{body,parameters}'=$3::jsonb ORDER BY i.id LIMIT 1`, *f.QHead(), version, corrJSON(o.Parameters)).Scan(&raw); e != nil {
			f.t.Fatal(e)
		}
		if e := json.Unmarshal(raw, &replacement); e != nil {
			f.t.Fatal(e)
		}
		in.Mappings = append(in.Mappings, correction.Mapping{Original: o.Identity, OriginalPublicationID: oldPub, Replacement: correction.PublishedInstance{Identity: replacement.Identity, PublicationID: *f.QHead()}})
	}
	created, e := f.repo.CreateCorrectionPlan(f.ctx, f.Access("admin_a", false), cid, in)
	if e != nil {
		f.t.Fatal(e)
	}
	plan := f.submitCorrectionPlan(*created.Data.Plan, "admin_a")
	r, e := f.repo.DecideCorrectionPlan(f.ctx, f.Access("reviewer_b", false), plan.Ref, correction.DecisionInput{ExpectedSequence: plan.Sequence, Decision: "approve", Reason: "Each fixed original and actual replacement was reviewed for equivalent input and exact rational answers."})
	if e != nil {
		f.t.Fatal(e)
	}
	return r.Data.Plan.Ref
}
func TestCorrectionWithdrawalCommitRace(t *testing.T) {
	for _, order := range []string{"withdrawal-first", "correction-first"} {
		t.Run(order, func(t *testing.T) {
			f := newCorrectionFixture(t)
			aid, b := f.cSubmittedBasis("learner_a")
			oldPub := *f.QHead()
			w := f.QWithdraw(question.WithdrawalTarget{Kind: "template", ID: b.OriginalItems[0].Template.ID, Version: 1})
			f.cFinishRoots()
			var cid string
			if e := f.db.QueryRow(`SELECT id::text FROM correction_cases WHERE withdrawal_id=$1`, w.EventID).Scan(&cid); e != nil {
				t.Fatal(e)
			}
			f.cReplacementPlan(cid, b, oldPub, 2)
			l, e := f.repo.ClaimCorrectionJob(f.ctx)
			if e != nil || l == nil {
				t.Fatal(e)
			}
			target := question.WithdrawalTarget{Kind: "template", ID: b.OriginalItems[0].Template.ID, Version: 2}
			if order == "withdrawal-first" {
				f.QWithdraw(target)
				if _, e = f.repo.ProcessCorrectionJob(f.ctx, *l, 50); e != nil {
					t.Fatal(e)
				}
				if f.count(`SELECT count(*) FROM correction_events WHERE kind='qualification_granted'`) != 0 {
					t.Fatal("withdrawn replacement granted")
				}
			} else {
				block, e := f.db.Begin()
				if e != nil {
					t.Fatal(e)
				}
				defer block.Rollback()
				var id string
				if e = block.QueryRow(`SELECT id::text FROM auth_users WHERE id=$1 FOR UPDATE`, f.ids["learner_a"]).Scan(&id); e != nil {
					t.Fatal(e)
				}
				processed := make(chan error, 1)
				go func() { _, e := f.repo.ProcessCorrectionJob(f.ctx, *l, 50); processed <- e }()
				waitCorrectionSQL(t, f, "%FROM auth_users WHERE id=%FOR UPDATE%")
				withdrawn := make(chan error, 1)
				a := f.Access("admin_a", true)
				in := question.WithdrawalInput{Target: target, ExpectedKnowledgeHead: f.KHead(), ExpectedQuestionHead: f.QHead(), Reason: "Actual replacement withdrawal while a correction holds the shared content locks."}
				go func() { _, e := f.repo.WithdrawQuestionVersion(f.ctx, a, in); withdrawn <- e }()
				waitCorrectionSQL(t, f, "%pg_advisory_xact_lock($1)%")
				if e = block.Commit(); e != nil {
					t.Fatal(e)
				}
				if e = <-processed; e != nil {
					t.Fatal(e)
				}
				if e = <-withdrawn; e != nil {
					t.Fatal(e)
				}
			}
			v, e := f.repo.LearningEvidenceForTest(f.ctx, f.Access("learner_a", false), f.knowledge)
			if e != nil || v.Qualified {
				t.Fatal("current qualification survived withdrawal", v, e)
			}
			if f.count(`SELECT count(*) FROM assessment_results WHERE attempt_id=$1 AND score=5`, aid) != 1 {
				t.Fatal("original changed")
			}
		})
	}
}
func TestCorrectionCumulativeResultsParentReplacementWithdrawn(t *testing.T) {
	f := newCorrectionFixture(t)
	aid, b := f.cSubmittedBasis("learner_a")
	oldPub := *f.QHead()
	w := f.QWithdraw(question.WithdrawalTarget{Kind: "template", ID: b.EffectiveItems[0].Template.ID, Version: 1})
	f.cFinishRoots()
	var a string
	if e := f.db.QueryRow(`SELECT id::text FROM correction_cases WHERE withdrawal_id=$1`, w.EventID).Scan(&a); e != nil {
		t.Fatal(e)
	}
	f.cReplacementPlan(a, b, oldPub, 2)
	f.cRunAll()
	var raw []byte
	var rid string
	if e := f.db.QueryRow(`SELECT id::text,basis#>'{body}' FROM correction_results WHERE evidence_id=$1 AND status='corrected_passed'`, aid).Scan(&rid, &raw); e != nil {
		t.Fatal(e)
	}
	if e := json.Unmarshal(raw, &b); e != nil {
		t.Fatal(e)
	}
	if f.count(`SELECT count(*) FROM correction_dependencies WHERE result_id=$1 AND role='audit' AND kind='template' AND version=2 AND positions=ARRAY[1,2,3,4,5]`, rid) != 1 {
		t.Fatal("replacement audit positions missing")
	}
	if f.count(`SELECT count(*) FROM correction_dependencies WHERE result_id=$1 AND role='effective' AND kind='template' AND version=1`, rid) != 0 {
		t.Fatal("fully replaced old template remained effective")
	}
	aPub := *f.QHead()
	w2 := f.QWithdraw(question.WithdrawalTarget{Kind: "template", ID: b.EffectiveItems[0].Template.ID, Version: 2})
	f.cFinishRoots()
	var c string
	if e := f.db.QueryRow(`SELECT id::text FROM correction_cases WHERE withdrawal_id=$1`, w2.EventID).Scan(&c); e != nil {
		t.Fatal(e)
	}
	f.cReplacementPlan(c, b, aPub, 3)
	f.cRunAll()
	v, e := f.repo.LearningEvidenceForTest(f.ctx, f.Access("learner_a", false), f.knowledge)
	if e != nil || !v.Qualified {
		t.Fatal("second replacement did not compose", v, e)
	}
	if f.count(`SELECT count(*) FROM correction_results WHERE evidence_id=$1 AND case_id=$2 AND status='corrected_passed' AND basis#>'{body,parentResultIds}' @> jsonb_build_array($3::text) AND handled_case_ids @> jsonb_build_array($4::text)`, aid, c, rid, a) != 1 {
		t.Fatal("cumulative parent/handled proof lost")
	}
}

func TestCorrectionProcessPracticeNeverGrantsQualification(t *testing.T) {
	f := newCorrectionFixture(t)
	p := f.createPractice("learner_a")
	var raw string
	if e := f.db.QueryRow(`SELECT (body#>>'{body,body,correctNumeric,numerator}')||'/'||(body#>>'{body,body,correctNumeric,denominator}') FROM question_instances WHERE id=$1 AND version=$2`, p.Question.Instance.ID, p.Question.Instance.Version).Scan(&raw); e != nil {
		t.Fatal(e)
	}
	if _, e := f.repo.AnswerPractice(f.ctx, f.Access("learner_a", false), p.Summary.ID, assessment.Answer{Kind: "numeric", Raw: &raw}); e != nil {
		t.Fatal(e)
	}
	c := f.registerRule(nil, 1)
	f.cFinishRoots()
	f.cApproveAPI(c.ID)
	f.cRunAll()
	if f.count(`SELECT count(*) FROM correction_results WHERE evidence_kind='practice' AND status='corrected_passed' AND score IS NULL AND passed IS NULL AND correctness='[true]'::jsonb`) != 1 || f.count(`SELECT count(*) FROM correction_events WHERE kind='qualification_granted'`) != 0 || f.count(`SELECT count(*) FROM learning_records`) != 0 || f.count(`SELECT count(*) FROM learning_unlocks`) != 0 {
		t.Fatal("practice fabricated formal progress")
	}
}
func TestCorrectionProcessAwaitingApprovalHasNoScore(t *testing.T) {
	f := newCorrectionFixture(t)
	aid, _ := f.cLegacyFailedFour()
	f.registerRule(nil, 1)
	f.cRunAll()
	if f.count(`SELECT count(*) FROM correction_results WHERE evidence_id=$1 AND status='awaiting_review' AND score IS NULL AND passed IS NULL`, aid) != 1 || f.count(`SELECT count(*) FROM notifications WHERE type='checking'`) != 1 {
		t.Fatal("unapproved evidence mislabeled")
	}
}
func TestCorrectionProcessCurrentAccountRecheckedNoPersonnelQuota(t *testing.T) {
	f := newCorrectionFixture(t)
	f.cLegacyFailedFour()
	c := f.registerRule(nil, 1)
	f.cFinishRoots()
	f.cApproveAPI(c.ID)
	l, e := f.repo.ClaimCorrectionJob(f.ctx)
	if e != nil || l == nil {
		t.Fatal(e)
	}
	f.exec(`UPDATE auth_users SET must_change_password=true WHERE id=$1`, f.ids["learner_a"])
	if _, e = f.repo.ProcessCorrectionJob(f.ctx, *l, 50); !errors.Is(e, auth.ErrPasswordChangeRequired) {
		t.Fatal("disabled account was not checked", e)
	}
	if f.count(`SELECT count(*) FROM correction_results`) != 0 {
		t.Fatal("inactive account had worker writes")
	}
	f.exec(`UPDATE auth_users SET must_change_password=false WHERE id=$1`, f.ids["learner_a"])
	if _, e = f.repo.ProcessCorrectionJob(f.ctx, *l, 50); e != nil {
		t.Fatal(e)
	}
	if f.count(`SELECT count(*) FROM correction_rate_limits WHERE owner_user_id=$1`, f.ids["learner_a"]) != 0 {
		t.Fatal("system consumed personnel quota")
	}
}

func TestCorrectionCumulativeResultsIncompatibleApprovedMappings(t *testing.T) {
	f := newCorrectionFixture(t)
	aid, b := f.cSubmittedBasis("learner_a")
	pub := *f.QHead()
	w := f.QWithdraw(question.WithdrawalTarget{Kind: "template", ID: b.OriginalItems[0].Template.ID, Version: 1})
	f.cFinishRoots()
	var cid string
	if e := f.db.QueryRow(`SELECT id::text FROM correction_cases WHERE withdrawal_id=$1`, w.EventID).Scan(&cid); e != nil {
		t.Fatal(e)
	}
	f.cReplacementPlan(cid, b, pub, 2)
	f.cReplacementPlan(cid, b, pub, 3)
	f.cRunAll()
	if f.count(`SELECT count(*) FROM correction_results WHERE evidence_id=$1 AND status='awaiting_review' AND reason='conflicting_basis' AND score IS NULL AND passed IS NULL AND jsonb_array_length(basis#>'{body,planRefs}')=2`, aid) != 2 || f.count(`SELECT count(*) FROM correction_events WHERE kind='qualification_granted'`) != 0 {
		t.Fatal("incompatible approval picked timestamp or lost proofs")
	}
	if f.count(`SELECT count(DISTINCT version) FROM correction_dependencies WHERE role='audit' AND kind='template' AND version IN (2,3)`) != 2 {
		t.Fatal("conflict lost complete audit union")
	}
}
func TestCorrectionCumulativeResultsApprovedChildRetakeSupersedesPass(t *testing.T) {
	f := newCorrectionFixture(t)
	aid, b := f.cSubmittedBasis("learner_a")
	pub := *f.QHead()
	w := f.QWithdraw(question.WithdrawalTarget{Kind: "template", ID: b.OriginalItems[0].Template.ID, Version: 1})
	f.cFinishRoots()
	var cid string
	if e := f.db.QueryRow(`SELECT id::text FROM correction_cases WHERE withdrawal_id=$1`, w.EventID).Scan(&cid); e != nil {
		t.Fatal(e)
	}
	parent := f.cReplacementPlan(cid, b, pub, 2)
	f.cRunAll()
	f.questionInput.QuestionPackage.Templates[0].PromptTemplate = "Use the revised wording to calculate {a} + {b}."
	f.cReplacementPlan(cid, b, pub, 3, parent)
	f.cRunAll()
	if f.count(`SELECT count(*) FROM correction_results WHERE evidence_id=$1 AND status='retake_required' AND reason='intent_changed'`, aid) != 1 {
		t.Fatal("changed intent was not retake")
	}
	v, e := f.repo.LearningEvidenceForTest(f.ctx, f.Access("learner_a", false), f.knowledge)
	if e != nil || v.Qualified {
		t.Fatal("explicit child let ancestor pass return", v, e)
	}
}
