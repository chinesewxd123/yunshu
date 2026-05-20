import { DeleteOutlined, EditOutlined, PlusOutlined, ReloadOutlined, SettingOutlined } from "@ant-design/icons";
import {
  Alert,
  Button,
  Card,
  Drawer,
  Form,
  Input,
  InputNumber,
  Popconfirm,
  Select,
  Space,
  Switch,
  Table,
  Tabs,
  Tag,
  Typography,
  message,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { getClusters, type ClusterItem } from "../services/clusters";
import {
  createK8sEventForwardRule,
  deleteK8sEventForwardRule,
  getK8sEventForwardSettings,
  listK8sEventForwardRules,
  updateK8sEventForwardRule,
  updateK8sEventForwardSettings,
  type K8sEventForwardRule,
  type K8sEventForwardRulePayload,
  type K8sEventForwardSetting,
} from "../services/k8s-event-forward";
import { formatDateTime } from "../utils/format";

const EMPTY_JSON_ARRAY = "[]";

type RuleFormValues = {
  name: string;
  description?: string;
  cluster_ids: number[];
  webhook_url?: string;
  enabled: boolean;
  rule_namespaces_json: string;
  rule_names_json: string;
  rule_reasons_json: string;
  rule_reverse: boolean;
};

function parseClusterIds(raw: string): number[] {
  return raw
    .split(",")
    .map((s) => Number(s.trim()))
    .filter((n) => Number.isFinite(n) && n > 0);
}

function clusterIdsToString(ids: number[]): string {
  return ids.filter((n) => n > 0).join(",");
}

function ruleToForm(row: K8sEventForwardRule): RuleFormValues {
  return {
    name: row.name,
    description: row.description || "",
    cluster_ids: parseClusterIds(row.cluster_ids || ""),
    webhook_url: row.webhook_url || "",
    enabled: row.enabled,
    rule_namespaces_json: row.rule_namespaces?.trim() || EMPTY_JSON_ARRAY,
    rule_names_json: row.rule_names?.trim() || EMPTY_JSON_ARRAY,
    rule_reasons_json: row.rule_reasons?.trim() || EMPTY_JSON_ARRAY,
    rule_reverse: Boolean(row.rule_reverse),
  };
}

function formToPayload(v: RuleFormValues): K8sEventForwardRulePayload {
  return {
    name: v.name.trim(),
    description: v.description?.trim(),
    cluster_ids: clusterIdsToString(v.cluster_ids),
    webhook_url: v.webhook_url?.trim(),
    enabled: v.enabled,
    rule_namespaces: v.rule_namespaces_json.trim() || EMPTY_JSON_ARRAY,
    rule_names: v.rule_names_json.trim() || EMPTY_JSON_ARRAY,
    rule_reasons: v.rule_reasons_json.trim() || EMPTY_JSON_ARRAY,
    rule_reverse: v.rule_reverse,
  };
}

function RulesPanel() {
  const [loading, setLoading] = useState(false);
  const [list, setList] = useState<K8sEventForwardRule[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [clusters, setClusters] = useState<ClusterItem[]>([]);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [editing, setEditing] = useState<K8sEventForwardRule | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [form] = Form.useForm<RuleFormValues>();

  const clusterNameById = useMemo(() => new Map(clusters.map((c) => [c.id, c.name])), [clusters]);

  const loadClusters = useCallback(async () => {
    try {
      const res = await getClusters({ page: 1, page_size: 200 });
      setClusters(res.list || []);
    } catch {
      setClusters([]);
    }
  }, []);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await listK8sEventForwardRules({ page, page_size: pageSize });
      setList(res.list || []);
      setTotal(res.total || 0);
    } catch (e) {
      message.error(String(e));
    } finally {
      setLoading(false);
    }
  }, [page, pageSize]);

  useEffect(() => {
    void loadClusters();
  }, [loadClusters]);

  useEffect(() => {
    void load();
  }, [load]);

  const openCreate = () => {
    setEditing(null);
    form.setFieldsValue({
      name: "",
      description: "",
      cluster_ids: [],
      webhook_url: "internal",
      enabled: true,
      rule_namespaces_json: EMPTY_JSON_ARRAY,
      rule_names_json: EMPTY_JSON_ARRAY,
      rule_reasons_json: EMPTY_JSON_ARRAY,
      rule_reverse: false,
    });
    setDrawerOpen(true);
  };

  const openEdit = (row: K8sEventForwardRule) => {
    setEditing(row);
    form.setFieldsValue(ruleToForm(row));
    setDrawerOpen(true);
  };

  const onSubmit = async () => {
    const values = await form.validateFields();
    setSubmitting(true);
    try {
      const payload = formToPayload(values);
      if (editing?.id) {
        await updateK8sEventForwardRule(editing.id, payload);
        message.success("规则已更新");
      } else {
        await createK8sEventForwardRule(payload);
        message.success("规则已创建");
      }
      setDrawerOpen(false);
      void load();
    } catch (e) {
      message.error(String(e));
    } finally {
      setSubmitting(false);
    }
  };

  const columns: ColumnsType<K8sEventForwardRule> = [
    { title: "名称", dataIndex: "name", width: 160, ellipsis: true },
    {
      title: "集群",
      dataIndex: "cluster_ids",
      render: (v: string) => {
        const ids = parseClusterIds(v || "");
        if (!ids.length) return <Typography.Text type="secondary">未配置</Typography.Text>;
        return (
          <Space size={[4, 4]} wrap>
            {ids.map((id) => (
              <Tag key={id}>{clusterNameById.get(id) ? `${clusterNameById.get(id)} (#${id})` : `#${id}`}</Tag>
            ))}
          </Space>
        );
      },
    },
    {
      title: "Webhook",
      dataIndex: "webhook_url",
      width: 140,
      ellipsis: true,
      render: (v?: string) => {
        const u = (v || "").trim();
        if (!u || u.toLowerCase() === "internal" || u.toLowerCase() === "alertmanager") {
          return <Tag color="blue">告警平台入站</Tag>;
        }
        return u;
      },
    },
    {
      title: "启用",
      dataIndex: "enabled",
      width: 72,
      render: (v: boolean) => (v ? <Tag color="success">是</Tag> : <Tag>否</Tag>),
    },
    {
      title: "反选",
      dataIndex: "rule_reverse",
      width: 72,
      render: (v?: boolean) => (v ? <Tag color="orange">是</Tag> : <Tag>否</Tag>),
    },
    {
      title: "更新时间",
      dataIndex: "updated_at",
      width: 168,
      render: (v?: string) => formatDateTime(v),
    },
    {
      title: "操作",
      key: "actions",
      width: 120,
      fixed: "right",
      render: (_, row) => (
        <Space>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openEdit(row)}>
            编辑
          </Button>
          <Popconfirm title="确定删除该规则？" onConfirm={() => void onDelete(row.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  async function onDelete(id: number) {
    try {
      await deleteK8sEventForwardRule(id);
      message.success("已删除");
      void load();
    } catch (e) {
      message.error(String(e));
    }
  }

  return (
    <>
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 12 }}
        message="转发说明"
        description={
          <span>
            K8s Event 标准类型为 <Typography.Text code>Normal</Typography.Text> / <Typography.Text code>Warning</Typography.Text>，仅{" "}
            <Typography.Text code>Normal</Typography.Text> 不转发；告警自动携带集群归属项目的 <Typography.Text code>project_id</Typography.Text>，走项目订阅路由（请在集群管理设置「归属项目」）。Webhook 留空或填{" "}
            <Typography.Text code>internal</Typography.Text> / <Typography.Text code>alertmanager</Typography.Text> 时，将 POST 到告警平台{" "}
            <Typography.Text code>/api/v1/alerts/webhook/alertmanager</Typography.Text>（鉴权使用数据字典{" "}
            <Typography.Text code>alert_webhook_token</Typography.Text>）。全局开关请在{" "}
            <Link to="/dict-entries?keyword=k8s_event_forward_">数据字典</Link> 维护{" "}
            <Typography.Text code>k8s_event_forward_*</Typography.Text>。
          </span>
        }
      />
      <Space style={{ marginBottom: 12 }}>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
          新建规则
        </Button>
        <Button icon={<ReloadOutlined />} onClick={() => void load()}>
          刷新
        </Button>
      </Space>
      <Table<K8sEventForwardRule>
        rowKey="id"
        loading={loading}
        columns={columns}
        dataSource={list}
        scroll={{ x: 960 }}
        pagination={{
          current: page,
          pageSize,
          total,
          showSizeChanger: true,
          onChange: (p, ps) => {
            setPage(p);
            setPageSize(ps);
          },
        }}
      />
      <Drawer
        title={editing ? "编辑转发规则" : "新建转发规则"}
        width={560}
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        destroyOnClose
        extra={
          <Space>
            <Button onClick={() => setDrawerOpen(false)}>取消</Button>
            <Button type="primary" loading={submitting} onClick={() => void onSubmit()}>
              保存
            </Button>
          </Space>
        }
      >
        <Form form={form} layout="vertical" autoComplete="off">
          <Form.Item name="name" label="规则名称" rules={[{ required: true, message: "请输入名称" }]}>
            <Input placeholder="例如 prod-warning-forward" />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={2} placeholder="可选" />
          </Form.Item>
          <Form.Item
            name="cluster_ids"
            label="目标集群"
            rules={[{ required: true, type: "array", min: 1, message: "请至少选择一个集群" }]}
          >
            <Select
              mode="multiple"
              placeholder="选择已接入集群"
              options={clusters.map((c) => ({ label: `${c.name} (#${c.id})`, value: c.id }))}
              optionFilterProp="label"
            />
          </Form.Item>
          <Form.Item
            name="webhook_url"
            label="Webhook 地址"
            extra="留空、internal 或 alertmanager 表示复用本机告警平台 Alertmanager Webhook"
          >
            <Input placeholder="internal" allowClear />
          </Form.Item>
          <Form.Item name="enabled" label="启用规则" valuePropName="checked">
            <Switch checkedChildren="启用" unCheckedChildren="停用" />
          </Form.Item>
          <Form.Item name="rule_reverse" label="反选过滤" valuePropName="checked" extra="开启后：不匹配下列条件的才转发">
            <Switch />
          </Form.Item>
          <Form.Item
            name="rule_namespaces_json"
            label="命名空间（JSON 数组，精确匹配）"
            rules={[{ required: true, message: "请输入 JSON" }]}
          >
            <Input.TextArea rows={3} placeholder='["kube-system","default"]' />
          </Form.Item>
          <Form.Item name="rule_names_json" label="资源名（JSON 数组，子串匹配）" rules={[{ required: true }]}>
            <Input.TextArea rows={3} placeholder='["nginx","api-"]' />
          </Form.Item>
          <Form.Item name="rule_reasons_json" label="原因/消息（JSON 数组，子串匹配 reason 或 message）" rules={[{ required: true }]}>
            <Input.TextArea rows={3} placeholder='["Failed","BackOff"]' />
          </Form.Item>
        </Form>
      </Drawer>
    </>
  );
}

function WorkerSettingsPanel() {
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [form] = Form.useForm<K8sEventForwardSetting>();

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const st = await getK8sEventForwardSettings();
      form.setFieldsValue(st);
    } catch (e) {
      message.error(String(e));
    } finally {
      setLoading(false);
    }
  }, [form]);

  useEffect(() => {
    void load();
  }, [load]);

  const onSave = async () => {
    const values = await form.validateFields();
    setSaving(true);
    try {
      await updateK8sEventForwardSettings(values);
      message.success("Worker 参数已保存（下一批处理周期内生效）");
    } catch (e) {
      message.error(String(e));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Card loading={loading} bordered={false} styles={{ body: { paddingTop: 8 } }}>
      <Alert
        type="warning"
        showIcon
        style={{ marginBottom: 16 }}
        message="与数据字典的关系"
        description={
          <span>
            下列参数与 <Link to="/dict-entries?keyword=k8s_event_forward_">数据字典</Link> 中{" "}
            <Typography.Text code>k8s_event_forward_*</Typography.Text> 为同一数据源，启动时覆盖{" "}
            <Typography.Text code>config.yaml</Typography.Text>。全局总开关请使用字典项{" "}
            <Typography.Text code>k8s_event_forward_enabled</Typography.Text>。
          </span>
        }
      />
      <Form form={form} layout="vertical" style={{ maxWidth: 480 }} autoComplete="off">
        <Form.Item name="id" hidden>
          <InputNumber />
        </Form.Item>
        <Form.Item name="watcher_buffer_size" label="监听通道缓冲" rules={[{ required: true }]}>
          <InputNumber min={100} max={100000} style={{ width: "100%" }} />
        </Form.Item>
        <Form.Item name="process_interval_seconds" label="批处理周期（秒）" rules={[{ required: true }]}>
          <InputNumber min={1} max={3600} style={{ width: "100%" }} />
        </Form.Item>
        <Form.Item name="batch_size" label="批大小" rules={[{ required: true }]}>
          <InputNumber min={1} max={500} style={{ width: "100%" }} />
        </Form.Item>
        <Form.Item name="max_retries" label="最大重试次数" rules={[{ required: true }]}>
          <InputNumber min={0} max={20} style={{ width: "100%" }} />
        </Form.Item>
        <Button type="primary" icon={<SettingOutlined />} loading={saving} onClick={() => void onSave()}>
          保存 Worker 参数
        </Button>
      </Form>
    </Card>
  );
}

export function K8sEventForwardPage() {
  return (
    <Card className="table-card" title="K8s Event 多集群转发">
      <Tabs
        items={[
          { key: "rules", label: "转发规则", children: <RulesPanel /> },
          { key: "settings", label: "Worker 参数", children: <WorkerSettingsPanel /> },
        ]}
      />
    </Card>
  );
}
