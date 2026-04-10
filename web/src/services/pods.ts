import { getData, http } from "./http";

export interface PodItem {
  name: string;
  namespace: string;
  phase: string;
  node_name: string;
  ready: boolean;
  start_time: string;
  pod_ip: string;
  host_ip: string;
  qos_class: string;
  restart_count: number;
  images: string[];
}

export interface PodListQuery {
  cluster_id: number;
  namespace?: string;
  keyword?: string;
}

export interface PodLogsQuery {
  cluster_id: number;
  namespace: string;
  name: string;
  container?: string;
  tail_lines?: number;
  follow?: boolean;
  keyword?: string;
  start_time?: string;
  end_time?: string;
}

export interface PodDeletePayload {
  cluster_id: number;
  namespace: string;
  name: string;
}

export interface PodExecPayload {
  cluster_id: number;
  namespace: string;
  name: string;
  container?: string;
  command: string;
}
export interface PodCreateYAMLPayload {
  cluster_id: number;
  namespace: string;
  manifest: string;
}
export interface PodCreateSimplePayload {
  cluster_id: number;
  namespace: string;
  name: string;
  image: string;
  command?: string;
  container_name?: string;
  image_pull_policy?: "Always" | "IfNotPresent" | "Never";
  restart_policy?: "Always" | "OnFailure" | "Never";
  port?: number;
  env?: Record<string, string>;
  labels?: Record<string, string>;
  requests_cpu?: string;
  requests_memory?: string;
  limits_cpu?: string;
  limits_memory?: string;
  tolerations?: Array<{
    key?: string;
    operator?: "Equal" | "Exists";
    value?: string;
    effect?: "NoSchedule" | "PreferNoSchedule" | "NoExecute";
    toleration_seconds?: number;
  }>;
  node_selector?: Record<string, string>;
  priority_class_name?: string;
  affinity?: Record<string, unknown>;
}

export interface PodContainerInfo {
  name: string;
  image: string;
  ready: boolean;
  restart_count: number;
  state: string;
}

export interface PodDetail {
  name: string;
  namespace: string;
  uid: string;
  phase: string;
  node_name: string;
  service_account: string;
  pod_ip: string;
  host_ip: string;
  qos_class: string;
  labels: Record<string, string>;
  annotations: Record<string, string>;
  containers: PodContainerInfo[];
  init_containers: PodContainerInfo[];
  start_time: string;
  creation_time: string;
  volumes: Array<{
    name: string;
    configMap?: { name?: string };
    secret?: { secretName?: string };
    persistentVolumeClaim?: { claimName?: string };
    emptyDir?: Record<string, unknown>;
    hostPath?: { path?: string };
    [key: string]: unknown;
  }>;
  tolerations?: Array<{
    key?: string;
    operator?: "Equal" | "Exists";
    value?: string;
    effect?: "NoSchedule" | "PreferNoSchedule" | "NoExecute";
    tolerationSeconds?: number;
  }>;
  node_selector?: Record<string, string>;
  priority_class_name?: string;
  affinity?: Record<string, unknown>;
}

export interface PodEventItem {
  type: string;
  reason: string;
  message: string;
  count: number;
  first_timestamp: string;
  last_timestamp: string;
}

export interface PodFileQuery {
  cluster_id: number;
  namespace: string;
  name: string;
  container?: string;
  path?: string;
}

export interface PodFileItem {
  name: string;
  path: string;
  type: string;
  is_dir: boolean;
  size: number;
  permissions: string;
  owner: string;
  group: string;
  mod_time: string;
}

export function getPods(query: PodListQuery) {
  return getData<{ list: PodItem[] }>(http.get("/pods", { params: query }));
}

export function getPodLogs(query: PodLogsQuery) {
  return getData<{ logs: string }>(http.get("/pods/logs", { params: query }));
}
export async function downloadPodLogs(query: PodLogsQuery) {
  const blob = (await http.get("/pods/logs/download", { params: query, responseType: "blob" })) as unknown as Blob;
  return blob;
}

export function getPodDetail(query: { cluster_id: number; namespace: string; name: string }) {
  return getData<PodDetail>(http.get("/pods/detail", { params: query }));
}

export function getPodEvents(query: { cluster_id: number; namespace: string; name: string }) {
  return getData<{ list: PodEventItem[] }>(http.get("/pods/events", { params: query }));
}

export function listPodFiles(query: PodFileQuery) {
  return getData<{ list: PodFileItem[] }>(http.get("/pods/files", { params: query }));
}

export function readPodFile(query: PodFileQuery) {
  return getData<{ content: string }>(http.get("/pods/file", { params: query }));
}

export async function downloadPodFile(query: PodFileQuery) {
  const blob = (await http.get("/pods/file/download", { params: query, responseType: "blob" })) as unknown as Blob;
  return blob;
}

export function deletePodFile(payload: PodFileQuery) {
  return getData<{ message: string }>(http.post("/pods/file/delete", payload));
}

export function uploadPodFile(payload: PodFileQuery & { file: File }) {
  const form = new FormData();
  form.append("cluster_id", String(payload.cluster_id));
  form.append("namespace", payload.namespace);
  form.append("name", payload.name);
  if (payload.container) form.append("container", payload.container);
  if (payload.path) form.append("path", payload.path);
  form.append("file", payload.file);
  return getData<{ message: string }>(
    http.post("/pods/file/upload", form, {
      headers: { "Content-Type": "multipart/form-data" },
    }),
  );
}

export function deletePod(payload: PodDeletePayload) {
  return getData<{ message: string }>(http.delete("/pods", { data: payload }));
}

export function execPod(payload: PodExecPayload) {
  return getData<{ output: string }>(http.post("/pods/exec", payload));
}
export function restartPod(payload: { cluster_id: number; namespace: string; name: string }) {
  return getData<{ message: string }>(http.post("/pods/restart", payload));
}
export function createPodByYAML(payload: PodCreateYAMLPayload) {
  return getData<{ message: string }>(http.post("/pods/create/yaml", payload));
}
export function createPodSimple(payload: PodCreateSimplePayload) {
  return getData<{ message: string }>(http.post("/pods/create/simple", payload));
}
export function updatePodSimple(payload: PodCreateSimplePayload) {
  return getData<{ message: string }>(http.post("/pods/update/simple", payload));
}
