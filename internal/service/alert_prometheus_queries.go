package service

import (
	"context"
	"fmt"
	neturl "net/url"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/promapi"
	"yunshu/internal/pkg/platformhttp"
)

func metricValueFromAnnotations(annotations map[string]string) string {
	if annotations == nil {
		return ""
	}
	for _, key := range []string{"value", "Value", "current_value"} {
		if v := strings.TrimSpace(annotations[key]); v != "" && v != "-" {
			return v
		}
	}
	return ""
}

// queryCurrentValueByMonitorRule 按监控规则绑定的 Prometheus 数据源即时查询 PromQL，取首个样本值。
func (s *AlertService) queryCurrentValueByMonitorRule(ctx context.Context, labels map[string]string) (string, error) {
	rid, ok := parseLabelUint(labels["monitor_rule_id"])
	if !ok || rid == 0 {
		return "", nil
	}
	var rule model.AlertMonitorRule
	if err := s.db.WithContext(ctx).First(&rule, rid).Error; err != nil {
		return "", err
	}
	var ds model.AlertDatasource
	if err := s.db.WithContext(ctx).First(&ds, rule.DatasourceID).Error; err != nil {
		return "", err
	}
	if !ds.Enabled || strings.TrimSpace(ds.Type) != "prometheus" {
		return "", nil
	}
	expr := strings.TrimSpace(rule.Expr)
	if expr == "" {
		return "", nil
	}
	cli := &promapi.Client{
		BaseURL:       strings.TrimRight(strings.TrimSpace(ds.BaseURL), "/"),
		BearerToken:   ds.BearerToken,
		BasicUser:     ds.BasicUser,
		BasicPassword: ds.BasicPassword,
		SkipTLSVerify: ds.SkipTLSVerify,
	}
	timeout := maxInt(3, s.cfg.PromQueryTimeout)
	qctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()
	body, _, err := cli.QueryInstant(qctx, expr, "")
	if err != nil {
		return "", err
	}
	_, value := parsePromFirstSample(body)
	return strings.TrimSpace(value), nil
}

// resolveIngressMetricValues 解析入站告警的触发侧指标值与恢复时再查值。
func (s *AlertService) resolveIngressMetricValues(ctx context.Context, source, status string, labels, annotations map[string]string, alert AlertManagerAlert) (current, resolved string) {
	fp := strings.TrimSpace(alert.Fingerprint)
	if strings.EqualFold(status, "resolved") {
		current = strings.TrimSpace(s.getCachedCurrentValue(ctx, fp))
		if current == "" {
			current = metricValueFromAnnotations(annotations)
		}
		if strings.EqualFold(source, "platform_monitor") {
			if v, err := s.queryCurrentValueByMonitorRule(ctx, labels); err == nil {
				resolved = strings.TrimSpace(v)
			}
		}
		if resolved == "" && strings.TrimSpace(s.cfg.PrometheusURL) != "" && strings.TrimSpace(alert.GeneratorURL) != "" {
			if v, err := s.queryCurrentValueByGeneratorURL(ctx, alert.GeneratorURL); err == nil {
				resolved = strings.TrimSpace(v)
			}
		}
		if resolved == "" {
			resolved = metricValueFromAnnotations(annotations)
		}
		return current, resolved
	}
	current = metricValueFromAnnotations(annotations)
	if current == "" {
		current = strings.TrimSpace(s.getCachedCurrentValue(ctx, fp))
	}
	if current == "" && strings.EqualFold(source, "platform_monitor") {
		if v, err := s.queryCurrentValueByMonitorRule(ctx, labels); err == nil {
			current = strings.TrimSpace(v)
		}
	}
	if current == "" && strings.TrimSpace(s.cfg.PrometheusURL) != "" && strings.TrimSpace(alert.GeneratorURL) != "" {
		if v, err := s.queryCurrentValueByGeneratorURL(ctx, alert.GeneratorURL); err == nil {
			current = strings.TrimSpace(v)
		}
	}
	if current != "" && fp != "" {
		s.setCachedCurrentValue(ctx, fp, current)
	}
	return current, ""
}

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
