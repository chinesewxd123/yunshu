import { getData, http } from "./http";
import { normalizePagedPayload, parseNumberArray, parseStringArray, parseStringMap } from "./alert-mappers";

export interface AlertDatasourceItem {
  id: number;
  project_id: number;
  project_name?: string;
  name: string;
  type: string;
  base_url: string;
  bearer_token?: string;
  /** 编辑时后端可能返回；新建后列表接口可能脱敏 */
  basic_user?: string;
  basic_password?: string;
  skip_tls_verify: boolean;
  enabled: boolean;
  remark?: string;
  created_at: string;
  updated_at: string;
}

export interface AlertSilenceItem {
  id: number;
  name: string;
  matchers_json: string;
  matchers?: Array<{ name: string; value: string; is_regex: boolean }>;
  starts_at: string;
  ends_at: string;
  comment?: string;
  created_by: number;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface AlertMonitorRuleItem {
  id: number;
  datasource_id: number;
  project_id?: number;
  project_name?: string;
  datasource_name?: string;
  name: string;
  expr: string;
  for_seconds: number;
  eval_interval_seconds: number;
  severity: string;
  threshold_unit?: string;
  labels_json?: string;
  annotations_json?: string;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface AlertRuleAssigneeItem {
  id: number;
  monitor_rule_id: number;
  user_ids_json?: string;
  department_ids_json?: string;
  extra_emails_json?: string;
  user_ids?: number[];
  department_ids?: number[];
  extra_emails?: string[];
  notify_on_resolved: boolean;
  remark?: string;
}

export interface AlertDutyBlockItem {
  id: number;
  monitor_rule_id: number;
  starts_at: string;
  ends_at: string;
  title?: string;
  user_ids_json?: string;
  department_ids_json?: string;
  extra_emails_json?: string;
  user_ids?: number[];
  department_ids?: number[];
  extra_emails?: string[];
  remark?: string;
  created_at: string;
  updated_at: string;
}

export interface CloudExpiryRuleItem {
  id: number;
  project_id: number;
  name: string;
  provider: string;
  region_scope: string;
  advance_days: number;
  severity: string;
  labels_json?: string;
  labels?: Record<string, string>;
  eval_interval_seconds: number;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export type Paged<T> = { list: T[]; total: number; page: number; page_size: number };

function parseSilenceMatchers(raw?: string): Array<{ name: string; value: string; is_regex: boolean }> {
  const s = String(raw || "").trim();
  if (!s) return [];
  try {
    const parsed = JSON.parse(s) as Array<Record<string, unknown>>;
    if (!Array.isArray(parsed)) return [];
    return parsed
      .map((item) => ({
        name: String(item?.name ?? "").trim(),
        value: String(item?.value ?? "").trim(),
        is_regex: Boolean(item?.is_regex),
      }))
      .filter((item) => item.name);
  } catch {
    return [];
  }
}

function mapMonitorRule(item: AlertMonitorRuleItem): AlertMonitorRuleItem {
  return {
    ...item,
    labels: parseStringMap(item.labels_json),
    annotations: parseStringMap(item.annotations_json),
  };
}

function mapAssignee(item: AlertRuleAssigneeItem): AlertRuleAssigneeItem {
  return {
    ...item,
    user_ids: parseNumberArray(item.user_ids_json),
    department_ids: parseNumberArray(item.department_ids_json),
    extra_emails: parseStringArray(item.extra_emails_json),
  };
}

function mapDutyBlock(item: AlertDutyBlockItem): AlertDutyBlockItem {
  return {
    ...item,
    user_ids: parseNumberArray(item.user_ids_json),
    department_ids: parseNumberArray(item.department_ids_json),
    extra_emails: parseStringArray(item.extra_emails_json),
  };
}

export function listAlertDatasources(params?: { project_id?: number; keyword?: string; page?: number; page_size?: number }) {
  return getData<{ list?: AlertDatasourceItem[]; items?: AlertDatasourceItem[]; total: number; page: number; page_size: number }>(
    http.get("/alerts/datasources", { params }),
  ).then((payload) => normalizePagedPayload(payload));
}

export function createAlertDatasource(payload: Record<string, unknown>) {
  return getData<AlertDatasourceItem>(http.post("/alerts/datasources", payload));
}

export function updateAlertDatasource(id: number, payload: Record<string, unknown>) {
  return getData<AlertDatasourceItem>(http.put(`/alerts/datasources/${id}`, payload));
}

export function deleteAlertDatasource(id: number) {
  return getData<void>(http.delete(`/alerts/datasources/${id}`));
}

/** GET Prometheus /api/v1/alerts，返回原始 JSON（含 data.alerts）。 */
export function promActiveAlerts(id: number) {
  return getData<{ data: unknown }>(http.get(`/alerts/datasources/${id}/prometheus-alerts`)).then((r) => r.data);
}

export function promInstantQuery(id: number, payload: { query: string; time?: string }) {
  return getData<{ data: unknown }>(http.post(`/alerts/datasources/${id}/query`, payload));
}

export function promRangeQuery(id: number, payload: { query: string; start: string; end: string; step: string }) {
  return getData<{ data: unknown }>(http.post(`/alerts/datasources/${id}/query_range`, payload));
}

export function listAlertSilences(params?: { keyword?: string; page?: number; page_size?: number }) {
  return getData<{ list?: AlertSilenceItem[]; items?: AlertSilenceItem[]; total: number; page: number; page_size: number }>(
    http.get("/alerts/silences", { params }),
  ).then((payload) =>
    normalizePagedPayload(payload, (item) => ({
      ...item,
      matchers: parseSilenceMatchers(item.matchers_json),
    })),
  );
}

export function createAlertSilence(payload: Record<string, unknown>) {
  return getData<AlertSilenceItem>(http.post("/alerts/silences", payload));
}

export function updateAlertSilence(id: number, payload: Record<string, unknown>) {
  return getData<AlertSilenceItem>(http.put(`/alerts/silences/${id}`, payload));
}

export function deleteAlertSilence(id: number) {
  return getData<void>(http.delete(`/alerts/silences/${id}`));
}

export function listAlertMonitorRules(params?: { datasource_id?: number; project_id?: number; keyword?: string; page?: number; page_size?: number }) {
  return getData<{ list?: AlertMonitorRuleItem[]; items?: AlertMonitorRuleItem[]; total: number; page: number; page_size: number }>(
    http.get("/alerts/monitor-rules", { params }),
  ).then((payload) => normalizePagedPayload(payload, mapMonitorRule));
}

export function createAlertMonitorRule(payload: Record<string, unknown>) {
  return getData<AlertMonitorRuleItem>(http.post("/alerts/monitor-rules", payload));
}

export function updateAlertMonitorRule(id: number, payload: Record<string, unknown>) {
  return getData<AlertMonitorRuleItem>(http.put(`/alerts/monitor-rules/${id}`, payload));
}

export function deleteAlertMonitorRule(id: number) {
  return getData<void>(http.delete(`/alerts/monitor-rules/${id}`));
}

export function listCloudExpiryRules(params?: { project_id?: number; provider?: string; keyword?: string; page?: number; page_size?: number }) {
  return getData<{ list?: CloudExpiryRuleItem[]; items?: CloudExpiryRuleItem[]; total: number; page: number; page_size: number }>(
    http.get("/alerts/cloud-expiry-rules", { params }),
  ).then((payload) =>
    normalizePagedPayload(payload, (item) => ({
      ...item,
      labels: parseStringMap(item.labels_json),
    })),
  );
}

export function createCloudExpiryRule(payload: Record<string, unknown>) {
  return getData<CloudExpiryRuleItem>(http.post("/alerts/cloud-expiry-rules", payload));
}

export function updateCloudExpiryRule(id: number, payload: Record<string, unknown>) {
  return getData<CloudExpiryRuleItem>(http.put(`/alerts/cloud-expiry-rules/${id}`, payload));
}

export function deleteCloudExpiryRule(id: number) {
  return getData<void>(http.delete(`/alerts/cloud-expiry-rules/${id}`));
}

export function evaluateCloudExpiryRulesNow() {
  return getData<{ message: string }>(http.post("/alerts/cloud-expiry-rules/evaluate-now", {}));
}

export function getMonitorRuleAssignees(ruleId: number) {
  return getData<{ list?: AlertRuleAssigneeItem[]; items?: AlertRuleAssigneeItem[] }>(http.get(`/alerts/monitor-rules/${ruleId}/assignees`)).then((payload) => {
    const source = Array.isArray(payload.items) ? payload.items : Array.isArray(payload.list) ? payload.list : [];
    const items = source.map(mapAssignee);
    return { items, list: items };
  });
}

export function upsertMonitorRuleAssignees(ruleId: number, payload: Record<string, unknown>) {
  return getData<AlertRuleAssigneeItem>(http.put(`/alerts/monitor-rules/${ruleId}/assignees`, payload));
}

export function listDutyBlocks(params?: { monitor_rule_id?: number; project_id?: number; page?: number; page_size?: number }) {
  return getData<{ list?: AlertDutyBlockItem[]; items?: AlertDutyBlockItem[]; total: number; page: number; page_size: number }>(
    http.get("/alerts/duty-blocks", { params }),
  ).then((payload) => normalizePagedPayload(payload, mapDutyBlock));
}

export function createDutyBlock(payload: Record<string, unknown>) {
  return getData<AlertDutyBlockItem>(http.post("/alerts/duty-blocks", payload));
}

export function updateDutyBlock(id: number, payload: Record<string, unknown>) {
  return getData<AlertDutyBlockItem>(http.put(`/alerts/duty-blocks/${id}`, payload));
}

export function deleteDutyBlock(id: number) {
  return getData<{ message: string }>(http.delete(`/alerts/duty-blocks/${id}`));
}
