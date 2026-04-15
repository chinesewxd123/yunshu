import { createK8sResourceService, k8sParams } from "./service-factory";

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

const deploymentsSvc = createK8sResourceService<WorkloadItem, WorkloadDetail>("/deployments");
const statefulsetsSvc = createK8sResourceService<WorkloadItem, WorkloadDetail>("/statefulsets");
const daemonsetsSvc = createK8sResourceService<WorkloadItem, WorkloadDetail>("/daemonsets");
const jobsSvc = createK8sResourceService<WorkloadItem, WorkloadDetail>("/jobs");
const cronjobsSvc = createK8sResourceService<WorkloadItem, WorkloadDetail>("/cronjobs");

export function listDeployments(clusterId: number, namespace: string, keyword?: string) {
  return deploymentsSvc.list(k8sParams(clusterId, { namespace, keyword }));
}
export function getDeploymentDetail(clusterId: number, namespace: string, name: string) {
  return deploymentsSvc.detail(k8sParams(clusterId, { namespace, name }));
}
export function applyDeployment(clusterId: number, manifest: string) {
  return deploymentsSvc.apply({ cluster_id: clusterId, manifest });
}
export function deleteDeployment(clusterId: number, namespace: string, name: string) {
  return deploymentsSvc.remove(k8sParams(clusterId, { namespace, name }));
}
export function scaleDeployment(clusterId: number, namespace: string, name: string, replicas: number) {
  return deploymentsSvc.post<boolean>("/scale", { cluster_id: clusterId, namespace, name, replicas });
}
export function restartDeployment(clusterId: number, namespace: string, name: string) {
  return deploymentsSvc.post<boolean>("/restart", { cluster_id: clusterId, namespace, name });
}
export function listDeploymentPods(clusterId: number, namespace: string, name: string) {
  return deploymentsSvc.get<RelatedPodItem[]>("/pods", k8sParams(clusterId, { namespace, name }));
}

export function listStatefulSets(clusterId: number, namespace: string, keyword?: string) {
  return statefulsetsSvc.list(k8sParams(clusterId, { namespace, keyword }));
}
export function getStatefulSetDetail(clusterId: number, namespace: string, name: string) {
  return statefulsetsSvc.detail(k8sParams(clusterId, { namespace, name }));
}
export function applyStatefulSet(clusterId: number, manifest: string) {
  return statefulsetsSvc.apply({ cluster_id: clusterId, manifest });
}
export function deleteStatefulSet(clusterId: number, namespace: string, name: string) {
  return statefulsetsSvc.remove(k8sParams(clusterId, { namespace, name }));
}
export function scaleStatefulSet(clusterId: number, namespace: string, name: string, replicas: number) {
  return statefulsetsSvc.post<boolean>("/scale", { cluster_id: clusterId, namespace, name, replicas });
}
export function restartStatefulSet(clusterId: number, namespace: string, name: string) {
  return statefulsetsSvc.post<boolean>("/restart", { cluster_id: clusterId, namespace, name });
}
export function listStatefulSetPods(clusterId: number, namespace: string, name: string) {
  return statefulsetsSvc.get<RelatedPodItem[]>("/pods", k8sParams(clusterId, { namespace, name }));
}

export function listDaemonSets(clusterId: number, namespace: string, keyword?: string) {
  return daemonsetsSvc.list(k8sParams(clusterId, { namespace, keyword }));
}
export function getDaemonSetDetail(clusterId: number, namespace: string, name: string) {
  return daemonsetsSvc.detail(k8sParams(clusterId, { namespace, name }));
}
export function applyDaemonSet(clusterId: number, manifest: string) {
  return daemonsetsSvc.apply({ cluster_id: clusterId, manifest });
}
export function deleteDaemonSet(clusterId: number, namespace: string, name: string) {
  return daemonsetsSvc.remove(k8sParams(clusterId, { namespace, name }));
}
export function restartDaemonSet(clusterId: number, namespace: string, name: string) {
  return daemonsetsSvc.post<boolean>("/restart", { cluster_id: clusterId, namespace, name });
}
export function listDaemonSetPods(clusterId: number, namespace: string, name: string) {
  return daemonsetsSvc.get<RelatedPodItem[]>("/pods", k8sParams(clusterId, { namespace, name }));
}

export function listJobs(clusterId: number, namespace: string, keyword?: string) {
  return jobsSvc.list(k8sParams(clusterId, { namespace, keyword }));
}
export function getJobDetail(clusterId: number, namespace: string, name: string) {
  return jobsSvc.detail(k8sParams(clusterId, { namespace, name }));
}
export function applyJob(clusterId: number, manifest: string) {
  return jobsSvc.apply({ cluster_id: clusterId, manifest });
}
export function deleteJob(clusterId: number, namespace: string, name: string) {
  return jobsSvc.remove(k8sParams(clusterId, { namespace, name }));
}
export function rerunJob(clusterId: number, namespace: string, name: string) {
  return jobsSvc.post<{ job_name: string }>("/rerun", { cluster_id: clusterId, namespace, name });
}
export function listJobPods(clusterId: number, namespace: string, name: string) {
  return jobsSvc.get<RelatedPodItem[]>("/pods", k8sParams(clusterId, { namespace, name }));
}

export function listCronJobs(clusterId: number, namespace: string, keyword?: string) {
  return cronjobsSvc.list(k8sParams(clusterId, { namespace, keyword }));
}
export function listCronJobsV2(clusterId: number, namespace: string, keyword?: string) {
  return cronjobsSvc.get<CronJobItemV2[]>("/v2", k8sParams(clusterId, { namespace, keyword }));
}
export function getCronJobDetail(clusterId: number, namespace: string, name: string) {
  return cronjobsSvc.detail(k8sParams(clusterId, { namespace, name }));
}
export function applyCronJob(clusterId: number, manifest: string) {
  return cronjobsSvc.apply({ cluster_id: clusterId, manifest });
}
export function deleteCronJob(clusterId: number, namespace: string, name: string) {
  return cronjobsSvc.remove(k8sParams(clusterId, { namespace, name }));
}
export function suspendCronJob(clusterId: number, namespace: string, name: string, suspend: boolean) {
  return cronjobsSvc.post<boolean>("/suspend", { cluster_id: clusterId, namespace, name, suspend });
}
export function triggerCronJob(clusterId: number, namespace: string, name: string) {
  return cronjobsSvc.post<{ job_name: string }>("/trigger", { cluster_id: clusterId, namespace, name });
}
export function listCronJobPods(clusterId: number, namespace: string, name: string) {
  return cronjobsSvc.get<RelatedPodItem[]>("/pods", k8sParams(clusterId, { namespace, name }));
}

