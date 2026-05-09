import { createK8sResourceService, k8sParams } from "./service-factory";

export type HPAItem = {
  name: string;
  namespace: string;
  min_replicas?: string;
  max_replicas?: string;
  scale_target_ref?: string;
  metrics_summary?: string;
  conditions_text?: string;
  labels?: Record<string, string>;
  age?: string;
  creation_time: string;
};

export type HPADetail = { yaml: string };

const svc = createK8sResourceService<HPAItem, HPADetail>("/horizontal-pod-autoscalers");

export function listHPA(clusterId: number, namespace: string, keyword?: string) {
  return svc.list(k8sParams(clusterId, { namespace, keyword }));
}

export function getHPADetail(clusterId: number, namespace: string, name: string) {
  return svc.detail(k8sParams(clusterId, { namespace, name }));
}

export function applyHPA(clusterId: number, manifest: string) {
  return svc.apply({ cluster_id: clusterId, manifest });
}

export function deleteHPA(clusterId: number, namespace: string, name: string) {
  return svc.remove(k8sParams(clusterId, { namespace, name }));
}
