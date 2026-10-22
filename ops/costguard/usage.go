// DesignDocument/12_Operation.md 12.12.3 監視する指標と閾値
package main

import (
	"time"
	_ "time/tzdata" // the container image has no zoneinfo; billing months use Pacific Time
)

// Usage is the month-to-date Cloud Run consumption summed over the monitored projects.
type Usage struct {
	Requests         float64 `json:"requests"`
	CPUSeconds       float64 `json:"cpuSeconds"`
	MemoryGiBSeconds float64 `json:"memoryGibSeconds"`
}

// Limits are the stop thresholds (90% of the official free tier, given by env vars).
type Limits struct {
	Requests         float64
	CPUSeconds       float64
	MemoryGiBSeconds float64
}

// Exceeded returns the names of the metrics whose usage reached its limit.
func (l Limits) Exceeded(u Usage) []string {
	var names []string
	if u.Requests >= l.Requests {
		names = append(names, "requests")
	}
	if u.CPUSeconds >= l.CPUSeconds {
		names = append(names, "cpu_seconds")
	}
	if u.MemoryGiBSeconds >= l.MemoryGiBSeconds {
		names = append(names, "memory_gib_seconds")
	}
	return names
}

var billingLocation = mustLoadLocation("America/Los_Angeles")

func mustLoadLocation(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

// billingMonthStart returns 00:00 on the first day of the current month in
// Pacific Time, which is where Cloud Billing (and the free tier) resets.
func billingMonthStart(now time.Time) time.Time {
	local := now.In(billingLocation)
	return time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, billingLocation)
}
