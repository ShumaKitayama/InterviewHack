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

	router "github.com/interviewhack/backend/internal/adapter/http"
	"github.com/interviewhack/backend/internal/infra/config"
	"github.com/interviewhack/backend/internal/infra/logging"
	"github.com/joho/godotenv"
)

func main() {
	// ローカル開発時: backend/.env を読み込む（Go は .env を自動読まない）
	// backend ディレクトリで go run ./cmd/api を実行すること
	_ = godotenv.Load(".env")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	logging.Setup(cfg.Release)
	slog.Info("starting server", "env", cfg.Env, "port", cfg.Port, "release", cfg.Release)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	app, err := initApp(ctx, cfg)
	if err != nil {
		log.Fatalf("failed to initialize app: %v", err)
	}
	defer app.DB.Close()

	r := router.NewRouter(app.Router)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
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

	slog.Info("shutting down server...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("server shutdown error: %v", err)
	}
	slog.Info("server stopped")
}
