import { getData, http } from "./http";

export interface IngressItem {
  name: string;
  namespace: string;
  class_name?: string;
  rules_text?: string;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  host_count: number;
  tls_count: number;
  load_balancer?: string;
  age?: string;
  creation_time: string;
}

export interface IngressDetail {
  yaml: string;
}

export interface IngressClassItem {
  name: string;
  controller?: string;
  ingress_count: number;
  is_default: boolean;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  age?: string;
  creation_time: string;
}

export interface IngressClassDetail {
  yaml: string;
}

export interface IngressNginxRestartResult {
  deleted_count: number;
  deleted_names: string[];
}

export function listIngresses(clusterId: number, namespace: string, keyword?: string) {
  return getData<IngressItem[]>(http.get("/ingresses", { params: { cluster_id: clusterId, namespace, keyword } }));
}

export function getIngressDetail(clusterId: number, namespace: string, name: string) {
  return getData<IngressDetail>(http.get("/ingresses/detail", { params: { cluster_id: clusterId, namespace, name } }));
}

export function applyIngress(clusterId: number, manifest: string) {
  return getData<boolean>(http.post("/ingresses/apply", { cluster_id: clusterId, manifest }));
}

export function restartIngressNginxPods(clusterId: number, namespace?: string, selector?: string) {
  return getData<IngressNginxRestartResult>(http.post("/ingresses/nginx/restart", { cluster_id: clusterId, namespace, selector }));
}

export function deleteIngress(clusterId: number, namespace: string, name: string) {
  return getData<boolean>(http.delete("/ingresses", { params: { cluster_id: clusterId, namespace, name } }));
}

export function listIngressClasses(clusterId: number, keyword?: string) {
  return getData<IngressClassItem[]>(http.get("/ingresses/classes", { params: { cluster_id: clusterId, keyword } }));
}

export function getIngressClassDetail(clusterId: number, name: string) {
  return getData<IngressClassDetail>(http.get("/ingresses/classes/detail", { params: { cluster_id: clusterId, name } }));
}

export function applyIngressClass(clusterId: number, manifest: string) {
  return getData<boolean>(http.post("/ingresses/classes/apply", { cluster_id: clusterId, manifest }));
}

export function deleteIngressClass(clusterId: number, name: string) {
  return getData<boolean>(http.delete("/ingresses/classes", { params: { cluster_id: clusterId, name } }));
}
