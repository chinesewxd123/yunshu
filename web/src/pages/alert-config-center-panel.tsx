import { CopyOutlined, DeleteOutlined, EditOutlined, MinusCircleOutlined, PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import { Alert, AutoComplete, Button, Card, Drawer, Form, Input, InputNumber, Modal, Popconfirm, Popover, Select, Space, Statistic, Switch, Table, Tabs, Tag, Tree, Typography, message } from "antd";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  getAlertHistoryStats,
  listAlertChannels,
  listAlertEvents,
  sendAlertmanagerWebhook,
  type AlertEventItem,
} from "../services/alerts";
import { stringifyPrettyJSON } from "../services/alert-mappers";
import { useDictOptions } from "../hooks/use-dict-options";
import { formatDateTime } from "../utils/format";
import {
  ALERT_EVENT_CATEGORY_OPTIONS,
  ALERT_HISTORY_PIPELINE_HELP,
  describeAlertEvent,
  summarizeAlertEventHint,
  type AlertEventCategory,
} from "../utils/alert-event-reasons";
import { ALERT_ROUTING_TERMS, formatReceiverGroupLabel, formatRouteNodeTreeTitle } from "../constants/alert-routing-terms";
import { listAlertDatasources, type AlertDatasourceItem } from "../services/alert-platform";
import { getProjects } from "../services/projects";
import {
  cloneSubscriptionFromProject,
  createReceiverGroup,
  createSubscriptionNode,
  deleteReceiverGroup,
  deleteSubscriptionNode,
  getSubscriptionTree,
  listReceiverGroups,
  updateReceiverGroup,
  updateSubscriptionNode,
  type AlertReceiverGroup,
  type AlertSubscriptionNode,
} from "../services/alert-subscriptions";

export type AlertConfigTab = "subscriptions" | "history";

export type AlertConfigCenterPanelProps = {
  /** 当前子 Tab（策略 / 历史 / 模板） */
  activeTab: AlertConfigTab;
  onTabChange: (key: AlertConfigTab) => void;
  /** 嵌入「告警监控平台」时不显示最外层标题 Card */
  embedded?: boolean;
  /** 嵌入主页面时，仅展示当前视图内容，不再显示内部 tabs */
  hideTabs?: boolean;
  /** 历史 Tab 初始策略分类（如从抑制页跳转 ?event_category=inhibition） */
  initialEventCategory?: AlertEventCategory;
  /** 告警监控平台顶栏「全局项目上下文」；有值时同步订阅/历史筛选 */
  projectContextId?: number;
};

const webhookPayloadTemplates: Record<string, Record<string, unknown>> = {
  warning_prod: {
    receiver: "yunshu-webhook",
    status: "firing",
    alerts: [
      {
        status: "firing",
        labels: {
          alertname: "KubernetesPodUnhealthy",
          severity: "warning",
          cluster: "prodK8s",
          namespace: "default",
          pod: "demo-pod-1",
        },
        annotations: {
          summary: "Pod 异常（warning）",
          description: "演示告警：warning 路由",
        },
        startsAt: "2026-04-18T09:20:00Z",
        endsAt: "0001-01-01T00:00:00Z",
        generatorURL: "http://prometheus.example/graph?g0.expr=up",
        fingerprint: "demo-warning-prod-001",
      },
    ],
  },
  critical_prod: {
    receiver: "yunshu-webhook",
    status: "firing",
    alerts: [
      {
        status: "firing",
        labels: {
          alertname: "KubernetesNodeNotReady",
          severity: "critical",
          cluster: "prodK8s",
          namespace: "kube-system",
          node: "worker-1",
        },
        annotations: {
          summary: "节点不可用（critical）",
          description: "演示告警：critical 路由",
        },
        startsAt: "2026-04-18T09:21:00Z",
        endsAt: "0001-01-01T00:00:00Z",
        generatorURL: "http://prometheus.example/graph?g0.expr=node_ready",
        fingerprint: "demo-critical-prod-001",
      },
    ],
  },
  resolved_prod: {
    receiver: "yunshu-webhook",
    status: "resolved",
    alerts: [
      {
        status: "resolved",
        labels: {
          alertname: "KubernetesNodeNotReady",
          severity: "critical",
          cluster: "prodK8s",
          namespace: "kube-system",
          node: "worker-1",
        },
        annotations: {
          summary: "节点恢复（resolved）",
          description: "演示恢复通知",
        },
        startsAt: "2026-04-18T09:21:00Z",
        endsAt: "2026-04-18T09:25:00Z",
        generatorURL: "http://prometheus.example/graph?g0.expr=node_ready",
        fingerprint: "demo-critical-prod-001",
      },
    ],
  },
};

/** 从告警历史入库体中解析顶层 `labels`（与后端 hydrate 逻辑一致）。 */
function parseLabelsFromAlertEventRequestPayload(raw?: string): Record<string, string> {
  const s = String(raw || "").trim();
  if (!s) return {};
  try {
    const payload = JSON.parse(s) as Record<string, unknown>;
    const labels = payload?.labels;
    if (labels && typeof labels === "object" && !Array.isArray(labels)) {
      const out: Record<string, string> = {};
      for (const [k, v] of Object.entries(labels as Record<string, unknown>)) {
        const vs = String(v ?? "").trim();
        if (vs && vs !== "<nil>") {
          out[String(k).trim()] = vs;
        }
      }
      return out;
    }
  } catch {
    /* ignore */
  }
  return {};
}

function parseReceiverGroupChannelIds(g: AlertReceiverGroup): number[] {
  if (Array.isArray(g.channel_ids) && g.channel_ids.length) {
    return g.channel_ids.map((id) => Number(id)).filter((id) => id > 0);
  }
  try {
    const parsed = JSON.parse(String(g.channel_ids_json || "[]")) as unknown;
    if (Array.isArray(parsed)) {
      return parsed.map((id) => Number(id)).filter((id) => id > 0);
    }
  } catch {
    /* ignore */
  }
  return [];
}

function parseReceiverGroupEmails(g: AlertReceiverGroup): string[] {
  if (Array.isArray(g.email_recipients) && g.email_recipients.length) {
    return g.email_recipients.map((e) => String(e).trim()).filter(Boolean);
  }
  try {
    const parsed = JSON.parse(String(g.email_recipients_json || "[]")) as unknown;
    if (Array.isArray(parsed)) {
      return parsed.map((e) => String(e).trim()).filter(Boolean);
    }
  } catch {
    /* ignore */
  }
  return [];
}

function prettifyAlertRequestPayload(raw?: string): string {
  const s = String(raw || "").trim();
  if (!s) return "";
  try {
    return stringifyPrettyJSON(JSON.parse(s) as unknown, s);
  } catch {
    return s;
  }
}

export function AlertConfigCenterPanel({
  activeTab: tab,
  onTabChange: setTab,
  embedded,
  hideTabs,
  initialEventCategory,
  projectContextId,
}: AlertConfigCenterPanelProps) {
  const [stats, setStats] = useState<{
    total: number;
    firing: number;
    resolved: number;
    success: number;
    failed: number;
    today_created: number;
    cluster_values?: string[];
    monitor_pipeline_values?: string[];
    datasource_filter_options?: Array<{ id: number; name: string }>;
  }>();

  const [channels, setChannels] = useState<Array<{ id: number; name: string }>>([]);

  // subscriptions (新策略)
  const [projects, setProjects] = useState<Array<{ id: number; name: string }>>([]);
  const [subProjectID, setSubProjectID] = useState<number>(0);
  const [subTree, setSubTree] = useState<AlertSubscriptionNode[]>([]);
  const [subSelectedID, setSubSelectedID] = useState<number>(0);
  const [subLoading, setSubLoading] = useState(false);
  const [receiverGroups, setReceiverGroups] = useState<AlertReceiverGroup[]>([]);
  const [projectDatasources, setProjectDatasources] = useState<AlertDatasourceItem[]>([]);
  const [subForm] = Form.useForm();
  const [cloneModalOpen, setCloneModalOpen] = useState(false);
  const [cloneSubmitting, setCloneSubmitting] = useState(false);
  const [cloneForm] = Form.useForm<{
    source_project_id: number;
    target_project_id: number;
    replace_cluster?: string;
    replace_route?: string;
    include_disabled?: boolean;
    skip_if_target_has_nodes?: boolean;
  }>();
  const [rgDrawerOpen, setRgDrawerOpen] = useState(false);
  const [rgModalOpen, setRgModalOpen] = useState(false);
  const [rgEditingId, setRgEditingId] = useState<number | null>(null);
  const [rgSaving, setRgSaving] = useState(false);
  const [rgForm] = Form.useForm<{
    name: string;
    description?: string;
    channel_ids?: number[];
    email_recipients?: string[];
    enabled?: boolean;
  }>();

  const [eventsLoading, setEventsLoading] = useState(false);
  const [events, setEvents] = useState<AlertEventItem[]>([]);
  const [eventsPage, setEventsPage] = useState(1);
  const [eventsPageSize, setEventsPageSize] = useState(10);
  const [eventsTotal, setEventsTotal] = useState(0);
  const [eventKeyword, setEventKeyword] = useState("");
  const [eventAlertIP, setEventAlertIP] = useState("");
  const [eventStatus, setEventStatus] = useState("");
  /** 格式：`ds:<数据源ID>` 或 `mp:<monitor_pipeline slug>`（兼容历史 prometheus/platform） */
  const [eventSourceFilter, setEventSourceFilter] = useState("");
  const [eventGroupKey, setEventGroupKey] = useState("");
  const [eventCategory, setEventCategory] = useState<AlertEventCategory | "">(initialEventCategory ?? "");
  const eventsPageSizeRef = useRef(eventsPageSize);
  eventsPageSizeRef.current = eventsPageSize;
  const [webhookSending, setWebhookSending] = useState(false);
  const [webhookToken, setWebhookToken] = useState("");
  const [webhookTemplate, setWebhookTemplate] = useState<"warning_prod" | "critical_prod" | "resolved_prod">("warning_prod");
  const [webhookPayload, setWebhookPayload] = useState(
    JSON.stringify(webhookPayloadTemplates.warning_prod, null, 2),
  );
  const webhookTokenOptions = useDictOptions("alert_webhook_token");
  const promqlLabelKeyOptions = useDictOptions("alert_promql_label_key");
  const webhookTokenPick = useMemo(() => {
    const t = String(webhookToken || "").trim();
    const m = webhookTokenOptions.find((o) => String(o.value ?? "").trim() === t);
    return m?.value;
  }, [webhookToken, webhookTokenOptions]);

  const channelOptions = useMemo(() => channels.map((c) => ({ label: c.name, value: c.id })), [channels]);
  const labelKeyAutoCompleteOptions = useMemo(
    () => {
      return promqlLabelKeyOptions
        .map((opt) => {
          const value = String(opt.value ?? "").trim();
          const label = String(opt.label ?? "").trim() || value;
          return { value, label: `${label} (${value})` };
        })
        .filter((it) => it.value)
        .sort((a, b) => a.value.localeCompare(b.value, "zh-CN"));
    },
    [promqlLabelKeyOptions],
  );

  const alertIPOptions = useMemo(() => {
    const fromPage = (events ?? [])
      .map((it) => String(it.alertIP || "").trim())
      .filter(Boolean);
    const merged = Array.from(new Set([...fromPage])).sort((a, b) => a.localeCompare(b, "zh-CN"));
    return merged.map((v) => ({ label: v, value: v }));
  }, [events]);

  const sourceFilterOptions = useMemo(() => {
    const opts: { label: string; value: string }[] = [];
    const seen = new Set<string>();
    const push = (label: string, value: string) => {
      if (!value || seen.has(value)) return;
      seen.add(value);
      opts.push({ label, value });
    };
    push("Alertmanager", "mp:alertmanager");
    push("云资源到期", "mp:cloud_expiry");
    for (const ds of projectDatasources) {
      const id = Number(ds.id);
      if (!Number.isFinite(id) || id <= 0) continue;
      const name = String(ds.name ?? "").trim() || `数据源 ${id}`;
      push(name, `ds:${id}`);
    }
    if (opts.length <= 2) {
      for (const row of stats?.datasource_filter_options ?? []) {
        const id = Number(row?.id);
        if (!Number.isFinite(id) || id <= 0) continue;
        const name = String(row?.name ?? "").trim() || `数据源 ${id}`;
        push(name, `ds:${id}`);
      }
    }
    return opts;
  }, [projectDatasources, stats?.datasource_filter_options]);
  const webhookTemplateOptions = useMemo(
    () => [
      { label: "warning（prod）", value: "warning_prod" },
      { label: "critical（prod）", value: "critical_prod" },
      { label: "resolved（prod）", value: "resolved_prod" },
    ],
    [],
  );

  function applyWebhookTemplate(key: "warning_prod" | "critical_prod" | "resolved_prod") {
    setWebhookTemplate(key);
    setWebhookPayload(JSON.stringify(webhookPayloadTemplates[key], null, 2));
  }

  async function loadBase() {
    const [statsRes, channelRes] = await Promise.all([getAlertHistoryStats(), listAlertChannels()]);
    setStats(statsRes);
    setChannels((channelRes.list ?? []).map((c) => ({ id: c.id, name: c.name })));
  }

  async function loadProjects() {
    const res = await getProjects({ page: 1, page_size: 200 });
    const list = (res?.list ?? []) as Array<{ id: number; name: string }>;
    const normalized = list.map((it) => ({ id: Number((it as any).id), name: String((it as any).name || "") })).filter((it) => it.id > 0);
    setProjects(normalized);
    if (!subProjectID && normalized.length) setSubProjectID(normalized[0].id);
  }

  /** 同名接收组只展示一条，保留 id 较大者（通常为最近迁移/创建），避免下拉重复 */
  const receiverGroupOptions = useMemo(() => {
    const byNameKey = new Map<string, { label: string; value: number }>();
    for (const g of receiverGroups) {
      const id = Number(g.id);
      if (!Number.isFinite(id) || id <= 0) continue;
      const label = formatReceiverGroupLabel(String(g.name ?? ""), id);
      const key = label.toLowerCase();
      const prev = byNameKey.get(key);
      if (!prev || id > prev.value) {
        byNameKey.set(key, { label, value: id });
      }
    }
    return Array.from(byNameKey.values()).sort((a, b) => a.label.localeCompare(b.label, "zh-CN"));
  }, [receiverGroups]);

  const subscriptionSeverityOptions = useMemo(
    () =>
      ["critical", "warning", "info", "error", "none"].map((v) => ({
        label: v,
        value: v,
      })),
    [],
  );

  type SubscriptionAntTreeNode = { key: string; title: string; children?: SubscriptionAntTreeNode[] };

  const subscriptionTreeData = useMemo(() => {
    const toTree = (nodes: AlertSubscriptionNode[]): SubscriptionAntTreeNode[] =>
      (nodes ?? []).map((n) => {
        const ch = toTree(n.children ?? []);
        const row: SubscriptionAntTreeNode = {
          key: String(n.id),
          title: formatRouteNodeTreeTitle(n.name, n.enabled),
        };
        if (ch.length > 0) row.children = ch;
        return row;
      });
    return toTree(subTree);
  }, [subTree]);

  const selectedSubscriptionNode = useMemo(() => {
    const walk = (nodes: AlertSubscriptionNode[]): AlertSubscriptionNode | null => {
      for (const n of nodes ?? []) {
        if (n.id === subSelectedID) return n;
        const hit = walk(n.children ?? []);
        if (hit) return hit;
      }
      return null;
    };
    return walk(subTree);
  }, [subTree, subSelectedID]);

  const loadSubscriptions = useCallback(async (overrideProjectId?: number) => {
    const pid = overrideProjectId ?? (projectContextId && projectContextId > 0 ? projectContextId : subProjectID);
    if (!pid) return;
    setSubLoading(true);
    try {
      const [tree, groups] = await Promise.all([
        getSubscriptionTree({ project_id: pid }),
        listReceiverGroups({ project_id: pid, page: 1, page_size: 200 }),
      ]);
      setSubTree(tree ?? []);
      setReceiverGroups(groups.list ?? groups.items ?? []);
    } finally {
      setSubLoading(false);
    }
  }, [subProjectID, projectContextId]);

  function openReceiverGroupCreate() {
    setRgEditingId(null);
    rgForm.resetFields();
    rgForm.setFieldsValue({ enabled: true, channel_ids: [], email_recipients: [] });
    setRgModalOpen(true);
  }

  function openReceiverGroupEdit(g: AlertReceiverGroup) {
    setRgEditingId(g.id);
    rgForm.setFieldsValue({
      name: g.name,
      description: g.description ?? "",
      channel_ids: parseReceiverGroupChannelIds(g),
      email_recipients: parseReceiverGroupEmails(g),
      enabled: g.enabled,
    });
    setRgModalOpen(true);
  }

  async function saveReceiverGroup() {
    const pid = effectiveProjectId;
    if (!pid) {
      message.warning("请先选择项目");
      return;
    }
    const values = await rgForm.validateFields();
    const payload = {
      project_id: pid,
      name: String(values.name ?? "").trim(),
      description: String(values.description ?? "").trim(),
      channel_ids_json: JSON.stringify(values.channel_ids ?? []),
      email_recipients_json: JSON.stringify(values.email_recipients ?? []),
      enabled: values.enabled !== false,
    };
    setRgSaving(true);
    try {
      if (rgEditingId) {
        await updateReceiverGroup(rgEditingId, payload);
      } else {
        await createReceiverGroup(payload);
      }
      message.success("接收组已保存");
      setRgModalOpen(false);
      await loadSubscriptions();
    } finally {
      setRgSaving(false);
    }
  }

  async function removeReceiverGroup(id: number) {
    await deleteReceiverGroup(id);
    message.success("已删除");
    await loadSubscriptions();
  }

  async function onSelectSubscriptionNode(id: number) {
    setSubSelectedID(id);
    const node = (() => {
      const walk = (nodes: AlertSubscriptionNode[]): AlertSubscriptionNode | null => {
        for (const n of nodes ?? []) {
          if (n.id === id) return n;
          const hit = walk(n.children ?? []);
          if (hit) return hit;
        }
        return null;
      };
      return walk(subTree);
    })();
    if (!node) return;
    subForm.setFieldsValue({
      id: node.id,
      project_id: node.project_id,
      parent_id: node.parent_id ?? null,
      name: node.name,
      code: node.code,
      enabled: node.enabled,
      continue: node.continue,
      match_labels_json: node.match_labels_json ?? "{}",
      match_regex_json: node.match_regex_json ?? "{}",
      match_severity: node.match_severity
        ? String(node.match_severity)
            .split(/[,，;|]/)
            .map((s) => s.trim())
            .filter(Boolean)
        : [],
      receiver_group_ids: node.receiver_group_ids ?? [],
      silence_seconds: node.silence_seconds ?? 0,
      notify_resolved: node.notify_resolved,
    });
  }

  async function createSubscription(parentID?: number | null) {
    if (!effectiveProjectId) return;
    const payload: any = {
      project_id: effectiveProjectId,
      parent_id: parentID ?? null,
      name: !parentID ? ALERT_ROUTING_TERMS.rootPolicyName : "新路由节点",
      code: "",
      enabled: true,
      continue: false,
      match_labels_json: "{}",
      match_regex_json: "{}",
      match_severity: "",
      receiver_group_ids_json: "[]",
      silence_seconds: 0,
      notify_resolved: true,
    };
    const created = await createSubscriptionNode(payload);
    message.success("已创建");
    await loadSubscriptions();
    await onSelectSubscriptionNode(created.id);
  }

  async function saveSubscription() {
    const v = await subForm.validateFields();
    const id = Number(v.id || 0);
    const payload: any = {
      project_id: effectiveProjectId,
      parent_id: v.parent_id ?? null,
      name: String(v.name || "").trim(),
      code: String(v.code || "").trim(),
      enabled: !!v.enabled,
      continue: !!v.continue,
      match_labels_json: String(v.match_labels_json || "{}"),
      match_regex_json: String(v.match_regex_json || "{}"),
      match_severity: Array.isArray(v.match_severity)
        ? (v.match_severity as string[]).map((x) => String(x).trim()).filter(Boolean).join(",")
        : String(v.match_severity || "")
            .trim()
            .split(/[,，;|]/)
            .map((s) => s.trim())
            .filter(Boolean)
            .join(","),
      receiver_group_ids_json: JSON.stringify((v.receiver_group_ids ?? []).map((x: any) => Number(x)).filter((x: number) => x > 0)),
      silence_seconds: Number(v.silence_seconds || 0),
      notify_resolved: !!v.notify_resolved,
    };
    if (!id) {
      const created = await createSubscriptionNode(payload);
      message.success("已保存");
      await loadSubscriptions();
      await onSelectSubscriptionNode(created.id);
      return;
    }
    await updateSubscriptionNode(id, payload);
    message.success("已保存");
    await loadSubscriptions();
  }

  async function removeSubscription() {
    if (!subSelectedID) return;
    await deleteSubscriptionNode(subSelectedID);
    message.success("已删除");
    setSubSelectedID(0);
    subForm.resetFields();
    await loadSubscriptions();
  }

  const effectiveProjectId = projectContextId && projectContextId > 0 ? projectContextId : subProjectID;

  const loadEvents = useCallback(
    async (page: number, pageSize: number) => {
      setEventsLoading(true);
      try {
        const src = String(eventSourceFilter || "").trim();
        let datasourceId: number | undefined;
        let monitorPipeline: string | undefined;
        if (src.startsWith("ds:")) {
          const id = Number(src.slice(3));
          if (Number.isFinite(id) && id > 0) datasourceId = id;
        } else if (src.startsWith("mp:")) {
          const slug = src.slice(3).trim();
          if (slug) monitorPipeline = slug;
        }
        const res = await listAlertEvents({
          page,
          page_size: pageSize,
          keyword: eventKeyword.trim() || undefined,
          alertIP: eventAlertIP.trim() || undefined,
          status: eventStatus.trim() || undefined,
          monitorPipeline,
          datasourceId,
          groupKey: eventGroupKey.trim() || undefined,
          category: eventCategory || undefined,
          projectId: effectiveProjectId > 0 ? effectiveProjectId : undefined,
        });
        setEvents(res.list ?? []);
        setEventsTotal(res.total ?? 0);
        setEventsPage(res.page ?? page);
        setEventsPageSize(res.page_size ?? pageSize);
      } finally {
        setEventsLoading(false);
      }
    },
    [eventKeyword, eventAlertIP, eventStatus, eventSourceFilter, eventGroupKey, eventCategory, effectiveProjectId],
  );

  useEffect(() => {
    if (initialEventCategory) {
      setEventCategory(initialEventCategory);
    }
  }, [initialEventCategory]);

  useEffect(() => {
    void loadBase();
    void loadProjects();
  }, []);

  useEffect(() => {
    if (projectContextId && projectContextId > 0) {
      setSubProjectID(projectContextId);
    }
  }, [projectContextId]);

  useEffect(() => {
    if (tab !== "history") {
      return;
    }
    const delay =
      eventKeyword || eventAlertIP || eventStatus || eventSourceFilter || eventGroupKey || eventCategory ? 300 : 0;
    const timer = window.setTimeout(() => {
      void loadEvents(1, eventsPageSizeRef.current);
    }, delay);
    return () => window.clearTimeout(timer);
  }, [
    tab,
    eventKeyword,
    eventAlertIP,
    eventStatus,
    eventSourceFilter,
    eventGroupKey,
    eventCategory,
    effectiveProjectId,
    loadEvents,
  ]);

  useEffect(() => {
    if (tab !== "subscriptions") {
      return;
    }
    void loadSubscriptions();
  }, [tab, loadSubscriptions]);

  useEffect(() => {
    if (tab !== "history") return;
    const pid = effectiveProjectId > 0 ? effectiveProjectId : undefined;
    void listAlertDatasources({ project_id: pid, page: 1, page_size: 200 }).then((r) => {
      setProjectDatasources(r.list ?? r.items ?? []);
    });
  }, [tab, effectiveProjectId]);

  async function sendWebhookDemo() {
    let payloadObj: Record<string, unknown>;
    try {
      payloadObj = JSON.parse(webhookPayload || "{}") as Record<string, unknown>;
    } catch {
      message.error("Webhook Payload 不是合法 JSON");
      return;
    }
    setWebhookSending(true);
    try {
      await sendAlertmanagerWebhook(payloadObj, webhookToken);
      message.success("Webhook 已发送，告警链路已触发");
      await loadEvents(1, eventsPageSize);
      await loadBase();
    } finally {
      setWebhookSending(false);
    }
  }

  const tabItems = [
    {
      key: "subscriptions",
      label: ALERT_ROUTING_TERMS.tabRouting,
      children: (
        <>
          <Space style={{ width: "100%", marginBottom: 12 }} wrap>
            {projectContextId ? (
              <Typography.Text type="secondary">
                当前项目：{projects.find((p) => p.id === projectContextId)?.name ?? `项目 ${projectContextId}`}（跟随顶栏「全局项目上下文」）
              </Typography.Text>
            ) : (
              <Select
                style={{ width: 260 }}
                placeholder="选择项目"
                value={subProjectID || undefined}
                options={projects.map((p) => ({ label: p.name, value: p.id }))}
                onChange={(v) => setSubProjectID(Number(v) || 0)}
                showSearch
                filterOption={(input, option) => String(option?.label ?? "").toLowerCase().includes(input.toLowerCase())}
              />
            )}
            <Button icon={<ReloadOutlined />} loading={subLoading} onClick={() => void loadSubscriptions()}>
              刷新
            </Button>
            <Button disabled={!effectiveProjectId} onClick={() => setRgDrawerOpen(true)}>
              {ALERT_ROUTING_TERMS.receiverGroupManage}
            </Button>
            <Button
              icon={<CopyOutlined />}
              disabled={projects.length < 2}
              onClick={() => {
                cloneForm.setFieldsValue({
                  source_project_id: effectiveProjectId || projects[0]?.id,
                  target_project_id: undefined,
                  replace_cluster: "",
                  replace_route: "",
                  include_disabled: false,
                  skip_if_target_has_nodes: true,
                });
                setCloneModalOpen(true);
              }}
            >
              {ALERT_ROUTING_TERMS.copyTemplate}
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => void createSubscription(null)}>
              新增根节点
            </Button>
            <Button disabled={!subSelectedID} icon={<PlusOutlined />} onClick={() => void createSubscription(subSelectedID)}>
              新增子节点
            </Button>
            <Popconfirm title="确认删除节点？（有子节点会失败）" onConfirm={() => void removeSubscription()}>
              <Button danger disabled={!subSelectedID} icon={<DeleteOutlined />}>
                删除
              </Button>
            </Popconfirm>
            <Button type="primary" disabled={!effectiveProjectId} onClick={() => void saveSubscription()}>
              保存
            </Button>
          </Space>
          <div style={{ display: "grid", gridTemplateColumns: "360px 1fr", gap: 12, alignItems: "start" }}>
            <Card size="small" title={ALERT_ROUTING_TERMS.treeTitle} loading={subLoading} styles={{ body: { padding: 8 } }}>
              <Tree
                treeData={subscriptionTreeData}
                selectedKeys={subSelectedID ? [String(subSelectedID)] : []}
                onSelect={(keys) => {
                  const id = Number(keys?.[0] ?? 0);
                  if (id > 0) void onSelectSubscriptionNode(id);
                }}
                defaultExpandAll
              />
            </Card>
            <Card
              size="small"
              title={
                selectedSubscriptionNode
                  ? `编辑路由节点：${formatRouteNodeTreeTitle(selectedSubscriptionNode.name, true)}`
                  : ALERT_ROUTING_TERMS.selectNodeHint
              }
            >
              <Form form={subForm} layout="vertical">
                <Form.Item name="id" hidden>
                  <Input />
                </Form.Item>
                <Form.Item name="parent_id" hidden>
                  <Input />
                </Form.Item>
                <Form.Item name="name" label={ALERT_ROUTING_TERMS.nodeName} rules={[{ required: true }]}>
                  <Input />
                </Form.Item>
                <Form.Item name="code" label={ALERT_ROUTING_TERMS.nodeCode}>
                  <Input />
                </Form.Item>
                <Space wrap style={{ width: "100%" }}>
                  <Form.Item name="enabled" label="启用" valuePropName="checked" style={{ marginBottom: 0 }}>
                    <Switch />
                  </Form.Item>
                  <Form.Item name="continue" label={ALERT_ROUTING_TERMS.continueMatchChildren} valuePropName="checked" style={{ marginBottom: 0 }}>
                    <Switch />
                  </Form.Item>
                  <Form.Item name="notify_resolved" label="恢复通知" valuePropName="checked" style={{ marginBottom: 0 }}>
                    <Switch />
                  </Form.Item>
                  <Form.Item name="silence_seconds" label="静默(s)" style={{ marginBottom: 0 }}>
                    <InputNumber min={0} />
                  </Form.Item>
                </Space>
                <Form.Item
                  name="match_severity"
                  label={`${ALERT_ROUTING_TERMS.matchSeverity}（可选，多选）`}
                  extra="告警 labels.severity 命中任一即通过；不选表示不按级别过滤。"
                >
                  <Select
                    mode="multiple"
                    allowClear
                    placeholder="不选则不限级别"
                    options={subscriptionSeverityOptions}
                  />
                </Form.Item>
                <Form.Item
                  name="receiver_group_ids"
                  label={ALERT_ROUTING_TERMS.receiverGroup}
                  dependencies={["parent_id"]}
                  rules={[
                    {
                      validator: async (_, value) => {
                        const pid = subForm.getFieldValue("parent_id");
                        const isRoot = pid === null || pid === undefined || pid === "";
                        if (isRoot) return;
                        const ids = Array.isArray(value) ? value : [];
                        if (ids.length === 0) {
                          throw new Error("非根节点须至少选择一个接收组");
                        }
                      },
                    },
                  ]}
                  extra={
                    <>
                      根节点可留空：仅作路由分流，通知由子节点上的接收组发出。请先点击上方「
                      {ALERT_ROUTING_TERMS.receiverGroupManage}」创建接收组并绑定告警通道。
                    </>
                  }
                >
                  <Select mode="multiple" options={receiverGroupOptions} placeholder="选择通知接收组" allowClear />
                </Form.Item>
                <Form.Item name="match_labels_json" label="match_labels_json（精确匹配 JSON）">
                  <Input.TextArea rows={4} />
                </Form.Item>
                <Form.Item name="match_regex_json" label="match_regex_json（正则匹配 JSON）">
                  <Input.TextArea rows={4} />
                </Form.Item>
              </Form>
            </Card>
          </div>
        </>
      ),
    },
    {
      key: "history",
      label: "历史告警记录",
      children: (
        <>
          <Alert
            type="info"
            showIcon
            style={{ marginBottom: 12 }}
            message="历史告警：通知投递审计与抑制原因码"
            description={
              <>
                <p style={{ marginBottom: 8 }}>
                  每一行对应一次<strong>外发尝试或策略留痕</strong>（<Typography.Text code>alert_events</Typography.Text>
                  ）。<Typography.Text code>success=true</Typography.Text> 且带 <Typography.Text code>error_message</Typography.Text>{" "}
                  时，通常表示「未外发但已记录原因」，并非通道 HTTP 失败。
                </p>
                <ul style={{ marginBottom: 8, paddingLeft: 18 }}>
                  {ALERT_HISTORY_PIPELINE_HELP.map((item) => (
                    <li key={item.title}>
                      <strong>{item.title}</strong>：{item.body}
                    </li>
                  ))}
                </ul>
                <p style={{ marginBottom: 0 }}>
                  <strong>Prometheus 活跃告警</strong>（/api/v1/alerts）请在「告警监控平台 → PromQL / 平台静默」查看；与本表「是否已进 Webhook
                  链路」不是同一数据源。抑制规则配置见「告警监控平台 → 告警抑制」。
                </p>
              </>
            }
          />
          {embedded && projectContextId ? (
            <Alert
              type="info"
              showIcon
              style={{ marginBottom: 12 }}
              message={`已按顶栏项目筛选历史记录（项目 #${projectContextId}）`}
            />
          ) : null}
          <Space style={{ width: "100%", marginBottom: 12 }} wrap>
            <Input
              style={{ width: 260 }}
              placeholder="关键词（标题/错误/通道）"
              value={eventKeyword}
              onChange={(e) => setEventKeyword(e.target.value)}
              allowClear
            />
            <Select
              style={{ width: 220 }}
              placeholder="告警IP（labels.instance / pod_ip）"
              value={eventAlertIP || undefined}
              options={alertIPOptions}
              showSearch
              allowClear
              onSearch={(v) => setEventAlertIP(v)}
              onChange={(v) => setEventAlertIP((v as string) || "")}
              filterOption={(input, option) => String(option?.value ?? "").toLowerCase().includes(input.toLowerCase())}
            />
            <Select
              style={{ width: 280 }}
              placeholder={ALERT_ROUTING_TERMS.historySourceFilter}
              value={eventSourceFilter || undefined}
              options={sourceFilterOptions}
              showSearch
              allowClear
              onChange={(v) => setEventSourceFilter((v as string) || "")}
              filterOption={(input, option) => String(option?.label ?? "").toLowerCase().includes(input.toLowerCase())}
            />
            <Select
              style={{ width: 160 }}
              placeholder="状态"
              value={eventStatus || undefined}
              options={[
                { label: "firing", value: "firing" },
                { label: "resolved", value: "resolved" },
              ]}
              allowClear
              onChange={(v) => setEventStatus((v as string) || "")}
            />
            <Select
              style={{ width: 160 }}
              placeholder="策略分类"
              value={eventCategory || undefined}
              options={ALERT_EVENT_CATEGORY_OPTIONS}
              allowClear
              onChange={(v) => setEventCategory((v as AlertEventCategory) || "")}
            />
            <Input
              style={{ width: 220 }}
              placeholder="groupKey"
              value={eventGroupKey}
              onChange={(e) => setEventGroupKey(e.target.value)}
              allowClear
            />
            <Button icon={<ReloadOutlined />} onClick={() => void loadEvents(eventsPage, eventsPageSize)}>
              刷新
            </Button>
          </Space>
          <Table
            rowKey="id"
            loading={eventsLoading}
            dataSource={events}
            scroll={{ x: 2460 }}
            pagination={{
              current: eventsPage,
              pageSize: eventsPageSize,
              total: eventsTotal,
              showSizeChanger: true,
              pageSizeOptions: [10, 20, 50, 100],
              showQuickJumper: true,
              onChange: (p, ps) => void loadEvents(p, ps),
            }}
            columns={[
              { title: "ID", dataIndex: "id", width: 80 },
              {
                title: "标题",
                dataIndex: "title",
                width: 360,
                render: (v: string) => (
                  <Typography.Text style={{ whiteSpace: "normal", wordBreak: "break-word" }}>
                    {v || "-"}
                  </Typography.Text>
                ),
              },
              { title: "告警IP", dataIndex: "alertIP", width: 160, ellipsis: true, render: (v: string) => v || "-" },
              {
                title: "数据源 / 来源",
                key: "datasourceDisplay",
                width: 200,
                render: (_: unknown, row: AlertEventItem) => {
                  const name = String(row.datasourceName ?? "").trim();
                  const typ = String(row.datasourceType ?? "").trim();
                  const slug = String(row.monitorPipeline ?? "").trim();
                  if (name) {
                    return (
                      <Space align="start" size={4} wrap>
                        <Typography.Text style={{ maxWidth: 160 }} ellipsis={{ tooltip: name }}>
                          {name}
                        </Typography.Text>
                        {typ ? <Tag>{typ}</Tag> : null}
                      </Space>
                    );
                  }
                  if (slug === "alertmanager") return <Tag color="blue">Alertmanager</Tag>;
                  if (slug === "cloud_expiry") return <Tag color="volcano">云到期</Tag>;
                  if (slug === "platform_monitor") return <Tag color="purple">平台规则</Tag>;
                  if (slug === "platform") return <Tag color="purple">platform（历史）</Tag>;
                  if (slug === "prometheus") return <Tag color="blue">prometheus（历史）</Tag>;
                  if (slug.startsWith("ds:")) return <Tag>{slug}</Tag>;
                  return slug ? <Tag>{slug}</Tag> : <span>-</span>;
                },
              },
              { title: "GroupKey", dataIndex: "groupKey", width: 140, ellipsis: true, render: (v: string) => v || "-" },
              {
                title: "级别",
                dataIndex: "severity",
                width: 100,
                render: (v: string) => (
                  <Tag color={v === "critical" ? "red" : v === "warning" ? "orange" : "blue"}>{v || "-"}</Tag>
                ),
              },
              { title: "状态", dataIndex: "status", width: 90, render: (v: string) => <Tag>{v || "-"}</Tag> },
              {
                title: "告警值 / 恢复时值",
                key: "metric_values",
                width: 200,
                ellipsis: true,
                render: (_: unknown, row: AlertEventItem) => {
                  const a = String(row.metricCurrent ?? "").trim();
                  const b = String(row.metricResolved ?? "").trim();
                  if (!a && !b) return <span className="inline-muted">-</span>;
                  if (String(row.status).toLowerCase() === "resolved" && b) {
                    return (
                      <Typography.Text ellipsis={{ tooltip: `触发侧快照: ${a || "—"}\n恢复时再查: ${b}` }} style={{ fontSize: 12 }}>
                        触发: {a || "—"} / 恢复: {b}
                      </Typography.Text>
                    );
                  }
                  return (
                    <Typography.Text ellipsis={{ tooltip: a }} style={{ fontSize: 12 }}>
                      {a || "—"}
                    </Typography.Text>
                  );
                },
              },
              {
                title: "命中策略",
                dataIndex: "matchedPolicyNames",
                width: 200,
                ellipsis: true,
                render: (_: string, row: AlertEventItem) => (row.matchedPolicyNameList?.length ? row.matchedPolicyNameList.join(", ") : "-"),
              },
              { title: "通道", dataIndex: "channelName", width: 160, ellipsis: true },
              {
                title: "接收人",
                dataIndex: "receiverList",
                width: 220,
                ellipsis: true,
                render: (_: unknown, row: AlertEventItem) => {
                  if (!row.receiverList?.length) return "-";
                  return (
                    <Space size={[4, 4]} wrap>
                      {row.receiverList.map((one) => (
                        <Tag key={`${row.id}-${one}`}>{one}</Tag>
                      ))}
                    </Space>
                  );
                },
              },
              {
                title: "发送结果",
                dataIndex: "success",
                width: 100,
                render: (v: boolean, row: AlertEventItem) => {
                  const reason = String(row.errorMessage || "").trim();
                  if (v && reason) {
                    return <Tag color="default">留痕</Tag>;
                  }
                  return v ? <Tag color="success">成功</Tag> : <Tag color="error">失败</Tag>;
                },
              },
              {
                title: "策略摘要",
                key: "reason_hint",
                width: 120,
                render: (_: unknown, row: AlertEventItem) => {
                  const hint = summarizeAlertEventHint(row);
                  if (hint === "-") return <span>-</span>;
                  return (
                    <Typography.Text ellipsis={{ tooltip: describeAlertEvent(row) }} style={{ fontSize: 12 }}>
                      {hint}
                    </Typography.Text>
                  );
                },
              },
              {
                title: "标签组",
                key: "labels_group",
                width: 110,
                render: (_: unknown, row: AlertEventItem) => {
                  const labels = parseLabelsFromAlertEventRequestPayload(row.requestPayload);
                  const entries = Object.entries(labels);
                  if (!entries.length) return <span>-</span>;
                  return (
                    <Popover
                      title="标签组（labels）"
                      trigger={["click"]}
                      overlayStyle={{ maxWidth: 560 }}
                      content={
                        <div style={{ maxHeight: 400, overflow: "auto" }}>
                          <Space size={[4, 8]} wrap>
                            {entries.map(([k, v]) => (
                              <Tag key={`${row.id}-${k}`} style={{ marginInlineEnd: 0 }}>
                                {k}={v}
                              </Tag>
                            ))}
                          </Space>
                        </div>
                      }
                    >
                      <Button size="small">查看标签</Button>
                    </Popover>
                  );
                },
              },
              {
                title: "告警数据原始 JSON",
                key: "raw_request_payload",
                width: 110,
                render: (_: unknown, row: AlertEventItem) => {
                  const raw = String(row.requestPayload || "").trim();
                  if (!raw) return <span>-</span>;
                  const pretty = prettifyAlertRequestPayload(raw);
                  const http = row.httpStatusCode;
                  return (
                    <Popover
                      title={`入库 requestPayload · HTTP ${http ?? "-"}`}
                      trigger={["click"]}
                      overlayStyle={{ maxWidth: 760 }}
                      content={
                        <pre
                          style={{
                            maxHeight: 480,
                            overflow: "auto",
                            margin: 0,
                            fontSize: 12,
                            whiteSpace: "pre-wrap",
                            wordBreak: "break-word",
                          }}
                        >
                          {pretty}
                        </pre>
                      }
                    >
                      <Button size="small">查看 JSON</Button>
                    </Popover>
                  );
                },
              },
              {
                title: "告警产生时间",
                dataIndex: "alertStartedAt",
                width: 170,
                render: (v: string) => (v ? formatDateTime(v) : "-"),
              },
              {
                title: "链路说明",
                key: "flow_explain",
                width: 340,
                ellipsis: true,
                render: (_: unknown, row: AlertEventItem) => {
                  const msg = describeAlertEvent(row);
                  return (
                    <Typography.Text type="secondary" ellipsis={{ tooltip: msg }}>
                      {msg}
                    </Typography.Text>
                  );
                },
              },
              { title: "发送/记录时间", dataIndex: "createdAt", width: 170, render: (v: string) => formatDateTime(v) },
            ]}
          />
        </>
      ),
    },
  ] as const;

  const activeContent = tabItems.find((item) => item.key === tab)?.children ?? null;

  const showOverviewAndDebug = !(embedded && hideTabs && tab === "subscriptions");

  const body = (
    <>
      {showOverviewAndDebug ? (
        <>
          <div
            style={{
              display: "grid",
              gridTemplateColumns: "repeat(auto-fit, minmax(130px, 1fr))",
              gap: 12,
              marginBottom: 12,
            }}
          >
            <Card size="small" styles={{ body: { padding: 12 } }}>
              <Statistic title="总告警数" value={stats?.total ?? 0} />
            </Card>
            <Card size="small" styles={{ body: { padding: 12 } }}>
              <Statistic title="Firing" value={stats?.firing ?? 0} valueStyle={{ color: "#cf1322" }} />
            </Card>
            <Card size="small" styles={{ body: { padding: 12 } }}>
              <Statistic title="Resolved" value={stats?.resolved ?? 0} valueStyle={{ color: "#389e0d" }} />
            </Card>
            <Card size="small" styles={{ body: { padding: 12 } }}>
              <Statistic title="发送成功" value={stats?.success ?? 0} valueStyle={{ color: "#1677ff" }} />
            </Card>
            <Card size="small" styles={{ body: { padding: 12 } }}>
              <Statistic title="发送失败" value={stats?.failed ?? 0} valueStyle={{ color: "#cf1322" }} />
            </Card>
            <Card size="small" styles={{ body: { padding: 12 } }}>
              <Statistic title="今日新增" value={stats?.today_created ?? 0} />
            </Card>
          </div>
          <Card size="small" title="Webhook 联调（Alertmanager -> 平台）" style={{ marginBottom: 12 }}>
            <Space direction="vertical" style={{ width: "100%" }} size={12}>
              <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
                与生产链路一致：Alertmanager 的 <Typography.Text code>webhook_configs</Typography.Text> 指向本平台的{" "}
                <Typography.Text code>POST /api/v1/alerts/webhook/alertmanager</Typography.Text>，携带与配置一致的 Token（
                <Typography.Text code>X-Webhook-Token</Typography.Text> / <Typography.Text code>Authorization</Typography.Text> / 查询参数{" "}
                <Typography.Text code>token</Typography.Text>
                ）。下方「发送模拟 Webhook」走同一接口，记录出现在「历史告警记录」Tab，并与策略命中、通道分发一致。
              </Typography.Paragraph>
              <Space wrap>
                <Input
                  style={{ width: 360 }}
                  value={webhookToken}
                  placeholder="Webhook Token（可为空）"
                  onChange={(e) => setWebhookToken(e.target.value)}
                />
                <Select
                  style={{ width: 300 }}
                  allowClear
                  value={webhookTokenPick}
                  options={webhookTokenOptions}
                  placeholder="从字典选择 webhook token"
                  onChange={(v) => setWebhookToken(String(v ?? ""))}
                />
                <Button type="primary" loading={webhookSending} onClick={() => void sendWebhookDemo()}>
                  发送模拟Webhook
                </Button>
                <Select
                  style={{ width: 220 }}
                  value={webhookTemplate}
                  options={webhookTemplateOptions}
                  onChange={(v) => applyWebhookTemplate(v as "warning_prod" | "critical_prod" | "resolved_prod")}
                />
                <Button onClick={() => applyWebhookTemplate(webhookTemplate)}>套用模板</Button>
              </Space>
              <Typography.Text type="secondary">
                已选择模板会立即同步到下方 JSON；发送前请确认 status 和 alerts[0].status 是否符合预期（firing/resolved）。
              </Typography.Text>
              <Input.TextArea
                rows={8}
                value={webhookPayload}
                onChange={(e) => setWebhookPayload(e.target.value)}
                placeholder="Alertmanager webhook JSON"
              />
            </Space>
          </Card>
        </>
      ) : null}

      {hideTabs ? (
        activeContent
      ) : (
        <Tabs activeKey={tab} onChange={(k) => setTab(k as AlertConfigTab)} items={tabItems as never} />
      )}

      <Drawer
        title={ALERT_ROUTING_TERMS.receiverGroupManage}
        width={720}
        open={rgDrawerOpen}
        onClose={() => setRgDrawerOpen(false)}
        extra={
          <Button type="primary" icon={<PlusOutlined />} disabled={!effectiveProjectId} onClick={openReceiverGroupCreate}>
            新建接收组
          </Button>
        }
      >
        <Alert type="info" showIcon style={{ marginBottom: 12 }} message={ALERT_ROUTING_TERMS.receiverGroupManageHint} />
        <Table
          rowKey="id"
          size="small"
          loading={subLoading}
          dataSource={receiverGroups}
          pagination={false}
          columns={[
            { title: "名称", dataIndex: "name", width: 160, render: (n: string, r: AlertReceiverGroup) => formatReceiverGroupLabel(n, r.id) },
            {
              title: "告警通道",
              render: (_: unknown, r: AlertReceiverGroup) => {
                const ids = parseReceiverGroupChannelIds(r);
                if (!ids.length) return <Typography.Text type="secondary">未绑定</Typography.Text>;
                return ids
                  .map((id) => channels.find((c) => c.id === id)?.name ?? `#${id}`)
                  .join("、");
              },
            },
            {
              title: "额外邮箱",
              render: (_: unknown, r: AlertReceiverGroup) => {
                const emails = parseReceiverGroupEmails(r);
                return emails.length ? emails.join("、") : "—";
              },
            },
            {
              title: "状态",
              dataIndex: "enabled",
              width: 72,
              render: (v: boolean) => (v ? <Tag color="green">启用</Tag> : <Tag>停用</Tag>),
            },
            {
              title: "操作",
              width: 120,
              render: (_: unknown, r: AlertReceiverGroup) => (
                <Space>
                  <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openReceiverGroupEdit(r)}>
                    编辑
                  </Button>
                  <Popconfirm title="确认删除该接收组？" onConfirm={() => void removeReceiverGroup(r.id)}>
                    <Button type="link" size="small" danger icon={<DeleteOutlined />}>
                      删除
                    </Button>
                  </Popconfirm>
                </Space>
              ),
            },
          ]}
        />
      </Drawer>

      <Modal
        title={rgEditingId ? "编辑通知接收组" : "新建通知接收组"}
        open={rgModalOpen}
        confirmLoading={rgSaving}
        onCancel={() => setRgModalOpen(false)}
        onOk={() => void saveReceiverGroup()}
        destroyOnClose
      >
        <Form form={rgForm} layout="vertical">
          <Form.Item name="name" label="接收组名称" rules={[{ required: true, message: "请输入名称" }]}>
            <Input placeholder="例如 prod-critical-dingding" />
          </Form.Item>
          <Form.Item name="description" label="说明">
            <Input.TextArea rows={2} placeholder="可选" />
          </Form.Item>
          <Form.Item
            name="channel_ids"
            label="绑定告警通道"
            rules={[
              {
                validator: async (_, value) => {
                  const ids = Array.isArray(value) ? value : [];
                  const emails = rgForm.getFieldValue("email_recipients") as string[] | undefined;
                  if (ids.length === 0 && (!emails || emails.length === 0)) {
                    throw new Error("请至少绑定一个告警通道或填写额外邮箱");
                  }
                },
              },
            ]}
            extra="通道在「告警通道」菜单维护；此处选择后，命中该接收组的路由节点将按通道类型投递。"
          >
            <Select
              mode="multiple"
              allowClear
              placeholder="选择钉钉 / 邮件 / 企微等通道"
              options={channels.map((c) => ({ label: c.name, value: c.id }))}
            />
          </Form.Item>
          <Form.Item name="email_recipients" label="额外邮箱（邮件兜底）">
            <Select mode="tags" tokenSeparators={[",", " ", ";"]} placeholder="输入邮箱后回车，可与通道并存" />
          </Form.Item>
          <Form.Item name="enabled" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={ALERT_ROUTING_TERMS.copyTemplate}
        open={cloneModalOpen}
        confirmLoading={cloneSubmitting}
        onCancel={() => setCloneModalOpen(false)}
        onOk={async () => {
          const v = await cloneForm.validateFields();
          setCloneSubmitting(true);
          try {
            const rep = await cloneSubscriptionFromProject({
              source_project_id: v.source_project_id,
              target_project_id: v.target_project_id,
              replace_cluster: v.replace_cluster?.trim() || undefined,
              replace_route: v.replace_route?.trim() || undefined,
              include_disabled: !!v.include_disabled,
              skip_if_target_has_nodes: v.skip_if_target_has_nodes !== false,
            });
            if (rep.skipped) {
              message.warning(rep.message || "目标项目已有节点，已跳过");
            } else {
              message.success(
                `已复制：接收组 ${rep.receiver_groups_created} 个，订阅节点 ${rep.nodes_created} 个${rep.message ? `（${rep.message}）` : ""}`,
              );
            }
            setCloneModalOpen(false);
            if (v.target_project_id === effectiveProjectId) {
              await loadSubscriptions(v.target_project_id);
            }
          } finally {
            setCloneSubmitting(false);
          }
        }}
      >
        <Form form={cloneForm} layout="vertical">
          <Form.Item name="source_project_id" label="源项目（已调配好的模板）" rules={[{ required: true }]}>
            <Select options={projects.map((p) => ({ label: p.name, value: p.id }))} showSearch />
          </Form.Item>
          <Form.Item name="target_project_id" label="目标项目" rules={[{ required: true }]}>
            <Select options={projects.map((p) => ({ label: p.name, value: p.id }))} showSearch />
          </Form.Item>
          <Form.Item name="replace_cluster" label="覆盖 cluster（可选，写入 match_labels）">
            <Input placeholder="例如 腾讯云告警链路" />
          </Form.Item>
          <Form.Item name="replace_route" label="覆盖 route（可选）">
            <Input placeholder="例如 prod-critical-dingding" />
          </Form.Item>
          <Form.Item name="include_disabled" label="包含已停用节点/接收组" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item
            name="skip_if_target_has_nodes"
            label="目标已有订阅树时跳过（推荐）"
            valuePropName="checked"
            extra="关闭后将清空目标项目已有订阅节点与接收组再复制（慎用）"
          >
            <Switch defaultChecked />
          </Form.Item>
        </Form>
      </Modal>

    </>
  );

  if (embedded) {
    return (
      <div className="alert-config-embedded">
        <div style={{ display: "flex", justifyContent: "flex-end", marginBottom: 8 }}>
          <Button icon={<ReloadOutlined />} onClick={() => void loadBase()}>
            刷新统计
          </Button>
        </div>
        {body}
      </div>
    );
  }

  return (
    <Card className="table-card" title="告警配置中心" extra={<Button icon={<ReloadOutlined />} onClick={() => void loadBase()}>刷新统计</Button>}>
      {body}
    </Card>
  );
}

