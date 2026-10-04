package store_test

import (
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"github.com/yyl1212/math_master/backend/internal/question"
	"strings"
	"testing"
)

func TestCorrectionSourceProofExactFrozenInstances(t *testing.T) {
	f := newCorrectionFixture(t)
	f.publishLearningGraph()
	v := f.createDiagnostic("learner_a")
	c := f.registerRule(&f.knowledge, 1)
	original := v.Questions[0].Instance
	head := *f.QHead()
	in := correctionPlanInput()
	in.Mappings = []correction.Mapping{{Original: original, OriginalPublicationID: head, Replacement: correction.PublishedInstance{Identity: original, PublicationID: head}}}
	created, e := f.repo.CreateCorrectionPlan(f.ctx, f.Access("author_b", false), c.ID, in)
	if e != nil {
		t.Fatal(e)
	}
	p := f.submitCorrectionPlan(*created.Data.Plan, "author_b")
	var frozen []byte
	if e = f.db.QueryRow(`SELECT frozen_body FROM correction_plans WHERE id=$1 AND version=1`, p.Ref.ID).Scan(&frozen); e != nil {
		t.Fatal(e)
	}
	var env struct {
		Body struct {
			Input correction.PlanInput `json:"input"`
			Proof correction.PlanProof `json:"proof"`
		} `json:"body"`
	}
	if e = json.Unmarshal(frozen, &env); e != nil {
		t.Fatal(e)
	}
	proof := env.Body.Proof
	if len(proof.Mappings) != 1 || proof.Mappings[0].Original.Identity != original || proof.Mappings[0].Replacement.Identity != original || len(proof.Mappings[0].Original.Parameters) == 0 || len(proof.ContentApprovalIDs) == 0 || len(proof.QuestionApprovalIDs) == 0 {
		t.Fatal("frozen source omitted runtime proof", proof)
	}
	if !strings.Contains(correctionReceiptJSON(t, proof.Authors), f.ids["author_a"]) {
		t.Fatal("source author omitted")
	}
	if _, e = f.repo.DecideCorrectionPlan(f.ctx, f.Access("reviewer_b", false), p.Ref, correction.DecisionInput{ExpectedSequence: p.Sequence, Decision: "approve", Reason: "Actual immutable published source and rational grading reviewed independently."}); e != nil {
		t.Fatal(e)
	}
	for _, mode := range []string{"sha", "publication", "knowledge"} {
		t.Run(mode, func(t *testing.T) {
			bad := in
			bad.Mappings = append([]correction.Mapping{}, in.Mappings...)
			switch mode {
			case "sha":
				bad.Mappings[0].Replacement.Identity.SHA256 = strings.Repeat("e", 64)
			case "publication":
				bad.Mappings[0].Replacement.PublicationID = f.ID()
			case "knowledge":
				k := f.knowledge
				k.ID = "workflow-dependent"
				if e := f.db.QueryRow(`SELECT sha256 FROM knowledge_versions WHERE id=$1 AND version=1`, k.ID).Scan(&k.SHA256); e == nil {
					other := f.registerRule(&k, 1)
					if _, e := f.repo.CreateCorrectionPlan(f.ctx, f.Access("author_b", false), other.ID, bad); !errors.Is(e, correction.ErrSourceStale) {
						t.Fatal("mapping outside knowledge scope", e)
					}
					return
				} else {
					t.Fatal("missing scope fixture")
				}
			}
			if _, e := f.repo.CreateCorrectionPlan(f.ctx, f.Access("author_b", false), c.ID, bad); !errors.Is(e, correction.ErrSourceStale) {
				t.Fatal("invented source accepted", e)
			}
		})
	}

}
func TestCorrectionIndependentRoleRevocationDuringPlanWait(t *testing.T) {
	f := newCorrectionFixture(t)
	c := f.registerRule(nil, 1)
	p := f.submitCorrectionPlan(f.createCorrectionPlan(c.ID, "author_b"), "author_b")
	block, e := f.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer block.Rollback()
	var id string
	if e = block.QueryRow(`SELECT id::text FROM auth_users WHERE id=$1 FOR UPDATE`, f.ids["reviewer_b"]).Scan(&id); e != nil {
		t.Fatal(e)
	}
	done := make(chan error, 1)
	key := f.Access("reviewer_b", false)
	go func() {
		_, e := f.repo.DecideCorrectionPlan(f.ctx, key, p.Ref, correction.DecisionInput{ExpectedSequence: p.Sequence, Decision: "approve", Reason: "Independent current reviewer role is required at commit."})
		done <- e
	}()
	waitCorrectionSQL(t, f, "%FROM auth_users%FOR UPDATE%")
	if _, e = block.Exec(`DELETE FROM auth_user_roles WHERE user_id=$1 AND role='reviewer'`, id); e != nil {
		t.Fatal(e)
	}
	if e = block.Commit(); e != nil {
		t.Fatal(e)
	}
	if e = <-done; !errors.Is(e, auth.ErrForbidden) {
		t.Fatal("revoked reviewer committed", e)
	}
	if f.count(`SELECT count(*) FROM correction_events WHERE kind='plan_approved'`) != 0 {
		t.Fatal("revoked review persisted")
	}
}

func TestCorrectionSourceProofReplacementKeepsActualParameters(t *testing.T) {
	f := newCorrectionFixture(t)
	_, basis := f.cSubmittedBasis("learner_a")
	original := basis.OriginalItems[0]
	oldHead := *f.QHead()
	f.questionInput.QuestionPackage.Version = 2
	f.questionInput.QuestionPackage.Templates[0].Version = 2
	f.questionInput.QuestionPackage.Templates[0].ExplanationTemplate = "An original independently reviewed replacement explanation."
	f.questionInput.QuestionPackage.Blueprints[0].Version = 2
	f.questionInput.QuestionPackage.Blueprints[0].Sources[0].Ref.Version = 2
	sub := f.QApproved("author_b", "reviewer_a")
	f.QActivate(f.QPrepare(sub.ID))
	var replacement question.Instance
	var raw []byte
	if e := f.db.QueryRow(`SELECT jsonb_set(i.body->'body','{identity,sha256}',to_jsonb(i.sha256)) FROM question_instances i JOIN question_publication_members m ON m.kind='instance' AND m.id=i.id AND m.version=i.version AND m.sha256=i.sha256 WHERE m.publication_id=$1 AND i.template_version=2 AND i.body#>'{body,parameters}'=$2::jsonb LIMIT 1`, *f.QHead(), corrJSON(original.Parameters)).Scan(&raw); e != nil {
		t.Fatal(e)
	}
	if e := json.Unmarshal(raw, &replacement); e != nil {
		t.Fatal(e)
	}
	c := f.registerRule(nil, 1)
	in := correctionPlanInput()
	in.Mappings = []correction.Mapping{{Original: original.Identity, OriginalPublicationID: oldHead, Replacement: correction.PublishedInstance{Identity: replacement.Identity, PublicationID: *f.QHead()}}}
	created, e := f.repo.CreateCorrectionPlan(f.ctx, f.Access("admin_a", false), c.ID, in)
	if e != nil {
		t.Fatal(e)
	}
	p := f.submitCorrectionPlan(*created.Data.Plan, "admin_a")
	if e = f.db.QueryRow(`SELECT frozen_body#>'{body,proof}' FROM correction_plans WHERE id=$1 AND version=1`, p.Ref.ID).Scan(&raw); e != nil {
		t.Fatal(e)
	}
	var proof correction.PlanProof
	if e = json.Unmarshal(raw, &proof); e != nil {
		t.Fatal(e)
	}
	m := proof.Mappings[0]
	if m.Original.Identity == m.Replacement.Identity || m.OriginalPublicationID != oldHead || m.ReplacementPublicationID != *f.QHead() || corrJSON(m.Original.Parameters) != corrJSON(m.Replacement.Parameters) || m.Original.Template.Version != 1 || m.Replacement.Template.Version != 2 {
		t.Fatal("replacement provenance or runtime parameters lost", m)
	}
	f.exec(`INSERT INTO auth_user_roles(user_id,role) VALUES($1,'reviewer') ON CONFLICT DO NOTHING`, f.ids["author_b"])
	decision := correction.DecisionInput{ExpectedSequence: p.Sequence, Decision: "approve", Reason: "The exact original and replacement sources have been independently reviewed."}
	if _, e = f.repo.DecideCorrectionPlan(f.ctx, f.Access("author_b", false), p.Ref, decision); !errors.Is(e, auth.ErrForbidden) {
		t.Fatal("replacement author self reviewed", e)
	}
	if _, e = f.repo.DecideCorrectionPlan(f.ctx, f.Access("reviewer_b", false), p.Ref, decision); e != nil {
		t.Fatal("real replacement approval rejected", e)
	}
}
func TestCorrectionIndependentFeedbackResolvedIsNotApproval(t *testing.T) {
	f := newCorrectionFixture(t)
	c := f.registerRule(nil, 1)
	p := f.submitCorrectionPlan(f.createCorrectionPlan(c.ID, "author_b"), "author_b")
	ff := &feedbackFixture{f.learningFixture}
	ticket := ff.feedbackCreate("learner_a", ff.feedbackInput())
	ticket = ff.feedbackTransition("reviewer_b", ticket, feedback.Processing, nil)
	ff.feedbackTransition("reviewer_b", ticket, feedback.Resolved, &feedback.Resolution{Kind: "clarified"})
	if f.count(`SELECT count(*) FROM correction_plans WHERE id=$1 AND status='approved'`, p.Ref.ID) != 0 || f.count(`SELECT count(*) FROM correction_jobs WHERE type='approved_plan'`) != 0 {
		t.Fatal("feedback resolution forged correction approval")
	}
	if _, e := f.repo.CreateAssessment(f.ctx, f.Access("learner_b", false), f.assessmentInput("diagnostic")); e == nil {
		t.Fatal("resolved feedback opened grading")
	}
}
func TestCorrectionIndependentRevokeAtCommitAfterCaseWait(t *testing.T) {
	f := newCorrectionFixture(t)
	c := f.registerRule(nil, 1)
	p := f.submitCorrectionPlan(f.createCorrectionPlan(c.ID, "author_b"), "author_b")
	block, e := f.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer block.Rollback()
	var id string
	if e = block.QueryRow(`SELECT id::text FROM correction_cases WHERE id=$1 FOR UPDATE`, c.ID).Scan(&id); e != nil {
		t.Fatal(e)
	}
	done := make(chan error, 1)
	key := f.Access("reviewer_b", false)
	go func() {
		_, e := f.repo.DecideCorrectionPlan(f.ctx, key, p.Ref, correction.DecisionInput{ExpectedSequence: p.Sequence, Decision: "approve", Reason: "Fresh authorization is required after resource waits."})
		done <- e
	}()
	waitCorrectionSQL(t, f, "%FROM correction_cases WHERE id=%")
	if _, e = block.Exec(`DELETE FROM auth_user_roles WHERE user_id=$1 AND role='reviewer'`, f.ids["reviewer_b"]); e != nil {
		t.Fatal(e)
	}
	if e = block.Commit(); e != nil {
		t.Fatal(e)
	}
	if e = <-done; !errors.Is(e, auth.ErrForbidden) {
		t.Fatal("revoked after initial auth committed", e)
	}
	if f.count(`SELECT count(*) FROM correction_events WHERE kind='plan_approved'`) != 0 || f.count(`SELECT count(*) FROM correction_idempotency WHERE action='decidePlan'`) != 0 {
		t.Fatal("commit revalidation failure left facts")
	}
}

func TestCorrectionIndependentBoundContentAuthorWithoutMapping(t *testing.T) {
	f := newCorrectionFixture(t)
	p := &f.questionInput.QuestionPackage
	p.ID = "independent-source-package"
	p.Version = 1
	p.Templates[0].ID = "independent-source-template"
	p.Blueprints[0].ID = "independent-source-five"
	p.Blueprints[0].Sources[0].Ref.ID = p.Templates[0].ID
	sub := f.QApproved("author_b", "reviewer_a")
	if f.count(`SELECT count(*) FROM question_submission_authors WHERE submission_id=$1 AND user_id=$2`, sub.ID, f.ids["author_a"]) != 0 {
		t.Fatal("test needs distinct content and question authors")
	}
	f.QActivate(f.QPrepare(sub.ID))
	w := f.QWithdraw(question.WithdrawalTarget{Kind: "template", ID: p.Templates[0].ID, Version: 1})
	var caseID string
	if e := f.db.QueryRow(`SELECT id::text FROM correction_cases WHERE withdrawal_space='question' AND withdrawal_id=$1 AND sealed`, w.EventID).Scan(&caseID); e != nil {
		t.Fatal(e)
	}
	plan := f.submitCorrectionPlan(f.createCorrectionPlan(caseID, "admin_a"), "admin_a")
	f.exec(`INSERT INTO auth_user_roles(user_id,role) VALUES($1,'reviewer') ON CONFLICT DO NOTHING`, f.ids["author_a"])
	if _, e := f.repo.DecideCorrectionPlan(f.ctx, f.Access("author_a", false), plan.Ref, correction.DecisionInput{ExpectedSequence: plan.Sequence, Decision: "approve", Reason: "The knowledge and unit author must not self-review their bound mathematical sources."}); !errors.Is(e, auth.ErrForbidden) {
		t.Fatal("bound content author self-reviewed unmapped template case", e)
	}
}
