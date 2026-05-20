package k8seventforward

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type WebhookClient struct {
	httpClient *http.Client
	token      string
}

func NewWebhookClient(token string, timeout time.Duration) *WebhookClient {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &WebhookClient{
		httpClient: &http.Client{Timeout: timeout},
		token:      strings.TrimSpace(token),
	}
}

func (c *WebhookClient) PostAlertmanager(ctx context.Context, url string, payload alertManagerPayload) error {
	url = strings.TrimSpace(url)
	if url == "" {
		return fmt.Errorf("webhook url is empty")
	}
	if len(payload.Alerts) == 0 {
		return nil
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("X-Webhook-Token", c.token)
		req.Header.Set("X-Alert-Token", c.token)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}
