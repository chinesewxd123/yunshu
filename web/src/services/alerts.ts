import { getData, http } from "./http";
import { normalizePagedPayload, parseCommaSeparatedList, parseCommaSeparatedNumbers } from "./alert-mappers";

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
  alertIP?: string;
  alertStartedAt?: string;
  /** prometheus = Prometheus 规则+Alertmanager Webhook；platform = 平台内监控规则 */
  monitorPipeline?: string;
  groupKey?: string;
  labelsDigest?: string;
  matchedPolicyIds?: string;
  matchedPolicyNames?: string;
  matchedPolicyIdList?: number[];
  matchedPolicyNameList?: string[];
  receiverList?: string[];
  channelId: number;
  channelName: string;
  success: boolean;
  httpStatusCode: number;
  errorMessage?: string;
  requestPayload?: string;
  responsePayload?: string;
  createdAt: string;
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

/** 与后端 alertdispatch 模板变量说明一致（通道 Go template {{.Name}}） */
export interface AlertTemplateVariableDoc {
  name: string;
  description: string;
}

export interface AlertTemplatePreviewResult {
  rendered: string;
  sample_payload: Record<string, unknown>;
  available_fields: string[];
  raw_payload_fields: string[];
  combined_fields: string[];
  suggested_label_keys: string[];
  /** 固定模板变量及含义（WatchAlert 式「通知模板」文档化） */
  template_variables?: AlertTemplateVariableDoc[];
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
  alertIP?: string;
  status?: string;
  monitorPipeline?: string;
  groupKey?: string;
}) {
  return getData<{ list?: AlertEventItem[]; items?: AlertEventItem[]; total: number; page: number; page_size: number }>(
    http.get("/alerts/events", { params }),
  ).then((payload) =>
    normalizePagedPayload(payload, (item) => ({
      ...item,
      matchedPolicyIdList: parseCommaSeparatedNumbers(item.matchedPolicyIds),
      matchedPolicyNameList: parseCommaSeparatedList(item.matchedPolicyNames),
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
