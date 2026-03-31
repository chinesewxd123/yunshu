export interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
}

export interface PageData<T> {
  list: T[];
  total: number;
  page: number;
  page_size: number;
}

export interface RoleItem {
  id: number;
  name: string;
  code: string;
  description: string;
  status: number;
}

export interface PermissionItem {
  id: number;
  name: string;
  resource: string;
  action: string;
  description: string;
}

export interface UserItem {
  id: number;
  username: string;
  nickname: string;
  status: number;
  roles: RoleItem[];
  created_at: string;
  updated_at: string;
}

export interface LoginPayload {
  username: string;
  password: string;
}

export interface LoginResult {
  token: string;
  expires_at: string;
  user: UserItem;
}

export interface UserCreatePayload {
  username: string;
  password: string;
  nickname: string;
  status: number;
  role_ids: number[];
}

export interface UserUpdatePayload {
  nickname?: string;
  password?: string;
  status?: number;
}

export interface AssignRolesPayload {
  role_ids: number[];
}

export interface UserQuery {
  keyword?: string;
  page?: number;
  page_size?: number;
}

export interface RolePayload {
  name: string;
  code: string;
  description?: string;
  status: number;
}

export interface RoleQuery {
  keyword?: string;
  page?: number;
  page_size?: number;
}

export interface PermissionPayload {
  name: string;
  resource: string;
  action: string;
  description?: string;
}

export interface PermissionQuery {
  keyword?: string;
  page?: number;
  page_size?: number;
}

export interface PolicyItem {
  role_id: number;
  role_name: string;
  role_code: string;
  permission_id: number;
  permission_name: string;
  resource: string;
  action: string;
}

export interface PolicyPayload {
  role_id: number;
  permission_id: number;
}

export interface MessageData {
  message: string;
}