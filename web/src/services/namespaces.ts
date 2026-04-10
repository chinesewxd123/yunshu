import { getData, http } from "./http";

export interface NamespaceItem {
  name: string;
  status: string;
  creation_time: string;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  pod_count?: number;
  cpu_requests?: string;
  cpu_limits?: string;
  mem_requests?: string;
  mem_limits?: string;
}

export interface NamespaceDetail {
  item: NamespaceItem;
  finalizers?: string[];
  resource_quotas?: Array<{
    name: string;
    hard?: Record<string, string>;
    used?: Record<string, string>;
    scope?: string[];
  }>;
  limit_ranges?: Array<{
    name: string;
    limits?: Array<{
      type?: string;
      max?: Record<string, string>;
      min?: Record<string, string>;
      default?: Record<string, string>;
      defaultRequest?: Record<string, string>;
      maxLimitRequestRatio?: Record<string, string>;
    }>;
  }>;
  recent_events?: Array<{
    type: string;
    reason: string;
    message: string;
    last_time?: string;
    count: number;
  }>;
  yaml: string;
}

export function listNamespaces(clusterId: number, keyword?: string) {
  return getData<NamespaceItem[]>(http.get("/namespaces", { params: { cluster_id: clusterId, keyword } }));
}

export function getNamespaceDetail(clusterId: number, name: string) {
  return getData<NamespaceDetail>(http.get("/namespaces/detail", { params: { cluster_id: clusterId, name } }));
}

export function applyNamespace(clusterId: number, manifest: string) {
  return getData<boolean>(http.post("/namespaces/apply", { cluster_id: clusterId, manifest }));
}

export function deleteNamespace(clusterId: number, name: string) {
  return getData<boolean>(http.delete("/namespaces", { params: { cluster_id: clusterId, name } }));
}

