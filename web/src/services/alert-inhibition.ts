import { getData, http } from "./http";

export interface AlertInhibitionRule {
  id: number;
  name: string;
  description?: string;
  enabled: boolean;
  priority: number;
  source_match_labels_json?: string;
  source_match_regex_json?: string;
  target_match_labels_json?: string;
  target_match_regex_json?: string;
  equal_labels_json?: string;
  duration_seconds: number;
  source_match_labels?: Record<string, string>;
  source_match_regex?: Record<string, string>;
  target_match_labels?: Record<string, string>;
  target_match_regex?: Record<string, string>;
  equal_labels?: string[];
  created_at?: string;
  updated_at?: string;
}

export type AlertInhibitionRulePayload = {
  name: string;
  description?: string;
  enabled?: boolean;
  priority?: number;
  source_match_labels_json?: string;
  source_match_regex_json?: string;
  target_match_labels_json?: string;
  target_match_regex_json?: string;
  equal_labels_json?: string;
  duration_seconds?: number;
};

export function listInhibitionRules(params?: { page?: number; page_size?: number; keyword?: string; enabled?: boolean }) {
  return getData<{ list: AlertInhibitionRule[]; total: number; page: number; page_size: number }>(
    http.get("/alerts/inhibition-rules", { params }),
  );
}

export function createInhibitionRule(payload: AlertInhibitionRulePayload) {
  return getData<AlertInhibitionRule>(http.post("/alerts/inhibition-rules", payload));
}

export function updateInhibitionRule(id: number, payload: AlertInhibitionRulePayload) {
  return getData<AlertInhibitionRule>(http.put(`/alerts/inhibition-rules/${id}`, payload));
}

export function deleteInhibitionRule(id: number) {
  return getData<void>(http.delete(`/alerts/inhibition-rules/${id}`));
}

export function refreshInhibitionCache() {
  return getData<{ refreshed: boolean }>(http.post("/alerts/inhibition-rules/refresh-cache"));
}
