import { DeleteOutlined, EditOutlined, MinusCircleOutlined, PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import { Alert, AutoComplete, Button, Card, Drawer, Form, Input, InputNumber, Popconfirm, Select, Space, Statistic, Switch, Table, Tabs, Tag, Tree, Typography, message } from "antd";
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
import { getProjects } from "../services/projects";
import {
  createSubscriptionNode,
  deleteSubscriptionNode,
  getSubscriptionTree,
  listReceiverGroups,
  migratePoliciesToSubscriptions,
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

function describeAlertEvent(row: AlertEventItem): string {
  const reason = String(row.errorMessage || "").trim();
  const channelText = String(row.channelName || "").trim() || "未匹配通道";
  const receiverText = row.receiverList?.length ? row.receiverList.join(", ") : "-";
  const hasMatchedPolicy = !!row.matchedPolicyNameList?.length;
  if (row.success) {
    if (reason === "silence_suppressed") return "已命中平台静默，告警写入历史但未向通道发送。";
    if (reason === "subscription_suppressed") return "已命中订阅静默窗口，本次不再重复外发。";
    if (reason === "dedup_suppressed") return "已命中指纹去重，本次告警被去重抑制。";
    if (reason === "group_wait_suppressed") return "已进入 group_wait 收集窗口，稍后会统一发送。";
    if (reason === "group_interval_suppressed") return "组内发生新变化但仍处于 group_interval 窗口，本次暂不外发。";
    if (reason === "repeat_suppressed") return "处于 repeat_interval 窗口，本次重复提醒被抑制。";
    if (reason === "resolved_aggregate_suppressed") return "同一告警实例的重复恢复事件已抑制（恢复仅发送一次）。";
    if (row.channelName?.includes("静默抑制")) return "平台在分发前拦截了本次告警。";
    return `通道[${channelText}] 已发送，接收人[${receiverText}]。`;
  }
  if (reason === "no_enabled_channels") return "当前没有启用的通知通道，因此仅记录历史。";
  if (reason === "no_policy_matched") return "未命中任何订阅节点，已按订阅树门禁拦截，不会发送到通道。";
  if (reason === "no_channel_matched") return "有通道存在，但本次告警未匹配到可发送通道。";
  if (reason) return `通道[${channelText}] 发送失败，接收人[${receiverText}]，原因：${reason}`;
  return "告警进入了平台链路，但未获取到更多说明。";
}

function summarizeAlertHint(row: AlertEventItem): string {
  const reason = String(row.errorMessage || "").trim();
  if (!row.success) return "-";
  if (!reason) return "-";
  if (reason === "silence_suppressed") return "已命中平台静默，通知已拦截";
  if (reason === "subscription_suppressed") return "已命中订阅静默窗口，本次未外发";
  if (reason === "dedup_suppressed") return "已触发去重策略，本次不再重复发送";
  if (reason === "group_wait_suppressed") return "group_wait 收集窗口";
  if (reason === "group_interval_suppressed") return "group_interval 窗口";
  if (reason === "repeat_suppressed") return "repeat_interval 窗口";
  if (reason === "resolved_aggregate_suppressed") return "重复恢复事件已抑制（恢复仅发送一次）";
  if (/suppressed/i.test(reason)) return "已被系统策略抑制，本次未发送";
  return reason;
}

export function AlertConfigCenterPanel({ activeTab: tab, onTabChange: setTab, embedded, hideTabs }: AlertConfigCenterPanelProps) {
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
  const [subForm] = Form.useForm();

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
    for (const row of stats?.datasource_filter_options ?? []) {
      const id = Number(row?.id);
      if (!Number.isFinite(id) || id <= 0) continue;
      const name = String(row?.name ?? "").trim();
      const value = `ds:${id}`;
      if (seen.has(value)) continue;
      seen.add(value);
      opts.push({
        label: name ? `${name}（数据源 #${id}）` : `数据源 #${id}`,
        value,
      });
    }
    const slugLabels: Record<string, string> = {
      alertmanager: "Alertmanager（未绑定数据源）",
      platform_monitor: "平台监控规则（无数据源记录）",
      cloud_expiry: "云资源到期",
      prometheus: "历史：Prometheus + Alertmanager",
      platform: "历史：平台监控链路",
    };
    const slugFromPage = (events ?? []).map((it) => String(it.monitorPipeline ?? "").trim()).filter(Boolean);
    const fromStats = (stats?.monitor_pipeline_values ?? []).map((v) => String(v).trim()).filter(Boolean);
    for (const slug of Array.from(new Set([...fromStats, ...slugFromPage])).sort((a, b) => a.localeCompare(b, "zh-CN"))) {
      if (!slug || slug.startsWith("ds:")) continue;
      const value = `mp:${slug}`;
      if (seen.has(value)) continue;
      seen.add(value);
      opts.push({ label: slugLabels[slug] ?? `来源：${slug}`, value });
    }
    return opts.sort((a, b) => a.label.localeCompare(b.label, "zh-CN"));
  }, [stats?.datasource_filter_options, stats?.monitor_pipeline_values, events]);
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

  const receiverGroupOptions = useMemo(() => {
    const seen = new Set<number>();
    const out: { label: string; value: number }[] = [];
    for (const g of receiverGroups) {
      const id = Number(g.id);
      if (!Number.isFinite(id) || id <= 0 || seen.has(id)) continue;
      seen.add(id);
      out.push({ label: g.name, value: id });
    }
    return out;
  }, [receiverGroups]);

  type SubscriptionAntTreeNode = { key: string; title: string; children?: SubscriptionAntTreeNode[] };

  const subscriptionTreeData = useMemo(() => {
    const toTree = (nodes: AlertSubscriptionNode[]): SubscriptionAntTreeNode[] =>
      (nodes ?? []).map((n) => {
        const ch = toTree(n.children ?? []);
        const row: SubscriptionAntTreeNode = {
          key: String(n.id),
          title: `${n.name}${n.enabled ? "" : "（停用）"}`,
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
    const pid = overrideProjectId ?? subProjectID;
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
  }, [subProjectID]);

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
      match_severity: node.match_severity ?? "",
      receiver_group_ids: node.receiver_group_ids ?? [],
      silence_seconds: node.silence_seconds ?? 0,
      notify_resolved: node.notify_resolved,
    });
  }

  async function createSubscription(parentID?: number | null) {
    if (!subProjectID) return;
    const payload: any = {
      project_id: subProjectID,
      parent_id: parentID ?? null,
      name: "新节点",
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
      project_id: subProjectID,
      parent_id: v.parent_id ?? null,
      name: String(v.name || "").trim(),
      code: String(v.code || "").trim(),
      enabled: !!v.enabled,
      continue: !!v.continue,
      match_labels_json: String(v.match_labels_json || "{}"),
      match_regex_json: String(v.match_regex_json || "{}"),
      match_severity: String(v.match_severity || "").trim(),
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

  async function migrateOldPolicies() {
    const rep = await migratePoliciesToSubscriptions({
      disable_old: true,
      ...(subProjectID ? { default_project_id: subProjectID } : {}),
    });
    const targetPid = Number(rep.resolved_default_project_id) || subProjectID;
    if (targetPid > 0) {
      setSubProjectID(targetPid);
    }
    message.success(
      `迁移完成：策略${rep.policies_total}，迁移${rep.policies_migrated}，节点${rep.nodes_created}，接收组${rep.receiver_groups_created}` +
        (targetPid > 0 ? `（已切换到项目 ID ${targetPid} 查看订阅树）` : ""),
    );
    await loadSubscriptions(targetPid > 0 ? targetPid : undefined);
  }

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
        });
        setEvents(res.list ?? []);
        setEventsTotal(res.total ?? 0);
        setEventsPage(res.page ?? page);
        setEventsPageSize(res.page_size ?? pageSize);
      } finally {
        setEventsLoading(false);
      }
    },
    [eventKeyword, eventAlertIP, eventStatus, eventSourceFilter, eventGroupKey],
  );

  useEffect(() => {
    void loadBase();
    void loadProjects();
  }, []);

  useEffect(() => {
    if (tab !== "history") {
      return;
    }
    const delay = eventKeyword || eventAlertIP || eventStatus || eventSourceFilter || eventGroupKey ? 300 : 0;
    const timer = window.setTimeout(() => {
      void loadEvents(1, eventsPageSizeRef.current);
    }, delay);
    return () => window.clearTimeout(timer);
  }, [tab, eventKeyword, eventAlertIP, eventStatus, eventSourceFilter, eventGroupKey, loadEvents]);

  useEffect(() => {
    if (tab !== "subscriptions") {
      return;
    }
    void loadSubscriptions();
  }, [tab, loadSubscriptions]);

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
      label: "订阅树（新策略）",
      children: (
        <>
          <Space style={{ width: "100%", marginBottom: 12 }} wrap>
            <Select
              style={{ width: 260 }}
              placeholder="选择项目"
              value={subProjectID || undefined}
              options={projects.map((p) => ({ label: p.name, value: p.id }))}
              onChange={(v) => setSubProjectID(Number(v) || 0)}
              showSearch
              filterOption={(input, option) => String(option?.label ?? "").toLowerCase().includes(input.toLowerCase())}
            />
            <Button icon={<ReloadOutlined />} loading={subLoading} onClick={() => void loadSubscriptions()}>
              刷新
            </Button>
            <Button onClick={() => void migrateOldPolicies()}>迁移旧策略</Button>
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
            <Button type="primary" disabled={!subProjectID} onClick={() => void saveSubscription()}>
              保存
            </Button>
          </Space>
          <div style={{ display: "grid", gridTemplateColumns: "360px 1fr", gap: 12, alignItems: "start" }}>
            <Card size="small" title="订阅树" loading={subLoading} styles={{ body: { padding: 8 } }}>
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
            <Card size="small" title={selectedSubscriptionNode ? `编辑节点：${selectedSubscriptionNode.name}` : "选择节点进行编辑"}>
              <Form form={subForm} layout="vertical">
                <Form.Item name="id" hidden>
                  <Input />
                </Form.Item>
                <Form.Item name="parent_id" hidden>
                  <Input />
                </Form.Item>
                <Form.Item name="name" label="节点名称" rules={[{ required: true }]}>
                  <Input />
                </Form.Item>
                <Form.Item name="code" label="节点编码">
                  <Input />
                </Form.Item>
                <Space wrap style={{ width: "100%" }}>
                  <Form.Item name="enabled" label="启用" valuePropName="checked" style={{ marginBottom: 0 }}>
                    <Switch />
                  </Form.Item>
                  <Form.Item name="continue" label="继续匹配子节点" valuePropName="checked" style={{ marginBottom: 0 }}>
                    <Switch />
                  </Form.Item>
                  <Form.Item name="notify_resolved" label="恢复通知" valuePropName="checked" style={{ marginBottom: 0 }}>
                    <Switch />
                  </Form.Item>
                  <Form.Item name="silence_seconds" label="静默(s)" style={{ marginBottom: 0 }}>
                    <InputNumber min={0} />
                  </Form.Item>
                </Space>
                <Form.Item name="match_severity" label="匹配级别（可选）">
                  <Input placeholder="critical/warning/info..." />
                </Form.Item>
                <Form.Item name="receiver_group_ids" label="接收组" rules={[{ required: true, message: "至少选择一个接收组" }]}>
                  <Select mode="multiple" options={receiverGroupOptions} placeholder="选择接收组（先在接收组里配置通道）" />
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
            message="历史告警与「当前活跃」说明"
            description={
              <>
                <p style={{ marginBottom: 8 }}>
                  本表为<strong>通知投递审计</strong>（每次向通道外发或抑制链路写入的记录），对应 WatchAlert
                  文档中的<strong>历史告警 / 推送结果查询</strong>语义；可按 firing / resolved、监控链路、订阅命中等筛选。
                </p>
                <p style={{ marginBottom: 0 }}>
                  <strong>Prometheus 当前仍在触发的活跃告警</strong>请在「告警监控平台 → PromQL」使用数据源联查（如「活跃告警」）或自建
                  PromQL；平台内规则触发明细见「监控规则」与下方统计卡片。
                </p>
                <p style={{ marginTop: 8, marginBottom: 0 }}>
                  设计参考：{" "}
                  <Typography.Link href="https://cairry.github.io/docs/" target="_blank" rel="noreferrer">
                    WatchAlert 功能介绍
                  </Typography.Link>
                  。
                </p>
              </>
            }
          />
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
              placeholder="数据源 / 来源（WatchAlert 式）"
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
            scroll={{ x: 1640 }}
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
                render: (v: boolean) => (v ? <Tag color="success">成功</Tag> : <Tag color="error">失败</Tag>),
              },
              { title: "HTTP", dataIndex: "httpStatusCode", width: 80 },
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

