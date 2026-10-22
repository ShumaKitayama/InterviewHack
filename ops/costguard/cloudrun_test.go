package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

const testServicePath = "/v2/projects/proj/locations/asia-northeast1/services/api"

func newTestCloudRun(srv *httptest.Server) *cloudRunClient {
	return &cloudRunClient{httpClient: srv.Client(), baseURL: srv.URL, project: "proj", region: "asia-northeast1", service: "api"}
}

func TestCloudRunIsDisabled(t *testing.T) {
	tests := []struct {
		name string
		body string
		want bool
	}{
		{"automatic scaling", `{"name":"api","scaling":{"minInstanceCount":0}}`, false},
		{"no scaling block", `{"name":"api"}`, false},
		{"manual with instances", `{"scaling":{"scalingMode":"MANUAL","manualInstanceCount":2}}`, false},
		{"manual zero", `{"scaling":{"scalingMode":"MANUAL","manualInstanceCount":0}}`, true},
		{"manual count omitted", `{"scaling":{"scalingMode":"MANUAL"}}`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != testServicePath {
					t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
				}
				respond(w, tt.body)
			}))
			defer srv.Close()

			got, err := newTestCloudRun(srv).IsDisabled(t.Context())
			if err != nil {
				t.Fatalf("IsDisabled() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("IsDisabled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCloudRunDisable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != testServicePath {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if got := r.URL.Query().Get("updateMask"); got != "scaling.scalingMode,scaling.manualInstanceCount" {
			t.Errorf("updateMask = %q", got)
		}
		var body struct {
			Scaling map[string]any `json:"scaling"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		// manualInstanceCount must be sent explicitly as 0, not omitted.
		if body.Scaling["scalingMode"] != "MANUAL" || body.Scaling["manualInstanceCount"] != float64(0) {
			t.Errorf("scaling = %v", body.Scaling)
		}
		respond(w, `{"name":"operations/123"}`)
	}))
	defer srv.Close()

	if err := newTestCloudRun(srv).Disable(t.Context()); err != nil {
		t.Fatalf("Disable() error = %v", err)
	}
}

func TestCloudRunDisableError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "denied", http.StatusForbidden)
	}))
	defer srv.Close()

	if err := newTestCloudRun(srv).Disable(t.Context()); err == nil {
		t.Fatal("Disable() error = nil, want error")
	}
}
