import { CodeOutlined, DeleteOutlined, DownloadOutlined, EditOutlined, EyeOutlined, FileSearchOutlined, FileTextOutlined, FolderOpenOutlined, PlusOutlined, ReloadOutlined, UndoOutlined, UploadOutlined } from "@ant-design/icons";
import { Button, Card, Divider, Drawer, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Table, Tag, Tabs, Typography, message } from "antd";
import { useEffect, useMemo, useRef, useState } from "react";
import { Terminal } from "xterm";
import { FitAddon } from "xterm-addon-fit";
import "xterm/css/xterm.css";
import { formatDateTime } from "../utils/format";
import { getClusters, listNamespaces, type ClusterItem, type NamespaceItem } from "../services/clusters";
import { createPodByYAML, createPodSimple, deletePod, deletePodFile, downloadPodFile, downloadPodLogs, getPodDetail, getPodEvents, getPodLogs, getPods, listPodFiles, readPodFile, restartPod, updatePodSimple, uploadPodFile, type PodDetail, type PodEventItem, type PodFileItem, type PodItem } from "../services/pods";
import { getToken } from "../services/storage";

function phaseColor(phase: string): string {
  const p = (phase || "").toLowerCase();
  if (p === "running") return "green";
  if (p === "pending") return "orange";
  if (p === "failed") return "red";
  if (p === "succeeded") return "blue";
  return "default";
}

export function PodPage() {
  const rfc1123Subdomain = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$/;
  const rfc1123Label = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/;
  const [clusters, setClusters] = useState<ClusterItem[]>([]);
  const [clusterId, setClusterId] = useState<number>();
  const [namespaces, setNamespaces] = useState<NamespaceItem[]>([]);
  const [namespace, setNamespace] = useState("default");
  const [keyword, setKeyword] = useState("");
  const [loading, setLoading] = useState(false);
  const [pods, setPods] = useState<PodItem[]>([]);

  const [selected, setSelected] = useState<PodItem | null>(null);
  const [detail, setDetail] = useState<PodDetail | null>(null);
  const [events, setEvents] = useState<PodEventItem[]>([]);

  const [logsOpen, setLogsOpen] = useState(false);
  const [logsLoading, setLogsLoading] = useState(false);
  const [logsText, setLogsText] = useState("");
  const [logsTitle, setLogsTitle] = useState("");
  const [logsKeyword, setLogsKeyword] = useState("");
  const [logsStartTime, setLogsStartTime] = useState("");
  const [logsEndTime, setLogsEndTime] = useState("");
  const streamAbortRef = useRef<AbortController | null>(null);
  const [streaming, setStreaming] = useState(false);
  const [execOpen, setExecOpen] = useState(false);
  const [fileOpen, setFileOpen] = useState(false);
  const [filePath, setFilePath] = useState("/");
  const [fileList, setFileList] = useState<PodFileItem[]>([]);
  const [fileLoading, setFileLoading] = useState(false);
  const [fileContent, setFileContent] = useState("");
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const [detailOpen, setDetailOpen] = useState(false);
  const [execCommand, setExecCommand] = useState("sh");
  const execTermHostRef = useRef<HTMLDivElement | null>(null);
  const execTermRef = useRef<Terminal | null>(null);
  const execFitRef = useRef<FitAddon | null>(null);
  const execWsRef = useRef<WebSocket | null>(null);
  const [createOpen, setCreateOpen] = useState(false);
  const [creating, setCreating] = useState(false);
  const [simpleMode, setSimpleMode] = useState<"create" | "edit">("create");
  const [editTarget, setEditTarget] = useState<PodItem | null>(null);
  const [simpleForm] = Form.useForm<{
    name: string;
    image: string;
    command?: string;
    container_name?: string;
    image_pull_policy?: "Always" | "IfNotPresent" | "Never";
    restart_policy?: "Always" | "OnFailure" | "Never";
    port?: number;
    env_pairs?: Array<{ key?: string; value?: string }>;
    label_pairs?: Array<{ key?: string; value?: string }>;
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
    node_selector_pairs?: Array<{ key?: string; value?: string }>;
    priority_class_name?: string;
    affinity?: {
      node?: {
        required?: Array<{
          match_expressions?: Array<{
            key?: string;
            operator?: "In" | "NotIn" | "Exists" | "DoesNotExist" | "Gt" | "Lt";
            values?: string[];
          }>;
        }>;
        preferred?: Array<{
          weight?: number;
          match_expressions?: Array<{
            key?: string;
            operator?: "In" | "NotIn" | "Exists" | "DoesNotExist" | "Gt" | "Lt";
            values?: string[];
          }>;
        }>;
      };
      pod?: {
        required?: Array<{
          match_labels?: Array<{ key?: string; value?: string }>;
          topology_key?: string;
        }>;
        preferred?: Array<{
          weight?: number;
          match_labels?: Array<{ key?: string; value?: string }>;
          topology_key?: string;
        }>;
      };
      pod_anti?: {
        required?: Array<{
          match_labels?: Array<{ key?: string; value?: string }>;
          topology_key?: string;
        }>;
        preferred?: Array<{
          weight?: number;
          match_labels?: Array<{ key?: string; value?: string }>;
          topology_key?: string;
        }>;
      };
    };
  }>();
  const [yamlForm] = Form.useForm<{ manifest: string }>();

  async function loadNamespaces(id: number) {
    const res = await listNamespaces(id);
    const ns = res.list || [];
    setNamespaces(ns);
    const hasCurrent = ns.some((item) => item.name === namespace);
    if (!hasCurrent) {
      setNamespace(ns[0]?.name || "default");
    }
  }

  async function loadPods(overrideKeyword?: string) {
    if (!clusterId) {
      setPods([]);
      return;
    }
    setLoading(true);
    try {
      const effectiveKeyword = (overrideKeyword ?? keyword).trim();
      const res = await getPods({ cluster_id: clusterId, namespace, keyword: effectiveKeyword || undefined });
      setPods(res.list || []);
    } finally {
      setLoading(false);
    }
  }

  async function loadClusters() {
    const res = await getClusters({ page: 1, page_size: 200 });
    setClusters(res.list || []);
    if (!clusterId && res.list?.length) setClusterId(res.list[0].id);
  }

  useEffect(() => { void loadClusters(); }, []);

  useEffect(() => {
    if (!clusterId) return;
    void loadNamespaces(clusterId);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [clusterId]);

  useEffect(() => {
    void loadPods();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [clusterId, namespace]);

  useEffect(() => {
    if (!clusterId) return;
    const timer = window.setInterval(() => {
      void loadPods();
    }, 10000);
    return () => window.clearInterval(timer);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [clusterId, namespace]);

  const namespaceOptions = useMemo(() => namespaces.map((n) => ({ label: n.name, value: n.name })), [namespaces]);
  const clusterOptions = useMemo(
    () => clusters.map((c) => ({ label: c.name, value: c.id })),
    [clusters],
  );

  async function handleDeletePod(record: PodItem) {
    if (!clusterId) return;
    await deletePod({ cluster_id: clusterId, namespace: record.namespace, name: record.name });
    message.success("Pod 已删除");
    await loadPods();
  }

  async function handleViewLogs(record: PodItem) {
    if (!clusterId) return;
    setLogsOpen(true);
    setLogsLoading(true);
    setLogsText("");
    setLogsTitle(`${record.namespace}/${record.name}`);
    setLogsKeyword("");
    setLogsStartTime("");
    setLogsEndTime("");
    try {
      const res = await getPodLogs({ cluster_id: clusterId, namespace: record.namespace, name: record.name, tail_lines: 500 });
      setLogsText(res.logs || "");
    } finally {
      setLogsLoading(false);
    }
  }

  async function handleFilterLogs() {
    if (!clusterId || !selected) return;
    setLogsLoading(true);
    try {
      const res = await getPodLogs({
        cluster_id: clusterId,
        namespace: selected.namespace,
        name: selected.name,
        tail_lines: 1000,
        keyword: logsKeyword || undefined,
        start_time: logsStartTime || undefined,
        end_time: logsEndTime || undefined,
      });
      setLogsText(res.logs || "");
    } finally {
      setLogsLoading(false);
    }
  }

  async function handleDownloadLogs() {
    if (!clusterId || !selected) return;
    const blob = await downloadPodLogs({
      cluster_id: clusterId,
      namespace: selected.namespace,
      name: selected.name,
      tail_lines: 2000,
      keyword: logsKeyword || undefined,
      start_time: logsStartTime || undefined,
      end_time: logsEndTime || undefined,
    });
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `${selected.namespace}-${selected.name}.log`;
    document.body.appendChild(a);
    a.click();
    a.remove();
    window.URL.revokeObjectURL(url);
  }

  async function startLogStream() {
    if (!clusterId || !selected) return;
    stopLogStream();
    const aborter = new AbortController();
    streamAbortRef.current = aborter;
    setStreaming(true);
    const token = getToken();
    const params = new URLSearchParams({
      cluster_id: String(clusterId),
      namespace: selected.namespace,
      name: selected.name,
      tail_lines: "50",
    });
    try {
      const resp = await fetch(`/api/v1/pods/logs/stream?${params.toString()}`, {
        headers: { Authorization: token ? `Bearer ${token}` : "" },
        signal: aborter.signal,
      });
      if (!resp.ok || !resp.body) throw new Error("日志流连接失败");
      const reader = resp.body.getReader();
      const decoder = new TextDecoder("utf-8");
      let buffer = "";
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });
        const lines = buffer.split("\n");
        buffer = lines.pop() || "";
        for (const line of lines) {
          if (line.startsWith("data: ")) setLogsText((prev) => `${prev}${line.slice(6)}\n`);
        }
      }
    } catch (error) {
      if ((error as Error).name !== "AbortError") message.error((error as Error).message || "日志流断开");
    } finally {
      setStreaming(false);
    }
  }

  function stopLogStream() {
    streamAbortRef.current?.abort();
    streamAbortRef.current = null;
    setStreaming(false);
  }

  async function loadDetail(record: PodItem) {
    if (!clusterId) return;
    setSelected(record);
    const [d, e] = await Promise.all([
      getPodDetail({ cluster_id: clusterId, namespace: record.namespace, name: record.name }),
      getPodEvents({ cluster_id: clusterId, namespace: record.namespace, name: record.name }),
    ]);
    setDetail(d);
    setEvents(e.list || []);
  }

  function closeExecSocket() {
    try {
      execWsRef.current?.close();
    } catch {
      // ignore
    }
    execWsRef.current = null;
  }

  useEffect(() => {
    if (!execOpen) return;
    if (!clusterId || !selected) return;
    const host = execTermHostRef.current;
    if (!host) return;

    // reset container
    host.innerHTML = "";

    const term = new Terminal({
      cursorBlink: true,
      fontFamily:
        'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
      fontSize: 12,
      theme: { background: "#0f172a" },
      scrollback: 5000,
    });
    const fit = new FitAddon();
    term.loadAddon(fit);
    term.open(host);
    fit.fit();
    term.focus();

    execTermRef.current = term;
    execFitRef.current = fit;

    const token = getToken();
    const qs = new URLSearchParams({
      cluster_id: String(clusterId),
        namespace: selected.namespace,
        name: selected.name,
      token: token || "",
    });
    const proto = window.location.protocol === "https:" ? "wss" : "ws";
    const ws = new WebSocket(`${proto}://${window.location.host}/api/v1/pods/exec/ws?${qs.toString()}`);
    execWsRef.current = ws;

    const sendResize = () => {
      try {
        const cols = term.cols;
        const rows = term.rows;
        ws.readyState === WebSocket.OPEN &&
          ws.send(JSON.stringify({ type: "resize", cols, rows }));
      } catch {
        // ignore
      }
    };

    const onDataDispose = term.onData((data) => {
      if (ws.readyState !== WebSocket.OPEN) return;
      ws.send(JSON.stringify({ type: "input", data }));
    });
    const onResizeDispose = term.onResize(({ cols, rows }) => {
      if (ws.readyState !== WebSocket.OPEN) return;
      ws.send(JSON.stringify({ type: "resize", cols, rows }));
    });

    ws.onopen = () => {
      term.writeln(`Connected: ${selected.namespace}/${selected.name}`);
      sendResize();
      // start shell
      ws.send(JSON.stringify({ type: "input", data: `${execCommand}\n` }));
    };
    ws.onmessage = (ev) => {
      // server sends JSON frames: stdout/error/exit/ready
      try {
        const msg = JSON.parse(String(ev.data));
        if (msg.type === "stdout" && typeof msg.data === "string") {
          term.write(msg.data);
        } else if (msg.type === "error") {
          term.writeln(`\r\n[error] ${msg.data || "unknown"}`);
        } else if (msg.type === "exit") {
          term.writeln("\r\n[disconnected]");
        }
      } catch {
        term.write(String(ev.data));
      }
    };
    ws.onclose = () => {
      term.writeln("\r\n[connection closed]");
    };
    ws.onerror = () => {
      term.writeln("\r\n[connection error]");
    };

    const onWindowResize = () => {
      try {
        fit.fit();
        sendResize();
      } catch {
        // ignore
      }
    };
    window.addEventListener("resize", onWindowResize);

    return () => {
      window.removeEventListener("resize", onWindowResize);
      try {
        onDataDispose.dispose();
        onResizeDispose.dispose();
      } catch {
        // ignore
      }
      closeExecSocket();
      try {
        term.dispose();
      } catch {
        // ignore
      }
      execTermRef.current = null;
      execFitRef.current = null;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [execOpen, clusterId, selected?.namespace, selected?.name]);

  async function handleRestartPod(record: PodItem) {
    if (!clusterId) return;
    await restartPod({ cluster_id: clusterId, namespace: record.namespace, name: record.name });
    message.success("Pod 已重启");
    await loadPods();
  }

  async function loadFiles(target: PodItem, path: string) {
    if (!clusterId) return;
    setFileLoading(true);
    try {
      const res = await listPodFiles({
        cluster_id: clusterId,
        namespace: target.namespace,
        name: target.name,
        path: path || "/",
      });
      setFileList(res.list || []);
      setFilePath(path || "/");
    } finally {
      setFileLoading(false);
    }
  }

  async function submitCreateSimple() {
    if (!clusterId) return;
    const values = await simpleForm.validateFields();
    const env: Record<string, string> = {};
    const labels: Record<string, string> = {};
    const nodeSelector: Record<string, string> = {};
    (values.env_pairs || []).forEach((item) => {
      const k = (item?.key || "").trim();
      const v = (item?.value || "").trim();
      if (k) env[k] = v;
    });
    (values.label_pairs || []).forEach((item) => {
      const k = (item?.key || "").trim();
      const v = (item?.value || "").trim();
      if (k) labels[k] = v;
    });
    (values.node_selector_pairs || []).forEach((item) => {
      const k = (item?.key || "").trim();
      const v = (item?.value || "").trim();
      if (k) nodeSelector[k] = v;
    });
    let affinityObj: Record<string, unknown> | undefined;
    if (values.affinity) {
      const a = values.affinity;
      const nodeRequiredTerms =
        a.node?.required
          ?.map((t) => ({
            matchExpressions: (t.match_expressions || [])
              .map((e) => ({
                key: (e.key || "").trim(),
                operator: e.operator,
                values: (e.values || []).map((v) => String(v).trim()).filter(Boolean),
              }))
              .filter((e) => e.key && e.operator),
          }))
          .filter((t) => t.matchExpressions.length > 0) || [];

      const nodePreferred =
        a.node?.preferred
          ?.map((p) => ({
            weight: Math.min(100, Math.max(1, Number(p.weight || 1))),
            preference: {
              matchExpressions: (p.match_expressions || [])
                .map((e) => ({
                  key: (e.key || "").trim(),
                  operator: e.operator,
                  values: (e.values || []).map((v) => String(v).trim()).filter(Boolean),
                }))
                .filter((e) => e.key && e.operator),
            },
          }))
          .filter((p) => p.preference.matchExpressions.length > 0) || [];

      const buildPodAffinityTerms = (
        list?: Array<{ match_labels?: Array<{ key?: string; value?: string }>; topology_key?: string }>,
      ) =>
        (list || [])
          .map((it) => {
            const labels = (it.match_labels || [])
              .map((kv) => ({ key: (kv.key || "").trim(), value: (kv.value || "").trim() }))
              .filter((kv) => kv.key);
            const matchLabels: Record<string, string> = {};
            labels.forEach((kv) => {
              matchLabels[kv.key] = kv.value;
            });
            const topologyKey = (it.topology_key || "").trim();
            if (!topologyKey || Object.keys(matchLabels).length === 0) return null;
            return {
              labelSelector: { matchLabels },
              topologyKey,
            };
          })
          .filter(Boolean);

      const buildPodPreferredTerms = (
        list?: Array<{ weight?: number; match_labels?: Array<{ key?: string; value?: string }>; topology_key?: string }>,
      ) =>
        (list || [])
          .map((it) => {
            const labels = (it.match_labels || [])
              .map((kv) => ({ key: (kv.key || "").trim(), value: (kv.value || "").trim() }))
              .filter((kv) => kv.key);
            const matchLabels: Record<string, string> = {};
            labels.forEach((kv) => {
              matchLabels[kv.key] = kv.value;
            });
            const topologyKey = (it.topology_key || "").trim();
            if (!topologyKey || Object.keys(matchLabels).length === 0) return null;
            return {
              weight: Math.min(100, Math.max(1, Number(it.weight || 1))),
              podAffinityTerm: {
                labelSelector: { matchLabels },
                topologyKey,
              },
            };
          })
          .filter(Boolean);

      const podRequired = buildPodAffinityTerms(a.pod?.required);
      const podPreferred = buildPodPreferredTerms(a.pod?.preferred);
      const podAntiRequired = buildPodAffinityTerms(a.pod_anti?.required);
      const podAntiPreferred = buildPodPreferredTerms(a.pod_anti?.preferred);

      const affinity: any = {};
      if (nodeRequiredTerms.length > 0 || nodePreferred.length > 0) {
        affinity.nodeAffinity = {};
        if (nodeRequiredTerms.length > 0) {
          affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution = {
            nodeSelectorTerms: nodeRequiredTerms,
          };
        }
        if (nodePreferred.length > 0) {
          affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution = nodePreferred;
        }
      }
      if ((podRequired as any[]).length > 0 || (podPreferred as any[]).length > 0) {
        affinity.podAffinity = {};
        if ((podRequired as any[]).length > 0) affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution = podRequired;
        if ((podPreferred as any[]).length > 0) affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution = podPreferred;
      }
      if ((podAntiRequired as any[]).length > 0 || (podAntiPreferred as any[]).length > 0) {
        affinity.podAntiAffinity = {};
        if ((podAntiRequired as any[]).length > 0) affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution = podAntiRequired;
        if ((podAntiPreferred as any[]).length > 0) affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution = podAntiPreferred;
      }
      if (Object.keys(affinity).length > 0) affinityObj = affinity;
    }
    setCreating(true);
    try {
      const payload = {
        cluster_id: clusterId,
        namespace,
        name: values.name,
        image: values.image,
        command: values.command,
        container_name: values.container_name,
        image_pull_policy: values.image_pull_policy,
        restart_policy: values.restart_policy,
        port: values.port,
        env: Object.keys(env).length > 0 ? env : undefined,
        labels: Object.keys(labels).length > 0 ? labels : undefined,
        requests_cpu: values.requests_cpu,
        requests_memory: values.requests_memory,
        limits_cpu: values.limits_cpu,
        limits_memory: values.limits_memory,
        node_selector: Object.keys(nodeSelector).length > 0 ? nodeSelector : undefined,
        priority_class_name: values.priority_class_name,
        affinity: affinityObj,
        tolerations: (values.tolerations || [])
          .filter((item) => (item.key || "").trim() !== "")
          .map((item) => ({
            key: (item.key || "").trim(),
            operator: item.operator || "Equal",
            value: (item.value || "").trim(),
            effect: item.effect,
            toleration_seconds: item.toleration_seconds,
          })),
      };
      if (simpleMode === "edit") {
        await updatePodSimple(payload);
        message.success("Pod 已更新并重建");
      } else {
        await createPodSimple(payload);
      message.success("Pod 创建成功");
      }
      setCreateOpen(false);
      await loadPods();
    } finally {
      setCreating(false);
    }
  }

  async function openEditPod(record: PodItem) {
    if (!clusterId) return;
    const d = await getPodDetail({ cluster_id: clusterId, namespace: record.namespace, name: record.name });
    setSimpleMode("edit");
    setEditTarget(record);
    setCreateOpen(true);
    simpleForm.setFieldsValue({
      name: d.name,
      container_name: d.containers?.[0]?.name || d.name,
      image: d.containers?.[0]?.image || "",
      command: "",
      image_pull_policy: "IfNotPresent",
      restart_policy: "Always",
      env_pairs: [],
      label_pairs: Object.entries(d.labels || {}).map(([key, value]) => ({ key, value })),
      node_selector_pairs: Object.entries(d.node_selector || {}).map(([key, value]) => ({ key, value })),
      priority_class_name: d.priority_class_name || "",
      affinity: (() => {
        const a: any = d.affinity || {};
        const out: any = {};
        if (a.nodeAffinity) {
          const na: any = {};
          const reqTerms = a.nodeAffinity?.requiredDuringSchedulingIgnoredDuringExecution?.nodeSelectorTerms || [];
          na.required = reqTerms.map((t: any) => ({
            match_expressions: (t.matchExpressions || []).map((e: any) => ({
              key: e.key,
              operator: e.operator,
              values: e.values || [],
            })),
          }));
          const pref = a.nodeAffinity?.preferredDuringSchedulingIgnoredDuringExecution || [];
          na.preferred = pref.map((p: any) => ({
            weight: p.weight,
            match_expressions: (p.preference?.matchExpressions || []).map((e: any) => ({
              key: e.key,
              operator: e.operator,
              values: e.values || [],
            })),
          }));
          out.node = na;
        }
        function parsePodTerms(list: any[]) {
          return (list || []).map((t: any) => {
            const ml = t.labelSelector?.matchLabels || {};
            return {
              topology_key: t.topologyKey,
              match_labels: Object.entries(ml).map(([key, value]) => ({ key, value })),
            };
          });
        }
        function parsePodPreferred(list: any[]) {
          return (list || []).map((p: any) => {
            const term = p.podAffinityTerm || {};
            const ml = term.labelSelector?.matchLabels || {};
            return {
              weight: p.weight,
              topology_key: term.topologyKey,
              match_labels: Object.entries(ml).map(([key, value]) => ({ key, value })),
            };
          });
        }
        if (a.podAffinity) {
          out.pod = {
            required: parsePodTerms(a.podAffinity.requiredDuringSchedulingIgnoredDuringExecution || []),
            preferred: parsePodPreferred(a.podAffinity.preferredDuringSchedulingIgnoredDuringExecution || []),
          };
        }
        if (a.podAntiAffinity) {
          out.pod_anti = {
            required: parsePodTerms(a.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution || []),
            preferred: parsePodPreferred(a.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution || []),
          };
        }
        return out;
      })(),
      tolerations: (d.tolerations || []).map((t) => ({
        key: t.key,
        operator: t.operator,
        value: t.value,
        effect: t.effect,
        toleration_seconds: t.tolerationSeconds,
      })),
    });
  }

  async function submitCreateYAML() {
    if (!clusterId) return;
    const values = await yamlForm.validateFields();
    setCreating(true);
    try {
      await createPodByYAML({ cluster_id: clusterId, namespace, manifest: values.manifest });
      message.success("YAML 创建成功");
      setCreateOpen(false);
      await loadPods();
    } finally {
      setCreating(false);
    }
  }

  return (
    <div>
      <div style={{ display: "flex", gap: 12, alignItems: "center", justifyContent: "space-between", marginBottom: 12 }}>
        <Space wrap>
        <Select
          placeholder="选择集群"
          style={{ minWidth: 240 }}
          value={clusterId}
          onChange={(v) => setClusterId(v)}
          options={clusterOptions}
        />
        <Select
          placeholder="命名空间"
          style={{ minWidth: 200 }}
          value={namespace}
          onChange={setNamespace}
          options={namespaceOptions}
        />
        <Input.Search
          allowClear
          placeholder="搜索 Pod 名称/节点"
          style={{ width: 260 }}
          onSearch={(v) => {
            setKeyword(v);
            void loadPods(v);
          }}
        />
        </Space>
        <Space>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => {
              setSimpleMode("create");
              setEditTarget(null);
              simpleForm.resetFields();
              yamlForm.resetFields();
              setCreateOpen(true);
            }}
          >
            创建 Pod
          </Button>
        <Button icon={<ReloadOutlined />} onClick={() => void loadPods()}>
          刷新
        </Button>
      </Space>
      </div>
      <Table
            rowKey={(r) => `${r.namespace}/${r.name}`}
            loading={loading}
            dataSource={pods}
            pagination={{ pageSize: 10 }}
            onRow={(record) => ({ onClick: () => void loadDetail(record) })}
            rowClassName={(record) => (selected && `${record.namespace}/${record.name}` === `${selected.namespace}/${selected.name}` ? "ant-table-row-selected" : "")}
            columns={[
              { title: "Pod 名称", dataIndex: "name" },
              { title: "命名空间", dataIndex: "namespace", width: 120 },
              { title: "节点", dataIndex: "node_name", width: 140 },
              { title: "PodIP", dataIndex: "pod_ip", width: 130 },
              { title: "QoS", dataIndex: "qos_class", width: 90 },
              { title: "重启", dataIndex: "restart_count", width: 70 },
              { title: "状态", dataIndex: "phase", width: 110, render: (phase: string) => <Tag color={phaseColor(phase)}>{phase || "-"}</Tag> },
              { title: "启动时间", dataIndex: "start_time", width: 170, render: (v: string) => formatDateTime(v) },
              {
                title: "操作",
                key: "action",
                width: 280,
                render: (_: unknown, record: PodItem) => (
                  <Space>
                    <Button type="link" icon={<EyeOutlined />} onClick={() => { void loadDetail(record); setDetailOpen(true); }}>
                      详情
                    </Button>
                    <Button type="link" icon={<FileTextOutlined />} onClick={() => void handleViewLogs(record)}>日志</Button>
                    <Button type="link" icon={<EditOutlined />} onClick={() => void openEditPod(record)}>高级编辑</Button>
                    <Button
                      type="link"
                      icon={<FolderOpenOutlined />}
                      onClick={() => {
                        setSelected(record);
                        setFileOpen(true);
                        setFileContent("");
                        void loadFiles(record, "/");
                      }}
                    >
                      文件
                    </Button>
                    <Button type="link" icon={<CodeOutlined />} onClick={() => { setSelected(record); setExecOpen(true); }}>Exec</Button>
                    <Button type="link" icon={<UndoOutlined />} onClick={() => void handleRestartPod(record)}>重启</Button>
                    <Popconfirm title="确认删除该 Pod 吗？" onConfirm={() => void handleDeletePod(record)}>
                      <Button type="link" danger icon={<DeleteOutlined />}>删除</Button>
                    </Popconfirm>
                  </Space>
                ),
              },
            ]}
      />

      {/* 日志查看对话框 */}
      <Modal
        title={`Pod 日志 - ${logsTitle}`}
        open={logsOpen}
        onCancel={() => {
          stopLogStream();
          setLogsOpen(false);
        }}
        footer={null}
        width={980}
      >
        <Space wrap style={{ marginBottom: 12 }}>
          <Input placeholder="关键字过滤" value={logsKeyword} onChange={(e) => setLogsKeyword(e.target.value)} style={{ width: 160 }} />
          <Input placeholder="开始时间 2026-01-02 15:04:05" value={logsStartTime} onChange={(e) => setLogsStartTime(e.target.value)} style={{ width: 210 }} />
          <Input placeholder="结束时间 2026-01-02 15:04:05" value={logsEndTime} onChange={(e) => setLogsEndTime(e.target.value)} style={{ width: 210 }} />
          <Button icon={<FileSearchOutlined />} onClick={() => void handleFilterLogs()}>过滤</Button>
          <Button icon={<DownloadOutlined />} onClick={() => void handleDownloadLogs()}>下载</Button>
          <Button type={streaming ? "default" : "primary"} onClick={() => void startLogStream()} disabled={streaming}>开始实时流</Button>
          <Button danger={streaming} onClick={stopLogStream} disabled={!streaming}>停止流</Button>
          <Button onClick={() => selected && void handleViewLogs(selected)}>获取当前日志</Button>
        </Space>
        {logsLoading ? (
          <Typography.Text>日志加载中...</Typography.Text>
        ) : (
          <pre
            style={{
              maxHeight: 520,
              overflow: "auto",
              background: "#0f172a",
              color: "#e2e8f0",
              padding: 12,
              borderRadius: 8,
              margin: 0,
              whiteSpace: "pre-wrap",
              wordBreak: "break-word",
            }}
          >
            {logsText || "暂无日志"}
          </pre>
        )}
      </Modal>
      <Drawer
        title={selected ? `Pod 文件管理 - ${selected.namespace}/${selected.name}` : "Pod 文件管理"}
        open={fileOpen}
        onClose={() => setFileOpen(false)}
        width={920}
      >
        <Space wrap style={{ marginBottom: 12 }}>
          <Input
            value={filePath}
            onChange={(e) => setFilePath(e.target.value)}
            placeholder="目录路径，例如 / /tmp /var/log"
            style={{ width: 360 }}
          />
          <Button onClick={() => selected && void loadFiles(selected, filePath)} icon={<ReloadOutlined />}>
            刷新目录
          </Button>
          <Button
            icon={<UploadOutlined />}
            onClick={() => fileInputRef.current?.click()}
            disabled={!selected}
          >
            上传到当前目录
          </Button>
          <input
            ref={fileInputRef}
            type="file"
            style={{ display: "none" }}
            onChange={(e) => {
              const f = e.target.files?.[0];
              if (!f || !selected || !clusterId) return;
              void (async () => {
                await uploadPodFile({
                  cluster_id: clusterId,
                  namespace: selected.namespace,
                  name: selected.name,
                  path: filePath || "/",
                  file: f,
                });
                message.success("上传成功");
                await loadFiles(selected, filePath);
              })();
              e.currentTarget.value = "";
            }}
          />
        </Space>
        <Table
          rowKey={(r) => r.path}
          loading={fileLoading}
          dataSource={fileList}
          size="small"
          pagination={{ pageSize: 8 }}
          columns={[
            { title: "名称", dataIndex: "name" },
            { title: "类型", dataIndex: "type", width: 100 },
            { title: "大小", dataIndex: "size", width: 110 },
            { title: "权限", dataIndex: "permissions", width: 120 },
            { title: "修改时间", dataIndex: "mod_time", width: 140 },
            {
              title: "操作",
              width: 280,
              render: (_: unknown, row: PodFileItem) => (
                <Space>
                  {row.is_dir ? (
                    <Button type="link" onClick={() => selected && void loadFiles(selected, row.path)}>
                      进入
                    </Button>
                  ) : (
                    <>
                      <Button
                        type="link"
                        onClick={() => {
                          if (!selected || !clusterId) return;
                          void (async () => {
                            const res = await readPodFile({
                              cluster_id: clusterId,
                              namespace: selected.namespace,
                              name: selected.name,
                              path: row.path,
                            });
                            setFileContent(res.content || "");
                          })();
                        }}
                      >
                        查看
                      </Button>
                      <Button
                        type="link"
                        icon={<DownloadOutlined />}
                        onClick={() => {
                          if (!selected || !clusterId) return;
                          void (async () => {
                            const blob = await downloadPodFile({
                              cluster_id: clusterId,
                              namespace: selected.namespace,
                              name: selected.name,
                              path: row.path,
                            });
                            const url = window.URL.createObjectURL(blob);
                            const a = document.createElement("a");
                            a.href = url;
                            a.download = row.name;
                            document.body.appendChild(a);
                            a.click();
                            a.remove();
                            window.URL.revokeObjectURL(url);
                          })();
                        }}
                      >
                        下载
                      </Button>
                    </>
                  )}
                  <Popconfirm
                    title={`确认删除 ${row.path} ?`}
                    onConfirm={() => {
                      if (!selected || !clusterId) return;
                      void (async () => {
                        await deletePodFile({
                          cluster_id: clusterId,
                          namespace: selected.namespace,
                          name: selected.name,
                          path: row.path,
                        });
                        message.success("删除成功");
                        await loadFiles(selected, filePath);
                      })();
                    }}
                  >
                    <Button type="link" danger icon={<DeleteOutlined />}>删除</Button>
                  </Popconfirm>
                </Space>
              ),
            },
          ]}
        />
        <Divider />
        <Typography.Text strong>文件内容预览</Typography.Text>
        <Input.TextArea rows={14} value={fileContent} readOnly style={{ marginTop: 8 }} placeholder="点击“查看”显示文本内容" />
      </Drawer>
      <Drawer
        title={selected ? `Exec 进入容器 - ${selected.namespace}/${selected.name}` : "Exec 进入容器"}
        open={execOpen}
        onClose={() => {
          setExecOpen(false);
          closeExecSocket();
        }}
        width={760}
      >
        <div style={{ display: "flex", gap: 12, marginBottom: 10 }}>
          <Input
            value={execCommand}
            onChange={(e) => setExecCommand(e.target.value)}
            placeholder="启动命令（默认 sh），例如：bash"
            style={{ maxWidth: 320 }}
          />
          <Button
            icon={<CodeOutlined />}
            onClick={() => {
              // reopen to restart session
              closeExecSocket();
              execTermRef.current?.reset();
              if (execTermRef.current) execTermRef.current.writeln("\r\n[reconnecting…]");
              // trigger effect by toggling
              setExecOpen(false);
              setTimeout(() => setExecOpen(true), 0);
            }}
          >
            重连
          </Button>
        </div>
        <div
          ref={execTermHostRef}
          style={{
            height: "calc(100vh - 230px)",
            maxHeight: 720,
            borderRadius: 12,
            overflow: "hidden",
            border: "1px solid rgba(142, 162, 192, 0.28)",
          }}
        />
      </Drawer>
      <Drawer
        title={selected ? `Pod 详情 - ${selected.namespace}/${selected.name}` : "Pod 详情"}
        open={detailOpen}
        onClose={() => setDetailOpen(false)}
        width={760}
        className="detail-edit-drawer"
      >
        {!detail ? (
          <Typography.Text type="secondary">加载详情中...</Typography.Text>
        ) : (
          <Space direction="vertical" size={12} style={{ width: "100%" }}>
            <Form layout="vertical" className="detail-edit-form">
              <Form.Item label="名称">
                <Input value={`${detail.namespace}/${detail.name}`} readOnly />
              </Form.Item>
              <Form.Item label="UID">
                <Input value={detail.uid || "-"} readOnly />
              </Form.Item>
              <Form.Item label="状态">
                <Input value={detail.phase || "-"} readOnly />
              </Form.Item>
              <Form.Item label="节点">
                <Input value={detail.node_name || "-"} readOnly />
              </Form.Item>
              <Form.Item label="IP">
                <Input value={detail.pod_ip || "-"} readOnly />
              </Form.Item>
              <Form.Item label="宿主机 IP">
                <Input value={detail.host_ip || "-"} readOnly />
              </Form.Item>
              <Form.Item label="QoS">
                <Input value={detail.qos_class || "-"} readOnly />
              </Form.Item>
              <Form.Item label="ServiceAccount">
                <Input value={detail.service_account || "-"} readOnly />
              </Form.Item>
              <Form.Item label="创建时间">
                <Input value={formatDateTime(detail.creation_time)} readOnly />
              </Form.Item>
              <Form.Item label="启动时间">
                <Input value={formatDateTime(detail.start_time)} readOnly />
              </Form.Item>
              <Form.Item label="镜像">
                <Input value={detail.containers?.[0]?.image || "-"} readOnly />
              </Form.Item>
              <Form.Item label="容器名">
                <Input value={detail.containers?.[0]?.name || detail.name} readOnly />
              </Form.Item>
              <Form.Item label="启动命令">
                <Input value="-（仅支持通过高级编辑修改）" readOnly />
              </Form.Item>
              <Space style={{ width: "100%" }} size="middle">
                <Form.Item label="镜像拉取策略" style={{ flex: 1 }}>
                  <Input value={(detail as any).image_pull_policy || "-"} readOnly />
                </Form.Item>
                <Form.Item label="重启策略" style={{ flex: 1 }}>
                  <Input value={(detail as any).restart_policy || "-"} readOnly />
                </Form.Item>
              </Space>
              <Form.Item label="PriorityClassName">
                <Input value={detail.priority_class_name || "-"} readOnly />
              </Form.Item>
              <Form.Item label="标签（每行 key=value）">
                <Input.TextArea
                  rows={3}
                  value={
                    detail.labels && Object.keys(detail.labels).length > 0
                      ? Object.entries(detail.labels).map(([k, v]) => `${k}=${v}`).join("\n")
                      : "-"
                  }
                  readOnly
                />
              </Form.Item>
              <Form.Item label="NodeSelector（每行 key=value）">
                <Input.TextArea
                  rows={3}
                  value={
                    detail.node_selector && Object.keys(detail.node_selector).length > 0
                      ? Object.entries(detail.node_selector).map(([k, v]) => `${k}=${v}`).join("\n")
                      : "-"
                  }
                  readOnly
                />
              </Form.Item>
              <Form.Item label="注解">
                <Input.TextArea
                  rows={3}
                  value={
                    detail.annotations && Object.keys(detail.annotations).length > 0
                      ? Object.entries(detail.annotations).map(([k, v]) => `${k}=${v}`).join("\n")
                      : "-"
                  }
                  readOnly
                />
              </Form.Item>
            </Form>
            <Divider style={{ margin: "8px 0" }} />
            <Typography.Text strong>容器信息</Typography.Text>
            <Table
              size="small"
              rowKey="name"
              pagination={false}
              dataSource={detail.containers}
              columns={[
                { title: "容器", dataIndex: "name" },
                {
                  title: "镜像",
                  dataIndex: "image",
                  render: (image: string) => (
                    <Typography.Text
                      copyable
                      ellipsis={{ tooltip: image }}
                      style={{ maxWidth: 340, display: "inline-block" }}
                    >
                      {image}
                    </Typography.Text>
                  ),
                },
                { title: "状态", dataIndex: "state", width: 90, render: (v: string) => <Tag>{v}</Tag> },
                { title: "重启", dataIndex: "restart_count", width: 70 },
              ]}
            />
            <Divider style={{ margin: "8px 0" }} />
            <Typography.Text strong>卷信息</Typography.Text>
            <Table
              size="small"
              rowKey={(r) => r.name}
              pagination={false}
              dataSource={detail.volumes || []}
              locale={{ emptyText: "无卷信息" }}
              columns={[
                { title: "卷名", dataIndex: "name", width: 160 },
                {
                  title: "类型",
                  render: (_: unknown, v: any) => {
                    if (v.configMap) return "ConfigMap";
                    if (v.secret) return "Secret";
                    if (v.persistentVolumeClaim) return "PVC";
                    if (v.emptyDir) return "EmptyDir";
                    if (v.hostPath) return "HostPath";
                    return "其他";
                  },
                  width: 120,
                },
                {
                  title: "详情",
                  render: (_: unknown, v: any) => (
                    <Typography.Text ellipsis={{ tooltip: JSON.stringify(v) }} style={{ maxWidth: 320 }}>
                      {v.configMap?.name || v.secret?.secretName || v.persistentVolumeClaim?.claimName || v.hostPath?.path || "-"}
                    </Typography.Text>
                  ),
                },
              ]}
            />
            <Divider style={{ margin: "8px 0" }} />
            <Typography.Text strong>最近事件</Typography.Text>
            <Table
              size="small"
              rowKey={(r) => `${r.reason}-${r.last_timestamp}-${r.message}`}
              pagination={{ pageSize: 5 }}
              dataSource={events}
              columns={[
                { title: "类型", dataIndex: "type", width: 70, render: (v: string) => <Tag>{v}</Tag> },
                { title: "原因", dataIndex: "reason", width: 110 },
                { title: "消息", dataIndex: "message", ellipsis: true },
                { title: "时间", dataIndex: "last_timestamp", width: 140, render: (v: string) => formatDateTime(v) },
              ]}
            />
          </Space>
        )}
      </Drawer>
      <Drawer
        title={
          <Space direction="vertical" size={0}>
            <span>
              {simpleMode === "edit"
                ? `编辑 Pod（重建） - ${editTarget?.namespace || namespace}/${editTarget?.name || ""}`
                : "创建 Pod"}
            </span>
            <Typography.Text type="secondary" style={{ fontSize: 13, fontWeight: "normal" }}>
              目标命名空间：{namespace}
            </Typography.Text>
          </Space>
        }
        placement="right"
        width={960}
        open={createOpen}
        onClose={() => {
          setCreateOpen(false);
          setSimpleMode("create");
          setEditTarget(null);
        }}
        destroyOnClose
        maskClosable={false}
        styles={{ body: { paddingBottom: 24 } }}
        extra={
          <Button
            onClick={() => {
              setCreateOpen(false);
              setSimpleMode("create");
              setEditTarget(null);
            }}
          >
            取消
          </Button>
        }
      >
        <Tabs
          items={[
            {
              key: "simple",
              label: "表单创建",
              children: (
                <Form
                  form={simpleForm}
                  layout="vertical"
                  requiredMark="optional"
                  scrollToFirstError
                  initialValues={{
                    name: "",
                    image: "nginx:latest",
                    command: "",
                    image_pull_policy: "IfNotPresent",
                    restart_policy: "Always",
                    env_pairs: [],
                    label_pairs: [],
                    node_selector_pairs: [],
                    tolerations: [],
                    priority_class_name: "",
                    affinity: {},
                  }}
                >
                  <Form.Item
                    name="name"
                    label="Pod 名称"
                    rules={[
                      { required: true, message: "请输入 Pod 名称" },
                      {
                        validator: async (_, value) => {
                          const v = String(value || "").trim();
                          if (!v) return;
                          if (!rfc1123Subdomain.test(v)) {
                            throw new Error("Pod 名称不合法：必须全小写，且仅包含字母/数字/短横线/点，首尾为字母或数字");
                          }
                        },
                      },
                    ]}
                  >
                    <Input />
                  </Form.Item>
                  <Form.Item
                    name="container_name"
                    label="容器名称"
                    extra="默认同 Pod 名称"
                    rules={[
                      {
                        validator: async (_, value) => {
                          const v = String(value || "").trim();
                          if (!v) return;
                          if (!rfc1123Label.test(v)) {
                            throw new Error("容器名称不合法：必须全小写，且仅包含字母/数字/短横线，首尾为字母或数字");
                          }
                        },
                      },
                    ]}
                  >
                    <Input placeholder="默认同 Pod 名称" />
                  </Form.Item>
                  <Form.Item name="image" label="镜像" rules={[{ required: true, message: "请输入镜像" }]}>
                    <Input />
                  </Form.Item>
                  <Space style={{ width: "100%" }} size="middle">
                    <Form.Item name="image_pull_policy" label="镜像拉取策略" style={{ flex: 1 }}>
                      <Select
                        options={[
                          { label: "IfNotPresent", value: "IfNotPresent" },
                          { label: "Always", value: "Always" },
                          { label: "Never", value: "Never" },
                        ]}
                      />
                    </Form.Item>
                    <Form.Item name="restart_policy" label="重启策略" style={{ flex: 1 }}>
                      <Select
                        options={[
                          { label: "Always", value: "Always" },
                          { label: "OnFailure", value: "OnFailure" },
                          { label: "Never", value: "Never" },
                        ]}
                      />
                    </Form.Item>
                    <Form.Item name="port" label="容器端口" style={{ width: 140 }}>
                      <InputNumber min={1} max={65535} style={{ width: "100%" }} />
                    </Form.Item>
                  </Space>
                  <Form.Item name="command" label="启动命令" extra="例如覆盖镜像默认 CMD；留空则使用镜像入口">
                    <Input placeholder="例如：sleep 3600" />
                  </Form.Item>
                  <Space style={{ width: "100%" }} size="middle">
                    <Form.Item name="requests_cpu" label="CPU 请求" style={{ flex: 1 }}>
                      <Input placeholder="例如：100m" />
                    </Form.Item>
                    <Form.Item name="requests_memory" label="内存请求" style={{ flex: 1 }}>
                      <Input placeholder="例如：128Mi" />
                    </Form.Item>
                  </Space>
                  <Space style={{ width: "100%" }} size="middle">
                    <Form.Item name="limits_cpu" label="CPU 限制" style={{ flex: 1 }}>
                      <Input placeholder="例如：500m" />
                    </Form.Item>
                    <Form.Item name="limits_memory" label="内存限制" style={{ flex: 1 }}>
                      <Input placeholder="例如：512Mi" />
                    </Form.Item>
                  </Space>
                  <Form.List name="env_pairs">
                    {(fields, { add, remove }) => (
                      <Form.Item label="环境变量" extra="按键值对添加，KEY 不可重复">
                        <Space direction="vertical" style={{ width: "100%" }}>
                          {fields.map((field) => (
                            <Space key={field.key} style={{ width: "100%" }} align="start">
                              <Form.Item
                                {...field}
                                name={[field.name, "key"]}
                                rules={[
                                  { required: true, message: "请输入变量名" },
                                  {
                                    validator: async (_, value) => {
                                      const key = String(value || "").trim();
                                      if (!key) return;
                                      const list = simpleForm.getFieldValue("env_pairs") || [];
                                      const count = list.filter((it: { key?: string }) => String(it?.key || "").trim() === key).length;
                                      if (count > 1) throw new Error("变量名不能重复");
                                    },
                                  },
                                ]}
                                style={{ marginBottom: 0, flex: 1 }}
                              >
                                <Input placeholder="KEY" />
                              </Form.Item>
                              <Form.Item
                                {...field}
                                name={[field.name, "value"]}
                                style={{ marginBottom: 0, flex: 1 }}
                              >
                                <Input placeholder="VALUE" />
                              </Form.Item>
                              <Button danger onClick={() => remove(field.name)}>
                                删除
                              </Button>
                            </Space>
                          ))}
                          <Button type="dashed" onClick={() => add()}>
                            新增环境变量
                          </Button>
                        </Space>
                      </Form.Item>
                    )}
                  </Form.List>
                  <Form.List name="node_selector_pairs">
                    {(fields, { add, remove }) => (
                      <Form.Item label="NodeSelector" extra="按键值对添加，用于节点选择">
                        <Space direction="vertical" style={{ width: "100%" }}>
                          {fields.map((field) => (
                            <Space key={field.key} style={{ width: "100%" }} align="start">
                              <Form.Item
                                {...field}
                                name={[field.name, "key"]}
                                rules={[{ required: true, message: "请输入选择器键" }]}
                                style={{ marginBottom: 0, flex: 1 }}
                              >
                                <Input placeholder="key" />
                              </Form.Item>
                              <Form.Item {...field} name={[field.name, "value"]} style={{ marginBottom: 0, flex: 1 }}>
                                <Input placeholder="value" />
                              </Form.Item>
                              <Button danger onClick={() => remove(field.name)}>删除</Button>
                            </Space>
                          ))}
                          <Button type="dashed" onClick={() => add()}>新增 NodeSelector</Button>
                        </Space>
                      </Form.Item>
                    )}
                  </Form.List>
                  <Form.Item name="priority_class_name" label="PriorityClassName">
                    <Input placeholder="例如：system-cluster-critical" />
                  </Form.Item>
                  <Divider style={{ margin: "8px 0" }} />
                  <Typography.Text strong>Affinity</Typography.Text>
                  <Typography.Paragraph className="inline-muted" style={{ margin: "6px 0 0" }}>
                    以表单方式配置 NodeAffinity / PodAffinity / PodAntiAffinity；未填写的块不会下发到 PodSpec。
                  </Typography.Paragraph>

                  <Card size="small" title="NodeAffinity（节点亲和）" style={{ marginTop: 10 }}>
                    <Form.List name={["affinity", "node", "required"]}>
                      {(fields, { add, remove }) => (
                        <Form.Item label="Required（必须满足）" style={{ marginBottom: 0 }}>
                          <Space direction="vertical" style={{ width: "100%" }}>
                            {fields.map((field) => (
                              <Card
                                key={field.key}
                                size="small"
                                type="inner"
                                title={`Term #${field.name + 1}`}
                                extra={<Button danger onClick={() => remove(field.name)}>删除 Term</Button>}
                              >
                                <Form.List name={[field.name, "match_expressions"]}>
                                  {(expFields, expOps) => (
                                    <Space direction="vertical" style={{ width: "100%" }}>
                                      {expFields.map((ef) => (
                                        <Space key={ef.key} style={{ width: "100%" }} align="start" wrap>
                                          <Form.Item
                                            {...ef}
                                            name={[ef.name, "key"]}
                                            rules={[{ required: true, message: "key 必填" }]}
                                            style={{ marginBottom: 0, width: 220 }}
                                          >
                                            <Input placeholder="key" />
                                          </Form.Item>
                                          <Form.Item
                                            {...ef}
                                            name={[ef.name, "operator"]}
                                            rules={[{ required: true, message: "operator 必填" }]}
                                            style={{ marginBottom: 0, width: 180 }}
                                          >
                                            <Select
                                              options={[
                                                { label: "In", value: "In" },
                                                { label: "NotIn", value: "NotIn" },
                                                { label: "Exists", value: "Exists" },
                                                { label: "DoesNotExist", value: "DoesNotExist" },
                                                { label: "Gt", value: "Gt" },
                                                { label: "Lt", value: "Lt" },
                                              ]}
                                            />
                                          </Form.Item>
                                          <Form.Item
                                            {...ef}
                                            name={[ef.name, "values"]}
                                            style={{ marginBottom: 0, width: 320 }}
                                            tooltip="In/NotIn 需要 values；Exists/DoesNotExist 可留空；Gt/Lt 建议填单个数字"
                                          >
                                            <Select mode="tags" placeholder="values" />
                                          </Form.Item>
                                          <Button danger onClick={() => expOps.remove(ef.name)}>删除</Button>
                                        </Space>
                                      ))}
                                      <Button type="dashed" onClick={() => expOps.add({ operator: "In", values: [] })}>
                                        新增 MatchExpression
                                      </Button>
                                    </Space>
                                  )}
                                </Form.List>
                              </Card>
                            ))}
                            <Button type="dashed" onClick={() => add({ match_expressions: [] })}>
                              新增 Required Term
                            </Button>
                          </Space>
                        </Form.Item>
                      )}
                    </Form.List>

                    <Divider style={{ margin: "12px 0" }} />
                    <Form.List name={["affinity", "node", "preferred"]}>
                      {(fields, { add, remove }) => (
                        <Form.Item label="Preferred（尽量满足）" style={{ marginBottom: 0 }}>
                          <Space direction="vertical" style={{ width: "100%" }}>
                            {fields.map((field) => (
                              <Card
                                key={field.key}
                                size="small"
                                type="inner"
                                title={`Preference #${field.name + 1}`}
                                extra={<Button danger onClick={() => remove(field.name)}>删除</Button>}
                              >
                                <Space style={{ width: "100%" }} align="start" wrap>
                                  <Form.Item
                                    {...field}
                                    name={[field.name, "weight"]}
                                    rules={[{ required: true, message: "weight 必填" }]}
                                    style={{ marginBottom: 0, width: 180 }}
                                  >
                                    <InputNumber min={1} max={100} style={{ width: "100%" }} placeholder="weight(1-100)" />
                                  </Form.Item>
                                </Space>
                                <Form.List name={[field.name, "match_expressions"]}>
                                  {(expFields, expOps) => (
                                    <Space direction="vertical" style={{ width: "100%", marginTop: 10 }}>
                                      {expFields.map((ef) => (
                                        <Space key={ef.key} style={{ width: "100%" }} align="start" wrap>
                                          <Form.Item
                                            {...ef}
                                            name={[ef.name, "key"]}
                                            rules={[{ required: true, message: "key 必填" }]}
                                            style={{ marginBottom: 0, width: 220 }}
                                          >
                                            <Input placeholder="key" />
                                          </Form.Item>
                                          <Form.Item
                                            {...ef}
                                            name={[ef.name, "operator"]}
                                            rules={[{ required: true, message: "operator 必填" }]}
                                            style={{ marginBottom: 0, width: 180 }}
                                          >
                                            <Select
                                              options={[
                                                { label: "In", value: "In" },
                                                { label: "NotIn", value: "NotIn" },
                                                { label: "Exists", value: "Exists" },
                                                { label: "DoesNotExist", value: "DoesNotExist" },
                                                { label: "Gt", value: "Gt" },
                                                { label: "Lt", value: "Lt" },
                                              ]}
                                            />
                                          </Form.Item>
                                          <Form.Item
                                            {...ef}
                                            name={[ef.name, "values"]}
                                            style={{ marginBottom: 0, width: 320 }}
                                          >
                                            <Select mode="tags" placeholder="values" />
                                          </Form.Item>
                                          <Button danger onClick={() => expOps.remove(ef.name)}>删除</Button>
                                        </Space>
                                      ))}
                                      <Button type="dashed" onClick={() => expOps.add({ operator: "In", values: [] })}>
                                        新增 MatchExpression
                                      </Button>
                                    </Space>
                                  )}
                                </Form.List>
                              </Card>
                            ))}
                            <Button type="dashed" onClick={() => add({ weight: 50, match_expressions: [] })}>
                              新增 Preferred 规则
                            </Button>
                          </Space>
                        </Form.Item>
                      )}
                    </Form.List>
                  </Card>

                  <Card size="small" title="PodAffinity（Pod 亲和）" style={{ marginTop: 12 }}>
                    <Form.List name={["affinity", "pod", "required"]}>
                      {(fields, { add, remove }) => (
                        <Form.Item label="Required" style={{ marginBottom: 0 }}>
                          <Space direction="vertical" style={{ width: "100%" }}>
                            {fields.map((field) => (
                              <Card
                                key={field.key}
                                size="small"
                                type="inner"
                                title={`Term #${field.name + 1}`}
                                extra={<Button danger onClick={() => remove(field.name)}>删除</Button>}
                              >
                                <Form.Item name={[field.name, "topology_key"]} rules={[{ required: true, message: "topologyKey 必填" }]}>
                                  <Input placeholder="topologyKey，例如：kubernetes.io/hostname" />
                                </Form.Item>
                                <Form.List name={[field.name, "match_labels"]}>
                                  {(kvFields, kvOps) => (
                                    <Space direction="vertical" style={{ width: "100%" }}>
                                      {kvFields.map((kv) => (
                                        <Space key={kv.key} style={{ width: "100%" }} align="start">
                                          <Form.Item {...kv} name={[kv.name, "key"]} rules={[{ required: true, message: "key 必填" }]} style={{ marginBottom: 0, flex: 1 }}>
                                            <Input placeholder="key" />
                                          </Form.Item>
                                          <Form.Item {...kv} name={[kv.name, "value"]} style={{ marginBottom: 0, flex: 1 }}>
                                            <Input placeholder="value" />
                                          </Form.Item>
                                          <Button danger onClick={() => kvOps.remove(kv.name)}>删除</Button>
                                        </Space>
                                      ))}
                                      <Button type="dashed" onClick={() => kvOps.add()}>新增 matchLabel</Button>
                                    </Space>
                                  )}
                                </Form.List>
                              </Card>
                            ))}
                            <Button type="dashed" onClick={() => add({ match_labels: [] })}>新增 Required Term</Button>
                          </Space>
                        </Form.Item>
                      )}
                    </Form.List>

                    <Divider style={{ margin: "12px 0" }} />
                    <Form.List name={["affinity", "pod", "preferred"]}>
                      {(fields, { add, remove }) => (
                        <Form.Item label="Preferred" style={{ marginBottom: 0 }}>
                          <Space direction="vertical" style={{ width: "100%" }}>
                            {fields.map((field) => (
                              <Card
                                key={field.key}
                                size="small"
                                type="inner"
                                title={`Preferred #${field.name + 1}`}
                                extra={<Button danger onClick={() => remove(field.name)}>删除</Button>}
                              >
                                <Space style={{ width: "100%" }} wrap>
                                  <Form.Item name={[field.name, "weight"]} rules={[{ required: true, message: "weight 必填" }]} style={{ width: 180 }}>
                                    <InputNumber min={1} max={100} style={{ width: "100%" }} placeholder="weight(1-100)" />
                                  </Form.Item>
                                  <Form.Item name={[field.name, "topology_key"]} rules={[{ required: true, message: "topologyKey 必填" }]} style={{ flex: 1 }}>
                                    <Input placeholder="topologyKey，例如：kubernetes.io/hostname" />
                                  </Form.Item>
                                </Space>
                                <Form.List name={[field.name, "match_labels"]}>
                                  {(kvFields, kvOps) => (
                                    <Space direction="vertical" style={{ width: "100%" }}>
                                      {kvFields.map((kv) => (
                                        <Space key={kv.key} style={{ width: "100%" }} align="start">
                                          <Form.Item {...kv} name={[kv.name, "key"]} rules={[{ required: true, message: "key 必填" }]} style={{ marginBottom: 0, flex: 1 }}>
                                            <Input placeholder="key" />
                                          </Form.Item>
                                          <Form.Item {...kv} name={[kv.name, "value"]} style={{ marginBottom: 0, flex: 1 }}>
                                            <Input placeholder="value" />
                                          </Form.Item>
                                          <Button danger onClick={() => kvOps.remove(kv.name)}>删除</Button>
                                        </Space>
                                      ))}
                                      <Button type="dashed" onClick={() => kvOps.add()}>新增 matchLabel</Button>
                                    </Space>
                                  )}
                                </Form.List>
                              </Card>
                            ))}
                            <Button type="dashed" onClick={() => add({ weight: 50, match_labels: [] })}>新增 Preferred 规则</Button>
                          </Space>
                        </Form.Item>
                      )}
                    </Form.List>
                  </Card>

                  <Card size="small" title="PodAntiAffinity（Pod 反亲和）" style={{ marginTop: 12 }}>
                    <Form.List name={["affinity", "pod_anti", "required"]}>
                      {(fields, { add, remove }) => (
                        <Form.Item label="Required" style={{ marginBottom: 0 }}>
                          <Space direction="vertical" style={{ width: "100%" }}>
                            {fields.map((field) => (
                              <Card key={field.key} size="small" type="inner" title={`Term #${field.name + 1}`} extra={<Button danger onClick={() => remove(field.name)}>删除</Button>}>
                                <Form.Item name={[field.name, "topology_key"]} rules={[{ required: true, message: "topologyKey 必填" }]}>
                                  <Input placeholder="topologyKey，例如：kubernetes.io/hostname" />
                                </Form.Item>
                                <Form.List name={[field.name, "match_labels"]}>
                                  {(kvFields, kvOps) => (
                                    <Space direction="vertical" style={{ width: "100%" }}>
                                      {kvFields.map((kv) => (
                                        <Space key={kv.key} style={{ width: "100%" }} align="start">
                                          <Form.Item {...kv} name={[kv.name, "key"]} rules={[{ required: true, message: "key 必填" }]} style={{ marginBottom: 0, flex: 1 }}>
                                            <Input placeholder="key" />
                                          </Form.Item>
                                          <Form.Item {...kv} name={[kv.name, "value"]} style={{ marginBottom: 0, flex: 1 }}>
                                            <Input placeholder="value" />
                                          </Form.Item>
                                          <Button danger onClick={() => kvOps.remove(kv.name)}>删除</Button>
                                        </Space>
                                      ))}
                                      <Button type="dashed" onClick={() => kvOps.add()}>新增 matchLabel</Button>
                                    </Space>
                                  )}
                                </Form.List>
                              </Card>
                            ))}
                            <Button type="dashed" onClick={() => add({ match_labels: [] })}>新增 Required Term</Button>
                          </Space>
                        </Form.Item>
                      )}
                    </Form.List>

                    <Divider style={{ margin: "12px 0" }} />
                    <Form.List name={["affinity", "pod_anti", "preferred"]}>
                      {(fields, { add, remove }) => (
                        <Form.Item label="Preferred" style={{ marginBottom: 0 }}>
                          <Space direction="vertical" style={{ width: "100%" }}>
                            {fields.map((field) => (
                              <Card key={field.key} size="small" type="inner" title={`Preferred #${field.name + 1}`} extra={<Button danger onClick={() => remove(field.name)}>删除</Button>}>
                                <Space style={{ width: "100%" }} wrap>
                                  <Form.Item name={[field.name, "weight"]} rules={[{ required: true, message: "weight 必填" }]} style={{ width: 180 }}>
                                    <InputNumber min={1} max={100} style={{ width: "100%" }} placeholder="weight(1-100)" />
                                  </Form.Item>
                                  <Form.Item name={[field.name, "topology_key"]} rules={[{ required: true, message: "topologyKey 必填" }]} style={{ flex: 1 }}>
                                    <Input placeholder="topologyKey，例如：kubernetes.io/hostname" />
                                  </Form.Item>
                                </Space>
                                <Form.List name={[field.name, "match_labels"]}>
                                  {(kvFields, kvOps) => (
                                    <Space direction="vertical" style={{ width: "100%" }}>
                                      {kvFields.map((kv) => (
                                        <Space key={kv.key} style={{ width: "100%" }} align="start">
                                          <Form.Item {...kv} name={[kv.name, "key"]} rules={[{ required: true, message: "key 必填" }]} style={{ marginBottom: 0, flex: 1 }}>
                                            <Input placeholder="key" />
                                          </Form.Item>
                                          <Form.Item {...kv} name={[kv.name, "value"]} style={{ marginBottom: 0, flex: 1 }}>
                                            <Input placeholder="value" />
                                          </Form.Item>
                                          <Button danger onClick={() => kvOps.remove(kv.name)}>删除</Button>
                                        </Space>
                                      ))}
                                      <Button type="dashed" onClick={() => kvOps.add()}>新增 matchLabel</Button>
                                    </Space>
                                  )}
                                </Form.List>
                              </Card>
                            ))}
                            <Button type="dashed" onClick={() => add({ weight: 50, match_labels: [] })}>新增 Preferred 规则</Button>
                          </Space>
                        </Form.Item>
                      )}
                    </Form.List>
                  </Card>
                  <Form.List name="label_pairs">
                    {(fields, { add, remove }) => (
                      <Form.Item label="标签" extra="按键值对添加，key 不可重复">
                        <Space direction="vertical" style={{ width: "100%" }}>
                          {fields.map((field) => (
                            <Space key={field.key} style={{ width: "100%" }} align="start">
                              <Form.Item
                                {...field}
                                name={[field.name, "key"]}
                                rules={[
                                  { required: true, message: "请输入标签键" },
                                  {
                                    validator: async (_, value) => {
                                      const key = String(value || "").trim();
                                      if (!key) return;
                                      const list = simpleForm.getFieldValue("label_pairs") || [];
                                      const count = list.filter((it: { key?: string }) => String(it?.key || "").trim() === key).length;
                                      if (count > 1) throw new Error("标签键不能重复");
                                    },
                                  },
                                ]}
                                style={{ marginBottom: 0, flex: 1 }}
                              >
                                <Input placeholder="key" />
                              </Form.Item>
                              <Form.Item
                                {...field}
                                name={[field.name, "value"]}
                                style={{ marginBottom: 0, flex: 1 }}
                              >
                                <Input placeholder="value" />
                              </Form.Item>
                              <Button danger onClick={() => remove(field.name)}>
                                删除
                              </Button>
                            </Space>
                          ))}
                          <Button type="dashed" onClick={() => add()}>
                            新增标签
                          </Button>
                        </Space>
                      </Form.Item>
                    )}
                  </Form.List>
                  <Form.List name="tolerations">
                    {(fields, { add, remove }) => (
                      <Form.Item label="容忍（Tolerations）" extra="用于匹配节点污点；污点(Taints)是节点配置，不在 Pod 内创建">
                        <Space direction="vertical" style={{ width: "100%" }}>
                          {fields.map((field) => (
                            <Space key={field.key} style={{ width: "100%" }} align="start" wrap>
                              <Form.Item
                                {...field}
                                name={[field.name, "key"]}
                                rules={[{ required: true, message: "请输入 key" }]}
                                style={{ marginBottom: 0, width: 150 }}
                              >
                                <Input placeholder="key" />
                              </Form.Item>
                              <Form.Item
                                {...field}
                                name={[field.name, "operator"]}
                                initialValue="Equal"
                                style={{ marginBottom: 0, width: 130 }}
                              >
                                <Select
                                  options={[
                                    { label: "Equal", value: "Equal" },
                                    { label: "Exists", value: "Exists" },
                                  ]}
                                />
                              </Form.Item>
                              <Form.Item
                                {...field}
                                name={[field.name, "value"]}
                                style={{ marginBottom: 0, width: 160 }}
                              >
                                <Input placeholder="value" />
                              </Form.Item>
                              <Form.Item
                                {...field}
                                name={[field.name, "effect"]}
                                style={{ marginBottom: 0, width: 170 }}
                              >
                                <Select
                                  allowClear
                                  placeholder="effect"
                                  options={[
                                    { label: "NoSchedule", value: "NoSchedule" },
                                    { label: "PreferNoSchedule", value: "PreferNoSchedule" },
                                    { label: "NoExecute", value: "NoExecute" },
                                  ]}
                                />
                              </Form.Item>
                              <Form.Item
                                {...field}
                                name={[field.name, "toleration_seconds"]}
                                style={{ marginBottom: 0, width: 160 }}
                              >
                                <InputNumber min={1} style={{ width: "100%" }} placeholder="seconds" />
                              </Form.Item>
                              <Button danger onClick={() => remove(field.name)}>
                                删除
                              </Button>
                            </Space>
                          ))}
                          <Button
                            type="dashed"
                            onClick={() =>
                              add({ operator: "Equal" })
                            }
                          >
                            新增容忍
                          </Button>
                        </Space>
                      </Form.Item>
                    )}
                  </Form.List>
                  <Button type="primary" loading={creating} onClick={() => void submitCreateSimple()}>
                    {simpleMode === "edit" ? "保存并重建" : "创建"}
                  </Button>
                </Form>
              ),
            },
            ...(simpleMode === "create"
              ? [
                  {
                    key: "yaml",
                    label: "YAML 创建",
                    children: (
                      <Form form={yamlForm} layout="vertical" requiredMark="optional" scrollToFirstError initialValues={{ manifest: "" }}>
                        <Form.Item name="manifest" label="YAML 内容" rules={[{ required: true, message: "请输入 YAML" }]}>
                          <Input.TextArea rows={12} placeholder="apiVersion: v1&#10;kind: Pod&#10;metadata:&#10;  name: demo-pod&#10;spec:&#10;  containers:&#10;  - name: main&#10;    image: nginx:latest" />
                        </Form.Item>
                        <Button type="primary" loading={creating} onClick={() => void submitCreateYAML()}>
                          创建
                        </Button>
                      </Form>
                    ),
                  },
                ]
              : []),
          ]}
        />
      </Drawer>
    </div>
  );
}
