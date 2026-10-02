package e2etest

import (
	"context"
	"encoding/json"
	"github.com/yyl1212/math_master/backend/internal/assessment"
	"github.com/yyl1212/math_master/backend/internal/store"
	"github.com/yyl1212/math_master/backend/internal/testutil"
	"net/http"
	"testing"
	"time"
)

func TestLearningRealFixtureAndReset(t *testing.T) {
	db := testutil.Database(t)
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	root, e := rootDir()
	if e != nil {
		t.Fatal(e)
	}
	if e = store.Up(ctx, db, root+"/db/migrations"); e != nil {
		t.Fatal(e)
	}
	s := store.New(db)
	a, admin, e := fixtureAccounts(s)
	if e != nil {
		t.Fatal(e)
	}
	normal, e := loadFixture(root, false)
	if e != nil {
		t.Fatal(e)
	}
	if e = resetLearning(ctx, db, s, a, admin, root, normal, LearningBasic); e != nil {
		t.Fatal(e)
	}
	access, e := fixtureAccess(ctx, a, "auth_learner", false)
	if e != nil {
		t.Fatal(e)
	}
	detail, e := s.ReadLearningKnowledge(ctx, access, "learning-root", 1)
	if e != nil || len(detail.Blueprints) != 1 || !detail.Blueprints[0].Ready {
		t.Fatal(detail, e)
	}
	v, e := s.CreateAssessment(ctx, access, assessment.CreateInput{Knowledge: detail.State.Knowledge, Blueprint: detail.Blueprints[0].Blueprint, Mode: assessment.ModeDiagnostic, ExpectedKnowledgeHead: detail.KnowledgeHead, ExpectedQuestionHead: *detail.QuestionHead})
	if e != nil || len(v.Questions) != 5 {
		t.Fatal(v, e)
	}
	access, _ = nextFixtureAccess(access)
	_, e = s.SubmitAssessment(ctx, access, v.Summary.ID, learningSkipped(v))
	if e != nil {
		t.Fatal(e)
	}
	if handled, err := learningChange(ctx, db, s, a, root, "learning-withdraw-first-assessed"); !handled || err != nil {
		t.Fatal(handled, err)
	}
	if handled, err := learningChange(ctx, db, s, a, root, "learning-new-route"); !handled || err != nil {
		t.Fatal(handled, err)
	}
	state, err := learningState(ctx, db)
	if err != nil || state.Path.Version != 2 {
		t.Fatal(state, err)
	}
	if handled, err := learningChange(ctx, db, s, a, root, "learning-withdraw-root"); !handled || err != nil {
		t.Fatal(handled, err)
	}
	if e = resetAccounts(ctx, db, a, admin); e != nil {
		t.Fatal(e)
	}
	var n int
	if e = db.QueryRowContext(ctx, "SELECT count(*) FROM assessment_attempts").Scan(&n); e != nil || n != 0 {
		t.Fatal(n, e)
	}
}
func learningSkipped(v assessment.AttemptView) assessment.SubmitInput {
	in := assessment.SubmitInput{Answers: []assessment.PositionAnswer{}}
	for _, q := range v.Questions {
		in.Answers = append(in.Answers, assessment.PositionAnswer{Position: q.Position, Instance: q.Instance, Answer: assessment.Answer{Kind: "skipped"}})
	}
	return in
}
func TestLearningHarnessScenesArePrivate(t *testing.T) {
	s, _, _, _ := startHarness(t)
	client := &http.Client{Timeout: 40 * time.Second}
	for _, scene := range []string{"learning-basic", "learning-choice", "learning-diagnostic", "learning-withdrawal", "learning-exposure", "learning-history", "learning-capacity", "auth"} {
		req, _ := http.NewRequest("POST", s.ControlURL+"/scene/"+scene, nil)
		req.Header.Set("Authorization", "Bearer "+s.Token)
		sceneClient := client
		if scene == "learning-capacity" {
			sceneClient = &http.Client{Timeout: 4 * time.Minute}
		}
		r, e := sceneClient.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		r.Body.Close()
		if r.StatusCode != 204 {
			t.Fatal(scene, r.StatusCode)
		}
		if scene == "learning-capacity" {
			req, _ := http.NewRequest("GET", s.ControlURL+"/learning/state", nil)
			req.Header.Set("Authorization", "Bearer "+s.Token)
			r, e := client.Do(req)
			if e != nil {
				t.Fatal(e)
			}
			var state learningDatabaseState
			e = json.NewDecoder(r.Body).Decode(&state)
			r.Body.Close()
			if e != nil || r.StatusCode != 200 || len(state.Knowledge) != 1000 || len(state.Blueprints) != 1000 || state.Path.Version != 2 {
				t.Fatalf("capacity scene is not the actual maximum published fixture: knowledge=%d blueprints=%d pathVersion=%d err=%v", len(state.Knowledge), len(state.Blueprints), state.Path.Version, e)
			}
			for kind, want := range map[string]int{"knowledge": 1000, "unit": 4000, "path": 200, "asset": 1000, "template": 200, "instance": 10000, "blueprint": 1000} {
				if state.PublishedCounts[kind] != want {
					t.Fatal("capacity scene published count", kind, state.PublishedCounts[kind], want)
				}
			}
			if state.MaxPathNodes != 100 {
				t.Fatal("unapproved maximum route size", state.MaxPathNodes)
			}
		}
	}
	for _, endpoint := range []string{s.ControlURL + "/learning/state", s.APIURL + "/learning/state"} {
		r, e := client.Get(endpoint)
		if e != nil {
			t.Fatal(e)
		}
		r.Body.Close()
		if r.StatusCode != 401 && r.StatusCode != 404 {
			t.Fatal(endpoint, r.StatusCode)
		}
	}
}

func TestLearningChoiceFixtureUsesActualPublishedQuestions(t *testing.T) {
	db := testutil.Database(t)
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	root, e := rootDir()
	if e != nil {
		t.Fatal(e)
	}
	if e = store.Up(ctx, db, root+"/db/migrations"); e != nil {
		t.Fatal(e)
	}
	s := store.New(db)
	accounts, admin, e := fixtureAccounts(s)
	if e != nil {
		t.Fatal(e)
	}
	normal, e := loadFixture(root, false)
	if e != nil {
		t.Fatal(e)
	}
	if e = resetLearning(ctx, db, s, accounts, admin, root, normal, LearningChoice); e != nil {
		t.Fatal(e)
	}
	a, e := fixtureAccess(ctx, accounts, "auth_learner", false)
	if e != nil {
		t.Fatal(e)
	}
	d, e := s.ReadLearningKnowledge(ctx, a, "learning-root", 1)
	if e != nil {
		t.Fatal(e)
	}
	v, e := s.CreateAssessment(ctx, a, assessment.CreateInput{Knowledge: d.State.Knowledge, Blueprint: d.Blueprints[0].Blueprint, Mode: assessment.ModeNode, ExpectedKnowledgeHead: d.KnowledgeHead, ExpectedQuestionHead: *d.QuestionHead})
	if e != nil || len(v.Questions) != 5 {
		t.Fatal(v, e)
	}
	choices, numeric := 0, 0
	for _, q := range v.Questions {
		if q.Type == "single_choice" {
			choices++
		} else if q.Type == "numeric" {
			numeric++
		}
	}
	if choices != 1 || numeric != 4 {
		t.Fatal("actual mixed safe types", choices, numeric, v)
	}
	a, _ = nextFixtureAccess(a)
	if _, e = s.AbandonAssessment(ctx, a, v.Summary.ID); e != nil {
		t.Fatal(e)
	}
	d, e = s.ReadLearningKnowledge(ctx, a, "learning-practice-only", 1)
	if e != nil {
		t.Fatal(e)
	}
	a, _ = nextFixtureAccess(a)
	p, e := s.CreatePractice(ctx, a, assessment.PracticeCreateInput{Knowledge: d.State.Knowledge, ExpectedKnowledgeHead: d.KnowledgeHead, ExpectedQuestionHead: *d.QuestionHead})
	if e != nil || p.Question.Type != "single_choice" {
		t.Fatal(p, e)
	}
}
