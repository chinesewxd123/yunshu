import { getData, http } from "./http";

export interface WorkloadItem {
  name: string;
  namespace: string;
  ready?: string;
  replicas?: string;
  available?: string;
  updated?: string;
  ready_percent?: number;
  resource_text?: string;
  containers_text?: string;
  conditions_text?: string;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  active?: string;
  failed?: string;
  start_time?: string;
  completion_time?: string;
  age?: string;
  creation_time: string;
}

export interface WorkloadDetail {
  yaml: string;
  object?: any;
}

export interface RelatedPodItem {
  name: string;
  namespace: string;
  phase: string;
  node_name: string;
  pod_ip: string;
  restart_count: number;
  start_time?: string;
}

export interface CronJobItemV2 {
  name: string;
  namespace: string;
  schedule: string;
  suspend: boolean;
  ready_percent?: number;
  resource_text?: string;
  containers_text?: string;
  conditions_text?: string;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  last_schedule_time?: string;
  last_successful_time?: string;
  active_count?: string;
  age?: string;
  creation_time: string;
}

export function listDeployments(clusterId: number, namespace: string, keyword?: string) {
  return getData<WorkloadItem[]>(
    http.get("/deployments", { params: { cluster_id: clusterId, namespace, keyword } }),
  );
}
export function getDeploymentDetail(clusterId: number, namespace: string, name: string) {
  return getData<WorkloadDetail>(http.get("/deployments/detail", { params: { cluster_id: clusterId, namespace, name } }));
}
export function applyDeployment(clusterId: number, manifest: string) {
  return getData<boolean>(http.post("/deployments/apply", { cluster_id: clusterId, manifest }));
}
export function deleteDeployment(clusterId: number, namespace: string, name: string) {
  return getData<boolean>(http.delete("/deployments", { params: { cluster_id: clusterId, namespace, name } }));
}
export function scaleDeployment(clusterId: number, namespace: string, name: string, replicas: number) {
  return getData<boolean>(http.post("/deployments/scale", { cluster_id: clusterId, namespace, name, replicas }));
}
export function restartDeployment(clusterId: number, namespace: string, name: string) {
  return getData<boolean>(http.post("/deployments/restart", { cluster_id: clusterId, namespace, name }));
}
export function listDeploymentPods(clusterId: number, namespace: string, name: string) {
  return getData<RelatedPodItem[]>(http.get("/deployments/pods", { params: { cluster_id: clusterId, namespace, name } }));
}

export function listStatefulSets(clusterId: number, namespace: string, keyword?: string) {
  return getData<WorkloadItem[]>(
    http.get("/statefulsets", { params: { cluster_id: clusterId, namespace, keyword } }),
  );
}
export function getStatefulSetDetail(clusterId: number, namespace: string, name: string) {
  return getData<WorkloadDetail>(
    http.get("/statefulsets/detail", { params: { cluster_id: clusterId, namespace, name } }),
  );
}
export function applyStatefulSet(clusterId: number, manifest: string) {
  return getData<boolean>(http.post("/statefulsets/apply", { cluster_id: clusterId, manifest }));
}
export function deleteStatefulSet(clusterId: number, namespace: string, name: string) {
  return getData<boolean>(http.delete("/statefulsets", { params: { cluster_id: clusterId, namespace, name } }));
}
export function scaleStatefulSet(clusterId: number, namespace: string, name: string, replicas: number) {
  return getData<boolean>(http.post("/statefulsets/scale", { cluster_id: clusterId, namespace, name, replicas }));
}
export function restartStatefulSet(clusterId: number, namespace: string, name: string) {
  return getData<boolean>(http.post("/statefulsets/restart", { cluster_id: clusterId, namespace, name }));
}
export function listStatefulSetPods(clusterId: number, namespace: string, name: string) {
  return getData<RelatedPodItem[]>(
    http.get("/statefulsets/pods", { params: { cluster_id: clusterId, namespace, name } }),
  );
}

export function listDaemonSets(clusterId: number, namespace: string, keyword?: string) {
  return getData<WorkloadItem[]>(
    http.get("/daemonsets", { params: { cluster_id: clusterId, namespace, keyword } }),
  );
}
export function getDaemonSetDetail(clusterId: number, namespace: string, name: string) {
  return getData<WorkloadDetail>(http.get("/daemonsets/detail", { params: { cluster_id: clusterId, namespace, name } }));
}
export function applyDaemonSet(clusterId: number, manifest: string) {
  return getData<boolean>(http.post("/daemonsets/apply", { cluster_id: clusterId, manifest }));
}
export function deleteDaemonSet(clusterId: number, namespace: string, name: string) {
  return getData<boolean>(http.delete("/daemonsets", { params: { cluster_id: clusterId, namespace, name } }));
}
export function restartDaemonSet(clusterId: number, namespace: string, name: string) {
  return getData<boolean>(http.post("/daemonsets/restart", { cluster_id: clusterId, namespace, name }));
}
export function listDaemonSetPods(clusterId: number, namespace: string, name: string) {
  return getData<RelatedPodItem[]>(http.get("/daemonsets/pods", { params: { cluster_id: clusterId, namespace, name } }));
}

export function listJobs(clusterId: number, namespace: string, keyword?: string) {
  return getData<WorkloadItem[]>(http.get("/jobs", { params: { cluster_id: clusterId, namespace, keyword } }));
}
export function getJobDetail(clusterId: number, namespace: string, name: string) {
  return getData<WorkloadDetail>(http.get("/jobs/detail", { params: { cluster_id: clusterId, namespace, name } }));
}
export function applyJob(clusterId: number, manifest: string) {
  return getData<boolean>(http.post("/jobs/apply", { cluster_id: clusterId, manifest }));
}
export function deleteJob(clusterId: number, namespace: string, name: string) {
  return getData<boolean>(http.delete("/jobs", { params: { cluster_id: clusterId, namespace, name } }));
}
export function rerunJob(clusterId: number, namespace: string, name: string) {
  return getData<{ job_name: string }>(http.post("/jobs/rerun", { cluster_id: clusterId, namespace, name }));
}
export function listJobPods(clusterId: number, namespace: string, name: string) {
  return getData<RelatedPodItem[]>(http.get("/jobs/pods", { params: { cluster_id: clusterId, namespace, name } }));
}

export function listCronJobs(clusterId: number, namespace: string, keyword?: string) {
  return getData<WorkloadItem[]>(http.get("/cronjobs", { params: { cluster_id: clusterId, namespace, keyword } }));
}
export function listCronJobsV2(clusterId: number, namespace: string, keyword?: string) {
  return getData<CronJobItemV2[]>(http.get("/cronjobs/v2", { params: { cluster_id: clusterId, namespace, keyword } }));
}
export function getCronJobDetail(clusterId: number, namespace: string, name: string) {
  return getData<WorkloadDetail>(http.get("/cronjobs/detail", { params: { cluster_id: clusterId, namespace, name } }));
}
export function applyCronJob(clusterId: number, manifest: string) {
  return getData<boolean>(http.post("/cronjobs/apply", { cluster_id: clusterId, manifest }));
}
export function deleteCronJob(clusterId: number, namespace: string, name: string) {
  return getData<boolean>(http.delete("/cronjobs", { params: { cluster_id: clusterId, namespace, name } }));
}
export function suspendCronJob(clusterId: number, namespace: string, name: string, suspend: boolean) {
  return getData<boolean>(http.post("/cronjobs/suspend", { cluster_id: clusterId, namespace, name, suspend }));
}
export function triggerCronJob(clusterId: number, namespace: string, name: string) {
  return getData<{ job_name: string }>(http.post("/cronjobs/trigger", { cluster_id: clusterId, namespace, name }));
}
export function listCronJobPods(clusterId: number, namespace: string, name: string) {
  return getData<RelatedPodItem[]>(http.get("/cronjobs/pods", { params: { cluster_id: clusterId, namespace, name } }));
}

