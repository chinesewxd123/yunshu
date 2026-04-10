import { getData, http } from "./http";

export type RbacRoleItem = {
  name: string;
  namespace: string;
  rules: number;
  creation_time: string;
};

export type RbacRoleBindingItem = {
  name: string;
  namespace: string;
  role_ref: string;
  subjects?: string[];
  creation_time: string;
};

export type RbacClusterRoleItem = {
  name: string;
  rules: number;
  creation_time: string;
};

export type RbacClusterRoleBindingItem = {
  name: string;
  role_ref: string;
  subjects?: string[];
  creation_time: string;
};

export type RbacDetail = {
  kind: string;
  name: string;
  namespace?: string;
  yaml: string;
};

export function listRoles(clusterId: number, namespace: string, keyword?: string) {
  return getData<{ list: RbacRoleItem[] }>(http.get("/rbac/roles", { params: { cluster_id: clusterId, namespace, keyword } }));
}

export function listRoleBindings(clusterId: number, namespace: string, keyword?: string) {
  return getData<{ list: RbacRoleBindingItem[] }>(http.get("/rbac/rolebindings", { params: { cluster_id: clusterId, namespace, keyword } }));
}

export function listClusterRoles(clusterId: number, keyword?: string) {
  return getData<{ list: RbacClusterRoleItem[] }>(http.get("/rbac/clusterroles", { params: { cluster_id: clusterId, keyword } }));
}

export function listClusterRoleBindings(clusterId: number, keyword?: string) {
  return getData<{ list: RbacClusterRoleBindingItem[] }>(http.get("/rbac/clusterrolebindings", { params: { cluster_id: clusterId, keyword } }));
}

export function getRbacDetail(params: { cluster_id: number; kind: string; name: string; namespace?: string }) {
  return getData<RbacDetail>(http.get("/rbac/detail", { params }));
}

export function applyRbac(clusterId: number, manifest: string) {
  return getData<boolean>(http.post("/rbac/apply", { cluster_id: clusterId, manifest }));
}

export function deleteRbac(params: { cluster_id: number; kind: string; name: string; namespace?: string }) {
  return getData<boolean>(http.delete("/rbac", { params }));
}

