// DesignDocument/12_Operation.md 12.12.3 予算通知 / 12.12.4 POST /budget
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// BudgetNotification is the Cloud Billing budget message delivered through Pub/Sub.
// Only the fields the guard decides on are decoded.
type BudgetNotification struct {
	BudgetDisplayName      string  `json:"budgetDisplayName"`
	CostAmount             float64 `json:"costAmount"`
	BudgetAmount           float64 `json:"budgetAmount"`
	CurrencyCode           string  `json:"currencyCode"`
	AlertThresholdExceeded float64 `json:"alertThresholdExceeded"`
}

// pushEnvelope is the body of a Pub/Sub push request. Data is base64 in JSON,
// which encoding/json decodes into []byte.
type pushEnvelope struct {
	Message struct {
		Data []byte `json:"data"`
	} `json:"message"`
}

var errEmptyBudgetMessage = errors.New("pub/sub message has no data")

// parseBudgetPush decodes a Pub/Sub push body into a budget notification.
func parseBudgetPush(body io.Reader) (BudgetNotification, error) {
	var envelope pushEnvelope
	if err := json.NewDecoder(body).Decode(&envelope); err != nil {
		return BudgetNotification{}, fmt.Errorf("decode push envelope: %w", err)
	}
	if len(envelope.Message.Data) == 0 {
		return BudgetNotification{}, errEmptyBudgetMessage
	}
	var notification BudgetNotification
	if err := json.Unmarshal(envelope.Message.Data, &notification); err != nil {
		return BudgetNotification{}, fmt.Errorf("decode budget notification: %w", err)
	}
	return notification, nil
}
