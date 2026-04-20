import { createK8sResourceService, k8sParams } from "./service-factory";

export type NetworkPolicyItem = {
  name: string;
  namespace: string;
  policy_types?: string;
  pod_selector_count: number;
  ingress_rule_count: number;
  egress_rule_count: number;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  age?: string;
  creation_time: string;
};

export type NetworkPolicyDetail = { yaml: string };

const svc = createK8sResourceService<NetworkPolicyItem, NetworkPolicyDetail>("/network-policies");

export function listNetworkPolicies(clusterId: number, namespace: string, keyword?: string) {
  return svc.list(k8sParams(clusterId, { namespace, keyword }));
}

export function getNetworkPolicyDetail(clusterId: number, namespace: string, name: string) {
  return svc.detail(k8sParams(clusterId, { namespace, name }));
}

export function applyNetworkPolicy(clusterId: number, manifest: string) {
  return svc.apply({ cluster_id: clusterId, manifest });
}

export function deleteNetworkPolicy(clusterId: number, namespace: string, name: string) {
  return svc.remove(k8sParams(clusterId, { namespace, name }));
}
