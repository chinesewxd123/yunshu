import { EyeOutlined, FileTextOutlined, TagsOutlined } from "@ant-design/icons";
import { Button, Descriptions, Divider, Modal, Progress, Table, Tag, Tooltip, Typography } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useState } from "react";
import { YamlCrudPage } from "../components/k8s/yaml-crud-page";
import { getNodeDetail, listNodes } from "../services/nodes";

type Item = {
  name: string;
  status: string;
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
  const [kvOpen, setKvOpen] = useState(false);
  const [kvTitle, setKvTitle] = useState("详情");
  const [kvData, setKvData] = useState<Record<string, string>>({});

  const openKV = (title: string, data?: Record<string, string>) => {
    setKvTitle(title);
    setKvData(data ?? {});
    setKvOpen(true);
  };

  const renderKVIcon = (title: string, icon: JSX.Element, data?: Record<string, string>) => (
    <Tooltip title={title}>
      <Button type="link" size="small" icon={icon} onClick={() => openKV(title, data)} />
    </Tooltip>
  );

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
    { title: "名称", dataIndex: "name", width: 180 },
    { title: "存在时长", dataIndex: "age", width: 90, fixed: "right" },
  ];

  return (
    <>
    <YamlCrudPage<Item, Detail>
      title="Node 资源管理"
      needNamespace={false}
      disableMutations
      columns={columns}
      api={{
        list: async ({ clusterId, keyword }) => await listNodes(clusterId, keyword),
        detail: async ({ clusterId, name }) => await getNodeDetail(clusterId, name),
      }}
      detailExtra={(d) => (
        <div>
          <Descriptions size="small" bordered column={2} style={{ marginBottom: 10 }}>
            <Descriptions.Item label="Node">{d.item.name}</Descriptions.Item>
            <Descriptions.Item label="状态">
              <Tag color={d.item.status === "Ready" ? "green" : "red"}>{d.item.status || "-"}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="架构">{d.item.architecture || "-"}</Descriptions.Item>
            <Descriptions.Item label="内核">{d.item.kernel || "-"}</Descriptions.Item>
            <Descriptions.Item label="Kubelet">{d.item.kubelet || "-"}</Descriptions.Item>
            <Descriptions.Item label="容器运行时">{d.item.container_runtime || "-"}</Descriptions.Item>
          </Descriptions>
          <Typography.Paragraph type="secondary" style={{ marginBottom: 6 }}>
            地址：
            {d.addresses?.length ? d.addresses.map((a) => (
              <Tag key={`${a.type}-${a.address}`} style={{ marginLeft: 8 }}>
                {a.type}:{a.address}
              </Tag>
            )) : " -"}
          </Typography.Paragraph>
          <Divider style={{ margin: "10px 0" }} />
          <Typography.Text strong>资源（Capacity / Allocatable）</Typography.Text>
          <Table
            size="small"
            pagination={false}
            rowKey={(r) => r.name}
            style={{ marginTop: 8 }}
            dataSource={Object.keys({ ...(d.capacity || {}), ...(d.allocatable || {}) }).sort().map((k) => ({
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
              { title: "状态", dataIndex: "status", width: 100, render: (v: string) => <Tag color={v === "True" ? "green" : v === "False" ? "red" : "default"}>{v}</Tag> },
              { title: "原因", dataIndex: "reason", width: 180, render: (v?: string) => v || "-" },
              { title: "心跳时间", dataIndex: "last_heartbeat_time", width: 170, render: (v?: string) => v || "-" },
              { title: "变化时间", dataIndex: "last_transition_time", width: 170, render: (v?: string) => v || "-" },
              { title: "消息", dataIndex: "message", render: (v?: string) => v || "-" },
            ]}
          />
        </div>
      )}
    />
    <Modal
      title={kvTitle}
      open={kvOpen}
      onCancel={() => setKvOpen(false)}
      footer={null}
      width={760}
      destroyOnClose
    >
      <Table
        size="small"
        rowKey={(r) => r.key}
        pagination={{ pageSize: 10 }}
        dataSource={Object.entries(kvData).map(([key, value]) => ({ key, value }))}
        locale={{ emptyText: `暂无${kvTitle}` }}
        columns={[
          {
            title: "Key",
            dataIndex: "key",
            width: 280,
            render: (v: string) => (
              <Typography.Text copyable style={{ whiteSpace: "pre-wrap" }}>
                {v || "-"}
              </Typography.Text>
            ),
          },
          {
            title: "Value",
            dataIndex: "value",
            render: (v: string) => (
              <Typography.Text copyable style={{ whiteSpace: "pre-wrap" }}>
                {v || "-"}
              </Typography.Text>
            ),
          },
        ]}
      />
    </Modal>
    </>
  );
}

