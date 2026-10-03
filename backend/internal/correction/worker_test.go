package correction

import (
	"context"
	"errors"
	"testing"
)

func TestCorrectionWorkerContracts(t *testing.T) {
	if _, e := NewRunner(nil, Options{Enabled: true, Limit: 50}); e == nil {
		t.Fatal("nil repository")
	}
	for _, n := range []int{-1, 51} {
		if _, e := NewRunner(&runnerRepo{}, Options{Enabled: true, Limit: n}); e == nil {
			t.Fatal("unsafe limit", n)
		}
	}
	if !errors.Is(ErrNeverEnabled, ErrNotConfigured) {
		t.Fatal("legacy error must retain not-configured classification")
	}
	for _, v := range []struct {
		e    error
		want string
	}{{context.DeadlineExceeded, "deadline"}, {ErrLeaseLost, "lease"}, {ErrSourceStale, "source"}, {ErrNotConfigured, "configuration"}, {errors.New("answer-secret"), "database"}} {
		if got := workerErrorClass(v.e); got != v.want {
			t.Fatal(got, v.want)
		}
	}
}
