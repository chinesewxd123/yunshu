import type { AlertEventItem } from "../services/alerts";

/**
 * 与后端 `alert_events.error_message` 及入站留痕逻辑对齐。
 * 参考：internal/service/alert_ingest_canonical.go、alert_aggregate_state.go、alert_inhibition_integration.go
 */

export type AlertEventCategory =
  | "delivery"
  | "routing"
  | "silence"
  | "inhibition"
  | "timing"
  | "resolved"
  | "failure"
  | "other";

export const ALERT_EVENT_CATEGORY_OPTIONS: { label: string; value: AlertEventCategory }[] = [
  { label: "告警抑制", value: "inhibition" },
  { label: "平台/订阅静默", value: "silence" },
  { label: "通知合并", value: "timing" },
  { label: "恢复策略", value: "resolved" },
  { label: "订阅/路由", value: "routing" },
  { label: "成功外发", value: "delivery" },
  { label: "发送失败", value: "failure" },
  { label: "其它留痕", value: "other" },
];

export type AlertEventReasonMeta = {
  /** 列表「说明」列短文案 */
  hint: string;
  /** 详情/Popover 长文案 */
  detail: string;
  /** 策略分类，便于筛选展示 */
  category: AlertEventCategory;
};

const EXACT: Record<string, AlertEventReasonMeta> = {
  silence_suppressed: {
    category: "silence",
    hint: "平台静默",
    detail: "命中平台静默规则（alert_silences），入站后不再向任何通道外发，本条为审计留痕。",
  },
  subscription_suppressed: {
    category: "silence",
    hint: "订阅静默窗口",
    detail: "命中订阅节点的 silence_seconds：同一 groupKey + 订阅节点在静默窗口内不重复外发。",
  },
  group_wait_suppressed: {
    category: "timing",
    hint: "首次同组等待",
    detail:
      "decideFiringGroupTiming：首次见到该 groupKey 后，在 alert.group_wait_seconds 内聚合同组告警，窗口结束前不向渠道推送。",
  },
  group_interval_suppressed: {
    category: "timing",
    hint: "同组变化间隔",
    detail:
      "通知合并 digest 仅基于 alert.group_by + alert.digest_by 维度（非全量 Prometheus labels）。摘要相对上次成功通知有变化，但未满 alert.group_interval_seconds（默认 60s）则不推送；摘要不变则按 repeat_interval_seconds（默认 300s）重复提醒。",
  },
  repeat_suppressed: {
    category: "timing",
    hint: "重复提醒间隔",
    detail: "告警持续 firing 且无标签摘要变化，距上次成功通知未满 alert.repeat_interval_seconds，本轮不重复打扰。",
  },
  group_throttled: {
    category: "timing",
    hint: "聚合限流（遗留码）",
    detail: "历史聚合限流留痕码；当前 firing 节流以 group_wait / group_interval / repeat 为准。",
  },
  resolved_aggregate_suppressed: {
    category: "resolved",
    hint: "重复恢复已抑制",
    detail: "同一 fingerprint 的恢复通知仅发送一次（markResolvedNotificationSent），本条为重复 resolved 留痕。",
  },
  resolved_no_prior_firing_delivery: {
    category: "resolved",
    hint: "无成功触发投递",
    detail:
      "恢复事件到达，但该 fingerprint 从未成功向通道投递过 firing（可能此前被分组节流或通道失败），已抑制恢复外发。",
  },
  no_policy_matched: {
    category: "routing",
    hint: "未命中订阅",
    detail: "订阅树未匹配到接收组（含项目订阅与 project_id=0 全局订阅）；不会向通道发送。",
  },
  no_enabled_channels: {
    category: "routing",
    hint: "无启用通道",
    detail: "系统中没有启用的通知通道，仅写入历史。",
  },
  no_channel_matched: {
    category: "routing",
    hint: "通道标签不匹配",
    detail: "已命中订阅与接收组，但通道级 match_labels 过滤后无可用通道。",
  },
  no_channel_matched_subscription: {
    category: "routing",
    hint: "订阅通道为空",
    detail: "订阅命中但接收组未绑定通道，或通道均未通过启用/时段校验。",
  },
  all_channel_delivery_failed: {
    category: "failure",
    hint: "全部通道失败",
    detail: "至少尝试向一个通道发送，但 HTTP/邮件/IM 均未返回 2xx。",
  },
};

function parseInhibitionSuppressed(reason: string): AlertEventReasonMeta | null {
  if (!reason.startsWith("inhibition_suppressed:")) return null;
  const rest = reason.slice("inhibition_suppressed:".length).trim();
  const [rulePart, sourcePart] = rest.split(/\s+source=/);
  const ruleName = rulePart.replace(/^rule=/, "").trim() || "未知规则";
  const sourceFP = sourcePart?.trim() || "-";
  return {
    category: "inhibition",
    hint: "告警抑制",
    detail: `命中抑制规则「${ruleName}」：存在活跃源告警（指纹 ${sourceFP}），且目标告警满足 target 匹配与 equal 标签约束，不向通道外发。需 Redis 可用。`,
  };
}

/** 根据 error_message 解析展示元数据；未知码回退为通用抑制/失败文案。 */
export function resolveAlertEventReason(errorMessage: string | undefined | null): AlertEventReasonMeta | null {
  const reason = String(errorMessage || "").trim();
  if (!reason) return null;
  if (EXACT[reason]) return EXACT[reason];
  const inh = parseInhibitionSuppressed(reason);
  if (inh) return inh;
  if (/suppressed/i.test(reason)) {
    return {
      category: "other",
      hint: "策略抑制",
      detail: `系统策略抑制（${reason}），本次未向渠道外发。`,
    };
  }
  return null;
}

export function summarizeAlertEventHint(row: AlertEventItem): string {
  if (!row.success) return "-";
  const meta = resolveAlertEventReason(row.errorMessage);
  if (meta) return meta.hint;
  const reason = String(row.errorMessage || "").trim();
  if (!reason) return "-";
  return reason.length > 24 ? `${reason.slice(0, 24)}…` : reason;
}

export function describeAlertEvent(row: AlertEventItem): string {
  const reason = String(row.errorMessage || "").trim();
  const channelText = String(row.channelName || "").trim() || "未匹配通道";
  const receiverText = row.receiverList?.length ? row.receiverList.join(", ") : "-";
  if (row.success) {
    const meta = resolveAlertEventReason(reason);
    if (meta) return meta.detail;
    if (row.channelName?.includes("静默抑制") || row.channelName?.includes("被抑制")) {
      return "平台在分发前拦截了本次告警，通道列展示抑制原因。";
    }
    return `通道[${channelText}] 已发送，接收人[${receiverText}]。`;
  }
  if (reason === "no_enabled_channels") return EXACT.no_enabled_channels.detail;
  if (reason === "no_policy_matched") return EXACT.no_policy_matched.detail;
  if (reason === "no_channel_matched" || reason === "no_channel_matched_subscription") {
    return EXACT[reason]?.detail ?? "有通道配置但本次告警未匹配到可发送通道。";
  }
  if (reason) return `通道[${channelText}] 发送失败，接收人[${receiverText}]，原因：${reason}`;
  return "告警已进入平台链路，但未获取到更多说明。";
}

/** 历史 Tab 顶部的策略说明（与当前后端实现一致）。 */
export const ALERT_HISTORY_PIPELINE_HELP = [
  {
    title: "订阅树路由",
    body: "channelIDSetForAlert：按 project_id（或 datasource 反查项目）+ 全局 project_id=0 订阅匹配接收组与通道；未命中则 no_policy_matched。",
  },
  {
    title: "平台静默",
    body: "AlertSilenceService：入站最早阶段拦截，error_message=silence_suppressed。",
  },
  {
    title: "告警抑制（inhibit）",
    body: "源告警 firing 且匹配 source 条件时写入 Redis；目标告警匹配 target 且 equal 标签与源一致则被抑制。需 Redis。历史码 inhibition_suppressed:rule=… source=…",
  },
  {
    title: "通知合并（firing）",
    body: "decideFiringGroupTiming：group_wait_suppressed / group_interval_suppressed / repeat_suppressed（读取 alert.group_* 配置）。",
  },
  {
    title: "恢复（resolved）",
    body: "无成功 firing 投递则 resolved_no_prior_firing_delivery；重复恢复 resolved_aggregate_suppressed；处理人恢复通知受 notify_on_resolved 控制。",
  },
] as const;
