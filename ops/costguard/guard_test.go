package main

import (
	"context"
	"errors"
	"log/slog"
	"slices"
	"testing"
	"time"
)

type fakeUsage struct {
	usage      Usage
	err        error
	start, end time.Time
}

func (f *fakeUsage) MonthToDate(_ context.Context, start, end time.Time) (Usage, error) {
	f.start, f.end = start, end
	return f.usage, f.err
}

type fakeService struct {
	disabled     bool
	inspectErr   error
	disableErr   error
	disableCalls int
}

func (f *fakeService) IsDisabled(context.Context) (bool, error) { return f.disabled, f.inspectErr }

func (f *fakeService) Disable(context.Context) error {
	f.disableCalls++
	if f.disableErr != nil {
		return f.disableErr
	}
	f.disabled = true
	return nil
}

var testNow = time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)

func newTestGuard(usage UsageReader, service ServiceController) *Guard {
	return &Guard{
		usage:           usage,
		service:         service,
		limits:          Limits{Requests: 1000, CPUSeconds: 100, MemoryGiBSeconds: 200},
		budgetStopRatio: 1.0,
		logger:          slog.New(slog.DiscardHandler),
		now:             func() time.Time { return testNow },
	}
}

func TestGuardCheck(t *testing.T) {
	usageErr := errors.New("monitoring unavailable")
	tests := []struct {
		name         string
		usage        fakeUsage
		service      fakeService
		want         Outcome
		wantErr      bool
		wantDisables int
	}{
		{name: "below limits", usage: fakeUsage{usage: Usage{Requests: 999, CPUSeconds: 99, MemoryGiBSeconds: 199}}},
		{name: "cpu reached stops the API", usage: fakeUsage{usage: Usage{CPUSeconds: 100}},
			want: Outcome{Exceeded: []string{"cpu_seconds"}, Disabled: true}, wantDisables: 1},
		{name: "already disabled is left alone", usage: fakeUsage{usage: Usage{Requests: 5000}}, service: fakeService{disabled: true},
			want: Outcome{Exceeded: []string{"requests"}, AlreadyDisabled: true}},
		{name: "usage error never stops", usage: fakeUsage{err: usageErr, usage: Usage{Requests: 1e9}}, wantErr: true},
		{name: "inspect error", usage: fakeUsage{usage: Usage{Requests: 5000}}, service: fakeService{inspectErr: errors.New("403")},
			want: Outcome{Exceeded: []string{"requests"}}, wantErr: true},
		{name: "disable error", usage: fakeUsage{usage: Usage{Requests: 5000}}, service: fakeService{disableErr: errors.New("500")},
			want: Outcome{Exceeded: []string{"requests"}}, wantErr: true, wantDisables: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			guard := newTestGuard(&tt.usage, &tt.service)
			got, err := guard.Check(t.Context())
			if (err != nil) != tt.wantErr {
				t.Fatalf("Check() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got.Disabled != tt.want.Disabled || got.AlreadyDisabled != tt.want.AlreadyDisabled || !slices.Equal(got.Exceeded, tt.want.Exceeded) {
				t.Fatalf("Check() = %+v, want %+v", got, tt.want)
			}
			if tt.service.disableCalls != tt.wantDisables {
				t.Fatalf("Disable calls = %d, want %d", tt.service.disableCalls, tt.wantDisables)
			}
		})
	}
}

func TestGuardCheckUsesBillingMonth(t *testing.T) {
	usage := &fakeUsage{}
	if _, err := newTestGuard(usage, &fakeService{}).Check(t.Context()); err != nil {
		t.Fatalf("Check() error = %v", err)
	}
	if !usage.start.Equal(billingMonthStart(testNow)) || !usage.end.Equal(testNow) {
		t.Fatalf("interval = %s - %s", usage.start, usage.end)
	}
}

func TestGuardCheckTwiceDisablesOnce(t *testing.T) {
	service := &fakeService{}
	guard := newTestGuard(&fakeUsage{usage: Usage{Requests: 5000}}, service)
	for range 2 {
		if _, err := guard.Check(t.Context()); err != nil {
			t.Fatalf("Check() error = %v", err)
		}
	}
	if service.disableCalls != 1 {
		t.Fatalf("Disable calls = %d, want 1", service.disableCalls)
	}
}

func TestGuardHandleBudget(t *testing.T) {
	tests := []struct {
		name         string
		ratio        float64
		notification BudgetNotification
		wantDisabled bool
	}{
		{"below budget", 1.0, BudgetNotification{CostAmount: 299, BudgetAmount: 300}, false},
		{"budget reached", 1.0, BudgetNotification{CostAmount: 300, BudgetAmount: 300}, true},
		{"lower stop ratio", 0.9, BudgetNotification{CostAmount: 270, BudgetAmount: 300}, true},
		{"no budget amount is ignored", 1.0, BudgetNotification{CostAmount: 100}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &fakeService{}
			guard := newTestGuard(&fakeUsage{}, service)
			guard.budgetStopRatio = tt.ratio
			got, err := guard.HandleBudget(t.Context(), tt.notification)
			if err != nil {
				t.Fatalf("HandleBudget() error = %v", err)
			}
			if got.Disabled != tt.wantDisabled || service.disabled != tt.wantDisabled {
				t.Fatalf("HandleBudget() = %+v (service disabled %v), want disabled %v", got, service.disabled, tt.wantDisabled)
			}
		})
	}
}
