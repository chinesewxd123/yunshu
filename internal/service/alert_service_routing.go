package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"yunshu/internal/model"
	"yunshu/internal/pkg/alertnotify"
)

func (s *AlertService) channelIDSetForAlert(ctx context.Context, status string, labels map[string]string) (map[uint]struct{}, string, string, int, []uint) {
	// 彻底弃用旧策略：仅使用订阅树路由（订阅节点 -> 接收组 -> 通道）
	if s.subscriptionSvc == nil || s.receiverGroupCache == nil {
		return nil, "", "", 0, nil
	}
	projectID := s.resolveProjectIDForAlertRouting(ctx, labels)
	severity := strings.TrimSpace(labels["severity"])
	var merged AlertRouteResult
	var anyMatched bool
	tryMatch := func(pid uint) {
		route, ok := s.subscriptionSvc.MatchRouteDetailed(ctx, pid, labels, severity, status)
		if !ok || len(route.ReceiverGroupIDs) == 0 {
			return
		}
		anyMatched = true
		merged.ReceiverGroupIDs = append(merged.ReceiverGroupIDs, route.ReceiverGroupIDs...)
		merged.MatchedNodeIDs = append(merged.MatchedNodeIDs, route.MatchedNodeIDs...)
		merged.MatchedNodeNames = append(merged.MatchedNodeNames, route.MatchedNodeNames...)
		if route.SilenceSeconds > merged.SilenceSeconds {
			merged.SilenceSeconds = route.SilenceSeconds
		}
		if merged.MatchedPath == "" && route.MatchedPath != "" {
			merged.MatchedPath = route.MatchedPath
		}
	}
	if projectID > 0 {
		tryMatch(projectID)
	}
	tryMatch(0)
	if !anyMatched {
		return nil, "", "", 0, nil
	}
	out := map[uint]struct{}{}
	for _, gid := range merged.ReceiverGroupIDs {
		g, err := s.receiverGroupCache.Get(gid)
		if err != nil || g == nil {
			continue
		}
		if !g.IsActiveNow() {
			continue
		}
		for _, cid := range g.ChannelIDs {
			if cid > 0 {
				out[cid] = struct{}{}
			}
		}
	}
	ids := make([]string, 0, len(merged.MatchedNodeIDs))
	for _, id := range merged.MatchedNodeIDs {
		ids = append(ids, fmt.Sprintf("%d", id))
	}
	return out, strings.Join(ids, ","), strings.Join(merged.MatchedNodeNames, ","), merged.SilenceSeconds, uniqUint(merged.ReceiverGroupIDs)
}

// ChannelRouteForAlert 订阅匹配后的通道与接收组信息。
type ChannelRouteForAlert struct {
	ChannelIDs       map[uint]struct{}
	MatchedPolicyIDs string
	MatchedPolicyNames string
	SilenceSeconds   int
	ReceiverGroupIDs []uint
}

func (s *AlertService) channelRouteForAlert(ctx context.Context, status string, labels map[string]string) ChannelRouteForAlert {
	ch, ids, names, silence, rgs := s.channelIDSetForAlert(ctx, status, labels)
	return ChannelRouteForAlert{
		ChannelIDs:         ch,
		MatchedPolicyIDs:   ids,
		MatchedPolicyNames: names,
		SilenceSeconds:     silence,
		ReceiverGroupIDs:   rgs,
	}
}

func (s *AlertService) resolveProjectIDForAlertRouting(ctx context.Context, labels map[string]string) uint {
	if labels == nil {
		return 0
	}
	if pid := parseLabelUintOrZero(labels["project_id"]); pid > 0 {
		return pid
	}
	for _, key := range []string{"datasource_id", "yunshu_datasource_id"} {
		if dsID, ok := parseLabelUint(labels[key]); ok && dsID > 0 {
			var ds model.AlertDatasource
			if err := s.db.WithContext(ctx).Select("project_id").First(&ds, dsID).Error; err == nil && ds.ProjectID > 0 {
				return ds.ProjectID
			}
		}
	}
	return 0
}

func parseLabelUintOrZero(s string) uint {
	n, ok := parseLabelUint(s)
	if !ok {
		return 0
	}
	return n
}

func (s *AlertService) shouldSuppressByRouteSilence(ctx context.Context, status, groupKey, matchedNodeIDs string, silenceSeconds int, labels map[string]string) bool {
	if s.redis == nil || silenceSeconds <= 0 || status != "firing" {
		return false
	}
	gk := strings.TrimSpace(groupKey)
	nid := strings.TrimSpace(matchedNodeIDs)
	if gk == "" || nid == "" {
		return false
	}
	key := "alert:subscription:silence:" + gk + ":" + nid
	ok, err := s.redis.SetNX(ctx, key, "1", time.Duration(silenceSeconds)*time.Second).Result()
	if err != nil {
		return false
	}
	if labels != nil {
		if ruleID, parsed := parseLabelUint(labels["monitor_rule_id"]); parsed && ruleID > 0 {
			// 为规则列表页提供可观测状态：订阅静默窗口剩余时间
			_ = s.redis.Set(ctx, fmt.Sprintf("alert:subscription:silence:rule:%d", ruleID), nid, time.Duration(silenceSeconds)*time.Second).Err()
		}
	}
	return !ok
}

func (s *AlertService) logSuppressedRouteSilence(ctx context.Context, title, severity, status, cluster, groupKey, labelsDigest string, silenceSeconds int, payload map[string]interface{}) {
	reqBytes, _ := json.Marshal(payload)
	event := model.AlertEvent{
		Source:          alertEventSourceFromPayload(payload),
		Title:           title + " (subscription silence suppressed)",
		Severity:        severity,
		Status:          status,
		Cluster:         cluster,
		MonitorPipeline: strings.TrimSpace(fmt.Sprintf("%v", payload["monitorPipeline"])),
		GroupKey:        strings.TrimSpace(groupKey),
		LabelsDigest:    strings.TrimSpace(labelsDigest),
		ChannelName:     "（未外发·订阅静默窗口抑制）",
		Success:         true,
		HTTPStatusCode:  200,
		ErrorMessage:    "subscription_suppressed",
		RequestPayload:  truncateText(string(reqBytes), s.cfg.MaxPayloadChars),
		ResponsePayload: truncateText(fmt.Sprintf("suppressed by subscription silence_seconds=%d", silenceSeconds), s.cfg.MaxPayloadChars),
	}
	fillAlertEventDatasourceFromPayload(&event, payload)
	_ = s.db.WithContext(ctx).Create(&event).Error
}

func (s *AlertService) computeGroupKey(receiver, status, severity, alertname string, labels map[string]string, dims alertnotify.Dims) string {
	fields := s.cfg.GroupBy
	if len(fields) == 0 {
		fields = []string{"alertname", "cluster", "namespace", "severity", "receiver"}
	}
	parts := make([]string, 0, len(fields))
	for _, f := range fields {
		parts = append(parts, strings.ToLower(strings.TrimSpace(f))+"="+s.resolveGroupByLabelValue(f, receiver, status, severity, alertname, labels, dims))
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return "gk_" + hex.EncodeToString(sum[:8])
}

func channelMatchesAlert(settings map[string]interface{}, labels map[string]string, dims alertnotify.Dims) bool {
	if settings == nil {
		return true
	}
	getLabel := func(k string) string {
		switch k {
		case "cluster":
			return dims.Cluster
		case "namespace":
			return dims.Namespace
		default:
			if labels == nil {
				return ""
			}
			return labels[k]
		}
	}

	// matchLabels: {"cluster":"prod-1","namespace":"kube-system"}
	if raw, ok := settings["matchLabels"]; ok {
		if m, ok := raw.(map[string]interface{}); ok {
			for k, v := range m {
				expected := strings.TrimSpace(fmt.Sprintf("%v", v))
				if expected == "" {
					continue
				}
				actual := strings.TrimSpace(getLabel(k))
				if actual != expected {
					return false
				}
			}
		}
	}
	// matchRegex: {"namespace":"^(kube-system|monitoring)$"}
	if raw, ok := settings["matchRegex"]; ok {
		if m, ok := raw.(map[string]interface{}); ok {
			for k, v := range m {
				pat := strings.TrimSpace(fmt.Sprintf("%v", v))
				if pat == "" {
					continue
				}
				re, err := regexp.Compile(pat)
				if err != nil {
					return false
				}
				if !re.MatchString(strings.TrimSpace(getLabel(k))) {
					return false
				}
			}
		}
	}
	return true
}

func parseLabelUint(v string) (uint, bool) {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0, false
	}
	n, err := strconv.ParseUint(v, 10, 32)
	if err != nil {
		return 0, false
	}
	return uint(n), true
}

func mergeNotifyEmailsUnique(emails []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, e := range emails {
		e = strings.TrimSpace(strings.ToLower(e))
		if e == "" {
			continue
		}
		if _, ok := seen[e]; ok {
			continue
		}
		seen[e] = struct{}{}
		out = append(out, e)
	}
	return out
}

func mergeNotifyPhonesUnique(phones []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, p := range phones {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

func (s *AlertService) enrichAssigneeAndDutyEmails(ctx context.Context, outgoing map[string]interface{}, labels map[string]string) {
	rid, ok := parseLabelUint(labels["monitor_rule_id"])
	if !ok {
		return
	}
	status := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", outgoing["status"])))
	var emails []string
	if s.assigneeSvc != nil && (status != "resolved" || s.assigneeSvc.NotifyOnResolvedEnabled(ctx, rid)) {
		// 邮件仅「用户」勾选 +「邮箱」额外项；部门子树只参与 IM @（ResolveNotifyPhones）
		e, _ := s.assigneeSvc.ResolveNotifyEmailsDirectUsers(ctx, rid)
		emails = append(emails, e...)
	}
	if s.dutySvc != nil {
		e, _ := s.dutySvc.ResolveNotifyEmailsAtRule(ctx, rid, time.Now())
		emails = append(emails, e...)
	}
	emails = mergeNotifyEmailsUnique(emails)
	if len(emails) > 0 {
		outgoing["assignee_emails"] = emails
	}
	var phones []string
	if s.assigneeSvc != nil && (status != "resolved" || s.assigneeSvc.NotifyOnResolvedEnabled(ctx, rid)) {
		p, _ := s.assigneeSvc.ResolveNotifyPhones(ctx, rid)
		phones = append(phones, p...)
	}
	if s.dutySvc != nil {
		p, _ := s.dutySvc.ResolveNotifyPhonesAtRule(ctx, rid, time.Now())
		phones = append(phones, p...)
	}
	phones = mergeNotifyPhonesUnique(phones)
	if len(phones) > 0 {
		outgoing["assignee_phones"] = phones
	}
}

