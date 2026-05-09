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

export type ApplyNamespaceOptions = {
  /** 为 true 时若集群中已有同名 Namespace 则拒绝（表单创建）；YAML 更新场景勿传 */
  failIfExists?: boolean;
  /** 为 true 时不弹出全局错误 toast，由调用方用 extractApiErrorMessage 展示（避免重复提示） */
  silentErrorToast?: boolean;
};

export function applyNamespace(clusterId: number, manifest: string, opts?: ApplyNamespaceOptions) {
  const failIfExists = opts?.failIfExists;
  const silent = opts?.silentErrorToast;
  return getData<boolean>(
    http.post(
      "/namespaces/apply",
      {
        cluster_id: clusterId,
        manifest,
        ...(failIfExists ? { fail_if_exists: true } : {}),
      },
      silent ? { silentErrorToast: true } : {},
    ),
  );
}

export function deleteNamespace(clusterId: number, name: string) {
  return getData<boolean>(http.delete("/namespaces", { params: { cluster_id: clusterId, name } }));
}

