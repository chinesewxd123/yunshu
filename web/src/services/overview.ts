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

