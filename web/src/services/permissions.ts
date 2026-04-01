import type {
  MessageData,
  PageData,
  PermissionItem,
  PermissionPayload,
  PermissionQuery,
} from "../types/api";
import { getData, http } from "./http";

export function getPermissions(params: PermissionQuery) {
  return getData<PageData<PermissionItem>>(http.get("/permissions", { params }));
}

export function getPermission(id: number) {
  return getData<PermissionItem>(http.get(`/permissions/${id}`));
}

export function createPermission(payload: PermissionPayload) {
  return getData<PermissionItem>(http.post("/permissions", payload));
}

export function updatePermission(id: number, payload: Partial<PermissionPayload>) {
  return getData<PermissionItem>(http.put(`/permissions/${id}`, payload));
}

export function deletePermission(id: number) {
  return getData<MessageData>(http.delete(`/permissions/${id}`));
}

export function getPermissionOptions() {
  return getData<PageData<PermissionItem>>(http.get("/permissions", { params: { page: 1, page_size: 100 } }));
}