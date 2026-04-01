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