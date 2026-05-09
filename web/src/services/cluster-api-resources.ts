import { getData, http } from "./http";

export type APIResourceDiscoveryItem = {
  group_version: string;
  name: string;
  namespaced: boolean;
  kind: string;
  verbs: string[];
};

export function listClusterAPIResources(clusterId: number, namespaced?: boolean | null) {
  const params: Record<string, string> = {};
  if (namespaced === true) params.namespaced = "true";
  if (namespaced === false) params.namespaced = "false";
  return getData<{ list: APIResourceDiscoveryItem[]; cluster_id: string }>(
    http.get(`/clusters/${clusterId}/api-resources`, { params, timeout: 60000 }),
  );
}
