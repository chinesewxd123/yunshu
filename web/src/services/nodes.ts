import { getData, http } from "./http";

export interface NodeItem {
  name: string;
  status: string;
  /** 为 true 时表示已 cordon，新 Pod 不会调度到该节点 */
  unschedulable?: boolean;
  roles?: string[];
  kernel: string;
  kubelet: string;
  os_image: string;
  container_runtime: string;
  architecture: string;
  internal_ip?: string;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  taints?: Array<{
    key: string;
    value?: string;
    effect?: string;
    time_added?: string;
  }>;
  creation_time: string;
  age?: string;
  pod_count?: number;
  pod_capacity?: number;
  pod_usage?: string;
  pod_usage_percent?: number;
  cpu_usage?: string;
  cpu_requests?: string;
  cpu_limits?: string;
  mem_usage?: string;
  mem_requests?: string;
  mem_limits?: string;
  cpu_usage_percent?: number;
  mem_usage_percent?: number;
}

export interface NodeAddressItem {
  type: string;
  address: string;
}

export interface NodeDetail {
  item: NodeItem;
  addresses: NodeAddressItem[];
  conditions: Array<{
    type: string;
    status: string;
    reason?: string;
    message?: string;
    last_heartbeat_time?: string;
    last_transition_time?: string;
  }>;
  taints: Array<{
    key: string;
    value?: string;
    effect?: string;
    time_added?: string;
  }>;
  capacity: Record<string, string>;
  allocatable: Record<string, string>;
  yaml: string;
}

export function listNodes(clusterId: number, keyword?: string) {
  return getData<NodeItem[]>(http.get("/nodes", { params: { cluster_id: clusterId, keyword } }));
}

export function getNodeDetail(clusterId: number, name: string) {
  return getData<NodeDetail>(http.get("/nodes/detail", { params: { cluster_id: clusterId, name } }));
}

export type NodeTaintInput = { key: string; value?: string; effect?: string };

export function setNodeSchedulability(clusterId: number, name: string, unschedulable: boolean) {
  return getData<{ ok: boolean }>(
    http.post("/nodes/schedulability", { cluster_id: clusterId, name, unschedulable }),
  );
}

export function replaceNodeTaints(clusterId: number, name: string, taints: NodeTaintInput[]) {
  return getData<{ ok: boolean }>(http.put("/nodes/taints", { cluster_id: clusterId, name, taints }));
}

