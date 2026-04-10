import { getData, http } from "./http";

export interface ConfigMapItem {
  name: string;
  namespace: string;
  data_count: number;
  creation_time: string;
}

export interface SecretItem {
  name: string;
  namespace: string;
  type: string;
  data_count: number;
  creation_time: string;
}

export interface ConfigDetail {
  yaml: string;
  decoded_data?: Record<string, string>;
  binary_keys?: string[];
}

export function listConfigMaps(clusterId: number, namespace: string, keyword?: string) {
  return getData<ConfigMapItem[]>(http.get("/configmaps", { params: { cluster_id: clusterId, namespace, keyword } }));
}
export function getConfigMapDetail(clusterId: number, namespace: string, name: string) {
  return getData<ConfigDetail>(http.get("/configmaps/detail", { params: { cluster_id: clusterId, namespace, name } }));
}
export function applyConfigMap(clusterId: number, manifest: string) {
  return getData<boolean>(http.post("/configmaps/apply", { cluster_id: clusterId, manifest }));
}
export function deleteConfigMap(clusterId: number, namespace: string, name: string) {
  return getData<boolean>(http.delete("/configmaps", { params: { cluster_id: clusterId, namespace, name } }));
}

export function listSecrets(clusterId: number, namespace: string, keyword?: string) {
  return getData<SecretItem[]>(http.get("/secrets", { params: { cluster_id: clusterId, namespace, keyword } }));
}
export function getSecretDetail(clusterId: number, namespace: string, name: string) {
  return getData<ConfigDetail>(http.get("/secrets/detail", { params: { cluster_id: clusterId, namespace, name } }));
}
export function applySecret(clusterId: number, manifest: string) {
  return getData<boolean>(http.post("/secrets/apply", { cluster_id: clusterId, manifest }));
}
export function deleteSecret(clusterId: number, namespace: string, name: string) {
  return getData<boolean>(http.delete("/secrets", { params: { cluster_id: clusterId, namespace, name } }));
}

