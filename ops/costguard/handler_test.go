package main

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func serve(t *testing.T, guard *Guard, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	newHandler(guard, slog.New(slog.DiscardHandler)).ServeHTTP(rec, req)
	return rec
}

func TestHandlerCheck(t *testing.T) {
	service := &fakeService{}
	rec := serve(t, newTestGuard(&fakeUsage{usage: Usage{Requests: 5000}}, service), http.MethodPost, "/check", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"disabled":true`) || !service.disabled {
		t.Fatalf("body = %s, service disabled = %v", rec.Body, service.disabled)
	}
}

func TestHandlerCheckFailure(t *testing.T) {
	service := &fakeService{}
	rec := serve(t, newTestGuard(&fakeUsage{err: errors.New("boom")}, service), http.MethodPost, "/check", "")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if service.disableCalls != 0 {
		t.Fatal("a failed check must not stop the API")
	}
}

func TestHandlerMethodAndPath(t *testing.T) {
	guard := newTestGuard(&fakeUsage{}, &fakeService{})
	if rec := serve(t, guard, http.MethodGet, "/check", ""); rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("GET /check status = %d, want 405", rec.Code)
	}
	if rec := serve(t, guard, http.MethodPost, "/unknown", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("POST /unknown status = %d, want 404", rec.Code)
	}
}

func TestHandlerBudget(t *testing.T) {
	tests := []struct {
		name         string
		body         string
		service      fakeService
		wantStatus   int
		wantDisabled bool
	}{
		{"over budget stops", pushBody(`{"costAmount":500,"budgetAmount":300,"currencyCode":"JPY"}`), fakeService{}, http.StatusOK, true},
		{"under budget", pushBody(`{"costAmount":10,"budgetAmount":300}`), fakeService{}, http.StatusOK, false},
		{"invalid message is acknowledged", "garbage", fakeService{}, http.StatusOK, false},
		{"stop failure is retried by Pub/Sub", pushBody(`{"costAmount":500,"budgetAmount":300}`),
			fakeService{disableErr: errors.New("500")}, http.StatusInternalServerError, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := serve(t, newTestGuard(&fakeUsage{}, &tt.service), http.MethodPost, "/budget", tt.body)
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d (body %s)", rec.Code, tt.wantStatus, rec.Body)
			}
			if tt.service.disabled != tt.wantDisabled {
				t.Fatalf("service disabled = %v, want %v", tt.service.disabled, tt.wantDisabled)
			}
		})
	}
}
