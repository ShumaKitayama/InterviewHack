package main

import (
	"slices"
	"strings"
	"testing"
)

func setValidEnv(t *testing.T) {
	t.Helper()
	t.Setenv("TARGET_PROJECT", "interviewhack-prod")
	t.Setenv("TARGET_REGION", "asia-northeast1")
	t.Setenv("TARGET_SERVICE", "interviewhack-api")
	t.Setenv("MONITORED_PROJECTS", "")
	t.Setenv("LIMIT_REQUESTS", "1800000")
	t.Setenv("LIMIT_CPU_SECONDS", "162000")
	t.Setenv("LIMIT_MEMORY_GIB_SECONDS", "324000")
	t.Setenv("BUDGET_STOP_RATIO", "")
}

func TestLoadConfig(t *testing.T) {
	setValidEnv(t)
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if !slices.Equal(cfg.MonitoredProjects, []string{"interviewhack-prod"}) {
		t.Fatalf("MonitoredProjects = %v, want the target project", cfg.MonitoredProjects)
	}
	if cfg.Limits != (Limits{Requests: 1800000, CPUSeconds: 162000, MemoryGiBSeconds: 324000}) {
		t.Fatalf("Limits = %+v", cfg.Limits)
	}
	if cfg.BudgetStopRatio != 1.0 || cfg.Port != "8080" {
		t.Fatalf("defaults: ratio %v port %q", cfg.BudgetStopRatio, cfg.Port)
	}
}

func TestLoadConfigMonitoredProjects(t *testing.T) {
	setValidEnv(t)
	t.Setenv("MONITORED_PROJECTS", " a, ,b ")
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if !slices.Equal(cfg.MonitoredProjects, []string{"a", "b"}) {
		t.Fatalf("MonitoredProjects = %v", cfg.MonitoredProjects)
	}
}

func TestLoadConfigErrors(t *testing.T) {
	tests := []struct {
		name, key, value, want string
	}{
		{"missing target", "TARGET_SERVICE", "", "TARGET_SERVICE"},
		{"missing limit", "LIMIT_REQUESTS", "", "LIMIT_REQUESTS is required"},
		{"zero limit", "LIMIT_CPU_SECONDS", "0", "LIMIT_CPU_SECONDS must be a positive number"},
		{"not a number", "LIMIT_MEMORY_GIB_SECONDS", "lots", "LIMIT_MEMORY_GIB_SECONDS must be a positive number"},
		{"negative ratio", "BUDGET_STOP_RATIO", "-1", "BUDGET_STOP_RATIO must be a positive number"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setValidEnv(t)
			t.Setenv(tt.key, tt.value)
			_, err := LoadConfig()
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("LoadConfig() error = %v, want %q", err, tt.want)
			}
		})
	}
}
