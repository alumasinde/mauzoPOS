package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// Serve runs the HTTP server until ctx is cancelled, then shuts down gracefully.
func Serve(ctx context.Context, c *Container, handler http.Handler) error {
	srv := &http.Server{
		Addr:              c.Config.HTTP.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       c.Config.HTTP.ReadTimeout,
		WriteTimeout:      c.Config.HTTP.WriteTimeout, // SSE routes will opt out via ResponseController
		IdleTimeout:       120 * time.Second,
	}

	go cleanupLoop(ctx, c)

	errCh := make(chan error, 1)
	go func() {
		c.Log.Info("api listening", "addr", srv.Addr, "env", c.Config.Env, "version", Version)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("http server: %w", err)
	case <-ctx.Done():
	}

	c.Log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), c.Config.HTTP.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("http shutdown: %w", err)
	}
	return <-errCh
}

func cleanupLoop(ctx context.Context, c *Container) {
	t := time.NewTicker(time.Minute)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			c.RateLimiter.Cleanup(10 * time.Minute)
		}
	}
}
