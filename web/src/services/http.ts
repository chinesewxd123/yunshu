import axios from "axios";
import { message } from "antd";
import type { ApiResponse } from "../types/api";
import { clearAuthStorage, getToken } from "./storage";
import { resolveApiErrorDisplayMessage } from "../utils/api-error-messages";

declare module "axios" {
  interface AxiosRequestConfig<D = any> {
    silentErrorToast?: boolean;
  }

  interface InternalAxiosRequestConfig<D = any> {
    silentErrorToast?: boolean;
  }
}

export const http = axios.create({
  baseURL: "/api/v1",
  timeout: 15000,
});

function toastOnce(key: string, content: string) {
  message.error({ content, key });
}

function nextRequestId(): string {
  try {
    return crypto.randomUUID();
  } catch {
    return `req-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
  }
}

http.interceptors.request.use((config) => {
  const token = getToken();
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  // 与后端 RequestLogger 的 X-Request-ID 对齐，便于日志关联与排障
  if (!config.headers["X-Request-ID"]) {
    config.headers["X-Request-ID"] = nextRequestId();
  }
  return config;
});

http.interceptors.response.use(
  (response) => response.data,
  (error) => {
    const status = error.response?.status;
    const rawData = error.response?.data as { message?: string; error_code?: string } | undefined;
    const resolved = resolveApiErrorDisplayMessage(rawData?.error_code, rawData?.message);
    const errorMessage = resolved || error.message || "请求失败";
    const silentErrorToast = Boolean(error.config?.silentErrorToast);

    if (silentErrorToast) {
      return Promise.reject(error);
    }

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

/** 从 axios 错误中取出后端 Body.Message（与 response.Error 业务话术对齐）；非 axios 时回退 Error.message。 */
export function extractApiErrorMessage(error: unknown, fallback = "请求失败"): string {
  if (axios.isAxiosError(error)) {
    const data = error.response?.data as { message?: string; error_code?: string } | undefined;
    const resolved = resolveApiErrorDisplayMessage(data?.error_code, data?.message);
    if (resolved) {
      return resolved;
    }
  }
  if (error instanceof Error && error.message) {
    return error.message;
  }
  return fallback;
}
