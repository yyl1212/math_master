package correction

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type runnerRepo struct {
	backfill  func(context.Context, int) (int, error)
	claim     func(context.Context) (*Lease, error)
	process   func(context.Context, Lease, int) (Batch, error)
	renew     func(context.Context, Lease) (Lease, error)
	finish    func(context.Context, Lease, JobState, string) error
	backfills atomic.Int32
	claims    atomic.Int32
	renews    atomic.Int32
	finishes  atomic.Int32
	active    atomic.Int32
	peak      atomic.Int32
}

func (f *runnerRepo) enter() func() {
	n := f.active.Add(1)
	for p := f.peak.Load(); n > p && !f.peak.CompareAndSwap(p, n); p = f.peak.Load() {
	}
	return func() { f.active.Add(-1) }
}
func (f *runnerRepo) BackfillCorrections(c context.Context, n int) (int, error) {
	defer f.enter()()
	f.backfills.Add(1)
	if f.backfill != nil {
		return f.backfill(c, n)
	}
	return 0, nil
}
func (f *runnerRepo) ClaimCorrectionJob(c context.Context) (*Lease, error) {
	defer f.enter()()
	f.claims.Add(1)
	if f.claim != nil {
		return f.claim(c)
	}
	return nil, nil
}
func (f *runnerRepo) ProcessCorrectionJob(c context.Context, l Lease, n int) (Batch, error) {
	defer f.enter()()
	if f.process != nil {
		return f.process(c, l, n)
	}
	return Batch{Claimed: true, State: Succeeded}, nil
}
func (f *runnerRepo) RenewCorrectionLease(c context.Context, l Lease) (Lease, error) {
	defer f.enter()()
	f.renews.Add(1)
	if f.renew != nil {
		return f.renew(c, l)
	}
	l.Until = time.Now().Add(30 * time.Second)
	return l, nil
}
func (f *runnerRepo) FinishCorrectionJob(c context.Context, l Lease, s JobState, k string) error {
	defer f.enter()()
	f.finishes.Add(1)
	if f.finish != nil {
		return f.finish(c, l, s, k)
	}
	return nil
}
func runnerLease() *Lease {
	return &Lease{JobID: "11111111-1111-4111-8111-111111111111", Token: 1, Epoch: 1, Attempt: 1, Until: time.Now().Add(30 * time.Second)}
}
func newTestRunner(t *testing.T, f *runnerRepo) *Runner {
	t.Helper()
	r, e := NewRunner(f, Options{Enabled: true, Limit: 50})
	if e != nil {
		t.Fatal(e)
	}
	return r
}
func TestCorrectionRunnerBudget(t *testing.T) {
	f := &runnerRepo{}
	began := time.Now()
	bound := func(c context.Context, max time.Duration) {
		t.Helper()
		d, ok := c.Deadline()
		if !ok || time.Until(d) > max || d.After(began.Add(30*time.Second+time.Millisecond)) {
			t.Fatal("budget reset or unbounded repository call")
		}
	}
	f.backfill = func(c context.Context, n int) (int, error) {
		bound(c, 8*time.Second)
		if n != 50 {
			t.Fatal(n)
		}
		return 0, nil
	}
	f.claim = func(c context.Context) (*Lease, error) { bound(c, 8*time.Second); return runnerLease(), nil }
	f.process = func(c context.Context, l Lease, n int) (Batch, error) {
		bound(c, 30*time.Second)
		return Batch{Claimed: true, Processed: n, State: Succeeded}, nil
	}
	r := newTestRunner(t, f)
	v, e := r.RunOnce(context.Background())
	if e != nil || v.Processed != 50 || f.peak.Load() > 2 {
		t.Fatal(v, e, f.peak.Load())
	}
}
func TestCorrectionRunnerDeadlineIncludesMetadata(t *testing.T) {
	f := &runnerRepo{}
	f.backfill = func(c context.Context, n int) (int, error) { time.Sleep(20 * time.Millisecond); return 0, nil }
	f.claim = func(c context.Context) (*Lease, error) { return runnerLease(), nil }
	f.process = func(c context.Context, l Lease, n int) (Batch, error) {
		<-c.Done()
		return Batch{Claimed: true, Processed: 3}, c.Err()
	}
	r := newTestRunner(t, f)
	r.batchBudget = 40 * time.Millisecond
	start := time.Now()
	v, e := r.RunOnce(context.Background())
	if !errors.Is(e, context.DeadlineExceeded) || v.Processed != 3 || time.Since(start) > time.Second {
		t.Fatal("batch budget restarted", v, e, time.Since(start))
	}
}
func TestCorrectionRunnerRenewAndConnectionBudget(t *testing.T) {
	f := &runnerRepo{}
	f.claim = func(context.Context) (*Lease, error) { return runnerLease(), nil }
	f.process = func(c context.Context, l Lease, n int) (Batch, error) {
		for f.renews.Load() == 0 {
			select {
			case <-c.Done():
				return Batch{}, c.Err()
			case <-time.After(time.Millisecond):
			}
		}
		return Batch{Claimed: true, State: Succeeded}, nil
	}
	r := newTestRunner(t, f)
	r.renewInterval = 2 * time.Millisecond
	if _, e := r.RunOnce(context.Background()); e != nil || f.renews.Load() < 1 || f.peak.Load() > 2 || f.active.Load() != 0 {
		t.Fatal(e, f.renews.Load(), f.peak.Load(), f.active.Load())
	}
}
func TestCorrectionRunnerCancelDrain(t *testing.T) {
	f := &runnerRepo{}
	entered, release := make(chan struct{}), make(chan struct{})
	f.claim = func(context.Context) (*Lease, error) { return runnerLease(), nil }
	f.process = func(c context.Context, l Lease, n int) (Batch, error) {
		close(entered)
		<-release
		return Batch{Claimed: true}, c.Err()
	}
	r := newTestRunner(t, f)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { _, e := r.RunOnce(ctx); done <- e }()
	<-entered
	cancel()
	select {
	case <-done:
		t.Fatal("returned while transaction was still active")
	case <-time.After(10 * time.Millisecond):
	}
	close(release)
	if e := <-done; !errors.Is(e, context.Canceled) || f.active.Load() != 0 {
		t.Fatal("did not drain", e)
	}
}
func TestCorrectionRunnerLeaseLostCancelsBody(t *testing.T) {
	f := &runnerRepo{}
	f.claim = func(context.Context) (*Lease, error) { return runnerLease(), nil }
	f.process = func(c context.Context, l Lease, n int) (Batch, error) {
		<-c.Done()
		return Batch{Claimed: true}, c.Err()
	}
	f.renew = func(context.Context, Lease) (Lease, error) { return Lease{}, ErrLeaseLost }
	r := newTestRunner(t, f)
	r.renewInterval = time.Millisecond
	if _, e := r.RunOnce(context.Background()); !errors.Is(e, ErrLeaseLost) || f.finishes.Load() != 0 || f.active.Load() != 0 {
		t.Fatal(e, f.finishes.Load())
	}
}
func TestCorrectionRunnerFailureIsClosed(t *testing.T) {
	f := &runnerRepo{}
	f.claim = func(context.Context) (*Lease, error) { return runnerLease(), nil }
	f.process = func(context.Context, Lease, int) (Batch, error) {
		return Batch{Claimed: true, Processed: 2}, errors.New("dsn-and-answer-secret")
	}
	f.finish = func(c context.Context, l Lease, s JobState, k string) error {
		if s != RetryWait || k != "database" {
			t.Fatal(s, k)
		}
		d, ok := c.Deadline()
		if !ok || time.Until(d) > 8*time.Second {
			t.Fatal("finish unbounded")
		}
		return nil
	}
	r := newTestRunner(t, f)
	v, e := r.RunOnce(context.Background())
	if e == nil || v.Processed != 2 || f.finishes.Load() != 1 {
		t.Fatal(v, e)
	}
}
func TestCorrectionRunnerBackfillTick(t *testing.T) {
	f := &runnerRepo{}
	r := newTestRunner(t, f)
	r.backfillInterval = 25 * time.Millisecond
	r.pollInterval = time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), 175*time.Millisecond)
	defer cancel()
	if e := r.Run(ctx); e != nil {
		t.Fatal(e)
	}
	if n := f.backfills.Load(); n < 2 || n > 9 {
		t.Fatal("backfill must run at startup and its interval, not every poll", n)
	}
}
func TestCorrectionRunnerDisabledAndSchemaState(t *testing.T) {
	for _, disabled := range []bool{false, true} {
		f := &runnerRepo{}
		f.backfill = func(context.Context, int) (int, error) { return 0, ErrNeverEnabled }
		r, e := NewRunner(f, Options{Enabled: !disabled, Limit: 50})
		if e != nil {
			t.Fatal(e)
		}
		r.backfillInterval = 10 * time.Millisecond
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
		if e = r.Run(ctx); e != nil {
			t.Fatal("never-enabled must idle quietly", e)
		}
		cancel()
		if disabled && (f.claims.Load() != 0 || f.backfills.Load() != 0) {
			t.Fatal("disabled worker touched repository")
		}
	}
	f := &runnerRepo{backfill: func(context.Context, int) (int, error) { return 0, ErrNotConfigured }}
	if e := newTestRunner(t, f).Run(context.Background()); !errors.Is(e, ErrNotConfigured) {
		t.Fatal("damaged enabled schema silently idled", e)
	}
}

func TestCorrectionRunnerSerialCallsIncludeQueueBudget(t *testing.T) {
	f := &runnerRepo{}
	entered, release := make(chan struct{}), make(chan struct{})
	f.claim = func(context.Context) (*Lease, error) { return runnerLease(), nil }
	f.process = func(context.Context, Lease, int) (Batch, error) {
		close(entered)
		<-release
		return Batch{Claimed: true, State: Succeeded}, nil
	}
	r := newTestRunner(t, f)
	first := make(chan error, 1)
	go func() { _, e := r.RunOnce(context.Background()); first <- e }()
	<-entered
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if _, e := r.RunOnce(ctx); !errors.Is(e, context.DeadlineExceeded) || f.claims.Load() != 1 {
		t.Fatal("second caller entered body", e, f.claims.Load())
	}
	close(release)
	if e := <-first; e != nil {
		t.Fatal(e)
	}
}
