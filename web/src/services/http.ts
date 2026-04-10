import axios from "axios";
import { message } from "antd";
import type { ApiResponse } from "../types/api";
import { clearAuthStorage, getToken } from "./storage";

export const http = axios.create({
  baseURL: "/api/v1",
  timeout: 15000,
});

function toastOnce(key: string, content: string) {
  message.error({ content, key });
}

http.interceptors.request.use((config) => {
  const token = getToken();
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

http.interceptors.response.use(
  (response) => response.data,
  (error) => {
    const status = error.response?.status;
    const errorMessage = error.response?.data?.message || error.message || "请求失败";

    if (status === 401) {
      clearAuthStorage();
      if (window.location.pathname !== "/login") {
        toastOnce("auth-expired", "登录已失效，请重新登录");
        window.location.href = "/login";
      } else {
        toastOnce("http-error", errorMessage);
      }
    } else if (status === 403) {
      // 统一 403 提示：用固定 key，避免与页面内提示重复弹出
      toastOnce("forbidden", typeof errorMessage === "string" ? errorMessage : "无访问权限");
    } else {
      // 用于“同名探测/存在性检查”场景：先调用 detail 接口判断是否存在，
      // 若不存在会返回类似“xxx 不存在”的业务错误，但不应该弹 toast 干扰用户操作。
      const requestUrl = String(error.config?.url ?? "");
      const isExistenceProbe =
        typeof errorMessage === "string" &&
        errorMessage.includes("不存在") &&
        (requestUrl.includes("/detail") || requestUrl.includes("detail"));
      if (!isExistenceProbe) {
        toastOnce("http-error", errorMessage);
      }
    }

    return Promise.reject(error);
  },
);

export async function getData<T>(promise: Promise<ApiResponse<T>>) {
  const result = await promise;
  return result.data;
}
