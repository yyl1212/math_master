package learning

import (
	"context"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/publication"
	"github.com/yyl1212/math_master/backend/internal/question"
	"testing"
)

type serviceRepo struct {
	Repository
	user         auth.User
	calls, rates int
	fault        error
}

func (r *serviceRepo) LearningPreflight(context.Context, question.Access, Action) (auth.User, error) {
	r.calls++
	return r.user, r.fault
}
func (r *serviceRepo) ConsumeRates(_ context.Context, v []auth.RateKey) error {
	r.rates++
	if len(v) != 2 {
		panic("rates")
	}
	return nil
}
func (r *serviceRepo) ReadLearningOverview(context.Context, question.Access) (Overview, error) {
	return Overview{}, nil
}
func TestLearningServicePreflightOnce(t *testing.T) {
	r := &serviceRepo{user: auth.User{ID: "11111111-1111-4111-8111-111111111111", Roles: []auth.Role{auth.RoleLearner}}}
	p := publication.NewService(nil)
	s, e := NewService(r, p.AcquireValidation)
	if e != nil {
		t.Fatal(e)
	}
	for _, a := range []Action{ReadOverviewAction, ListKnowledgeAction, ReadKnowledgeAction, StartKnowledgeAction, CompleteKnowledgeAction, EnrollPathAction, ListPathsAction, ReadPathAction, ListPathNodesAction, CreatePracticeAction, ReadPracticeAction, AnswerPracticeAction, RevealPracticeAction, AbandonPracticeAction, CreateAssessmentAction, ReadAssessmentAction, SubmitAssessmentAction, AbandonAssessmentAction, ReadAssessmentResultAction, ListHistoryAction, ReadAssetAction} {
		before := r.rates
		if _, e = s.Preflight(context.Background(), question.Access{}, a); e != nil || r.rates != before+1 {
			t.Fatal(a, e)
		}
	}
	before := r.rates
	if _, e = s.ReadLearningOverview(context.Background(), question.Access{}); e != nil || r.rates != before {
		t.Fatal("business wrapper consumed rates", e)
	}
	r.user.MustChangePassword = true
	if _, e = s.Preflight(context.Background(), question.Access{}, ReadOverviewAction); !errors.Is(e, auth.ErrPasswordChangeRequired) || r.rates != before {
		t.Fatal(e)
	}
}
func TestLearningServiceSharedSlotsAndCancellation(t *testing.T) {
	p := publication.NewService(nil)
	s, _ := NewService(&serviceRepo{}, p.AcquireValidation)
	one, _ := p.AcquireValidation(context.Background())
	two, _ := s.AcquireValidation(context.Background())
	if _, e := s.AcquireValidation(context.Background()); !errors.Is(e, auth.ErrUnavailable) {
		t.Fatal(e)
	}
	one()
	one()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, e := s.AcquireValidation(ctx); !errors.Is(e, context.Canceled) {
		t.Fatal(e)
	}
	three, e := p.AcquireValidation(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	two()
	three()
	var typedNil *serviceRepo
	if _, e = NewService(typedNil, p.AcquireValidation); !errors.Is(e, ErrNotConfigured) {
		t.Fatal(e)
	}
}
