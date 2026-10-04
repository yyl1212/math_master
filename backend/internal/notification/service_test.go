package notification

import (
	"context"
	"errors"
	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/correction"
	"github.com/yyl1212/math_master/backend/internal/question"
	"testing"
)

type serviceRepo struct {
	Repository
	calls  int
	action correction.Action
}

func (r *serviceRepo) CorrectionPreflight(_ context.Context, _ question.Access, a correction.Action) (auth.User, error) {
	r.calls++
	r.action = a
	return auth.User{ID: "actor"}, nil
}
func TestNotificationService(t *testing.T) {
	if _, e := NewService(nil); !errors.Is(e, correction.ErrNotConfigured) {
		t.Fatal(e)
	}
	r := &serviceRepo{}
	s, e := NewService(r)
	if e != nil {
		t.Fatal(e)
	}
	u, e := s.Preflight(context.Background(), question.Access{}, MarkReadAction)
	if e != nil || u.ID != "actor" || r.calls != 1 || r.action != correction.ReadOwnAction {
		t.Fatal(e)
	}
	if _, e = s.Preflight(context.Background(), question.Access{}, Action("unknown")); !errors.Is(e, auth.ErrInvalidInput) {
		t.Fatal(e)
	}
}
