package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func respond(w http.ResponseWriter, body string) {
	_, _ = io.WriteString(w, body)
}

func TestMonitoringMonthToDate(t *testing.T) {
	start := time.Date(2026, 10, 1, 7, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		q := r.URL.Query()
		if q.Get("interval.startTime") != "2026-10-01T07:00:00Z" || q.Get("interval.endTime") != "2026-10-22T12:00:00Z" {
			t.Errorf("unexpected interval: %s - %s", q.Get("interval.startTime"), q.Get("interval.endTime"))
		}
		if q.Get("aggregation.perSeriesAligner") != "ALIGN_SUM" || q.Get("aggregation.crossSeriesReducer") != "REDUCE_SUM" {
			t.Errorf("unexpected aggregation: %v", q)
		}
		if !strings.Contains(q.Get("filter"), `resource.type="cloud_run_revision"`) {
			t.Errorf("filter %q does not select Cloud Run", q.Get("filter"))
		}
		project := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/v3/projects/"), "/timeSeries")
		filter := q.Get("filter")
		switch {
		case strings.Contains(filter, "request_count") && q.Get("pageToken") == "":
			// Requests arrive as int64 strings and span two pages.
			respond(w, `{"timeSeries":[{"points":[{"value":{"int64Value":"1000"}},{"value":{"int64Value":"500"}}]}],"nextPageToken":"p2"}`)
		case strings.Contains(filter, "request_count"):
			respond(w, `{"timeSeries":[{"points":[{"value":{"int64Value":"25"}}]}]}`)
		case strings.Contains(filter, "cpu/allocation_time"):
			respond(w, `{"timeSeries":[{"points":[{"value":{"doubleValue":1.5}},{"value":{"doubleValue":2.25}}]}]}`)
		case strings.Contains(filter, "memory/allocation_time") && project == "proj-b":
			respond(w, `{}`) // no data this month
		case strings.Contains(filter, "memory/allocation_time"):
			respond(w, `{"timeSeries":[{"points":[{"value":{"doubleValue":10}}]}]}`)
		default:
			t.Errorf("unexpected filter %q", filter)
		}
	}))
	defer srv.Close()

	client := &monitoringClient{httpClient: srv.Client(), baseURL: srv.URL, projects: []string{"proj-a", "proj-b"}}
	got, err := client.MonthToDate(t.Context(), start, end)
	if err != nil {
		t.Fatalf("MonthToDate() error = %v", err)
	}
	want := Usage{Requests: 2 * 1525, CPUSeconds: 2 * 3.75, MemoryGiBSeconds: 10}
	if got != want {
		t.Fatalf("MonthToDate() = %+v, want %+v", got, want)
	}
	if calls != 8 { // 2 projects x (2 request pages + cpu + memory)
		t.Fatalf("calls = %d, want 8", calls)
	}
}

func TestMonitoringErrorStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"error":{"code":403,"message":"Permission denied"}}`, http.StatusForbidden)
	}))
	defer srv.Close()

	client := &monitoringClient{httpClient: srv.Client(), baseURL: srv.URL, projects: []string{"p"}}
	_, err := client.MonthToDate(t.Context(), time.Now().Add(-time.Hour), time.Now())
	if err == nil || !strings.Contains(err.Error(), "status 403") {
		t.Fatalf("MonthToDate() error = %v, want status 403", err)
	}
}

func TestMonitoringInvalidInt64(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		respond(w, `{"timeSeries":[{"points":[{"value":{"int64Value":"abc"}}]}]}`)
	}))
	defer srv.Close()

	client := &monitoringClient{httpClient: srv.Client(), baseURL: srv.URL, projects: []string{"p"}}
	if _, err := client.MonthToDate(t.Context(), time.Now().Add(-time.Hour), time.Now()); err == nil {
		t.Fatal("MonthToDate() error = nil, want parse error")
	}
}
