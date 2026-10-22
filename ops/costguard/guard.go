// DesignDocument/12_Operation.md 12.12.2 3段の防御 / 12.12.4 コストガードの構成
package main

import (
	"context"
	"log/slog"
	"time"
)

// UsageReader returns month-to-date Cloud Run usage.
type UsageReader interface {
	MonthToDate(ctx context.Context, start, end time.Time) (Usage, error)
}

// ServiceController inspects and disables the protected API service.
type ServiceController interface {
	IsDisabled(ctx context.Context) (bool, error)
	Disable(ctx context.Context) error
}

// Guard decides whether the API must be stopped and stops it at most once.
type Guard struct {
	usage           UsageReader
	service         ServiceController
	limits          Limits
	budgetStopRatio float64
	logger          *slog.Logger
	now             func() time.Time
}

// Outcome describes what a check or a budget notification led to.
type Outcome struct {
	Exceeded        []string `json:"exceeded,omitempty"`
	Disabled        bool     `json:"disabled"`
	AlreadyDisabled bool     `json:"alreadyDisabled"`
}

// Check compares month-to-date usage with the limits (12.12.3). A failure to
// read usage never stops the API; the caller logs it and the next run retries.
func (g *Guard) Check(ctx context.Context) (Outcome, error) {
	now := g.now()
	start := billingMonthStart(now)
	usage, err := g.usage.MonthToDate(ctx, start, now)
	if err != nil {
		return Outcome{}, err
	}
	exceeded := g.limits.Exceeded(usage)
	g.logger.InfoContext(ctx, "usage checked",
		"event", "costguard_usage",
		"since", start.Format(time.RFC3339),
		"requests", usage.Requests, "limit_requests", g.limits.Requests,
		"cpu_seconds", usage.CPUSeconds, "limit_cpu_seconds", g.limits.CPUSeconds,
		"memory_gib_seconds", usage.MemoryGiBSeconds, "limit_memory_gib_seconds", g.limits.MemoryGiBSeconds,
		"exceeded", exceeded)
	if len(exceeded) == 0 {
		return Outcome{}, nil
	}
	outcome, err := g.disable(ctx, "usage", "exceeded", exceeded,
		"requests", usage.Requests, "cpu_seconds", usage.CPUSeconds, "memory_gib_seconds", usage.MemoryGiBSeconds)
	outcome.Exceeded = exceeded
	return outcome, err
}

// HandleBudget stops the API when the reported cost reaches the budget times
// the stop ratio (12.12.3). Notifications without a budget amount are ignored.
func (g *Guard) HandleBudget(ctx context.Context, n BudgetNotification) (Outcome, error) {
	g.logger.InfoContext(ctx, "budget notification",
		"event", "costguard_budget",
		"budget", n.BudgetDisplayName, "cost", n.CostAmount, "budget_amount", n.BudgetAmount,
		"currency", n.CurrencyCode, "threshold_exceeded", n.AlertThresholdExceeded)
	if n.BudgetAmount <= 0 || n.CostAmount < n.BudgetAmount*g.budgetStopRatio {
		return Outcome{}, nil
	}
	return g.disable(ctx, "budget",
		"budget", n.BudgetDisplayName, "cost", n.CostAmount, "budget_amount", n.BudgetAmount, "currency", n.CurrencyCode)
}

// disable is idempotent: an already disabled service is left untouched.
func (g *Guard) disable(ctx context.Context, reason string, attrs ...any) (Outcome, error) {
	disabled, err := g.service.IsDisabled(ctx)
	if err != nil {
		return Outcome{}, err
	}
	if disabled {
		g.logger.InfoContext(ctx, "service already disabled", "event", "costguard_already_disabled", "reason", reason)
		return Outcome{AlreadyDisabled: true}, nil
	}
	if err := g.service.Disable(ctx); err != nil {
		return Outcome{}, err
	}
	g.logger.ErrorContext(ctx, "cost guard disabled the service",
		append([]any{"event", "costguard_disabled_service", "reason", reason}, attrs...)...)
	return Outcome{Disabled: true}, nil
}
