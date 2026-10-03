package e2etest

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"github.com/yyl1212/math_master/backend/internal/question"
	"github.com/yyl1212/math_master/backend/internal/store"
)

type feedbackFixture struct {
	TicketID   string           `json:"ticketId"`
	PracticeID string           `json:"practiceId"`
	Instance   feedback.Context `json:"instance"`
}

// The caller first publishes the original learning fixture through its real review workflow.
func setupFeedback(ctx context.Context, s *store.Store) (feedbackFixture, error) {
	accounts, admin, e := fixtureAccounts(s)
	if e != nil {
		return feedbackFixture{}, e
	}
	owner, e := fixtureAccess(ctx, accounts, "learning_other", false)
	if e != nil {
		return feedbackFixture{}, e
	}
	detail, e := s.ReadLearningKnowledge(ctx, owner, "learning-root", 1)
	if e != nil {
		return feedbackFixture{}, e
	}
	p, e := s.CreatePractice(ctx, owner, assessment.PracticeCreateInput{Knowledge: detail.State.Knowledge, ExpectedKnowledgeHead: detail.KnowledgeHead, ExpectedQuestionHead: *detail.QuestionHead})
	if e != nil {
		return feedbackFixture{}, e
	}
	owner, _ = nextFixtureAccess(owner)
	c, e := s.ReadFeedbackContext(ctx, owner, feedback.ContextQuery{Kind: "practice", ID: p.Summary.ID})
	if e != nil {
		return feedbackFixture{}, e
	}
	owner, _ = nextFixtureAccess(owner)
	r, e := s.CreateFeedback(ctx, owner, feedback.CreateInput{Target: c.Data.Target, Source: c.Data.Source, Category: feedback.Category("math_error"), Title: `<img src="https://invalid.example/answer">`, Message: "answer-sentinel\nOriginal private addition explanation. 中", Location: "First line\nSecond line"})
	if e != nil {
		return feedbackFixture{}, e
	}
	owner, _ = nextFixtureAccess(owner)
	if _, e = s.AbandonPractice(ctx, owner, p.Summary.ID); e != nil {
		return feedbackFixture{}, e
	}
	anonymous, delta, e := accounts.Context(ctx, auth.Cookies{})
	if e != nil {
		return feedbackFixture{}, e
	}
	user, _, e := accounts.Register(ctx, auth.Cookies{Preauth: delta.SetPreauth}, anonymous.CSRFToken, auth.RegisterInput{Username: "feedback_editor", Password: fixturePassword}, "feedback-editor-register")
	if e != nil {
		return feedbackFixture{}, e
	}

	view, pre, e := accounts.Context(ctx, auth.Cookies{})
	if e != nil {
		return feedbackFixture{}, e
	}
	_, session, e := accounts.Login(ctx, auth.Cookies{Preauth: pre.SetPreauth}, view.CSRFToken, auth.LoginInput{Username: "content_admin", Password: fixturePassword}, "feedback-editor-admin")
	if e != nil {
		return feedbackFixture{}, e
	}
	cookies := auth.Cookies{Session: session.SetSession}
	view, _, e = accounts.Context(ctx, cookies)
	if e != nil {
		return feedbackFixture{}, e
	}
	if _, e = accounts.Reauthenticate(ctx, cookies, view.CSRFToken, auth.ReauthInput{Password: fixturePassword}, "feedback-editor-verify"); e != nil {
		return feedbackFixture{}, e
	}
	if _, e = admin.ReplaceRoles(ctx, cookies, view.CSRFToken, user.ID, auth.RolesInput{Roles: []auth.Role{auth.RoleLearner, auth.RoleEditor}, Reason: "Isolated editor-only feedback permission fixture."}, "feedback-editor-roles"); e != nil {
		return feedbackFixture{}, e
	}

	handlerView, handlerDelta, e := accounts.Context(ctx, auth.Cookies{})
	if e != nil {
		return feedbackFixture{}, e
	}
	handlerUser, _, e := accounts.Register(ctx, auth.Cookies{Preauth: handlerDelta.SetPreauth}, handlerView.CSRFToken, auth.RegisterInput{Username: "feedback_handler", Password: fixturePassword}, "feedback-handler-register")
	if e != nil {
		return feedbackFixture{}, e
	}
	if _, e = admin.ReplaceRoles(ctx, cookies, view.CSRFToken, handlerUser.ID, auth.RolesInput{Roles: []auth.Role{auth.RoleLearner, auth.RoleReviewer}, Reason: "Original unexposed feedback handler and assessment learner."}, "feedback-handler-roles"); e != nil {
		return feedbackFixture{}, e
	}
	return feedbackFixture{TicketID: r.Data.Ticket.ID, PracticeID: p.Summary.ID, Instance: c.Data}, nil
}

// Publish a new template version and blueprint through distinct author/reviewer accounts.
func publishFeedbackReplacement(ctx context.Context, db *sql.DB, s *store.Store, accounts *auth.Service, root string) (feedback.Replacement, error) {
	in, e := questionFixtureInput(root)
	if e != nil {
		return feedback.Replacement{}, e
	}
	t := in.QuestionPackage.Templates[0]
	t.ID = "learning-root-addition"
	t.Version = 2
	t.Knowledge = question.Ref{ID: "learning-root", Version: 1}
	t.Parameters = []question.Parameter{{Name: "left", Values: []string{"11", "12", "13", "14"}}, {Name: "right", Values: []string{"1", "2", "3", "4"}}}
	t.Constraints = []question.Constraint{}
	t.Coverage = []question.ObjectiveCoverage{{Knowledge: t.Knowledge, ObjectiveIndices: []int{0}}}
	t.Units = []question.Ref{{ID: "learning-root-unit", Version: 1}}
	t.Assets = []question.AssetRef{}
	in.QuestionPackage.ID = "feedback-original-replacement"
	in.QuestionPackage.Version = 1
	in.QuestionPackage.Templates = []question.Template{t}
	in.QuestionPackage.FixedQuestions = []question.FixedQuestion{}
	in.QuestionPackage.Blueprints = []question.Blueprint{{ID: "learning-root-five", Version: 2, Knowledge: t.Knowledge, CoreObjectiveIndices: []int{0}, Sources: []question.BlueprintSource{{Kind: "template", Ref: question.Ref{ID: t.ID, Version: 2}}}, CoverageNote: "Original replacement finite addition cases.", RuleVersion: 1, QuestionCount: 5, PassCount: 4}}
	author, e := fixtureAccess(ctx, accounts, "content_editor", false)
	if e != nil {
		return feedback.Replacement{}, e
	}
	d, e := s.CreateQuestionDraft(ctx, author, in)
	if e != nil {
		return feedback.Replacement{}, e
	}
	author, _ = nextFixtureAccess(author)
	gate, e := s.ValidateQuestionDraft(ctx, author, d.ID, question.ValidateInput{ExpectedRevision: d.Revision})
	if e != nil || !gate.ReadyToSubmit {
		return feedback.Replacement{}, errors.New("replacement validation failed")
	}
	author, _ = nextFixtureAccess(author)
	sub, e := s.SubmitQuestionDraft(ctx, author, d.ID, question.SubmitInput{ExpectedRevision: d.Revision, ExpectedDigest: gate.Digest})
	if e != nil {
		return feedback.Replacement{}, e
	}
	reviewer, e := fixtureAccess(ctx, accounts, "content_reviewer", false)
	if e != nil {
		return feedback.Replacement{}, e
	}
	_, e = s.DecideQuestionReview(ctx, reviewer, sub.ID, question.ReviewInput{Decision: "approve", Checks: question.ReviewChecks{Mathematics: true, Explanations: true, Objectives: true, Sources: true, Illustrations: true, Generation: true}, IndependenceNote: "Separate original replacement reviewer.", GenerationNote: "Finite addition cases verified independently.", Note: "Original isolated technical fixture."})
	if e != nil {
		return feedback.Replacement{}, e
	}
	var kh, qh string
	if e = db.QueryRowContext(ctx, `SELECT (SELECT snapshot_id::text FROM publication_heads),(SELECT publication_id::text FROM question_heads)`).Scan(&kh, &qh); e != nil {
		return feedback.Replacement{}, e
	}
	manager, e := fixtureAccess(ctx, accounts, "content_admin", true)
	if e != nil {
		return feedback.Replacement{}, e
	}
	pub, e := s.PrepareQuestionRelease(ctx, manager, question.PrepareInput{SubmissionIDs: []string{sub.ID}, ExpectedKnowledgeHead: &kh, ExpectedQuestionHead: &qh, Reason: "Prepare original independently reviewed replacement."})
	if e != nil {
		return feedback.Replacement{}, e
	}
	manager, _ = nextFixtureAccess(manager)
	if _, e = s.ActivateQuestionRelease(ctx, manager, pub.ID, question.ActivateInput{ExpectedKnowledgeHead: &kh, ExpectedQuestionHead: &qh, ExpectedManifestSHA: pub.ManifestSHA, Reason: "Activate reviewed replacement, without regrading."}); e != nil {
		return feedback.Replacement{}, e
	}
	var id question.Identity
	if e = db.QueryRowContext(ctx, `SELECT m.id,m.version,m.sha256 FROM question_publication_members m JOIN question_instances i ON i.id=m.id AND i.version=m.version WHERE m.publication_id=$1 AND m.kind='instance' AND i.template_id='learning-root-addition' AND i.template_version=2 ORDER BY m.id LIMIT 1`, pub.ID).Scan(&id.ID, &id.Version, &id.SHA256); e != nil {
		return feedback.Replacement{}, e
	}
	return feedback.Replacement{Kind: "instance", Identity: &id, PublicationID: pub.ID}, nil
}

// Decode only safe fixture context, never discussion text, from original persisted tickets.
func fixtureFeedbackTarget(ctx context.Context, db *sql.DB) (string, feedback.Target, error) {
	var id string
	var raw []byte
	e := db.QueryRowContext(ctx, `SELECT id,target FROM feedback_tickets WHERE target->>'kind'='instance' ORDER BY created_at,id LIMIT 1`).Scan(&id, &raw)
	var target feedback.Target
	if e == nil {
		e = json.Unmarshal(raw, &target)
	}
	return id, target, e
}
