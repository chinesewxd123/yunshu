package k8seventforward

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"yunshu/internal/model"
)

// alertManagerPayload 与 service.AlertManagerPayload JSON 对齐，避免包循环依赖。
type alertManagerPayload struct {
	Status   string              `json:"status"`
	Receiver string              `json:"receiver"`
	Alerts   []alertManagerAlert `json:"alerts"`
}

type alertManagerAlert struct {
	Status       string            `json:"status"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	StartsAt     time.Time         `json:"startsAt"`
	Fingerprint  string            `json:"fingerprint"`
}

func buildAlertManagerPayload(ruleName, clusterID, clusterName string, events []model.K8sForwardedEvent) alertManagerPayload {
	clusterLabel := strings.TrimSpace(clusterName)
	if clusterLabel == "" {
		clusterLabel = clusterID
	}
	alerts := make([]alertManagerAlert, 0, len(events))
	for _, ev := range events {
		alertname := strings.TrimSpace(ev.Reason)
		if alertname == "" {
			alertname = "K8sWarningEvent"
		}
		fp := sha256.Sum256([]byte(fmt.Sprintf("%s|%s|%s|%s|%s",
			clusterID, ev.Namespace, ev.Name, ev.Reason, ev.Message)))
		starts := ev.Timestamp
		if starts.IsZero() {
			starts = time.Now()
		}
		alerts = append(alerts, alertManagerAlert{
			Status: "firing",
			Labels: map[string]string{
				"alertname":  alertname,
				"severity":   "warning",
				"cluster":    clusterLabel,
				"cluster_id": clusterID,
				"namespace":  ev.Namespace,
				"resource":   ev.Name,
				"reason":     ev.Reason,
				"source":     "k8s_event",
			},
			Annotations: map[string]string{
				"summary":     fmt.Sprintf("[%s] %s/%s %s", clusterLabel, ev.Namespace, ev.Name, ev.Reason),
				"description": ev.Message,
				"rule":        ruleName,
			},
			StartsAt:    starts,
			Fingerprint: hex.EncodeToString(fp[:16]),
		})
	}
	return alertManagerPayload{
		Status:   "firing",
		Receiver: "k8s-events",
		Alerts:   alerts,
	}
}
