import type {
  AssignRolesPayload,
  MessageData,
  PageData,
  UserCreatePayload,
  UserItem,
  UserQuery,
  UserUpdatePayload,
} from "../types/api";
import { getData, http } from "./http";

export function getUsers(params: UserQuery) {
  return getData<PageData<UserItem>>(http.get("/users", { params }));
}

export function getUser(id: number) {
  return getData<UserItem>(http.get(`/users/${id}`));
}

export function createUser(payload: UserCreatePayload) {
  return getData<UserItem>(http.post("/users", payload));
}

export function updateUser(id: number, payload: UserUpdatePayload) {
  return getData<UserItem>(http.put(`/users/${id}`, payload));
}

export function deleteUser(id: number) {
  return getData<MessageData>(http.delete(`/users/${id}`));
}

export function assignUserRoles(id: number, payload: AssignRolesPayload) {
  return getData<UserItem>(http.put(`/users/${id}/roles`, payload));
}

export async function exportUsers(params?: Record<string, any>): Promise<Blob> {
  // axios instance returns `response.data` at runtime (see interceptors), but TS types still assume AxiosResponse.
  return (await http.get(`/users/export`, { params, responseType: "blob" })) as unknown as Blob;
}

export async function downloadUsersImportTemplate(): Promise<Blob> {
  return (await http.get(`/users/import-template`, { responseType: "blob" })) as unknown as Blob;
}

export function importUsers(file: File) {
  const fd = new FormData();
  fd.append("file", file);
  return http.post(`/users/import`, fd, { headers: { "Content-Type": "multipart/form-data" } });
}