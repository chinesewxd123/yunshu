import { Button, Card, Col, Divider, Drawer, Form, Input, InputNumber, Row, Select, Space, Typography } from "antd";
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

type ProbeType = "httpGet" | "tcpSocket" | "exec";
type KVPair = { key?: string; value?: string };

function kvPairsToMap(pairs?: KVPair[]): Record<string, string> {
  const out: Record<string, string> = {};
  for (const p of pairs ?? []) {
    const k = String(p?.key ?? "").trim();
    if (!k) continue;
    out[k] = String(p?.value ?? "").trim();
  }
  return out;
}

function mapToKvPairs(m?: Record<string, string>): KVPair[] {
  const out = Object.entries(m ?? {}).map(([key, value]) => ({ key, value: String(value ?? "") }));
  return out.length ? out : [{ key: "", value: "" }];
}

function parseIntOrStringPort(v: any): number | string | undefined {
  if (typeof v === "number" && Number.isFinite(v) && v > 0) return Math.trunc(v);
  const s = String(v ?? "").trim();
  if (!s) return undefined;
  if (/^\d+$/.test(s)) {
    const n = Number(s);
    if (Number.isFinite(n) && n > 0) return n;
  }
  return s;
}

function parseExecCommandJson(v?: string): string[] | undefined {
  const s = String(v ?? "").trim();
  if (!s) return undefined;
  try {
    const arr = JSON.parse(s);
    if (!Array.isArray(arr)) return undefined;
    const out = arr.map((x) => String(x)).filter((x) => x.trim() !== "");
    return out.length ? out : undefined;
  } catch {
    return undefined;
  }
}

function probeFromForm(prefix: "liveness" | "readiness" | "startup", v: any): any | undefined {
  const probeType: ProbeType | undefined = v?.[`${prefix}_probe_type`];
  if (!probeType) return undefined;

  const initialDelaySeconds = toNumberOrUndefined(v?.[`${prefix}_initial_delay_seconds`]);
  const periodSeconds = toNumberOrUndefined(v?.[`${prefix}_period_seconds`]);
  const timeoutSeconds = toNumberOrUndefined(v?.[`${prefix}_timeout_seconds`]);
  const failureThreshold = toNumberOrUndefined(v?.[`${prefix}_failure_threshold`]);
  const successThreshold = toNumberOrUndefined(v?.[`${prefix}_success_threshold`]);

  if (probeType === "httpGet") {
    const port = parseIntOrStringPort(v?.[`${prefix}_http_port`]);
    if (!port) return undefined;
    const path = String(v?.[`${prefix}_http_path`] ?? "").trim() || "/";
    const scheme = String(v?.[`${prefix}_http_scheme`] ?? "").trim() || undefined;
    return {
      httpGet: { path, port, ...(scheme ? { scheme } : {}) },
      ...(initialDelaySeconds !== undefined ? { initialDelaySeconds } : {}),
      ...(periodSeconds !== undefined ? { periodSeconds } : {}),
      ...(timeoutSeconds !== undefined ? { timeoutSeconds } : {}),
      ...(failureThreshold !== undefined ? { failureThreshold } : {}),
      ...(successThreshold !== undefined ? { successThreshold } : {}),
    };
  }

  if (probeType === "tcpSocket") {
    const port = parseIntOrStringPort(v?.[`${prefix}_tcp_port`]);
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

  if (probeType === "exec") {
    const command = parseExecCommandJson(v?.[`${prefix}_exec_command`]);
    if (!command?.length) return undefined;
    return {
      exec: { command },
      ...(initialDelaySeconds !== undefined ? { initialDelaySeconds } : {}),
      ...(periodSeconds !== undefined ? { periodSeconds } : {}),
      ...(timeoutSeconds !== undefined ? { timeoutSeconds } : {}),
      ...(failureThreshold !== undefined ? { failureThreshold } : {}),
      ...(successThreshold !== undefined ? { successThreshold } : {}),
    };
  }

  return undefined;
}

function probeToForm(prefix: "liveness" | "readiness" | "startup", probe: any): Record<string, unknown> {
  if (!probe) return {};
  const out: Record<string, unknown> = {};
  const type: ProbeType | undefined = probe?.httpGet ? "httpGet" : probe?.tcpSocket ? "tcpSocket" : probe?.exec ? "exec" : undefined;
  if (!type) return out;
  out[`${prefix}_probe_type`] = type;
  if (type === "httpGet") {
    out[`${prefix}_http_path`] = probe?.httpGet?.path != null ? String(probe.httpGet.path) : "";
    out[`${prefix}_http_port`] = probe?.httpGet?.port != null ? String(probe.httpGet.port) : undefined;
    out[`${prefix}_http_scheme`] = probe?.httpGet?.scheme ? String(probe.httpGet.scheme) : undefined;
  }
  if (type === "tcpSocket") {
    out[`${prefix}_tcp_port`] = probe?.tcpSocket?.port != null ? String(probe.tcpSocket.port) : undefined;
  }
  if (type === "exec") {
    out[`${prefix}_exec_command`] = Array.isArray(probe?.exec?.command) ? JSON.stringify(probe.exec.command) : "";
  }
  out[`${prefix}_initial_delay_seconds`] = toNumberOrUndefined(probe?.initialDelaySeconds);
  out[`${prefix}_period_seconds`] = toNumberOrUndefined(probe?.periodSeconds);
  out[`${prefix}_timeout_seconds`] = toNumberOrUndefined(probe?.timeoutSeconds);
  out[`${prefix}_failure_threshold`] = toNumberOrUndefined(probe?.failureThreshold);
  out[`${prefix}_success_threshold`] = toNumberOrUndefined(probe?.successThreshold);
  return out;
}

export type DeploymentFormValues = {
  name: string;
  namespace: string;
  replicas: number;
  container_name: string;
  image: string;
  image_pull_policy?: "Always" | "IfNotPresent" | "Never";
  port?: number;
  port_name?: string;
  command?: string;
  env_pairs?: EnvPair[];
  requests_cpu?: string;
  requests_memory?: string;
  limits_cpu?: string;
  limits_memory?: string;
  node_selector_pairs?: KVPair[];
  affinity_yaml?: string;
  strategy_type?: "RollingUpdate" | "Recreate";
  rolling_update_max_surge?: string;
  rolling_update_max_unavailable?: string;
  revision_history_limit?: number;
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
  liveness_probe_type?: ProbeType;
  liveness_http_path?: string;
  liveness_http_port?: string;
  liveness_http_scheme?: "HTTP" | "HTTPS";
  liveness_tcp_port?: string;
  liveness_exec_command?: string;
  liveness_initial_delay_seconds?: number;
  liveness_period_seconds?: number;
  liveness_timeout_seconds?: number;
  liveness_failure_threshold?: number;
  liveness_success_threshold?: number;
  readiness_probe_type?: ProbeType;
  readiness_http_path?: string;
  readiness_http_port?: string;
  readiness_http_scheme?: "HTTP" | "HTTPS";
  readiness_tcp_port?: string;
  readiness_exec_command?: string;
  readiness_initial_delay_seconds?: number;
  readiness_period_seconds?: number;
  readiness_timeout_seconds?: number;
  readiness_failure_threshold?: number;
  readiness_success_threshold?: number;
  startup_probe_type?: ProbeType;
  startup_http_path?: string;
  startup_http_port?: string;
  startup_http_scheme?: "HTTP" | "HTTPS";
  startup_tcp_port?: string;
  startup_exec_command?: string;
  startup_initial_delay_seconds?: number;
  startup_period_seconds?: number;
  startup_timeout_seconds?: number;
  startup_failure_threshold?: number;
  startup_success_threshold?: number;
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

  const livenessProbe = probeFromForm("liveness", v);
  const readinessProbe = probeFromForm("readiness", v);
  const startupProbe = probeFromForm("startup", v);
  const nodeSelector = kvPairsToMap(v.node_selector_pairs);
  const affinity = safeParseYaml(String(v.affinity_yaml ?? "").trim() || "");

  const obj: any = {
    apiVersion: "apps/v1",
    kind: "Deployment",
    metadata: { name: v.name, namespace: v.namespace },
    spec: {
      replicas: v.replicas,
      strategy: v.strategy_type
        ? {
            type: v.strategy_type,
            ...(v.strategy_type === "RollingUpdate"
              ? {
                  rollingUpdate: {
                    maxSurge: String(v.rolling_update_max_surge ?? "").trim() || undefined,
                    maxUnavailable: String(v.rolling_update_max_unavailable ?? "").trim() || undefined,
                  },
                }
              : {}),
          }
        : undefined,
      revisionHistoryLimit: typeof v.revision_history_limit === "number" ? v.revision_history_limit : undefined,
      selector: { matchLabels: { app: v.name } },
      template: {
        metadata: { labels: { app: v.name } },
        spec: {
          containers: [
            {
              name: v.container_name || v.name,
              image: v.image,
              imagePullPolicy: v.image_pull_policy || undefined,
              ports: v.port ? [{ name: String(v.port_name ?? "").trim() || undefined, containerPort: v.port }] : undefined,
              command: v.command?.trim() ? ["sh", "-c", v.command.trim()] : undefined,
              env,
              resources: Object.keys(resources).length ? resources : undefined,
              volumeMounts,
                  ...(livenessProbe ? { livenessProbe } : {}),
                  ...(readinessProbe ? { readinessProbe } : {}),
                  ...(startupProbe ? { startupProbe } : {}),
            },
          ],
              imagePullSecrets: imagePullSecrets.length ? imagePullSecrets : undefined,
          volumes: volumes.length ? volumes : undefined,
          tolerations: tolerations?.length ? tolerations : undefined,
          nodeSelector: Object.keys(nodeSelector).length ? nodeSelector : undefined,
          affinity: affinity || undefined,
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
  const sp = c?.startupProbe;

  return {
    name: String(obj?.metadata?.name ?? ""),
    namespace: String(obj?.metadata?.namespace ?? "default"),
    replicas: Number(obj?.spec?.replicas ?? 1),
    container_name: String(c?.name ?? ""),
    image: String(c?.image ?? ""),
    image_pull_policy: (c?.imagePullPolicy as "Always" | "IfNotPresent" | "Never" | undefined) ?? undefined,
    image_pull_secrets: imagePullSecrets,
    port: typeof port === "number" ? port : undefined,
    port_name: c?.ports?.[0]?.name ? String(c.ports[0].name) : undefined,
    command: cmd || undefined,
    env_pairs: envPairs,
    requests_cpu: resReq?.cpu ? String(resReq.cpu) : undefined,
    requests_memory: resReq?.memory ? String(resReq.memory) : undefined,
    limits_cpu: resLim?.cpu ? String(resLim.cpu) : undefined,
    limits_memory: resLim?.memory ? String(resLim.memory) : undefined,
    tolerations,
    volumes,
    volume_mounts: volumeMounts,
    node_selector_pairs: mapToKvPairs(obj?.spec?.template?.spec?.nodeSelector),
    affinity_yaml: obj?.spec?.template?.spec?.affinity ? YAML.stringify(obj.spec.template.spec.affinity) : undefined,
    strategy_type: obj?.spec?.strategy?.type as DeploymentFormValues["strategy_type"],
    rolling_update_max_surge: obj?.spec?.strategy?.rollingUpdate?.maxSurge != null ? String(obj.spec.strategy.rollingUpdate.maxSurge) : undefined,
    rolling_update_max_unavailable:
      obj?.spec?.strategy?.rollingUpdate?.maxUnavailable != null ? String(obj.spec.strategy.rollingUpdate.maxUnavailable) : undefined,
    revision_history_limit: toNumberOrUndefined(obj?.spec?.revisionHistoryLimit),
    ...probeToForm("liveness", lp),
    ...probeToForm("readiness", rp),
    ...probeToForm("startup", sp),
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
  const sp = c?.startupProbe;

  return {
    name: String(safeGet(obj, "metadata.name") ?? ""),
    namespace: String(safeGet(obj, "metadata.namespace") ?? "default"),
    replicas: Number(safeGet(obj, "spec.replicas") ?? 1),
    container_name: String(c?.name ?? ""),
    image: String(c?.image ?? ""),
    image_pull_policy: c?.imagePullPolicy as any,
    image_pull_secrets: imagePullSecrets,
    port: typeof port === "number" ? port : undefined,
    port_name: c?.ports?.[0]?.name ? String(c.ports[0].name) : undefined,
    command: cmd || undefined,
    env_pairs: envPairs,
    requests_cpu: resReq?.cpu ? String(resReq.cpu) : undefined,
    requests_memory: resReq?.memory ? String(resReq.memory) : undefined,
    limits_cpu: resLim?.cpu ? String(resLim.cpu) : undefined,
    limits_memory: resLim?.memory ? String(resLim.memory) : undefined,
    tolerations,
    volumes,
    volume_mounts: volumeMounts,
    node_selector_pairs: mapToKvPairs(safeGet(obj, "spec.template.spec.nodeSelector")),
    affinity_yaml: safeGet(obj, "spec.template.spec.affinity") ? YAML.stringify(safeGet(obj, "spec.template.spec.affinity")) : undefined,
    strategy_type: safeGet(obj, "spec.strategy.type") as DeploymentFormValues["strategy_type"],
    rolling_update_max_surge: safeGet(obj, "spec.strategy.rollingUpdate.maxSurge") != null ? String(safeGet(obj, "spec.strategy.rollingUpdate.maxSurge")) : undefined,
    rolling_update_max_unavailable:
      safeGet(obj, "spec.strategy.rollingUpdate.maxUnavailable") != null ? String(safeGet(obj, "spec.strategy.rollingUpdate.maxUnavailable")) : undefined,
    revision_history_limit: toNumberOrUndefined(safeGet(obj, "spec.revisionHistoryLimit")),
    ...probeToForm("liveness", lp),
    ...probeToForm("readiness", rp),
    ...probeToForm("startup", sp),
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
  port_name?: string;
  command?: string;
  env_pairs?: EnvPair[];
  requests_cpu?: string;
  requests_memory?: string;
  limits_cpu?: string;
  limits_memory?: string;
  node_selector_pairs?: KVPair[];
  affinity_yaml?: string;
  update_strategy_type?: "RollingUpdate" | "OnDelete";
  rolling_update_partition?: number;
  revision_history_limit?: number;
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
  const nodeSelector = kvPairsToMap(v.node_selector_pairs);
  const affinity = safeParseYaml(String(v.affinity_yaml ?? "").trim() || "");

  const obj: any = {
    apiVersion: "apps/v1",
    kind: "StatefulSet",
    metadata: { name: v.name, namespace: v.namespace },
    spec: {
      serviceName: v.service_name || `${v.name}-headless`,
      replicas: v.replicas,
      updateStrategy: v.update_strategy_type
        ? {
            type: v.update_strategy_type,
            ...(v.update_strategy_type === "RollingUpdate" && typeof v.rolling_update_partition === "number"
              ? { rollingUpdate: { partition: v.rolling_update_partition } }
              : {}),
          }
        : undefined,
      revisionHistoryLimit: typeof v.revision_history_limit === "number" ? v.revision_history_limit : undefined,
      selector: { matchLabels: { app: v.name } },
      template: {
        metadata: { labels: { app: v.name } },
        spec: {
          containers: [
            {
              name: v.container_name || v.name,
              image: v.image,
              imagePullPolicy: v.image_pull_policy || undefined,
              ports: v.port ? [{ name: String(v.port_name ?? "").trim() || undefined, containerPort: v.port }] : undefined,
              command: v.command?.trim() ? ["sh", "-c", v.command.trim()] : undefined,
              env,
              resources: Object.keys(resources).length ? resources : undefined,
              volumeMounts,
            },
          ],
          volumes: volumes.length ? volumes : undefined,
          tolerations: tolerations?.length ? tolerations : undefined,
          nodeSelector: Object.keys(nodeSelector).length ? nodeSelector : undefined,
          affinity: affinity || undefined,
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
    port_name: c?.ports?.[0]?.name ? String(c.ports[0].name) : undefined,
    command: cmd || undefined,
    env_pairs: envPairs,
    requests_cpu: resReq?.cpu ? String(resReq.cpu) : undefined,
    requests_memory: resReq?.memory ? String(resReq.memory) : undefined,
    limits_cpu: resLim?.cpu ? String(resLim.cpu) : undefined,
    limits_memory: resLim?.memory ? String(resLim.memory) : undefined,
    node_selector_pairs: mapToKvPairs(obj?.spec?.template?.spec?.nodeSelector),
    affinity_yaml: obj?.spec?.template?.spec?.affinity ? YAML.stringify(obj.spec.template.spec.affinity) : undefined,
    update_strategy_type: obj?.spec?.updateStrategy?.type as StatefulSetFormValues["update_strategy_type"],
    rolling_update_partition: toNumberOrUndefined(obj?.spec?.updateStrategy?.rollingUpdate?.partition),
    revision_history_limit: toNumberOrUndefined(obj?.spec?.revisionHistoryLimit),
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
    port_name: c?.ports?.[0]?.name ? String(c.ports[0].name) : undefined,
    command: cmd || undefined,
    env_pairs: envPairs,
    requests_cpu: resReq?.cpu ? String(resReq.cpu) : undefined,
    requests_memory: resReq?.memory ? String(resReq.memory) : undefined,
    limits_cpu: resLim?.cpu ? String(resLim.cpu) : undefined,
    limits_memory: resLim?.memory ? String(resLim.memory) : undefined,
    node_selector_pairs: mapToKvPairs(safeGet(obj, "spec.template.spec.nodeSelector")),
    affinity_yaml: safeGet(obj, "spec.template.spec.affinity") ? YAML.stringify(safeGet(obj, "spec.template.spec.affinity")) : undefined,
    update_strategy_type: safeGet(obj, "spec.updateStrategy.type") as StatefulSetFormValues["update_strategy_type"],
    rolling_update_partition: toNumberOrUndefined(safeGet(obj, "spec.updateStrategy.rollingUpdate.partition")),
    revision_history_limit: toNumberOrUndefined(safeGet(obj, "spec.revisionHistoryLimit")),
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
  node_selector_pairs?: KVPair[];
  affinity_yaml?: string;
  parallelism?: number;
  completions?: number;
  backoff_limit?: number;
  active_deadline_seconds?: number;
  ttl_seconds_after_finished?: number;
  liveness_probe_type?: DeploymentFormValues["liveness_probe_type"];
  liveness_http_path?: DeploymentFormValues["liveness_http_path"];
  liveness_http_port?: DeploymentFormValues["liveness_http_port"];
  liveness_http_scheme?: DeploymentFormValues["liveness_http_scheme"];
  liveness_tcp_port?: DeploymentFormValues["liveness_tcp_port"];
  liveness_exec_command?: DeploymentFormValues["liveness_exec_command"];
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
  readiness_exec_command?: DeploymentFormValues["readiness_exec_command"];
  readiness_initial_delay_seconds?: DeploymentFormValues["readiness_initial_delay_seconds"];
  readiness_period_seconds?: DeploymentFormValues["readiness_period_seconds"];
  readiness_timeout_seconds?: DeploymentFormValues["readiness_timeout_seconds"];
  readiness_failure_threshold?: DeploymentFormValues["readiness_failure_threshold"];
  readiness_success_threshold?: DeploymentFormValues["readiness_success_threshold"];
  startup_probe_type?: DeploymentFormValues["startup_probe_type"];
  startup_http_path?: DeploymentFormValues["startup_http_path"];
  startup_http_port?: DeploymentFormValues["startup_http_port"];
  startup_http_scheme?: DeploymentFormValues["startup_http_scheme"];
  startup_tcp_port?: DeploymentFormValues["startup_tcp_port"];
  startup_exec_command?: DeploymentFormValues["startup_exec_command"];
  startup_initial_delay_seconds?: DeploymentFormValues["startup_initial_delay_seconds"];
  startup_period_seconds?: DeploymentFormValues["startup_period_seconds"];
  startup_timeout_seconds?: DeploymentFormValues["startup_timeout_seconds"];
  startup_failure_threshold?: DeploymentFormValues["startup_failure_threshold"];
  startup_success_threshold?: DeploymentFormValues["startup_success_threshold"];
};

export type DaemonSetFormValues = {
  name: string;
  namespace: string;
  container_name: string;
  image: string;
  image_pull_policy?: "Always" | "IfNotPresent" | "Never";
  port?: number;
  port_name?: string;
  command?: string;
  env_pairs?: EnvPair[];
  requests_cpu?: string;
  requests_memory?: string;
  limits_cpu?: string;
  limits_memory?: string;
  node_selector_pairs?: KVPair[];
  affinity_yaml?: string;
  update_strategy_type?: "RollingUpdate" | "OnDelete";
  rolling_update_max_surge?: string;
  rolling_update_max_unavailable?: string;
  revision_history_limit?: number;
  tolerations?: DeploymentFormValues["tolerations"];
  volumes?: DeploymentFormValues["volumes"];
  volume_mounts?: DeploymentFormValues["volume_mounts"];
  image_pull_secrets?: DeploymentFormValues["image_pull_secrets"];
  liveness_probe_type?: DeploymentFormValues["liveness_probe_type"];
  liveness_http_path?: DeploymentFormValues["liveness_http_path"];
  liveness_http_port?: DeploymentFormValues["liveness_http_port"];
  liveness_http_scheme?: DeploymentFormValues["liveness_http_scheme"];
  liveness_tcp_port?: DeploymentFormValues["liveness_tcp_port"];
  liveness_exec_command?: DeploymentFormValues["liveness_exec_command"];
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
  readiness_exec_command?: DeploymentFormValues["readiness_exec_command"];
  readiness_initial_delay_seconds?: DeploymentFormValues["readiness_initial_delay_seconds"];
  readiness_period_seconds?: DeploymentFormValues["readiness_period_seconds"];
  readiness_timeout_seconds?: DeploymentFormValues["readiness_timeout_seconds"];
  readiness_failure_threshold?: DeploymentFormValues["readiness_failure_threshold"];
  readiness_success_threshold?: DeploymentFormValues["readiness_success_threshold"];
  startup_probe_type?: DeploymentFormValues["startup_probe_type"];
  startup_http_path?: DeploymentFormValues["startup_http_path"];
  startup_http_port?: DeploymentFormValues["startup_http_port"];
  startup_http_scheme?: DeploymentFormValues["startup_http_scheme"];
  startup_tcp_port?: DeploymentFormValues["startup_tcp_port"];
  startup_exec_command?: DeploymentFormValues["startup_exec_command"];
  startup_initial_delay_seconds?: DeploymentFormValues["startup_initial_delay_seconds"];
  startup_period_seconds?: DeploymentFormValues["startup_period_seconds"];
  startup_timeout_seconds?: DeploymentFormValues["startup_timeout_seconds"];
  startup_failure_threshold?: DeploymentFormValues["startup_failure_threshold"];
  startup_success_threshold?: DeploymentFormValues["startup_success_threshold"];
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

  const livenessProbe = probeFromForm("liveness", v);
  const readinessProbe = probeFromForm("readiness", v);
  const startupProbe = probeFromForm("startup", v);
  const nodeSelector = kvPairsToMap(v.node_selector_pairs);
  const affinity = safeParseYaml(String(v.affinity_yaml ?? "").trim() || "");

  const obj: any = {
    apiVersion: "apps/v1",
    kind: "DaemonSet",
    metadata: { name: v.name, namespace: v.namespace },
    spec: {
      updateStrategy: v.update_strategy_type
        ? {
            type: v.update_strategy_type,
            ...(v.update_strategy_type === "RollingUpdate"
              ? {
                  rollingUpdate: {
                    maxSurge: String(v.rolling_update_max_surge ?? "").trim() || undefined,
                    maxUnavailable: String(v.rolling_update_max_unavailable ?? "").trim() || undefined,
                  },
                }
              : {}),
          }
        : undefined,
      revisionHistoryLimit: typeof v.revision_history_limit === "number" ? v.revision_history_limit : undefined,
      selector: { matchLabels: { app: v.name } },
      template: {
        metadata: { labels: { app: v.name } },
        spec: {
          imagePullSecrets: imagePullSecrets.length ? imagePullSecrets : undefined,
          tolerations: tolerations?.length ? tolerations : undefined,
          nodeSelector: Object.keys(nodeSelector).length ? nodeSelector : undefined,
          affinity: affinity || undefined,
          volumes: volumes.length ? volumes : undefined,
          containers: [
            {
              name: v.container_name || v.name,
              image: v.image,
              imagePullPolicy: v.image_pull_policy || undefined,
              ports: v.port ? [{ name: String(v.port_name ?? "").trim() || undefined, containerPort: v.port }] : undefined,
              command: v.command?.trim() ? ["sh", "-c", v.command.trim()] : undefined,
              env,
              resources: Object.keys(resources).length ? resources : undefined,
              volumeMounts,
              ...(livenessProbe ? { livenessProbe } : {}),
              ...(readinessProbe ? { readinessProbe } : {}),
              ...(startupProbe ? { startupProbe } : {}),
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
  const sp = c?.startupProbe;

  return {
    name: String(safeGet(obj, "metadata.name") ?? ""),
    namespace: String(safeGet(obj, "metadata.namespace") ?? "default"),
    container_name: String(c?.name ?? ""),
    image: String(c?.image ?? ""),
    image_pull_policy: c?.imagePullPolicy as any,
    image_pull_secrets: imagePullSecrets,
    port: typeof port === "number" ? port : undefined,
    port_name: c?.ports?.[0]?.name ? String(c.ports[0].name) : undefined,
    command: cmd || undefined,
    env_pairs: envPairs,
    requests_cpu: resReq?.cpu ? String(resReq.cpu) : undefined,
    requests_memory: resReq?.memory ? String(resReq.memory) : undefined,
    limits_cpu: resLim?.cpu ? String(resLim.cpu) : undefined,
    limits_memory: resLim?.memory ? String(resLim.memory) : undefined,
    node_selector_pairs: mapToKvPairs(safeGet(obj, "spec.template.spec.nodeSelector")),
    affinity_yaml: safeGet(obj, "spec.template.spec.affinity") ? YAML.stringify(safeGet(obj, "spec.template.spec.affinity")) : undefined,
    update_strategy_type: safeGet(obj, "spec.updateStrategy.type") as DaemonSetFormValues["update_strategy_type"],
    rolling_update_max_surge: safeGet(obj, "spec.updateStrategy.rollingUpdate.maxSurge") != null ? String(safeGet(obj, "spec.updateStrategy.rollingUpdate.maxSurge")) : undefined,
    rolling_update_max_unavailable:
      safeGet(obj, "spec.updateStrategy.rollingUpdate.maxUnavailable") != null ? String(safeGet(obj, "spec.updateStrategy.rollingUpdate.maxUnavailable")) : undefined,
    revision_history_limit: toNumberOrUndefined(safeGet(obj, "spec.revisionHistoryLimit")),
    tolerations,
    volumes,
    volume_mounts: volumeMounts,
    ...probeToForm("liveness", lp),
    ...probeToForm("readiness", rp),
    ...probeToForm("startup", sp),
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
  node_selector_pairs?: KVPair[];
  affinity_yaml?: string;
  concurrency_policy?: "Allow" | "Forbid" | "Replace";
  successful_jobs_history_limit?: number;
  failed_jobs_history_limit?: number;
  starting_deadline_seconds?: number;
  parallelism?: number;
  completions?: number;
  backoff_limit?: number;
  active_deadline_seconds?: number;
  ttl_seconds_after_finished?: number;
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
  startup_probe_type?: DeploymentFormValues["startup_probe_type"];
  startup_http_path?: DeploymentFormValues["startup_http_path"];
  startup_http_port?: DeploymentFormValues["startup_http_port"];
  startup_http_scheme?: DeploymentFormValues["startup_http_scheme"];
  startup_tcp_port?: DeploymentFormValues["startup_tcp_port"];
  startup_exec_command?: DeploymentFormValues["startup_exec_command"];
  startup_initial_delay_seconds?: DeploymentFormValues["startup_initial_delay_seconds"];
  startup_period_seconds?: DeploymentFormValues["startup_period_seconds"];
  startup_timeout_seconds?: DeploymentFormValues["startup_timeout_seconds"];
  startup_failure_threshold?: DeploymentFormValues["startup_failure_threshold"];
  startup_success_threshold?: DeploymentFormValues["startup_success_threshold"];
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

  const livenessProbe = probeFromForm("liveness", v);
  const readinessProbe = probeFromForm("readiness", v);
  const startupProbe = probeFromForm("startup", v);
  const nodeSelector = kvPairsToMap(v.node_selector_pairs);
  const affinity = safeParseYaml(String(v.affinity_yaml ?? "").trim() || "");

  const obj: any = {
    apiVersion: "batch/v1",
    kind: "CronJob",
    metadata: { name: v.name, namespace: v.namespace },
    spec: {
      schedule: v.schedule,
      suspend: typeof v.suspend === "boolean" ? v.suspend : undefined,
      concurrencyPolicy: v.concurrency_policy || undefined,
      successfulJobsHistoryLimit:
        typeof v.successful_jobs_history_limit === "number" ? v.successful_jobs_history_limit : undefined,
      failedJobsHistoryLimit: typeof v.failed_jobs_history_limit === "number" ? v.failed_jobs_history_limit : undefined,
      startingDeadlineSeconds: typeof v.starting_deadline_seconds === "number" ? v.starting_deadline_seconds : undefined,
      jobTemplate: {
        spec: {
          parallelism: typeof v.parallelism === "number" ? v.parallelism : undefined,
          completions: typeof v.completions === "number" ? v.completions : undefined,
          backoffLimit: typeof v.backoff_limit === "number" ? v.backoff_limit : undefined,
          activeDeadlineSeconds: typeof v.active_deadline_seconds === "number" ? v.active_deadline_seconds : undefined,
          ttlSecondsAfterFinished: typeof v.ttl_seconds_after_finished === "number" ? v.ttl_seconds_after_finished : undefined,
          template: {
            spec: {
              restartPolicy: v.restart_policy,
              imagePullSecrets: imagePullSecrets.length ? imagePullSecrets : undefined,
              tolerations: tolerations?.length ? tolerations : undefined,
              nodeSelector: Object.keys(nodeSelector).length ? nodeSelector : undefined,
              affinity: affinity || undefined,
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
                  ...(startupProbe ? { startupProbe } : {}),
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
  const sp = c?.startupProbe;

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
    node_selector_pairs: mapToKvPairs(safeGet(obj, "spec.jobTemplate.spec.template.spec.nodeSelector")),
    affinity_yaml: safeGet(obj, "spec.jobTemplate.spec.template.spec.affinity")
      ? YAML.stringify(safeGet(obj, "spec.jobTemplate.spec.template.spec.affinity"))
      : undefined,
    concurrency_policy: safeGet(obj, "spec.concurrencyPolicy") as CronJobFormValues["concurrency_policy"],
    successful_jobs_history_limit: toNumberOrUndefined(safeGet(obj, "spec.successfulJobsHistoryLimit")),
    failed_jobs_history_limit: toNumberOrUndefined(safeGet(obj, "spec.failedJobsHistoryLimit")),
    starting_deadline_seconds: toNumberOrUndefined(safeGet(obj, "spec.startingDeadlineSeconds")),
    parallelism: toNumberOrUndefined(safeGet(obj, "spec.jobTemplate.spec.parallelism")),
    completions: toNumberOrUndefined(safeGet(obj, "spec.jobTemplate.spec.completions")),
    backoff_limit: toNumberOrUndefined(safeGet(obj, "spec.jobTemplate.spec.backoffLimit")),
    active_deadline_seconds: toNumberOrUndefined(safeGet(obj, "spec.jobTemplate.spec.activeDeadlineSeconds")),
    ttl_seconds_after_finished: toNumberOrUndefined(safeGet(obj, "spec.jobTemplate.spec.ttlSecondsAfterFinished")),
    ...probeToForm("liveness", lp),
    ...probeToForm("readiness", rp2),
    ...probeToForm("startup", sp),
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
  const livenessProbe = probeFromForm("liveness", v);
  const readinessProbe = probeFromForm("readiness", v);
  const startupProbe = probeFromForm("startup", v);
  const nodeSelector = kvPairsToMap(v.node_selector_pairs);
  const affinity = safeParseYaml(String(v.affinity_yaml ?? "").trim() || "");
  const obj: any = {
    apiVersion: "batch/v1",
    kind: "Job",
    metadata: { name: v.name, namespace: v.namespace },
    spec: {
      parallelism: typeof v.parallelism === "number" ? v.parallelism : undefined,
      completions: typeof v.completions === "number" ? v.completions : undefined,
      backoffLimit: typeof v.backoff_limit === "number" ? v.backoff_limit : undefined,
      activeDeadlineSeconds: typeof v.active_deadline_seconds === "number" ? v.active_deadline_seconds : undefined,
      ttlSecondsAfterFinished: typeof v.ttl_seconds_after_finished === "number" ? v.ttl_seconds_after_finished : undefined,
      template: {
        spec: {
          restartPolicy: v.restart_policy,
          tolerations: tolerations?.length ? tolerations : undefined,
          nodeSelector: Object.keys(nodeSelector).length ? nodeSelector : undefined,
          affinity: affinity || undefined,
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
              ...(startupProbe ? { startupProbe } : {}),
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
  const lp = c?.livenessProbe;
  const rp2 = c?.readinessProbe;
  const sp = c?.startupProbe;
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
    node_selector_pairs: mapToKvPairs(safeGet(obj, "spec.template.spec.nodeSelector")),
    affinity_yaml: safeGet(obj, "spec.template.spec.affinity") ? YAML.stringify(safeGet(obj, "spec.template.spec.affinity")) : undefined,
    parallelism: toNumberOrUndefined(safeGet(obj, "spec.parallelism")),
    completions: toNumberOrUndefined(safeGet(obj, "spec.completions")),
    backoff_limit: toNumberOrUndefined(safeGet(obj, "spec.backoffLimit")),
    active_deadline_seconds: toNumberOrUndefined(safeGet(obj, "spec.activeDeadlineSeconds")),
    ttl_seconds_after_finished: toNumberOrUndefined(safeGet(obj, "spec.ttlSecondsAfterFinished")),
    ...probeToForm("liveness", lp),
    ...probeToForm("readiness", rp2),
    ...probeToForm("startup", sp),
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
  const lp = c?.livenessProbe;
  const rp2 = c?.readinessProbe;
  const sp = c?.startupProbe;
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
    ...probeToForm("liveness", lp),
    ...probeToForm("readiness", rp2),
    ...probeToForm("startup", sp),
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
  /** 右侧抽屉宽度，默认 920 */
  drawerWidth?: number | string;
  /** 仅表单 + 底部确定，不包 Drawer（嵌入 YamlCrudPage「表单创建」Tab） */
  embedded?: boolean;
}) {
  const { title, open, form, onCancel, onSubmit, children, loading, drawerWidth = 920, embedded } = props;
  const formBody = (
    <Form form={form} layout="vertical" requiredMark="optional" scrollToFirstError>
      {children}
    </Form>
  );
  const footer = (
    <Space style={{ marginTop: 16 }}>
      <Button type="primary" loading={loading} onClick={() => void form.validateFields().then((v) => void onSubmit(v))}>
        确定
      </Button>
    </Space>
  );
  if (embedded) {
    return (
      <>
        {formBody}
        {footer}
      </>
    );
  }
  return (
    <Drawer
      title={title}
      placement="right"
      width={drawerWidth}
      open={open}
      onClose={onCancel}
      destroyOnClose
      maskClosable={false}
      styles={{ body: { paddingBottom: 24 } }}
      extra={
        <Space>
          <Button onClick={onCancel}>取消</Button>
          <Button type="primary" loading={loading} onClick={() => void form.validateFields().then((v) => void onSubmit(v))}>
            确定
          </Button>
        </Space>
      }
    >
      {formBody}
    </Drawer>
  );
}

export function NameNamespaceItems() {
  return (
    <Card size="small" title="基础信息" styles={{ body: { paddingBottom: 8 } }}>
      <Row gutter={16}>
        <Col xs={24} md={14}>
          <Form.Item name="name" label="名称" rules={[{ required: true, message: "请输入名称" }]}>
            <Input />
          </Form.Item>
        </Col>
        <Col xs={24} md={10}>
          <Form.Item name="namespace" label="命名空间" rules={[{ required: true, message: "请选择命名空间" }]}>
            <Input />
          </Form.Item>
        </Col>
      </Row>
    </Card>
  );
}

export function ContainerCommonItems(opts?: { showPort?: boolean; showRestartPolicy?: boolean }) {
  return (
    <Card size="small" title="容器信息" styles={{ body: { paddingBottom: 8 } }}>
      <Row gutter={16}>
        <Col xs={24} md={10}>
          <Form.Item name="container_name" label="容器名" rules={[{ required: true, message: "请输入容器名" }]}>
            <Input />
          </Form.Item>
        </Col>
        <Col xs={24} md={14}>
          <Form.Item name="image" label="镜像" rules={[{ required: true, message: "请输入镜像" }]}>
            <Input placeholder="nginx:latest" />
          </Form.Item>
        </Col>
      </Row>
      <Row gutter={16}>
        {opts?.showPort ? (
          <Col xs={24} md={8}>
            <Form.Item name="port" label="容器端口（可选）">
              <InputNumber min={1} max={65535} style={{ width: "100%" }} />
            </Form.Item>
          </Col>
        ) : null}
        {opts?.showPort ? (
          <Col xs={24} md={8}>
            <Form.Item name="port_name" label="端口名称（可选）" extra="例如：http、metrics（供 Probe/Service 引用）">
              <Input placeholder="http" />
            </Form.Item>
          </Col>
        ) : null}
        {opts?.showRestartPolicy ? (
          <Col xs={24} md={8}>
            <Form.Item name="restart_policy" label="RestartPolicy" rules={[{ required: true, message: "请选择" }]}>
              <Select
                options={[
                  { label: "Never", value: "Never" },
                  { label: "OnFailure", value: "OnFailure" },
                ]}
              />
            </Form.Item>
          </Col>
        ) : null}
      </Row>
      <Form.Item name="command" label="启动命令" extra="sh -c 执行；可不填">
        <Input placeholder='例如：echo hello && sleep 5' />
      </Form.Item>
      <Form.Item label="环境变量">
        <EnvPairsFormItem name="env_pairs" />
      </Form.Item>
    </Card>
  );
}

export function WorkloadAdvancedItems() {
  return (
    <Card size="small" title="资源与调度" styles={{ body: { paddingBottom: 8 } }}>
      <Row gutter={16}>
        <Col xs={24} md={8}>
          <Form.Item name="image_pull_policy" label="镜像拉取策略">
            <Select
              options={[
                { label: "IfNotPresent", value: "IfNotPresent" },
                { label: "Always", value: "Always" },
                { label: "Never", value: "Never" },
              ]}
            />
          </Form.Item>
        </Col>
      </Row>

      <Row gutter={16}>
        <Col xs={24} md={12}>
          <Form.Item name="requests_cpu" label="CPU Request">
            <Input placeholder="100m" />
          </Form.Item>
        </Col>
        <Col xs={24} md={12}>
          <Form.Item name="limits_cpu" label="CPU Limit">
            <Input placeholder="500m" />
          </Form.Item>
        </Col>
      </Row>
      <Row gutter={16}>
        <Col xs={24} md={12}>
          <Form.Item name="requests_memory" label="Memory Request">
            <Input placeholder="128Mi" />
          </Form.Item>
        </Col>
        <Col xs={24} md={12}>
          <Form.Item name="limits_memory" label="Memory Limit">
            <Input placeholder="512Mi" />
          </Form.Item>
        </Col>
      </Row>

      <Divider orientation="left" style={{ marginTop: 0 }}>容忍（Tolerations）</Divider>
      <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
        容忍本身是合理的，用来让 Pod 可以调度到带 `taint` 的节点上。
        `tolerationSeconds` 仅在 `effect=NoExecute` 时有意义，表示 Pod 被驱逐前还能继续保留多少秒；不填通常表示一直容忍。
      </Typography.Paragraph>
      <Form.List name="tolerations">
        {(fields, { add, remove }) => (
          <Space direction="vertical" style={{ width: "100%" }} size={12}>
            {fields.map((f) => (
              <Card key={f.key} size="small">
                <Row gutter={12}>
                  <Col xs={24} md={6}>
                    <Form.Item name={[f.name, "key"]} label="Key" style={{ marginBottom: 12 }}>
                      <Input placeholder="node-role.kubernetes.io/master" />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={4}>
                    <Form.Item name={[f.name, "operator"]} label="Operator" initialValue="Equal" style={{ marginBottom: 12 }}>
                      <Select
                        options={[
                          { label: "Equal", value: "Equal" },
                          { label: "Exists", value: "Exists" },
                        ]}
                      />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={5}>
                    <Form.Item name={[f.name, "value"]} label="Value" style={{ marginBottom: 12 }}>
                      <Input placeholder="true" />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={5}>
                    <Form.Item name={[f.name, "effect"]} label="Effect" style={{ marginBottom: 12 }}>
                      <Select
                        allowClear
                        options={[
                          { label: "NoSchedule", value: "NoSchedule" },
                          { label: "PreferNoSchedule", value: "PreferNoSchedule" },
                          { label: "NoExecute", value: "NoExecute" },
                        ]}
                      />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={4}>
                    <Form.Item name={[f.name, "toleration_seconds"]} label="持续秒数" tooltip="仅 NoExecute 时常用" style={{ marginBottom: 12 }}>
                      <InputNumber placeholder="3600" style={{ width: "100%" }} min={0} />
                    </Form.Item>
                  </Col>
                </Row>
                <Button onClick={() => remove(f.name)}>删除</Button>
              </Card>
            ))}
            <Button onClick={() => add({ key: "", operator: "Equal", value: "" })}>新增容忍</Button>
          </Space>
        )}
      </Form.List>

      <Divider orientation="left">卷（Volumes）</Divider>
      <Form.List name="volumes">
        {(fields, { add, remove }) => (
          <Space direction="vertical" style={{ width: "100%" }} size={12}>
            {fields.map((f) => (
              <Card key={f.key} size="small">
                <Row gutter={12}>
                  <Col xs={24} md={7}>
                    <Form.Item name={[f.name, "name"]} label="卷名" rules={[{ required: true, message: "卷名必填" }]} style={{ marginBottom: 12 }}>
                      <Input placeholder="config-volume" />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={5}>
                    <Form.Item name={[f.name, "type"]} label="类型" initialValue="emptyDir" style={{ marginBottom: 12 }}>
                      <Select
                        options={[
                          { label: "emptyDir", value: "emptyDir" },
                          { label: "configMap", value: "configMap" },
                          { label: "secret", value: "secret" },
                          { label: "pvc", value: "pvc" },
                        ]}
                      />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={12}>
                    <Form.Item name={[f.name, "source_name"]} label="来源名称" tooltip="configMap / secret / pvc 时填写" style={{ marginBottom: 12 }}>
                      <Input placeholder="source name (cm/secret/pvc)" />
                    </Form.Item>
                  </Col>
                </Row>
                <Button onClick={() => remove(f.name)}>删除</Button>
              </Card>
            ))}
            <Button onClick={() => add({ name: "", type: "emptyDir", source_name: "" })}>新增卷</Button>
          </Space>
        )}
      </Form.List>

      <Divider orientation="left">卷挂载（VolumeMounts）</Divider>
      <Form.List name="volume_mounts">
        {(fields, { add, remove }) => (
          <Space direction="vertical" style={{ width: "100%" }} size={12}>
            {fields.map((f) => (
              <Card key={f.key} size="small">
                <Row gutter={12}>
                  <Col xs={24} md={6}>
                    <Form.Item name={[f.name, "name"]} label="卷名" rules={[{ required: true, message: "卷名必填" }]} style={{ marginBottom: 12 }}>
                      <Input placeholder="volume name" />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={8}>
                    <Form.Item name={[f.name, "mount_path"]} label="挂载路径" rules={[{ required: true, message: "挂载路径必填" }]} style={{ marginBottom: 12 }}>
                      <Input placeholder="/data" />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={5}>
                    <Form.Item name={[f.name, "sub_path"]} label="SubPath" style={{ marginBottom: 12 }}>
                      <Input placeholder="subPath" />
                    </Form.Item>
                  </Col>
                  <Col xs={24} md={5}>
                    <Form.Item name={[f.name, "read_only"]} label="权限" initialValue={false} style={{ marginBottom: 12 }}>
                      <Select
                        options={[
                          { label: "读写", value: false },
                          { label: "只读", value: true },
                        ]}
                      />
                    </Form.Item>
                  </Col>
                </Row>
                <Button onClick={() => remove(f.name)}>删除</Button>
              </Card>
            ))}
            <Button onClick={() => add({ name: "", mount_path: "", read_only: false })}>新增挂载</Button>
          </Space>
        )}
      </Form.List>
    </Card>
  );
}

export function WorkloadPolicyItems(opts?: {
  showDeployStrategy?: boolean;
  showStatefulSetStrategy?: boolean;
  showDaemonSetStrategy?: boolean;
  showCronJobPolicy?: boolean;
  showJobPolicy?: boolean;
}) {
  return (
    <Card size="small" title="发布与调度策略" styles={{ body: { paddingBottom: 8 } }}>
      <Form.Item label="NodeSelector">
        <EnvPairsFormItem name="node_selector_pairs" />
      </Form.Item>
      <Form.Item name="affinity_yaml" label="Affinity YAML（可选）">
        <Input.TextArea rows={4} placeholder={"nodeAffinity:\n  requiredDuringSchedulingIgnoredDuringExecution:\n    nodeSelectorTerms: []"} />
      </Form.Item>
      {opts?.showDeployStrategy ? (
        <Row gutter={16}>
          <Col xs={24} md={7}>
            <Form.Item name="strategy_type" label="部署策略">
              <Select allowClear options={[{ label: "RollingUpdate", value: "RollingUpdate" }, { label: "Recreate", value: "Recreate" }]} />
            </Form.Item>
          </Col>
          <Col xs={24} md={5}>
            <Form.Item name="rolling_update_max_surge" label="maxSurge">
              <Input placeholder="25% / 1" />
            </Form.Item>
          </Col>
          <Col xs={24} md={5}>
            <Form.Item name="rolling_update_max_unavailable" label="maxUnavailable">
              <Input placeholder="25% / 0" />
            </Form.Item>
          </Col>
          <Col xs={24} md={7}>
            <Form.Item name="revision_history_limit" label="历史版本数">
              <InputNumber min={0} style={{ width: "100%" }} />
            </Form.Item>
          </Col>
        </Row>
      ) : null}
      {opts?.showStatefulSetStrategy ? (
        <Row gutter={16}>
          <Col xs={24} md={8}>
            <Form.Item name="update_strategy_type" label="StatefulSet UpdateStrategy">
              <Select allowClear options={[{ label: "RollingUpdate", value: "RollingUpdate" }, { label: "OnDelete", value: "OnDelete" }]} />
            </Form.Item>
          </Col>
          <Col xs={24} md={8}>
            <Form.Item name="rolling_update_partition" label="rollingUpdate.partition">
              <InputNumber min={0} style={{ width: "100%" }} />
            </Form.Item>
          </Col>
          <Col xs={24} md={8}>
            <Form.Item name="revision_history_limit" label="revisionHistoryLimit">
              <InputNumber min={0} style={{ width: "100%" }} />
            </Form.Item>
          </Col>
        </Row>
      ) : null}
      {opts?.showDaemonSetStrategy ? (
        <Row gutter={16}>
          <Col xs={24} md={7}>
            <Form.Item name="update_strategy_type" label="更新策略">
              <Select allowClear options={[{ label: "RollingUpdate", value: "RollingUpdate" }, { label: "OnDelete", value: "OnDelete" }]} />
            </Form.Item>
          </Col>
          <Col xs={24} md={5}>
            <Form.Item name="rolling_update_max_surge" label="maxSurge">
              <Input placeholder="25% / 1" />
            </Form.Item>
          </Col>
          <Col xs={24} md={5}>
            <Form.Item name="rolling_update_max_unavailable" label="maxUnavailable">
              <Input placeholder="25% / 0" />
            </Form.Item>
          </Col>
          <Col xs={24} md={7}>
            <Form.Item name="revision_history_limit" label="历史版本数">
              <InputNumber min={0} style={{ width: "100%" }} />
            </Form.Item>
          </Col>
        </Row>
      ) : null}
      {opts?.showCronJobPolicy ? (
        <Row gutter={16}>
          <Col xs={24} md={6}><Form.Item name="concurrency_policy" label="concurrencyPolicy"><Select allowClear options={[{ label: "Allow", value: "Allow" }, { label: "Forbid", value: "Forbid" }, { label: "Replace", value: "Replace" }]} /></Form.Item></Col>
          <Col xs={24} md={6}><Form.Item name="successful_jobs_history_limit" label="successfulJobsHistoryLimit"><InputNumber min={0} style={{ width: "100%" }} /></Form.Item></Col>
          <Col xs={24} md={6}><Form.Item name="failed_jobs_history_limit" label="failedJobsHistoryLimit"><InputNumber min={0} style={{ width: "100%" }} /></Form.Item></Col>
          <Col xs={24} md={6}><Form.Item name="starting_deadline_seconds" label="startingDeadlineSeconds"><InputNumber min={0} style={{ width: "100%" }} /></Form.Item></Col>
        </Row>
      ) : null}
      {opts?.showCronJobPolicy || opts?.showJobPolicy ? (
        <Row gutter={16}>
          <Col xs={24} md={5}><Form.Item name="parallelism" label="parallelism"><InputNumber min={0} style={{ width: "100%" }} /></Form.Item></Col>
          <Col xs={24} md={5}><Form.Item name="completions" label="completions"><InputNumber min={0} style={{ width: "100%" }} /></Form.Item></Col>
          <Col xs={24} md={5}><Form.Item name="backoff_limit" label="backoffLimit"><InputNumber min={0} style={{ width: "100%" }} /></Form.Item></Col>
          <Col xs={24} md={5}><Form.Item name="active_deadline_seconds" label="activeDeadlineSeconds"><InputNumber min={0} style={{ width: "100%" }} /></Form.Item></Col>
          <Col xs={24} md={4}><Form.Item name="ttl_seconds_after_finished" label="ttlSecondsAfterFinished"><InputNumber min={0} style={{ width: "100%" }} /></Form.Item></Col>
        </Row>
      ) : null}
    </Card>
  );
}

export function DeploymentHealthAndImagePullSecretsItems() {
  return (
    <Card size="small" title="探针与镜像拉取" styles={{ body: { paddingBottom: 8 } }}>
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

      <Divider orientation="left">Liveness Probe（存活探针）</Divider>
      <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
        K8s 探测动作有 `httpGet`、`tcpSocket`、`exec` 三种（与社区/源码一致）。`exec` 需要填写 JSON 数组形式的命令。
      </Typography.Paragraph>
      <Space style={{ width: "100%" }} align="start">
        <Form.Item name="liveness_probe_type" label="探针类型" style={{ width: 220 }}>
          <Select
            allowClear
            options={[
              { label: "httpGet", value: "httpGet" },
              { label: "tcpSocket", value: "tcpSocket" },
              { label: "exec", value: "exec" },
            ]}
          />
        </Form.Item>
        <Form.Item name="liveness_http_path" label="HTTP path" style={{ flex: 1 }}>
          <Input placeholder="/health" />
        </Form.Item>
      </Space>
      <Form.Item name="liveness_exec_command" label="Exec 命令" extra='例如：["sh","-c","test -f /tmp/ready"]'>
        <Input placeholder='例如：["sh","-c","curl -sf http://127.0.0.1:8080/health"]' />
      </Form.Item>
      <Space style={{ width: "100%" }} align="start">
        <Form.Item name="liveness_http_port" label="HTTP port" style={{ width: 180 }} extra="数字或端口名">
          <Input placeholder="8080 或 http" />
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
        <Form.Item name="liveness_tcp_port" label="TCP port" style={{ width: 180 }} extra="数字或端口名">
          <Input placeholder="8080 或 http" />
        </Form.Item>
      </Space>
      <Row gutter={16}>
        <Col xs={24} md={12}>
          <Form.Item name="liveness_initial_delay_seconds" label="initialDelaySeconds（首次探测延迟秒）">
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
        </Col>
        <Col xs={24} md={12}>
          <Form.Item name="liveness_period_seconds" label="periodSeconds（每次探测间隔秒）">
            <InputNumber min={1} style={{ width: "100%" }} />
          </Form.Item>
        </Col>
      </Row>
      <Row gutter={16}>
        <Col xs={24} md={12}>
          <Form.Item name="liveness_timeout_seconds" label="timeoutSeconds（单次超时秒）">
            <InputNumber min={1} style={{ width: "100%" }} />
          </Form.Item>
        </Col>
        <Col xs={24} md={12}>
          <Form.Item name="liveness_failure_threshold" label="failureThreshold（连续失败次数）">
            <InputNumber min={1} style={{ width: "100%" }} />
          </Form.Item>
        </Col>
      </Row>
      <Row gutter={16}>
        <Col xs={24} md={12}>
          <Form.Item name="liveness_success_threshold" label="successThreshold（连续成功次数）">
            <InputNumber min={1} style={{ width: "100%" }} />
          </Form.Item>
        </Col>
      </Row>

      <Divider orientation="left">Readiness Probe（就绪探针）</Divider>
      <Space style={{ width: "100%" }} align="start">
        <Form.Item name="readiness_probe_type" label="探针类型" style={{ width: 220 }}>
          <Select
            allowClear
            options={[
              { label: "httpGet", value: "httpGet" },
              { label: "tcpSocket", value: "tcpSocket" },
              { label: "exec", value: "exec" },
            ]}
          />
        </Form.Item>
        <Form.Item name="readiness_http_path" label="HTTP path" style={{ flex: 1 }}>
          <Input placeholder="/ready" />
        </Form.Item>
      </Space>
      <Form.Item name="readiness_exec_command" label="Exec 命令" extra='例如：["sh","-c","test -f /tmp/ready"]'>
        <Input placeholder='例如：["sh","-c","cat /tmp/ready"]' />
      </Form.Item>
      <Space style={{ width: "100%" }} align="start">
        <Form.Item name="readiness_http_port" label="HTTP port" style={{ width: 180 }} extra="数字或端口名">
          <Input placeholder="8080 或 http" />
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
        <Form.Item name="readiness_tcp_port" label="TCP port" style={{ width: 180 }} extra="数字或端口名">
          <Input placeholder="8080 或 http" />
        </Form.Item>
      </Space>
      <Row gutter={16}>
        <Col xs={24} md={12}>
          <Form.Item name="readiness_initial_delay_seconds" label="initialDelaySeconds（首次探测延迟秒）">
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
        </Col>
        <Col xs={24} md={12}>
          <Form.Item name="readiness_period_seconds" label="periodSeconds（每次探测间隔秒）">
            <InputNumber min={1} style={{ width: "100%" }} />
          </Form.Item>
        </Col>
      </Row>
      <Row gutter={16}>
        <Col xs={24} md={12}>
          <Form.Item name="readiness_timeout_seconds" label="timeoutSeconds（单次超时秒）">
            <InputNumber min={1} style={{ width: "100%" }} />
          </Form.Item>
        </Col>
        <Col xs={24} md={12}>
          <Form.Item name="readiness_failure_threshold" label="failureThreshold（连续失败次数）">
            <InputNumber min={1} style={{ width: "100%" }} />
          </Form.Item>
        </Col>
      </Row>
      <Row gutter={16}>
        <Col xs={24} md={12}>
          <Form.Item name="readiness_success_threshold" label="successThreshold（连续成功次数）">
            <InputNumber min={1} style={{ width: "100%" }} />
          </Form.Item>
        </Col>
      </Row>

      <Divider orientation="left">Startup Probe（启动探针）</Divider>
      <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
        启动探针通常用于启动慢的容器，避免应用还没真正启动完成时就被 liveness 误杀。
        当前先开放基础参数输入，后续我会把它连到完整的 YAML 构建与回填。
      </Typography.Paragraph>
      <Space style={{ width: "100%" }} align="start">
        <Form.Item name="startup_probe_type" label="探针类型" style={{ width: 220 }}>
          <Select
            allowClear
            options={[
              { label: "httpGet", value: "httpGet" },
              { label: "tcpSocket", value: "tcpSocket" },
              { label: "exec", value: "exec" },
            ]}
          />
        </Form.Item>
        <Form.Item name="startup_http_path" label="HTTP path" style={{ flex: 1 }}>
          <Input placeholder="/startup" />
        </Form.Item>
      </Space>
      <Space style={{ width: "100%" }} align="start">
        <Form.Item name="startup_http_port" label="HTTP port" style={{ width: 180 }} extra="数字或端口名">
          <Input placeholder="8080 或 http" />
        </Form.Item>
        <Form.Item name="startup_http_scheme" label="HTTP scheme" style={{ width: 180 }}>
          <Select allowClear options={[{ label: "HTTP", value: "HTTP" }, { label: "HTTPS", value: "HTTPS" }]} />
        </Form.Item>
        <Form.Item name="startup_tcp_port" label="TCP port" style={{ width: 180 }} extra="数字或端口名">
          <Input placeholder="8080 或 http" />
        </Form.Item>
      </Space>
      <Form.Item name="startup_exec_command" label="Exec 命令" extra='例如：["sh","-c","test -f /tmp/ready"]'>
        <Input placeholder='例如：["sh","-c","cat /tmp/ready"]' />
      </Form.Item>
      <Row gutter={16}>
        <Col xs={24} md={12}>
          <Form.Item name="startup_initial_delay_seconds" label="initialDelaySeconds（首次探测延迟秒）">
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
        </Col>
        <Col xs={24} md={12}>
          <Form.Item name="startup_period_seconds" label="periodSeconds（每次探测间隔秒）">
            <InputNumber min={1} style={{ width: "100%" }} />
          </Form.Item>
        </Col>
      </Row>
      <Row gutter={16}>
        <Col xs={24} md={12}>
          <Form.Item name="startup_timeout_seconds" label="timeoutSeconds（单次超时秒）">
            <InputNumber min={1} style={{ width: "100%" }} />
          </Form.Item>
        </Col>
        <Col xs={24} md={12}>
          <Form.Item name="startup_failure_threshold" label="failureThreshold（连续失败次数）">
            <InputNumber min={1} style={{ width: "100%" }} />
          </Form.Item>
        </Col>
      </Row>
      <Row gutter={16}>
        <Col xs={24} md={12}>
          <Form.Item name="startup_success_threshold" label="successThreshold（连续成功次数）">
            <InputNumber min={1} style={{ width: "100%" }} />
          </Form.Item>
        </Col>
      </Row>
    </Card>
  );
}

