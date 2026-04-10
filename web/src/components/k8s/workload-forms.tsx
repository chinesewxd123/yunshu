import { Button, Form, Input, InputNumber, Modal, Select, Space } from "antd";
import type { FormInstance } from "antd";
import YAML from "yaml";

type EnvPair = { key?: string; value?: string };

function envPairsToMap(pairs?: EnvPair[]): Record<string, string> {
  const out: Record<string, string> = {};
  for (const p of pairs ?? []) {
    const k = String(p?.key ?? "").trim();
    if (!k) continue;
    out[k] = String(p?.value ?? "");
  }
  return out;
}

function mapToEnvPairs(m?: Record<string, string>): EnvPair[] {
  const out: EnvPair[] = [];
  for (const [k, v] of Object.entries(m ?? {})) out.push({ key: k, value: v });
  return out.length ? out : [{ key: "", value: "" }];
}

function safeParseYaml(yaml: string): any | null {
  try {
    return YAML.parse(yaml);
  } catch {
    return null;
  }
}

function safeGet(obj: any, path: string): any {
  const parts = path.split(".").filter(Boolean);
  let cur = obj;
  for (const p of parts) {
    if (cur == null) return undefined;
    if (p.endsWith("]")) {
      const m = p.match(/^(\w+)\[(\d+)\]$/);
      if (!m) return undefined;
      const key = m[1];
      const idx = Number(m[2]);
      cur = cur?.[key]?.[idx];
      continue;
    }
    cur = cur?.[p];
  }
  return cur;
}

function toNumberOrUndefined(v: any): number | undefined {
  if (typeof v === "number" && Number.isFinite(v)) return v;
  if (typeof v === "string") {
    const s = v.trim();
    if (!s) return undefined;
    const n = Number(s);
    return Number.isFinite(n) ? n : undefined;
  }
  return undefined;
}

export type DeploymentFormValues = {
  name: string;
  namespace: string;
  replicas: number;
  container_name: string;
  image: string;
  image_pull_policy?: "Always" | "IfNotPresent" | "Never";
  port?: number;
  command?: string;
  env_pairs?: EnvPair[];
  requests_cpu?: string;
  requests_memory?: string;
  limits_cpu?: string;
  limits_memory?: string;
  tolerations?: Array<{
    key?: string;
    operator?: "Equal" | "Exists";
    value?: string;
    effect?: "NoSchedule" | "PreferNoSchedule" | "NoExecute";
    toleration_seconds?: number;
  }>;
  volumes?: Array<{
    name?: string;
    type?: "emptyDir" | "configMap" | "secret" | "pvc";
    source_name?: string;
  }>;
  volume_mounts?: Array<{
    name?: string;
    mount_path?: string;
    read_only?: boolean;
    sub_path?: string;
  }>;
  image_pull_secrets?: string[];
  liveness_probe_type?: "httpGet" | "tcpSocket";
  liveness_http_path?: string;
  liveness_http_port?: number;
  liveness_http_scheme?: "HTTP" | "HTTPS";
  liveness_tcp_port?: number;
  liveness_initial_delay_seconds?: number;
  liveness_period_seconds?: number;
  liveness_timeout_seconds?: number;
  liveness_failure_threshold?: number;
  liveness_success_threshold?: number;
  readiness_probe_type?: "httpGet" | "tcpSocket";
  readiness_http_path?: string;
  readiness_http_port?: number;
  readiness_http_scheme?: "HTTP" | "HTTPS";
  readiness_tcp_port?: number;
  readiness_initial_delay_seconds?: number;
  readiness_period_seconds?: number;
  readiness_timeout_seconds?: number;
  readiness_failure_threshold?: number;
  readiness_success_threshold?: number;
};

export function buildDeploymentYaml(v: DeploymentFormValues): string {
  const envMap = envPairsToMap(v.env_pairs);
  const env = Object.keys(envMap).length
    ? Object.entries(envMap).map(([name, value]) => ({ name, value }))
    : undefined;

  const imagePullSecrets =
    (v.image_pull_secrets ?? []).filter(Boolean).map((name) => ({ name: String(name).trim() })).filter((s) => !!s.name);
  const tolerations =
    (v.tolerations ?? [])
      .map((t) => ({
        key: String(t.key ?? "").trim(),
        operator: t.operator || "Equal",
        value: String(t.value ?? "").trim(),
        effect: t.effect || undefined,
        tolerationSeconds: typeof t.toleration_seconds === "number" ? t.toleration_seconds : undefined,
      }))
      .filter((t) => t.key || t.operator === "Exists")
      .map((t) => ({
        key: t.key || undefined,
        operator: t.operator,
        value: t.operator === "Exists" ? undefined : t.value || undefined,
        effect: t.effect,
        tolerationSeconds: t.tolerationSeconds,
      })) || undefined;

  const volumes =
    (v.volumes ?? [])
      .map((it) => {
        const name = String(it.name ?? "").trim();
        if (!name) return null;
        const type = it.type || "emptyDir";
        if (type === "configMap") {
          return { name, configMap: { name: String(it.source_name ?? "").trim() || name } };
        }
        if (type === "secret") {
          return { name, secret: { secretName: String(it.source_name ?? "").trim() || name } };
        }
        if (type === "pvc") {
          return { name, persistentVolumeClaim: { claimName: String(it.source_name ?? "").trim() || name } };
        }
        return { name, emptyDir: {} };
      })
      .filter(Boolean);

  const volumeMounts =
    (v.volume_mounts ?? [])
      .map((m) => ({
        name: String(m.name ?? "").trim(),
        mountPath: String(m.mount_path ?? "").trim(),
        readOnly: !!m.read_only,
        subPath: String(m.sub_path ?? "").trim() || undefined,
      }))
      .filter((m) => m.name && m.mountPath) || undefined;

  const resources: any = {};
  if (v.requests_cpu || v.requests_memory) {
    resources.requests = {};
    if (v.requests_cpu) resources.requests.cpu = v.requests_cpu;
    if (v.requests_memory) resources.requests.memory = v.requests_memory;
  }
  if (v.limits_cpu || v.limits_memory) {
    resources.limits = {};
    if (v.limits_cpu) resources.limits.cpu = v.limits_cpu;
    if (v.limits_memory) resources.limits.memory = v.limits_memory;
  }

  const livenessProbe = (() => {
    if (!v.liveness_probe_type) return undefined;
    const initialDelaySeconds = toNumberOrUndefined(v.liveness_initial_delay_seconds);
    const periodSeconds = toNumberOrUndefined(v.liveness_period_seconds);
    const timeoutSeconds = toNumberOrUndefined(v.liveness_timeout_seconds);
    const failureThreshold = toNumberOrUndefined(v.liveness_failure_threshold);
    const successThreshold = toNumberOrUndefined(v.liveness_success_threshold);

    if (v.liveness_probe_type === "httpGet") {
      const port = toNumberOrUndefined(v.liveness_http_port);
      if (!port) return undefined;
      const path = v.liveness_http_path?.trim() || "/";
      const scheme = v.liveness_http_scheme ? v.liveness_http_scheme : undefined;
      const obj: any = {
        httpGet: {
          path,
          port,
          ...(scheme ? { scheme } : {}),
        },
        ...(initialDelaySeconds !== undefined ? { initialDelaySeconds } : {}),
        ...(periodSeconds !== undefined ? { periodSeconds } : {}),
        ...(timeoutSeconds !== undefined ? { timeoutSeconds } : {}),
        ...(failureThreshold !== undefined ? { failureThreshold } : {}),
        ...(successThreshold !== undefined ? { successThreshold } : {}),
      };
      return Object.keys(obj.httpGet).length ? obj : undefined;
    }

    if (v.liveness_probe_type === "tcpSocket") {
      const port = toNumberOrUndefined(v.liveness_tcp_port);
      if (!port) return undefined;
      return {
        tcpSocket: { port },
        ...(initialDelaySeconds !== undefined ? { initialDelaySeconds } : {}),
        ...(periodSeconds !== undefined ? { periodSeconds } : {}),
        ...(timeoutSeconds !== undefined ? { timeoutSeconds } : {}),
        ...(failureThreshold !== undefined ? { failureThreshold } : {}),
        ...(successThreshold !== undefined ? { successThreshold } : {}),
      };
    }
    return undefined;
  })();

  const readinessProbe = (() => {
    if (!v.readiness_probe_type) return undefined;
    const initialDelaySeconds = toNumberOrUndefined(v.readiness_initial_delay_seconds);
    const periodSeconds = toNumberOrUndefined(v.readiness_period_seconds);
    const timeoutSeconds = toNumberOrUndefined(v.readiness_timeout_seconds);
    const failureThreshold = toNumberOrUndefined(v.readiness_failure_threshold);
    const successThreshold = toNumberOrUndefined(v.readiness_success_threshold);

    if (v.readiness_probe_type === "httpGet") {
      const port = toNumberOrUndefined(v.readiness_http_port);
      if (!port) return undefined;
      const path = v.readiness_http_path?.trim() || "/";
      const scheme = v.readiness_http_scheme ? v.readiness_http_scheme : undefined;
      const obj: any = {
        httpGet: {
          path,
          port,
          ...(scheme ? { scheme } : {}),
        },
        ...(initialDelaySeconds !== undefined ? { initialDelaySeconds } : {}),
        ...(periodSeconds !== undefined ? { periodSeconds } : {}),
        ...(timeoutSeconds !== undefined ? { timeoutSeconds } : {}),
        ...(failureThreshold !== undefined ? { failureThreshold } : {}),
        ...(successThreshold !== undefined ? { successThreshold } : {}),
      };
      return Object.keys(obj.httpGet).length ? obj : undefined;
    }

    if (v.readiness_probe_type === "tcpSocket") {
      const port = toNumberOrUndefined(v.readiness_tcp_port);
      if (!port) return undefined;
      return {
        tcpSocket: { port },
        ...(initialDelaySeconds !== undefined ? { initialDelaySeconds } : {}),
        ...(periodSeconds !== undefined ? { periodSeconds } : {}),
        ...(timeoutSeconds !== undefined ? { timeoutSeconds } : {}),
        ...(failureThreshold !== undefined ? { failureThreshold } : {}),
        ...(successThreshold !== undefined ? { successThreshold } : {}),
      };
    }
    return undefined;
  })();

  const obj: any = {
    apiVersion: "apps/v1",
    kind: "Deployment",
    metadata: { name: v.name, namespace: v.namespace },
    spec: {
      replicas: v.replicas,
      selector: { matchLabels: { app: v.name } },
      template: {
        metadata: { labels: { app: v.name } },
        spec: {
          containers: [
            {
              name: v.container_name || v.name,
              image: v.image,
              imagePullPolicy: v.image_pull_policy || undefined,
              ports: v.port ? [{ containerPort: v.port }] : undefined,
              command: v.command?.trim() ? ["sh", "-c", v.command.trim()] : undefined,
              env,
              resources: Object.keys(resources).length ? resources : undefined,
              volumeMounts,
                  ...(livenessProbe ? { livenessProbe } : {}),
                  ...(readinessProbe ? { readinessProbe } : {}),
            },
          ],
              imagePullSecrets: imagePullSecrets.length ? imagePullSecrets : undefined,
          volumes: volumes.length ? volumes : undefined,
          tolerations: tolerations?.length ? tolerations : undefined,
        },
      },
    },
  };
  return YAML.stringify(obj);
}

export function deploymentYamlToForm(yaml: string): DeploymentFormValues | null {
  const obj: any = safeParseYaml(yaml);
  if (!obj || obj.kind !== "Deployment") return null;
  const c = obj?.spec?.template?.spec?.containers?.[0] ?? {};
  const envPairs = mapToEnvPairs(
    Array.isArray(c.env)
      ? Object.fromEntries(c.env.filter((e: any) => e?.name).map((e: any) => [String(e.name), String(e.value ?? "")]))
      : undefined,
  );
  const port = c?.ports?.[0]?.containerPort;
  let cmd = "";
  if (Array.isArray(c.command) && c.command.length >= 3 && c.command[0] === "sh" && c.command[1] === "-c") {
    cmd = String(c.command.slice(2).join(" "));
  }
  const resReq = c?.resources?.requests ?? {};
  const resLim = c?.resources?.limits ?? {};
  const tolerations =
    Array.isArray(obj?.spec?.template?.spec?.tolerations) && obj.spec.template.spec.tolerations.length
      ? obj.spec.template.spec.tolerations.map((t: any) => ({
          key: t?.key,
          operator: (t?.operator || "Equal") as "Equal" | "Exists",
          value: t?.value,
          effect: t?.effect as "NoSchedule" | "PreferNoSchedule" | "NoExecute" | undefined,
          toleration_seconds: typeof t?.tolerationSeconds === "number" ? t.tolerationSeconds : undefined,
        }))
      : [{ key: "", operator: "Equal", value: "", effect: undefined }];
  const volumes =
    Array.isArray(obj?.spec?.template?.spec?.volumes) && obj.spec.template.spec.volumes.length
      ? obj.spec.template.spec.volumes.map((v: any) => ({
          name: v?.name,
          type: v?.configMap ? "configMap" : v?.secret ? "secret" : v?.persistentVolumeClaim ? "pvc" : "emptyDir",
          source_name: v?.configMap?.name || v?.secret?.secretName || v?.persistentVolumeClaim?.claimName || "",
        }))
      : [{ name: "", type: "emptyDir", source_name: "" }];
  const volumeMounts =
    Array.isArray(c?.volumeMounts) && c.volumeMounts.length
      ? c.volumeMounts.map((m: any) => ({
          name: m?.name,
          mount_path: m?.mountPath,
          read_only: !!m?.readOnly,
          sub_path: m?.subPath,
        }))
      : [{ name: "", mount_path: "", read_only: false, sub_path: "" }];

  const imagePullSecrets =
    Array.isArray(obj?.spec?.template?.spec?.imagePullSecrets) && obj.spec.template.spec.imagePullSecrets.length
      ? obj.spec.template.spec.imagePullSecrets.map((s: any) => String(s?.name ?? "")).filter((x: string) => !!x)
      : [];

  const lp = c?.livenessProbe;
  const rp = c?.readinessProbe;

  const liveness_probe_type: DeploymentFormValues["liveness_probe_type"] = lp?.httpGet ? "httpGet" : lp?.tcpSocket ? "tcpSocket" : undefined;
  const readiness_probe_type: DeploymentFormValues["readiness_probe_type"] = rp?.httpGet ? "httpGet" : rp?.tcpSocket ? "tcpSocket" : undefined;

  return {
    name: String(obj?.metadata?.name ?? ""),
    namespace: String(obj?.metadata?.namespace ?? "default"),
    replicas: Number(obj?.spec?.replicas ?? 1),
    container_name: String(c?.name ?? ""),
    image: String(c?.image ?? ""),
    image_pull_policy: (c?.imagePullPolicy as "Always" | "IfNotPresent" | "Never" | undefined) ?? undefined,
    image_pull_secrets: imagePullSecrets,
    port: typeof port === "number" ? port : undefined,
    command: cmd || undefined,
    env_pairs: envPairs,
    requests_cpu: resReq?.cpu ? String(resReq.cpu) : undefined,
    requests_memory: resReq?.memory ? String(resReq.memory) : undefined,
    limits_cpu: resLim?.cpu ? String(resLim.cpu) : undefined,
    limits_memory: resLim?.memory ? String(resLim.memory) : undefined,
    tolerations,
    volumes,
    volume_mounts: volumeMounts,
    liveness_probe_type,
    liveness_http_path: liveness_probe_type === "httpGet" ? String(lp?.httpGet?.path ?? "") : undefined,
    liveness_http_port: liveness_probe_type === "httpGet" ? toNumberOrUndefined(lp?.httpGet?.port) : undefined,
    liveness_http_scheme: liveness_probe_type === "httpGet" ? (lp?.httpGet?.scheme as any) : undefined,
    liveness_tcp_port: liveness_probe_type === "tcpSocket" ? toNumberOrUndefined(lp?.tcpSocket?.port) : undefined,
    liveness_initial_delay_seconds: toNumberOrUndefined(lp?.initialDelaySeconds),
    liveness_period_seconds: toNumberOrUndefined(lp?.periodSeconds),
    liveness_timeout_seconds: toNumberOrUndefined(lp?.timeoutSeconds),
    liveness_failure_threshold: toNumberOrUndefined(lp?.failureThreshold),
    liveness_success_threshold: toNumberOrUndefined(lp?.successThreshold),
    readiness_probe_type,
    readiness_http_path: readiness_probe_type === "httpGet" ? String(rp?.httpGet?.path ?? "") : undefined,
    readiness_http_port: readiness_probe_type === "httpGet" ? toNumberOrUndefined(rp?.httpGet?.port) : undefined,
    readiness_http_scheme: readiness_probe_type === "httpGet" ? (rp?.httpGet?.scheme as any) : undefined,
    readiness_tcp_port: readiness_probe_type === "tcpSocket" ? toNumberOrUndefined(rp?.tcpSocket?.port) : undefined,
    readiness_initial_delay_seconds: toNumberOrUndefined(rp?.initialDelaySeconds),
    readiness_period_seconds: toNumberOrUndefined(rp?.periodSeconds),
    readiness_timeout_seconds: toNumberOrUndefined(rp?.timeoutSeconds),
    readiness_failure_threshold: toNumberOrUndefined(rp?.failureThreshold),
    readiness_success_threshold: toNumberOrUndefined(rp?.successThreshold),
  };
}

export function deploymentObjToForm(obj: any): DeploymentFormValues | null {
  if (!obj) return null;
  // Some JSON objects from API may not include kind/type meta; rely on structure presence instead.
  const container0 = safeGet(obj, "spec.template.spec.containers[0]");
  if (!container0) return null;
  const c = safeGet(obj, "spec.template.spec.containers[0]") ?? {};
  const envPairs = mapToEnvPairs(
    Array.isArray(c.env)
      ? Object.fromEntries(c.env.filter((e: any) => e?.name).map((e: any) => [String(e.name), String(e.value ?? "")]))
      : undefined,
  );
  const port = c?.ports?.[0]?.containerPort;
  let cmd = "";
  if (Array.isArray(c.command) && c.command.length >= 3 && c.command[0] === "sh" && c.command[1] === "-c") {
    cmd = String(c.command.slice(2).join(" "));
  }
  const resReq = c?.resources?.requests ?? {};
  const resLim = c?.resources?.limits ?? {};
  const tolerations =
    Array.isArray(safeGet(obj, "spec.template.spec.tolerations")) && safeGet(obj, "spec.template.spec.tolerations").length
      ? safeGet(obj, "spec.template.spec.tolerations").map((t: any) => ({
          key: t?.key,
          operator: (t?.operator || "Equal") as "Equal" | "Exists",
          value: t?.value,
          effect: t?.effect as any,
          toleration_seconds: typeof t?.tolerationSeconds === "number" ? t.tolerationSeconds : undefined,
        }))
      : [];
  const volumes =
    Array.isArray(safeGet(obj, "spec.template.spec.volumes")) && safeGet(obj, "spec.template.spec.volumes").length
      ? safeGet(obj, "spec.template.spec.volumes").map((v: any) => ({
          name: v?.name,
          type: v?.configMap ? "configMap" : v?.secret ? "secret" : v?.persistentVolumeClaim ? "pvc" : "emptyDir",
          source_name: v?.configMap?.name || v?.secret?.secretName || v?.persistentVolumeClaim?.claimName || "",
        }))
      : [];
  const volumeMounts =
    Array.isArray(c?.volumeMounts) && c.volumeMounts.length
      ? c.volumeMounts.map((m: any) => ({
          name: m?.name,
          mount_path: m?.mountPath,
          read_only: !!m?.readOnly,
          sub_path: m?.subPath,
        }))
      : [];

  const imagePullSecrets =
    Array.isArray(safeGet(obj, "spec.template.spec.imagePullSecrets")) && safeGet(obj, "spec.template.spec.imagePullSecrets")?.length
      ? (safeGet(obj, "spec.template.spec.imagePullSecrets") ?? []).map((s: any) => String(s?.name ?? "")).filter((x: string) => !!x)
      : [];

  const lp = c?.livenessProbe;
  const rp = c?.readinessProbe;

  const liveness_probe_type: DeploymentFormValues["liveness_probe_type"] = lp?.httpGet ? "httpGet" : lp?.tcpSocket ? "tcpSocket" : undefined;
  const readiness_probe_type: DeploymentFormValues["readiness_probe_type"] = rp?.httpGet ? "httpGet" : rp?.tcpSocket ? "tcpSocket" : undefined;

  return {
    name: String(safeGet(obj, "metadata.name") ?? ""),
    namespace: String(safeGet(obj, "metadata.namespace") ?? "default"),
    replicas: Number(safeGet(obj, "spec.replicas") ?? 1),
    container_name: String(c?.name ?? ""),
    image: String(c?.image ?? ""),
    image_pull_policy: c?.imagePullPolicy as any,
    image_pull_secrets: imagePullSecrets,
    port: typeof port === "number" ? port : undefined,
    command: cmd || undefined,
    env_pairs: envPairs,
    requests_cpu: resReq?.cpu ? String(resReq.cpu) : undefined,
    requests_memory: resReq?.memory ? String(resReq.memory) : undefined,
    limits_cpu: resLim?.cpu ? String(resLim.cpu) : undefined,
    limits_memory: resLim?.memory ? String(resLim.memory) : undefined,
    tolerations,
    volumes,
    volume_mounts: volumeMounts,
    liveness_probe_type,
    liveness_http_path: liveness_probe_type === "httpGet" ? String(lp?.httpGet?.path ?? "") : undefined,
    liveness_http_port: liveness_probe_type === "httpGet" ? toNumberOrUndefined(lp?.httpGet?.port) : undefined,
    liveness_http_scheme: liveness_probe_type === "httpGet" ? (lp?.httpGet?.scheme as any) : undefined,
    liveness_tcp_port: liveness_probe_type === "tcpSocket" ? toNumberOrUndefined(lp?.tcpSocket?.port) : undefined,
    liveness_initial_delay_seconds: toNumberOrUndefined(lp?.initialDelaySeconds),
    liveness_period_seconds: toNumberOrUndefined(lp?.periodSeconds),
    liveness_timeout_seconds: toNumberOrUndefined(lp?.timeoutSeconds),
    liveness_failure_threshold: toNumberOrUndefined(lp?.failureThreshold),
    liveness_success_threshold: toNumberOrUndefined(lp?.successThreshold),
    readiness_probe_type,
    readiness_http_path: readiness_probe_type === "httpGet" ? String(rp?.httpGet?.path ?? "") : undefined,
    readiness_http_port: readiness_probe_type === "httpGet" ? toNumberOrUndefined(rp?.httpGet?.port) : undefined,
    readiness_http_scheme: readiness_probe_type === "httpGet" ? (rp?.httpGet?.scheme as any) : undefined,
    readiness_tcp_port: readiness_probe_type === "tcpSocket" ? toNumberOrUndefined(rp?.tcpSocket?.port) : undefined,
    readiness_initial_delay_seconds: toNumberOrUndefined(rp?.initialDelaySeconds),
    readiness_period_seconds: toNumberOrUndefined(rp?.periodSeconds),
    readiness_timeout_seconds: toNumberOrUndefined(rp?.timeoutSeconds),
    readiness_failure_threshold: toNumberOrUndefined(rp?.failureThreshold),
    readiness_success_threshold: toNumberOrUndefined(rp?.successThreshold),
  };
}

export function qosFromResources(v: Pick<DeploymentFormValues, "requests_cpu" | "requests_memory" | "limits_cpu" | "limits_memory">): "BestEffort" | "Burstable" | "Guaranteed" {
  const rc = String(v.requests_cpu ?? "").trim();
  const rm = String(v.requests_memory ?? "").trim();
  const lc = String(v.limits_cpu ?? "").trim();
  const lm = String(v.limits_memory ?? "").trim();
  const hasReq = !!(rc || rm);
  const hasLim = !!(lc || lm);
  if (!hasReq && !hasLim) return "BestEffort";
  const guaranteed = rc && rm && lc && lm && rc === lc && rm === lm;
  return guaranteed ? "Guaranteed" : "Burstable";
}

export type StatefulSetFormValues = {
  name: string;
  namespace: string;
  service_name: string;
  replicas: number;
  container_name: string;
  image: string;
  image_pull_policy?: "Always" | "IfNotPresent" | "Never";
  port?: number;
  command?: string;
  env_pairs?: EnvPair[];
  requests_cpu?: string;
  requests_memory?: string;
  limits_cpu?: string;
  limits_memory?: string;
  tolerations?: DeploymentFormValues["tolerations"];
  volumes?: DeploymentFormValues["volumes"];
  volume_mounts?: DeploymentFormValues["volume_mounts"];
};

export function buildStatefulSetYaml(v: StatefulSetFormValues): string {
  const envMap = envPairsToMap(v.env_pairs);
  const env = Object.keys(envMap).length
    ? Object.entries(envMap).map(([name, value]) => ({ name, value }))
    : undefined;

  const tolerations =
    (v.tolerations ?? [])
      .map((t) => ({
        key: String(t?.key ?? "").trim(),
        operator: t?.operator || "Equal",
        value: String(t?.value ?? "").trim(),
        effect: t?.effect || undefined,
        tolerationSeconds: typeof t?.toleration_seconds === "number" ? t.toleration_seconds : undefined,
      }))
      .filter((t) => t.key || t.operator === "Exists")
      .map((t) => ({
        key: t.key || undefined,
        operator: t.operator,
        value: t.operator === "Exists" ? undefined : t.value || undefined,
        effect: t.effect,
        tolerationSeconds: t.tolerationSeconds,
      })) || undefined;

  const volumes =
    (v.volumes ?? [])
      .map((it) => {
        const name = String(it?.name ?? "").trim();
        if (!name) return null;
        const type = it?.type || "emptyDir";
        if (type === "configMap") {
          return { name, configMap: { name: String(it?.source_name ?? "").trim() || name } };
        }
        if (type === "secret") {
          return { name, secret: { secretName: String(it?.source_name ?? "").trim() || name } };
        }
        if (type === "pvc") {
          return { name, persistentVolumeClaim: { claimName: String(it?.source_name ?? "").trim() || name } };
        }
        return { name, emptyDir: {} };
      })
      .filter(Boolean);

  const volumeMounts =
    (v.volume_mounts ?? [])
      .map((m) => ({
        name: String(m?.name ?? "").trim(),
        mountPath: String(m?.mount_path ?? "").trim(),
        readOnly: !!m?.read_only,
        subPath: String(m?.sub_path ?? "").trim() || undefined,
      }))
      .filter((m) => m.name && m.mountPath) || undefined;

  const resources: any = {};
  if (v.requests_cpu || v.requests_memory) {
    resources.requests = {};
    if (v.requests_cpu) resources.requests.cpu = v.requests_cpu;
    if (v.requests_memory) resources.requests.memory = v.requests_memory;
  }
  if (v.limits_cpu || v.limits_memory) {
    resources.limits = {};
    if (v.limits_cpu) resources.limits.cpu = v.limits_cpu;
    if (v.limits_memory) resources.limits.memory = v.limits_memory;
  }

  const obj: any = {
    apiVersion: "apps/v1",
    kind: "StatefulSet",
    metadata: { name: v.name, namespace: v.namespace },
    spec: {
      serviceName: v.service_name || `${v.name}-headless`,
      replicas: v.replicas,
      selector: { matchLabels: { app: v.name } },
      template: {
        metadata: { labels: { app: v.name } },
        spec: {
          containers: [
            {
              name: v.container_name || v.name,
              image: v.image,
              imagePullPolicy: v.image_pull_policy || undefined,
              ports: v.port ? [{ containerPort: v.port }] : undefined,
              command: v.command?.trim() ? ["sh", "-c", v.command.trim()] : undefined,
              env,
              resources: Object.keys(resources).length ? resources : undefined,
              volumeMounts,
            },
          ],
          volumes: volumes.length ? volumes : undefined,
          tolerations: tolerations?.length ? tolerations : undefined,
        },
      },
    },
  };
  return YAML.stringify(obj);
}

export function statefulSetYamlToForm(yaml: string): StatefulSetFormValues | null {
  const obj: any = safeParseYaml(yaml);
  if (!obj || obj.kind !== "StatefulSet") return null;
  const c = obj?.spec?.template?.spec?.containers?.[0] ?? {};
  const envPairs = mapToEnvPairs(
    Array.isArray(c.env)
      ? Object.fromEntries(c.env.filter((e: any) => e?.name).map((e: any) => [String(e.name), String(e.value ?? "")]))
      : undefined,
  );
  const port = c?.ports?.[0]?.containerPort;
  let cmd = "";
  if (Array.isArray(c.command) && c.command.length >= 3 && c.command[0] === "sh" && c.command[1] === "-c") {
    cmd = String(c.command.slice(2).join(" "));
  }
  const resReq = c?.resources?.requests ?? {};
  const resLim = c?.resources?.limits ?? {};
  const tolerations =
    Array.isArray(obj?.spec?.template?.spec?.tolerations) && obj.spec.template.spec.tolerations.length
      ? obj.spec.template.spec.tolerations.map((t: any) => ({
          key: t?.key,
          operator: (t?.operator || "Equal") as "Equal" | "Exists",
          value: t?.value,
          effect: t?.effect as "NoSchedule" | "PreferNoSchedule" | "NoExecute" | undefined,
          toleration_seconds: typeof t?.tolerationSeconds === "number" ? t.tolerationSeconds : undefined,
        }))
      : [{ key: "", operator: "Equal", value: "", effect: undefined }];
  const volumes =
    Array.isArray(obj?.spec?.template?.spec?.volumes) && obj.spec.template.spec.volumes.length
      ? obj.spec.template.spec.volumes.map((v: any) => ({
          name: v?.name,
          type: v?.configMap ? "configMap" : v?.secret ? "secret" : v?.persistentVolumeClaim ? "pvc" : "emptyDir",
          source_name: v?.configMap?.name || v?.secret?.secretName || v?.persistentVolumeClaim?.claimName || "",
        }))
      : [{ name: "", type: "emptyDir", source_name: "" }];
  const volumeMounts =
    Array.isArray(c?.volumeMounts) && c.volumeMounts.length
      ? c.volumeMounts.map((m: any) => ({
          name: m?.name,
          mount_path: m?.mountPath,
          read_only: !!m?.readOnly,
          sub_path: m?.subPath,
        }))
      : [{ name: "", mount_path: "", read_only: false, sub_path: "" }];
  return {
    name: String(obj?.metadata?.name ?? ""),
    namespace: String(obj?.metadata?.namespace ?? "default"),
    service_name: String(obj?.spec?.serviceName ?? ""),
    replicas: Number(obj?.spec?.replicas ?? 1),
    container_name: String(c?.name ?? ""),
    image: String(c?.image ?? ""),
    image_pull_policy: (c?.imagePullPolicy as "Always" | "IfNotPresent" | "Never" | undefined) ?? undefined,
    port: typeof port === "number" ? port : undefined,
    command: cmd || undefined,
    env_pairs: envPairs,
    requests_cpu: resReq?.cpu ? String(resReq.cpu) : undefined,
    requests_memory: resReq?.memory ? String(resReq.memory) : undefined,
    limits_cpu: resLim?.cpu ? String(resLim.cpu) : undefined,
    limits_memory: resLim?.memory ? String(resLim.memory) : undefined,
    tolerations,
    volumes,
    volume_mounts: volumeMounts,
  };
}

export function statefulSetObjToForm(obj: any): StatefulSetFormValues | null {
  if (!obj) return null;
  const c = safeGet(obj, "spec.template.spec.containers[0]") ?? null;
  if (!c) return null;
  const envPairs = mapToEnvPairs(
    Array.isArray(c.env)
      ? Object.fromEntries(c.env.filter((e: any) => e?.name).map((e: any) => [String(e.name), String(e.value ?? "")]))
      : undefined,
  );
  const port = c?.ports?.[0]?.containerPort;
  let cmd = "";
  if (Array.isArray(c.command) && c.command.length >= 3 && c.command[0] === "sh" && c.command[1] === "-c") {
    cmd = String(c.command.slice(2).join(" "));
  }
  const resReq = c?.resources?.requests ?? {};
  const resLim = c?.resources?.limits ?? {};
  const tolerations =
    Array.isArray(safeGet(obj, "spec.template.spec.tolerations")) && safeGet(obj, "spec.template.spec.tolerations")?.length
      ? (safeGet(obj, "spec.template.spec.tolerations") ?? []).map((t: any) => ({
          key: t?.key,
          operator: (t?.operator || "Equal") as "Equal" | "Exists",
          value: t?.value,
          effect: t?.effect as any,
          toleration_seconds: typeof t?.tolerationSeconds === "number" ? t.tolerationSeconds : undefined,
        }))
      : [];
  const volumes =
    Array.isArray(safeGet(obj, "spec.template.spec.volumes")) && safeGet(obj, "spec.template.spec.volumes")?.length
      ? (safeGet(obj, "spec.template.spec.volumes") ?? []).map((v: any) => ({
          name: v?.name,
          type: v?.configMap ? "configMap" : v?.secret ? "secret" : v?.persistentVolumeClaim ? "pvc" : "emptyDir",
          source_name: v?.configMap?.name || v?.secret?.secretName || v?.persistentVolumeClaim?.claimName || "",
        }))
      : [];
  const volumeMounts =
    Array.isArray(c?.volumeMounts) && c.volumeMounts.length
      ? c.volumeMounts.map((m: any) => ({
          name: m?.name,
          mount_path: m?.mountPath,
          read_only: !!m?.readOnly,
          sub_path: m?.subPath,
        }))
      : [];
  return {
    name: String(safeGet(obj, "metadata.name") ?? ""),
    namespace: String(safeGet(obj, "metadata.namespace") ?? "default"),
    service_name: String(safeGet(obj, "spec.serviceName") ?? ""),
    replicas: Number(safeGet(obj, "spec.replicas") ?? 1),
    container_name: String(c?.name ?? ""),
    image: String(c?.image ?? ""),
    image_pull_policy: c?.imagePullPolicy as any,
    port: typeof port === "number" ? port : undefined,
    command: cmd || undefined,
    env_pairs: envPairs,
    requests_cpu: resReq?.cpu ? String(resReq.cpu) : undefined,
    requests_memory: resReq?.memory ? String(resReq.memory) : undefined,
    limits_cpu: resLim?.cpu ? String(resLim.cpu) : undefined,
    limits_memory: resLim?.memory ? String(resLim.memory) : undefined,
    tolerations,
    volumes,
    volume_mounts: volumeMounts,
  };
}

export type JobFormValues = {
  name: string;
  namespace: string;
  restart_policy: "Never" | "OnFailure";
  container_name: string;
  image: string;
  image_pull_policy?: "Always" | "IfNotPresent" | "Never";
  command?: string;
  env_pairs?: EnvPair[];
  requests_cpu?: string;
  requests_memory?: string;
  limits_cpu?: string;
  limits_memory?: string;
  tolerations?: DeploymentFormValues["tolerations"];
  volumes?: DeploymentFormValues["volumes"];
  volume_mounts?: DeploymentFormValues["volume_mounts"];
};

export type DaemonSetFormValues = {
  name: string;
  namespace: string;
  container_name: string;
  image: string;
  image_pull_policy?: "Always" | "IfNotPresent" | "Never";
  port?: number;
  command?: string;
  env_pairs?: EnvPair[];
  requests_cpu?: string;
  requests_memory?: string;
  limits_cpu?: string;
  limits_memory?: string;
  tolerations?: DeploymentFormValues["tolerations"];
  volumes?: DeploymentFormValues["volumes"];
  volume_mounts?: DeploymentFormValues["volume_mounts"];
  image_pull_secrets?: DeploymentFormValues["image_pull_secrets"];
  liveness_probe_type?: DeploymentFormValues["liveness_probe_type"];
  liveness_http_path?: DeploymentFormValues["liveness_http_path"];
  liveness_http_port?: DeploymentFormValues["liveness_http_port"];
  liveness_http_scheme?: DeploymentFormValues["liveness_http_scheme"];
  liveness_tcp_port?: DeploymentFormValues["liveness_tcp_port"];
  liveness_initial_delay_seconds?: DeploymentFormValues["liveness_initial_delay_seconds"];
  liveness_period_seconds?: DeploymentFormValues["liveness_period_seconds"];
  liveness_timeout_seconds?: DeploymentFormValues["liveness_timeout_seconds"];
  liveness_failure_threshold?: DeploymentFormValues["liveness_failure_threshold"];
  liveness_success_threshold?: DeploymentFormValues["liveness_success_threshold"];
  readiness_probe_type?: DeploymentFormValues["readiness_probe_type"];
  readiness_http_path?: DeploymentFormValues["readiness_http_path"];
  readiness_http_port?: DeploymentFormValues["readiness_http_port"];
  readiness_http_scheme?: DeploymentFormValues["readiness_http_scheme"];
  readiness_tcp_port?: DeploymentFormValues["readiness_tcp_port"];
  readiness_initial_delay_seconds?: DeploymentFormValues["readiness_initial_delay_seconds"];
  readiness_period_seconds?: DeploymentFormValues["readiness_period_seconds"];
  readiness_timeout_seconds?: DeploymentFormValues["readiness_timeout_seconds"];
  readiness_failure_threshold?: DeploymentFormValues["readiness_failure_threshold"];
  readiness_success_threshold?: DeploymentFormValues["readiness_success_threshold"];
};

export function buildDaemonSetYaml(v: DaemonSetFormValues): string {
  const envMap = envPairsToMap(v.env_pairs);
  const env = Object.keys(envMap).length ? Object.entries(envMap).map(([name, value]) => ({ name, value })) : undefined;
  const imagePullSecrets =
    (v.image_pull_secrets ?? []).filter(Boolean).map((name) => ({ name: String(name).trim() })).filter((s) => !!s.name);

  const tolerations =
    (v.tolerations ?? [])
      .map((t) => ({
        key: String(t?.key ?? "").trim(),
        operator: t?.operator || "Equal",
        value: String(t?.value ?? "").trim(),
        effect: t?.effect || undefined,
        tolerationSeconds: typeof t?.toleration_seconds === "number" ? t.toleration_seconds : undefined,
      }))
      .filter((t) => t.key || t.operator === "Exists")
      .map((t) => ({
        key: t.key || undefined,
        operator: t.operator,
        value: t.operator === "Exists" ? undefined : t.value || undefined,
        effect: t.effect,
        tolerationSeconds: t.tolerationSeconds,
      })) || undefined;

  const volumes =
    (v.volumes ?? [])
      .map((it) => {
        const name = String(it?.name ?? "").trim();
        if (!name) return null;
        const type = it?.type || "emptyDir";
        if (type === "configMap") return { name, configMap: { name: String(it?.source_name ?? "").trim() || name } };
        if (type === "secret") return { name, secret: { secretName: String(it?.source_name ?? "").trim() || name } };
        if (type === "pvc") return { name, persistentVolumeClaim: { claimName: String(it?.source_name ?? "").trim() || name } };
        return { name, emptyDir: {} };
      })
      .filter(Boolean);

  const volumeMounts =
    (v.volume_mounts ?? [])
      .map((m) => ({
        name: String(m?.name ?? "").trim(),
        mountPath: String(m?.mount_path ?? "").trim(),
        readOnly: !!m?.read_only,
        subPath: String(m?.sub_path ?? "").trim() || undefined,
      }))
      .filter((m) => m.name && m.mountPath) || undefined;

  const resources: any = {};
  if (v.requests_cpu || v.requests_memory) {
    resources.requests = {};
    if (v.requests_cpu) resources.requests.cpu = v.requests_cpu;
    if (v.requests_memory) resources.requests.memory = v.requests_memory;
  }
  if (v.limits_cpu || v.limits_memory) {
    resources.limits = {};
    if (v.limits_cpu) resources.limits.cpu = v.limits_cpu;
    if (v.limits_memory) resources.limits.memory = v.limits_memory;
  }

  const livenessProbe = (() => {
    if (!v.liveness_probe_type) return undefined;
    const initialDelaySeconds = toNumberOrUndefined(v.liveness_initial_delay_seconds);
    const periodSeconds = toNumberOrUndefined(v.liveness_period_seconds);
    const timeoutSeconds = toNumberOrUndefined(v.liveness_timeout_seconds);
    const failureThreshold = toNumberOrUndefined(v.liveness_failure_threshold);
    const successThreshold = toNumberOrUndefined(v.liveness_success_threshold);
    if (v.liveness_probe_type === "httpGet") {
      const port = toNumberOrUndefined(v.liveness_http_port);
      if (!port) return undefined;
      const path = v.liveness_http_path?.trim() || "/";
      const scheme = v.liveness_http_scheme ? v.liveness_http_scheme : undefined;
      return {
        httpGet: { path, port, ...(scheme ? { scheme } : {}) },
        ...(initialDelaySeconds !== undefined ? { initialDelaySeconds } : {}),
        ...(periodSeconds !== undefined ? { periodSeconds } : {}),
        ...(timeoutSeconds !== undefined ? { timeoutSeconds } : {}),
        ...(failureThreshold !== undefined ? { failureThreshold } : {}),
        ...(successThreshold !== undefined ? { successThreshold } : {}),
      };
    }
    if (v.liveness_probe_type === "tcpSocket") {
      const port = toNumberOrUndefined(v.liveness_tcp_port);
      if (!port) return undefined;
      return {
        tcpSocket: { port },
        ...(initialDelaySeconds !== undefined ? { initialDelaySeconds } : {}),
        ...(periodSeconds !== undefined ? { periodSeconds } : {}),
        ...(timeoutSeconds !== undefined ? { timeoutSeconds } : {}),
        ...(failureThreshold !== undefined ? { failureThreshold } : {}),
        ...(successThreshold !== undefined ? { successThreshold } : {}),
      };
    }
    return undefined;
  })();

  const readinessProbe = (() => {
    if (!v.readiness_probe_type) return undefined;
    const initialDelaySeconds = toNumberOrUndefined(v.readiness_initial_delay_seconds);
    const periodSeconds = toNumberOrUndefined(v.readiness_period_seconds);
    const timeoutSeconds = toNumberOrUndefined(v.readiness_timeout_seconds);
    const failureThreshold = toNumberOrUndefined(v.readiness_failure_threshold);
    const successThreshold = toNumberOrUndefined(v.readiness_success_threshold);
    if (v.readiness_probe_type === "httpGet") {
      const port = toNumberOrUndefined(v.readiness_http_port);
      if (!port) return undefined;
      const path = v.readiness_http_path?.trim() || "/";
      const scheme = v.readiness_http_scheme ? v.readiness_http_scheme : undefined;
      return {
        httpGet: { path, port, ...(scheme ? { scheme } : {}) },
        ...(initialDelaySeconds !== undefined ? { initialDelaySeconds } : {}),
        ...(periodSeconds !== undefined ? { periodSeconds } : {}),
        ...(timeoutSeconds !== undefined ? { timeoutSeconds } : {}),
        ...(failureThreshold !== undefined ? { failureThreshold } : {}),
        ...(successThreshold !== undefined ? { successThreshold } : {}),
      };
    }
    if (v.readiness_probe_type === "tcpSocket") {
      const port = toNumberOrUndefined(v.readiness_tcp_port);
      if (!port) return undefined;
      return {
        tcpSocket: { port },
        ...(initialDelaySeconds !== undefined ? { initialDelaySeconds } : {}),
        ...(periodSeconds !== undefined ? { periodSeconds } : {}),
        ...(timeoutSeconds !== undefined ? { timeoutSeconds } : {}),
        ...(failureThreshold !== undefined ? { failureThreshold } : {}),
        ...(successThreshold !== undefined ? { successThreshold } : {}),
      };
    }
    return undefined;
  })();

  const obj: any = {
    apiVersion: "apps/v1",
    kind: "DaemonSet",
    metadata: { name: v.name, namespace: v.namespace },
    spec: {
      selector: { matchLabels: { app: v.name } },
      template: {
        metadata: { labels: { app: v.name } },
        spec: {
          imagePullSecrets: imagePullSecrets.length ? imagePullSecrets : undefined,
          tolerations: tolerations?.length ? tolerations : undefined,
          volumes: volumes.length ? volumes : undefined,
          containers: [
            {
              name: v.container_name || v.name,
              image: v.image,
              imagePullPolicy: v.image_pull_policy || undefined,
              ports: v.port ? [{ containerPort: v.port }] : undefined,
              command: v.command?.trim() ? ["sh", "-c", v.command.trim()] : undefined,
              env,
              resources: Object.keys(resources).length ? resources : undefined,
              volumeMounts,
              ...(livenessProbe ? { livenessProbe } : {}),
              ...(readinessProbe ? { readinessProbe } : {}),
            },
          ],
        },
      },
    },
  };
  return YAML.stringify(obj);
}

export function daemonSetYamlToForm(yaml: string): DaemonSetFormValues | null {
  const obj: any = safeParseYaml(yaml);
  if (!obj || obj.kind !== "DaemonSet") return null;
  return daemonSetObjToForm(obj);
}

export function daemonSetObjToForm(obj: any): DaemonSetFormValues | null {
  if (!obj) return null;
  const c = safeGet(obj, "spec.template.spec.containers[0]") ?? null;
  if (!c) return null;
  const envPairs = mapToEnvPairs(
    Array.isArray(c.env)
      ? Object.fromEntries(c.env.filter((e: any) => e?.name).map((e: any) => [String(e.name), String(e.value ?? "")]))
      : undefined,
  );
  const port = c?.ports?.[0]?.containerPort;
  let cmd = "";
  if (Array.isArray(c.command) && c.command.length >= 3 && c.command[0] === "sh" && c.command[1] === "-c") {
    cmd = String(c.command.slice(2).join(" "));
  }
  const resReq = c?.resources?.requests ?? {};
  const resLim = c?.resources?.limits ?? {};
  const tolerations =
    Array.isArray(safeGet(obj, "spec.template.spec.tolerations")) && safeGet(obj, "spec.template.spec.tolerations")?.length
      ? (safeGet(obj, "spec.template.spec.tolerations") ?? []).map((t: any) => ({
          key: t?.key,
          operator: (t?.operator || "Equal") as "Equal" | "Exists",
          value: t?.value,
          effect: t?.effect as any,
          toleration_seconds: typeof t?.tolerationSeconds === "number" ? t.tolerationSeconds : undefined,
        }))
      : [];
  const volumes =
    Array.isArray(safeGet(obj, "spec.template.spec.volumes")) && safeGet(obj, "spec.template.spec.volumes")?.length
      ? (safeGet(obj, "spec.template.spec.volumes") ?? []).map((v: any) => ({
          name: v?.name,
          type: v?.configMap ? "configMap" : v?.secret ? "secret" : v?.persistentVolumeClaim ? "pvc" : "emptyDir",
          source_name: v?.configMap?.name || v?.secret?.secretName || v?.persistentVolumeClaim?.claimName || "",
        }))
      : [];
  const volumeMounts =
    Array.isArray(c?.volumeMounts) && c.volumeMounts.length
      ? c.volumeMounts.map((m: any) => ({
          name: m?.name,
          mount_path: m?.mountPath,
          read_only: !!m?.readOnly,
          sub_path: m?.subPath,
        }))
      : [];
  const imagePullSecrets =
    Array.isArray(safeGet(obj, "spec.template.spec.imagePullSecrets")) && safeGet(obj, "spec.template.spec.imagePullSecrets")?.length
      ? (safeGet(obj, "spec.template.spec.imagePullSecrets") ?? []).map((s: any) => String(s?.name ?? "")).filter((x: string) => !!x)
      : [];

  const lp = c?.livenessProbe;
  const rp = c?.readinessProbe;
  const liveness_probe_type: DaemonSetFormValues["liveness_probe_type"] = lp?.httpGet ? "httpGet" : lp?.tcpSocket ? "tcpSocket" : undefined;
  const readiness_probe_type: DaemonSetFormValues["readiness_probe_type"] = rp?.httpGet ? "httpGet" : rp?.tcpSocket ? "tcpSocket" : undefined;

  return {
    name: String(safeGet(obj, "metadata.name") ?? ""),
    namespace: String(safeGet(obj, "metadata.namespace") ?? "default"),
    container_name: String(c?.name ?? ""),
    image: String(c?.image ?? ""),
    image_pull_policy: c?.imagePullPolicy as any,
    image_pull_secrets: imagePullSecrets,
    port: typeof port === "number" ? port : undefined,
    command: cmd || undefined,
    env_pairs: envPairs,
    requests_cpu: resReq?.cpu ? String(resReq.cpu) : undefined,
    requests_memory: resReq?.memory ? String(resReq.memory) : undefined,
    limits_cpu: resLim?.cpu ? String(resLim.cpu) : undefined,
    limits_memory: resLim?.memory ? String(resLim.memory) : undefined,
    tolerations,
    volumes,
    volume_mounts: volumeMounts,
    liveness_probe_type,
    liveness_http_path: liveness_probe_type === "httpGet" ? String(lp?.httpGet?.path ?? "") : undefined,
    liveness_http_port: liveness_probe_type === "httpGet" ? toNumberOrUndefined(lp?.httpGet?.port) : undefined,
    liveness_http_scheme: liveness_probe_type === "httpGet" ? (lp?.httpGet?.scheme as any) : undefined,
    liveness_tcp_port: liveness_probe_type === "tcpSocket" ? toNumberOrUndefined(lp?.tcpSocket?.port) : undefined,
    liveness_initial_delay_seconds: toNumberOrUndefined(lp?.initialDelaySeconds),
    liveness_period_seconds: toNumberOrUndefined(lp?.periodSeconds),
    liveness_timeout_seconds: toNumberOrUndefined(lp?.timeoutSeconds),
    liveness_failure_threshold: toNumberOrUndefined(lp?.failureThreshold),
    liveness_success_threshold: toNumberOrUndefined(lp?.successThreshold),
    readiness_probe_type,
    readiness_http_path: readiness_probe_type === "httpGet" ? String(rp?.httpGet?.path ?? "") : undefined,
    readiness_http_port: readiness_probe_type === "httpGet" ? toNumberOrUndefined(rp?.httpGet?.port) : undefined,
    readiness_http_scheme: readiness_probe_type === "httpGet" ? (rp?.httpGet?.scheme as any) : undefined,
    readiness_tcp_port: readiness_probe_type === "tcpSocket" ? toNumberOrUndefined(rp?.tcpSocket?.port) : undefined,
    readiness_initial_delay_seconds: toNumberOrUndefined(rp?.initialDelaySeconds),
    readiness_period_seconds: toNumberOrUndefined(rp?.periodSeconds),
    readiness_timeout_seconds: toNumberOrUndefined(rp?.timeoutSeconds),
    readiness_failure_threshold: toNumberOrUndefined(rp?.failureThreshold),
    readiness_success_threshold: toNumberOrUndefined(rp?.successThreshold),
  };
}

export type CronJobFormValues = {
  name: string;
  namespace: string;
  schedule: string;
  suspend?: boolean;
  restart_policy: "Never" | "OnFailure";
  container_name: string;
  image: string;
  image_pull_policy?: "Always" | "IfNotPresent" | "Never";
  command?: string;
  env_pairs?: EnvPair[];
  requests_cpu?: string;
  requests_memory?: string;
  limits_cpu?: string;
  limits_memory?: string;
  tolerations?: DeploymentFormValues["tolerations"];
  volumes?: DeploymentFormValues["volumes"];
  volume_mounts?: DeploymentFormValues["volume_mounts"];
  image_pull_secrets?: DeploymentFormValues["image_pull_secrets"];
  liveness_probe_type?: DeploymentFormValues["liveness_probe_type"];
  liveness_http_path?: DeploymentFormValues["liveness_http_path"];
  liveness_http_port?: DeploymentFormValues["liveness_http_port"];
  liveness_http_scheme?: DeploymentFormValues["liveness_http_scheme"];
  liveness_tcp_port?: DeploymentFormValues["liveness_tcp_port"];
  liveness_initial_delay_seconds?: DeploymentFormValues["liveness_initial_delay_seconds"];
  liveness_period_seconds?: DeploymentFormValues["liveness_period_seconds"];
  liveness_timeout_seconds?: DeploymentFormValues["liveness_timeout_seconds"];
  liveness_failure_threshold?: DeploymentFormValues["liveness_failure_threshold"];
  liveness_success_threshold?: DeploymentFormValues["liveness_success_threshold"];
  readiness_probe_type?: DeploymentFormValues["readiness_probe_type"];
  readiness_http_path?: DeploymentFormValues["readiness_http_path"];
  readiness_http_port?: DeploymentFormValues["readiness_http_port"];
  readiness_http_scheme?: DeploymentFormValues["readiness_http_scheme"];
  readiness_tcp_port?: DeploymentFormValues["readiness_tcp_port"];
  readiness_initial_delay_seconds?: DeploymentFormValues["readiness_initial_delay_seconds"];
  readiness_period_seconds?: DeploymentFormValues["readiness_period_seconds"];
  readiness_timeout_seconds?: DeploymentFormValues["readiness_timeout_seconds"];
  readiness_failure_threshold?: DeploymentFormValues["readiness_failure_threshold"];
  readiness_success_threshold?: DeploymentFormValues["readiness_success_threshold"];
};

export function buildCronJobYaml(v: CronJobFormValues): string {
  const envMap = envPairsToMap(v.env_pairs);
  const env = Object.keys(envMap).length ? Object.entries(envMap).map(([name, value]) => ({ name, value })) : undefined;
  const imagePullSecrets =
    (v.image_pull_secrets ?? []).filter(Boolean).map((name) => ({ name: String(name).trim() })).filter((s) => !!s.name);

  const tolerations =
    (v.tolerations ?? [])
      .map((t) => ({
        key: String(t?.key ?? "").trim(),
        operator: t?.operator || "Equal",
        value: String(t?.value ?? "").trim(),
        effect: t?.effect || undefined,
        tolerationSeconds: typeof t?.toleration_seconds === "number" ? t.toleration_seconds : undefined,
      }))
      .filter((t) => t.key || t.operator === "Exists")
      .map((t) => ({
        key: t.key || undefined,
        operator: t.operator,
        value: t.operator === "Exists" ? undefined : t.value || undefined,
        effect: t.effect,
        tolerationSeconds: t.tolerationSeconds,
      })) || undefined;

  const volumes =
    (v.volumes ?? [])
      .map((it) => {
        const name = String(it?.name ?? "").trim();
        if (!name) return null;
        const type = it?.type || "emptyDir";
        if (type === "configMap") return { name, configMap: { name: String(it?.source_name ?? "").trim() || name } };
        if (type === "secret") return { name, secret: { secretName: String(it?.source_name ?? "").trim() || name } };
        if (type === "pvc") return { name, persistentVolumeClaim: { claimName: String(it?.source_name ?? "").trim() || name } };
        return { name, emptyDir: {} };
      })
      .filter(Boolean);

  const volumeMounts =
    (v.volume_mounts ?? [])
      .map((m) => ({
        name: String(m?.name ?? "").trim(),
        mountPath: String(m?.mount_path ?? "").trim(),
        readOnly: !!m?.read_only,
        subPath: String(m?.sub_path ?? "").trim() || undefined,
      }))
      .filter((m) => m.name && m.mountPath) || undefined;

  const resources: any = {};
  if (v.requests_cpu || v.requests_memory) {
    resources.requests = {};
    if (v.requests_cpu) resources.requests.cpu = v.requests_cpu;
    if (v.requests_memory) resources.requests.memory = v.requests_memory;
  }
  if (v.limits_cpu || v.limits_memory) {
    resources.limits = {};
    if (v.limits_cpu) resources.limits.cpu = v.limits_cpu;
    if (v.limits_memory) resources.limits.memory = v.limits_memory;
  }

  const livenessProbe = (() => {
    if (!v.liveness_probe_type) return undefined;
    const initialDelaySeconds = toNumberOrUndefined(v.liveness_initial_delay_seconds);
    const periodSeconds = toNumberOrUndefined(v.liveness_period_seconds);
    const timeoutSeconds = toNumberOrUndefined(v.liveness_timeout_seconds);
    const failureThreshold = toNumberOrUndefined(v.liveness_failure_threshold);
    const successThreshold = toNumberOrUndefined(v.liveness_success_threshold);
    if (v.liveness_probe_type === "httpGet") {
      const port = toNumberOrUndefined(v.liveness_http_port);
      if (!port) return undefined;
      const path = v.liveness_http_path?.trim() || "/";
      const scheme = v.liveness_http_scheme ? v.liveness_http_scheme : undefined;
      return {
        httpGet: { path, port, ...(scheme ? { scheme } : {}) },
        ...(initialDelaySeconds !== undefined ? { initialDelaySeconds } : {}),
        ...(periodSeconds !== undefined ? { periodSeconds } : {}),
        ...(timeoutSeconds !== undefined ? { timeoutSeconds } : {}),
        ...(failureThreshold !== undefined ? { failureThreshold } : {}),
        ...(successThreshold !== undefined ? { successThreshold } : {}),
      };
    }
    if (v.liveness_probe_type === "tcpSocket") {
      const port = toNumberOrUndefined(v.liveness_tcp_port);
      if (!port) return undefined;
      return {
        tcpSocket: { port },
        ...(initialDelaySeconds !== undefined ? { initialDelaySeconds } : {}),
        ...(periodSeconds !== undefined ? { periodSeconds } : {}),
        ...(timeoutSeconds !== undefined ? { timeoutSeconds } : {}),
        ...(failureThreshold !== undefined ? { failureThreshold } : {}),
        ...(successThreshold !== undefined ? { successThreshold } : {}),
      };
    }
    return undefined;
  })();

  const readinessProbe = (() => {
    if (!v.readiness_probe_type) return undefined;
    const initialDelaySeconds = toNumberOrUndefined(v.readiness_initial_delay_seconds);
    const periodSeconds = toNumberOrUndefined(v.readiness_period_seconds);
    const timeoutSeconds = toNumberOrUndefined(v.readiness_timeout_seconds);
    const failureThreshold = toNumberOrUndefined(v.readiness_failure_threshold);
    const successThreshold = toNumberOrUndefined(v.readiness_success_threshold);
    if (v.readiness_probe_type === "httpGet") {
      const port = toNumberOrUndefined(v.readiness_http_port);
      if (!port) return undefined;
      const path = v.readiness_http_path?.trim() || "/";
      const scheme = v.readiness_http_scheme ? v.readiness_http_scheme : undefined;
      return {
        httpGet: { path, port, ...(scheme ? { scheme } : {}) },
        ...(initialDelaySeconds !== undefined ? { initialDelaySeconds } : {}),
        ...(periodSeconds !== undefined ? { periodSeconds } : {}),
        ...(timeoutSeconds !== undefined ? { timeoutSeconds } : {}),
        ...(failureThreshold !== undefined ? { failureThreshold } : {}),
        ...(successThreshold !== undefined ? { successThreshold } : {}),
      };
    }
    if (v.readiness_probe_type === "tcpSocket") {
      const port = toNumberOrUndefined(v.readiness_tcp_port);
      if (!port) return undefined;
      return {
        tcpSocket: { port },
        ...(initialDelaySeconds !== undefined ? { initialDelaySeconds } : {}),
        ...(periodSeconds !== undefined ? { periodSeconds } : {}),
        ...(timeoutSeconds !== undefined ? { timeoutSeconds } : {}),
        ...(failureThreshold !== undefined ? { failureThreshold } : {}),
        ...(successThreshold !== undefined ? { successThreshold } : {}),
      };
    }
    return undefined;
  })();

  const obj: any = {
    apiVersion: "batch/v1",
    kind: "CronJob",
    metadata: { name: v.name, namespace: v.namespace },
    spec: {
      schedule: v.schedule,
      suspend: typeof v.suspend === "boolean" ? v.suspend : undefined,
      jobTemplate: {
        spec: {
          template: {
            spec: {
              restartPolicy: v.restart_policy,
              imagePullSecrets: imagePullSecrets.length ? imagePullSecrets : undefined,
              tolerations: tolerations?.length ? tolerations : undefined,
              volumes: volumes.length ? volumes : undefined,
              containers: [
                {
                  name: v.container_name || v.name,
                  image: v.image,
                  imagePullPolicy: v.image_pull_policy || undefined,
                  command: v.command?.trim() ? ["sh", "-c", v.command.trim()] : undefined,
                  env,
                  resources: Object.keys(resources).length ? resources : undefined,
                  volumeMounts,
                  ...(livenessProbe ? { livenessProbe } : {}),
                  ...(readinessProbe ? { readinessProbe } : {}),
                },
              ],
            },
          },
        },
      },
    },
  };
  return YAML.stringify(obj);
}

export function cronJobYamlToForm(yaml: string): CronJobFormValues | null {
  const obj: any = safeParseYaml(yaml);
  if (!obj || obj.kind !== "CronJob") return null;
  return cronJobObjToForm(obj);
}

export function cronJobObjToForm(obj: any): CronJobFormValues | null {
  if (!obj) return null;
  const c = safeGet(obj, "spec.jobTemplate.spec.template.spec.containers[0]") ?? null;
  if (!c) return null;
  const envPairs = mapToEnvPairs(
    Array.isArray(c.env)
      ? Object.fromEntries(c.env.filter((e: any) => e?.name).map((e: any) => [String(e.name), String(e.value ?? "")]))
      : undefined,
  );
  let cmd = "";
  if (Array.isArray(c.command) && c.command.length >= 3 && c.command[0] === "sh" && c.command[1] === "-c") {
    cmd = String(c.command.slice(2).join(" "));
  }
  const rp = String(safeGet(obj, "spec.jobTemplate.spec.template.spec.restartPolicy") ?? "Never");
  const restart_policy: CronJobFormValues["restart_policy"] = rp === "OnFailure" ? "OnFailure" : "Never";
  const resReq = c?.resources?.requests ?? {};
  const resLim = c?.resources?.limits ?? {};

  const tolerations =
    Array.isArray(safeGet(obj, "spec.jobTemplate.spec.template.spec.tolerations")) && safeGet(obj, "spec.jobTemplate.spec.template.spec.tolerations")?.length
      ? (safeGet(obj, "spec.jobTemplate.spec.template.spec.tolerations") ?? []).map((t: any) => ({
          key: t?.key,
          operator: (t?.operator || "Equal") as "Equal" | "Exists",
          value: t?.value,
          effect: t?.effect as any,
          toleration_seconds: typeof t?.tolerationSeconds === "number" ? t.tolerationSeconds : undefined,
        }))
      : [];
  const volumes =
    Array.isArray(safeGet(obj, "spec.jobTemplate.spec.template.spec.volumes")) && safeGet(obj, "spec.jobTemplate.spec.template.spec.volumes")?.length
      ? (safeGet(obj, "spec.jobTemplate.spec.template.spec.volumes") ?? []).map((v: any) => ({
          name: v?.name,
          type: v?.configMap ? "configMap" : v?.secret ? "secret" : v?.persistentVolumeClaim ? "pvc" : "emptyDir",
          source_name: v?.configMap?.name || v?.secret?.secretName || v?.persistentVolumeClaim?.claimName || "",
        }))
      : [];
  const volumeMounts =
    Array.isArray(c?.volumeMounts) && c.volumeMounts.length
      ? c.volumeMounts.map((m: any) => ({
          name: m?.name,
          mount_path: m?.mountPath,
          read_only: !!m?.readOnly,
          sub_path: m?.subPath,
        }))
      : [];
  const imagePullSecrets =
    Array.isArray(safeGet(obj, "spec.jobTemplate.spec.template.spec.imagePullSecrets")) && safeGet(obj, "spec.jobTemplate.spec.template.spec.imagePullSecrets")?.length
      ? (safeGet(obj, "spec.jobTemplate.spec.template.spec.imagePullSecrets") ?? []).map((s: any) => String(s?.name ?? "")).filter((x: string) => !!x)
      : [];

  const lp = c?.livenessProbe;
  const rp2 = c?.readinessProbe;
  const liveness_probe_type: CronJobFormValues["liveness_probe_type"] = lp?.httpGet ? "httpGet" : lp?.tcpSocket ? "tcpSocket" : undefined;
  const readiness_probe_type: CronJobFormValues["readiness_probe_type"] = rp2?.httpGet ? "httpGet" : rp2?.tcpSocket ? "tcpSocket" : undefined;

  return {
    name: String(safeGet(obj, "metadata.name") ?? ""),
    namespace: String(safeGet(obj, "metadata.namespace") ?? "default"),
    schedule: String(safeGet(obj, "spec.schedule") ?? ""),
    suspend: typeof safeGet(obj, "spec.suspend") === "boolean" ? (safeGet(obj, "spec.suspend") as boolean) : undefined,
    restart_policy,
    container_name: String(c?.name ?? ""),
    image: String(c?.image ?? ""),
    image_pull_policy: c?.imagePullPolicy as any,
    image_pull_secrets: imagePullSecrets,
    command: cmd || undefined,
    env_pairs: envPairs,
    requests_cpu: resReq?.cpu ? String(resReq.cpu) : undefined,
    requests_memory: resReq?.memory ? String(resReq.memory) : undefined,
    limits_cpu: resLim?.cpu ? String(resLim.cpu) : undefined,
    limits_memory: resLim?.memory ? String(resLim.memory) : undefined,
    tolerations,
    volumes,
    volume_mounts: volumeMounts,
    liveness_probe_type,
    liveness_http_path: liveness_probe_type === "httpGet" ? String(lp?.httpGet?.path ?? "") : undefined,
    liveness_http_port: liveness_probe_type === "httpGet" ? toNumberOrUndefined(lp?.httpGet?.port) : undefined,
    liveness_http_scheme: liveness_probe_type === "httpGet" ? (lp?.httpGet?.scheme as any) : undefined,
    liveness_tcp_port: liveness_probe_type === "tcpSocket" ? toNumberOrUndefined(lp?.tcpSocket?.port) : undefined,
    liveness_initial_delay_seconds: toNumberOrUndefined(lp?.initialDelaySeconds),
    liveness_period_seconds: toNumberOrUndefined(lp?.periodSeconds),
    liveness_timeout_seconds: toNumberOrUndefined(lp?.timeoutSeconds),
    liveness_failure_threshold: toNumberOrUndefined(lp?.failureThreshold),
    liveness_success_threshold: toNumberOrUndefined(lp?.successThreshold),
    readiness_probe_type,
    readiness_http_path: readiness_probe_type === "httpGet" ? String(rp2?.httpGet?.path ?? "") : undefined,
    readiness_http_port: readiness_probe_type === "httpGet" ? toNumberOrUndefined(rp2?.httpGet?.port) : undefined,
    readiness_http_scheme: readiness_probe_type === "httpGet" ? (rp2?.httpGet?.scheme as any) : undefined,
    readiness_tcp_port: readiness_probe_type === "tcpSocket" ? toNumberOrUndefined(rp2?.tcpSocket?.port) : undefined,
    readiness_initial_delay_seconds: toNumberOrUndefined(rp2?.initialDelaySeconds),
    readiness_period_seconds: toNumberOrUndefined(rp2?.periodSeconds),
    readiness_timeout_seconds: toNumberOrUndefined(rp2?.timeoutSeconds),
    readiness_failure_threshold: toNumberOrUndefined(rp2?.failureThreshold),
    readiness_success_threshold: toNumberOrUndefined(rp2?.successThreshold),
  };
}

export function buildJobYaml(v: JobFormValues): string {
  const envMap = envPairsToMap(v.env_pairs);
  const env = Object.keys(envMap).length
    ? Object.entries(envMap).map(([name, value]) => ({ name, value }))
    : undefined;
  const tolerations =
    (v.tolerations ?? [])
      .map((t) => ({
        key: String(t?.key ?? "").trim(),
        operator: t?.operator || "Equal",
        value: String(t?.value ?? "").trim(),
        effect: t?.effect || undefined,
        tolerationSeconds: typeof t?.toleration_seconds === "number" ? t.toleration_seconds : undefined,
      }))
      .filter((t) => t.key || t.operator === "Exists")
      .map((t) => ({
        key: t.key || undefined,
        operator: t.operator,
        value: t.operator === "Exists" ? undefined : t.value || undefined,
        effect: t.effect,
        tolerationSeconds: t.tolerationSeconds,
      })) || undefined;

  const volumes =
    (v.volumes ?? [])
      .map((it) => {
        const name = String(it?.name ?? "").trim();
        if (!name) return null;
        const type = it?.type || "emptyDir";
        if (type === "configMap") {
          return { name, configMap: { name: String(it?.source_name ?? "").trim() || name } };
        }
        if (type === "secret") {
          return { name, secret: { secretName: String(it?.source_name ?? "").trim() || name } };
        }
        if (type === "pvc") {
          return { name, persistentVolumeClaim: { claimName: String(it?.source_name ?? "").trim() || name } };
        }
        return { name, emptyDir: {} };
      })
      .filter(Boolean);

  const volumeMounts =
    (v.volume_mounts ?? [])
      .map((m) => ({
        name: String(m?.name ?? "").trim(),
        mountPath: String(m?.mount_path ?? "").trim(),
        readOnly: !!m?.read_only,
        subPath: String(m?.sub_path ?? "").trim() || undefined,
      }))
      .filter((m) => m.name && m.mountPath) || undefined;

  const resources: any = {};
  if (v.requests_cpu || v.requests_memory) {
    resources.requests = {};
    if (v.requests_cpu) resources.requests.cpu = v.requests_cpu;
    if (v.requests_memory) resources.requests.memory = v.requests_memory;
  }
  if (v.limits_cpu || v.limits_memory) {
    resources.limits = {};
    if (v.limits_cpu) resources.limits.cpu = v.limits_cpu;
    if (v.limits_memory) resources.limits.memory = v.limits_memory;
  }
  const obj: any = {
    apiVersion: "batch/v1",
    kind: "Job",
    metadata: { name: v.name, namespace: v.namespace },
    spec: {
      template: {
        spec: {
          restartPolicy: v.restart_policy,
          tolerations: tolerations?.length ? tolerations : undefined,
          volumes: volumes.length ? volumes : undefined,
          containers: [
            {
              name: v.container_name || v.name,
              image: v.image,
              imagePullPolicy: v.image_pull_policy || undefined,
              command: v.command?.trim() ? ["sh", "-c", v.command.trim()] : undefined,
              env,
              resources: Object.keys(resources).length ? resources : undefined,
              volumeMounts,
            },
          ],
        },
      },
    },
  };
  return YAML.stringify(obj);
}

export function jobYamlToForm(yaml: string): JobFormValues | null {
  const obj: any = safeParseYaml(yaml);
  if (!obj || obj.kind !== "Job") return null;
  const c = obj?.spec?.template?.spec?.containers?.[0] ?? {};
  const envPairs = mapToEnvPairs(
    Array.isArray(c.env)
      ? Object.fromEntries(c.env.filter((e: any) => e?.name).map((e: any) => [String(e.name), String(e.value ?? "")]))
      : undefined,
  );
  let cmd = "";
  if (Array.isArray(c.command) && c.command.length >= 3 && c.command[0] === "sh" && c.command[1] === "-c") {
    cmd = String(c.command.slice(2).join(" "));
  }
  const rp = String(obj?.spec?.template?.spec?.restartPolicy ?? "Never");
  const restart_policy = rp === "OnFailure" ? "OnFailure" : "Never";
  const resReq = c?.resources?.requests ?? {};
  const resLim = c?.resources?.limits ?? {};
  const tolerations =
    Array.isArray(obj?.spec?.template?.spec?.tolerations) && obj.spec.template.spec.tolerations.length
      ? obj.spec.template.spec.tolerations.map((t: any) => ({
          key: t?.key,
          operator: (t?.operator || "Equal") as "Equal" | "Exists",
          value: t?.value,
          effect: t?.effect as "NoSchedule" | "PreferNoSchedule" | "NoExecute" | undefined,
          toleration_seconds: typeof t?.tolerationSeconds === "number" ? t.tolerationSeconds : undefined,
        }))
      : [{ key: "", operator: "Equal", value: "", effect: undefined }];
  const volumes =
    Array.isArray(obj?.spec?.template?.spec?.volumes) && obj.spec.template.spec.volumes.length
      ? obj.spec.template.spec.volumes.map((v: any) => ({
          name: v?.name,
          type: v?.configMap ? "configMap" : v?.secret ? "secret" : v?.persistentVolumeClaim ? "pvc" : "emptyDir",
          source_name: v?.configMap?.name || v?.secret?.secretName || v?.persistentVolumeClaim?.claimName || "",
        }))
      : [{ name: "", type: "emptyDir", source_name: "" }];
  const volumeMounts =
    Array.isArray(c?.volumeMounts) && c.volumeMounts.length
      ? c.volumeMounts.map((m: any) => ({
          name: m?.name,
          mount_path: m?.mountPath,
          read_only: !!m?.readOnly,
          sub_path: m?.subPath,
        }))
      : [{ name: "", mount_path: "", read_only: false, sub_path: "" }];
  return {
    name: String(obj?.metadata?.name ?? ""),
    namespace: String(obj?.metadata?.namespace ?? "default"),
    restart_policy,
    container_name: String(c?.name ?? ""),
    image: String(c?.image ?? ""),
    image_pull_policy: (c?.imagePullPolicy as "Always" | "IfNotPresent" | "Never" | undefined) ?? undefined,
    command: cmd || undefined,
    env_pairs: envPairs,
    requests_cpu: resReq?.cpu ? String(resReq.cpu) : undefined,
    requests_memory: resReq?.memory ? String(resReq.memory) : undefined,
    limits_cpu: resLim?.cpu ? String(resLim.cpu) : undefined,
    limits_memory: resLim?.memory ? String(resLim.memory) : undefined,
    tolerations,
    volumes,
    volume_mounts: volumeMounts,
  };
}

export function jobObjToForm(obj: any): JobFormValues | null {
  if (!obj) return null;
  const c = safeGet(obj, "spec.template.spec.containers[0]") ?? null;
  if (!c) return null;
  const envPairs = mapToEnvPairs(
    Array.isArray(c.env)
      ? Object.fromEntries(c.env.filter((e: any) => e?.name).map((e: any) => [String(e.name), String(e.value ?? "")]))
      : undefined,
  );
  let cmd = "";
  if (Array.isArray(c.command) && c.command.length >= 3 && c.command[0] === "sh" && c.command[1] === "-c") {
    cmd = String(c.command.slice(2).join(" "));
  }
  const rp = String(safeGet(obj, "spec.template.spec.restartPolicy") ?? "Never");
  const restart_policy: JobFormValues["restart_policy"] = rp === "OnFailure" ? "OnFailure" : "Never";
  const resReq = c?.resources?.requests ?? {};
  const resLim = c?.resources?.limits ?? {};
  const tolerations =
    Array.isArray(safeGet(obj, "spec.template.spec.tolerations")) && safeGet(obj, "spec.template.spec.tolerations")?.length
      ? (safeGet(obj, "spec.template.spec.tolerations") ?? []).map((t: any) => ({
          key: t?.key,
          operator: (t?.operator || "Equal") as "Equal" | "Exists",
          value: t?.value,
          effect: t?.effect as any,
          toleration_seconds: typeof t?.tolerationSeconds === "number" ? t.tolerationSeconds : undefined,
        }))
      : [];
  const volumes =
    Array.isArray(safeGet(obj, "spec.template.spec.volumes")) && safeGet(obj, "spec.template.spec.volumes")?.length
      ? (safeGet(obj, "spec.template.spec.volumes") ?? []).map((v: any) => ({
          name: v?.name,
          type: v?.configMap ? "configMap" : v?.secret ? "secret" : v?.persistentVolumeClaim ? "pvc" : "emptyDir",
          source_name: v?.configMap?.name || v?.secret?.secretName || v?.persistentVolumeClaim?.claimName || "",
        }))
      : [];
  const volumeMounts =
    Array.isArray(c?.volumeMounts) && c.volumeMounts.length
      ? c.volumeMounts.map((m: any) => ({
          name: m?.name,
          mount_path: m?.mountPath,
          read_only: !!m?.readOnly,
          sub_path: m?.subPath,
        }))
      : [];

  return {
    name: String(safeGet(obj, "metadata.name") ?? ""),
    namespace: String(safeGet(obj, "metadata.namespace") ?? "default"),
    restart_policy,
    container_name: String(c?.name ?? ""),
    image: String(c?.image ?? ""),
    image_pull_policy: c?.imagePullPolicy as any,
    command: cmd || undefined,
    env_pairs: envPairs,
    requests_cpu: resReq?.cpu ? String(resReq.cpu) : undefined,
    requests_memory: resReq?.memory ? String(resReq.memory) : undefined,
    limits_cpu: resLim?.cpu ? String(resLim.cpu) : undefined,
    limits_memory: resLim?.memory ? String(resLim.memory) : undefined,
    tolerations,
    volumes,
    volume_mounts: volumeMounts,
  };
}

export function EnvPairsFormItem({ name }: { name: string }) {
  return (
    <Form.List name={name}>
      {(fields, { add, remove }) => (
        <Space direction="vertical" style={{ width: "100%" }}>
          {fields.map((f) => (
            <Space key={f.key} style={{ display: "flex" }} align="baseline">
              <Form.Item name={[f.name, "key"]} rules={[{ required: true, message: "Key 必填" }]} style={{ marginBottom: 0 }}>
                <Input placeholder="KEY" style={{ width: 220 }} />
              </Form.Item>
              <Form.Item name={[f.name, "value"]} style={{ marginBottom: 0 }}>
                <Input placeholder="Value" style={{ width: 360 }} />
              </Form.Item>
              <Button onClick={() => remove(f.name)}>删除</Button>
            </Space>
          ))}
          <Button onClick={() => add({ key: "", value: "" })}>新增环境变量</Button>
        </Space>
      )}
    </Form.List>
  );
}

export function WorkloadFormModal<T extends object>(props: {
  title: string;
  open: boolean;
  loading?: boolean;
  form: FormInstance<T>;
  onCancel: () => void;
  onSubmit: (values: T) => void | Promise<void>;
  children: React.ReactNode;
}) {
  const { title, open, form, onCancel, onSubmit, children, loading } = props;
  return (
    <Modal
      title={title}
      open={open}
      onCancel={onCancel}
      onOk={() => void form.validateFields().then(onSubmit)}
      confirmLoading={loading}
      width={820}
      destroyOnClose
    >
      <Form form={form} layout="vertical">
        {children}
      </Form>
    </Modal>
  );
}

export function NameNamespaceItems() {
  return (
    <Space style={{ width: "100%" }} align="start">
      <Form.Item name="name" label="名称" rules={[{ required: true, message: "请输入名称" }]} style={{ flex: 1 }}>
        <Input />
      </Form.Item>
      <Form.Item name="namespace" label="命名空间" rules={[{ required: true, message: "请选择命名空间" }]} style={{ width: 240 }}>
        <Input />
      </Form.Item>
    </Space>
  );
}

export function ContainerCommonItems(opts?: { showPort?: boolean; showRestartPolicy?: boolean }) {
  return (
    <>
      <Space style={{ width: "100%" }} align="start">
        <Form.Item name="container_name" label="容器名" rules={[{ required: true, message: "请输入容器名" }]} style={{ flex: 1 }}>
          <Input />
        </Form.Item>
        <Form.Item name="image" label="镜像" rules={[{ required: true, message: "请输入镜像" }]} style={{ flex: 2 }}>
          <Input placeholder="nginx:latest" />
        </Form.Item>
      </Space>
      {opts?.showPort ? (
        <Form.Item name="port" label="容器端口（可选）">
          <InputNumber min={1} max={65535} style={{ width: 240 }} />
        </Form.Item>
      ) : null}
      {opts?.showRestartPolicy ? (
        <Form.Item name="restart_policy" label="RestartPolicy" rules={[{ required: true, message: "请选择" }]}>
          <Select
            style={{ width: 240 }}
            options={[
              { label: "Never", value: "Never" },
              { label: "OnFailure", value: "OnFailure" },
            ]}
          />
        </Form.Item>
      ) : null}
      <Form.Item name="command" label="启动命令（可选，sh -c）">
        <Input placeholder='例如：echo hello && sleep 5' />
      </Form.Item>
      <Form.Item label="环境变量">
        <EnvPairsFormItem name="env_pairs" />
      </Form.Item>
    </>
  );
}

export function WorkloadAdvancedItems() {
  return (
    <>
      <Space style={{ width: "100%" }} align="start">
        <Form.Item name="image_pull_policy" label="镜像拉取策略" style={{ width: 240 }}>
          <Select
            options={[
              { label: "IfNotPresent", value: "IfNotPresent" },
              { label: "Always", value: "Always" },
              { label: "Never", value: "Never" },
            ]}
          />
        </Form.Item>
      </Space>

      <Space style={{ width: "100%" }} align="start">
        <Form.Item name="requests_cpu" label="CPU Request" style={{ flex: 1 }}>
          <Input placeholder="100m" />
        </Form.Item>
        <Form.Item name="limits_cpu" label="CPU Limit" style={{ flex: 1 }}>
          <Input placeholder="500m" />
        </Form.Item>
      </Space>
      <Space style={{ width: "100%" }} align="start">
        <Form.Item name="requests_memory" label="Memory Request" style={{ flex: 1 }}>
          <Input placeholder="128Mi" />
        </Form.Item>
        <Form.Item name="limits_memory" label="Memory Limit" style={{ flex: 1 }}>
          <Input placeholder="512Mi" />
        </Form.Item>
      </Space>

      <Form.Item label="容忍（Tolerations）">
        <Form.List name="tolerations">
          {(fields, { add, remove }) => (
            <Space direction="vertical" style={{ width: "100%" }}>
              {fields.map((f) => (
                <Space key={f.key} style={{ display: "flex" }} align="baseline">
                  <Form.Item name={[f.name, "key"]} style={{ marginBottom: 0 }}>
                    <Input placeholder="key" style={{ width: 140 }} />
                  </Form.Item>
                  <Form.Item name={[f.name, "operator"]} initialValue="Equal" style={{ marginBottom: 0 }}>
                    <Select
                      style={{ width: 120 }}
                      options={[
                        { label: "Equal", value: "Equal" },
                        { label: "Exists", value: "Exists" },
                      ]}
                    />
                  </Form.Item>
                  <Form.Item name={[f.name, "value"]} style={{ marginBottom: 0 }}>
                    <Input placeholder="value" style={{ width: 140 }} />
                  </Form.Item>
                  <Form.Item name={[f.name, "effect"]} style={{ marginBottom: 0 }}>
                    <Select
                      allowClear
                      style={{ width: 150 }}
                      options={[
                        { label: "NoSchedule", value: "NoSchedule" },
                        { label: "PreferNoSchedule", value: "PreferNoSchedule" },
                        { label: "NoExecute", value: "NoExecute" },
                      ]}
                    />
                  </Form.Item>
                  <Form.Item name={[f.name, "toleration_seconds"]} style={{ marginBottom: 0 }}>
                    <InputNumber placeholder="seconds" style={{ width: 120 }} />
                  </Form.Item>
                  <Button onClick={() => remove(f.name)}>删除</Button>
                </Space>
              ))}
              <Button onClick={() => add({ key: "", operator: "Equal", value: "" })}>新增容忍</Button>
            </Space>
          )}
        </Form.List>
      </Form.Item>

      <Form.Item label="卷（Volumes）">
        <Form.List name="volumes">
          {(fields, { add, remove }) => (
            <Space direction="vertical" style={{ width: "100%" }}>
              {fields.map((f) => (
                <Space key={f.key} style={{ display: "flex" }} align="baseline">
                  <Form.Item name={[f.name, "name"]} rules={[{ required: true, message: "卷名必填" }]} style={{ marginBottom: 0 }}>
                    <Input placeholder="volume name" style={{ width: 180 }} />
                  </Form.Item>
                  <Form.Item name={[f.name, "type"]} initialValue="emptyDir" style={{ marginBottom: 0 }}>
                    <Select
                      style={{ width: 140 }}
                      options={[
                        { label: "emptyDir", value: "emptyDir" },
                        { label: "configMap", value: "configMap" },
                        { label: "secret", value: "secret" },
                        { label: "pvc", value: "pvc" },
                      ]}
                    />
                  </Form.Item>
                  <Form.Item name={[f.name, "source_name"]} style={{ marginBottom: 0 }}>
                    <Input placeholder="source name (cm/secret/pvc)" style={{ width: 240 }} />
                  </Form.Item>
                  <Button onClick={() => remove(f.name)}>删除</Button>
                </Space>
              ))}
              <Button onClick={() => add({ name: "", type: "emptyDir", source_name: "" })}>新增卷</Button>
            </Space>
          )}
        </Form.List>
      </Form.Item>

      <Form.Item label="卷挂载（VolumeMounts）">
        <Form.List name="volume_mounts">
          {(fields, { add, remove }) => (
            <Space direction="vertical" style={{ width: "100%" }}>
              {fields.map((f) => (
                <Space key={f.key} style={{ display: "flex" }} align="baseline">
                  <Form.Item name={[f.name, "name"]} rules={[{ required: true, message: "卷名必填" }]} style={{ marginBottom: 0 }}>
                    <Input placeholder="volume name" style={{ width: 180 }} />
                  </Form.Item>
                  <Form.Item name={[f.name, "mount_path"]} rules={[{ required: true, message: "挂载路径必填" }]} style={{ marginBottom: 0 }}>
                    <Input placeholder="/data" style={{ width: 220 }} />
                  </Form.Item>
                  <Form.Item name={[f.name, "sub_path"]} style={{ marginBottom: 0 }}>
                    <Input placeholder="subPath" style={{ width: 120 }} />
                  </Form.Item>
                  <Form.Item name={[f.name, "read_only"]} initialValue={false} style={{ marginBottom: 0 }}>
                    <Select
                      style={{ width: 110 }}
                      options={[
                        { label: "读写", value: false },
                        { label: "只读", value: true },
                      ]}
                    />
                  </Form.Item>
                  <Button onClick={() => remove(f.name)}>删除</Button>
                </Space>
              ))}
              <Button onClick={() => add({ name: "", mount_path: "", read_only: false })}>新增挂载</Button>
            </Space>
          )}
        </Form.List>
      </Form.Item>
    </>
  );
}

export function DeploymentHealthAndImagePullSecretsItems() {
  return (
    <>
      <Form.Item label="镜像拉取 Secret">
        <Form.List name="image_pull_secrets">
          {(fields, { add, remove }) => (
            <Space direction="vertical" style={{ width: "100%" }}>
              {fields.map((f) => (
                <Space key={f.key} style={{ display: "flex" }} align="baseline">
                  <Form.Item name={f.name} rules={[{ required: true, message: "请输入 Secret 名称" }]} style={{ marginBottom: 0 }}>
                    <Input placeholder="my-image-pull-secret" style={{ width: 260 }} />
                  </Form.Item>
                  <Button onClick={() => remove(f.name)}>删除</Button>
                </Space>
              ))}
              <Button onClick={() => add("")}>新增 Secret</Button>
            </Space>
          )}
        </Form.List>
      </Form.Item>

      <div style={{ fontWeight: 600, marginTop: 8 }}>Liveness Probe（存活探针）</div>
      <Space style={{ width: "100%" }} align="start">
        <Form.Item name="liveness_probe_type" label="探针类型" style={{ width: 220 }}>
          <Select
            allowClear
            options={[
              { label: "httpGet", value: "httpGet" },
              { label: "tcpSocket", value: "tcpSocket" },
            ]}
          />
        </Form.Item>
        <Form.Item name="liveness_http_path" label="HTTP path" style={{ flex: 1 }}>
          <Input placeholder="/health" />
        </Form.Item>
      </Space>
      <Space style={{ width: "100%" }} align="start">
        <Form.Item name="liveness_http_port" label="HTTP port" style={{ width: 180 }}>
          <InputNumber min={1} max={65535} style={{ width: "100%" }} />
        </Form.Item>
        <Form.Item name="liveness_http_scheme" label="HTTP scheme" style={{ width: 180 }}>
          <Select
            allowClear
            options={[
              { label: "HTTP", value: "HTTP" },
              { label: "HTTPS", value: "HTTPS" },
            ]}
          />
        </Form.Item>
        <Form.Item name="liveness_tcp_port" label="TCP port" style={{ width: 180 }}>
          <InputNumber min={1} max={65535} style={{ width: "100%" }} />
        </Form.Item>
      </Space>
      <Space style={{ width: "100%" }} align="start">
        <Form.Item name="liveness_initial_delay_seconds" label="initialDelaySeconds" style={{ flex: 1 }}>
          <InputNumber min={0} style={{ width: "100%" }} />
        </Form.Item>
        <Form.Item name="liveness_period_seconds" label="periodSeconds" style={{ flex: 1 }}>
          <InputNumber min={1} style={{ width: "100%" }} />
        </Form.Item>
      </Space>
      <Space style={{ width: "100%" }} align="start">
        <Form.Item name="liveness_timeout_seconds" label="timeoutSeconds" style={{ flex: 1 }}>
          <InputNumber min={1} style={{ width: "100%" }} />
        </Form.Item>
        <Form.Item name="liveness_failure_threshold" label="failureThreshold" style={{ flex: 1 }}>
          <InputNumber min={1} style={{ width: "100%" }} />
        </Form.Item>
      </Space>
      <Space style={{ width: "100%" }} align="start">
        <Form.Item name="liveness_success_threshold" label="successThreshold" style={{ flex: 1 }}>
          <InputNumber min={1} style={{ width: "100%" }} />
        </Form.Item>
      </Space>

      <div style={{ fontWeight: 600, marginTop: 16 }}>Readiness Probe（就绪探针）</div>
      <Space style={{ width: "100%" }} align="start">
        <Form.Item name="readiness_probe_type" label="探针类型" style={{ width: 220 }}>
          <Select
            allowClear
            options={[
              { label: "httpGet", value: "httpGet" },
              { label: "tcpSocket", value: "tcpSocket" },
            ]}
          />
        </Form.Item>
        <Form.Item name="readiness_http_path" label="HTTP path" style={{ flex: 1 }}>
          <Input placeholder="/ready" />
        </Form.Item>
      </Space>
      <Space style={{ width: "100%" }} align="start">
        <Form.Item name="readiness_http_port" label="HTTP port" style={{ width: 180 }}>
          <InputNumber min={1} max={65535} style={{ width: "100%" }} />
        </Form.Item>
        <Form.Item name="readiness_http_scheme" label="HTTP scheme" style={{ width: 180 }}>
          <Select
            allowClear
            options={[
              { label: "HTTP", value: "HTTP" },
              { label: "HTTPS", value: "HTTPS" },
            ]}
          />
        </Form.Item>
        <Form.Item name="readiness_tcp_port" label="TCP port" style={{ width: 180 }}>
          <InputNumber min={1} max={65535} style={{ width: "100%" }} />
        </Form.Item>
      </Space>
      <Space style={{ width: "100%" }} align="start">
        <Form.Item name="readiness_initial_delay_seconds" label="initialDelaySeconds" style={{ flex: 1 }}>
          <InputNumber min={0} style={{ width: "100%" }} />
        </Form.Item>
        <Form.Item name="readiness_period_seconds" label="periodSeconds" style={{ flex: 1 }}>
          <InputNumber min={1} style={{ width: "100%" }} />
        </Form.Item>
      </Space>
      <Space style={{ width: "100%" }} align="start">
        <Form.Item name="readiness_timeout_seconds" label="timeoutSeconds" style={{ flex: 1 }}>
          <InputNumber min={1} style={{ width: "100%" }} />
        </Form.Item>
        <Form.Item name="readiness_failure_threshold" label="failureThreshold" style={{ flex: 1 }}>
          <InputNumber min={1} style={{ width: "100%" }} />
        </Form.Item>
      </Space>
      <Space style={{ width: "100%" }} align="start">
        <Form.Item name="readiness_success_threshold" label="successThreshold" style={{ flex: 1 }}>
          <InputNumber min={1} style={{ width: "100%" }} />
        </Form.Item>
      </Space>
    </>
  );
}

