package e2etest

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/question"
	"github.com/yyl1212/math_master/backend/internal/store"
)

var correctionResetTableNames = []string{"notification_reads", "notifications", "correction_idempotency", "correction_rate_limits", "correction_events", "correction_dependencies", "correction_results", "correction_jobs", "correction_plans", "correction_cases"}

func correctionAttempt(ctx context.Context, db *sql.DB, s *store.Store, accounts *auth.Service) (assessment.AttemptView, error) {
	a, e := fixtureAccess(ctx, accounts, "auth_learner", false)
	if e != nil {
		return assessment.AttemptView{}, e
	}
	d, e := s.ReadLearningKnowledge(ctx, a, "learning-root", 1)
	if e != nil {
		return assessment.AttemptView{}, e
	}
	if d.QuestionHead == nil {
		return assessment.AttemptView{}, errors.New("missing published questions")
	}
	var b question.Identity
	e = db.QueryRowContext(ctx, `SELECT id,version,sha256 FROM question_publication_members WHERE publication_id=$1 AND kind='blueprint' AND id='learning-root-five'`, *d.QuestionHead).Scan(&b.ID, &b.Version, &b.SHA256)
	if e != nil {
		return assessment.AttemptView{}, e
	}
	a, e = nextFixtureAccess(a)
	if e != nil {
		return assessment.AttemptView{}, e
	}
	return s.CreateAssessment(ctx, a, assessment.CreateInput{Knowledge: d.State.Knowledge, Blueprint: b, Mode: assessment.ModeDiagnostic, ExpectedKnowledgeHead: d.KnowledgeHead, ExpectedQuestionHead: *d.QuestionHead})
}

// Answers are computed from the actual original numeric prompt, never rewritten later.
func setupCorrection(ctx context.Context, db *sql.DB, s *store.Store, accounts *auth.Service) error {
	p, e := correctionAttempt(ctx, db, s, accounts)
	if e != nil {
		return e
	}
	answers := make([]assessment.PositionAnswer, 0, 5)
	for _, q := range p.Questions {
		var left, right int
		if _, e = fmt.Sscanf(q.Prompt, "Calculate %d + %d.", &left, &right); e != nil {
			return errors.New("unexpected original numeric fixture prompt")
		}
		raw := fmt.Sprint(left + right)
		answers = append(answers, assessment.PositionAnswer{Position: q.Position, Instance: q.Instance, Answer: assessment.Answer{Kind: "numeric", Raw: &raw}})
	}
	a, e := fixtureAccess(ctx, accounts, "auth_learner", false)
	if e != nil {
		return e
	}
	r, e := s.SubmitAssessment(ctx, a, p.Summary.ID, assessment.SubmitInput{Answers: answers})
	if e == nil && (r.Score == nil || *r.Score != 5) {
		return errors.New("original fixture must pass honestly")
	}
	return e
}
func correctionMappings(ctx context.Context, db *sql.DB) ([]correction.Mapping, error) {
	rows, e := db.QueryContext(ctx, `SELECT i.instance_id,i.instance_version,i.instance_sha256,a.question_publication_id,m.id,m.version,m.sha256,m.publication_id FROM assessment_items i JOIN assessment_attempts a ON a.id=i.attempt_id JOIN question_instances original ON original.id=i.instance_id AND original.version=i.instance_version JOIN question_instances replacement ON replacement.template_id='learning-root-addition' AND replacement.template_version=2 AND replacement.body#>'{body,parameters}'=original.body#>'{body,parameters}' JOIN question_publication_members m ON m.id=replacement.id AND m.version=replacement.version AND m.kind='instance' AND m.publication_id=(SELECT publication_id FROM question_heads) WHERE a.state='submitted' AND EXISTS(SELECT 1 FROM question_withdrawals w WHERE w.kind='instance' AND w.target_id=i.instance_id AND w.target_version=i.instance_version AND w.sha256=i.instance_sha256) ORDER BY i.position`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []correction.Mapping{}
	for rows.Next() {
		var v correction.Mapping
		if e = rows.Scan(&v.Original.ID, &v.Original.Version, &v.Original.SHA256, &v.OriginalPublicationID, &v.Replacement.Identity.ID, &v.Replacement.Identity.Version, &v.Replacement.Identity.SHA256, &v.Replacement.PublicationID); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func correctionApprove(ctx context.Context, db *sql.DB, s *store.Store, accounts *auth.Service, equivalent bool) error {
	var id string
	if e := db.QueryRowContext(ctx, `SELECT id FROM correction_cases ORDER BY created_at,id LIMIT 1`).Scan(&id); e != nil {
		return e
	}
	m := []correction.Mapping{}
	if equivalent {
		var e error
		m, e = correctionMappings(ctx, db)
		if e != nil {
			return e
		}
		if len(m) != 1 {
			return errors.New("exact withdrawn instance mapping required")
		}
	}
	a, e := fixtureAccess(ctx, accounts, "content_editor", false)
	if e != nil {
		return e
	}
	p, e := s.CreateCorrectionPlan(ctx, a, id, correction.PlanInput{AlgorithmVersion: 1, Mappings: m, Reason: "Original independently reviewed exact answers and equivalent item intent."})
	if e != nil {
		return e
	}
	a, e = nextFixtureAccess(a)
	if e != nil {
		return e
	}
	v, e := s.SubmitCorrectionPlan(ctx, a, p.Data.Plan.Ref, correction.SubmitInput{ExpectedSequence: p.Data.Plan.Sequence})
	if e != nil {
		return e
	}
	a, e = fixtureAccess(ctx, accounts, "content_reviewer", true)
	if e != nil {
		return e
	}
	_, e = s.DecideCorrectionPlan(ctx, a, v.Data.Plan.Ref, correction.DecisionInput{ExpectedSequence: v.Data.Plan.Sequence, Decision: "approve", Reason: "Distinct reviewer checked exact original answers and every frozen source."})
	return e
}
func publishCorrectionEquivalent(ctx context.Context, db *sql.DB, s *store.Store, accounts *auth.Service, root string) error {
	in, e := questionFixtureInput(root)
	if e != nil {
		return e
	}
	t := in.QuestionPackage.Templates[0]
	t.ID = "learning-root-addition"
	t.Version = 2
	t.Knowledge = question.Ref{ID: "learning-root", Version: 1}
	t.Parameters = []question.Parameter{{Name: "left", Values: []string{"1", "2", "3", "4"}}, {Name: "right", Values: []string{"1", "2", "3", "4"}}}
	t.Constraints = []question.Constraint{}
	t.Coverage = []question.ObjectiveCoverage{{Knowledge: t.Knowledge, ObjectiveIndices: []int{0}}}
	t.Units = []question.Ref{{ID: "learning-root-unit", Version: 1}}
	learning, e := learningFixtureInput(root)
	if e != nil {
		return e
	}
	t.Assets = []question.AssetRef{{ID: learning.Package.Assets[0].ID, SHA256: learning.Package.Assets[0].SHA256}}
	t.ExplanationTemplate += "\nIndependently checked exact addition; keep the original parameters."
	in.QuestionPackage.ID = "correction-original-equivalent"
	in.QuestionPackage.Version = 1
	in.QuestionPackage.Templates = []question.Template{t}
	in.QuestionPackage.FixedQuestions = []question.FixedQuestion{}
	in.QuestionPackage.Blueprints = []question.Blueprint{{ID: "learning-root-five", Version: 2, Knowledge: t.Knowledge, CoreObjectiveIndices: []int{0}, Sources: []question.BlueprintSource{{Kind: "template", Ref: question.Ref{ID: t.ID, Version: 2}}}, CoverageNote: "Original replacement finite addition cases.", RuleVersion: 1, QuestionCount: 5, PassCount: 4}}
	author, e := fixtureAccess(ctx, accounts, "content_editor", false)
	if e != nil {
		return e
	}
	d, e := s.CreateQuestionDraft(ctx, author, in)
	if e != nil {
		return e
	}
	author, _ = nextFixtureAccess(author)
	gate, e := s.ValidateQuestionDraft(ctx, author, d.ID, question.ValidateInput{ExpectedRevision: d.Revision})
	if e != nil || !gate.ReadyToSubmit {
		return errors.New("replacement validation failed")
	}
	author, _ = nextFixtureAccess(author)
	sub, e := s.SubmitQuestionDraft(ctx, author, d.ID, question.SubmitInput{ExpectedRevision: d.Revision, ExpectedDigest: gate.Digest})
	if e != nil {
		return e
	}
	reviewer, e := fixtureAccess(ctx, accounts, "content_reviewer", false)
	if e != nil {
		return e
	}
	_, e = s.DecideQuestionReview(ctx, reviewer, sub.ID, question.ReviewInput{Decision: "approve", Checks: question.ReviewChecks{Mathematics: true, Explanations: true, Objectives: true, Sources: true, Illustrations: true, Generation: true}, IndependenceNote: "Separate original replacement reviewer.", GenerationNote: "Finite addition cases verified independently.", Note: "Original isolated technical fixture."})
	if e != nil {
		return e
	}
	var kh, qh string
	if e = db.QueryRowContext(ctx, `SELECT (SELECT snapshot_id::text FROM publication_heads),(SELECT publication_id::text FROM question_heads)`).Scan(&kh, &qh); e != nil {
		return e
	}
	manager, e := fixtureAccess(ctx, accounts, "content_admin", true)
	if e != nil {
		return e
	}
	pub, e := s.PrepareQuestionRelease(ctx, manager, question.PrepareInput{SubmissionIDs: []string{sub.ID}, ExpectedKnowledgeHead: &kh, ExpectedQuestionHead: &qh, Reason: "Prepare original independently reviewed replacement."})
	if e != nil {
		return e
	}
	manager, _ = nextFixtureAccess(manager)
	if _, e = s.ActivateQuestionRelease(ctx, manager, pub.ID, question.ActivateInput{ExpectedKnowledgeHead: &kh, ExpectedQuestionHead: &qh, ExpectedManifestSHA: pub.ManifestSHA, Reason: "Activate reviewed replacement, without regrading."}); e != nil {
		return e
	}
	return nil
}
