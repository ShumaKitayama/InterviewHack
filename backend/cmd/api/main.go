// DesignDocument/7_BackendDesign.md 7.2.3 cmd/api - サーバーブートストラップ
package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/interviewhack/backend/internal/app"
	"github.com/interviewhack/backend/internal/infra/config"
	"github.com/interviewhack/backend/internal/infra/firebase"
	"github.com/interviewhack/backend/internal/infra/logging"
	"github.com/interviewhack/backend/internal/infra/postgres"
	"github.com/interviewhack/backend/internal/infra/system"
)

// release is set at build time with -ldflags "-X main.release=<git sha>" (14.9).
var release = "dev"

func main() {
	// Local development: load backend/.env (run from the backend directory).
	_ = godotenv.Load(".env")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	if cfg.Release == "dev" {
		cfg.Release = release
	}
	logging.Setup(cfg.Release, cfg.LogLevel)
	slog.Info("starting server", "app_env", cfg.AppEnv, "port", cfg.Port)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	db, err := postgres.Open(ctx, cfg.DatabaseURL)
	cancel()
	if err != nil {
		log.Fatalf("failed to connect database: %v", err)
	}
	defer db.Close()

	r := app.NewRouter(app.Options{
		DB:                     db,
		Verifier:               firebase.NewVerifier(cfg.FirebaseProjectID),
		Tokens:                 system.TokenGenerator{},
		Clock:                  system.Clock{},
		IDs:                    system.IDGenerator{},
		ShareBaseURL:           cfg.ShareBaseURL,
		ShareViewEnabled:       cfg.FeatureShareView,
		ShareCreateEnabled:     cfg.FeatureShareCreate,
		CommitDisabledUseCases: cfg.FeatureCommitDisabledUseCases,
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		slog.Info("server listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	slog.Info("shutting down server")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown error: %v", err)
	}
}
