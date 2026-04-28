import { DeleteOutlined, EditOutlined, MinusCircleOutlined, PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import { Alert, AutoComplete, Button, Card, Drawer, Form, Input, InputNumber, Popconfirm, Select, Space, Statistic, Switch, Table, Tabs, Tag, Typography, message } from "antd";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  createAlertPolicy,
  deleteAlertPolicy,
  getAlertHistoryStats,
  listAlertChannels,
  listAlertEvents,
  listAlertPolicies,
  sendAlertmanagerWebhook,
  type AlertEventItem,
  type AlertPolicyItem,
  updateAlertPolicy,
} from "../services/alerts";
import { stringifyPrettyJSON } from "../services/alert-mappers";
import { useDictOptions } from "../hooks/use-dict-options";
import { formatDateTime } from "../utils/format";

export type AlertConfigTab = "policies" | "history";

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

type PolicyTemplateKey = "prod_critical_all" | "prod_warning_wecom_email" | "prod_info_email";
type MatcherPair = { key: string; value: string };
const policyTemplates: Record<
  PolicyTemplateKey,
  {
    name: string;
    description: string;
    enabled: boolean;
    priority: number;
    notify_resolved: boolean;
    silence_seconds: number;
    match_labels_json: string;
    match_regex_json: string;
    channelKeywords: string[];
  }
> = {
  prod_critical_all: {
    name: "prod-critical-all",
    description: "生产 critical 告警发送到钉钉+企业微信+邮件",
    enabled: true,
    priority: 10,
    notify_resolved: true,
    silence_seconds: 60,
    match_labels_json: JSON.stringify({ cluster: "prodK8s", severity: "critical" }, null, 2),
    match_regex_json: JSON.stringify({}, null, 2),
    channelKeywords: ["ding", "钉钉", "wecom", "企微", "企业微信", "email", "邮件", "mail"],
  },
  prod_warning_wecom_email: {
    name: "prod-warning-wecom-email",
    description: "生产 warning 告警发送到企业微信+邮件",
    enabled: true,
    priority: 20,
    notify_resolved: true,
    silence_seconds: 120,
    match_labels_json: JSON.stringify({ cluster: "prodK8s", severity: "warning" }, null, 2),
    match_regex_json: JSON.stringify({}, null, 2),
    channelKeywords: ["wecom", "企微", "企业微信", "email", "邮件", "mail"],
  },
  prod_info_email: {
    name: "prod-info-email",
    description: "生产 info 告警仅发送邮件",
    enabled: true,
    priority: 30,
    notify_resolved: true,
    silence_seconds: 180,
    match_labels_json: JSON.stringify({ cluster: "prodK8s", severity: "info" }, null, 2),
    match_regex_json: JSON.stringify({}, null, 2),
    channelKeywords: ["email", "邮件", "mail"],
  },
};

function describeAlertEvent(row: AlertEventItem): string {
  const reason = String(row.error_message || "").trim();
  if (row.success) {
    if (reason === "silence_suppressed") return "已命中平台静默，告警写入历史但未向通道发送。";
    if (reason === "policy_suppressed") return "已命中策略静默窗口，本次不再重复外发。";
    if (reason === "dedup_suppressed") return "已命中指纹去重，本次告警被去重抑制。";
    if (reason === "aggregate_suppressed") return "已命中 firing 聚合窗口，本次被汇总抑制。";
    if (reason === "resolved_aggregate_suppressed") return "已命中恢复聚合窗口，本次恢复通知被汇总抑制。";
    if (row.channel_name?.includes("静默抑制")) return "平台在分发前拦截了本次告警。";
    return "告警已完成分发并写入历史记录。";
  }
  if (reason === "no_enabled_channels") return "当前没有启用的通知通道，因此仅记录历史。";
  if (reason === "no_channel_matched") return "有通道存在，但本次告警未匹配到可发送通道。";
  if (reason === "no_channel_matched_policy") return "命中了策略，但策略绑定的通道未命中或不可用。";
  if (reason) return `发送失败：${reason}`;
  return "告警进入了平台链路，但未获取到更多说明。";
}

function summarizeAlertHint(row: AlertEventItem): string {
  const reason = String(row.error_message || "").trim();
  if (!row.success) return "-";
  if (!reason) return "-";
  if (reason === "silence_suppressed") return "已命中平台静默，通知已拦截";
  if (reason === "policy_suppressed") return "已命中策略静默窗口，本次未外发";
  if (reason === "dedup_suppressed") return "已触发去重策略，本次不再重复发送";
  if (reason === "aggregate_suppressed") return "已命中 firing 聚合窗口";
  if (reason === "resolved_aggregate_suppressed") return "已命中恢复聚合窗口";
  if (/suppressed/i.test(reason)) return "已被系统策略抑制，本次未发送";
  return reason;
}

function parseMatcherJSONToPairs(raw?: string): MatcherPair[] {
  const s = String(raw || "").trim();
  if (!s) return [];
  try {
    const parsed = JSON.parse(s) as Record<string, unknown>;
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) return [];
    return Object.entries(parsed)
      .map(([key, value]) => ({ key: String(key || "").trim(), value: String(value ?? "").trim() }))
      .filter((it) => it.key && it.value);
  } catch {
    return [];
  }
}

function normalizeMatcherPairs(raw: unknown): MatcherPair[] {
  if (!Array.isArray(raw)) return [];
  return raw
    .map((it) => {
      const row = it as Partial<MatcherPair>;
      return { key: String(row?.key || "").trim(), value: String(row?.value || "").trim() };
    })
    .filter((it) => it.key && it.value);
}

function matcherPairsToJSON(raw: unknown): string {
  const pairs = normalizeMatcherPairs(raw);
  const obj: Record<string, string> = {};
  for (const it of pairs) obj[it.key] = it.value;
  return stringifyPrettyJSON(obj, "{}");
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
  }>();

  const [channels, setChannels] = useState<Array<{ id: number; name: string }>>([]);

  const [policyLoading, setPolicyLoading] = useState(false);
  const [policyList, setPolicyList] = useState<AlertPolicyItem[]>([]);
  const [policyEditorOpen, setPolicyEditorOpen] = useState(false);
  const [policySubmitting, setPolicySubmitting] = useState(false);
  const [currentPolicy, setCurrentPolicy] = useState<AlertPolicyItem | null>(null);
  const [policyForm] = Form.useForm();
  const [policyTemplate, setPolicyTemplate] = useState<PolicyTemplateKey>("prod_warning_wecom_email");

  const [eventsLoading, setEventsLoading] = useState(false);
  const [events, setEvents] = useState<AlertEventItem[]>([]);
  const [eventsPage, setEventsPage] = useState(1);
  const [eventsPageSize, setEventsPageSize] = useState(10);
  const [eventsTotal, setEventsTotal] = useState(0);
  const [eventKeyword, setEventKeyword] = useState("");
  const [eventAlertIP, setEventAlertIP] = useState("");
  const [eventMonitorPipeline, setEventMonitorPipeline] = useState("");
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
  const policyTemplateOptions = useMemo(
    () => [
      { label: "prod-critical-all（钉钉+企微+邮件）", value: "prod_critical_all" },
      { label: "prod-warning-wecom-email（企微+邮件）", value: "prod_warning_wecom_email" },
      { label: "prod-info-email（仅邮件）", value: "prod_info_email" },
    ],
    [],
  );

  const alertIPOptions = useMemo(() => {
    const fromPage = (events ?? [])
      .map((it) => String(it.alert_ip || "").trim())
      .filter(Boolean);
    const merged = Array.from(new Set([...fromPage])).sort((a, b) => a.localeCompare(b, "zh-CN"));
    return merged.map((v) => ({ label: v, value: v }));
  }, [events]);

  const pipelineOptions = useMemo(() => {
    const fromStats = (stats?.monitor_pipeline_values ?? []).map((v) => String(v).trim()).filter(Boolean);
    const fromPage = (events ?? []).map((it) => (it.monitor_pipeline || "").trim()).filter(Boolean);
    const merged = Array.from(new Set([...fromStats, ...fromPage])).sort((a, b) => a.localeCompare(b, "zh-CN"));
    return merged.map((v) => ({
      label: v === "platform" ? `${v}（平台规则）` : v === "prometheus" ? `${v}（Prometheus+YAML）` : v,
      value: v,
    }));
  }, [stats?.monitor_pipeline_values, events]);
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

  async function loadPolicies() {
    setPolicyLoading(true);
    try {
      const res = await listAlertPolicies({ page: 1, page_size: 100 });
      setPolicyList(res.list ?? []);
    } finally {
      setPolicyLoading(false);
    }
  }

  const loadEvents = useCallback(
    async (page: number, pageSize: number) => {
      setEventsLoading(true);
      try {
        const res = await listAlertEvents({
          page,
          page_size: pageSize,
          keyword: eventKeyword.trim() || undefined,
          alert_ip: eventAlertIP.trim() || undefined,
          monitor_pipeline: eventMonitorPipeline.trim() || undefined,
          group_key: eventGroupKey.trim() || undefined,
        });
        setEvents(res.list ?? []);
        setEventsTotal(res.total ?? 0);
        setEventsPage(res.page ?? page);
        setEventsPageSize(res.page_size ?? pageSize);
      } finally {
        setEventsLoading(false);
      }
    },
    [eventKeyword, eventAlertIP, eventMonitorPipeline, eventGroupKey],
  );

  useEffect(() => {
    void loadBase();
    void loadPolicies();
  }, []);

  useEffect(() => {
    if (tab !== "history") {
      return;
    }
    const delay = eventKeyword || eventAlertIP || eventGroupKey ? 300 : 0;
    const timer = window.setTimeout(() => {
      void loadEvents(1, eventsPageSizeRef.current);
    }, delay);
    return () => window.clearTimeout(timer);
  }, [tab, eventKeyword, eventAlertIP, eventMonitorPipeline, eventGroupKey, loadEvents]);

  function openCreatePolicy() {
    setCurrentPolicy(null);
    setPolicyTemplate("prod_warning_wecom_email");
    policyForm.setFieldsValue({
      name: "",
      description: "",
      enabled: true,
      priority: 100,
      notify_resolved: true,
      silence_seconds: 0,
      match_labels_json: "{}",
      match_regex_json: "{}",
      match_labels_pairs: [],
      match_regex_pairs: [],
      channels_json_array: [],
    });
    setPolicyEditorOpen(true);
  }

  function openEditPolicy(row: AlertPolicyItem) {
    setCurrentPolicy(row);
    setPolicyTemplate("prod_warning_wecom_email");
    policyForm.setFieldsValue({
      ...row,
      channels_json_array: row.channel_ids ?? [],
      match_labels_json: stringifyPrettyJSON(row.match_labels ?? {}, "{}"),
      match_regex_json: stringifyPrettyJSON(row.match_regex ?? {}, "{}"),
      match_labels_pairs: parseMatcherJSONToPairs(row.match_labels_json),
      match_regex_pairs: parseMatcherJSONToPairs(row.match_regex_json),
    });
    setPolicyEditorOpen(true);
  }

  function applyPolicyTemplate(key: PolicyTemplateKey) {
    const tpl = policyTemplates[key];
    if (!tpl) return;
    const selectedChannelIDs = channelOptions
      .filter((opt) => {
        const name = String(opt.label || "").toLowerCase();
        return tpl.channelKeywords.some((kw) => name.includes(String(kw).toLowerCase()));
      })
      .map((opt) => Number(opt.value));
    policyForm.setFieldsValue({
      name: tpl.name,
      description: tpl.description,
      enabled: tpl.enabled,
      priority: tpl.priority,
      notify_resolved: tpl.notify_resolved,
      silence_seconds: tpl.silence_seconds,
      match_labels_json: tpl.match_labels_json,
      match_regex_json: tpl.match_regex_json,
      match_labels_pairs: parseMatcherJSONToPairs(tpl.match_labels_json),
      match_regex_pairs: parseMatcherJSONToPairs(tpl.match_regex_json),
      channels_json_array: selectedChannelIDs,
    });
    if (selectedChannelIDs.length > 0) {
      message.success(`已套用策略模板，并匹配到 ${selectedChannelIDs.length} 个通道`);
      return;
    }
    message.warning("已套用策略模板，但未自动匹配到通道，请手动选择通知通道");
  }

  async function submitPolicy() {
    const v = await policyForm.validateFields();
    setPolicySubmitting(true);
    try {
      const payload = {
        name: String(v.name || "").trim(),
        description: String(v.description || "").trim(),
        enabled: !!v.enabled,
        priority: Number(v.priority || 100),
        notify_resolved: !!v.notify_resolved,
        silence_seconds: Number(v.silence_seconds || 0),
        match_labels_json: matcherPairsToJSON(v.match_labels_pairs),
        match_regex_json: matcherPairsToJSON(v.match_regex_pairs),
        channels_json: JSON.stringify(v.channels_json_array ?? []),
      };
      if (currentPolicy) {
        await updateAlertPolicy(currentPolicy.id, payload);
        message.success("策略已更新");
      } else {
        await createAlertPolicy(payload);
        message.success("策略已创建");
      }
      setPolicyEditorOpen(false);
      await loadPolicies();
    } finally {
      setPolicySubmitting(false);
    }
  }

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
      key: "policies",
      label: "告警策略配置",
      children: (
        <>
          <div style={{ display: "flex", justifyContent: "space-between", marginBottom: 12 }}>
            <Space />
            <Button type="primary" icon={<PlusOutlined />} onClick={openCreatePolicy}>新增策略</Button>
          </div>
          <Table
            rowKey="id"
            loading={policyLoading}
            dataSource={policyList}
            pagination={{ pageSize: 10, showSizeChanger: true, pageSizeOptions: [10, 20, 50, 100], showQuickJumper: true }}
            columns={[
              { title: "策略名", dataIndex: "name", width: 180 },
              { title: "优先级", dataIndex: "priority", width: 90 },
              { title: "描述", dataIndex: "description", ellipsis: true },
              { title: "恢复通知", dataIndex: "notify_resolved", width: 100, render: (v: boolean) => (v ? <Tag color="success">是</Tag> : <Tag>否</Tag>) },
              { title: "静默(s)", dataIndex: "silence_seconds", width: 90 },
              { title: "状态", dataIndex: "enabled", width: 90, render: (v: boolean) => (v ? <Tag color="success">启用</Tag> : <Tag>停用</Tag>) },
              {
                title: "操作",
                width: 180,
                render: (_: unknown, row: AlertPolicyItem) => (
                  <Space>
                    <Button type="link" icon={<EditOutlined />} onClick={() => openEditPolicy(row)}>编辑</Button>
                    <Popconfirm title="确认删除策略？" onConfirm={() => void deleteAlertPolicy(row.id).then(() => { message.success("已删除"); void loadPolicies(); })}>
                      <Button type="link" danger icon={<DeleteOutlined />}>删除</Button>
                    </Popconfirm>
                  </Space>
                ),
              },
            ]}
          />
        </>
      ),
    },
    {
      key: "history",
      label: "历史告警记录",
      children: (
        <>
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
              style={{ width: 240 }}
              placeholder="监控链路（区分双来源）"
              value={eventMonitorPipeline || undefined}
              options={pipelineOptions}
              showSearch
              allowClear
              onChange={(v) => setEventMonitorPipeline((v as string) || "")}
              filterOption={(input, option) => String(option?.label ?? "").toLowerCase().includes(input.toLowerCase())}
            />
            <Input
              style={{ width: 220 }}
              placeholder="group_key"
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
              { title: "标题", dataIndex: "title", width: 220, ellipsis: true },
              { title: "告警IP", dataIndex: "alert_ip", width: 160, ellipsis: true, render: (v: string) => v || "-" },
              {
                title: "监控链路",
                dataIndex: "monitor_pipeline",
                width: 130,
                render: (v: string) => {
                  if (v === "platform") return <Tag color="purple">platform</Tag>;
                  if (v === "prometheus") return <Tag color="blue">prometheus</Tag>;
                  return v ? <Tag>{v}</Tag> : <span>-</span>;
                },
              },
              { title: "GroupKey", dataIndex: "group_key", width: 140, ellipsis: true, render: (v: string) => v || "-" },
              { title: "来源", dataIndex: "source", width: 120 },
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
                dataIndex: "matched_policy_names",
                width: 200,
                ellipsis: true,
                render: (_: string, row: AlertEventItem) => (row.matched_policy_name_list?.length ? row.matched_policy_name_list.join(", ") : "-"),
              },
              { title: "通道", dataIndex: "channel_name", width: 160, ellipsis: true },
              {
                title: "发送结果",
                dataIndex: "success",
                width: 100,
                render: (v: boolean) => (v ? <Tag color="success">成功</Tag> : <Tag color="error">失败</Tag>),
              },
              { title: "HTTP", dataIndex: "http_status_code", width: 80 },
              {
                title: "提示",
                key: "hint",
                width: 160,
                ellipsis: true,
                render: (_: unknown, row: AlertEventItem) => {
                  if (row.success && row.error_message) {
                    const msg = summarizeAlertHint(row);
                    return (
                      <Typography.Text type="secondary" ellipsis={{ tooltip: msg }}>
                        {msg}
                      </Typography.Text>
                    );
                  }
                  return "-";
                },
              },
              {
                title: "链路说明",
                key: "flow_explain",
                width: 260,
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
              {
                title: "错误信息",
                dataIndex: "error_message",
                width: 160,
                ellipsis: true,
                render: (_: unknown, row: AlertEventItem) => {
                  const v = row.success ? "" : row.error_message;
                  return v ? (
                    <Typography.Text type="danger" ellipsis={{ tooltip: v }}>
                      {v}
                    </Typography.Text>
                  ) : (
                    "-"
                  );
                },
              },
              { title: "时间", dataIndex: "created_at", width: 170, render: (v: string) => formatDateTime(v) },
            ]}
          />
        </>
      ),
    },
  ] as const;

  const activeContent = tabItems.find((item) => item.key === tab)?.children ?? null;

  const showOverviewAndDebug = !(embedded && hideTabs && tab === "policies");

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
        title={currentPolicy ? "编辑告警策略" : "新增告警策略"}
        placement="right"
        width={860}
        open={policyEditorOpen}
        onClose={() => setPolicyEditorOpen(false)}
        destroyOnClose
        styles={{ body: { paddingBottom: 24 } }}
        extra={
          <Space>
            <Button onClick={() => setPolicyEditorOpen(false)}>取消</Button>
            <Button type="primary" loading={policySubmitting} onClick={() => void submitPolicy()}>
              确定
            </Button>
          </Space>
        }
      >
        <Form form={policyForm} layout="vertical">
          <Card size="small" title="策略模板（可选）" style={{ marginBottom: 12 }}>
            <Space wrap>
              <Select
                style={{ width: 280 }}
                value={policyTemplate}
                options={policyTemplateOptions}
                onChange={(v) => setPolicyTemplate(v as PolicyTemplateKey)}
              />
              <Button onClick={() => applyPolicyTemplate(policyTemplate)}>套用模板</Button>
            </Space>
          </Card>
          <Form.Item name="name" label="策略名称" rules={[{ required: true, message: "请输入策略名称" }]}><Input /></Form.Item>
          <Form.Item name="description" label="描述"><Input /></Form.Item>
          <Space style={{ width: "100%" }} size="large">
            <Form.Item name="priority" label="优先级"><InputNumber min={1} max={9999} /></Form.Item>
            <Form.Item name="silence_seconds" label="静默窗口(s)"><InputNumber min={0} max={86400} /></Form.Item>
            <Form.Item name="enabled" label="启用" valuePropName="checked"><Switch /></Form.Item>
            <Form.Item name="notify_resolved" label="恢复通知" valuePropName="checked"><Switch /></Form.Item>
          </Space>
          <Form.Item name="channels_json_array" label="通知通道">
            <Select mode="multiple" allowClear options={channelOptions} placeholder="选择命中的通知通道" />
          </Form.Item>
          <Alert
            type="info"
            showIcon
            style={{ marginBottom: 12 }}
            message="关于 monitor_pipeline（避免策略误匹配）"
            description={
              <span>
                平台会把告警标记为 <Typography.Text code>monitor_pipeline</Typography.Text>：
                <Typography.Text code>prometheus</Typography.Text> 表示来自 Alertmanager Webhook 入站，
                <Typography.Text code>platform</Typography.Text> 表示来自平台监控规则评估。通常策略无需强制写该字段；
                如果在 <Typography.Text code>match_labels_json</Typography.Text> 写了 <Typography.Text code>{`{\"monitor_pipeline\":\"platform\"}`}</Typography.Text>，
                则只会命中平台规则告警，Webhook 入站告警会匹配不到。
              </span>
            }
          />
          <Card size="small" title="match labels（精确匹配）" style={{ marginBottom: 12 }}>
            <Form.List name="match_labels_pairs">
              {(fields, { add, remove }) => (
                <Space direction="vertical" style={{ width: "100%" }} size={8}>
                  {fields.map((field) => (
                    <Space key={field.key} align="start" wrap style={{ width: "100%" }}>
                      <Form.Item name={[field.name, "key"]} style={{ minWidth: 240, marginBottom: 0 }}>
                        <AutoComplete
                          options={labelKeyAutoCompleteOptions}
                          placeholder="标签键（可选可输）"
                          filterOption={(input, option) => String(option?.value ?? "").toLowerCase().includes(input.toLowerCase())}
                          allowClear
                        />
                      </Form.Item>
                      <Form.Item name={[field.name, "value"]} style={{ minWidth: 260, marginBottom: 0 }}>
                        <Input placeholder="值（精确匹配）" />
                      </Form.Item>
                      <Button icon={<MinusCircleOutlined />} onClick={() => remove(field.name)} />
                    </Space>
                  ))}
                  <Button type="dashed" icon={<PlusOutlined />} onClick={() => add({ key: "", value: "" })}>
                    新增精确匹配项
                  </Button>
                </Space>
              )}
            </Form.List>
          </Card>
          <Card size="small" title="match regex（正则匹配）" style={{ marginBottom: 12 }}>
            <Form.List name="match_regex_pairs">
              {(fields, { add, remove }) => (
                <Space direction="vertical" style={{ width: "100%" }} size={8}>
                  {fields.map((field) => (
                    <Space key={field.key} align="start" wrap style={{ width: "100%" }}>
                      <Form.Item name={[field.name, "key"]} style={{ minWidth: 240, marginBottom: 0 }}>
                        <AutoComplete
                          options={labelKeyAutoCompleteOptions}
                          placeholder="标签键（可选可输）"
                          filterOption={(input, option) => String(option?.value ?? "").toLowerCase().includes(input.toLowerCase())}
                          allowClear
                        />
                      </Form.Item>
                      <Form.Item name={[field.name, "value"]} style={{ minWidth: 260, marginBottom: 0 }}>
                        <Input placeholder='值（正则），例如 ^(prod|stg)-.*$' />
                      </Form.Item>
                      <Button icon={<MinusCircleOutlined />} onClick={() => remove(field.name)} />
                    </Space>
                  ))}
                  <Button type="dashed" icon={<PlusOutlined />} onClick={() => add({ key: "", value: "" })}>
                    新增正则匹配项
                  </Button>
                </Space>
              )}
            </Form.List>
          </Card>
        </Form>
      </Drawer>

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

