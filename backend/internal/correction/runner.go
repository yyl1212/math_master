package correction

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/yyl1212/math_master/backend/internal/auth"
	"github.com/yyl1212/math_master/backend/internal/question"
)

type Runner struct {
	repo                                                                        WorkerRepository
	options                                                                     Options
	lane                                                                        chan struct{}
	batchBudget, statementBudget, renewInterval, backfillInterval, pollInterval time.Duration
}

func NewRunner(repo WorkerRepository, o Options) (*Runner, error) {
	if repo == nil || o.Limit < 0 || o.Limit > 50 {
		return nil, auth.ErrInvalidInput
	}
	if o.Limit == 0 {
		o.Limit = 50
	}
	return &Runner{repo: repo, options: o, lane: make(chan struct{}, 1), batchBudget: 30 * time.Second, statementBudget: 8 * time.Second, renewInterval: 10 * time.Second, backfillInterval: time.Minute, pollInterval: time.Second}, nil
}
func (r *Runner) RunOnce(ctx context.Context) (Batch, error) { return r.runBatch(ctx, true) }

type workerOutcome struct {
	batch Batch
	err   error
}

func (r *Runner) runBatch(parent context.Context, backfill bool) (Batch, error) {
	ctx, cancel := context.WithTimeout(parent, r.batchBudget)
	defer cancel()
	select {
	case r.lane <- struct{}{}:
		defer func() { <-r.lane }()
	case <-ctx.Done():
		return Batch{}, ctx.Err()
	}
	if backfill {
		c, stop := context.WithTimeout(ctx, r.statementBudget)
		_, e := r.repo.BackfillCorrections(c, 50)
		stop()
		if e != nil {
			return Batch{}, e
		}
	}
	c, stop := context.WithTimeout(ctx, r.statementBudget)
	lease, e := r.repo.ClaimCorrectionJob(c)
	stop()
	if e != nil {
		return Batch{}, e
	}
	if lease == nil {
		return Batch{}, nil
	}
	bodyCtx, stopBody := context.WithCancel(ctx)
	defer stopBody()
	done := make(chan workerOutcome, 1)
	// One body transaction and one renewal transaction are the only concurrent
	// repository calls. Returning always waits for the body to finish or roll back.
	go func() {
		b, e := r.repo.ProcessCorrectionJob(bodyCtx, *lease, r.options.Limit)
		done <- workerOutcome{b, e}
	}()
	renew := time.NewTicker(r.renewInterval)
	defer renew.Stop()
	finish := func(v workerOutcome, cause error) (Batch, error) {
		if cause == nil {
			cause = v.err
		}
		if cause == nil {
			return v.batch, nil
		}
		class := workerErrorClass(cause)
		if ctx.Err() == nil && !errors.Is(cause, ErrLeaseLost) && !errors.Is(cause, ErrNotConfigured) {
			c, stop := context.WithTimeout(ctx, r.statementBudget)
			e := r.repo.FinishCorrectionJob(c, *lease, RetryWait, class)
			stop()
			if e != nil && !errors.Is(e, ErrLeaseLost) {
				cause = errors.Join(cause, e)
			}
		}
		id := "unavailable"
		if question.ValidID(lease.JobID) {
			id = lease.JobID
		}
		log.Printf("correction jobID=%s errorClass=%s", id, class)
		return v.batch, cause
	}
	for {
		select {
		case v := <-done:
			stopBody()
			return finish(v, nil)
		case <-ctx.Done():
			stopBody()
			v := <-done
			return finish(v, ctx.Err())
		case <-renew.C:
			select {
			case v := <-done:
				stopBody()
				return finish(v, nil)
			default:
			}
			c, stop := context.WithTimeout(ctx, r.statementBudget)
			_, e := r.repo.RenewCorrectionLease(c, *lease)
			stop()
			if e != nil {
				stopBody()
				v := <-done
				if v.err == nil && (v.batch.State == Succeeded || v.batch.State == Queued) {
					return v.batch, nil
				}
				return finish(v, e)
			}
		}
	}
}
func workerWait(ctx context.Context, d time.Duration) bool {
	if d < 0 {
		d = 0
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
func (r *Runner) Run(ctx context.Context) error {
	if !r.options.Enabled {
		<-ctx.Done()
		return nil
	}
	nextBackfill := time.Time{}
	legacy := false
	for {
		if ctx.Err() != nil {
			return nil
		}
		now := time.Now()
		due := nextBackfill.IsZero() || !now.Before(nextBackfill)
		if due {
			nextBackfill = now.Add(r.backfillInterval)
		}
		v, e := r.runBatch(ctx, due)
		if ctx.Err() != nil {
			return nil
		}
		if errors.Is(e, ErrNeverEnabled) {
			legacy = true
		} else if errors.Is(e, ErrNotConfigured) {
			return e
		} else {
			legacy = false
		}
		if legacy {
			if !workerWait(ctx, time.Until(nextBackfill)) {
				return nil
			}
			continue
		}
		if e != nil || !v.Claimed {
			if !workerWait(ctx, min(r.pollInterval, time.Until(nextBackfill))) {
				return nil
			}
		}
	}
}
