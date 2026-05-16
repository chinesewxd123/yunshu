import { DeleteOutlined, EditOutlined, PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import { Alert, Button, Drawer, Form, Input, InputNumber, Popconfirm, Space, Switch, Table, Tag, Typography, message } from "antd";
import { Link } from "react-router-dom";
import type { ColumnsType } from "antd/es/table";
import { useCallback, useEffect, useState } from "react";
import {
  createInhibitionRule,
  deleteInhibitionRule,
  listInhibitionRules,
  refreshInhibitionCache,
  updateInhibitionRule,
  type AlertInhibitionRule,
  type AlertInhibitionRulePayload,
} from "../services/alert-inhibition";
import { stringifyPrettyJSON } from "../services/alert-mappers";
import { formatDateTime } from "../utils/format";

const defaultSourceLabels = '{\n  "alertname": "NodeDown",\n  "severity": "critical"\n}';
const defaultTargetLabels = '{\n  "alertname": "PodCrashLooping"\n}';
const defaultEqualLabels = '["cluster", "namespace"]';

type FormValues = {
  name: string;
  description?: string;
  enabled: boolean;
  priority: number;
  duration_seconds: number;
  source_match_labels_json: string;
  source_match_regex_json: string;
  target_match_labels_json: string;
  target_match_regex_json: string;
  equal_labels_json: string;
};

function ruleToForm(row: AlertInhibitionRule): FormValues {
  return {
    name: row.name,
    description: row.description || "",
    enabled: row.enabled,
    priority: row.priority,
    duration_seconds: row.duration_seconds,
    source_match_labels_json: row.source_match_labels_json || stringifyPrettyJSON(row.source_match_labels ?? {}, "{}"),
    source_match_regex_json: row.source_match_regex_json || stringifyPrettyJSON(row.source_match_regex ?? {}, "{}"),
    target_match_labels_json: row.target_match_labels_json || stringifyPrettyJSON(row.target_match_labels ?? {}, "{}"),
    target_match_regex_json: row.target_match_regex_json || stringifyPrettyJSON(row.target_match_regex ?? {}, "{}"),
    equal_labels_json: row.equal_labels_json || stringifyPrettyJSON(row.equal_labels ?? [], "[]"),
  };
}

function formToPayload(v: FormValues): AlertInhibitionRulePayload {
  return {
    name: v.name.trim(),
    description: v.description?.trim(),
    enabled: v.enabled,
    priority: v.priority,
    duration_seconds: v.duration_seconds,
    source_match_labels_json: v.source_match_labels_json.trim() || "{}",
    source_match_regex_json: v.source_match_regex_json.trim() || "{}",
    target_match_labels_json: v.target_match_labels_json.trim() || "{}",
    target_match_regex_json: v.target_match_regex_json.trim() || "{}",
    equal_labels_json: v.equal_labels_json.trim() || "[]",
  };
}

export type AlertInhibitionPanelProps = {
  /** 告警监控平台顶栏项目筛选 */
  projectId?: number;
};

export function AlertInhibitionPanel({ projectId }: AlertInhibitionPanelProps = {}) {
  const [loading, setLoading] = useState(false);
  const [list, setList] = useState<AlertInhibitionRule[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [keyword, setKeyword] = useState("");
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [editing, setEditing] = useState<AlertInhibitionRule | null>(null);
  const [form] = Form.useForm<FormValues>();

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await listInhibitionRules({
        page,
        page_size: pageSize,
        keyword: keyword || undefined,
        project_id: projectId && projectId > 0 ? projectId : undefined,
      });
      setList(res.list || []);
      setTotal(res.total || 0);
    } catch (e) {
      message.error(String(e));
    } finally {
      setLoading(false);
    }
  }, [page, pageSize, keyword, projectId]);

  useEffect(() => {
    void load();
  }, [load]);

  const openCreate = () => {
    setEditing(null);
    form.setFieldsValue({
      name: "",
      description: "",
      enabled: true,
      priority: 100,
      duration_seconds: 3600,
      source_match_labels_json: defaultSourceLabels,
      source_match_regex_json: "{}",
      target_match_labels_json: defaultTargetLabels,
      target_match_regex_json: "{}",
      equal_labels_json: defaultEqualLabels,
    });
    setDrawerOpen(true);
  };

  const openEdit = (row: AlertInhibitionRule) => {
    setEditing(row);
    form.setFieldsValue(ruleToForm(row));
    setDrawerOpen(true);
  };

  const submit = async () => {
    const v = await form.validateFields();
    const payload = formToPayload(v);
    if (projectId && projectId > 0) {
      payload.project_id = projectId;
    }
    try {
      if (editing) {
        await updateInhibitionRule(editing.id, payload);
        message.success("已更新抑制规则");
      } else {
        await createInhibitionRule(payload);
        message.success("已创建抑制规则");
      }
      setDrawerOpen(false);
      await load();
      await refreshInhibitionCache().catch(() => undefined);
    } catch (e) {
      message.error(String(e));
    }
  };

  const columns: ColumnsType<AlertInhibitionRule> = [
    { title: "名称", dataIndex: "name", width: 160 },
    {
      title: "优先级",
      dataIndex: "priority",
      width: 80,
      render: (p: number) => <Tag>{p}</Tag>,
    },
    {
      title: "状态",
      dataIndex: "enabled",
      width: 80,
      render: (v: boolean) => (v ? <Tag color="green">启用</Tag> : <Tag>停用</Tag>),
    },
    {
      title: "源匹配（摘要）",
      key: "source",
      ellipsis: true,
      render: (_, r) => (
        <Typography.Text type="secondary" ellipsis>
          {r.source_match_labels_json || JSON.stringify(r.source_match_labels || {})}
        </Typography.Text>
      ),
    },
    {
      title: "目标匹配（摘要）",
      key: "target",
      ellipsis: true,
      render: (_, r) => (
        <Typography.Text type="secondary" ellipsis>
          {r.target_match_labels_json || JSON.stringify(r.target_match_labels || {})}
        </Typography.Text>
      ),
    },
    {
      title: "持续(s)",
      dataIndex: "duration_seconds",
      width: 90,
    },
    {
      title: "更新时间",
      dataIndex: "updated_at",
      width: 170,
      render: (t: string) => formatDateTime(t),
    },
    {
      title: "操作",
      key: "actions",
      width: 120,
      fixed: "right",
      render: (_, row) => (
        <Space>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openEdit(row)} />
          <Popconfirm title="确定删除该抑制规则？" onConfirm={() => void deleteInhibitionRule(row.id).then(() => load())}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <Space direction="vertical" style={{ width: "100%" }} size="middle">
      {projectId && projectId > 0 ? (
        <Alert type="info" showIcon message={`已按顶栏项目 #${projectId} 筛选抑制规则；新建规则将归属该项目`} />
      ) : (
        <Alert type="warning" showIcon message="未选择顶栏项目时显示全部抑制规则（含历史全局规则 project_id=0）" />
      )}
      <Alert
        type="info"
        showIcon
        message="告警抑制（Inhibition）≠ 平台静默 ≠ 订阅静默"
        description={
          <ul style={{ margin: "8px 0 0", paddingLeft: 18 }}>
            <li>
              <strong>抑制</strong>：类似 Alertmanager <Typography.Text code>inhibit_rules</Typography.Text>。当<strong>源告警</strong>（source 匹配）处于 firing 时，在 Redis 中记录；<strong>目标告警</strong>（target 匹配）且与源在 <Typography.Text code>equal_labels</Typography.Text> 上取值一致时，不向通道外发。源告警 resolved 后清除记录。
            </li>
            <li>
              <strong>平台静默</strong>：单条告警入站即拦截，见「平台静默」Tab（<Typography.Text code>silence_suppressed</Typography.Text>）。
            </li>
            <li>
              <strong>订阅静默</strong>：命中订阅节点后的 groupKey 窗口（<Typography.Text code>subscription_suppressed</Typography.Text>）。
            </li>
            <li>
              <strong>通知合并</strong>：firing 的 group_wait / repeat 等（<Typography.Text code>group_*_suppressed</Typography.Text>），与抑制无关。
            </li>
            <li>
              <strong>依赖</strong>：抑制运行时依赖 Redis；无 Redis 时规则可配置但不会生效。
            </li>
          </ul>
        }
      />
      <Space wrap>
        <Input.Search
          allowClear
          placeholder="搜索名称/描述"
          style={{ width: 240 }}
          onSearch={(v) => {
            setKeyword(v);
            setPage(1);
          }}
        />
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
          新建抑制规则
        </Button>
        <Button icon={<ReloadOutlined />} onClick={() => void load()}>
          刷新
        </Button>
        <Link to="/alert-monitor-platform?tab=history&event_category=inhibition">
          <Button>查看抑制留痕历史</Button>
        </Link>
      </Space>
      <Table
        rowKey="id"
        loading={loading}
        columns={columns}
        dataSource={list}
        scroll={{ x: 1100 }}
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
        title={editing ? `编辑抑制规则 #${editing.id}` : "新建抑制规则"}
        width={560}
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        destroyOnClose
        extra={
          <Space>
            <Button onClick={() => setDrawerOpen(false)}>取消</Button>
            <Button type="primary" onClick={() => void submit()}>
              保存
            </Button>
          </Space>
        }
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="规则名称" rules={[{ required: true, message: "请输入名称" }]}>
            <Input maxLength={128} />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={2} />
          </Form.Item>
          <Space>
            <Form.Item name="enabled" label="启用" valuePropName="checked">
              <Switch />
            </Form.Item>
            <Form.Item name="priority" label="优先级（越小越高）">
              <InputNumber min={1} max={9999} />
            </Form.Item>
            <Form.Item name="duration_seconds" label="源告警存活 TTL（秒）">
              <InputNumber min={60} step={60} />
            </Form.Item>
          </Space>
          <Form.Item
            name="source_match_labels_json"
            label="源告警 labels 精确匹配（JSON 对象）"
            extra="源告警 firing 且 labels 全部命中时，写入 Redis 作为抑制源"
          >
            <Input.TextArea rows={5} />
          </Form.Item>
          <Form.Item name="source_match_regex_json" label="源告警 labels 正则匹配（JSON 对象）">
            <Input.TextArea rows={3} />
          </Form.Item>
          <Form.Item
            name="target_match_labels_json"
            label="目标告警 labels 精确匹配（JSON 对象）"
            extra="被抑制的目标 firing 告警需命中此处条件"
          >
            <Input.TextArea rows={5} />
          </Form.Item>
          <Form.Item name="target_match_regex_json" label="目标告警 labels 正则匹配（JSON 对象）">
            <Input.TextArea rows={3} />
          </Form.Item>
          <Form.Item
            name="equal_labels_json"
            label="equal 标签（JSON 数组）"
            extra='源与目标在这些 label 上取值必须相同，例如 ["cluster","namespace"]'
          >
            <Input.TextArea rows={3} />
          </Form.Item>
        </Form>
      </Drawer>
    </Space>
  );
}
