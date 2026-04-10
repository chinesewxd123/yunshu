import { getData, http } from "./http";

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

export function listK8sServices(clusterId: number, namespace: string, keyword?: string) {
  return getData<K8sServiceItem[]>(
    http.get("/k8s-services", { params: { cluster_id: clusterId, namespace, keyword } }),
  );
}

export function getK8sServiceDetail(clusterId: number, namespace: string, name: string) {
  return getData<K8sServiceDetail>(
    http.get("/k8s-services/detail", { params: { cluster_id: clusterId, namespace, name } }),
  );
}

export function applyK8sService(clusterId: number, manifest: string) {
  return getData<boolean>(http.post("/k8s-services/apply", { cluster_id: clusterId, manifest }));
}

export function deleteK8sService(clusterId: number, namespace: string, name: string) {
  return getData<boolean>(http.delete("/k8s-services", { params: { cluster_id: clusterId, namespace, name } }));
}

