export interface ApiResponse<T> {
  code: number;
  message: string;
  /** 业务错误码数字字符串，与后端 response.Body.error_code 对齐（成功时通常为空） */
  error_code?: string;
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
  created_at: string;
  updated_at: string;
}

export interface PermissionItem {
  id: number;
  name: string;
  resource: string;
  action: string;
  description: string;
  k8s_scope_enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface UserGroupBrief {
  id: number;
  name: string;
  code: string;
}

export interface UserItem {
  id: number;
  username: string;
  email: string;
  nickname: string;
  status: number;
  department_id?: number;
  department_name?: string;
  roles: RoleItem[];
  /** 用户所属用户组（K8s 主体 principal_kind=group 等场景） */
  groups?: UserGroupBrief[];
  created_at: string;
  updated_at: string;
}

export interface DepartmentItem {
  id: number;
  parent_id?: number;
  name: string;
  code: string;
  ancestors: string;
  level: number;
  sort: number;
  status: number;
  leader_id?: number;
  leader_name?: string;
  phone?: string;
  email?: string;
  remark?: string;
  user_count?: number;
  created_at: string;
  updated_at: string;
  children?: DepartmentItem[];
}

export interface PasswordLoginPayload {
  username: string;
  password: string;
  captcha_key: string;
  code: string;
}

export type LoginPayload = PasswordLoginPayload;

export type EmailCodeScene = "login" | "register";

export interface SendEmailCodePayload {
  email: string;
  scene: EmailCodeScene;
}

export interface SendEmailCodeResult {
  email: string;
  scene: EmailCodeScene;
  expires_in: number;
  cooldown_in: number;
}

export interface SendPasswordLoginCodePayload {
  username: string;
}

export interface SendPasswordLoginCodeResult {
  captcha_key: string;
  image: string;
  expires_in: number;
  cooldown_in: number;
}

export interface EmailLoginPayload {
  email: string;
  code: string;
}

export interface RegisterPayload {
  username: string;
  email: string;
  nickname: string;
  password: string;
  code: string;
}

export interface RegisterResult {
  message: string;
  user: UserItem;
}

export interface LoginResult {
  token: string;
  expires_at: string;
  user: UserItem;
}

export interface UpdateProfilePayload {
  nickname: string;
  email?: string;
}

export interface ChangePasswordPayload {
  old_password: string;
  new_password: string;
}

export interface UserCreatePayload {
  username: string;
  email: string;
  password: string;
  nickname: string;
  status: number;
  department_id?: number;
  role_ids: number[];
}

export interface UserUpdatePayload {
  email?: string;
  nickname?: string;
  password?: string;
  status?: number;
  department_id?: number;
}

export interface AssignRolesPayload {
  role_ids: number[];
}

export interface UserQuery {
  keyword?: string;
  department_id?: number;
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
  k8s_scope_enabled?: boolean;
}

export interface PermissionQuery {
  keyword?: string;
  page?: number;
  page_size?: number;
  /** 空：全部；on：仅已纳入 K8s 范围校验；off：仅未纳入 */
  k8s_scope?: "" | "on" | "off";
  /** 空：全部；on：仅后端挂载 K8s 三元中间件的集群资源路径（与 router 前缀表一致） */
  k8s_related?: "" | "on";
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
