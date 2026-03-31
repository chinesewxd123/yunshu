import axios from "axios";
import { message } from "antd";
import type { ApiResponse } from "../types/api";
import { clearAuthStorage, getToken } from "./storage";

export const http = axios.create({
  baseURL: "/api/v1",
  timeout: 15000,
});

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
        message.error("登录已失效，请重新登录");
        window.location.href = "/login";
      }
    } else {
      message.error(errorMessage);
    }

    return Promise.reject(error);
  },
);

export async function getData<T>(promise: Promise<ApiResponse<T>>) {
  const result = await promise;
  return result.data;
}
