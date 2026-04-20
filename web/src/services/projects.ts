import type { ApiResponse, PageData } from "../types/api";
import { getData, http } from "./http";

export interface ProjectItem {
  id: number;
  name: string;
  code: string;
  description?: string | null;
  status: number;
  created_at: string;
}

export interface ProjectCreatePayload {
  name: string;
  code: string;
  description?: string;
  status: number;
}

export interface ProjectUpdatePayload {
  name?: string;
  code?: string;
  description?: string | null;
  status?: number;
}

export async function getProjects(params: { keyword?: string; page?: number; page_size?: number }) {
  return await getData(http.get<any, ApiResponse<PageData<ProjectItem>>>("/projects", { params }));
}

export async function createProject(payload: ProjectCreatePayload) {
  return await getData(http.post<any, ApiResponse<ProjectItem>>("/projects", payload));
}

export async function updateProject(id: number, payload: ProjectUpdatePayload) {
  return await getData(http.put<any, ApiResponse<ProjectItem>>(`/projects/${id}`, payload));
}

export async function deleteProject(id: number) {
  return await getData(http.delete<any, ApiResponse<{ message: string }>>(`/projects/${id}`));
}

/** 项目成员（project_members），与监控规则 project_id、告警通知收件人联动 */
export interface ProjectMemberItem {
  id: number;
  user_id: number;
  username: string;
  nickname: string;
  email?: string | null;
  created_at: string;
}

export async function listProjectMembers(projectId: number) {
  return await getData<{ list: ProjectMemberItem[] }>(http.get(`/projects/${projectId}/members`));
}

export async function addProjectMember(projectId: number, payload: { user_id: number }) {
  return await getData<ProjectMemberItem>(http.post(`/projects/${projectId}/members`, payload));
}

export async function updateProjectMember(projectId: number, memberId: number, payload: { role: string }) {
  return await getData<ProjectMemberItem>(http.put(`/projects/${projectId}/members/${memberId}`, payload));
}

export async function removeProjectMember(projectId: number, memberId: number) {
  return await getData<{ message: string }>(http.delete(`/projects/${projectId}/members/${memberId}`));
}

export interface ServerItem {
  id: number;
  project_id: number;
  group_id?: number | null;
  name: string;
  host: string;
  port: number;
  os_type: string;
  os_arch: string;
  tags: string;
  source_type: string;
  provider: string;
  cloud_instance_id: string;
  cloud_region: string;
  status: number;
  created_at: string;
  last_seen_at?: string | null;
  last_test_at?: string | null;
  last_test_error?: string | null;
}

export interface ServerUpsertPayload {
  id?: number;
  project_id: number;
  group_id?: number;
  name: string;
  host: string;
  port?: number;
  os_type?: string;
  tags?: string;
  status: number;
  source_type?: string;
  provider?: string;
  cloud_instance_id?: string;
  cloud_region?: string;

  auth_type?: "password" | "key";
  username?: string;
  password?: string;
  private_key?: string;
  passphrase?: string;
  /** 数据字典模板标签，便于编辑回显；与 username 可同时存在 */
  username_dict_label?: string;
  password_dict_label?: string;
}

/** GET 单台服务器详情（含 SSH 凭据元数据，不含密钥明文） */
export interface ServerDetailItem extends ServerItem {
  auth_type?: string;
  username?: string;
  password_set?: boolean;
  private_key_set?: boolean;
  username_dict_label?: string | null;
  password_dict_label?: string | null;
}

export async function getProjectServers(
  projectId: number,
  params: { keyword?: string; page?: number; page_size?: number; group_id?: number; source_type?: string; provider?: string },
) {
  return await getData(
    http.get<any, ApiResponse<PageData<ServerItem>>>(`/projects/${projectId}/servers`, { params: { ...params, project_id: projectId } }),
  );
}

export async function upsertProjectServer(projectId: number, payload: ServerUpsertPayload) {
  return await getData(http.post<any, ApiResponse<ServerItem>>(`/projects/${projectId}/servers`, payload));
}

export async function deleteProjectServer(projectId: number, serverId: number) {
  return await getData(http.delete<any, ApiResponse<{ message: string }>>(`/projects/${projectId}/servers/${serverId}`));
}

export async function getProjectServerDetail(projectId: number, serverId: number) {
  return await getData<ServerDetailItem>(http.get(`/projects/${projectId}/servers/${serverId}`));
}

export interface ServerExecPayload {
  command: string;
  timeout_sec?: number;
}

export interface ServerExecResult {
  server_id: number;
  command: string;
  stdout: string;
  stderr: string;
  exit_code: number;
  duration_ms: number;
  truncated: boolean;
}

export async function execProjectServerCommand(projectId: number, serverId: number, payload: ServerExecPayload) {
  return await getData<ServerExecResult>(http.post(`/projects/${projectId}/servers/${serverId}/exec`, payload));
}

export async function testProjectServer(projectId: number, serverId: number) {
  return await getData(http.post<any, ApiResponse<{ ok: boolean; message: string }>>(`/projects/${projectId}/servers/test`, { server_id: serverId }));
}

export interface BatchServerTestResult {
  server_id: number;
  ok: boolean;
  message: string;
}

export interface BatchServerTestResponse {
  total: number;
  success: number;
  failed: number;
  results: BatchServerTestResult[];
}

export async function batchTestProjectServers(projectId: number, serverIds: number[], parallel = 5) {
  return await getData(
    http.post<any, ApiResponse<BatchServerTestResponse>>(`/projects/${projectId}/servers/test/batch`, {
      project_id: projectId,
      server_ids: serverIds,
      parallel,
    }),
  );
}

export async function exportProjectServers(projectId: number, params?: { keyword?: string }): Promise<Blob> {
  return (await http.get(`/projects/${projectId}/servers/export`, { params, responseType: "blob" })) as unknown as Blob;
}

export async function importProjectServers(projectId: number, file: File) {
  const form = new FormData();
  form.append("file", file);
  return await getData(http.post<any, ApiResponse<{ imported: number }>>(`/projects/${projectId}/servers/import`, form, { headers: { "Content-Type": "multipart/form-data" } }));
}

export async function downloadProjectServersImportTemplate(projectId: number): Promise<Blob> {
  return (await http.get(`/projects/${projectId}/servers/import-template`, { responseType: "blob" })) as unknown as Blob;
}

export interface ServerGroupItem {
  id: number;
  project_id: number;
  parent_id?: number | null;
  name: string;
  category: "self_hosted" | "cloud" | string;
  provider: string;
  sort: number;
  status: number;
  children?: ServerGroupItem[];
}

export interface ServerGroupUpsertPayload {
  id?: number;
  project_id?: number;
  parent_id?: number | null;
  name: string;
  category: "self_hosted" | "cloud" | string;
  provider?: string;
  sort?: number;
  status?: number;
}

export async function getProjectServerGroupTree(projectId: number) {
  return await getData<ServerGroupItem[]>(http.get(`/projects/${projectId}/server-groups/tree`, { params: { project_id: projectId } }));
}

export async function upsertProjectServerGroup(projectId: number, payload: ServerGroupUpsertPayload) {
  return await getData<ServerGroupItem>(http.post(`/projects/${projectId}/server-groups`, payload));
}

export async function deleteProjectServerGroup(projectId: number, groupId: number) {
  return await getData<{ message: string }>(http.delete(`/projects/${projectId}/server-groups/${groupId}`));
}

export interface CloudAccountItem {
  id: number;
  project_id: number;
  group_id: number;
  provider: string;
  account_name: string;
  region_scope: string;
  status: number;
  last_sync_at?: string | null;
  last_sync_error?: string | null;
  created_at: string;
}

export interface CloudAccountUpsertPayload {
  id?: number;
  group_id: number;
  provider: string;
  account_name: string;
  region_scope?: string;
  ak?: string;
  sk?: string;
  status?: number;
}

export async function getProjectCloudAccounts(projectId: number, groupId?: number) {
  return await getData<CloudAccountItem[]>(
    http.get(`/projects/${projectId}/cloud-accounts`, { params: { project_id: projectId, group_id: groupId } }),
  );
}

export async function upsertProjectCloudAccount(projectId: number, payload: CloudAccountUpsertPayload) {
  return await getData<CloudAccountItem>(http.post(`/projects/${projectId}/cloud-accounts`, payload));
}

export interface CloudSyncResult {
  total: number;
  added: number;
  updated: number;
  disabled: number;
  unchanged: number;
  message: string;
}

export async function syncProjectCloudAccount(projectId: number, accountId: number) {
  return await getData<CloudSyncResult>(http.put(`/projects/${projectId}/cloud-accounts/${accountId}/sync`, {}));
}

export interface ServiceItem {
  id: number;
  server_id: number;
  name: string;
  env?: string | null;
  labels?: string | null;
  remark?: string | null;
  status: number;
  created_at: string;
}

export async function getProjectServices(projectId: number, params: { server_id?: number; keyword?: string; page?: number; page_size?: number }) {
  return await getData(
    http.get<any, ApiResponse<PageData<ServiceItem>>>(`/projects/${projectId}/services`, { params: { ...params, project_id: projectId } }),
  );
}

export async function upsertProjectService(projectId: number, payload: { id?: number; server_id: number; name: string; env?: string; labels?: string; remark?: string; status: number }) {
  return await getData(http.post<any, ApiResponse<ServiceItem>>(`/projects/${projectId}/services`, payload));
}

export async function deleteProjectService(projectId: number, serviceId: number) {
  return await getData(http.delete<any, ApiResponse<{ message: string }>>(`/projects/${projectId}/services/${serviceId}`));
}

export interface LogSourceItem {
  id: number;
  service_id: number;
  log_type: "file" | "journal" | string;
  path: string;
  encoding?: string | null;
  timezone?: string | null;
  multiline_rule?: string | null;
  include_regex?: string | null;
  exclude_regex?: string | null;
  status: number;
  created_at: string;
}

export async function getProjectLogSources(projectId: number, params: { service_id?: number; page?: number; page_size?: number }) {
  return await getData(
    http.get<any, ApiResponse<PageData<LogSourceItem>>>(`/projects/${projectId}/log-sources`, { params: { ...params, project_id: projectId } }),
  );
}

export async function upsertProjectLogSource(
  projectId: number,
  payload: { id?: number; service_id: number; log_type?: string; path: string; encoding?: string; timezone?: string; multiline_rule?: string; include_regex?: string; exclude_regex?: string; status: number },
) {
  return await getData(http.post<any, ApiResponse<LogSourceItem>>(`/projects/${projectId}/log-sources`, payload));
}

export async function deleteProjectLogSource(projectId: number, logSourceId: number) {
  return await getData(http.delete<any, ApiResponse<{ message: string }>>(`/projects/${projectId}/log-sources/${logSourceId}`));
}

export async function exportProjectLogs(
  projectId: number,
  params: { server_id: number; log_source_id: number; max_lines?: number; include?: string; exclude?: string },
): Promise<Blob> {
  return (await http.get(`/projects/${projectId}/logs/export`, { params, responseType: "blob" })) as unknown as Blob;
}

export async function listProjectLogFiles(projectId: number, params: { server_id: number; dir: string }) {
  return await getData(http.get<any, ApiResponse<{ list: string[] }>>(`/projects/${projectId}/log-files`, { params }));
}

export async function listProjectLogUnits(projectId: number, params: { server_id: number }) {
  return await getData(http.get<any, ApiResponse<{ list: string[] }>>(`/projects/${projectId}/log-units`, { params }));
}

export interface AgentDiscoveryItem {
  kind: "file" | "dir" | "unit" | string;
  value: string;
  extra?: string | null;
  last_seen_at: string;
}

export async function getProjectAgentDiscovery(
  projectId: number,
  params: { server_id: number; kind?: "file" | "dir" | "unit"; limit?: number },
) {
  return await getData(http.get<any, ApiResponse<{ list: AgentDiscoveryItem[] }>>(`/projects/${projectId}/agents/discovery`, { params }));
}

export interface ProjectAgentStatus {
  server_id: number;
  log_source_id: number;
  agent_id?: number;
  name?: string;
  version?: string;
  last_seen_at?: string | null;
  online: boolean;
  recent_publishing: boolean;
  mode_hint: "agent" | string;
  listen_port?: number;
  install_progress?: number;
  health_status?: string;
  last_error?: string;
}

export interface ProjectAgentListItem {
  server_id: number;
  server_name: string;
  server_host: string;
  /** 当前列表所属项目名称（与筛选项目一致） */
  project_name?: string;
  agent_id?: number;
  name?: string;
  version?: string;
  last_seen_at?: string | null;
  online: boolean;
  listen_port: number;
  install_progress: number;
  health_status: string;
  last_error?: string;
  recent_publishing?: boolean;
}

export interface ProjectAgentBootstrapPayload {
  server_id: number;
  log_source_id?: number;
  source_type?: "file" | "journal" | string;
  path?: string;
  platform_url: string;
  agent_name?: string;
  agent_version?: string;
}

export interface ProjectAgentBootstrapResult {
  agent_id: number;
  token: string;
  run_command: string;
  systemd_service: string;
}

export async function getProjectAgentStatus(projectId: number, params: { server_id: number; log_source_id?: number }) {
  return await getData(http.get<any, ApiResponse<ProjectAgentStatus>>(`/projects/${projectId}/agents/status`, { params }));
}

export async function listProjectAgents(projectId: number, params?: { keyword?: string; health_status?: string; online?: boolean }) {
  return await getData(http.get<any, ApiResponse<{ list: ProjectAgentListItem[] }>>(`/projects/${projectId}/agents/list`, { params }));
}

export async function batchRefreshProjectAgentHeartbeat(projectId: number, payload: { server_ids?: number[] }) {
  return await getData(
    http.post<any, ApiResponse<{ refreshed: number; list: ProjectAgentListItem[] }>>(
      `/projects/${projectId}/agents/heartbeat-refresh`,
      payload,
    ),
  );
}

export async function bootstrapProjectAgent(projectId: number, payload: ProjectAgentBootstrapPayload) {
  return await getData(http.post<any, ApiResponse<ProjectAgentBootstrapResult>>(`/projects/${projectId}/agents/bootstrap`, payload));
}

export async function rotateProjectAgentToken(projectId: number, payload: ProjectAgentBootstrapPayload) {
  return await getData(http.post<any, ApiResponse<ProjectAgentBootstrapResult>>(`/projects/${projectId}/agents/rotate-token`, payload));
}

