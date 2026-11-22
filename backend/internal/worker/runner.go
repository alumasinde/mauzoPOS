// Package worker runs periodic background jobs (eTIMS retries, M-Pesa status
// queries, report snapshots, sync housekeeping). Jobs must be idempotent: in
// production several worker replicas may run, and job locking moves to the DB
// or Redis when that happens.
package worker

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Job is a periodic background task. Run must be idempotent.
type Job interface {
	Name() string
	Interval() time.Duration
	Run(ctx context.Context) error
}

// Runner executes registered jobs on their own tickers.
type Runner struct {
	log  *slog.Logger
	jobs []Job
}

// NewRunner builds a Runner for the given jobs.
func NewRunner(log *slog.Logger, jobs ...Job) *Runner {
	return &Runner{log: log, jobs: jobs}
}

// Run starts every job on its own ticker and blocks until ctx is cancelled and
// in-flight runs finish.
func (r *Runner) Run(ctx context.Context) {
	r.log.Info("worker started", "jobs", len(r.jobs))
	var wg sync.WaitGroup
	for _, j := range r.jobs {
		wg.Add(1)
		go func(j Job) {
			defer wg.Done()
			r.loop(ctx, j)
		}(j)
	}
	<-ctx.Done()
	wg.Wait()
	r.log.Info("worker stopped")
}

func (r *Runner) loop(ctx context.Context, j Job) {
	t := time.NewTicker(j.Interval())
	defer t.Stop()
	for {
		r.runOnce(ctx, j)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		}
	}
}

func (r *Runner) runOnce(ctx context.Context, j Job) {
	defer func() {
		if p := recover(); p != nil {
			r.log.Error("job panic", "job", j.Name(), "panic", p)
		}
	}()
	start := time.Now()
	if err := j.Run(ctx); err != nil {
		r.log.Error("job failed", "job", j.Name(), "err", err, "duration", time.Since(start))
		return
	}
	r.log.Debug("job ok", "job", j.Name(), "duration", time.Since(start))
}
