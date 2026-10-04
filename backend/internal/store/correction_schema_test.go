package store_test

import (
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/question"
	"strings"
	"testing"
)

func TestCorrectionSchemaImmutable(t *testing.T) {
	f := newCorrectionFixture(t)
	id := f.cCase()
	for _, q := range []string{`DELETE FROM correction_cases WHERE id=$1`, `UPDATE correction_cases SET cutoff=clock_timestamp() WHERE id=$1`, `UPDATE correction_cases SET sealed=false WHERE id=$1`, `DELETE FROM correction_events WHERE subject_id=$1`, `UPDATE correction_events SET sequence=2 WHERE subject_id=$1`} {
		if _, e := f.db.Exec(q, id); e == nil {
			t.Fatal("correction source/event mutable", q)
		}
	}
	f.exec(`UPDATE correction_cases SET last_backfill_at=clock_timestamp() WHERE id=$1`, id)
	p := f.cDraft(id)
	if e := f.cPending(id, p); e != nil {
		t.Fatal("legal freeze rejected", e)
	}
	if _, e := f.db.Exec(`UPDATE correction_plans SET body=jsonb_set(body,'{reason}','"changed"') WHERE id=$1 AND version=1`, p.ID); e == nil {
		t.Fatal("pending body mutable")
	}
}
func TestCorrectionSchemaDirectApprovalRejected(t *testing.T) {
	f := newCorrectionFixture(t)
	id := f.cCase()
	p := f.cDraft(id)
	if e := f.cPending(id, p); e != nil {
		t.Fatal(e)
	}
	tx, e := f.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	_, e = tx.Exec(`UPDATE correction_plans SET status='approved',sequence=3 WHERE id=$1 AND version=1`, p.ID)
	if e == nil {
		e = tx.Commit()
	}
	if e == nil {
		t.Fatal("approved without independent decision event")
	}
	if e = correctionApprovalFixture(f, id, p, "author_a"); e == nil {
		t.Fatal("creator self-approved")
	}
	if e = correctionApprovalFixture(f, id, p, "reviewer_a"); e != nil {
		t.Fatal("legal independent approval rejected", e)
	}
	if _, e = f.db.Exec(`UPDATE correction_plans SET status='rejected' WHERE id=$1 AND version=1`, p.ID); e == nil {
		t.Fatal("approved decision overwritten")
	}
}
func correctionApprovalFixture(f *correctionFixture, caseID string, p correction.PlanRef, actor string) error {
	tx, e := f.db.Begin()
	if e != nil {
		return e
	}
	defer tx.Rollback()
	var sequence int64
	if e = tx.QueryRow(`SELECT sequence FROM correction_plans WHERE id=$1 AND version=$2`, p.ID, p.Version).Scan(&sequence); e != nil {
		return e
	}
	sequence++
	if e = f.cEvent(tx, "plan", p.ID, "plan_approved", f.ids[actor], caseID, &p.Version, sequence, map[string]any{"decision": "approve", "reason": "Independent mathematical verification."}); e != nil {
		return e
	}
	_, e = tx.Exec(`UPDATE correction_plans SET status='approved',sequence=$3 WHERE id=$1 AND version=$2`, p.ID, p.Version, sequence)
	if e != nil {
		return e
	}
	return tx.Commit()
}
func TestCorrectionSchemaOwnership(t *testing.T) {
	f := newCorrectionFixture(t)
	v, e := f.repo.CreateAssessment(f.ctx, f.Access("learner_a", false), f.assessmentInput(assessment.ModeDiagnostic))
	if e != nil {
		t.Fatal(e)
	}
	caseID := f.cCase()
	id := f.ID()
	f.exec(`INSERT INTO notifications(id,owner_user_id,dedup_key,type,evidence_kind,evidence_id,case_id) VALUES($1,$2,$3,'checking','assessment',$4,$5)`, id, f.ids["learner_a"], "notice:"+id, v.Summary.ID, caseID)
	if _, e := f.db.Exec(`INSERT INTO notification_reads(owner_user_id,notification_id) VALUES($1,$2)`, f.ids["learner_b"], id); e == nil {
		t.Fatal("cross-owner notification read inserted")
	}
	f.exec(`INSERT INTO notification_reads(owner_user_id,notification_id) VALUES($1,$2)`, f.ids["learner_a"], id)
	if _, e := f.db.Exec(`UPDATE notification_reads SET read_at=clock_timestamp() WHERE notification_id=$1`, id); e == nil {
		t.Fatal("first read time mutable")
	}
}
func TestCorrectionSchemaUnapprovedPassRejected(t *testing.T) {
	f := newCorrectionFixture(t)
	caseID := f.cCase()
	p := f.cDraft(caseID)
	tx, e := f.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	_, e = tx.Exec(`INSERT INTO correction_results(id,owner_user_id,case_id,plan_id,plan_version,evidence_kind,evidence_id,status,reason,score,passed,handled_case_ids,basis,basis_bytes,basis_digest,source_key) VALUES($1,$2,$3,$4,1,'assessment',$5,'corrected_passed','answer_corrected',5,true,'[]','{}',convert_to('{}','UTF8'),encode(sha256(convert_to('{}','UTF8')),'hex'),'fake')`, f.ID(), f.ids["learner_a"], caseID, p.ID, f.ID())
	if e == nil {
		e = tx.Commit()
	}
	if e == nil {
		t.Fatal("unapproved/foreign/incomplete five-item pass accepted")
	}
}

func TestCorrectionSchemaLegalResultAndAudit(t *testing.T) {
	f := newCorrectionFixture(t)
	attempt, b := f.cSubmittedBasis("learner_a")
	caseID := f.cCase()
	p := f.cDraft(caseID)
	if e := f.cPending(caseID, p); e != nil {
		t.Fatal(e)
	}
	if e := correctionApprovalFixture(f, caseID, p, "reviewer_a"); e != nil {
		t.Fatal(e)
	}
	b.HandledCaseIDs = []string{caseID}
	b.PlanRefs = []correction.PlanRef{p}
	verdict, err := correction.Evaluate(b)
	if err != nil || verdict.Status != correction.CorrectedPassed || verdict.Score == nil || *verdict.Score != 5 {
		t.Fatalf("real five-item fixture is not a corrected pass: %+v %v", verdict, err)
	}
	if _, e := f.cResult(attempt, f.ids["learner_a"], caseID, p, b, false); e == nil {
		t.Fatal("result sealed without audit event")
	}
	id, e := f.cResult(attempt, f.ids["learner_a"], caseID, p, b, true)
	if e != nil {
		t.Fatal("legal original-answer result rejected", e)
	}
	for _, q := range []string{`UPDATE correction_results SET score=0 WHERE id=$1`, `DELETE FROM correction_results WHERE id=$1`, `DELETE FROM correction_dependencies WHERE result_id=$1`, `UPDATE assessment_answers SET answer='{"kind":"skipped"}' WHERE attempt_id=$1`} {
		target := id
		if strings.Contains(q, "assessment_answers") {
			target = attempt
		}
		if _, e = f.db.Exec(q, target); e == nil {
			t.Fatal("immutable fact changed", q)
		}
	}
	if _, e = f.cResult(attempt, f.ids["learner_b"], caseID, p, b, true); e == nil {
		t.Fatal("foreign owner basis sealed")
	}
}
func TestCorrectionSchemaReplacementSHA(t *testing.T) {
	f := newCorrectionFixture(t)
	_, b := f.cSubmittedBasis("learner_a")
	caseID := f.cCase()
	p := f.cDraft(caseID)
	original := b.OriginalItems[0]
	replacement := original
	replacement.Identity.SHA256 = strings.Repeat("a", 64)
	input := correction.PlanInput{AlgorithmVersion: 1, Mappings: []correction.Mapping{{Original: original.Identity, OriginalPublicationID: *f.QHead(), Replacement: correction.PublishedInstance{Identity: replacement.Identity, PublicationID: *f.QHead()}}}, Reason: "Original exact source proof test."}
	proof := correction.PlanProof{CaseID: caseID, AlgorithmVersion: 1, Mappings: []correction.ResolvedMapping{{Original: original, Replacement: replacement, OriginalPublicationID: *f.QHead(), ReplacementPublicationID: *f.QHead(), OriginalApproval: b.OriginalSeal.Items[0].Approval, ReplacementApproval: b.OriginalSeal.Items[0].Approval}}, Authors: []string{}, ContentApprovalIDs: []string{}, QuestionApprovalIDs: []string{}}
	tx, e := f.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	if e = f.cEvent(tx, "plan", p.ID, "plan_updated", f.ids["author_a"], caseID, &p.Version, 2, input); e != nil {
		t.Fatal(e)
	}
	if _, e = tx.Exec(`UPDATE correction_plans SET body=$2,sequence=2 WHERE id=$1 AND version=1`, p.ID, corrJSON(input)); e != nil {
		t.Fatal(e)
	}
	raw, h, e := correction.Canonical("correction-plan-v1", struct {
		Input correction.PlanInput `json:"input"`
		Proof correction.PlanProof `json:"proof"`
	}{input, proof})
	if e != nil {
		t.Fatal(e)
	}
	if e = f.cEvent(tx, "plan", p.ID, "plan_submitted", f.ids["author_a"], caseID, &p.Version, 3, map[string]any{"digest": h}); e != nil {
		t.Fatal(e)
	}
	_, e = tx.Exec(`UPDATE correction_plans SET status='pending',sequence=3,sealed=true,frozen_body=$2,frozen_bytes=$3,frozen_digest=$4 WHERE id=$1 AND version=1`, p.ID, string(raw), raw, h)
	if e == nil {
		e = tx.Commit()
	}
	if e == nil {
		t.Fatal("fake replacement SHA passed accurate publication proof")
	}
}

func TestCorrectionSchemaPassingKnowledgeRequired(t *testing.T) {
	f := newCorrectionFixture(t)
	attempt, b := f.cSubmittedBasis("learner_a")
	caseID := f.cCase()
	p := f.cDraft(caseID)
	if e := f.cPending(caseID, p); e != nil {
		t.Fatal(e)
	}
	if e := correctionApprovalFixture(f, caseID, p, "reviewer_a"); e != nil {
		t.Fatal(e)
	}
	b.HandledCaseIDs = []string{caseID}
	b.PlanRefs = []correction.PlanRef{p}
	if _, e := f.cResult(attempt, f.ids["learner_a"], caseID, p, b, true, true); e == nil {
		t.Fatal("passing result detached from original knowledge identity")
	}
}

func TestCorrectionSchemaUnmappedReplacementRejected(t *testing.T) {
	f := newCorrectionFixture(t)
	attempt, b := f.cSubmittedBasis("learner_a")
	f.questionInput.QuestionPackage.Version = 2
	f.questionInput.QuestionPackage.Templates[0].Version = 2
	f.questionInput.QuestionPackage.Templates[0].ExplanationTemplate = "Original independently reviewed revised explanation."
	f.questionInput.QuestionPackage.Blueprints[0].Version = 2
	f.questionInput.QuestionPackage.Blueprints[0].Sources[0].Ref.Version = 2
	sub := f.QApproved("author_b", "reviewer_b")
	f.QActivate(f.QPrepare(sub.ID))
	var raw []byte
	parameters := corrJSON(b.OriginalItems[0].Parameters)
	if e := f.db.QueryRow(`SELECT jsonb_set(i.body->'body','{identity,sha256}',to_jsonb(i.sha256)) FROM question_instances i JOIN question_publication_members m ON m.kind='instance' AND m.id=i.id AND m.version=i.version AND m.sha256=i.sha256 WHERE m.publication_id=$1 AND i.template_version=2 AND i.body#>'{body,parameters}'=$2::jsonb LIMIT 1`, *f.QHead(), parameters).Scan(&raw); e != nil {
		t.Fatal(e)
	}
	var replacement question.Instance
	if e := json.Unmarshal(raw, &replacement); e != nil {
		t.Fatal(e)
	}
	if ok, _, e := correction.Equivalent(b.OriginalItems[0], replacement); e != nil || !ok {
		t.Fatal("real published replacement was not equivalent", e)
	}
	caseID := f.cCase()
	p := f.cDraft(caseID)
	if e := f.cPending(caseID, p); e != nil {
		t.Fatal(e)
	}
	if e := correctionApprovalFixture(f, caseID, p, "reviewer_a"); e != nil {
		t.Fatal(e)
	}
	old := b.OriginalItems[0].Identity
	b.EffectiveItems[0] = replacement
	deps := []correction.Dependency{}
	for _, d := range b.EffectiveDeps {
		if d.Kind == "instance" && d.ID == old.ID {
			continue
		}
		deps = append(deps, d)
	}
	v := replacement.Identity.Version
	tv := replacement.Template.Version
	fresh := []correction.Dependency{{Kind: "instance", ID: replacement.Identity.ID, Version: &v, SHA256: replacement.Identity.SHA256}, {Kind: "template", ID: replacement.Template.ID, Version: &tv, SHA256: replacement.Template.SHA256}}
	b.EffectiveDeps = append(deps, fresh...)
	b.AuditDeps = append(b.AuditDeps, fresh...)
	b.HandledCaseIDs = []string{caseID}
	b.PlanRefs = []correction.PlanRef{p}
	if _, e := f.cResult(attempt, f.ids["learner_a"], caseID, p, b, true); e == nil {
		t.Fatal("equivalent but unmapped published replacement granted a result")
	}
	mapped := f.cDraft(caseID)
	var approval question.MemberEvidence
	if e := f.db.QueryRow(`SELECT evidence FROM question_publication_members WHERE publication_id=$1 AND kind='instance' AND id=$2`, *f.QHead(), replacement.Identity.ID).Scan(&raw); e != nil {
		t.Fatal(e)
	}
	if e := json.Unmarshal(raw, &approval); e != nil {
		t.Fatal(e)
	}
	input := correction.PlanInput{AlgorithmVersion: 1, Mappings: []correction.Mapping{{Original: old, OriginalPublicationID: b.OriginalSeal.QuestionPublicationID, Replacement: correction.PublishedInstance{Identity: replacement.Identity, PublicationID: *f.QHead()}}}, Reason: "Approve this exact original-to-replacement source."}
	proof := correction.PlanProof{CaseID: caseID, AlgorithmVersion: 1, Mappings: []correction.ResolvedMapping{{Original: b.OriginalItems[0], Replacement: replacement, OriginalPublicationID: b.OriginalSeal.QuestionPublicationID, ReplacementPublicationID: *f.QHead(), OriginalApproval: b.OriginalSeal.Items[0].Approval, ReplacementApproval: approval}}, Authors: []string{f.ids["author_a"], f.ids["author_b"]}, ContentApprovalIDs: []string{}, QuestionApprovalIDs: []string{b.OriginalSeal.Items[0].Approval.DecisionID, approval.DecisionID}}
	tx, e := f.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	if e = f.cEvent(tx, "plan", mapped.ID, "plan_updated", f.ids["author_a"], caseID, &mapped.Version, 2, input); e != nil {
		t.Fatal(e)
	}
	if _, e = tx.Exec(`UPDATE correction_plans SET body=$2,sequence=2 WHERE id=$1 AND version=1`, mapped.ID, corrJSON(input)); e != nil {
		t.Fatal(e)
	}
	frozen, h, e := correction.Canonical("correction-plan-v1", struct {
		Input correction.PlanInput `json:"input"`
		Proof correction.PlanProof `json:"proof"`
	}{input, proof})
	if e != nil {
		t.Fatal(e)
	}
	if e = f.cEvent(tx, "plan", mapped.ID, "plan_submitted", f.ids["author_a"], caseID, &mapped.Version, 3, map[string]any{"digest": h}); e != nil {
		t.Fatal(e)
	}
	if _, e = tx.Exec(`UPDATE correction_plans SET status='pending',sequence=3,sealed=true,frozen_body=$2,frozen_bytes=$3,frozen_digest=$4 WHERE id=$1 AND version=1`, mapped.ID, string(frozen), frozen, h); e != nil {
		t.Fatal(e)
	}
	if e = tx.Commit(); e != nil {
		t.Fatal("legal exact mapped submission rejected", e)
	}
	if e = correctionApprovalFixture(f, caseID, mapped, "reviewer_a"); e != nil {
		t.Fatal("independent exact mapping rejected", e)
	}
	b.PlanRefs = []correction.PlanRef{mapped}
	verdict, e := correction.Evaluate(b)
	if e != nil || verdict.Score == nil || *verdict.Score != 5 {
		t.Fatalf("real equivalent mapping not five correct: %+v %v", verdict, e)
	}
	if _, e = f.cResult(attempt, f.ids["learner_a"], caseID, mapped, b, true); e != nil {
		t.Fatal("approved exact replacement result rejected", e)
	}

}
