package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

const (
	defaultTeamsAPIURL = "http://teams-api-service:80"
	defaultPollSeconds = 30
	healthPort         = 8081
)

type config struct {
	TeamsAPIURL  string
	PollInterval time.Duration
	LogLevel     slog.Level
}

func parseConfig() config {
	pollSecs, err := strconv.Atoi(os.Getenv("POLL_INTERVAL"))
	if err != nil || pollSecs < 1 {
		pollSecs = defaultPollSeconds
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(os.Getenv("LOG_LEVEL"))); err != nil {
		level = slog.LevelInfo
	}

	url := os.Getenv("TEAMS_API_URL")
	if url == "" {
		url = defaultTeamsAPIURL
	}

	return config{
		TeamsAPIURL:  url,
		PollInterval: time.Duration(pollSecs) * time.Second,
		LogLevel:     level,
	}
}

func run(ctx context.Context) error {
	cfg := parseConfig()
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel})))
	slog.Info("starting teams operator", "api_url", cfg.TeamsAPIURL, "poll_interval", cfg.PollInterval)

	op, err := NewOperator(cfg.TeamsAPIURL, cfg.PollInterval)
	if err != nil {
		return fmt.Errorf("init operator: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "ok")
	})
	srv := &http.Server{Addr: fmt.Sprintf(":%d", healthPort), Handler: mux}

	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("health server error", "error", err)
		}
	}()
	defer srv.Shutdown(context.Background())

	go op.Run(ctx)

	<-ctx.Done()
	slog.Info("shutting down")
	return nil
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := run(ctx); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}