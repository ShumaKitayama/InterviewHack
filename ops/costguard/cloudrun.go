// DesignDocument/12_Operation.md 12.12.1 停止は手動スケーリング 0（公式の「サービスの無効化」）
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

const cloudRunBaseURL = "https://run.googleapis.com"

// cloudRunClient disables one Cloud Run service through the Admin API v2.
type cloudRunClient struct {
	httpClient *http.Client
	baseURL    string
	project    string
	region     string
	service    string
}

type serviceScaling struct {
	ScalingMode         string `json:"scalingMode,omitempty"`
	ManualInstanceCount *int32 `json:"manualInstanceCount,omitempty"`
}

type serviceResource struct {
	Scaling *serviceScaling `json:"scaling,omitempty"`
}

func (c *cloudRunClient) serviceURL() string {
	return fmt.Sprintf("%s/v2/projects/%s/locations/%s/services/%s", c.baseURL,
		url.PathEscape(c.project), url.PathEscape(c.region), url.PathEscape(c.service))
}

// IsDisabled reports whether the service is in manual scaling with 0 instances.
// An absent manualInstanceCount is the proto default, 0.
func (c *cloudRunClient) IsDisabled(ctx context.Context) (bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.serviceURL(), nil)
	if err != nil {
		return false, err
	}
	var svc serviceResource
	if err := doJSON(c.httpClient, req, &svc); err != nil {
		return false, err
	}
	if svc.Scaling == nil || svc.Scaling.ScalingMode != "MANUAL" {
		return false, nil
	}
	return svc.Scaling.ManualInstanceCount == nil || *svc.Scaling.ManualInstanceCount == 0, nil
}

// Disable switches the service to manual scaling with 0 instances. Revisions,
// configuration and the URL are kept; `--scaling=auto` restores it.
func (c *cloudRunClient) Disable(ctx context.Context) error {
	body, err := json.Marshal(serviceResource{Scaling: &serviceScaling{
		ScalingMode:         "MANUAL",
		ManualInstanceCount: new(int32(0)),
	}})
	if err != nil {
		return err
	}
	query := url.Values{"updateMask": {"scaling.scalingMode,scaling.manualInstanceCount"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, c.serviceURL()+"?"+query.Encode(), bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return doJSON(c.httpClient, req, nil)
}
