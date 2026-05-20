package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"yunshu/internal/alertdispatch"
	"yunshu/internal/model"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/service/svcerr"
)

func (s *AlertService) ListEvents(ctx context.Context, q AlertEventListQuery) (list []model.AlertEvent, total int64, page int, pageSize int, err error) {
	page, pageSize = pagination.Normalize(q.Page, q.PageSize)
	tx := s.db.WithContext(ctx).Model(&model.AlertEvent{})
	if kw := strings.TrimSpace(q.Keyword); kw != "" {
		like := "%" + kw + "%"
		tx = tx.Where("title LIKE ? OR error_message LIKE ? OR channel_name LIKE ?", like, like, like)
	}
	if v := strings.TrimSpace(q.Cluster); v != "" {
		tx = tx.Where("cluster = ?", v)
	}
	if v := strings.TrimSpace(q.AlertIP); v != "" {
		like := "%" + v + "%"
		tx = tx.Where(
			"cluster = ? OR request_payload LIKE ? OR request_payload LIKE ? OR request_payload LIKE ? OR request_payload LIKE ? OR request_payload LIKE ?",
			v,
			"%\"instance\":\""+v+"\"%",
			"%\"pod_ip\":\""+v+"\"%",
			"%\"node\":\""+v+"\"%",
			"%\"ip\":\""+v+"\"%",
			like,
		)
	}
	if v := strings.ToLower(strings.TrimSpace(q.Status)); v != "" {
		tx = tx.Where("status = ?", v)
	}
	if v := strings.TrimSpace(q.MonitorPipeline); v != "" {
		tx = tx.Where("monitor_pipeline = ?", v)
	}
	if q.DatasourceID > 0 {
		tx = tx.Where("datasource_id = ?", q.DatasourceID)
	}
	if q.ProjectID > 0 {
		tx = applyAlertEventProjectFilter(tx, s.db, q.ProjectID)
	}
	if v := strings.TrimSpace(q.GroupKey); v != "" {
		tx = tx.Where("group_key = ?", v)
	}
	if v := strings.TrimSpace(q.Category); v != "" {
		if ValidAlertEventCategory(v) {
			tx = applyAlertEventCategoryFilter(tx, v)
		}
	}
	if err = tx.Count(&total).Error; err != nil {
		return nil, 0, page, pageSize, svcerr.Pass(ctx, "alert", "ListEvents", err)
	}
	if err = tx.
		Order("id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&list).Error; err != nil {
		return nil, 0, page, pageSize, svcerr.Pass(ctx, "alert", "ListEvents", err)
	}
	for i := range list {
		hydrateAlertEvent(&list[i])
	}
	s.backfillResolvedAlertIP(ctx, list)
	return list, total, page, pageSize, nil
}

func (s *AlertService) backfillResolvedAlertIP(ctx context.Context, list []model.AlertEvent) {
	if len(list) == 0 {
		return
	}
	missing := map[string]struct{}{}
	for i := range list {
		st := strings.ToLower(strings.TrimSpace(list[i].Status))
		if st != "resolved" {
			continue
		}
		if strings.TrimSpace(list[i].AlertIP) != "" {
			continue
		}
		gk := strings.TrimSpace(list[i].GroupKey)
		if gk == "" {
			continue
		}
		missing[gk] = struct{}{}
	}
	if len(missing) == 0 {
		return
	}
	groupKeys := make([]string, 0, len(missing))
	for k := range missing {
		groupKeys = append(groupKeys, k)
	}
	var firingRows []model.AlertEvent
	if err := s.db.WithContext(ctx).
		Where("group_key IN ? AND status = ?", groupKeys, "firing").
		Order("id DESC").
		Find(&firingRows).Error; err != nil {
		return
	}
	ipByGroup := map[string]string{}
	for i := range firingRows {
		row := firingRows[i]
		gk := strings.TrimSpace(row.GroupKey)
		if gk == "" {
			continue
		}
		if _, ok := ipByGroup[gk]; ok {
			continue
		}
		hydrateAlertEvent(&row)
		ip := strings.TrimSpace(row.AlertIP)
		if ip != "" {
			ipByGroup[gk] = ip
		}
	}
	for i := range list {
		if strings.ToLower(strings.TrimSpace(list[i].Status)) != "resolved" || strings.TrimSpace(list[i].AlertIP) != "" {
			continue
		}
		if ip := strings.TrimSpace(ipByGroup[strings.TrimSpace(list[i].GroupKey)]); ip != "" {
			list[i].AlertIP = ip
		}
	}
}

func extractAlertIPFromPayload(requestPayload, fallback string) string {
	labels, _ := extractAlertPayloadLabels(requestPayload)
	for _, key := range []string{"instance", "pod_ip", "ip", "node"} {
		v := strings.TrimSpace(labels[key])
		if v != "" {
			return v
		}
	}
	return strings.TrimSpace(fallback)
}

func extractAlertStartedAtFromPayload(requestPayload string) string {
	raw := strings.TrimSpace(requestPayload)
	if raw == "" {
		return ""
	}
	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return ""
	}
	for _, key := range []string{"startsAt"} {
		v := strings.TrimSpace(fmt.Sprintf("%v", payload[key]))
		if v != "" && v != "<nil>" {
			return v
		}
	}
	return ""
}

func extractAlertPayloadLabels(requestPayload string) (map[string]string, map[string]interface{}) {
	raw := strings.TrimSpace(requestPayload)
	if raw != "" {
		var payload map[string]interface{}
		if err := json.Unmarshal([]byte(raw), &payload); err == nil {
			if labelsAny, ok := payload["labels"]; ok {
				if labels, ok := labelsAny.(map[string]interface{}); ok {
					out := make(map[string]string, len(labels))
					for key, value := range labels {
						v := strings.TrimSpace(fmt.Sprintf("%v", value))
						if v != "" && v != "<nil>" {
							out[key] = v
						}
					}
					return out, payload
				}
			}
			// 非 labels 结构（如钉钉/企微下发体）也返回 payload，
			// 便于从 atMobiles/mentioned_mobile_list 等字段提取接收人。
			return map[string]string{}, payload
		}
	}
	return map[string]string{}, nil
}

func parseUintCSV(raw string) []uint {
	var out []uint
	for _, part := range strings.Split(strings.TrimSpace(raw), ",") {
		n, ok := parseLabelUint(part)
		if ok && n > 0 {
			out = append(out, n)
		}
	}
	return out
}

func parseTrimmedCSV(raw string) []string {
	var out []string
	for _, part := range strings.Split(strings.TrimSpace(raw), ",") {
		v := strings.TrimSpace(part)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func parseStringListAny(v interface{}) []string {
	raw := normalizeRecipientList(v)
	var out []string
	for _, one := range raw {
		s := strings.TrimSpace(one)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func payloadValueByPath(payload map[string]interface{}, path string) interface{} {
	if payload == nil {
		return nil
	}
	cur := interface{}(payload)
	for _, part := range strings.Split(strings.TrimSpace(path), ".") {
		key := strings.TrimSpace(part)
		if key == "" {
			continue
		}
		obj, ok := cur.(map[string]interface{})
		if !ok {
			return nil
		}
		cur, ok = obj[key]
		if !ok {
			return nil
		}
	}
	return cur
}

func uniqTrimmedStrings(in []string, lower bool) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, one := range in {
		s := strings.TrimSpace(one)
		if s == "" {
			continue
		}
		k := s
		if lower {
			k = strings.ToLower(s)
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, s)
	}
	return out
}

func extractEventReceivers(requestPayload, channelName string) []string {
	_, payload := extractAlertPayloadLabels(requestPayload)
	if payload == nil {
		return nil
	}
	ch := strings.ToLower(strings.TrimSpace(channelName))
	var out []string
	if strings.Contains(ch, "email") || strings.Contains(ch, "邮件") {
		for _, key := range []string{"assignee_emails", "to", "recipients", "emails"} {
			out = append(out, parseStringListAny(payloadValueByPath(payload, key))...)
		}
		return uniqTrimmedStrings(out, true)
	}
	if strings.Contains(ch, "ding") || strings.Contains(ch, "钉") || strings.Contains(ch, "wecom") || strings.Contains(ch, "wechat") || strings.Contains(ch, "企微") || strings.Contains(ch, "企业微信") {
		for _, key := range []string{"at.atMobiles", "text.mentioned_mobile_list", "atMobiles", "mentioned_mobile_list"} {
			out = append(out, parseStringListAny(payloadValueByPath(payload, key))...)
		}
		return uniqTrimmedStrings(out, false)
	}
	return nil
}

func fillMetricFieldsFromRequestPayload(ev *model.AlertEvent) {
	if ev == nil {
		return
	}
	raw := strings.TrimSpace(ev.RequestPayload)
	if raw == "" {
		return
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return
	}
	norm := func(v interface{}) string {
		s := strings.TrimSpace(fmt.Sprintf("%v", v))
		if s == "" || s == "<nil>" {
			return ""
		}
		return s
	}
	ev.MetricCurrent = norm(m["current"])
	ev.MetricResolved = norm(m["current_resolved"])
	if ev.MetricCurrent == "-" {
		ev.MetricCurrent = ""
	}
	if ev.MetricResolved == "-" {
		ev.MetricResolved = ""
	}
}

func hydrateAlertEvent(it *model.AlertEvent) {
	if it == nil {
		return
	}
	labels, _ := extractAlertPayloadLabels(it.RequestPayload)
	it.Environment = strings.TrimSpace(it.Cluster)
	if rawCluster := strings.TrimSpace(labels["cluster"]); rawCluster != "" {
		it.Cluster = rawCluster
	}
	it.AlertIP = extractAlertIPFromPayload(it.RequestPayload, it.Environment)
	it.AlertStartedAt = extractAlertStartedAtFromPayload(it.RequestPayload)
	it.MatchedPolicyIDList = parseUintCSV(it.MatchedPolicyIDs)
	it.MatchedPolicyNameList = parseTrimmedCSV(it.MatchedPolicyNames)
	it.ReceiverList = extractEventReceivers(it.RequestPayload, it.ChannelName)
	fillMetricFieldsFromRequestPayload(it)
}

func previewPayloadFieldCatalog(payload map[string]interface{}) ([]string, []string, []string) {
	rawFields := make([]string, 0)
	labelKeys := make([]string, 0)
	for key := range payload {
		rawFields = append(rawFields, key)
	}
	sort.Strings(rawFields)
	if labelsAny, ok := payload["labels"]; ok {
		if labels, ok := labelsAny.(map[string]interface{}); ok {
			for key := range labels {
				labelKeys = append(labelKeys, key)
			}
		}
	}
	sort.Strings(labelKeys)
	combined := append([]string{}, alertdispatch.ChannelTemplateFieldList()...)
	combined = append(combined, rawFields...)
	sort.Strings(combined)
	return rawFields, combined, labelKeys
}

// 钉钉 markdown：正文里需出现 @手机号 / @userid / @all；企业微信仅用 mentioned_* 即可，勿再拼此段以免双重 @。
func atNotifyPlainMentionsFooter(atMobiles, atUserIds []string, isAtAll bool) string {
	var parts []string
	if isAtAll {
		parts = append(parts, "@all")
	}
	for _, m := range atMobiles {
		m = strings.TrimSpace(m)
		if m != "" {
			parts = append(parts, "@"+m)
		}
	}
	for _, u := range atUserIds {
		u = strings.TrimSpace(u)
		if u != "" {
			parts = append(parts, "@"+u)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "\n\n" + strings.Join(parts, " ")
}

func appendDingTalkMarkdownText(body map[string]interface{}, extra string) {
	if body == nil || strings.TrimSpace(extra) == "" {
		return
	}
	if mm, ok := body["markdown"].(map[string]string); ok {
		nm := map[string]string{}
		for k, v := range mm {
			nm[k] = v
		}
		nm["text"] = nm["text"] + extra
		body["markdown"] = nm
		return
	}
	if mm, ok := body["markdown"].(map[string]interface{}); ok {
		nm := map[string]interface{}{}
		for k, v := range mm {
			nm[k] = v
		}
		prev := strings.TrimSpace(fmt.Sprintf("%v", nm["text"]))
		nm["text"] = prev + extra
		body["markdown"] = nm
	}
}

