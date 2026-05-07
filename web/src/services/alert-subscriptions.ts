import { getData, http } from "./http";

export interface AlertSubscriptionNode {
  id: number;
  project_id: number;
  parent_id?: number | null;
  level: number;
  path: string;
  name: string;
  code?: string;
  match_labels_json?: string;
  match_regex_json?: string;
  match_severity?: string;
  continue: boolean;
  enabled: boolean;
  receiver_group_ids_json?: string;
  silence_seconds: number;
  notify_resolved: boolean;
  receiver_group_ids?: number[];
  children?: AlertSubscriptionNode[];
  created_at?: string;
  updated_at?: string;
}

export interface AlertReceiverGroup {
  id: number;
  project_id: number;
  name: string;
  description?: string;
  channel_ids_json?: string;
  email_recipients_json?: string;
  active_time_start?: string | null;
  active_time_end?: string | null;
  weekdays_json?: string;
  escalation_level: number;
  enabled: boolean;
  channel_ids?: number[];
  email_recipients?: string[];
  weekdays?: number[];
  created_at?: string;
  updated_at?: string;
}

export function listSubscriptionNodes(params: { project_id: number; parent_id?: number; keyword?: string; enabled?: boolean; page?: number; page_size?: number }) {
  return getData<{ list?: AlertSubscriptionNode[]; items?: AlertSubscriptionNode[]; total: number; page: number; page_size: number }>(
    http.get("/alerts/subscriptions", { params }),
  );
}

/** 拉取订阅树；兼容旧版直接返回 JSON 数组、新版统一 { code, data }（axios 拦截器已解包为 body） */
export async function getSubscriptionTree(params: { project_id: number }): Promise<AlertSubscriptionNode[]> {
  const raw: unknown = await http.get("/alerts/subscriptions/tree", { params });
  if (Array.isArray(raw)) return raw as AlertSubscriptionNode[];
  if (raw !== null && typeof raw === "object" && "data" in raw) {
    const d = (raw as { data?: unknown }).data;
    if (Array.isArray(d)) return d as AlertSubscriptionNode[];
  }
  return [];
}

export function createSubscriptionNode(payload: Partial<AlertSubscriptionNode> & { project_id: number; name: string }) {
  return getData<AlertSubscriptionNode>(http.post("/alerts/subscriptions", payload));
}

export function updateSubscriptionNode(id: number, payload: Partial<AlertSubscriptionNode> & { project_id: number; name: string }) {
  return getData<AlertSubscriptionNode>(http.put(`/alerts/subscriptions/${id}`, payload));
}

export function deleteSubscriptionNode(id: number) {
  return getData<void>(http.delete(`/alerts/subscriptions/${id}`));
}

export function moveSubscriptionNode(id: number, payload: { new_parent_id?: number | null }) {
  return getData<AlertSubscriptionNode>(http.post(`/alerts/subscriptions/${id}/move`, payload));
}

export function migratePoliciesToSubscriptions(payload?: {
  disable_old?: boolean;
  /** 未在 match_labels 中写 project_id 的策略归入该项目；不传则由服务端取首个启用项目 */
  default_project_id?: number;
}) {
  return getData<{
    policies_total: number;
    policies_migrated: number;
    receiver_groups_created: number;
    nodes_created: number;
    policies_disabled: number;
    resolved_default_project_id?: number;
  }>(http.post("/alerts/subscriptions/migrate-from-policies", payload ?? { disable_old: true }));
}

export function listReceiverGroups(params: { project_id?: number; keyword?: string; enabled?: boolean; page?: number; page_size?: number }) {
  return getData<{ list?: AlertReceiverGroup[]; items?: AlertReceiverGroup[]; total: number; page: number; page_size: number }>(
    http.get("/alerts/receiver-groups", { params }),
  );
}

export function createReceiverGroup(payload: Partial<AlertReceiverGroup> & { project_id: number; name: string }) {
  return getData<AlertReceiverGroup>(http.post("/alerts/receiver-groups", payload));
}

export function updateReceiverGroup(id: number, payload: Partial<AlertReceiverGroup> & { project_id: number; name?: string }) {
  return getData<AlertReceiverGroup>(http.put(`/alerts/receiver-groups/${id}`, payload));
}

export function deleteReceiverGroup(id: number) {
  return getData<void>(http.delete(`/alerts/receiver-groups/${id}`));
}

