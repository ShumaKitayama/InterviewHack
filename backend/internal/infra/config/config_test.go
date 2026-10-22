// DesignDocument/14_CICDDesign.md 14.8.1 / 12_Operation.md 12.11
package config

import "testing"

func TestLoad(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("FIREBASE_PROJECT_ID", "ih-dev")
	t.Setenv("SHARE_BASE_URL", "https://api.example.com/")
	t.Setenv("FEATURE_SHARE_VIEW", "")
	t.Setenv("FEATURE_SHARE_CREATE", "OFF")
	t.Setenv("FEATURE_COMMIT_DISABLED_USE_CASES", "qa_generate, es_refine,")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.FeatureShareView || cfg.FeatureShareCreate {
		t.Fatalf("flags: view=%v create=%v (unset means on, OFF means off)", cfg.FeatureShareView, cfg.FeatureShareCreate)
	}
	if len(cfg.FeatureCommitDisabledUseCases) != 2 || cfg.FeatureCommitDisabledUseCases[1] != "es_refine" {
		t.Fatalf("disabled use cases = %v", cfg.FeatureCommitDisabledUseCases)
	}
	if cfg.ShareBaseURL != "https://api.example.com" || cfg.AppEnv != "dev" || cfg.Port != "8080" {
		t.Fatalf("cfg = %+v", cfg)
	}
}

func TestLoadRequired(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("FIREBASE_PROJECT_ID", "ih-dev")
	if _, err := Load(); err == nil {
		t.Fatal("DATABASE_URL is required")
	}
	t.Setenv("DATABASE_URL", "postgres://x")
	t.Setenv("FIREBASE_PROJECT_ID", "")
	if _, err := Load(); err == nil {
		t.Fatal("FIREBASE_PROJECT_ID is required")
	}
	t.Setenv("FIREBASE_PROJECT_ID", "ih-dev")
	t.Setenv("APP_ENV", "production")
	if _, err := Load(); err == nil {
		t.Fatal("APP_ENV must be dev/stg/prod")
	}
}
