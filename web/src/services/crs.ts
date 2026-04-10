import { getData, http } from "./http";

export interface CrResourceItem {
  name: string;
  group: string;
  version: string;
  resource: string;
  kind: string;
  scope: string;
  namespaced: boolean;
}

export interface CrItem {
  name: string;
  namespace?: string;
  api_version: string;
  kind: string;
  creation_time: string;
}

export interface CrDetail {
  yaml: string;
}

export function listCrResources(clusterId: number) {
  return getData<CrResourceItem[]>(http.get("/crs/resources", { params: { cluster_id: clusterId } }));
}

export function listCrs(args: {
  clusterId: number;
  group: string;
  version: string;
  resource: string;
  namespace?: string;
  keyword?: string;
}) {
  return getData<CrItem[]>(
    http.get("/crs", {
      params: {
        cluster_id: args.clusterId,
        group: args.group,
        version: args.version,
        resource: args.resource,
        namespace: args.namespace,
        keyword: args.keyword,
      },
    }),
  );
}

export function getCrDetail(args: {
  clusterId: number;
  group: string;
  version: string;
  resource: string;
  namespace?: string;
  name: string;
}) {
  return getData<CrDetail>(
    http.get("/crs/detail", {
      params: {
        cluster_id: args.clusterId,
        group: args.group,
        version: args.version,
        resource: args.resource,
        namespace: args.namespace,
        name: args.name,
      },
    }),
  );
}

export function applyCr(clusterId: number, manifest: string) {
  return getData<boolean>(http.post("/crs/apply", { cluster_id: clusterId, manifest }));
}

export function deleteCr(args: {
  clusterId: number;
  group: string;
  version: string;
  resource: string;
  namespace?: string;
  name: string;
}) {
  return getData<boolean>(
    http.delete("/crs", {
      params: {
        cluster_id: args.clusterId,
        group: args.group,
        version: args.version,
        resource: args.resource,
        namespace: args.namespace,
        name: args.name,
      },
    }),
  );
}
