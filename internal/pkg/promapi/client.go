package promapi

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	neturl "net/url"
	"strings"
	"time"
)

// Client 调用 Prometheus HTTP API（/api/v1/query、/api/v1/query_range）。
type Client struct {
	BaseURL       string
	BearerToken   string
	BasicUser     string
	BasicPassword string
	SkipTLSVerify bool
	HTTPClient    *http.Client
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	tr := http.DefaultTransport.(*http.Transport).Clone()
	if c.SkipTLSVerify {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	}
	return &http.Client{Timeout: 30 * time.Second, Transport: tr}
}

func (c *Client) authHeader(req *http.Request) {
	t := strings.TrimSpace(c.BearerToken)
	if t != "" {
		req.Header.Set("Authorization", "Bearer "+t)
		return
	}
	u := strings.TrimSpace(c.BasicUser)
	p := strings.TrimSpace(c.BasicPassword)
	if u != "" || p != "" {
		req.SetBasicAuth(u, p)
	}
}

// QueryInstant GET /api/v1/query
func (c *Client) QueryInstant(ctx context.Context, query, evalTime string) (json.RawMessage, int, error) {
	base := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if base == "" {
		return nil, 0, fmt.Errorf("prometheus base_url is empty")
	}
	u, err := neturl.Parse(base + "/api/v1/query")
	if err != nil {
		return nil, 0, err
	}
	q := u.Query()
	q.Set("query", query)
	if strings.TrimSpace(evalTime) != "" {
		q.Set("time", evalTime)
	}
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, 0, err
	}
	c.authHeader(req)
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, fmt.Errorf("prometheus status %d: %s", resp.StatusCode, truncate(string(body), 512))
	}
	return json.RawMessage(body), resp.StatusCode, nil
}

// QueryRange GET /api/v1/query_range
func (c *Client) QueryRange(ctx context.Context, query, start, end, step string) (json.RawMessage, int, error) {
	base := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if base == "" {
		return nil, 0, fmt.Errorf("prometheus base_url is empty")
	}
	u, err := neturl.Parse(base + "/api/v1/query_range")
	if err != nil {
		return nil, 0, err
	}
	q := u.Query()
	q.Set("query", query)
	q.Set("start", start)
	q.Set("end", end)
	q.Set("step", step)
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, 0, err
	}
	c.authHeader(req)
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, fmt.Errorf("prometheus status %d: %s", resp.StatusCode, truncate(string(body), 512))
	}
	return json.RawMessage(body), resp.StatusCode, nil
}

// ActiveAlerts GET /api/v1/alerts — 返回 Prometheus 当前规则评估产生的活跃告警（与 Alertmanager UI 的「来源」不同，但可与平台 Webhook 历史对照）。
func (c *Client) ActiveAlerts(ctx context.Context) (json.RawMessage, int, error) {
	base := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if base == "" {
		return nil, 0, fmt.Errorf("prometheus base_url is empty")
	}
	u := base + "/api/v1/alerts"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, 0, err
	}
	c.authHeader(req)
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, resp.StatusCode, fmt.Errorf("prometheus status %d: %s", resp.StatusCode, truncate(string(body), 512))
	}
	return json.RawMessage(body), resp.StatusCode, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// VectorResultNonEmpty 解析 instant query 响应：vector 且 result 非空则认为「有告警样本」。
func VectorResultNonEmpty(body json.RawMessage) (bool, error) {
	var wrap struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string          `json:"resultType"`
			Result     json.RawMessage `json:"result"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &wrap); err != nil {
		return false, err
	}
	if wrap.Status != "success" {
		return false, fmt.Errorf("prometheus status field: %s", wrap.Status)
	}
	if wrap.Data.ResultType == "scalar" {
		var pair []interface{}
		if err := json.Unmarshal(wrap.Data.Result, &pair); err != nil {
			return false, err
		}
		if len(pair) < 2 {
			return false, nil
		}
		switch v := pair[1].(type) {
		case string:
			s := strings.TrimSpace(v)
			return s != "" && s != "0" && s != "+Inf" && strings.ToLower(s) != "nan", nil
		case float64:
			return v != 0 && !math.IsNaN(v), nil
		default:
			s := strings.TrimSpace(fmt.Sprintf("%v", pair[1]))
			return s != "" && s != "0", nil
		}
	}
	if wrap.Data.ResultType != "vector" {
		return len(strings.TrimSpace(string(wrap.Data.Result))) > 2, nil
	}
	var vec []json.RawMessage
	if err := json.Unmarshal(wrap.Data.Result, &vec); err != nil {
		return false, err
	}
	return len(vec) > 0, nil
}
