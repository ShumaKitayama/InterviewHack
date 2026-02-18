// DesignDocument/14_CICDDesign.md 14.8 環境変数の注入方針
package config

import (
	"fmt"
	"os"
)

type Config struct {
	Env                 string
	Port                string
	Release             string
	DatabaseURL         string
	SupabaseJWTSecret   string
	StripeSecretKey     string
	StripeWebhookSecret string
	OpenAIAPIKey        string
	GeminiAPIKey        string
	AllowOrigins        []string
	CheckoutSuccessURL  string
	CheckoutCancelURL   string
}

func Load() (*Config, error) {
	cfg := &Config{
		Env:                 getEnv("ENV", "dev"),
		Port:                getEnv("PORT", "8080"),
		Release:             getEnv("RELEASE", "dev"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		SupabaseJWTSecret:   os.Getenv("SUPABASE_JWT_SECRET"),
		StripeSecretKey:     os.Getenv("STRIPE_SECRET_KEY"),
		StripeWebhookSecret: os.Getenv("STRIPE_WEBHOOK_SECRET"),
		OpenAIAPIKey:        os.Getenv("OPENAI_API_KEY"),
		GeminiAPIKey:        os.Getenv("GEMINI_API_KEY"),
		AllowOrigins:        []string{getEnv("ALLOW_ORIGIN", "http://localhost:5173")},
		CheckoutSuccessURL:  getEnv("CHECKOUT_SUCCESS_URL", "http://localhost:5173/billing?status=success"),
		CheckoutCancelURL:   getEnv("CHECKOUT_CANCEL_URL", "http://localhost:5173/billing?status=cancel"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
