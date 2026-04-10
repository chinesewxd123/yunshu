import { getData, http } from "./http";

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

export function listCrds(clusterId: number, keyword?: string) {
  return getData<CrdItem[]>(http.get("/crds", { params: { cluster_id: clusterId, keyword } }));
}

export function getCrdDetail(clusterId: number, name: string) {
  return getData<CrdDetail>(http.get("/crds/detail", { params: { cluster_id: clusterId, name } }));
}

export function applyCrd(clusterId: number, manifest: string) {
  return getData<boolean>(http.post("/crds/apply", { cluster_id: clusterId, manifest }));
}

export function deleteCrd(clusterId: number, name: string) {
  return getData<boolean>(http.delete("/crds", { params: { cluster_id: clusterId, name } }));
}
