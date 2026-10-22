// Command costguard stops the prod API on Cloud Run when month-to-date usage
// or the budget nears its limit (DesignDocument/12_Operation.md 12.12).
package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/oauth2/google"
)

// release is set at build time with -ldflags "-X main.release=<git sha>" (14.9).
var release = "dev"

func main() {
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	if cfg.Release == "dev" {
		cfg.Release = release
	}
	logger := newLogger(cfg.Release, cfg.LogLevel)
	slog.SetDefault(logger)

	// Application Default Credentials: the Cloud Run service account in prod,
	// `gcloud auth application-default login` locally. No key files (14.8.3).
	httpClient, err := google.DefaultClient(context.Background(), "https://www.googleapis.com/auth/cloud-platform")
	if err != nil {
		log.Fatalf("failed to create Google API client: %v", err)
	}
	httpClient.Timeout = 30 * time.Second

	guard := &Guard{
		usage: &monitoringClient{httpClient: httpClient, baseURL: monitoringBaseURL, projects: cfg.MonitoredProjects},
		service: &cloudRunClient{httpClient: httpClient, baseURL: cloudRunBaseURL,
			project: cfg.TargetProject, region: cfg.TargetRegion, service: cfg.TargetService},
		limits:          cfg.Limits,
		budgetStopRatio: cfg.BudgetStopRatio,
		logger:          logger,
		now:             time.Now,
	}

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           newHandler(guard, logger),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      90 * time.Second,
	}
	go func() {
		logger.Info("starting cost guard", "target_service", cfg.TargetService,
			"monitored_projects", strings.Join(cfg.MonitoredProjects, ","), "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "error", err.Error())
	}
}

// newLogger writes JSON logs with the Cloud Logging keys used by the BFF
// (DesignDocument/7_BackendDesign.md 7.8.1).
func newLogger(release, level string) *slog.Logger {
	var lv slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lv = slog.LevelDebug
	case "warn":
		lv = slog.LevelWarn
	case "error":
		lv = slog.LevelError
	default:
		lv = slog.LevelInfo
	}
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: lv,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			switch a.Key {
			case slog.LevelKey:
				a.Key = "severity"
				if a.Value.String() == "WARN" {
					a.Value = slog.StringValue("WARNING")
				}
			case slog.TimeKey:
				a.Key = "timestamp"
			case slog.MessageKey:
				a.Key = "message"
			}
			return a
		},
	})
	return slog.New(handler).With("release", release, "service", "costguard")
}
