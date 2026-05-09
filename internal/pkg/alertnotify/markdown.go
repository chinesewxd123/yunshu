package alertnotify

import (
	"fmt"
	"strings"
	"time"
)

// BuildMarkdownCard 通用夜莺风格 Markdown：适用于 Prometheus/Alertmanager 全类告警（主机、容器、中间件、K8s 等）。
// 段落间使用「---」分隔，列表项双换行以改善企业微信等客户端渲染。
func BuildMarkdownCard(title, status, severity, summary string, dims Dims, payload map[string]interface{}) string {
	occurredAt := strings.TrimSpace(fmt.Sprintf("%v", payload["occurredAt"]))
	current := strings.TrimSpace(fmt.Sprintf("%v", payload["current"]))
	receiver := strings.TrimSpace(fmt.Sprintf("%v", payload["receiver"]))
	fingerprint := strings.TrimSpace(fmt.Sprintf("%v", payload["fingerprint"]))
	count := strings.TrimSpace(fmt.Sprintf("%v", payload["count"]))
	generatorURL := strings.TrimSpace(fmt.Sprintf("%v", payload["generatorURL"]))
	description := strings.TrimSpace(PayloadAnnotation(payload, "description"))
	labels := PayloadLabels(payload)
	hints := AuxiliaryStateHints(labels)
	cluster := StringFromPayload(payload, "cluster")
	if cluster == "" {
		cluster = strings.TrimSpace(dims.Cluster)
	}
	if cluster == "" {
		cluster = InferEnvironmentDisplay(labels, dims)
	}
	if cluster == "" {
		cluster = "unknown"
	}
	isResolved := strings.EqualFold(strings.TrimSpace(status), "resolved")
	wl := FormatWorkload(dims.WorkloadKind, dims.WorkloadName)

	var blocks []string
	var hdr strings.Builder
	if isResolved {
		hdr.WriteString("【**告警恢复**】**")
		hdr.WriteString(title)
		hdr.WriteString("**\n\n")
		hdr.WriteString("**级别**: ")
		hdr.WriteString(strings.TrimSpace(severity))
		hdr.WriteString("\n\n**环境标识**: `")
		hdr.WriteString(cluster)
		hdr.WriteString("`\n\n**通知时间**: ")
		hdr.WriteString(SafeOr(occurredAt, "-"))
	} else {
		hdr.WriteString("【**告警触发**】**")
		hdr.WriteString(title)
		hdr.WriteString("**\n\n")
		hdr.WriteString("**级别**: ")
		hdr.WriteString(strings.TrimSpace(severity))
		hdr.WriteString("\n\n**环境标识**: `")
		hdr.WriteString(cluster)
		hdr.WriteString("`\n\n**通知时间**: ")
		hdr.WriteString(SafeOr(occurredAt, "-"))
	}
	if ts := FormatPayloadTime(payload["startsAt"]); ts != "" {
		hdr.WriteString("\n\n**起始时间**: ")
		hdr.WriteString(ts)
	}
	if te := FormatPayloadTime(payload["endsAt"]); te != "" && isResolved {
		hdr.WriteString("\n\n**结束时间**: ")
		hdr.WriteString(te)
	}
	if isResolved {
		if st, ok1 := ParsePayloadTime(payload["startsAt"]); ok1 {
			if en, ok2 := ParsePayloadTime(payload["endsAt"]); ok2 && en.After(st) {
				if d := en.Sub(st); d > 0 {
					hdr.WriteString("\n\n**持续时长**: ")
					hdr.WriteString(d.Round(time.Second).String())
				}
			}
		}
		rc := strings.TrimSpace(fmt.Sprintf("%v", payload["resolved_count"]))
		if rc != "" && rc != "<nil>" && rc != "0" {
			hdr.WriteString("\n\n**窗口内恢复条数**: ")
			hdr.WriteString(rc)
			rf := strings.TrimSpace(fmt.Sprintf("%v", payload["resolved_first_seen"]))
			rl := strings.TrimSpace(fmt.Sprintf("%v", payload["resolved_last_seen"]))
			if rf != "" && rf != "<nil>" {
				rfDisp := FormatPayloadTime(payload["resolved_first_seen"])
				if rfDisp == "" {
					rfDisp = rf
				}
				rlDisp := FormatPayloadTime(payload["resolved_last_seen"])
				if rlDisp == "" {
					rlDisp = SafeOr(rl, rf)
				}
				hdr.WriteString("\n\n**聚合窗口**: ")
				hdr.WriteString(rfDisp)
				hdr.WriteString(" ~ ")
				hdr.WriteString(rlDisp)
			}
		}
	}
	blocks = append(blocks, hdr.String())

	// 资源与实例（通用顺序：先 exporter 常见维度，再容器云专有维度）
	var res strings.Builder
	writeRes := func(line string) {
		if res.Len() > 0 {
			res.WriteString("\n\n")
		}
		res.WriteString(line)
	}
	if inst := strings.TrimSpace(dims.Instance); inst != "" {
		writeRes("- **监控实例**: `" + inst + "`")
	}
	if job := strings.TrimSpace(dims.Job); job != "" {
		writeRes("- **监控任务**: `" + job + "`")
	}
	if no := strings.TrimSpace(dims.Node); no != "" {
		writeRes("- **节点 / 主机**: `" + no + "`")
	}
	if ns := strings.TrimSpace(dims.Namespace); ns != "" {
		writeRes("- **命名空间**: `" + ns + "`")
	}
	if po := strings.TrimSpace(dims.Pod); po != "" {
		writeRes("- **Pod**: `" + po + "`")
	}
	if wl != "" && wl != "-" {
		writeRes("- **工作负载**: `" + wl + "`")
	}
	if co := strings.TrimSpace(dims.Container); co != "" {
		writeRes("- **容器**: `" + co + "`")
	}
	if se := strings.TrimSpace(dims.Service); se != "" {
		writeRes("- **Service**: `" + se + "`")
	}
	if ep := strings.TrimSpace(dims.Endpoint); ep != "" {
		writeRes("- **Endpoint**: `" + ep + "`")
	}
	if mp := strings.TrimSpace(dims.MetricsPath); mp != "" {
		writeRes("- **Metrics 路径**: `" + mp + "`")
	}
	if ig := strings.TrimSpace(dims.Ingress); ig != "" {
		writeRes("- **Ingress**: `" + ig + "`")
	}
	if hints != "" {
		writeRes("- **关键状态**: `" + hints + "`")
	}
	if res.Len() > 0 {
		blocks = append(blocks, "**资源与实例**\n\n"+res.String())
	} else if hints != "" {
		blocks = append(blocks, "**资源与实例**\n\n- **关键状态**: `"+hints+"`")
	}

	var metric strings.Builder
	addMet := func(line string) {
		if metric.Len() > 0 {
			metric.WriteString("\n\n")
		}
		metric.WriteString(line)
	}
	if !isResolved {
		if current != "" && current != "-" {
			addMet("- **当前观测值**: `" + current + "` _（Prometheus 瞬时查询或缓存）_")
		}
	} else if current != "" && current != "-" {
		addMet("- **恢复前观测值**: `" + current + "`")
	}
	// job / instance 已在「资源与实例」展示，此处不再重复「指标来源」
	if metric.Len() > 0 {
		secTitle := "**观测数据**"
		if isResolved {
			secTitle = "**观测与来源**"
		}
		blocks = append(blocks, secTitle+"\n\n"+metric.String())
	}

	if labels != nil && res.Len() == 0 && hints == "" {
		if compact := FormatCompactLabels(labels, 18); compact != "" {
			blocks = append(blocks, "**标签（节选）**\n\n- "+strings.ReplaceAll(compact, ", ", "\n\n- "))
		}
	}

	sum := strings.TrimSpace(summary)
	if isResolved {
		if sum != "" {
			blocks = append(blocks, "**恢复说明**\n\n"+sum)
		}
		if description != "" && !strings.EqualFold(description, sum) {
			blocks = append(blocks, "**备注**\n\n"+description)
		}
	} else {
		if sum != "" {
			blocks = append(blocks, "**告警摘要**\n\n"+sum)
		}
		if description != "" && !strings.EqualFold(description, sum) {
			blocks = append(blocks, "**详细说明**\n\n"+description)
		}
	}

	if generatorURL != "" && generatorURL != "<nil>" {
		blocks = append(blocks, "**相关链接**\n\n[打开数据源 / 表达式 Graph]("+generatorURL+")")
	}

	var meta strings.Builder
	addMeta := func(line string) {
		if meta.Len() > 0 {
			meta.WriteString("\n\n")
		}
		meta.WriteString(line)
	}
	if fingerprint != "" {
		addMeta("- **事件指纹**: `" + fingerprint + "`")
	}
	if count != "" && count != "-" && count != "<nil>" {
		addMeta("- **本指纹累计通知次数**: " + count)
	}
	if receiver != "" {
		addMeta("- **Alertmanager receiver**: `" + receiver + "`")
	}
	if meta.Len() > 0 {
		blocks = append(blocks, "**事件元数据**\n\n"+meta.String())
	}

	out := strings.Join(blocks, "\n\n---\n\n")
	out = strings.ReplaceAll(out, "\n\n\n", "\n\n")
	return strings.TrimSpace(out)
}

// RenderMarkdownCard 从 webhook payload 解析状态与维度并渲染 Markdown。
func RenderMarkdownCard(title string, payload map[string]interface{}) string {
	status := strings.TrimSpace(fmt.Sprintf("%v", payload["status"]))
	severity := strings.TrimSpace(fmt.Sprintf("%v", payload["severity"]))
	summary := strings.TrimSpace(fmt.Sprintf("%v", payload["summary"]))
	if severity == "" {
		severity = "warning"
	}
	dims := ExtractDims(PayloadLabels(payload))
	return BuildMarkdownCard(title, status, severity, summary, dims, payload)
}
