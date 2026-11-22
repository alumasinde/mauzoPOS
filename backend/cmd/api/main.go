// Command api runs the Mauzo POS HTTP API.
package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"mauzo/internal/app"
	"mauzo/internal/config"
	"mauzo/internal/routes"
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

	return app.Serve(ctx, c, routes.New(c))
}
