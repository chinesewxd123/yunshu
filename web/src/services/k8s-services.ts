import { createK8sResourceService, k8sParams } from "./service-factory";

export type K8sServiceItem = {
  name: string;
  namespace: string;
  type: string;
  internal_traffic?: string;
  cluster_ip?: string;
  external_ips?: string;
  ports?: string;
  ip_families?: string;
  ip_family_policy?: string;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  selectors?: Record<string, string>;
  selector_count: number;
  session_affinity?: string;
  age?: string;
  creation_time: string;
};

export type K8sServiceDetail = { yaml: string };

const svc = createK8sResourceService<K8sServiceItem, K8sServiceDetail>("/k8s-services");

export function listK8sServices(clusterId: number, namespace: string, keyword?: string) {
  return svc.list(k8sParams(clusterId, { namespace, keyword }));
}

export function getK8sServiceDetail(clusterId: number, namespace: string, name: string) {
  return svc.detail(k8sParams(clusterId, { namespace, name }));
}

export function applyK8sService(clusterId: number, manifest: string) {
  return svc.apply({ cluster_id: clusterId, manifest });
}

export function deleteK8sService(clusterId: number, namespace: string, name: string) {
  return svc.remove(k8sParams(clusterId, { namespace, name }));
}

