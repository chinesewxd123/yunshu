import { getData, http } from "./http";

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
  group_key?: string;
  labels_digest?: string;
  channel_id: number;
  channel_name: string;
  success: boolean;
  http_status_code: number;
  error_message?: string;
  request_payload?: string;
  response_payload?: string;
  created_at: string;
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
  payload?: { title?: string; content?: string; severity?: string },
) {
  return getData<void>(http.post(`/alerts/channels/${id}/test`, payload ?? {}));
}

export function listAlertEvents(params: { page: number; page_size: number; keyword?: string; cluster?: string; group_key?: string }) {
  return getData<{ list: AlertEventItem[]; total: number; page: number; page_size: number }>(
    http.get("/alerts/events", { params }),
  );
}

