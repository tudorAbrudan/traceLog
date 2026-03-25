package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/tudorAbrudan/tracelog/internal/agent"
	"github.com/tudorAbrudan/tracelog/internal/hub"
)

func runBoth(h *hub.Hub, a *agent.Agent) error {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	errCh := make(chan error, 2)

	go func() {
		if err := h.Start(ctx); err != nil {
			errCh <- err
		}
	}()

	go func() {
		if err := a.Start(ctx); err != nil {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		slog.Info("Shutting down gracefully...")
		h.Shutdown()
		a.Shutdown()
		return nil
	case err := <-errCh:
		cancel()
		h.Shutdown()
		a.Shutdown()
		return err
	}
}

func setupLogger() {
	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})
	slog.SetDefault(slog.New(handler))
}

func init() {
	setupLogger()
}
