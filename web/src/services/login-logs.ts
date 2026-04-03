import type { MessageData, PageData } from "../types/api";
import { getData, http } from "./http";

export interface LoginLogItem {
  id: number;
  created_at: string;
  username: string;
  ip: string;
  status: number;
  detail: string;
  user_agent: string;
  source: string;
  user_id?: number;
}

export interface LoginLogQuery {
  username?: string;
  status?: number;
  source?: string;
  page?: number;
  page_size?: number;
}

export function getLoginLogs(params: LoginLogQuery) {
  return getData<PageData<LoginLogItem>>(http.get("/login-logs", { params }));
}

export function deleteLoginLog(id: number) {
  return getData<MessageData>(http.delete(`/login-logs/${id}`));
}

export function batchDeleteLoginLogs(ids: number[]) {
  return getData<MessageData>(http.post("/login-logs/delete", { ids }));
}

export async function exportLoginLogs(params?: Record<string, any>): Promise<Blob> {
  // axios instance returns `response.data` at runtime (see interceptors), but TS types still assume AxiosResponse.
  return (await http.get(`/login-logs/export`, { params, responseType: "blob" })) as unknown as Blob;
}
