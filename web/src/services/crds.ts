import { createK8sResourceService, k8sParams } from "./service-factory";

export interface CrdItem {
  name: string;
  group: string;
  scope: string;
  kind: string;
  plural: string;
  current_version: string;
  established: boolean;
  creation_time: string;
}

export interface CrdDetail {
  yaml: string;
}

const crdsSvc = createK8sResourceService<CrdItem, CrdDetail>("/crds");

export function listCrds(clusterId: number, keyword?: string) {
  return crdsSvc.list(k8sParams(clusterId, { keyword }));
}

export function getCrdDetail(clusterId: number, name: string) {
  return crdsSvc.detail(k8sParams(clusterId, { name }));
}

export function applyCrd(clusterId: number, manifest: string) {
  return crdsSvc.apply({ cluster_id: clusterId, manifest });
}

export function deleteCrd(clusterId: number, name: string) {
  return crdsSvc.remove(k8sParams(clusterId, { name }));
}
