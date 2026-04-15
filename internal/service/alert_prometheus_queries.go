package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	neturl "net/url"
	"strings"
	"time"
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
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(s.cfg.PrometheusToken) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(s.cfg.PrometheusToken))
	}
	client := &http.Client{Timeout: time.Duration(s.cfg.PromQueryTimeout) * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("prometheus status %d", resp.StatusCode)
	}
	var data struct {
		Status string `json:"status"`
		Data   struct {
			Result []struct {
				Value []interface{} `json:"value"`
			} `json:"result"`
		} `json:"data"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	if len(data.Data.Result) == 0 || len(data.Data.Result[0].Value) < 2 {
		return "", nil
	}
	return strings.TrimSpace(fmt.Sprintf("%v", data.Data.Result[0].Value[1])), nil
}
