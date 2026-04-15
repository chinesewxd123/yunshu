import {
  CheckCircleOutlined,
  DeleteOutlined,
  EditOutlined,
  EyeOutlined,
  FileTextOutlined,
  PlusOutlined,
  StopOutlined,
  TagsOutlined,
} from "@ant-design/icons";
import {
  Button,
  Descriptions,
  Divider,
  Input,
  Modal,
  Popconfirm,
  Progress,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  message,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { useState } from "react";
import { useKeyValueViewer } from "../components/k8s/key-value-viewer";
import { YamlCrudPage } from "../components/k8s/yaml-crud-page";
import { getNodeDetail, listNodes, replaceNodeTaints, setNodeSchedulability, type NodeTaintInput } from "../services/nodes";

const TAINT_EFFECT_OPTIONS = [
  { value: "NoSchedule", label: "NoSchedule（不调度新 Pod）" },
  { value: "PreferNoSchedule", label: "PreferNoSchedule（尽量不调度）" },
  { value: "NoExecute", label: "NoExecute（驱逐无容忍 Pod）" },
];

type TaintRow = { key: string; value: string; effect: string };

type Item = {
  name: string;
  status: string;
  unschedulable?: boolean;
  roles?: string[];
  kernel: string;
  kubelet: string;
  os_image: string;
  container_runtime: string;
  architecture: string;
  internal_ip?: string;
  creation_time: string;
  age?: string;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  taints?: Array<{ key: string; value?: string; effect?: string; time_added?: string }>;
  pod_count?: number;
  pod_capacity?: number;
  pod_usage?: string;
  pod_usage_percent?: number;
  cpu_usage?: string;
  cpu_requests?: string;
  cpu_limits?: string;
  mem_usage?: string;
  mem_requests?: string;
  mem_limits?: string;
  cpu_usage_percent?: number;
  mem_usage_percent?: number;
};

type Detail = {
  item: Item;
  addresses: Array<{ type: string; address: string }>;
  conditions: Array<{
    type: string;
    status: string;
    reason?: string;
    message?: string;
    last_heartbeat_time?: string;
    last_transition_time?: string;
  }>;
  taints: Array<{ key: string; value?: string; effect?: string; time_added?: string }>;
  capacity: Record<string, string>;
  allocatable: Record<string, string>;
  yaml: string;
};

export function NodesPage() {
  const { renderKVIcon, viewer } = useKeyValueViewer({
    width: 760,
    compact: true,
    pageSize: 10,
    destroyOnClose: true,
    emptyText: (title) => `暂无${title}`,
  });

  const [taintOpen, setTaintOpen] = useState(false);
  const [taintSaving, setTaintSaving] = useState(false);
  const [taintNodeName, setTaintNodeName] = useState("");
  const [taintClusterId, setTaintClusterId] = useState(0);
  const [taintRows, setTaintRows] = useState<TaintRow[]>([]);
  const [taintReload, setTaintReload] = useState<(() => void) | null>(null);

  function openTaintEditor(clusterId: number, record: Item, reload: () => void) {
    setTaintClusterId(clusterId);
    setTaintNodeName(record.name);
    setTaintRows(
      (record.taints ?? []).map((t) => ({
        key: t.key,
        value: t.value ?? "",
        effect: t.effect && TAINT_EFFECT_OPTIONS.some((o) => o.value === t.effect) ? t.effect : "NoSchedule",
      })),
    );
    setTaintReload(() => reload);
    setTaintOpen(true);
  }

  async function submitTaints() {
    for (const r of taintRows) {
      if (!r.key.trim()) {
        message.warning("污点 key 不能为空");
        return;
      }
      if (!r.effect.trim()) {
        message.warning("请选择 effect");
        return;
      }
    }
    const taints: NodeTaintInput[] = taintRows.map((r) => ({
      key: r.key.trim(),
      value: r.value.trim() || undefined,
      effect: r.effect.trim(),
    }));
    setTaintSaving(true);
    try {
      await replaceNodeTaints(taintClusterId, taintNodeName, taints);
      message.success("污点已更新");
      setTaintOpen(false);
      taintReload?.();
    } finally {
      setTaintSaving(false);
    }
  }

  const columns: ColumnsType<Item> = [
    { title: "节点", dataIndex: "name", width: 220 },
    {
      title: "节点角色",
      dataIndex: "roles",
      width: 150,
      render: (v?: string[]) => (v?.length ? v.map((x) => <Tag key={x}>{x}</Tag>) : <span className="inline-muted">-</span>),
    },
    {
      title: "污点",
      key: "taints",
      width: 70,
      align: "center",
      render: (_, r) =>
        renderKVIcon(
          "查看污点",
          <EyeOutlined />,
          Object.fromEntries((r.taints ?? []).map((t, i) => [`${t.key}/${t.effect || "-"}#${i}`, `${t.value || "-"} @ ${t.time_added || "-"}`])),
        ),
    },
    { title: "状态", dataIndex: "status", width: 100, render: (v: string) => <Tag color={v === "Ready" ? "green" : "red"}>{v || "-"}</Tag> },
    {
      title: "调度",
      key: "sched",
      width: 100,
      render: (_: unknown, r: Item) =>
        r.unschedulable ? <Tag color="orange">禁止调度</Tag> : <Tag color="processing">可调度</Tag>,
    },
    { title: "IP", dataIndex: "internal_ip", width: 140 },
    {
      title: "标签",
      key: "labels",
      width: 70,
      align: "center",
      render: (_, r) => renderKVIcon("标签", <TagsOutlined />, r.labels),
    },
    {
      title: "注解",
      key: "annotations",
      width: 70,
      align: "center",
      render: (_, r) => renderKVIcon("注解", <FileTextOutlined />, r.annotations),
    },
    {
      title: "Pod数量",
      key: "pod_usage",
      width: 180,
      render: (_: unknown, r: Item) => {
        const p = typeof r.pod_usage_percent === "number" ? r.pod_usage_percent : undefined;
        if (!p || p <= 0) return <span className="inline-muted">{r.pod_usage || `${r.pod_count ?? 0}/-`}</span>;
        return <Progress percent={Math.min(100, Math.round(p))} size="small" format={() => `${r.pod_usage || "-"}`} />;
      },
    },
    {
      title: "CPU",
      key: "cpu_usage",
      width: 240,
      render: (_: unknown, r: Item) => {
        const p = typeof r.cpu_usage_percent === "number" ? r.cpu_usage_percent : undefined;
        const extra = `${r.cpu_requests || "-"}/${r.cpu_limits || "-"}`;
        if (!p || p <= 0) return <span className="inline-muted">{`${r.cpu_usage || "-"} | ${extra}`}</span>;
        return <Progress percent={Math.min(100, Math.round(p))} size="small" format={() => `${r.cpu_usage || "-"} | ${extra}`} />;
      },
    },
    {
      title: "内存",
      key: "mem_usage",
      width: 240,
      render: (_: unknown, r: Item) => {
        const p = typeof r.mem_usage_percent === "number" ? r.mem_usage_percent : undefined;
        const extra = `${r.mem_requests || "-"}/${r.mem_limits || "-"}`;
        if (!p || p <= 0) return <span className="inline-muted">{`${r.mem_usage || "-"} | ${extra}`}</span>;
        return <Progress percent={Math.min(100, Math.round(p))} size="small" format={() => `${r.mem_usage || "-"} | ${extra}`} />;
      },
    },
    { title: "K8s版本", dataIndex: "kubelet", width: 120 },
    { title: "系统镜像", dataIndex: "os_image", width: 220 },
    { title: "容器镜像", dataIndex: "architecture", width: 80 },
    { title: "操作系统", dataIndex: "kernel", width: 150 },
    { title: "存在时长", dataIndex: "age", width: 90, fixed: "right" },
  ];

  return (
    <>
      <YamlCrudPage<Item, Detail>
        title="Node 资源管理"
        needNamespace={false}
        disableMutations
        actionColumnWidth={380}
        columns={columns}
        api={{
          list: async ({ clusterId, keyword }) => await listNodes(clusterId, keyword),
          detail: async ({ clusterId, name }) => await getNodeDetail(clusterId, name),
        }}
        extraRowActions={(record, { clusterId, reload }) => (
          <Space size={0} wrap>
            <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openTaintEditor(clusterId, record, reload)}>
              污点
            </Button>
            {!record.unschedulable ? (
              <Popconfirm
                title="禁止调度？"
                description="新 Pod 将不会调度到该节点（已有 Pod 不受影响）。"
                onConfirm={async () => {
                  try {
                    await setNodeSchedulability(clusterId, record.name, true);
                    message.success("已禁止调度");
                    reload();
                  } catch {
                    /* http 拦截器已提示 */
                  }
                }}
              >
                <Button type="link" size="small" icon={<StopOutlined />}>
                  禁止调度
                </Button>
              </Popconfirm>
            ) : (
              <Popconfirm
                title="恢复调度？"
                description="取消 cordon 后，新 Pod 可以再次调度到该节点。"
                onConfirm={async () => {
                  try {
                    await setNodeSchedulability(clusterId, record.name, false);
                    message.success("已允许调度");
                    reload();
                  } catch {
                    /* http 拦截器已提示 */
                  }
                }}
              >
                <Button type="link" size="small" icon={<CheckCircleOutlined />}>
                  允许调度
                </Button>
              </Popconfirm>
            )}
          </Space>
        )}
        detailExtra={(d) => (
          <div>
            <Descriptions size="small" bordered column={2} style={{ marginBottom: 10 }}>
              <Descriptions.Item label="Node">{d.item.name}</Descriptions.Item>
              <Descriptions.Item label="状态">
                <Tag color={d.item.status === "Ready" ? "green" : "red"}>{d.item.status || "-"}</Tag>
              </Descriptions.Item>
              <Descriptions.Item label="调度" span={2}>
                {d.item.unschedulable ? <Tag color="orange">禁止调度（Cordon）</Tag> : <Tag color="processing">可调度</Tag>}
              </Descriptions.Item>
              <Descriptions.Item label="架构">{d.item.architecture || "-"}</Descriptions.Item>
              <Descriptions.Item label="内核">{d.item.kernel || "-"}</Descriptions.Item>
              <Descriptions.Item label="Kubelet">{d.item.kubelet || "-"}</Descriptions.Item>
              <Descriptions.Item label="容器运行时">{d.item.container_runtime || "-"}</Descriptions.Item>
            </Descriptions>
            <Typography.Paragraph type="secondary" style={{ marginBottom: 6 }}>
              地址：
              {d.addresses?.length ? (
                d.addresses.map((a) => (
                  <Tag key={`${a.type}-${a.address}`} style={{ marginLeft: 8 }}>
                    {a.type}:{a.address}
                  </Tag>
                ))
              ) : (
                " -"
              )}
            </Typography.Paragraph>
            <Divider style={{ margin: "10px 0" }} />
            <Typography.Text strong>资源（Capacity / Allocatable）</Typography.Text>
            <Table
              size="small"
              pagination={false}
              rowKey={(r) => r.name}
              style={{ marginTop: 8 }}
              dataSource={Object.keys({ ...(d.capacity || {}), ...(d.allocatable || {}) })
                .sort()
                .map((k) => ({
                  name: k,
                  capacity: d.capacity?.[k] || "-",
                  allocatable: d.allocatable?.[k] || "-",
                }))}
              columns={[
                { title: "资源", dataIndex: "name", width: 180 },
                { title: "Capacity", dataIndex: "capacity", width: 200 },
                { title: "Allocatable", dataIndex: "allocatable", width: 200 },
              ]}
            />
            <Divider style={{ margin: "10px 0" }} />
            <Typography.Text strong>污点（Taints）</Typography.Text>
            <Table
              size="small"
              pagination={false}
              rowKey={(r) => `${r.key}-${r.effect}-${r.value}`}
              style={{ marginTop: 8 }}
              dataSource={d.taints || []}
              locale={{ emptyText: "无污点" }}
              columns={[
                { title: "Key", dataIndex: "key", width: 200 },
                { title: "Value", dataIndex: "value", width: 200, render: (v?: string) => v || "-" },
                { title: "Effect", dataIndex: "effect", width: 140, render: (v?: string) => v || "-" },
                { title: "TimeAdded", dataIndex: "time_added", render: (v?: string) => v || "-" },
              ]}
            />
            <Divider style={{ margin: "10px 0" }} />
            <Typography.Text strong>条件（Conditions）</Typography.Text>
            <Table
              size="small"
              pagination={false}
              rowKey={(r) => `${r.type}-${r.last_transition_time || ""}`}
              style={{ marginTop: 8 }}
              dataSource={d.conditions || []}
              columns={[
                { title: "类型", dataIndex: "type", width: 180 },
                {
                  title: "状态",
                  dataIndex: "status",
                  width: 100,
                  render: (v: string) => <Tag color={v === "True" ? "green" : v === "False" ? "red" : "default"}>{v}</Tag>,
                },
                { title: "原因", dataIndex: "reason", width: 180, render: (v?: string) => v || "-" },
                { title: "心跳时间", dataIndex: "last_heartbeat_time", width: 170, render: (v?: string) => v || "-" },
                { title: "变化时间", dataIndex: "last_transition_time", width: 170, render: (v?: string) => v || "-" },
                { title: "消息", dataIndex: "message", render: (v?: string) => v || "-" },
              ]}
            />
          </div>
        )}
      />
      {viewer}

      <Modal
        title={`编辑污点 — ${taintNodeName || "-"}`}
        open={taintOpen}
        onCancel={() => setTaintOpen(false)}
        onOk={() => void submitTaints()}
        confirmLoading={taintSaving}
        width={720}
        destroyOnClose
      >
        <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
          保存后将<strong>替换</strong>该节点上的全部污点。留空列表可清空所有污点。
        </Typography.Paragraph>
        <Space direction="vertical" style={{ width: "100%" }}>
          <Button
            type="dashed"
            icon={<PlusOutlined />}
            onClick={() => setTaintRows((rows) => [...rows, { key: "", value: "", effect: "NoSchedule" }])}
            block
          >
            添加污点
          </Button>
          <Table
            size="small"
            pagination={false}
            rowKey={(_, i) => String(i)}
            dataSource={taintRows.map((r, i) => ({ ...r, _i: i }))}
            columns={[
              {
                title: "Key",
                dataIndex: "key",
                width: 200,
                render: (v: string, row: TaintRow & { _i: number }) => (
                  <Input
                    value={v}
                    placeholder="例如 node.kubernetes.io/unschedulable"
                    onChange={(e) => {
                      const i = row._i;
                      setTaintRows((prev) => prev.map((x, j) => (j === i ? { ...x, key: e.target.value } : x)));
                    }}
                  />
                ),
              },
              {
                title: "Value",
                dataIndex: "value",
                width: 160,
                render: (v: string, row: TaintRow & { _i: number }) => (
                  <Input
                    value={v}
                    placeholder="可空"
                    onChange={(e) => {
                      const i = row._i;
                      setTaintRows((prev) => prev.map((x, j) => (j === i ? { ...x, value: e.target.value } : x)));
                    }}
                  />
                ),
              },
              {
                title: "Effect",
                dataIndex: "effect",
                width: 220,
                render: (v: string, row: TaintRow & { _i: number }) => (
                  <Select
                    style={{ width: "100%" }}
                    value={v}
                    options={TAINT_EFFECT_OPTIONS}
                    onChange={(eff) => {
                      const i = row._i;
                      setTaintRows((prev) => prev.map((x, j) => (j === i ? { ...x, effect: eff } : x)));
                    }}
                  />
                ),
              },
              {
                title: "操作",
                key: "op",
                width: 72,
                render: (_: unknown, row: TaintRow & { _i: number }) => (
                  <Button
                    type="link"
                    danger
                    size="small"
                    icon={<DeleteOutlined />}
                    onClick={() => setTaintRows((prev) => prev.filter((_, j) => j !== row._i))}
                  />
                ),
              },
            ]}
          />
        </Space>
      </Modal>
    </>
  );
}
