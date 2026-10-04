package httpapi

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/notification"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"net/http"
	"os"
	"strings"
	"testing"
)

type correctionRealFixture struct {
	db                                        *sql.DB
	repo                                      *store.Store
	ctx                                       context.Context
	h                                         http.Handler
	author, reviewer, manager, learner, other map[string]string
	authorID                                  string
	accounts                                  *auth.Service
}

func correctionRealHTTPFixture(t *testing.T) *correctionRealFixture {
	db := testutil.Database(t)
	ctx := context.Background()
	if e := store.Up(ctx, db, "../../../db/migrations"); e != nil {
		t.Fatal(e)
	}
	repo := store.New(db)
	cf, pf, assets := testutil.Seed(t)
	cfile, _ := os.Open(cf)
	c, e := content.DecodeCatalogue(cfile)
	cfile.Close()
	if e != nil {
		t.Fatal(e)
	}
	pfile, _ := os.Open(pf)
	p, e := content.DecodePackage(pfile)
	pfile.Close()
	if e != nil {
		t.Fatal(e)
	}
	sealed, report := content.ValidateAndSeal(c, p, assets)
	if len(report.Errors) > 0 {
		t.Fatal(report.Errors)
	}
	if _, e = repo.ImportDraft(ctx, sealed); e != nil {
		t.Fatal(e)
	}
	hasher := auth.NewArgon2Hasher(rand.Reader)
	accounts, e := auth.NewService(repo, hasher, rand.Reader)
	if e != nil {
		t.Fatal(e)
	}
	pub := publication.NewService(repo)
	qs, _ := question.NewService(repo, pub.AcquireValidation)
	ls, _ := learning.NewService(repo, pub.AcquireValidation)
	feedbackService, _ := feedback.NewService(repo)
	correctionService, _ := correction.NewService(repo)
	notificationService, _ := notification.NewService(repo)
	h := NewApplicationHandler(repo, db, AuthOptions{Accounts: accounts, Correction: &CorrectionOptions{Service: correctionService, PublicOrigin: privateOrigin}, Notification: &NotificationOptions{Service: notificationService, PublicOrigin: privateOrigin}, Feedback: &FeedbackOptions{Service: feedbackService, PublicOrigin: privateOrigin}, PublicOrigin: privateOrigin, Content: &ContentOptions{Service: pub, PublicOrigin: privateOrigin, Configured: true}, Question: &QuestionOptions{Service: qs, PublicOrigin: privateOrigin, Configured: true}, Learning: &LearningOptions{Learning: ls, PublicOrigin: privateOrigin}})
	author, au := contentRealActor(t, h, "learning_http_author")
	reviewer, ru := contentRealActor(t, h, "learning_http_reviewer")
	manager, mu := contentRealActor(t, h, "learning_http_manager")
	learner, _ := contentRealActor(t, h, "learning_http_learner")
	other, _ := contentRealActor(t, h, "learning_http_other")
	for _, v := range []struct{ id, role string }{{au.ID, "editor"}, {ru.ID, "reviewer"}, {mu.ID, "admin"}} {
		if _, e = db.Exec(`INSERT INTO auth_user_roles(user_id,role)VALUES($1,$2)`, v.id, v.role); e != nil {
			t.Fatal(e)
		}
	}
	raw, e := os.ReadFile("../content/testdata/workflow-ready.json")
	if e != nil {
		t.Fatal(e)
	}
	if json.Unmarshal(raw, &p) != nil {
		t.Fatal("original package")
	}
	p.Knowledge[0].ID = "learning-http-root"
	p.Units[0].Knowledge.ID = p.Knowledge[0].ID
	p.Assets[0].Knowledge.ID = p.Knowledge[0].ID
	p.Paths = []content.Path{{ID: "feedback-http-path", Version: 1, DomainIDs: p.Knowledge[0].DomainIDs, Title: "Original learning route", TitleZh: "原创学习路线", Nodes: []content.VersionRef{{ID: p.Knowledge[0].ID, Version: 1}}}}
	svg, _ := os.ReadFile("../content/testdata/workflow-ready.svg")
	var draft publication.DraftView
	contentHTTPCall(t, h, author, "POST", "/api/v1/content/drafts", publication.DraftInput{CatalogueVersion: 1, Package: p, AssetBytes: []publication.AssetInput{{ID: p.Assets[0].ID, Base64: base64.StdEncoding.EncodeToString(svg)}}, SourceMap: []publication.SourceLink{}}, &draft, 201)
	var sub publication.SubmissionView
	contentHTTPCall(t, h, author, "POST", "/api/v1/content/drafts/"+draft.ID+"/submit", publication.SubmitInput{ExpectedRevision: draft.Revision, ExpectedDigest: draft.Gate.Digest}, &sub, 201)
	contentHTTPCall(t, h, reviewer, "POST", "/api/v1/content/submissions/"+sub.ID+"/decision", publication.ReviewInput{Decision: "approve", Checks: publication.ReviewChecks{Mathematics: true, Explanations: true, Relationships: true, Sources: true, Illustrations: true}, IndependenceNote: "Different original technical test author and reviewer.", Note: "Original fraction addition fixture reviewed."}, nil, 200)
	var published publication.PublicationView
	contentHTTPCall(t, h, manager, "POST", "/api/v1/content/publications/prepare", publication.PrepareInput{SubmissionIDs: []string{sub.ID}, Reason: "Prepare original technical fixture only."}, &published, 201)
	contentHTTPCall(t, h, manager, "POST", "/api/v1/auth/reauth", auth.ReauthInput{Password: httpTestPassword}, nil, 200)
	contentHTTPCall(t, h, manager, "POST", "/api/v1/content/publications/"+published.ID+"/activate", publication.ActivateInput{ExpectedManifestSHA: published.ManifestSHA, Reason: "Activate approved original technical fixture only."}, &published, 200)
	raw, e = os.ReadFile("../../../content/questions/elementary-rationals.v1.json")
	if e != nil {
		t.Fatal(e)
	}
	var pkg question.QuestionPackage
	if json.Unmarshal(raw, &pkg) != nil {
		t.Fatal("original questions")
	}
	template := pkg.Templates[0]
	template.ID = "http-learning-addition"
	template.Knowledge = question.Ref{ID: p.Knowledge[0].ID, Version: 1}
	template.Coverage = []question.ObjectiveCoverage{{Knowledge: template.Knowledge, ObjectiveIndices: []int{0}}}
	template.Parameters = []question.Parameter{{Name: "left", Values: []string{"1", "2", "3", "4"}}, {Name: "right", Values: []string{"1", "2", "3", "4"}}}
	template.Constraints = []question.Constraint{}
	template.Units = []question.Ref{{ID: p.Units[0].ID, Version: 1}}
	template.Assets = []question.AssetRef{{ID: p.Assets[0].ID, SHA256: p.Assets[0].SHA256}}
	pkg.ID = "http-learning-questions"
	pkg.Templates = []question.Template{template}
	pkg.FixedQuestions = []question.FixedQuestion{}
	pkg.Blueprints = []question.Blueprint{{ID: "http-learning-five", Version: 1, Knowledge: template.Knowledge, CoreObjectiveIndices: []int{0}, Sources: []question.BlueprintSource{{Kind: "template", Ref: question.Ref{ID: template.ID, Version: 1}}}, CoverageNote: "Original finite addition cases cover objective zero.", RuleVersion: 1, QuestionCount: 5, PassCount: 4}}
	var qdraft question.DraftView
	contentHTTPCall(t, h, author, "POST", "/api/v1/question-bank/drafts", question.DraftInput{CatalogueVersion: 1, QuestionPackage: pkg, SourceMap: []question.SourceLink{}}, &qdraft, 201)
	var gate question.ValidationReport
	contentHTTPCall(t, h, author, "POST", "/api/v1/question-bank/drafts/"+qdraft.ID+"/validate", question.ValidateInput{ExpectedRevision: qdraft.Revision}, &gate, 200)
	if !gate.ReadyToSubmit {
		t.Fatal(gate)
	}
	var qsub question.SubmissionView
	contentHTTPCall(t, h, author, "POST", "/api/v1/question-bank/drafts/"+qdraft.ID+"/submit", question.SubmitInput{ExpectedRevision: qdraft.Revision, ExpectedDigest: gate.Digest}, &qsub, 201)
	contentHTTPCall(t, h, reviewer, "POST", "/api/v1/question-bank/submissions/"+qsub.ID+"/decision", question.ReviewInput{Decision: "approve", Checks: question.ReviewChecks{Mathematics: true, Explanations: true, Objectives: true, Sources: true, Illustrations: true, Generation: true}, IndependenceNote: "Different original author and reviewer actors.", GenerationNote: "Every generated finite case is independently verified.", Note: "Original technical fixture only."}, nil, 200)
	var qp question.PublicationSummary
	contentHTTPCall(t, h, manager, "POST", "/api/v1/question-bank/publications/prepare", question.PrepareInput{SubmissionIDs: []string{qsub.ID}, ExpectedKnowledgeHead: &published.ID, Reason: "Prepare actual approved finite HTTP questions."}, &qp, 201)
	contentHTTPCall(t, h, manager, "POST", "/api/v1/question-bank/publications/"+qp.ID+"/activate", question.ActivateInput{ExpectedKnowledgeHead: &published.ID, ExpectedManifestSHA: qp.ManifestSHA, Reason: "Activate actual approved HTTP questions."}, nil, 200)

	return &correctionRealFixture{db: db, repo: repo, ctx: ctx, h: h, author: author, reviewer: reviewer, manager: manager, learner: learner, other: other, authorID: au.ID, accounts: accounts}
}
func TestCorrectionHTTPIntegration(t *testing.T) {
	f := correctionRealHTTPFixture(t)
	h := f.h
	var detail learning.KnowledgeDetail
	contentHTTPCall(t, h, f.learner, "GET", "/api/v1/learning/knowledge/learning-http-root?version=1", nil, &detail, 200)
	var exam assessment.AttemptView
	contentHTTPCall(t, h, f.learner, "POST", "/api/v1/learning/assessments", assessment.CreateInput{Knowledge: detail.State.Knowledge, Blueprint: detail.Blueprints[0].Blueprint, Mode: assessment.ModeDiagnostic, ExpectedKnowledgeHead: detail.KnowledgeHead, ExpectedQuestionHead: *detail.QuestionHead}, &exam, 201)
	answers := assessment.SubmitInput{Answers: []assessment.PositionAnswer{}}
	for n, q := range exam.Questions {
		var raw string
		if e := f.db.QueryRow(`SELECT (body#>>'{body,body,correctNumeric,numerator}')||'/'||(body#>>'{body,body,correctNumeric,denominator}') FROM question_instances WHERE id=$1 AND version=$2`, q.Instance.ID, q.Instance.Version).Scan(&raw); e != nil {
			t.Fatal(e)
		}
		answers.Answers = append(answers.Answers, assessment.PositionAnswer{Position: n + 1, Instance: q.Instance, Answer: assessment.Answer{Kind: "numeric", Raw: &raw}})
	}
	var original assessment.ResultView
	contentHTTPCall(t, h, f.learner, "POST", "/api/v1/learning/assessments/"+exam.Summary.ID+"/submit", answers, &original, 200)
	if original.Score == nil || *original.Score != 5 || original.Outcome != assessment.Passed {
		t.Fatal("original fact")
	}
	var originalBytes []byte
	if e := f.db.QueryRow(`SELECT convert_to(to_jsonb(r)::text,'UTF8') FROM assessment_results r WHERE attempt_id=$1`, exam.Summary.ID).Scan(&originalBytes); e != nil {
		t.Fatal(e)
	}
	cIn := correction.CaseInput{Kind: correction.GradingRuleCase, Rule: &correction.RuleScope{RuleVersion: 1, Kind: "all"}}
	var command correction.Envelope[correction.Receipt]
	contentHTTPCall(t, h, f.learner, "POST", "/api/v1/corrections/cases", cIn, nil, 403)
	contentHTTPCall(t, h, f.manager, "POST", "/api/v1/corrections/cases", cIn, &command, 201)
	cid := command.Data.Case.ID
	contentHTTPCall(t, h, f.reviewer, "GET", "/api/v1/corrections/cases", nil, nil, 200)
	contentHTTPCall(t, h, f.reviewer, "GET", "/api/v1/corrections/cases/"+cid, nil, nil, 200)
	in := correction.PlanInput{AlgorithmVersion: 1, Mappings: []correction.Mapping{}, Reason: "Original correction reason sentinel."}
	contentHTTPCall(t, h, f.author, "POST", "/api/v1/corrections/cases/"+cid+"/plans", in, &command, 201)
	p := *command.Data.Plan
	path := fmt.Sprintf("/api/v1/corrections/plans/%s/versions/%d", p.Ref.ID, p.Ref.Version)
	contentHTTPCall(t, h, f.reviewer, "GET", "/api/v1/corrections/cases/"+cid+"/plans", nil, nil, 200)
	var pd correction.Envelope[correction.PlanDetail]
	contentHTTPCall(t, h, f.reviewer, "GET", path, nil, &pd, 200)
	if pd.Data.Reason != nil || len(pd.Data.Mappings) != 0 {
		t.Fatal("metadata reason leak")
	}
	in.ExpectedSequence = &p.Sequence
	in.Reason = "Updated private correction reason sentinel."
	contentHTTPCall(t, h, f.author, "PUT", path, in, &command, 200)
	p = *command.Data.Plan
	contentHTTPCall(t, h, f.reviewer, "GET", path+"/detail", nil, &pd, 200)
	if pd.Data.Reason == nil || *pd.Data.Reason != in.Reason {
		t.Fatal("protected plan")
	}
	contentHTTPCall(t, h, f.author, "POST", path+"/submit", correction.SubmitInput{ExpectedSequence: p.Sequence}, &command, 200)
	p = *command.Data.Plan
	if _, e := f.db.Exec(`INSERT INTO auth_user_roles(user_id,role) VALUES($1,'reviewer')`, f.authorID); e != nil {
		t.Fatal(e)
	}
	decision := correction.DecisionInput{ExpectedSequence: p.Sequence, Decision: "approve", Reason: "Independent exact algorithm and source verification."}
	contentHTTPCall(t, h, f.author, "POST", path+"/decision", decision, nil, 403)
	contentHTTPCall(t, h, f.reviewer, "POST", path+"/decision", decision, &command, 200)
	contentHTTPCall(t, h, f.reviewer, "GET", "/api/v1/corrections/cases/"+cid+"/jobs", nil, nil, 200)
	var job string
	if e := f.db.QueryRow(`SELECT id::text FROM correction_jobs WHERE case_id=$1 ORDER BY created_at LIMIT 1`, cid).Scan(&job); e != nil {
		t.Fatal(e)
	}
	contentHTTPCall(t, h, f.manager, "POST", "/api/v1/corrections/jobs/"+job+"/retry", correction.RetryInput{ExpectedSequence: 1}, nil, 409)
	for n := 0; n < 10; n++ {
		lease, e := f.repo.ClaimCorrectionJob(f.ctx)
		if e != nil {
			t.Fatal(e)
		}
		if lease == nil {
			break
		}
		if _, e = f.repo.ProcessCorrectionJob(f.ctx, *lease, 50); e != nil {
			t.Fatal(e)
		}
	}
	var page correction.Envelope[correction.Page[correction.ResultMetadata]]
	ep := "/api/v1/corrections/evidence?kind=assessment&id=" + exam.Summary.ID
	contentHTTPCall(t, h, f.learner, "GET", ep, nil, &page, 200)
	if len(page.Data.Items) == 0 {
		t.Fatal("own correction absent")
	}
	contentHTTPCall(t, h, f.other, "GET", ep, nil, nil, 404)
	rid := ""
	for _, v := range page.Data.Items {
		if v.Status == correction.CorrectedPassed {
			rid = v.ID
		}
	}
	if rid == "" {
		t.Fatal("approved exact result absent")
	}
	var result correction.Envelope[correction.ResultDetail]
	contentHTTPCall(t, h, f.learner, "GET", "/api/v1/corrections/results/"+rid, nil, &result, 200)
	if len(result.Data.Items) != 0 || result.Data.PlanReason != nil {
		t.Fatal("result metadata leak")
	}
	contentHTTPCall(t, h, f.learner, "GET", "/api/v1/corrections/results/"+rid+"/detail", nil, &result, 200)
	if len(result.Data.Items) != 5 || result.Data.Result.Score == nil || *result.Data.Result.Score != 5 {
		t.Fatal("owned full result")
	}
	contentHTTPCall(t, h, f.other, "GET", "/api/v1/corrections/results/"+rid+"/detail", nil, nil, 404)
	var after []byte
	if e := f.db.QueryRow(`SELECT convert_to(to_jsonb(r)::text,'UTF8') FROM assessment_results r WHERE attempt_id=$1`, exam.Summary.ID).Scan(&after); e != nil || !bytes.Equal(originalBytes, after) {
		t.Fatal("original score or outcome bytes changed", e)
	}
	contentHTTPCall(t, h, f.learner, "GET", "/api/v1/notifications", nil, nil, 200)
	contentHTTPCall(t, h, f.learner, "GET", "/api/v1/notifications/count", nil, nil, 200)
	unconfigured := NewApplicationHandler(f.repo, f.db, AuthOptions{Accounts: f.accounts, PublicOrigin: privateOrigin})
	w := privateRequest(unconfigured, "GET", "/api/v1/corrections/cases", "", f.manager)
	if w.Code != 503 || !strings.Contains(w.Body.String(), "CORRECTION_NOT_CONFIGURED") {
		t.Fatal("unconfigured", w.Code)
	}
}
