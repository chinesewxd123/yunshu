import type { PageData } from "../types/api";
import { getData, http } from "./http";

export interface RegistrationRequestItem {
  id: number;
  username: string;
  email: string;
  nickname: string;
  status: number;
  reviewer_id?: number;
  review_comment?: string;
  reviewed_at?: string;
  created_at: string;
}

export interface RegistrationListParams {
  keyword?: string;
  status?: number;
  page?: number;
  page_size?: number;
}

export function getRegistrations(params: RegistrationListParams) {
  return getData<PageData<RegistrationRequestItem>>(http.get("/registrations", { params }));
}

export function reviewRegistration(id: number, data: { status: number; comment: string }) {
  return getData<void>(http.post(`/registrations/${id}/review`, data));
}
