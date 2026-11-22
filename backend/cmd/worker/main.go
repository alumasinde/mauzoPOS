// Command worker runs background jobs.
package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"mauzo/internal/app"
	"mauzo/internal/config"
	"mauzo/internal/worker"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}
	c, err := app.NewContainer(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() { _ = c.Close() }()

	// Jobs are registered here as modules add them (Phase 9+).
	worker.NewRunner(c.Log).Run(ctx)
	return nil
}
