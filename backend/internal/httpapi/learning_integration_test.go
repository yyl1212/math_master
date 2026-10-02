package httpapi

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/learning"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"os"
	"strings"
	"testing"
)

func TestLearningHTTPRealFiveAnswersAndOwnership(t *testing.T) {
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
	h := NewApplicationHandler(repo, db, AuthOptions{Accounts: accounts, PublicOrigin: privateOrigin, Content: &ContentOptions{Service: pub, PublicOrigin: privateOrigin, Configured: true}, Question: &QuestionOptions{Service: qs, PublicOrigin: privateOrigin, Configured: true}, Learning: &LearningOptions{Learning: ls, PublicOrigin: privateOrigin}})
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
	var attempt assessment.AttemptView
	contentHTTPCall(t, h, learner, "POST", "/api/v1/learning/assessments", assessment.CreateInput{Knowledge: detail.State.Knowledge, Blueprint: detail.Blueprints[0].Blueprint, Mode: assessment.ModeDiagnostic, ExpectedKnowledgeHead: detail.KnowledgeHead, ExpectedQuestionHead: *detail.QuestionHead}, &attempt, 201)
	safe, _ := json.Marshal(attempt)
	for _, f := range []string{"correctNumeric", "correctChoiceId", "explanation", "sourceMap", "witness", "parameters", "seal"} {
		if strings.Contains(string(safe), f) {
			t.Fatal("active answer field", f)
		}
	}
	for _, input := range []string{`{"answers":[]}`, `{"answers":[{"position":1}]}`} {
		learner["Idempotency-Key"], _ = auth.NewID(rand.Reader)
		w := privateRequest(h, "POST", "/api/v1/learning/assessments/"+attempt.Summary.ID+"/submit", input, learner)
		if w.Code != 400 || strings.Contains(w.Body.String(), "correctNumeric") {
			t.Fatal(w.Code, w.Body.String())
		}
		var n int
		if e = db.QueryRow("SELECT count(*) FROM assessment_answers").Scan(&n); e != nil || n != 0 {
			t.Fatal(n, e)
		}
	}
	w := privateRequest(h, "GET", "/api/v1/learning/assessments/"+attempt.Summary.ID, "", other)
	if w.Code != 404 {
		t.Fatal("other attempt", w.Code)
	}
	w = privateRequest(h, "GET", "/api/v1/learning/assets/"+attempt.Summary.ID+"/"+p.Assets[0].SHA256, "", other)
	if w.Code != 404 {
		t.Fatal("other asset", w.Code)
	}
	w = privateRequest(h, "GET", "/api/v1/learning/assets/"+attempt.Summary.ID+"/"+p.Assets[0].SHA256, "", learner)
	if w.Code != 200 || w.Header().Get("Content-Type") != "image/svg+xml" || w.Header().Get("Content-Security-Policy") != "sandbox; default-src 'none'" || w.Body.String() != string(svg) {
		t.Fatal("owned SVG", w.Code)
	}
	in := assessment.SubmitInput{Answers: []assessment.PositionAnswer{}}
	for j, q := range attempt.Questions {
		var a, b int
		if _, e = fmt.Sscanf(q.Prompt, "Calculate %d + %d.", &a, &b); e != nil {
			t.Fatal(e)
		}
		v := fmt.Sprint(a + b)
		answer := assessment.Answer{Kind: "numeric", Raw: &v}
		if j == 4 {
			answer = assessment.Answer{Kind: "skipped"}
		}
		in.Answers = append(in.Answers, assessment.PositionAnswer{Position: q.Position, Instance: q.Instance, Answer: answer})
	}
	var result assessment.ResultView
	contentHTTPCall(t, h, learner, "POST", "/api/v1/learning/assessments/"+attempt.Summary.ID+"/submit", in, &result, 200)
	if result.Score == nil || *result.Score != 4 || !result.Progress.QualificationGranted {
		t.Fatal(result)
	}
	oldkey := learner["Idempotency-Key"]
	body, _ := json.Marshal(in)
	w = privateRequest(h, "POST", "/api/v1/learning/assessments/"+attempt.Summary.ID+"/submit", string(body), learner)
	if w.Code != 200 || learner["Idempotency-Key"] != oldkey {
		t.Fatal(w.Code)
	}
	var n int
	if e = db.QueryRow("SELECT count(*) FROM assessment_results").Scan(&n); e != nil || n != 1 {
		t.Fatal(n, e)
	}
	w = privateRequest(h, "GET", "/api/v1/learning/assessments/"+attempt.Summary.ID+"/result", "", other)
	if w.Code != 404 {
		t.Fatal("other result", w.Code)
	}
}
