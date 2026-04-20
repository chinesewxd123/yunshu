import type {
  ChangePasswordPayload,
  EmailLoginPayload,
  LoginResult,
  MessageData,
  PasswordLoginPayload,
  RegisterPayload,
  RegisterResult,
  SendEmailCodePayload,
  SendEmailCodeResult,
  SendPasswordLoginCodePayload,
  SendPasswordLoginCodeResult,
  UpdateProfilePayload,
  UserItem,
} from "../types/api";
import { getData, http } from "./http";

export function sendEmailCode(payload: SendEmailCodePayload) {
  return getData<SendEmailCodeResult>(http.post("/auth/verification-code", payload));
}

export function sendLoginCodeByUsername(username: string) {
  return getData<SendEmailCodeResult>(http.post("/auth/login-code", { username }));
}

export function sendPasswordLoginCode(payload: SendPasswordLoginCodePayload) {
  return getData<SendPasswordLoginCodeResult>(http.post("/auth/password-login-code", payload));
}

export function passwordLogin(payload: PasswordLoginPayload) {
  return getData<LoginResult>(http.post("/auth/login", payload));
}

export function emailLogin(payload: EmailLoginPayload) {
  return getData<LoginResult>(http.post("/auth/email-login", payload));
}

export function registerByEmail(payload: RegisterPayload) {
  // 注册失败提示由页面自行处理，避免拦截器通用文案覆盖后端细节。
  return getData<MessageData>(http.post("/auth/register", payload, { silentErrorToast: true }));
}

export function logout() {
  return getData<MessageData>(http.post("/auth/logout"));
}

export function getCurrentUser() {
  return getData<UserItem>(http.get("/auth/me"));
}

export function updateProfile(payload: UpdateProfilePayload) {
  return getData<UserItem>(http.put("/auth/me", payload));
}

export function changePassword(payload: ChangePasswordPayload) {
  return getData<MessageData>(http.put("/auth/password", payload));
}

export interface HealthData {
  status: string;
  version: string;
  uptime: number;
}

export function getHealth() {
  return getData<HealthData>(http.get("/health"));
}
