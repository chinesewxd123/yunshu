package service

import (
	"context"
	"fmt"
	neturl "net/url"
	"strings"

	"go-permission-system/internal/pkg/platformhttp"
)

func (s *AlertService) queryCurrentValueByGeneratorURL(ctx context.Context, generatorURL string) (string, error) {
	if strings.TrimSpace(s.cfg.PrometheusURL) == "" || strings.TrimSpace(generatorURL) == "" {
		return "", nil
	}
	u, err := neturl.Parse(generatorURL)
	if err != nil {
		return "", err
	}
	expr := strings.TrimSpace(u.Query().Get("g0.expr"))
	if expr == "" {
		return "", nil
	}
	apiURL := strings.TrimRight(strings.TrimSpace(s.cfg.PrometheusURL), "/") + "/api/v1/query?query=" + neturl.QueryEscape(expr)
	var data struct {
		Status string `json:"status"`
		Data   struct {
			Result []struct {
				Value []interface{} `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}
	headers := map[string]string{}
	if token := strings.TrimSpace(s.cfg.PrometheusToken); token != "" {
		headers["Authorization"] = "Bearer " + token
	}
	client := s.platformHTTPClient()
	if client == (platformhttp.Client{}) {
		client = platformhttp.Client{}
	}
	if _, err := client.DoJSON(ctx, "GET", apiURL, headers, nil, &data); err != nil {
		return "", fmt.Errorf("prometheus query failed: %w", err)
	}
	if len(data.Data.Result) == 0 || len(data.Data.Result[0].Value) < 2 {
		return "", nil
	}
	return strings.TrimSpace(fmt.Sprintf("%v", data.Data.Result[0].Value[1])), nil
}
