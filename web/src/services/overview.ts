import { getData, http } from "./http";

export interface OverviewResponse {
  users_count: number;
  clusters_count: number;
  pending_registrations_count: number;
  servers_count: number;
  pod_normal_count: number;
  pod_abnormal_count: number;
  pod_cluster_errors: number;
  event_total_count: number;
  event_warning_count: number;
  event_cluster_errors: number;
  /** 当前 firing 状态告警事件数 */
  alert_firing_count: number;
  /** 本自然日新建的告警事件条数 */
  alert_events_today_count: number;
  /** 日志 Agent 在线（心跳 90s 内） */
  log_agents_online_count: number;
  /** 启用 Agent 总数减去在线 */
  log_agents_offline_count: number;
}

export interface OverviewTrendsResponse {
  days: string[];
  login_success: number[];
  login_fail: number[];
  operation_total: number[];
}

export function getOverview() {
  return getData<OverviewResponse>(http.get("/overview", { silentErrorToast: true }));
}

export function getOverviewTrends() {
  return getData<OverviewTrendsResponse>(http.get("/overview/trends", { silentErrorToast: true }));
}

