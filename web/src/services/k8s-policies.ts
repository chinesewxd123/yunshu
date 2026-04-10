import { getData, http } from "./http";

export type K8sActionItem = {
  code: string;
  name: string;
  description: string;
};

export type K8sScopedPolicyItem = {
  role_code: string;
  cluster_id: string;
  namespace: string;
  path: string;
  action: string;
  resource: string;
};

export type K8sScopedPolicyGrantPayload = {
  role_id: number;
  cluster_ids: number[];
  namespaces: string[];
  actions: string[];
  paths: string[];
};

export type K8sScopedPolicyGrantResponse = {
  added: number;
  skipped: number;
  policies: string[];
};

export function listK8sPolicyActions() {
  return getData<{ list: K8sActionItem[] }>(http.get("/k8s-policies/actions"));
}

export function listK8sPolicyPaths() {
  return getData<{ list: string[] }>(http.get("/k8s-policies/paths"));
}

export function listK8sPoliciesByRole(roleId: number) {
  // 避免 roleId 在运行态被传成 undefined 导致 query 被省略
  return getData<{ list: K8sScopedPolicyItem[] }>(
    http.get("/k8s-policies", { params: { role_id: String(roleId) } }),
  );
}

export function grantK8sScopedPolicies(payload: K8sScopedPolicyGrantPayload) {
  return getData<K8sScopedPolicyGrantResponse>(http.post("/k8s-policies/grant", payload));
}

