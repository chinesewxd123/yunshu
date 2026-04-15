import { getData, http } from "./http";

export function k8sParams(clusterId: number, params?: Record<string, any>) {
  return { cluster_id: clusterId, ...(params ?? {}) };
}

export function createK8sResourceService<Item, Detail>(basePath: string) {
  return {
    list: (params: Record<string, any>) => getData<Item[]>(http.get(basePath, { params })),
    detail: (params: Record<string, any>) => getData<Detail>(http.get(`${basePath}/detail`, { params })),
    apply: (body: Record<string, any>) => getData<boolean>(http.post(`${basePath}/apply`, body)),
    remove: (params: Record<string, any>) => getData<boolean>(http.delete(basePath, { params })),
    get: <T>(subPath: string, params: Record<string, any>) => getData<T>(http.get(`${basePath}${subPath}`, { params })),
    post: <T>(subPath: string, body: Record<string, any>) => getData<T>(http.post(`${basePath}${subPath}`, body)),
  };
}

