// DesignDocument/14_CICDDesign.md 14.8.1 / 12_Operation.md 12.11 / DirectionChange_20261022.md 2.10.6
package config

import (
	"fmt"
	"os"
	"strings"
)

// Config is the backend configuration from environment variables.
type Config struct {
	AppEnv            string
	Port              string
	Release           string
	LogLevel          string
	DatabaseURL       string
	FirebaseProjectID string
	ShareBaseURL      string
	// Feature flags (12.11).
	FeatureShareView              bool
	FeatureShareCreate            bool
	FeatureCommitDisabledUseCases []string
}

// Load reads the configuration.
func Load() (*Config, error) {
	cfg := &Config{
		AppEnv:   getEnv("APP_ENV", "dev"),
		Port:     getEnv("PORT", "8080"),
		Release:  getEnv("RELEASE", "dev"),
		LogLevel: getEnv("LOG_LEVEL", "info"),
		// Secret Manager はファイル末尾の改行をそのまま渡す。pgx はその改行を channel_binding の値に含めて拒否する。
		DatabaseURL:                   strings.TrimSpace(os.Getenv("DATABASE_URL")),
		FirebaseProjectID:             os.Getenv("FIREBASE_PROJECT_ID"),
		ShareBaseURL:                  strings.TrimRight(getEnv("SHARE_BASE_URL", "http://localhost:8080"), "/"),
		FeatureShareView:              flagOn(os.Getenv("FEATURE_SHARE_VIEW")),
		FeatureShareCreate:            flagOn(os.Getenv("FEATURE_SHARE_CREATE")),
		FeatureCommitDisabledUseCases: splitList(os.Getenv("FEATURE_COMMIT_DISABLED_USE_CASES")),
	}
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.FirebaseProjectID == "" {
		return nil, fmt.Errorf("FIREBASE_PROJECT_ID is required")
	}
	if cfg.AppEnv != "dev" && cfg.AppEnv != "prod" && cfg.AppEnv != "stg" {
		return nil, fmt.Errorf("APP_ENV must be dev, stg or prod")
	}
	return cfg, nil
}

// flagOn treats an unset flag as on (default "on", 12.11).
func flagOn(v string) bool {
	return strings.ToLower(strings.TrimSpace(v)) != "off"
}

func splitList(v string) []string {
	var out []string
	for _, s := range strings.Split(v, ",") {
		if s = strings.TrimSpace(s); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
