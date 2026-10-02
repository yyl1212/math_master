package e2etest

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"github.com/yyl1212/math_master/backend/internal/store"
	"os"
	"path/filepath"
)

type LearningScenario string

const (
	LearningBasic      LearningScenario = "basic"
	LearningDiagnostic LearningScenario = "diagnostic"
	LearningWithdrawal LearningScenario = "withdrawal"
	LearningExposure   LearningScenario = "exposure"
	LearningHistory    LearningScenario = "history"
	LearningCapacity   LearningScenario = "capacity"
)

func learningScenario(s string) (LearningScenario, bool) {
	for _, a := range []LearningScenario{LearningBasic, LearningDiagnostic, LearningWithdrawal, LearningExposure, LearningHistory, LearningCapacity} {
		if s == "learning-"+string(a) {
			return a, true
		}
	}
	return "", false
}
func learningFixtureInput(root string) (publication.DraftInput, error) {
	raw, e := os.ReadFile(filepath.Join(root, "backend/internal/content/testdata/workflow-ready.json"))
	if e != nil {
		return publication.DraftInput{}, e
	}
	var p content.Package
	if json.Unmarshal(raw, &p) != nil {
		return publication.DraftInput{}, errors.New("learning package invalid")
	}
	p.ID = "learning-original-lessons"
	base := p.Knowledge[0]
	p.Knowledge = []content.Knowledge{}
	p.Units = []content.Unit{}
	ids := []string{"learning-root", "learning-middle", "learning-target", "learning-practice-only"}
	for i, id := range ids {
		k := base
		k.ID = id
		k.Title = []string{"Addition basics", "Connected addition", "Addition fluency", "Independent practice"}[i]
		k.TitleZh = "加法练习"
		k.Statement = "The sum of rational numbers is a rational number."
		k.Objectives = []string{"Add two rational numbers exactly."}
		k.Relations = []content.Relation{}
		if i == 1 || i == 2 {
			k.Relations = append(k.Relations, content.Relation{Kind: "prerequisite", Target: content.VersionRef{ID: ids[i-1], Version: 1}})
		}
		if i == 2 {
			k.Relations = append(k.Relations, content.Relation{Kind: "prerequisite", Target: content.VersionRef{ID: ids[0], Version: 1}})
		}
		p.Knowledge = append(p.Knowledge, k)
		u := content.Unit{ID: id + "-unit", Version: 1, Knowledge: content.VersionRef{ID: id, Version: 1}, Angles: []content.Angle{{Kind: "formal", Body: "For rational numbers, use a common denominator to add."}, {Kind: "intuitive", Body: "One half plus one half forms one whole."}}, Examples: []string{"One half plus one half equals one."}, Counterexamples: []string{}, AssetIDs: []string{}}
		if i == 0 {
			u.AssetIDs = []string{"halves"}
			u.Angles = append(u.Angles, content.Angle{Kind: "visual", Body: "![Two equal halves](asset:halves)"})
		}
		p.Units = append(p.Units, u)
	}
	p.Assets[0].Knowledge = content.VersionRef{ID: ids[0], Version: 1}
	p.Paths = []content.Path{{ID: "learning-route", Version: 1, DomainIDs: []string{"elementary-mathematics"}, Title: "Addition learning route", TitleZh: "加法学习路线", Nodes: []content.VersionRef{{ID: ids[0], Version: 1}, {ID: ids[1], Version: 1}, {ID: ids[2], Version: 1}}}}
	svg, e := os.ReadFile(filepath.Join(root, "backend/internal/content/testdata/workflow-ready.svg"))
	if e != nil {
		return publication.DraftInput{}, e
	}
	return publication.DraftInput{CatalogueVersion: 1, Package: p, AssetBytes: []publication.AssetInput{{ID: "halves", Base64: base64.StdEncoding.EncodeToString(svg)}}, SourceMap: []publication.SourceLink{}}, nil
}
func learningPublishContent(ctx context.Context, db *sql.DB, s *store.Store, accounts *auth.Service, in publication.DraftInput) error {
	author, e := fixtureAccess(ctx, accounts, "content_editor", false)
	if e != nil {
		return e
	}
	d, e := s.CreateDraft(ctx, author, in)
	if e != nil {
		return e
	}
	author, _ = nextFixtureAccess(author)
	sub, e := s.SubmitDraft(ctx, author, d.ID, publication.SubmitInput{ExpectedRevision: d.Revision, ExpectedDigest: d.Gate.Digest})
	if e != nil {
		return e
	}
	r, e := fixtureAccess(ctx, accounts, "content_reviewer", false)
	if e != nil {
		return e
	}
	_, e = s.DecideReview(ctx, r, sub.ID, publication.ReviewInput{Decision: "approve", Checks: publication.ReviewChecks{Mathematics: true, Explanations: true, Relationships: true, Sources: true, Illustrations: true}, IndependenceNote: "Distinct original fixture author and reviewer.", Note: "Technical fixture only; original rational addition and SVG."})
	if e != nil {
		return e
	}
	manager, e := fixtureAccess(ctx, accounts, "content_admin", true)
	if e != nil {
		return e
	}
	var old *string
	var h string
	e = db.QueryRowContext(ctx, "SELECT snapshot_id FROM publication_heads").Scan(&h)
	if e == nil {
		old = &h
	} else if e != sql.ErrNoRows {
		return e
	}
	pub, e := s.PrepareRelease(ctx, manager, publication.PrepareInput{SubmissionIDs: []string{sub.ID}, ExpectedHead: old, Reason: "Prepare original isolated learning fixture."})
	if e != nil {
		return e
	}
	manager, _ = nextFixtureAccess(manager)
	_, e = s.ActivateRelease(ctx, manager, pub.ID, publication.ActivateInput{ExpectedHead: old, ExpectedManifestSHA: pub.ManifestSHA, Reason: "Publish approved isolated learning fixture."})
	return e
}
func learningPublishQuestions(ctx context.Context, db *sql.DB, s *store.Store, accounts *auth.Service, root string, p content.Package, scenes ...LearningScenario) error {
	in, e := questionFixtureInput(root)
	if e != nil {
		return e
	}
	base := in.QuestionPackage.Templates[0]
	in.QuestionPackage.ID = "learning-original-questions"
	in.QuestionPackage.Templates = []question.Template{}
	in.QuestionPackage.FixedQuestions = []question.FixedQuestion{}
	in.QuestionPackage.Blueprints = []question.Blueprint{}
	for index, k := range p.Knowledge {
		t := base
		t.ID = k.ID + "-addition"
		t.Knowledge = question.Ref{ID: k.ID, Version: 1}
		t.Parameters = []question.Parameter{{Name: "left", Values: []string{"1", "2", "3", "4"}}, {Name: "right", Values: []string{"1", "2", "3", "4"}}}
		t.Constraints = []question.Constraint{}
		t.Coverage = []question.ObjectiveCoverage{{Knowledge: t.Knowledge, ObjectiveIndices: []int{0}}}
		t.Units = []question.Ref{{ID: k.ID + "-unit", Version: 1}}
		t.Assets = []question.AssetRef{}
		if index == 0 {
			t.Assets = []question.AssetRef{{ID: p.Assets[0].ID, SHA256: p.Assets[0].SHA256}}
		}
		in.QuestionPackage.Templates = append(in.QuestionPackage.Templates, t)
		if index != 3 {
			in.QuestionPackage.Blueprints = append(in.QuestionPackage.Blueprints, question.Blueprint{ID: k.ID + "-five", Version: 1, Knowledge: t.Knowledge, CoreObjectiveIndices: []int{0}, Sources: []question.BlueprintSource{{Kind: "template", Ref: question.Ref{ID: t.ID, Version: 1}}}, CoverageNote: "Original finite addition cases cover the declared objective.", RuleVersion: 1, QuestionCount: 5, PassCount: 4})
		}
	}
	if len(scenes) > 0 && scenes[0] == LearningDiagnostic {
		b := in.QuestionPackage.Blueprints[0]
		b.ID = "learning-root-alternate"
		in.QuestionPackage.Blueprints = append(in.QuestionPackage.Blueprints, b)
	}
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
		return fmt.Errorf("learning question gate: %w", e)
	}
	author, _ = nextFixtureAccess(author)
	sub, e := s.SubmitQuestionDraft(ctx, author, d.ID, question.SubmitInput{ExpectedRevision: d.Revision, ExpectedDigest: gate.Digest})
	if e != nil {
		return e
	}
	r, e := fixtureAccess(ctx, accounts, "content_reviewer", false)
	if e != nil {
		return e
	}
	_, e = s.DecideQuestionReview(ctx, r, sub.ID, question.ReviewInput{Decision: "approve", Checks: question.ReviewChecks{Mathematics: true, Explanations: true, Objectives: true, Sources: true, Illustrations: true, Generation: true}, IndependenceNote: "Different isolated author and reviewer accounts.", GenerationNote: "All finite original addition cases verified through the independent verifier.", Note: "Original technical fixture only."})
	if e != nil {
		return e
	}
	manager, e := fixtureAccess(ctx, accounts, "content_admin", true)
	if e != nil {
		return e
	}
	var kh string
	if e = db.QueryRowContext(ctx, "SELECT snapshot_id FROM publication_heads").Scan(&kh); e != nil {
		return e
	}
	pub, e := s.PrepareQuestionRelease(ctx, manager, question.PrepareInput{SubmissionIDs: []string{sub.ID}, ExpectedKnowledgeHead: &kh, ExpectedQuestionHead: nil, Reason: "Prepare actual approved finite learning questions."})
	if e != nil {
		return e
	}
	manager, _ = nextFixtureAccess(manager)
	_, e = s.ActivateQuestionRelease(ctx, manager, pub.ID, question.ActivateInput{ExpectedKnowledgeHead: &kh, ExpectedQuestionHead: nil, ExpectedManifestSHA: pub.ManifestSHA, Reason: "Activate original approved learning question fixture."})
	return e
}
func resetLearning(ctx context.Context, db *sql.DB, s *store.Store, accounts *auth.Service, admin *auth.AdminService, root string, normal content.ValidatedPackage, scene LearningScenario) error {
	if _, ok := learningScenario("learning-" + string(scene)); !ok {
		return errors.New("unknown learning scenario")
	}
	if e := resetWorkflow(ctx, db, accounts, admin); e != nil {
		return e
	}
	if _, e := s.ImportDraft(ctx, normal); e != nil {
		return e
	}
	in, e := learningFixtureInput(root)
	if e != nil {
		return e
	}
	if e = learningPublishContent(ctx, db, s, accounts, in); e != nil {
		return fmt.Errorf("learning content setup: %w", e)
	}
	if e = learningPublishQuestions(ctx, db, s, accounts, root, in.Package, scene); e != nil {
		return fmt.Errorf("learning question setup: %w", e)
	}
	v, d, e := accounts.Context(ctx, auth.Cookies{})
	if e != nil {
		return e
	}
	if _, _, e = accounts.Register(ctx, auth.Cookies{Preauth: d.SetPreauth}, v.CSRFToken, auth.RegisterInput{Username: "learning_other", Password: fixturePassword}, "learning-other-register"); e != nil {
		return e
	}
	if scene == LearningHistory {
		a, e := fixtureAccess(ctx, accounts, "auth_learner", false)
		if e != nil {
			return e
		}
		detail, e := s.ReadLearningKnowledge(ctx, a, "learning-root", 1)
		if e != nil {
			return e
		}
		for n := 0; n < 101; n++ {
			a, _ = nextFixtureAccess(a)
			p, e := s.CreatePractice(ctx, a, assessment.PracticeCreateInput{Knowledge: detail.State.Knowledge, ExpectedKnowledgeHead: detail.KnowledgeHead, ExpectedQuestionHead: *detail.QuestionHead})
			if e != nil {
				return e
			}
			a, _ = nextFixtureAccess(a)
			if _, e = s.AbandonPractice(ctx, a, p.Summary.ID); e != nil {
				return e
			}
		}
	}
	return nil
}
