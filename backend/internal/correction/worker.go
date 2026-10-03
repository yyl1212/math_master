package correction

import (
	"context"
	"errors"
	"fmt"
)

// This internal distinction keeps a never-enabled database idle without hiding
// damage to an enabled schema. HTTP still sees the existing not-configured error.
var ErrNeverEnabled = fmt.Errorf("%w: never enabled", ErrNotConfigured)

type WorkerRepository interface {
	BackfillCorrections(context.Context, int) (int, error)
	ClaimCorrectionJob(context.Context) (*Lease, error)
	RenewCorrectionLease(context.Context, Lease) (Lease, error)
	ProcessCorrectionJob(context.Context, Lease, int) (Batch, error)
	FinishCorrectionJob(context.Context, Lease, JobState, string) error
}
type Options struct {
	Enabled bool
	Limit   int
}

func workerErrorClass(e error) string {
	switch {
	case errors.Is(e, context.Canceled) || errors.Is(e, context.DeadlineExceeded):
		return "deadline"
	case errors.Is(e, ErrLeaseLost):
		return "lease"
	case errors.Is(e, ErrSourceStale):
		return "source"
	case errors.Is(e, ErrNotConfigured):
		return "configuration"
	default:
		return "database"
	}
}
