import { getData, http } from "./http";
import type { UserItem } from "../types/api";

export type PersistentVolumeItem = {
  name: string;
  capacity: string;
  access_modes: string;
  reclaim_policy: string;
  status: string;
  claim?: string;
  storage_class?: string;
  creation_time: string;
};

export type PersistentVolumeClaimItem = {
  name: string;
  namespace: string;
  status: string;
  volume?: string;
  capacity?: string;
  access_modes?: string;
  storage_class?: string;
  creation_time: string;
};

export type StorageClassItem = {
  name: string;
  provisioner: string;
  reclaim_policy?: string;
  volume_binding_mode?: string;
  allow_volume_expansion: boolean;
  creation_time: string;
};

export type StorageDetail = { yaml: string };

export function listPersistentVolumes(clusterId: number, keyword?: string) {
  return getData<PersistentVolumeItem[]>(http.get("/persistentvolumes", { params: { cluster_id: clusterId, keyword } }));
}

export function getPersistentVolumeDetail(clusterId: number, name: string) {
  return getData<StorageDetail>(
    http.get("/persistentvolumes/detail", { params: { kind: "PersistentVolume", cluster_id: clusterId, name } }),
  );
}

export function deletePersistentVolume(clusterId: number, name: string) {
  return getData<boolean>(
    http.delete("/persistentvolumes", { params: { kind: "PersistentVolume", cluster_id: clusterId, name } }),
  );
}

export function listPersistentVolumeClaims(clusterId: number, namespace: string, keyword?: string) {
  return getData<PersistentVolumeClaimItem[]>(
    http.get("/persistentvolumeclaims", { params: { cluster_id: clusterId, namespace, keyword } }),
  );
}

export function getPersistentVolumeClaimDetail(clusterId: number, namespace: string, name: string) {
  return getData<StorageDetail>(
    http.get("/persistentvolumeclaims/detail", {
      params: { kind: "PersistentVolumeClaim", cluster_id: clusterId, namespace, name },
    }),
  );
}

export function deletePersistentVolumeClaim(clusterId: number, namespace: string, name: string) {
  return getData<boolean>(
    http.delete("/persistentvolumeclaims", {
      params: { kind: "PersistentVolumeClaim", cluster_id: clusterId, namespace, name },
    }),
  );
}

export function listStorageClasses(clusterId: number, keyword?: string) {
  return getData<StorageClassItem[]>(http.get("/storageclasses", { params: { cluster_id: clusterId, keyword } }));
}

export function getStorageClassDetail(clusterId: number, name: string) {
  return getData<StorageDetail>(
    http.get("/storageclasses/detail", { params: { kind: "StorageClass", cluster_id: clusterId, name } }),
  );
}

export function deleteStorageClass(clusterId: number, name: string) {
  return getData<boolean>(
    http.delete("/storageclasses", { params: { kind: "StorageClass", cluster_id: clusterId, name } }),
  );
}

export function applyPersistentVolume(clusterId: number, manifest: string) {
  return getData<boolean>(http.post("/persistentvolumes/apply", { cluster_id: clusterId, manifest }));
}

export function applyPersistentVolumeClaim(clusterId: number, manifest: string) {
  return getData<boolean>(http.post("/persistentvolumeclaims/apply", { cluster_id: clusterId, manifest }));
}

export function applyStorageClass(clusterId: number, manifest: string) {
  return getData<boolean>(http.post("/storageclasses/apply", { cluster_id: clusterId, manifest }));
}

const TOKEN_KEY = "permission-system-token";
const USER_KEY = "permission-system-user";

export function getToken() {
  return window.localStorage.getItem(TOKEN_KEY) ?? "";
}

export function setToken(token: string) {
  window.localStorage.setItem(TOKEN_KEY, token);
}

export function clearToken() {
  window.localStorage.removeItem(TOKEN_KEY);
}

export function getUser() {
  const raw = window.localStorage.getItem(USER_KEY);
  if (!raw) return null;
  try {
    return JSON.parse(raw) as UserItem;
  } catch {
    return null;
  }
}

export function setUser(user: UserItem) {
  window.localStorage.setItem(USER_KEY, JSON.stringify(user));
}

export function clearUser() {
  window.localStorage.removeItem(USER_KEY);
}

export function clearAuthStorage() {
  clearToken();
  clearUser();
}