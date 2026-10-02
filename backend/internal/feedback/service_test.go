package feedback

import (
	"context"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/question"
	"testing"
)

type serviceRepo struct {
	Repository
	preflights int
}

func (r *serviceRepo) FeedbackPreflight(context.Context, question.Access, Action) (auth.User, error) {
	r.preflights++
	return auth.User{ID: "actor"}, nil
}
func TestFeedbackServicePreflightAndConstructor(t *testing.T) {
	if _, e := NewService(nil); e == nil {
		t.Fatal("nil repository")
	}
	r := &serviceRepo{}
	s, e := NewService(r)
	if e != nil {
		t.Fatal(e)
	}
	u, e := s.Preflight(context.Background(), question.Access{}, CreateAction)
	if e != nil || u.ID != "actor" || r.preflights != 1 {
		t.Fatal("preflight must only delegate authentication", u, e)
	}
}
