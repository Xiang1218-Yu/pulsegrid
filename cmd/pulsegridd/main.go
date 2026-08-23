package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"pulsegrid/internal/analytics"
	"pulsegrid/internal/events"
	"pulsegrid/internal/httpapi"
	"pulsegrid/internal/jobs"
	"pulsegrid/internal/service"
	"pulsegrid/internal/store"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	repository := store.NewMemory()
	bus := events.New(events.Config{Buffer: 256, DropWhenBusy: true})
	workers := jobs.New(jobs.Config{Workers: 4, Capacity: 128, Logger: logger})
	metrics := analytics.New()
	app := service.New(service.Config{
		Repository: repository,
		Events:     bus,
		Jobs:       workers,
		Metrics:    metrics,
		Logger:     logger,
	})
	workers.Register("recalculate", app.HandleRecalculate)
	workers.Register("deliver-message", app.HandleDelivery)
	workers.Start()
	app.Seed(context.Background())

	server := httpapi.New(httpapi.Config{
		App: app, Events: bus, Jobs: workers, Metrics: metrics, Logger: logger,
	})
	httpServer := &http.Server{
		Addr:              envOr("PULSEGRID_ADDR", ":8091"),
		Handler:           server.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       45 * time.Second,
	}
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	go func() {
		<-ctx.Done()
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		_ = httpServer.Shutdown(shutdownCtx)
		workers.Stop()
		bus.Close()
	}()
	logger.Info("pulsegrid listening", "addr", httpServer.Addr)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
