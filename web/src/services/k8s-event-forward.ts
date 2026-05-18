import { getData, http } from "./http";

export interface K8sEventForwardRule {
  id: number;
  name: string;
  description?: string;
  cluster_ids: string;
  webhook_url?: string;
  enabled: boolean;
  rule_namespaces?: string;
  rule_names?: string;
  rule_reasons?: string;
  rule_reverse?: boolean;
  created_at?: string;
  updated_at?: string;
}

export type K8sEventForwardRulePayload = {
  name: string;
  description?: string;
  cluster_ids: string;
  webhook_url?: string;
  enabled?: boolean;
  rule_namespaces?: string;
  rule_names?: string;
  rule_reasons?: string;
  rule_reverse?: boolean;
};

export interface K8sEventForwardSetting {
  id: number;
  process_interval_seconds: number;
  batch_size: number;
  max_retries: number;
  watcher_buffer_size: number;
}

export function listK8sEventForwardRules(params?: { page?: number; page_size?: number }) {
  return getData<{ list: K8sEventForwardRule[]; total: number; page: number; page_size: number }>(
    http.get("/k8s/event-forward/rules", { params }),
  );
}

export function getK8sEventForwardRule(id: number) {
  return getData<K8sEventForwardRule>(http.get(`/k8s/event-forward/rules/${id}`));
}

export function createK8sEventForwardRule(payload: K8sEventForwardRulePayload) {
  return getData<K8sEventForwardRule>(http.post("/k8s/event-forward/rules", payload));
}

export function updateK8sEventForwardRule(id: number, payload: K8sEventForwardRulePayload) {
  return getData<K8sEventForwardRule>(http.put(`/k8s/event-forward/rules/${id}`, payload));
}

export function deleteK8sEventForwardRule(id: number) {
  return getData<{ deleted: boolean }>(http.delete(`/k8s/event-forward/rules/${id}`));
}

export function getK8sEventForwardSettings() {
  return getData<K8sEventForwardSetting>(http.get("/k8s/event-forward/settings"));
}

export function updateK8sEventForwardSettings(payload: K8sEventForwardSetting) {
  return getData<K8sEventForwardSetting>(http.put("/k8s/event-forward/settings", payload));
}
