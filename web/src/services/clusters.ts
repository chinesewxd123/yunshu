import { getData, http } from "./http";

export interface ClusterItem {
  id: number;
  name: string;
  /** 可选归属项目；非空时按项目成员隔离可见性 */
  owning_project_id?: number | null;
  connection_mode?: "kubeconfig" | "direct";
  kubeconfig?: string;
  /** 后端不回传明文 kubeconfig，仅标识是否已配置 */
  kubeconfig_configured?: boolean;
  direct_config?: DirectConfigPayload;
  status: number;
  created_at: string;
  updated_at: string;
}

export interface ClusterListResponse {
  list: ClusterItem[];
  total: number;
  page: number;
  page_size: number;
}

export interface DirectConfigPayload {
  server?: string;
  insecure_skip_tls_verify?: boolean;
  ca_data?: string;
  token?: string;
  username?: string;
  password?: string;
  client_cert_data?: string;
  client_key_data?: string;
  dict_config_key?: string;
}

export interface ClusterCreatePayload {
  name: string;
  owning_project_id?: number;
  connection_mode?: "kubeconfig" | "direct";
  kubeconfig?: string;
  direct_config?: DirectConfigPayload;
}

export interface ClusterUpdatePayload {
  name?: string;
  /** 传 0 表示清空归属项目 */
  owning_project_id?: number;
  connection_mode?: "kubeconfig" | "direct";
  kubeconfig?: string;
  direct_config?: DirectConfigPayload;
}

export interface ClusterStatusResponse {
  server_version: string;
  connection_state?: string;
  last_error?: string;
  last_attempt_at?: string;
  last_success_at?: string;
  consecutive_failures?: number;
}

export interface NamespaceItem {
  name: string;
  phase: string;
}

export interface PodItem {
  name: string;
  namespace: string;
  phase: string;
  node_name: string;
  ready: boolean;
  start_time: string;
}

export interface ComponentStatusItem {
  name: string;
  status: string;
  healthy: boolean;
  message?: string;
  error?: string;
  last_probe_at?: string;
}

export function getClusters(query: { keyword?: string; page?: number; page_size?: number }) {
  return getData<ClusterListResponse>(http.get("/clusters", { params: query }));
}

export function getClusterDetail(id: number) {
  return getData<ClusterItem>(http.get(`/clusters/${id}`));
}

export function createCluster(payload: ClusterCreatePayload) {
  return getData<ClusterItem>(http.post("/clusters", payload));
}

export function updateCluster(id: number, payload: ClusterUpdatePayload) {
  return getData<ClusterItem>(http.put(`/clusters/${id}`, payload));
}

export function deleteCluster(id: number) {
  return getData<void>(http.delete(`/clusters/${id}`));
}

export function getClusterStatus(id: number) {
  return getData<ClusterStatusResponse>(http.get(`/clusters/${id}/status`));
}

export function setClusterStatus(id: number, status: 0 | 1) {
  return getData<ClusterItem>(http.put(`/clusters/${id}/status`, { status }));
}

export function listNamespaces(id: number) {
  return getData<{ list: NamespaceItem[] }>(http.get(`/clusters/${id}/namespaces`));
}

export function listPods(id: number, namespace: string) {
  return getData<{ list: PodItem[] }>(http.get("/pods", { params: { cluster_id: id, namespace } }));
}

export function listComponentStatuses(id: number) {
  return getData<{ list: ComponentStatusItem[] }>(http.get(`/clusters/${id}/component-statuses`));
}

