// DesignDocument/14_CICDDesign.md 14.8.3 コストガードの環境変数
package main

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config is loaded from environment variables only; the guard holds no secrets.
type Config struct {
	Port              string
	LogLevel          string
	Release           string
	TargetProject     string
	TargetRegion      string
	TargetService     string
	MonitoredProjects []string
	Limits            Limits
	BudgetStopRatio   float64
}

// LoadConfig reads and validates the configuration.
func LoadConfig() (Config, error) {
	cfg := Config{
		Port:          cmp.Or(os.Getenv("PORT"), "8080"),
		LogLevel:      cmp.Or(os.Getenv("LOG_LEVEL"), "info"),
		Release:       cmp.Or(os.Getenv("RELEASE"), "dev"),
		TargetProject: strings.TrimSpace(os.Getenv("TARGET_PROJECT")),
		TargetRegion:  strings.TrimSpace(os.Getenv("TARGET_REGION")),
		TargetService: strings.TrimSpace(os.Getenv("TARGET_SERVICE")),
	}
	var errs []error
	if cfg.TargetProject == "" || cfg.TargetRegion == "" || cfg.TargetService == "" {
		errs = append(errs, errors.New("TARGET_PROJECT, TARGET_REGION and TARGET_SERVICE are required"))
	}

	for project := range strings.SplitSeq(os.Getenv("MONITORED_PROJECTS"), ",") {
		if project = strings.TrimSpace(project); project != "" {
			cfg.MonitoredProjects = append(cfg.MonitoredProjects, project)
		}
	}
	if len(cfg.MonitoredProjects) == 0 && cfg.TargetProject != "" {
		cfg.MonitoredProjects = []string{cfg.TargetProject}
	}

	var err error
	if cfg.Limits.Requests, err = positiveFloat("LIMIT_REQUESTS", ""); err != nil {
		errs = append(errs, err)
	}
	if cfg.Limits.CPUSeconds, err = positiveFloat("LIMIT_CPU_SECONDS", ""); err != nil {
		errs = append(errs, err)
	}
	if cfg.Limits.MemoryGiBSeconds, err = positiveFloat("LIMIT_MEMORY_GIB_SECONDS", ""); err != nil {
		errs = append(errs, err)
	}
	if cfg.BudgetStopRatio, err = positiveFloat("BUDGET_STOP_RATIO", "1.0"); err != nil {
		errs = append(errs, err)
	}
	return cfg, errors.Join(errs...)
}

// positiveFloat parses a required (fallback == "") or defaulted env var that
// must be a positive number; a zero limit would stop the API immediately.
func positiveFloat(name, fallback string) (float64, error) {
	raw := cmp.Or(strings.TrimSpace(os.Getenv(name)), fallback)
	if raw == "" {
		return 0, fmt.Errorf("%s is required", name)
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil || v <= 0 {
		return 0, fmt.Errorf("%s must be a positive number, got %q", name, raw)
	}
	return v, nil
}
