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
  cpu_usage?: string;
  mem_usage?: string;
  cpu_pct_request?: number;
  cpu_pct_limit?: number;
  mem_pct_request?: number;
  mem_pct_limit?: number;
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

/** 联调常用：从表单拼接 cpu/memory 的 requests/limits（仅非空字段写入） */
export function buildCpuMemoryResourceMaps(form: {
  requests_cpu?: string;
  requests_memory?: string;
  limits_cpu?: string;
  limits_memory?: string;
}): { requests: Record<string, string>; limits: Record<string, string> } {
  const requests: Record<string, string> = {};
  const limits: Record<string, string> = {};
  const t = (s?: string) => (s ?? "").trim();
  if (t(form.requests_cpu)) requests.cpu = t(form.requests_cpu);
  if (t(form.requests_memory)) requests.memory = t(form.requests_memory);
  if (t(form.limits_cpu)) limits.cpu = t(form.limits_cpu);
  if (t(form.limits_memory)) limits.memory = t(form.limits_memory);
  return { requests, limits };
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
  cpu_usage?: string;
  mem_usage?: string;
  cpu_pct_request?: number;
  cpu_pct_limit?: number;
  mem_pct_request?: number;
  mem_pct_limit?: number;
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

/** 垂直扩缩：更新 Pod 模板内指定容器的 requests/limits（key 为 cpu、memory 等；空字符串可删除该键） */
export function patchDeploymentContainerResources(
  clusterId: number,
  namespace: string,
  name: string,
  body: { container_name?: string; requests?: Record<string, string>; limits?: Record<string, string> },
) {
  return deploymentsSvc.post<boolean>("/container-resources", {
    cluster_id: clusterId,
    namespace,
    name,
    container_name: body.container_name ?? "",
    requests: body.requests,
    limits: body.limits,
  });
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

export function patchStatefulSetContainerResources(
  clusterId: number,
  namespace: string,
  name: string,
  body: { container_name?: string; requests?: Record<string, string>; limits?: Record<string, string> },
) {
  return statefulsetsSvc.post<boolean>("/container-resources", {
    cluster_id: clusterId,
    namespace,
    name,
    container_name: body.container_name ?? "",
    requests: body.requests,
    limits: body.limits,
  });
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

/** DaemonSet 仅支持垂直扩缩（修改 Pod 模板 resources）；不支持水平副本扩缩（与 HPA scale 语义一致）。 */
export function patchDaemonSetContainerResources(
  clusterId: number,
  namespace: string,
  name: string,
  body: { container_name?: string; requests?: Record<string, string>; limits?: Record<string, string> },
) {
  return daemonsetsSvc.post<boolean>("/container-resources", {
    cluster_id: clusterId,
    namespace,
    name,
    container_name: body.container_name ?? "",
    requests: body.requests,
    limits: body.limits,
  });
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

/** Job 不支持 HPA 式水平副本扩缩；仅可改 Pod 模板 resources（对齐 VPA 纳管范围，批量任务建议配合 VPA Initial/Off 等策略）。 */
export function patchJobContainerResources(
  clusterId: number,
  namespace: string,
  name: string,
  body: { container_name?: string; requests?: Record<string, string>; limits?: Record<string, string> },
) {
  return jobsSvc.post<boolean>("/container-resources", {
    cluster_id: clusterId,
    namespace,
    name,
    container_name: body.container_name ?? "",
    requests: body.requests,
    limits: body.limits,
  });
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

/** CronJob 不支持水平副本扩缩；修改 jobTemplate 内 resources 主要影响后续创建的 Job（类似 VPA 对模板资源的调整）。 */
export function patchCronJobContainerResources(
  clusterId: number,
  namespace: string,
  name: string,
  body: { container_name?: string; requests?: Record<string, string>; limits?: Record<string, string> },
) {
  return cronjobsSvc.post<boolean>("/container-resources", {
    cluster_id: clusterId,
    namespace,
    name,
    container_name: body.container_name ?? "",
    requests: body.requests,
    limits: body.limits,
  });
}

export function listCronJobPods(clusterId: number, namespace: string, name: string) {
  return cronjobsSvc.get<RelatedPodItem[]>("/pods", k8sParams(clusterId, { namespace, name }));
}

