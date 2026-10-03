package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/feedback"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"os"
	"strings"
	"testing"
)

func TestFeedbackHTTPRealWorkflow(t *testing.T) {
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
	h := NewApplicationHandler(repo, db, AuthOptions{Accounts: accounts, Feedback: &FeedbackOptions{Service: feedbackService, PublicOrigin: privateOrigin}, PublicOrigin: privateOrigin, Content: &ContentOptions{Service: pub, PublicOrigin: privateOrigin, Configured: true}, Question: &QuestionOptions{Service: qs, PublicOrigin: privateOrigin, Configured: true}, Learning: &LearningOptions{Learning: ls, PublicOrigin: privateOrigin}})
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
	var detail learning.KnowledgeDetail
	contentHTTPCall(t, h, learner, "GET", "/api/v1/learning/knowledge/learning-http-root?version=1", nil, &detail, 200)
	var practice assessment.PracticeView
	contentHTTPCall(t, h, learner, "POST", "/api/v1/learning/practice", assessment.PracticeCreateInput{Knowledge: detail.State.Knowledge, ExpectedKnowledgeHead: detail.KnowledgeHead, ExpectedQuestionHead: *detail.QuestionHead}, &practice, 201)
	var attempt assessment.AttemptView
	contentHTTPCall(t, h, learner, "POST", "/api/v1/learning/assessments", assessment.CreateInput{Knowledge: detail.State.Knowledge, Blueprint: detail.Blueprints[0].Blueprint, Mode: assessment.ModeDiagnostic, ExpectedKnowledgeHead: detail.KnowledgeHead, ExpectedQuestionHead: *detail.QuestionHead}, &attempt, 201)
	var root, private feedback.Envelope[feedback.Context]
	for _, path := range []string{"site?area=home", "knowledge/learning-http-root", "path/feedback-http-path", "practice/" + practice.Summary.ID, "assessment/" + attempt.Summary.ID + "?position=5"} {
		var v feedback.Envelope[feedback.Context]
		contentHTTPCall(t, h, learner, "GET", "/api/v1/feedback/contexts/"+path, nil, &v, 200)
		if strings.HasPrefix(path, "knowledge") {
			root = v
		}
		if strings.HasPrefix(path, "assessment") {
			private = v
		}
		safe, _ := json.Marshal(v)
		if strings.Contains(string(safe), "correctNumeric") {
			t.Fatal("context answer")
		}
	}
	in := feedback.CreateInput{Target: root.Data.Target, Source: root.Data.Source, Category: "math_error", Title: "title-answer-sentinel", Message: strings.Repeat("中", 4000), Location: "location-answer-sentinel"}
	var created feedback.Envelope[feedback.Receipt]
	contentHTTPCall(t, h, learner, "POST", "/api/v1/feedback/tickets", in, &created, 201)
	key := learner["Idempotency-Key"]
	id := created.Data.Ticket.ID
	for _, path := range []string{"tickets", "tickets/" + id, "tickets/" + id + "/events"} {
		contentHTTPCall(t, h, learner, "GET", "/api/v1/feedback/"+path, nil, nil, 200)
	}
	for _, path := range []string{"review/tickets", "review/tickets/" + id, "review/tickets/" + id + "/events"} {
		contentHTTPCall(t, h, reviewer, "GET", "/api/v1/feedback/"+path, nil, nil, 200)
	}
	var advanced feedback.Envelope[feedback.Receipt]
	contentHTTPCall(t, h, reviewer, "POST", "/api/v1/feedback/review/tickets/"+id+"/transition", feedback.TransitionInput{ExpectedSequence: 1, Status: feedback.Processing, Message: "Independent check"}, &advanced, 200)
	contentHTTPCall(t, h, learner, "POST", "/api/v1/feedback/tickets/"+id+"/reply", feedback.ReplyInput{ExpectedSequence: 2, Message: "supplement-answer-sentinel"}, &advanced, 200)
	learner["Idempotency-Key"] = key
	b, _ := json.Marshal(in)
	w := privateRequest(h, "POST", "/api/v1/feedback/tickets", string(b), learner)
	var replay feedback.Envelope[feedback.Receipt]
	if w.Code != 201 || json.Unmarshal(w.Body.Bytes(), &replay) != nil || replay.Data.Ticket.Sequence != 1 || strings.Contains(w.Body.String(), "answer-sentinel") {
		t.Fatal("original HTTP receipt", w.Code)
	}
	var live feedback.Envelope[feedback.Metadata]
	contentHTTPCall(t, h, learner, "GET", "/api/v1/feedback/tickets/"+id, nil, &live, 200)
	if live.Data.Sequence != 3 {
		t.Fatal("replay advanced")
	}
	if w = privateRequest(h, "GET", "/api/v1/feedback/review/tickets", "", author); w.Code != 403 {
		t.Fatal("editor queue", w.Code)
	}
	if w = privateRequest(h, "GET", "/api/v1/feedback/tickets/"+id, "", other); w.Code != 404 {
		t.Fatal("other owner", w.Code)
	}
	in.Target = private.Data.Target
	in.Source = private.Data.Source
	contentHTTPCall(t, h, learner, "POST", "/api/v1/feedback/tickets", in, &created, 201)
	iid := created.Data.Ticket.ID
	w = privateRequest(h, "GET", "/api/v1/feedback/tickets/"+iid+"/events", "", learner)
	if w.Code != 409 || !strings.Contains(w.Body.String(), "FEEDBACK_ANSWER_OVERLAP") || strings.Contains(w.Body.String(), "answer-sentinel") {
		t.Fatal("HTTP overlap", w.Code)
	}
	contentHTTPCall(t, h, reviewer, "GET", "/api/v1/feedback/review/tickets/"+iid+"/events", nil, nil, 200)
	var exposures int
	if e = db.QueryRow(`SELECT count(*) FROM learner_answer_exposures WHERE owner_user_id=$1 AND kind='template'`, ru.ID).Scan(&exposures); e != nil || exposures < 1 {
		t.Fatal("handler exposure", e)
	}
	contentHTTPCall(t, h, learner, "POST", "/api/v1/learning/assessments/"+attempt.Summary.ID+"/abandon", struct{}{}, nil, 200)
	contentHTTPCall(t, h, learner, "GET", "/api/v1/feedback/tickets/"+iid+"/events", nil, nil, 200)
	unconfigured := NewApplicationHandler(repo, db, AuthOptions{Accounts: accounts, PublicOrigin: privateOrigin})
	w = privateRequest(unconfigured, "GET", "/api/v1/feedback/tickets", "", learner)
	if w.Code != 503 || !strings.Contains(w.Body.String(), "FEEDBACK_NOT_CONFIGURED") {
		t.Fatal("missing feedback", w.Code)
	}
}
