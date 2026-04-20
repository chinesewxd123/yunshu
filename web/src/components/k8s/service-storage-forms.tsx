import { AutoComplete, Button, Form, Input, InputNumber, Select, Space } from "antd";
import YAML from "yaml";

export type KVPair = { key?: string; value?: string };
export type PortPair = { name?: string; protocol?: "TCP" | "UDP" | "SCTP"; port?: number; targetPort?: string; nodePort?: number };

function pairsToMap(pairs?: KVPair[]): Record<string, string> {
  const out: Record<string, string> = {};
  for (const p of pairs ?? []) {
    const k = String(p?.key ?? "").trim();
    if (!k) continue;
    out[k] = String(p?.value ?? "").trim();
  }
  return out;
}

export type ServiceFormValues = {
  name: string;
  namespace: string;
  type?: "ClusterIP" | "NodePort" | "LoadBalancer" | "ExternalName";
  externalName?: string;
  sessionAffinity?: "None" | "ClientIP";
  externalTrafficPolicy?: "Cluster" | "Local";
  internalTrafficPolicy?: "Cluster" | "Local";
  loadBalancerSourceRanges?: string[];
  healthCheckNodePort?: number;
  ipFamilyPolicy?: "SingleStack" | "PreferDualStack" | "RequireDualStack";
  ipFamilies?: Array<"IPv4" | "IPv6">;
  selector_pairs?: KVPair[];
  ports?: PortPair[];
};

function parseIntOrStringPort(v?: string): number | string | undefined {
  const s = String(v ?? "").trim();
  if (!s) return undefined;
  if (/^\d+$/.test(s)) {
    const n = Number(s);
    if (Number.isFinite(n) && n > 0) return n;
  }
  return s;
}

export function buildServiceYaml(v: ServiceFormValues): string {
  const selector = pairsToMap(v.selector_pairs);
  const ports =
    (v.ports ?? [])
      .map((p) => ({
        name: String(p.name ?? "").trim() || undefined,
        protocol: p.protocol || "TCP",
        port: Number(p.port ?? 0),
        targetPort: parseIntOrStringPort(p.targetPort),
        nodePort: p.nodePort && p.nodePort > 0 ? Number(p.nodePort) : undefined,
      }))
      .filter((p) => p.port > 0) || undefined;
  const type = v.type || "ClusterIP";
  const obj: any = {
    apiVersion: "v1",
    kind: "Service",
    metadata: { name: v.name, namespace: v.namespace },
    spec: {
      type,
      sessionAffinity: v.sessionAffinity || undefined,
      externalTrafficPolicy: v.externalTrafficPolicy || undefined,
      internalTrafficPolicy: v.internalTrafficPolicy || undefined,
      loadBalancerSourceRanges: (v.loadBalancerSourceRanges ?? []).filter(Boolean),
      healthCheckNodePort: v.healthCheckNodePort && v.healthCheckNodePort > 0 ? v.healthCheckNodePort : undefined,
      ipFamilyPolicy: v.ipFamilyPolicy || undefined,
      ipFamilies: (v.ipFamilies ?? []).length ? v.ipFamilies : undefined,
      selector: Object.keys(selector).length ? selector : undefined,
      ports: type === "ExternalName" ? undefined : ports,
      externalName: type === "ExternalName" ? String(v.externalName ?? "").trim() || undefined : undefined,
    },
  };
  return YAML.stringify(obj);
}

export function serviceYamlToForm(yaml: string): ServiceFormValues | null {
  try {
    const obj: any = YAML.parse(yaml);
    if (!obj || obj.kind !== "Service") return null;
    const selector = obj?.spec?.selector ?? {};
    const selector_pairs = Object.entries(selector).map(([key, value]) => ({ key, value: String(value ?? "") }));
    const ports = Array.isArray(obj?.spec?.ports)
      ? obj.spec.ports.map((p: any) => ({
          name: p?.name,
          protocol: (p?.protocol || "TCP") as "TCP" | "UDP" | "SCTP",
          port: typeof p?.port === "number" ? p.port : undefined,
          targetPort: p?.targetPort != null ? String(p.targetPort) : undefined,
          nodePort: typeof p?.nodePort === "number" ? p.nodePort : undefined,
        }))
      : [{ protocol: "TCP", port: 80, targetPort: "80" }];
    return {
      name: String(obj?.metadata?.name ?? ""),
      namespace: String(obj?.metadata?.namespace ?? "default"),
      type: (obj?.spec?.type || "ClusterIP") as ServiceFormValues["type"],
      externalName: obj?.spec?.externalName ? String(obj.spec.externalName) : undefined,
      sessionAffinity: obj?.spec?.sessionAffinity as ServiceFormValues["sessionAffinity"],
      externalTrafficPolicy: obj?.spec?.externalTrafficPolicy as ServiceFormValues["externalTrafficPolicy"],
      internalTrafficPolicy: obj?.spec?.internalTrafficPolicy as ServiceFormValues["internalTrafficPolicy"],
      loadBalancerSourceRanges: Array.isArray(obj?.spec?.loadBalancerSourceRanges) ? obj.spec.loadBalancerSourceRanges.map((x: any) => String(x ?? "")) : [],
      healthCheckNodePort: typeof obj?.spec?.healthCheckNodePort === "number" ? obj.spec.healthCheckNodePort : undefined,
      ipFamilyPolicy: obj?.spec?.ipFamilyPolicy as ServiceFormValues["ipFamilyPolicy"],
      ipFamilies: Array.isArray(obj?.spec?.ipFamilies) ? obj.spec.ipFamilies : undefined,
      selector_pairs: selector_pairs.length ? selector_pairs : [{ key: "app", value: "" }],
      ports,
    };
  } catch {
    return null;
  }
}

export type PersistentVolumeClaimFormValues = {
  name: string;
  namespace: string;
  storageClassName?: string;
  accessModes?: Array<"ReadWriteOnce" | "ReadOnlyMany" | "ReadWriteMany" | "ReadWriteOncePod">;
  requestStorage: string;
  limitStorage?: string;
  volumeMode?: "Filesystem" | "Block";
  volumeName?: string;
};

export function buildPVCYaml(v: PersistentVolumeClaimFormValues): string {
  const obj: any = {
    apiVersion: "v1",
    kind: "PersistentVolumeClaim",
    metadata: { name: v.name, namespace: v.namespace },
    spec: {
      storageClassName: String(v.storageClassName ?? "").trim() || undefined,
      accessModes: (v.accessModes ?? []).length ? v.accessModes : ["ReadWriteOnce"],
      resources: {
        requests: { storage: v.requestStorage },
        ...(String(v.limitStorage ?? "").trim() ? { limits: { storage: String(v.limitStorage).trim() } } : {}),
      },
      volumeMode: v.volumeMode || undefined,
      volumeName: String(v.volumeName ?? "").trim() || undefined,
    },
  };
  return YAML.stringify(obj);
}

export function pvcYamlToForm(yaml: string): PersistentVolumeClaimFormValues | null {
  try {
    const obj: any = YAML.parse(yaml);
    if (!obj || obj.kind !== "PersistentVolumeClaim") return null;
    return {
      name: String(obj?.metadata?.name ?? ""),
      namespace: String(obj?.metadata?.namespace ?? "default"),
      storageClassName: obj?.spec?.storageClassName ? String(obj.spec.storageClassName) : undefined,
      accessModes: Array.isArray(obj?.spec?.accessModes) ? obj.spec.accessModes : ["ReadWriteOnce"],
      requestStorage: String(obj?.spec?.resources?.requests?.storage ?? "1Gi"),
      limitStorage: obj?.spec?.resources?.limits?.storage ? String(obj.spec.resources.limits.storage) : undefined,
      volumeMode: obj?.spec?.volumeMode as PersistentVolumeClaimFormValues["volumeMode"],
      volumeName: obj?.spec?.volumeName ? String(obj.spec.volumeName) : undefined,
    };
  } catch {
    return null;
  }
}

export type StorageClassFormValues = {
  name: string;
  provisioner: string;
  reclaimPolicy?: "Delete" | "Retain";
  volumeBindingMode?: "Immediate" | "WaitForFirstConsumer";
  allowVolumeExpansion?: boolean;
  mountOptions?: string[];
  params?: KVPair[];
};

export function buildStorageClassYaml(v: StorageClassFormValues): string {
  const parameters = pairsToMap(v.params);
  const obj: any = {
    apiVersion: "storage.k8s.io/v1",
    kind: "StorageClass",
    metadata: { name: v.name },
    provisioner: v.provisioner,
    reclaimPolicy: v.reclaimPolicy || undefined,
    volumeBindingMode: v.volumeBindingMode || undefined,
    allowVolumeExpansion: typeof v.allowVolumeExpansion === "boolean" ? v.allowVolumeExpansion : undefined,
    mountOptions: (v.mountOptions ?? []).filter(Boolean),
    parameters: Object.keys(parameters).length ? parameters : undefined,
  };
  return YAML.stringify(obj);
}

export function storageClassYamlToForm(yaml: string): StorageClassFormValues | null {
  try {
    const obj: any = YAML.parse(yaml);
    if (!obj || obj.kind !== "StorageClass") return null;
    const params = obj?.parameters ?? {};
    return {
      name: String(obj?.metadata?.name ?? ""),
      provisioner: String(obj?.provisioner ?? ""),
      reclaimPolicy: obj?.reclaimPolicy as StorageClassFormValues["reclaimPolicy"],
      volumeBindingMode: obj?.volumeBindingMode as StorageClassFormValues["volumeBindingMode"],
      allowVolumeExpansion: typeof obj?.allowVolumeExpansion === "boolean" ? obj.allowVolumeExpansion : undefined,
      mountOptions: Array.isArray(obj?.mountOptions) ? obj.mountOptions.map((x: any) => String(x ?? "")) : [],
      params: Object.entries(params).map(([key, value]) => ({ key, value: String(value ?? "") })),
    };
  } catch {
    return null;
  }
}

export type PersistentVolumeFormValues = {
  name: string;
  storageClassName?: string;
  accessModes?: Array<"ReadWriteOnce" | "ReadOnlyMany" | "ReadWriteMany" | "ReadWriteOncePod">;
  capacityStorage: string;
  reclaimPolicy?: "Delete" | "Retain" | "Recycle";
  volumeSourceType?: "hostPath" | "nfs" | "local";
  hostPath?: string;
  nfsServer?: string;
  nfsPath?: string;
  localPath?: string;
};

export function buildPVYaml(v: PersistentVolumeFormValues): string {
  const sourceType = v.volumeSourceType || "hostPath";
  let source: any = { hostPath: { path: String(v.hostPath ?? "").trim() || "/tmp/" + v.name } };
  if (sourceType === "nfs") {
    source = {
      nfs: {
        server: String(v.nfsServer ?? "").trim() || "127.0.0.1",
        path: String(v.nfsPath ?? "").trim() || "/",
      },
    };
  } else if (sourceType === "local") {
    source = {
      local: {
        path: String(v.localPath ?? "").trim() || "/mnt/disks/" + v.name,
      },
    };
  }
  const obj: any = {
    apiVersion: "v1",
    kind: "PersistentVolume",
    metadata: { name: v.name },
    spec: {
      capacity: { storage: v.capacityStorage },
      accessModes: (v.accessModes ?? []).length ? v.accessModes : ["ReadWriteOnce"],
      persistentVolumeReclaimPolicy: v.reclaimPolicy || "Delete",
      storageClassName: String(v.storageClassName ?? "").trim() || undefined,
      ...source,
    },
  };
  return YAML.stringify(obj);
}

export function pvYamlToForm(yaml: string): PersistentVolumeFormValues | null {
  try {
    const obj: any = YAML.parse(yaml);
    if (!obj || obj.kind !== "PersistentVolume") return null;
    return {
      name: String(obj?.metadata?.name ?? ""),
      storageClassName: obj?.spec?.storageClassName ? String(obj.spec.storageClassName) : undefined,
      accessModes: Array.isArray(obj?.spec?.accessModes) ? obj.spec.accessModes : ["ReadWriteOnce"],
      capacityStorage: String(obj?.spec?.capacity?.storage ?? "1Gi"),
      reclaimPolicy: (obj?.spec?.persistentVolumeReclaimPolicy || "Delete") as PersistentVolumeFormValues["reclaimPolicy"],
      volumeSourceType: obj?.spec?.nfs ? "nfs" : obj?.spec?.local ? "local" : "hostPath",
      hostPath: obj?.spec?.hostPath?.path ? String(obj.spec.hostPath.path) : undefined,
      nfsServer: obj?.spec?.nfs?.server ? String(obj.spec.nfs.server) : undefined,
      nfsPath: obj?.spec?.nfs?.path ? String(obj.spec.nfs.path) : undefined,
      localPath: obj?.spec?.local?.path ? String(obj.spec.local.path) : undefined,
    };
  } catch {
    return null;
  }
}

export function LabelsFormList({ name, addLabel }: { name: string; addLabel: string }) {
  return (
    <Form.List name={name}>
      {(fields, { add, remove }) => (
        <Space direction="vertical" style={{ width: "100%" }}>
          {fields.map((f) => (
            <Space key={f.key} style={{ display: "flex" }} align="baseline">
              <Form.Item name={[f.name, "key"]} style={{ marginBottom: 0 }}>
                <Input placeholder="key" style={{ width: 180 }} />
              </Form.Item>
              <Form.Item name={[f.name, "value"]} style={{ marginBottom: 0 }}>
                <Input placeholder="value" style={{ width: 280 }} />
              </Form.Item>
              <Button onClick={() => remove(f.name)}>删除</Button>
            </Space>
          ))}
          <Button onClick={() => add({ key: "", value: "" })}>{addLabel}</Button>
        </Space>
      )}
    </Form.List>
  );
}

export function ServicePortsFormList(props?: { recommendedPortNames?: string[] }) {
  const portNameOptions = (props?.recommendedPortNames ?? [])
    .map((name) => String(name).trim())
    .filter(Boolean)
    .map((name) => ({ label: name, value: name }));
  return (
    <Form.List name="ports">
      {(fields, { add, remove }) => (
        <Space direction="vertical" style={{ width: "100%" }}>
          {fields.map((f) => (
            <Space key={f.key} style={{ display: "flex" }} align="baseline">
              <Form.Item name={[f.name, "name"]} style={{ marginBottom: 0 }}>
                <AutoComplete
                  options={portNameOptions}
                  placeholder="name"
                  style={{ width: 140 }}
                  filterOption={(input, option) => String(option?.value ?? "").toLowerCase().includes(input.toLowerCase())}
                />
              </Form.Item>
              <Form.Item name={[f.name, "protocol"]} initialValue="TCP" style={{ marginBottom: 0 }}>
                <Select
                  style={{ width: 100 }}
                  options={[
                    { label: "TCP", value: "TCP" },
                    { label: "UDP", value: "UDP" },
                    { label: "SCTP", value: "SCTP" },
                  ]}
                />
              </Form.Item>
              <Form.Item name={[f.name, "port"]} rules={[{ required: true, message: "port" }]} style={{ marginBottom: 0 }}>
                <InputNumber placeholder="port" min={1} max={65535} style={{ width: 100 }} />
              </Form.Item>
              <Form.Item name={[f.name, "targetPort"]} style={{ marginBottom: 0 }} extra="可填端口名">
                <AutoComplete
                  options={portNameOptions}
                  placeholder="targetPort，如 8080 或 http"
                  style={{ width: 180 }}
                  filterOption={(input, option) => String(option?.value ?? "").toLowerCase().includes(input.toLowerCase())}
                />
              </Form.Item>
              <Form.Item name={[f.name, "nodePort"]} style={{ marginBottom: 0 }}>
                <InputNumber placeholder="nodePort" min={1} max={65535} style={{ width: 120 }} />
              </Form.Item>
              <Button onClick={() => remove(f.name)}>删除</Button>
            </Space>
          ))}
          <Button onClick={() => add({ protocol: "TCP", port: 80, targetPort: "80" })}>新增端口</Button>
        </Space>
      )}
    </Form.List>
  );
}

