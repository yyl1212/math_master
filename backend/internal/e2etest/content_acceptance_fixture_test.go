package e2etest

import (
	"context"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/content"
	"github.com/yyl1212/math_master/backend/internal/question"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"math/big"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestContentAcceptanceFixture(t *testing.T) {
	db := testutil.Database(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	root, e := rootDir()
	if e != nil {
		t.Fatal(e)
	}
	if e = store.Up(ctx, db, root+"/db/migrations"); e != nil {
		t.Fatal(e)
	}
	repo := store.New(db)
	accounts, admin, e := fixtureAccounts(repo)
	if e != nil {
		t.Fatal(e)
	}
	c := &questionControls{db: db, repo: repo, accounts: accounts, admin: admin, root: root}
	if e = c.contentAcceptanceScene(ctx); e != nil {
		t.Fatal(e)
	}
	facts, e := repo.ReadContentAudit(ctx, content.VersionRef{ID: "elementary-foundations", Version: 1})
	if e != nil || !facts.FixtureOnly || len(facts.Content.Knowledge) != 30 || len(facts.Bank.Templates) != 24 || len(facts.Bank.Instances) != 834 || len(facts.Bank.Blueprints) != 30 {
		t.Fatal("new fixture", e)
	}
	instances := map[question.Identity]question.Instance{}
	for _, i := range facts.Bank.Instances {
		instances[i.Identity] = i
		ok, err := assessment.GradeAnswer(i, acceptanceAnswer(i))
		if err != nil || !ok {
			t.Fatal("technical answer adapter", i.Identity.ID, err)
		}
	}
	access, e := fixtureAccess(ctx, accounts, "auth_learner", false)
	if e != nil {
		t.Fatal(e)
	}
	next := func() {
		access, e = nextFixtureAccess(access)
		if e != nil {
			t.Fatal(e)
		}
	}
	for _, node := range facts.Path.Nodes {
		d, e := repo.ReadLearningKnowledge(ctx, access, node.ID, node.Version)
		if e != nil || len(d.Blueprints) != 1 || !d.Blueprints[0].Ready {
			t.Fatal(node, "first ready", e)
		}
		next()
		a, e := repo.CreateAssessment(ctx, access, assessment.CreateInput{Knowledge: d.State.Knowledge, Blueprint: d.Blueprints[0].Blueprint, Mode: assessment.ModeDiagnostic, ExpectedKnowledgeHead: d.KnowledgeHead, ExpectedQuestionHead: *d.QuestionHead})
		if e != nil || len(a.Questions) != 5 {
			t.Fatal(node, "five", e)
		}
		raw, _ := json.Marshal(a)
		if strings.Contains(string(raw), `"correctChoiceId":`) || strings.Contains(string(raw), `"correctNumeric":`) || strings.Contains(string(raw), `"explanation":`) {
			t.Fatal("attempt answer leaked")
		}
		for _, q := range a.Questions {
			if q.Knowledge.ID != node.ID || q.Knowledge.Version != node.Version {
				t.Fatal("secondary counted")
			}
		}
		answers := assessment.SubmitInput{Answers: []assessment.PositionAnswer{}}
		for _, q := range a.Questions {
			answers.Answers = append(answers.Answers, assessment.PositionAnswer{Position: q.Position, Instance: q.Instance, Answer: acceptanceAnswer(instances[q.Instance])})
		}
		next()
		r, e := repo.SubmitAssessment(ctx, access, a.Summary.ID, answers)
		if e != nil || r.Score == nil || *r.Score != 5 || !r.Progress.QualificationGranted {
			t.Fatal(node, "honest pass", e)
		}
		next()
		p, e := repo.CreatePractice(ctx, access, assessment.PracticeCreateInput{Knowledge: d.State.Knowledge, ExpectedKnowledgeHead: d.KnowledgeHead, ExpectedQuestionHead: *d.QuestionHead})
		if e != nil {
			t.Fatal(node, "practice", e)
		}
		next()
		if _, e = repo.RevealPractice(ctx, access, p.Summary.ID); e != nil {
			t.Fatal(e)
		}
		next()
		second, e := repo.CreateAssessment(ctx, access, assessment.CreateInput{Knowledge: d.State.Knowledge, Blueprint: d.Blueprints[0].Blueprint, Mode: assessment.ModeReview, ExpectedKnowledgeHead: d.KnowledgeHead, ExpectedQuestionHead: *d.QuestionHead})
		if e != nil || len(second.Questions) != 5 {
			t.Fatal(node, "after exposure", e)
		}
		for _, q := range second.Questions {
			if q.Instance == p.Question.Instance {
				t.Fatal(node, "practice instance exposure ignored")
			}
		}
		next()
		if _, e = repo.AbandonAssessment(ctx, access, second.Summary.ID); e != nil {
			t.Fatal(e)
		}
	}
}
func acceptanceAnswer(i question.Instance) assessment.Answer {
	if i.Body.CorrectChoiceID != nil {
		return assessment.Answer{Kind: "choice", ChoiceID: i.Body.CorrectChoiceID}
	}
	r := i.Body.CorrectNumeric.Numerator + "/" + i.Body.CorrectNumeric.Denominator
	if i.Body.AnswerFormat != nil && *i.Body.AnswerFormat == "percentage" {
		value, _ := new(big.Rat).SetString(r)
		value.Mul(value, big.NewRat(100, 1))
		r = strings.TrimRight(strings.TrimRight(value.FloatString(20), "0"), ".") + "%"
	}
	return assessment.Answer{Kind: "numeric", Raw: &r}
}
func TestContentAcceptanceOldScenesReset(t *testing.T) {
	s, _, _, _ := startHarness(t)
	client := &http.Client{Timeout: 45 * time.Second}
	for _, scene := range []string{"content-acceptance", "draft", "question", "learning-basic", "feedback", "correction", "content-acceptance"} {
		req, _ := http.NewRequest("POST", s.ControlURL+"/scene/"+scene, nil)
		req.Header.Set("Authorization", "Bearer "+s.Token)
		r, e := client.Do(req)
		if e != nil {
			t.Fatal(scene, "request failure")
		}
		r.Body.Close()
		if r.StatusCode != 204 {
			t.Fatal(scene, r.StatusCode)
		}
	}
}
