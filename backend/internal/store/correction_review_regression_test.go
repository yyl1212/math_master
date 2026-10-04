package store_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/question"
	"testing"
	"time"
)

func TestCorrectionUnprocessedApprovedChain(t *testing.T) {
	f := newCorrectionFixture(t)
	aid, b := f.cSubmittedBasis("learner_a")
	oldPub := *f.QHead()
	w := f.QWithdraw(question.WithdrawalTarget{Kind: "template", ID: b.EffectiveItems[0].Template.ID, Version: 1})
	f.cFinishRoots()
	var cid string
	if e := f.db.QueryRow(`SELECT id::text FROM correction_cases WHERE withdrawal_id=$1`, w.EventID).Scan(&cid); e != nil {
		t.Fatal(e)
	}
	p := f.cReplacementPlan(cid, b, oldPub, 2)
	var raw []byte
	if e := f.db.QueryRow(`SELECT frozen_body#>'{body,proof,mappings}' FROM correction_plans WHERE id=$1 AND version=$2`, p.ID, p.Version).Scan(&raw); e != nil {
		t.Fatal(e)
	}
	var mappings []correction.ResolvedMapping
	if e := json.Unmarshal(raw, &mappings); e != nil {
		t.Fatal(e)
	}
	for i, m := range mappings {
		b.EffectiveItems[i] = m.Replacement
	}
	p2 := f.cReplacementPlan(cid, b, *f.QHead(), 3, p)
	if e := f.db.QueryRow(`SELECT frozen_body#>'{body,proof,mappings}' FROM correction_plans WHERE id=$1 AND version=$2`, p2.ID, p2.Version).Scan(&raw); e != nil {
		t.Fatal(e)
	}
	if e := json.Unmarshal(raw, &mappings); e != nil {
		t.Fatal(e)
	}
	for i, m := range mappings {
		b.EffectiveItems[i] = m.Replacement
	}
	b.PlanRefs = []correction.PlanRef{p, p2}
	b.HandledCaseIDs = []string{cid}
	var authorized bool
	if e := f.db.QueryRow(`SELECT correction_replacement_authorized($1::jsonb,0)`, corrJSON(b)).Scan(&authorized); e != nil {
		t.Fatal(e)
	}
	if !authorized {
		t.Fatal("approved A-to-B-to-C chain rejected before any parent result exists")
	}
	if f.count(`SELECT count(*) FROM correction_results WHERE status='corrected_passed'`) != 0 {
		t.Fatal("intermediate correction already exists")
	}
	f.cRunAll()
	if f.count(`SELECT count(*) FROM correction_results WHERE evidence_id=$1 AND status='corrected_passed'`, aid) != 2 {
		t.Fatal("both approved sources were not processed")
	}
	// Two different approved source jobs append one audit per sealed result;
	// draining the same jobs again must not create a duplicate result or grant.
	if f.count(`SELECT count(*) FROM correction_events WHERE kind='qualification_granted'`) != 2 || f.count(`SELECT count(DISTINCT result_id) FROM correction_events WHERE kind='qualification_granted'`) != 2 {
		t.Fatal("per-result qualification audit duplicated or missing")
	}
	f.cRunAll()
	if f.count(`SELECT count(*) FROM correction_events WHERE kind='qualification_granted'`) != 2 {
		t.Fatal("repeated drain duplicated qualification")
	}
	if f.count(`SELECT count(*) FROM assessment_results WHERE attempt_id=$1 AND score=5`, aid) != 1 {
		t.Fatal("original result changed")
	}

	for _, refs := range [][]correction.PlanRef{{p2}, {p2, {ID: "22222222-2222-4222-8222-222222222222", Version: 1}}} {
		b.PlanRefs = refs
		if e := f.db.QueryRow(`SELECT correction_replacement_authorized($1::jsonb,0)`, corrJSON(b)).Scan(&authorized); e != nil {
			t.Fatal(e)
		}
		if authorized {
			t.Fatal("chain accepted without its actual first approved edge")
		}
	}
	b.PlanRefs = make([]correction.PlanRef, 101)
	for i := range b.PlanRefs {
		b.PlanRefs[i] = p
	}
	if e := f.db.QueryRow(`SELECT correction_replacement_authorized($1::jsonb,0)`, corrJSON(b)).Scan(&authorized); e != nil {
		t.Fatal(e)
	}
	if authorized {
		t.Fatal("unbounded approved chain accepted")
	}

}

func TestCorrectionResultAssetsUseEffectiveBasis(t *testing.T) {
	f := newCorrectionFixture(t)
	aid, b := f.cSubmittedBasis("learner_a")
	oldPub := *f.QHead()
	w := f.QWithdraw(question.WithdrawalTarget{Kind: "template", ID: b.EffectiveItems[0].Template.ID, Version: 1})
	f.cFinishRoots()
	var cid string
	if e := f.db.QueryRow(`SELECT id::text FROM correction_cases WHERE withdrawal_id=$1`, w.EventID).Scan(&cid); e != nil {
		t.Fatal(e)
	}
	f.cReplacementPlan(cid, b, oldPub, 2)
	f.cRunAll()
	var rid string
	if e := f.db.QueryRow(`SELECT id::text FROM correction_results WHERE evidence_id=$1 AND status='corrected_passed'`, aid).Scan(&rid); e != nil {
		t.Fatal(e)
	}
	detail, e := f.repo.ReadOwnCorrection(f.ctx, f.Access("learner_a", false), rid, true)
	if e != nil || detail.Data.Result.Validity != "effective" || len(detail.Data.Items[0].Assets) == 0 {
		t.Fatal("real corrected detail", detail, e)
	}
	digest := detail.Data.Items[0].Assets[0].SHA256
	if _, e = f.repo.ReadLearningAsset(f.ctx, f.Access("learner_a", false), aid, digest); !errors.Is(e, auth.ErrNotFound) {
		t.Fatal("old withdrawn attempt asset protection changed", e)
	}
	reader, ok := any(f.repo).(interface {
		ReadOwnCorrectionAsset(context.Context, question.Access, string, string) ([]byte, error)
	})
	if !ok {
		t.Fatal("private corrected asset read is missing")
	}
	data, e := reader.ReadOwnCorrectionAsset(f.ctx, f.Access("learner_a", false), rid, digest)
	if e != nil || len(data) == 0 || fmtDigest(data) != digest {
		t.Fatal("effective corrected asset unavailable", e)
	}
	original := append([]byte{}, data...)
	if _, e = reader.ReadOwnCorrectionAsset(f.ctx, f.Access("learner_b", false), rid, digest); !errors.Is(e, auth.ErrNotFound) {
		t.Fatal("other owner read corrected image", e)
	}
	if _, e = reader.ReadOwnCorrectionAsset(f.ctx, f.Access("learner_a", false), rid, wrongCorrectionSHA(digest)); !errors.Is(e, auth.ErrNotFound) {
		t.Fatal("unreferenced SHA accepted", e)
	}
	f.QWithdraw(question.WithdrawalTarget{Kind: "template", ID: b.EffectiveItems[0].Template.ID, Version: 2})
	if _, e = reader.ReadOwnCorrectionAsset(f.ctx, f.Access("learner_a", false), rid, digest); !errors.Is(e, auth.ErrNotFound) {
		t.Fatal("withdrawn replacement image still exposed", e)
	}
	var persisted []byte
	if e := f.db.QueryRow(`SELECT bytes FROM assets WHERE sha256=$1`, digest).Scan(&persisted); e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(original, persisted) {
		t.Fatal("original image bytes changed")
	}
}
func fmtDigest(raw []byte) string {
	h := sha256.Sum256(raw)
	const chars = "0123456789abcdef"
	out := make([]byte, 64)
	for i, v := range h {
		out[i*2] = chars[v>>4]
		out[i*2+1] = chars[v&15]
	}
	return string(out)
}

func wrongCorrectionSHA(s string) string {
	if s[0] == 'a' {
		return "b" + s[1:]
	}
	return "a" + s[1:]
}
func TestCorrectionReplacementChainRejectsConflictsAndCycles(t *testing.T) {
	for _, kind := range []string{"conflict", "cycle"} {
		t.Run(kind, func(t *testing.T) {
			f := newCorrectionFixture(t)
			_, base := f.cSubmittedBasis("learner_a")
			oldPub := *f.QHead()
			w := f.QWithdraw(question.WithdrawalTarget{Kind: "template", ID: base.OriginalItems[0].Template.ID, Version: 1})
			f.cFinishRoots()
			var cid string
			if e := f.db.QueryRow(`SELECT id::text FROM correction_cases WHERE withdrawal_id=$1`, w.EventID).Scan(&cid); e != nil {
				t.Fatal(e)
			}
			p1 := f.cReplacementPlan(cid, base, oldPub, 2)
			pub2 := *f.QHead()
			b := base
			b.EffectiveItems = append([]question.Instance{}, base.EffectiveItems...)
			readMapping := func(ref correction.PlanRef) {
				var raw []byte
				var mappings []correction.ResolvedMapping
				if e := f.db.QueryRow(`SELECT frozen_body#>'{body,proof,mappings}' FROM correction_plans WHERE id=$1 AND version=$2`, ref.ID, ref.Version).Scan(&raw); e != nil {
					t.Fatal(e)
				}
				if json.Unmarshal(raw, &mappings) != nil {
					t.Fatal("frozen mapping")
				}
				for i, m := range mappings {
					b.EffectiveItems[i] = m.Replacement
				}
			}
			readMapping(p1)
			middle := append([]question.Instance{}, b.EffectiveItems...)
			p2 := f.cReplacementPlan(cid, b, pub2, 3, p1)
			pub3 := *f.QHead()
			readMapping(p2)
			var extra correction.PlanRef
			if kind == "conflict" {
				extra = f.cReplacementPlan(cid, base, oldPub, 4)
			} else {
				in := correctionPlanInput()
				in.Parent = &p2
				for i, current := range b.EffectiveItems {
					in.Mappings = append(in.Mappings, correction.Mapping{Original: current.Identity, OriginalPublicationID: pub3, Replacement: correction.PublishedInstance{Identity: middle[i].Identity, PublicationID: pub2}})
				}
				created, e := f.repo.CreateCorrectionPlan(f.ctx, f.Access("admin_a", false), cid, in)
				if e != nil {
					t.Fatal(e)
				}
				plan := f.submitCorrectionPlan(*created.Data.Plan, "admin_a")
				v, e := f.repo.DecideCorrectionPlan(f.ctx, f.Access("reviewer_b", false), plan.Ref, correction.DecisionInput{ExpectedSequence: plan.Sequence, Decision: "approve", Reason: "Independent exact mapping regression."})
				if e != nil {
					t.Fatal(e)
				}
				extra = v.Data.Plan.Ref
			}
			b.PlanRefs = []correction.PlanRef{p1, p2, extra}
			b.HandledCaseIDs = []string{cid}
			var allowed bool
			if e := f.db.QueryRow(`SELECT correction_replacement_authorized($1::jsonb,0)`, corrJSON(b)).Scan(&allowed); e != nil {
				t.Fatal(e)
			}
			if allowed {
				t.Fatal("ambiguous or cyclic actual approved chain accepted", kind)
			}
		})
	}
}
func TestCorrectionAssetExposureAndCooldown(t *testing.T) {
	f := newCorrectionFixture(t)
	id, _, _ := f.cReadableResult()
	active := f.timedAssessment(time.Minute)
	var digest string
	if e := f.db.QueryRow(`SELECT basis#>>'{body,effectiveItems,0,body,assets,0,sha256}' FROM correction_results WHERE id=$1`, id).Scan(&digest); e != nil {
		t.Fatal(e)
	}
	before := f.exposureSequence("learner_a")
	data, e := f.repo.ReadOwnCorrectionAsset(f.ctx, f.Access("learner_a", false), id, digest)
	if !errors.Is(e, correction.ErrAnswerOverlap) || len(data) != 0 || f.exposureSequence("learner_a") != before {
		t.Fatal("asset bypassed active-answer overlap", e)
	}
	f.exec(`UPDATE assessment_attempts SET state='abandoned',terminal_at=clock_timestamp() WHERE id=$1`, active)
	if data, e = f.repo.ReadOwnCorrectionAsset(f.ctx, f.Access("learner_a", false), id, digest); e != nil || len(data) == 0 {
		t.Fatal("safe owned image read", e)
	}
	_, e = f.repo.CreateAssessment(f.ctx, f.Access("learner_a", false), f.assessmentInput(assessment.ModeDiagnostic))
	var ready *learning.NotReadyError
	if !errors.As(e, &ready) {
		t.Fatal("asset bypassed 30-minute exposure cooldown", e)
	}
}

func TestCorrectionReplacementChainIgnoresUnrelatedFork(t *testing.T) {
	f := newCorrectionFixture(t)
	aid, b := f.cSubmittedBasis("learner_a")
	pub1 := *f.QHead()
	var raw []byte
	var unrelated question.Instance
	if e := f.db.QueryRow(`SELECT jsonb_set(i.body->'body','{identity,sha256}',to_jsonb(i.sha256)) FROM question_instances i WHERE i.template_version=1 AND i.knowledge_id=$1 AND NOT EXISTS(SELECT 1 FROM assessment_items a WHERE a.attempt_id=$2 AND a.instance_id=i.id) ORDER BY i.id LIMIT 1`, b.OriginalSeal.Knowledge.ID, aid).Scan(&raw); e != nil {
		t.Fatal(e)
	}
	if json.Unmarshal(raw, &unrelated) != nil {
		t.Fatal("actual unrelated instance")
	}
	w := f.QWithdraw(question.WithdrawalTarget{Kind: "template", ID: b.OriginalItems[0].Template.ID, Version: 1})
	f.cFinishRoots()
	var cid string
	if e := f.db.QueryRow(`SELECT id::text FROM correction_cases WHERE withdrawal_id=$1`, w.EventID).Scan(&cid); e != nil {
		t.Fatal(e)
	}
	apply := func(ref correction.PlanRef) {
		var data []byte
		var maps []correction.ResolvedMapping
		if e := f.db.QueryRow(`SELECT frozen_body#>'{body,proof,mappings}' FROM correction_plans WHERE id=$1 AND version=$2`, ref.ID, ref.Version).Scan(&data); e != nil {
			t.Fatal(e)
		}
		if json.Unmarshal(data, &maps) != nil {
			t.Fatal("actual approved mapping")
		}
		for i, m := range maps {
			b.EffectiveItems[i] = m.Replacement
		}
	}
	p1 := f.cReplacementPlan(cid, b, pub1, 2)
	pub2 := *f.QHead()
	apply(p1)
	p2 := f.cReplacementPlan(cid, b, pub2, 3, p1)
	apply(p2)
	other := b
	other.EffectiveItems = []question.Instance{unrelated}
	p3 := f.cReplacementPlan(cid, other, pub1, 4)
	p4 := f.cReplacementPlan(cid, other, pub1, 5)
	b.PlanRefs = []correction.PlanRef{p1, p2, p3, p4}
	b.HandledCaseIDs = []string{cid}
	var allowed bool
	if e := f.db.QueryRow(`SELECT correction_replacement_authorized($1::jsonb,0)`, corrJSON(b)).Scan(&allowed); e != nil {
		t.Fatal(e)
	}
	if !allowed {
		t.Fatal("another original instance's approved fork blocked this exact deterministic chain")
	}
	f.cRunAll()
	if f.count(`SELECT count(*) FROM correction_results WHERE evidence_id=$1 AND status='corrected_passed'`, aid) != 4 {
		t.Fatal("unrelated mappings stalled valid evidence")
	}
}
