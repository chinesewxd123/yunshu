import { getData, http } from "./http";

export interface AlertDatasourceItem {
  id: number;
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
  name: string;
  expr: string;
  for_seconds: number;
  eval_interval_seconds: number;
  severity: string;
  threshold_unit?: string;
  labels_json?: string;
  annotations_json?: string;
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
  remark?: string;
  created_at: string;
  updated_at: string;
}

export type Paged<T> = { list: T[]; total: number; page: number; page_size: number };

export function listAlertDatasources(params?: { keyword?: string; page?: number; page_size?: number }) {
  return getData<Paged<AlertDatasourceItem>>(http.get("/alerts/datasources", { params }));
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
  return getData<Paged<AlertSilenceItem>>(http.get("/alerts/silences", { params }));
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
  return getData<Paged<AlertMonitorRuleItem>>(http.get("/alerts/monitor-rules", { params }));
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

export function getMonitorRuleAssignees(ruleId: number) {
  return getData<{ list: AlertRuleAssigneeItem[] }>(http.get(`/alerts/monitor-rules/${ruleId}/assignees`));
}

export function upsertMonitorRuleAssignees(ruleId: number, payload: Record<string, unknown>) {
  return getData<AlertRuleAssigneeItem>(http.put(`/alerts/monitor-rules/${ruleId}/assignees`, payload));
}

export function listDutyBlocks(params?: { monitor_rule_id?: number; project_id?: number; page?: number; page_size?: number }) {
  return getData<Paged<AlertDutyBlockItem>>(http.get("/alerts/duty-blocks", { params }));
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
