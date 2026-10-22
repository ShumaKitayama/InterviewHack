package main

import (
	"encoding/base64"
	"errors"
	"strings"
	"testing"
)

func pushBody(data string) string {
	return `{"message":{"data":"` + base64.StdEncoding.EncodeToString([]byte(data)) + `","messageId":"1"},"subscription":"projects/p/subscriptions/s"}`
}

func TestParseBudgetPush(t *testing.T) {
	got, err := parseBudgetPush(strings.NewReader(pushBody(
		`{"budgetDisplayName":"guard","alertThresholdExceeded":1.0,"costAmount":312.5,"costIntervalStart":"2026-10-01T07:00:00Z","budgetAmount":300,"budgetAmountType":"SPECIFIED_AMOUNT","currencyCode":"JPY"}`)))
	if err != nil {
		t.Fatalf("parseBudgetPush() error = %v", err)
	}
	want := BudgetNotification{BudgetDisplayName: "guard", CostAmount: 312.5, BudgetAmount: 300, CurrencyCode: "JPY", AlertThresholdExceeded: 1}
	if got != want {
		t.Fatalf("parseBudgetPush() = %+v, want %+v", got, want)
	}
}

func TestParseBudgetPushErrors(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"not json", "hello"},
		{"no data", `{"message":{"messageId":"1"}}`},
		{"data is not base64", `{"message":{"data":"%%%"}}`},
		{"data is not json", pushBody("not json")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := parseBudgetPush(strings.NewReader(tt.body)); err == nil {
				t.Fatal("parseBudgetPush() error = nil, want error")
			}
		})
	}
	if _, err := parseBudgetPush(strings.NewReader(`{"message":{}}`)); !errors.Is(err, errEmptyBudgetMessage) {
		t.Fatalf("empty message error = %v, want errEmptyBudgetMessage", err)
	}
}
