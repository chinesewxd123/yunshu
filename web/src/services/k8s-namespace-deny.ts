import { getData, http } from "./http";

export type K8sNamespaceDenyRule = {
  id: number;
  principal_kind: string;
  principal_ref: string;
  cluster_id: number;
  namespace: string;
  created_at: string;
};

export function listK8sNamespaceDenyRules(params?: {
  principal_kind?: string;
  principal_ref?: string;
  /** 兼容旧参数 */
  role_code?: string;
  cluster_id?: number;
}) {
  return getData<{ list: K8sNamespaceDenyRule[] }>(http.get("/k8s-namespace-deny-rules", { params }));
}

export function createK8sNamespaceDenyRule(payload: {
  principal_kind: string;
  principal_ref: string;
  cluster_id: number;
  namespace: string;
}) {
  return getData<K8sNamespaceDenyRule>(http.post("/k8s-namespace-deny-rules", payload));
}

export function deleteK8sNamespaceDenyRule(id: number) {
  return getData<{ message: string }>(http.delete(`/k8s-namespace-deny-rules/${id}`));
}
