import { getData, http } from "./http";

export type MysqlBackupScope = "all" | "database" | "table";

export interface MysqlBackupInstance {
  id: number;
  project_id: number;
  server_id: number;
  server_name?: string;
  name: string;
  enabled: boolean;
  mysql_host: string;
  mysql_port: number;
  mysql_user: string;
  backup_mode: "mysqldump" | "remote_check" | string;
  backup_scope?: MysqlBackupScope | string;
  database_name?: string;
  table_name?: string;
  database_names?: string;
  remote_data_dir?: string;
  remote_log_dir?: string;
  mysql_datadir?: string;
  upload_to_minio?: boolean;
  mysqldump_work_dir?: string;
  mysqldump_options?: string[];
  mysqldump_extra_args?: string;
  schedule_enabled?: boolean;
  cron_spec?: string;
  last_scheduled_at?: string;
  created_at?: string;
  updated_at?: string;
}

export type MysqlBackupInstancePayload = {
  server_id: number;
  name: string;
  enabled?: boolean;
  mysql_host?: string;
  mysql_port?: number;
  mysql_user: string;
  mysql_password?: string;
  backup_mode?: string;
  backup_scope?: MysqlBackupScope | string;
  database_name?: string;
  table_name?: string;
  database_names?: string;
  remote_data_dir?: string;
  remote_log_dir?: string;
  mysql_datadir?: string;
  upload_to_minio?: boolean;
  mysqldump_work_dir?: string;
  mysqldump_options?: string[];
  mysqldump_extra_args?: string;
  schedule_enabled?: boolean;
  cron_spec?: string;
};

export type MysqldumpOptionItem = { id: string; label: string; flag: string; group?: string };

export function listMysqldumpOptions(projectId: number) {
  return getData<MysqldumpOptionItem[]>(http.get(`/projects/${projectId}/mysql-backup/mysqldump-options`));
}

export interface MysqlBackupJob {
  id: number;
  instance_id: number;
  project_id: number;
  status: string;
  backup_mode?: string;
  trigger_type?: string;
  backup_scope?: string;
  database_name?: string;
  table_name?: string;
  remote_path?: string;
  minio_bucket?: string;
  minio_object?: string;
  file_size?: number;
  check_ok?: boolean;
  log_excerpt?: string;
  error_message?: string;
  started_at?: string;
  finished_at?: string;
  created_at?: string;
}

export function listMysqlBackupInstances(projectId: number, params?: { page?: number; page_size?: number }) {
  return getData<{ list: MysqlBackupInstance[]; total: number }>(
    http.get(`/projects/${projectId}/mysql-backup/instances`, { params }),
  );
}

export function createMysqlBackupInstance(projectId: number, payload: MysqlBackupInstancePayload) {
  return getData<MysqlBackupInstance>(http.post(`/projects/${projectId}/mysql-backup/instances`, payload));
}

export function updateMysqlBackupInstance(projectId: number, instanceId: number, payload: MysqlBackupInstancePayload) {
  return getData<MysqlBackupInstance>(http.put(`/projects/${projectId}/mysql-backup/instances/${instanceId}`, payload));
}

export function deleteMysqlBackupInstance(projectId: number, instanceId: number) {
  return getData<{ deleted: boolean }>(http.delete(`/projects/${projectId}/mysql-backup/instances/${instanceId}`));
}

export function pingMysqlBackupInstance(projectId: number, instanceId: number) {
  return getData<{ ok: boolean; message: string }>(http.post(`/projects/${projectId}/mysql-backup/instances/${instanceId}/ping`));
}

export function checkMysqlRemoteBackup(projectId: number, instanceId: number) {
  return getData<{ ok: boolean; message: string; backup_file?: string; log_file?: string }>(
    http.post(`/projects/${projectId}/mysql-backup/instances/${instanceId}/check-remote`),
  );
}

export function runMysqlBackup(projectId: number, instanceId: number) {
  return getData<MysqlBackupJob>(http.post(`/projects/${projectId}/mysql-backup/instances/${instanceId}/run`));
}

export function listMysqlBackupJobs(projectId: number, params?: { instance_id?: number; page?: number; page_size?: number }) {
  return getData<{ list: MysqlBackupJob[]; total: number; page: number; page_size: number }>(
    http.get(`/projects/${projectId}/mysql-backup/jobs`, { params }),
  );
}

export function presignMysqlBackupJob(projectId: number, jobId: number) {
  return getData<{ url: string }>(http.get(`/projects/${projectId}/mysql-backup/jobs/${jobId}/presign`));
}
