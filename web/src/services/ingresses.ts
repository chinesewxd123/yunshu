import { createK8sResourceService, k8sParams } from "./service-factory";

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

const ingressesSvc = createK8sResourceService<IngressItem, IngressDetail>("/ingresses");
const ingressClassesSvc = createK8sResourceService<IngressClassItem, IngressClassDetail>("/ingresses/classes");

export function listIngresses(clusterId: number, namespace: string, keyword?: string) {
  return ingressesSvc.list(k8sParams(clusterId, { namespace, keyword }));
}

export function getIngressDetail(clusterId: number, namespace: string, name: string) {
  return ingressesSvc.detail(k8sParams(clusterId, { namespace, name }));
}

export function applyIngress(clusterId: number, manifest: string) {
  return ingressesSvc.apply({ cluster_id: clusterId, manifest });
}

export function restartIngressNginxPods(clusterId: number, namespace?: string, selector?: string) {
  return ingressesSvc.post<IngressNginxRestartResult>("/nginx/restart", { cluster_id: clusterId, namespace, selector });
}

export function deleteIngress(clusterId: number, namespace: string, name: string) {
  return ingressesSvc.remove(k8sParams(clusterId, { namespace, name }));
}

export function listIngressClasses(clusterId: number, keyword?: string) {
  return ingressClassesSvc.list(k8sParams(clusterId, { keyword }));
}

export function getIngressClassDetail(clusterId: number, name: string) {
  return ingressClassesSvc.detail(k8sParams(clusterId, { name }));
}

export function applyIngressClass(clusterId: number, manifest: string) {
  return ingressClassesSvc.apply({ cluster_id: clusterId, manifest });
}

export function deleteIngressClass(clusterId: number, name: string) {
  return ingressClassesSvc.remove(k8sParams(clusterId, { name }));
}
