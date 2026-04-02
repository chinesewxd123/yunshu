import type { MessageData, PageData } from "../types/api";
import { getData, http } from "./http";

export interface OperationLogItem {
  id: number;
  created_at: string;
  user_id: number;
  username: string;
  nickname: string;
  ip: string;
  method: string;
  path: string;
  status_code: number;
  request_body: string;
  response_body: string;
  latency_ms: number;
}

export interface OperationLogQuery {
  method?: string;
  path?: string;
  status_code?: number;
  page?: number;
  page_size?: number;
}

export function getOperationLogs(params: OperationLogQuery) {
  return getData<PageData<OperationLogItem>>(http.get("/operation-logs", { params }));
}

export function deleteOperationLog(id: number) {
  return getData<MessageData>(http.delete(`/operation-logs/${id}`));
}

export function batchDeleteOperationLogs(ids: number[]) {
  return getData<MessageData>(http.post("/operation-logs/delete", { ids }));
}

export function exportOperationLogs(params?: Record<string, any>) {
  return http.get(`/operation-logs/export`, { params, responseType: "blob" });
}
