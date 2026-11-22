package worker

import (
	"context"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"
)

type countJob struct {
	n     atomic.Int32
	panic bool
}

func (c *countJob) Name() string            { return "count" }
func (c *countJob) Interval() time.Duration { return 5 * time.Millisecond }
func (c *countJob) Run(context.Context) error {
	c.n.Add(1)
	if c.panic {
		panic("boom")
	}
	return nil
}

func TestRunnerRunsJobsAndSurvivesPanics(t *testing.T) {
	ok, bad := &countJob{}, &countJob{panic: true}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	NewRunner(slog.New(slog.NewTextHandler(io.Discard, nil)), ok, bad).Run(ctx)

	if ok.n.Load() < 2 || bad.n.Load() < 2 {
		t.Fatalf("jobs did not repeat: ok=%d bad=%d", ok.n.Load(), bad.n.Load())
	}
}
