import { getData, http } from "./http";

export type K8sNamespaceAllowRule = {
  id: number;
  principal_kind: string;
  principal_ref: string;
  cluster_id: number;
  namespace: string;
  created_at: string;
};

export function listK8sNamespaceAllowRules(params?: {
  principal_kind?: string;
  principal_ref?: string;
  cluster_id?: number;
}) {
  return getData<{ list: K8sNamespaceAllowRule[] }>(http.get("/k8s-namespace-allow-rules", { params }));
}

export function createK8sNamespaceAllowRule(payload: {
  principal_kind: string;
  principal_ref: string;
  cluster_id: number;
  namespace: string;
}) {
  return getData<K8sNamespaceAllowRule>(http.post("/k8s-namespace-allow-rules", payload));
}

export function deleteK8sNamespaceAllowRule(id: number) {
  return getData<{ message: string }>(http.delete(`/k8s-namespace-allow-rules/${id}`));
}
