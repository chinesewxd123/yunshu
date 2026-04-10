import { getData, http } from "./http";

export interface OverviewResponse {
  users_count: number;
  clusters_count: number;
  pod_normal_count: number;
  pod_abnormal_count: number;
  pod_cluster_errors: number;
  event_total_count: number;
  event_warning_count: number;
  event_cluster_errors: number;
}

export function getOverview() {
  return getData<OverviewResponse>(http.get("/overview"));
}

