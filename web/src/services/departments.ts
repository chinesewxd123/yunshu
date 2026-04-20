import type { DepartmentItem } from "../types/api";
import { getData, http } from "./http";

export interface DepartmentCreatePayload {
  parent_id?: number;
  name: string;
  code: string;
  sort?: number;
  status: number;
  leader_id?: number;
  phone?: string;
  email?: string;
  remark?: string;
}

export type DepartmentUpdatePayload = DepartmentCreatePayload;

export function getDepartmentTree() {
  return getData<DepartmentItem[]>(http.get("/departments/tree"));
}

export function getDepartmentDetail(id: number) {
  return getData<DepartmentItem>(http.get(`/departments/${id}`));
}

export function createDepartment(payload: DepartmentCreatePayload) {
  return getData<DepartmentItem>(http.post("/departments", payload));
}

export function updateDepartment(id: number, payload: DepartmentUpdatePayload) {
  return getData<DepartmentItem>(http.put(`/departments/${id}`, payload));
}

export function deleteDepartment(id: number) {
  return getData<{ message: string }>(http.delete(`/departments/${id}`));
}
