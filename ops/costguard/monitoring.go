// DesignDocument/12_Operation.md 12.12.3 Cloud Monitoring から当月の使用量を集計する
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const monitoringBaseURL = "https://monitoring.googleapis.com"

// usageMetrics maps each Cloud Run metric to the Usage field it is added to.
var usageMetrics = []struct {
	metricType string
	add        func(*Usage, float64)
}{
	{"run.googleapis.com/request_count", func(u *Usage, v float64) { u.Requests += v }},
	{"run.googleapis.com/container/cpu/allocation_time", func(u *Usage, v float64) { u.CPUSeconds += v }},
	{"run.googleapis.com/container/memory/allocation_time", func(u *Usage, v float64) { u.MemoryGiBSeconds += v }},
}

// monitoringClient reads Cloud Run usage with the Cloud Monitoring v3 REST API.
type monitoringClient struct {
	httpClient *http.Client
	baseURL    string
	projects   []string
}

// MonthToDate sums every usage metric over all Cloud Run services of the
// monitored projects between start and end.
func (c *monitoringClient) MonthToDate(ctx context.Context, start, end time.Time) (Usage, error) {
	var usage Usage
	for _, project := range c.projects {
		for _, metric := range usageMetrics {
			total, err := c.sum(ctx, project, metric.metricType, start, end)
			if err != nil {
				return Usage{}, fmt.Errorf("%s %s: %w", project, metric.metricType, err)
			}
			metric.add(&usage, total)
		}
	}
	return usage, nil
}

type timeSeriesListResponse struct {
	TimeSeries []struct {
		Points []struct {
			Value struct {
				Int64Value  string   `json:"int64Value"` // int64 is a JSON string in Google APIs
				DoubleValue *float64 `json:"doubleValue"`
			} `json:"value"`
		} `json:"points"`
	} `json:"timeSeries"`
	NextPageToken string `json:"nextPageToken"`
}

// sum aligns the metric into hourly sums, reduces all series into one and
// adds up every point, following page tokens.
func (c *monitoringClient) sum(ctx context.Context, project, metricType string, start, end time.Time) (float64, error) {
	query := url.Values{}
	query.Set("filter", fmt.Sprintf(`metric.type=%q AND resource.type="cloud_run_revision"`, metricType))
	query.Set("interval.startTime", start.UTC().Format(time.RFC3339))
	query.Set("interval.endTime", end.UTC().Format(time.RFC3339))
	query.Set("aggregation.alignmentPeriod", "3600s")
	query.Set("aggregation.perSeriesAligner", "ALIGN_SUM")
	query.Set("aggregation.crossSeriesReducer", "REDUCE_SUM")
	endpoint := fmt.Sprintf("%s/v3/projects/%s/timeSeries", c.baseURL, url.PathEscape(project))

	var total float64
	for {
		var page timeSeriesListResponse
		if err := c.get(ctx, endpoint+"?"+query.Encode(), &page); err != nil {
			return 0, err
		}
		for _, series := range page.TimeSeries {
			for _, point := range series.Points {
				switch {
				case point.Value.DoubleValue != nil:
					total += *point.Value.DoubleValue
				case point.Value.Int64Value != "":
					n, err := strconv.ParseInt(point.Value.Int64Value, 10, 64)
					if err != nil {
						return 0, fmt.Errorf("parse int64Value %q: %w", point.Value.Int64Value, err)
					}
					total += float64(n)
				}
			}
		}
		if page.NextPageToken == "" {
			return total, nil
		}
		query.Set("pageToken", page.NextPageToken)
	}
}

func (c *monitoringClient) get(ctx context.Context, rawURL string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	return doJSON(c.httpClient, req, out)
}

// doJSON sends req and decodes a 2xx JSON response into out (if non-nil).
// Error bodies from Google APIs carry no secrets, so a bounded prefix is kept
// for diagnosis.
func doJSON(client *http.Client, req *http.Request, out any) error {
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("%s %s: status %d: %s", req.Method, req.URL.Path, resp.StatusCode, snippet)
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode %s response: %w", req.URL.Path, err)
	}
	return nil
}
