// DesignDocument/12_Operation.md 12.12.4 エンドポイント（POST /check, POST /budget）
package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

const (
	maxBodyBytes   = 64 << 10
	requestTimeout = 60 * time.Second
)

// newHandler routes the two endpoints. Authentication is enforced in front of
// the container by Cloud Run IAM (only costguard-invoker may call it).
func newHandler(guard *Guard, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /check", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
		defer cancel()
		outcome, err := guard.Check(ctx)
		if err != nil {
			logger.ErrorContext(ctx, "usage check failed", "event", "costguard_check_failed", "error", err.Error())
			http.Error(w, "check failed", http.StatusInternalServerError)
			return
		}
		writeJSON(w, outcome)
	})
	mux.HandleFunc("POST /budget", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
		defer cancel()
		notification, err := parseBudgetPush(http.MaxBytesReader(w, r.Body, maxBodyBytes))
		if err != nil {
			// Acknowledge so Pub/Sub does not redeliver a message that can never be parsed.
			logger.ErrorContext(ctx, "invalid budget notification", "event", "costguard_budget_invalid", "error", err.Error())
			writeJSON(w, map[string]string{"status": "ignored"})
			return
		}
		outcome, err := guard.HandleBudget(ctx, notification)
		if err != nil {
			// A non-2xx response makes Pub/Sub retry the stop.
			logger.ErrorContext(ctx, "budget stop failed", "event", "costguard_budget_failed", "error", err.Error())
			http.Error(w, "budget handling failed", http.StatusInternalServerError)
			return
		}
		writeJSON(w, outcome)
	})
	return mux
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}
