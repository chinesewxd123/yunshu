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

export interface ServerItem {
  id: number;
  project_id: number;
  name: string;
  host: string;
  port: number;
  os_type: string;
  os_arch: string;
  tags: string;
  status: number;
  created_at: string;
  last_seen_at?: string | null;
  last_test_at?: string | null;
  last_test_error?: string | null;
}

export interface ServerUpsertPayload {
  id?: number;
  project_id: number;
  name: string;
  host: string;
  port?: number;
  os_type?: string;
  tags?: string;
  status: number;

  auth_type?: "password" | "key";
  username?: string;
  password?: string;
  private_key?: string;
  passphrase?: string;
}

export async function getProjectServers(projectId: number, params: { keyword?: string; page?: number; page_size?: number }) {
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

export async function testProjectServer(projectId: number, serverId: number) {
  return await getData(http.post<any, ApiResponse<{ ok: boolean; message: string }>>(`/projects/${projectId}/servers/test`, { server_id: serverId }));
}

export async function exportProjectServers(projectId: number, params?: { keyword?: string }): Promise<Blob> {
  return (await http.get(`/projects/${projectId}/servers/export`, { params, responseType: "blob" })) as unknown as Blob;
}

export async function importProjectServers(projectId: number, file: File) {
  const form = new FormData();
  form.append("file", file);
  return await getData(http.post<any, ApiResponse<{ imported: number }>>(`/projects/${projectId}/servers/import`, form, { headers: { "Content-Type": "multipart/form-data" } }));
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

export async function bootstrapProjectAgent(projectId: number, payload: ProjectAgentBootstrapPayload) {
  return await getData(http.post<any, ApiResponse<ProjectAgentBootstrapResult>>(`/projects/${projectId}/agents/bootstrap`, payload));
}

export async function rotateProjectAgentToken(projectId: number, payload: ProjectAgentBootstrapPayload) {
  return await getData(http.post<any, ApiResponse<ProjectAgentBootstrapResult>>(`/projects/${projectId}/agents/rotate-token`, payload));
}

