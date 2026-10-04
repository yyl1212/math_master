package correction

import (
	"context"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/question"
	"testing"
)

type serviceRepo struct {
	Repository
	calls int
	got   CaseInput
}

func (r *serviceRepo) CorrectionPreflight(context.Context, question.Access, Action) (auth.User, error) {
	r.calls++
	return auth.User{ID: "actor"}, nil
}
func (r *serviceRepo) CreateCorrectionCase(_ context.Context, _ question.Access, in CaseInput) (Envelope[Receipt], error) {
	r.got = in
	return Envelope[Receipt]{ActorID: "actor", Data: Receipt{Status: 201}}, nil
}
func TestCorrectionService(t *testing.T) {
	if _, e := NewService(nil); !errors.Is(e, ErrNotConfigured) {
		t.Fatal(e)
	}
	r := &serviceRepo{}
	s, e := NewService(r)
	if e != nil {
		t.Fatal(e)
	}
	u, e := s.Preflight(context.Background(), question.Access{}, CreateCaseAction)
	if e != nil || u.ID != "actor" || r.calls != 1 {
		t.Fatal(e)
	}
	in := CaseInput{Kind: GradingRuleCase, Rule: &RuleScope{RuleVersion: 1, Kind: "all"}}
	got, e := s.CreateCorrectionCase(context.Background(), question.Access{}, in)
	if e != nil || got.Data.Status != 201 || r.got.Kind != GradingRuleCase {
		t.Fatal(e)
	}
}
