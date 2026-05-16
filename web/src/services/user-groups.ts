import { getData, http } from "./http";

export type UserGroupMemberRow = {
  user_id: number;
  username: string;
  nickname: string;
};

export type UserGroupItem = {
  id: number;
  name: string;
  code: string;
  description: string;
  status: number;
  /** 非空时仅该项目成员可维护该组；成员须为项目成员 */
  scope_project_id?: number | null;
  member_count: number;
  created_at: string;
  updated_at: string;
};

export type UserGroupDetail = UserGroupItem & {
  members: UserGroupMemberRow[];
};

export type UserGroupQuery = {
  keyword?: string;
  page?: number;
  page_size?: number;
  /** 仅列出该项目的专属组或全局组（与后端一致） */
  scope_project_id?: number;
};

export type UserGroupCreatePayload = {
  name: string;
  code: string;
  description?: string;
  status?: number;
  scope_project_id?: number;
};

export type UserGroupUpdatePayload = {
  name?: string;
  description?: string;
  status?: number;
  /** 传 0 表示取消项目专属作用域 */
  scope_project_id?: number;
};

export function listUserGroups(params: UserGroupQuery) {
  return getData<{ list: UserGroupItem[]; total: number; page: number; page_size: number }>(
    http.get("/user-groups", { params }),
  );
}

export function getUserGroup(id: number) {
  return getData<UserGroupDetail>(http.get(`/user-groups/${id}`));
}

export function createUserGroup(payload: UserGroupCreatePayload) {
  return getData<UserGroupItem>(http.post("/user-groups", payload));
}

export function updateUserGroup(id: number, payload: UserGroupUpdatePayload) {
  return getData<UserGroupItem>(http.put(`/user-groups/${id}`, payload));
}

export function deleteUserGroup(id: number) {
  return getData<{ message: string }>(http.delete(`/user-groups/${id}`));
}

export function assignUserGroupMembers(id: number, payload: { user_ids: number[] }) {
  return getData<{ message: string }>(http.put(`/user-groups/${id}/users`, payload));
}
