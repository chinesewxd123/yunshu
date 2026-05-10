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
};

export type UserGroupCreatePayload = {
  name: string;
  code: string;
  description?: string;
  status?: number;
};

export type UserGroupUpdatePayload = {
  name?: string;
  description?: string;
  status?: number;
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
