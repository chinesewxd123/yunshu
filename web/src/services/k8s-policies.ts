import { getData, http } from "./http";

export type K8sActionItem = {
  code: string;
  name: string;
  description: string;
};

/** 集群 K8s 访问档位（DB）；主体可为 role / user / group（对齐 k8m） */
export type K8sClusterAccessItem = {
  id: number;
  principal_kind: string;
  principal_ref: string;
  /** 兼容：kind=role 时与 principal_ref 相同 */
  role_code?: string;
  cluster_id: number;
  preset: string;
};

export type K8sScopedPolicyGrantPresetPayload = {
  principal_kind?: "role" | "user" | "group";
  role_id?: number;
  user_id?: number;
  group_id?: number;
  cluster_ids: number[];
  preset: "readonly" | "readonly_exec" | "admin";
  /** 可选；须配合明确集群 ID，写入命名空间黑名单 */
  deny_namespaces?: string[];
  /** 可选；须配合明确集群 ID，写入命名空间白名单 */
  allow_namespaces?: string[];
};

export type K8sScopedPolicyGrantPresetResponse = {
  added: number;
  skipped: number;
  deny_rules_added: number;
  deny_rules_skipped: number;
  allow_rules_added: number;
  allow_rules_skipped: number;
};

export function listK8sPolicyActions() {
  return getData<{ list: K8sActionItem[] }>(http.get("/k8s-policies/actions"));
}

export function listK8sPolicyPaths() {
  return getData<{ list: string[] }>(http.get("/k8s-policies/paths"));
}

export function listK8sPoliciesByRole(roleId: number) {
  return getData<{ list: K8sClusterAccessItem[] }>(
    http.get("/k8s-policies", { params: { role_id: String(roleId) } }),
  );
}

export function listK8sClusterGrants(params: { role_id?: number; user_id?: number; group_id?: number }) {
  const p: Record<string, string> = {};
  if (params.role_id != null && params.role_id > 0) p.role_id = String(params.role_id);
  if (params.user_id != null && params.user_id > 0) p.user_id = String(params.user_id);
  if (params.group_id != null && params.group_id > 0) p.group_id = String(params.group_id);
  return getData<{ list: K8sClusterAccessItem[] }>(http.get("/k8s-policies", { params: p }));
}

export function grantK8sScopedPoliciesPreset(payload: K8sScopedPolicyGrantPresetPayload) {
  return getData<K8sScopedPolicyGrantPresetResponse>(http.post("/k8s-policies/grant-preset", payload));
}

export function deleteK8sClusterGrant(id: number) {
  return getData<{ message: string }>(http.delete(`/k8s-policies/cluster-grants/${id}`));
}

/** 集群管理：某集群下已授权用户（角色/组按成员展开） */
export type K8sAuthMatrixRow = {
  row_key: string;
  grant_id: number;
  username: string;
  nickname?: string;
  principal_kind: string;
  principal_ref: string;
  principal_show: string;
  cluster_id: number;
  cluster_name: string;
  grant_scope_all: boolean;
  preset: string;
  preset_label: string;
  allow_namespaces: string;
  via: string;
};

/** 用户管理：某用户已授权集群汇总 */
export type K8sUserClusterAuthRow = {
  row_key: string;
  grant_id: number;
  username: string;
  cluster_id: number;
  cluster_name: string;
  grant_scope_all: boolean;
  preset: string;
  preset_label: string;
  allow_namespaces: string;
  via: string;
};

export function listClusterAuthMatrix(clusterId: number) {
  return getData<{ list: K8sAuthMatrixRow[] }>(
    http.get("/k8s-policies/cluster-auth-matrix", { params: { cluster_id: String(clusterId) } }),
  );
}

export function listUserClusterAuth(userId: number) {
  return getData<{ list: K8sUserClusterAuthRow[] }>(
    http.get("/k8s-policies/user-cluster-auth", { params: { user_id: String(userId) } }),
  );
}

export function batchDeleteK8sClusterGrants(ids: number[]) {
  return getData<{ deleted: number }>(http.post("/k8s-policies/cluster-grants/batch-delete", { ids }));
}
