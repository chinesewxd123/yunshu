import type { UserItem } from "../types/api";

const TOKEN_KEY = "permission-system-token";
const USER_KEY = "permission-system-user";

export function getToken() {
  return window.localStorage.getItem(TOKEN_KEY) ?? "";
}

export function setToken(token: string) {
  window.localStorage.setItem(TOKEN_KEY, token);
}

export function clearToken() {
  window.localStorage.removeItem(TOKEN_KEY);
}

export function getUser() {
  const raw = window.localStorage.getItem(USER_KEY);
  if (!raw) {
    return null;
  }
  try {
    return JSON.parse(raw) as UserItem;
  } catch {
    return null;
  }
}

export function setUser(user: UserItem) {
  window.localStorage.setItem(USER_KEY, JSON.stringify(user));
}

export function clearUser() {
  window.localStorage.removeItem(USER_KEY);
}

export function clearAuthStorage() {
  clearToken();
  clearUser();
}