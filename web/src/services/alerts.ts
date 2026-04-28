import { getData, http } from "./http";
import { normalizePagedPayload, parseCommaSeparatedList, parseCommaSeparatedNumbers, parseNumberArray, parseStringMap } from "./alert-mappers";

export interface AlertChannelItem {
  id: number;
  name: string;
  type: string;
  url: string;
  secret?: string;
  headers_json?: string;
  enabled: boolean;
  timeout_ms: number;
  created_at: string;
  updated_at: string;
}

export interface AlertEventItem {
  id: number;
  source: string;
  title: string;
  severity: string;
  status: string;
  cluster?: string;
  alert_ip?: string;
  /** prometheus = Prometheus 规则+Alertmanager Webhook；platform = 平台内监控规则 */
  monitor_pipeline?: string;
  group_key?: string;
  labels_digest?: string;
  matched_policy_ids?: string;
  matched_policy_names?: string;
  matched_policy_id_list?: number[];
  matched_policy_name_list?: string[];
  receiver_list?: string[];
  channel_id: number;
  channel_name: string;
  success: boolean;
  http_status_code: number;
  error_message?: string;
  request_payload?: string;
  response_payload?: string;
  created_at: string;
}

export interface AlertPolicyItem {
  id: number;
  name: string;
  description?: string;
  enabled: boolean;
  priority: number;
  match_labels_json?: string;
  match_regex_json?: string;
  channels_json?: string;
  match_labels?: Record<string, string>;
  match_regex?: Record<string, string>;
  channel_ids?: number[];
  template_id?: number;
  notify_resolved: boolean;
  silence_seconds: number;
  created_at: string;
  updated_at: string;
}

export function listAlertChannels(params?: { keyword?: string }) {
  return getData<{ list: AlertChannelItem[] }>(http.get("/alerts/channels", { params }));
}

export function createAlertChannel(payload: {
  name: string;
  type?: string;
  url: string;
  secret?: string;
  headers_json?: string;
  enabled?: boolean;
  timeout_ms?: number;
}) {
  return getData<AlertChannelItem>(http.post("/alerts/channels", payload));
}

export function updateAlertChannel(
  id: number,
  payload: {
    name: string;
    type?: string;
    url: string;
    secret?: string;
    headers_json?: string;
    enabled?: boolean;
    timeout_ms?: number;
  },
) {
  return getData<AlertChannelItem>(http.put(`/alerts/channels/${id}`, payload));
}

export function deleteAlertChannel(id: number) {
  return getData<void>(http.delete(`/alerts/channels/${id}`));
}

export function testAlertChannel(
  id: number,
  payload?: { title?: string; content?: string; severity?: string; status?: "firing" | "resolved" },
) {
  return getData<void>(http.post(`/alerts/channels/${id}/test`, payload ?? {}));
}

export interface AlertTemplatePreviewResult {
  rendered: string;
  sample_payload: Record<string, unknown>;
  available_fields: string[];
  raw_payload_fields: string[];
  combined_fields: string[];
  suggested_label_keys: string[];
}

export function previewAlertChannelTemplate(payload: {
  template_firing?: string;
  template_resolved?: string;
  status?: "firing" | "resolved";
  title?: string;
  content?: string;
  severity?: string;
  project_id?: number;
  raw_payload_json?: string;
}) {
  return getData<AlertTemplatePreviewResult>(http.post("/alerts/channels/preview-template", payload));
}

export function listAlertEvents(params: {
  page: number;
  page_size: number;
  keyword?: string;
  cluster?: string;
  alert_ip?: string;
  status?: string;
  monitor_pipeline?: string;
  group_key?: string;
}) {
  return getData<{ list?: AlertEventItem[]; items?: AlertEventItem[]; total: number; page: number; page_size: number }>(
    http.get("/alerts/events", { params }),
  ).then((payload) =>
    normalizePagedPayload(payload, (item) => ({
      ...item,
      matched_policy_id_list: parseCommaSeparatedNumbers(item.matched_policy_ids),
      matched_policy_name_list: parseCommaSeparatedList(item.matched_policy_names),
    })),
  );
}

export function getAlertHistoryStats() {
  return getData<{
    total: number;
    firing: number;
    resolved: number;
    success: number;
    failed: number;
    today_created: number;
    /** K8s / Prometheus external_labels.cluster 等（历史记录中去重） */
    cluster_values?: string[];
    /** prometheus、platform（历史记录中去重） */
    monitor_pipeline_values?: string[];
  }>(http.get("/alerts/history/stats"));
}

export function sendAlertmanagerWebhook(payload: Record<string, unknown>, token?: string) {
  const headers: Record<string, string> = {};
  if ((token || "").trim()) {
    headers["X-Webhook-Token"] = String(token).trim();
  }
  return getData<{ message: string }>(http.post("/alerts/webhook/alertmanager", payload, { headers }));
}

export function listAlertPolicies(params: { page: number; page_size: number; keyword?: string; enabled?: boolean }) {
  return getData<{ list?: AlertPolicyItem[]; items?: AlertPolicyItem[]; total: number; page: number; page_size: number }>(http.get("/alerts/policies", { params })).then(
    (payload) =>
      normalizePagedPayload(payload, (item) => ({
        ...item,
        match_labels: parseStringMap(item.match_labels_json),
        match_regex: parseStringMap(item.match_regex_json),
        channel_ids: parseNumberArray(item.channels_json),
      })),
  );
}

export function createAlertPolicy(payload: Partial<AlertPolicyItem> & { name: string }) {
  return getData<AlertPolicyItem>(http.post("/alerts/policies", payload));
}

export function updateAlertPolicy(id: number, payload: Partial<AlertPolicyItem> & { name: string }) {
  return getData<AlertPolicyItem>(http.put(`/alerts/policies/${id}`, payload));
}

export function deleteAlertPolicy(id: number) {
  return getData<void>(http.delete(`/alerts/policies/${id}`));
}
