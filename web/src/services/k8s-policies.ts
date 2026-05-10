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
