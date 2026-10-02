package e2etest

import (
	"context"
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
	for _, scene := range []string{"learning-basic", "learning-diagnostic", "learning-withdrawal", "learning-exposure", "learning-history", "learning-capacity", "auth"} {
		req, _ := http.NewRequest("POST", s.ControlURL+"/scene/"+scene, nil)
		req.Header.Set("Authorization", "Bearer "+s.Token)
		r, e := client.Do(req)
		if e != nil {
			t.Fatal(e)
		}
		r.Body.Close()
		if r.StatusCode != 204 {
			t.Fatal(scene, r.StatusCode)
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
