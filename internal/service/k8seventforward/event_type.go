package k8seventforward

import "strings"

// K8s Event API（events.k8s.io/v1）标准 type 仅 Normal、Warning；历史 core/v1 亦同。
// 转发策略：Normal 不发送，其余类型均发送。
func eventSeverity(eventType string) string {
	t := strings.TrimSpace(eventType)
	switch {
	case strings.EqualFold(t, "Warning"):
		return "warning"
	case strings.EqualFold(t, "Normal"):
		return "info"
	case t == "":
		return "info"
	default:
		return "info"
	}
}

func defaultAlertname(eventType, reason string) string {
	if r := strings.TrimSpace(reason); r != "" {
		return r
	}
	if t := strings.TrimSpace(eventType); t != "" {
		return "K8s" + t + "Event"
	}
	return "K8sEvent"
}
