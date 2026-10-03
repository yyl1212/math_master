package store_test

import (
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"github.com/yyl1212/math_master/backend/internal/store"
	"testing"
	"time"
)

func TestCorrectionBroadRuleExposure(t *testing.T) {
	f := newCorrectionFixture(t)
	id, _, p := f.cReadableResult()
	active := f.timedAssessment(time.Minute)
	for _, plan := range []bool{false, true} {
		before := f.exposureSequence("learner_a")
		var e error
		if plan {
			_, e = f.repo.ReadCorrectionPlan(f.ctx, f.Access("reviewer_a", false), p, true)
		} else {
			v, err := f.repo.ReadOwnCorrection(f.ctx, f.Access("learner_a", false), id, true)
			e = err
			if len(v.Data.Items) != 0 || v.Data.PlanReason != nil {
				t.Fatal("blocked body delivered")
			}
		}
		if !plan && (!errors.Is(e, correction.ErrAnswerOverlap) || f.exposureSequence("learner_a") != before) {
			t.Fatal("broad overlap", e)
		}
		if plan && e != nil {
			t.Fatal("other reader's assessment blocks reviewer", e)
		}
	}
	f.exec(`UPDATE assessment_attempts SET state='abandoned',terminal_at=clock_timestamp() WHERE id=$1`, active)
	if _, e := f.repo.ReadOwnCorrection(f.ctx, f.Access("learner_a", false), id, true); e != nil {
		t.Fatal(e)
	}
	// All candidate templates, including otherwise unseen generated instances,
	// must observe the scope exposure in the original selection path.
	_, e := f.repo.CreateAssessment(f.ctx, f.Access("learner_a", false), f.assessmentInput(assessment.ModeDiagnostic))
	var ready *learning.NotReadyError
	if !errors.As(e, &ready) {
		t.Fatal("broad reason bypassed 30-minute selection cooldown", e)
	}
}
func TestCorrectionReplacementTemplateExposure(t *testing.T) {
	f := newCorrectionFixture(t)
	aid, b := f.cLegacyFailedFour()
	orig := *f.QHead()
	w := f.QWithdraw(question.WithdrawalTarget{Kind: "template", ID: b.OriginalItems[0].Template.ID, Version: 1})
	var cid string
	if e := f.db.QueryRow(`SELECT id::text FROM correction_cases WHERE withdrawal_id=$1`, w.EventID).Scan(&cid); e != nil {
		t.Fatal(e)
	}
	f.cFinishRoots()
	p := f.cReplacementPlan(cid, b, orig, 2)
	f.cRunAll()
	var id string
	if e := f.db.QueryRow(`SELECT id::text FROM correction_results WHERE evidence_id=$1 AND plan_id=$2`, aid, p.ID).Scan(&id); e != nil {
		t.Fatal(e)
	}
	f.setLearningTarget("workflow-fractions", "lf-five", 0)
	f.blueprint.Version = 2
	if e := f.db.QueryRow(`SELECT sha256 FROM question_blueprints WHERE id=$1 AND version=2`, f.blueprint.ID).Scan(&f.blueprint.SHA256); e != nil {
		t.Fatal(e)
	}
	originalID := b.OriginalItems[0].Identity
	replacement := f.items[0].Instance
	if originalID == replacement {
		t.Fatal("fixture must replace instance identity")
	}
	// Use five other actual instances sharing only the replacement template.
	rows, e := f.db.Query(`SELECT i.id,i.version,i.sha256,i.template_id,i.template_version,i.template_sha256,m.evidence FROM question_instances i JOIN question_publication_members m ON m.publication_id=$1 AND m.kind='instance' AND m.id=i.id AND m.version=i.version AND m.sha256=i.sha256 WHERE i.template_version=2 AND NOT EXISTS(SELECT 1 FROM correction_results cr CROSS JOIN jsonb_array_elements(cr.basis#>'{body,effectiveItems}') b WHERE cr.id=$2 AND b#>>'{identity,id}'=i.id) ORDER BY i.id LIMIT 5`, *f.QHead(), id)
	if e != nil {
		t.Fatal(e)
	}
	prototype := f.items[0]
	f.items = nil
	for rows.Next() {
		item := prototype
		var ti question.Identity
		var proof []byte
		if e = rows.Scan(&item.Instance.ID, &item.Instance.Version, &item.Instance.SHA256, &ti.ID, &ti.Version, &ti.SHA256, &proof); e != nil {
			t.Fatal(e)
		}
		item.Template = &ti
		if e = json.Unmarshal(proof, &item.Approval); e != nil {
			t.Fatal(e)
		}
		item.Position = len(f.items) + 1
		f.items = append(f.items, item)
	}
	e = rows.Err()
	rows.Close()
	if e != nil || len(f.items) != 5 {
		t.Fatal("finite nonoverlapping template fixture", e)
	}
	active := f.timedAssessment(time.Minute)
	before := f.exposureSequence("learner_a")
	out, e := f.repo.ReadOwnCorrection(f.ctx, f.Access("learner_a", false), id, true)
	if !errors.Is(e, correction.ErrAnswerOverlap) || len(out.Data.Items) != 0 || f.exposureSequence("learner_a") != before {
		t.Fatal("replacement template overlap", e)
	}
	f.exec(`UPDATE assessment_attempts SET state='abandoned',terminal_at=clock_timestamp() WHERE id=$1`, active)
	out, e = f.repo.ReadOwnCorrection(f.ctx, f.Access("learner_a", false), id, true)
	if e != nil || len(out.Data.Items) != 5 {
		t.Fatal(e)
	}
	for _, version := range []int{1, 2} {
		if f.count(`SELECT count(*) FROM learner_answer_exposures WHERE owner_user_id=$1 AND kind='template' AND id='lf-addition' AND version=$2`, f.ids["learner_a"], version) != 1 {
			t.Fatal("original/replacement template exposure missing", version)
		}
	}
}
func TestCorrectionOwnExposureFailureClosedAndExpired(t *testing.T) {
	f := newCorrectionFixture(t)
	id, _, _ := f.cReadableResult()
	f.timedAssessment(-time.Minute)
	if _, e := f.repo.ReadOwnCorrection(f.ctx, f.Access("learner_a", false), id, true); e != nil {
		t.Fatal("expired active blocked", e)
	}
	before := f.exposureSequence("learner_a")
	f.exec(`CREATE FUNCTION isolated_correction_exposure() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'isolated exposure failure'; END $$`)
	f.exec(`CREATE TRIGGER isolated_correction_exposure BEFORE INSERT ON correction_events FOR EACH ROW WHEN (NEW.kind='detail_exposed') EXECUTE FUNCTION isolated_correction_exposure()`)
	out, e := f.repo.ReadOwnCorrection(f.ctx, f.Access("learner_a", false), id, true)
	if e == nil || out.ActorID != "" || len(out.Data.Items) != 0 || f.exposureSequence("learner_a") != before {
		t.Fatal("body delivered without exposure", e)
	}
	f.exec(`DROP TRIGGER isolated_correction_exposure ON correction_events`)
	f.exec(`UPDATE auth_users SET must_change_password=true WHERE id=$1`, f.ids["learner_a"])
	if _, e = f.repo.ReadOwnCorrection(f.ctx, f.Access("learner_a", false), id, false); !errors.Is(e, auth.ErrPasswordChangeRequired) {
		t.Fatal("current account not rechecked", e)
	}

}

func TestCorrectionBroadCandidateQueryIsExecutable(t *testing.T) {
	f := newCorrectionFixture(t)
	rows, e := f.db.Query(store.CorrectionCandidateSQLForTest(), f.ids["learner_a"], f.knowledge.ID, f.knowledge.Version, *f.KHead(), *f.QHead(), f.blueprint.ID, f.blueprint.Version)
	if e != nil {
		t.Fatal(e)
	}
	defer rows.Close()
	n := 0
	for rows.Next() {
		n++
	}
	if e = rows.Err(); e != nil || n == 0 {
		t.Fatal("compiled bounded metadata query", n, e)
	}
}
