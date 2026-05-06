package alertdispatch

import (
	"strings"
	"time"

	"yunshu/internal/pkg/alertnotify"
)

func stringFromPayload(payload map[string]interface{}, key string) string {
	if payload == nil {
		return ""
	}
	return strings.TrimSpace(alertnotify.StringFromPayload(payload, key))
}

func payloadLabelsMap(payload map[string]interface{}) map[string]string {
	labels := alertnotify.PayloadLabels(payload)
	if labels == nil {
		return map[string]string{}
	}
	return labels
}

// BuildChannelTemplateData 构建通道自定义模板可用的变量表（对齐 WatchAlert「通知模板变量」概念）。
// projectName 由调用方提供：云枢在 enrichOutgoingProjectName / resolveNotifyProjectName 中解析 project_id → 名称。
func BuildChannelTemplateData(title, severity, status string, payload map[string]interface{}, projectName string) map[string]interface{} {
	if payload == nil {
		payload = map[string]interface{}{}
	}
	isResolved := strings.EqualFold(strings.TrimSpace(status), "resolved")
	summary := stringFromPayload(payload, "summary")
	if summary == "" {
		summary = strings.TrimSpace(alertnotify.PayloadAnnotation(payload, "summary"))
	}
	description := strings.TrimSpace(alertnotify.PayloadAnnotation(payload, "description"))
	labels := payloadLabelsMap(payload)
	labelsText := "-"
	if compact := alertnotify.FormatCompactLabels(labels, 24); strings.TrimSpace(compact) != "" {
		labelsText = compact
	}
	occurredAt := stringFromPayload(payload, "occurredAt")
	if occurredAt == "" {
		occurredAt = time.Now().In(time.Local).Format("2006-01-02 15:04:05 MST")
	}
	return map[string]interface{}{
		"Title":        title,
		"Severity":     strings.TrimSpace(severity),
		"Status":       strings.TrimSpace(status),
		"StatusText":   map[bool]string{true: "告警恢复", false: "告警触发"}[isResolved],
		"IsResolved":   isResolved,
		"Summary":      summary,
		"Description":  description,
		"ProjectName":  strings.TrimSpace(projectName),
		"Cluster":      stringFromPayload(payload, "cluster"),
		"OccurredAt":   occurredAt,
		"StartsAt":     alertnotify.FormatPayloadTime(payload["startsAt"]),
		"EndsAt":       alertnotify.FormatPayloadTime(payload["endsAt"]),
		"Current":      stringFromPayload(payload, "current"),
		"Count":        stringFromPayload(payload, "count"),
		"Fingerprint":  stringFromPayload(payload, "fingerprint"),
		"GeneratorURL": stringFromPayload(payload, "generatorURL"),
		"Labels":       labels,
		"LabelsText":   labelsText,
	}
}
