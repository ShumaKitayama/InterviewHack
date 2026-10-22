package main

import (
	"slices"
	"testing"
	"time"
)

func TestLimitsExceeded(t *testing.T) {
	limits := Limits{Requests: 100, CPUSeconds: 10, MemoryGiBSeconds: 20}
	tests := []struct {
		name  string
		usage Usage
		want  []string
	}{
		{"all below", Usage{Requests: 99, CPUSeconds: 9.9, MemoryGiBSeconds: 19.9}, nil},
		{"requests reached exactly", Usage{Requests: 100}, []string{"requests"}},
		{"cpu only", Usage{CPUSeconds: 10.5}, []string{"cpu_seconds"}},
		{"memory only", Usage{MemoryGiBSeconds: 25}, []string{"memory_gib_seconds"}},
		{"all", Usage{Requests: 1e9, CPUSeconds: 1e9, MemoryGiBSeconds: 1e9},
			[]string{"requests", "cpu_seconds", "memory_gib_seconds"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := limits.Exceeded(tt.usage); !slices.Equal(got, tt.want) {
				t.Fatalf("Exceeded() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBillingMonthStart(t *testing.T) {
	jst := time.FixedZone("JST", 9*60*60)
	tests := []struct {
		name string
		now  time.Time
		want string // RFC3339 in UTC
	}{
		// 2026-11-01 08:00 JST is still 2026-10-31 16:00 PDT: October's month.
		{"JST new month but PT previous month", time.Date(2026, 11, 1, 8, 0, 0, 0, jst), "2026-10-01T07:00:00Z"},
		// 2026-11-01 17:00 JST = 2026-11-01 00:00 PDT: November begins.
		{"PT midnight on the first", time.Date(2026, 11, 1, 17, 0, 0, 0, jst), "2026-11-01T07:00:00Z"},
		// December is in PST (UTC-8).
		{"standard time", time.Date(2026, 12, 15, 12, 0, 0, 0, time.UTC), "2026-12-01T08:00:00Z"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := billingMonthStart(tt.now).UTC().Format(time.RFC3339); got != tt.want {
				t.Fatalf("billingMonthStart() = %s, want %s", got, tt.want)
			}
		})
	}
}
