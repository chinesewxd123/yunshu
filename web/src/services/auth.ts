import type { LoginPayload, LoginResult, MessageData, UserItem } from "../types/api";
import { getData, http } from "./http";

export function login(payload: LoginPayload) {
  return getData<LoginResult>(http.post("/auth/login", payload));
}

export function logout() {
  return getData<MessageData>(http.post("/auth/logout"));
}

export function getCurrentUser() {
  return getData<UserItem>(http.get("/auth/me"));
}