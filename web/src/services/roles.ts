import type { MessageData, PageData, RoleItem, RolePayload, RoleQuery } from "../types/api";
import { getData, http } from "./http";

export function getRoles(params: RoleQuery) {
  return getData<PageData<RoleItem>>(http.get("/roles", { params }));
}

export function getRole(id: number) {
  return getData<RoleItem>(http.get(`/roles/${id}`));
}

export function createRole(payload: RolePayload) {
  return getData<RoleItem>(http.post("/roles", payload));
}

export function updateRole(id: number, payload: Partial<RolePayload>) {
  return getData<RoleItem>(http.put(`/roles/${id}`, payload));
}

export function deleteRole(id: number) {
  return getData<MessageData>(http.delete(`/roles/${id}`));
}

export function getRoleOptions() {
  return getData<PageData<RoleItem>>(http.get("/roles", { params: { page: 1, page_size: 100 } }));
}