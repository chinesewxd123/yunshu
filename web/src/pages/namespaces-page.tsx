import { EyeOutlined, TagsOutlined } from "@ant-design/icons";
import { Button, Descriptions, Divider, Modal, Table, Tag, Tooltip, Typography } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useState } from "react";
import { YamlCrudPage } from "../components/k8s/yaml-crud-page";
import { applyNamespace, deleteNamespace, getNamespaceDetail, listNamespaces } from "../services/namespaces";

type Item = {
  name: string;
  status: string;
  creation_time: string;
  labels?: Record<string, string>;
  annotations?: Record<string, string>;
  pod_count?: number;
  cpu_requests?: string;
  cpu_limits?: string;
  mem_requests?: string;
  mem_limits?: string;
};
type Detail = {
  yaml: string;
  item: Item;
  finalizers?: string[];
  resource_quotas?: Array<{ name: string; hard?: Record<string, string>; used?: Record<string, string>; scope?: string[] }>;
  limit_ranges?: Array<{ name: string; limits?: Array<{ type?: string; max?: Record<string, string>; min?: Record<string, string>; default?: Record<string, string>; defaultRequest?: Record<string, string>; maxLimitRequestRatio?: Record<string, string> }> }>;
  recent_events?: Array<{ type: string; reason: string; message: string; last_time?: string; count: number }>;
};

export function NamespacesPage() {
  const [metaOpen, setMetaOpen] = useState(false);
  const [metaTitle, setMetaTitle] = useState<"标签" | "注解">("标签");
  const [metaData, setMetaData] = useState<Record<string, string>>({});

  const openMetaModal = (title: "标签" | "注解", data?: Record<string, string>) => {
    setMetaTitle(title);
    setMetaData(data ?? {});
    setMetaOpen(true);
  };

  const renderMetaCell = (title: "标签" | "注解", data?: Record<string, string>) => {
    const icon = title === "标签" ? <TagsOutlined /> : <EyeOutlined />;
    return (
      <Tooltip title={`查看${title}`}>
        <Button
          type="link"
          size="small"
          icon={icon}
          onClick={() => openMetaModal(title, data)}
        />
      </Tooltip>
    );
  };

  const columns: ColumnsType<Item> = [
    { title: "命名空间", dataIndex: "name" },
    { title: "状态", dataIndex: "status", width: 120, render: (v: string) => <Tag color={v === "Active" ? "green" : "default"}>{v || "-"}</Tag> },
    { title: "Pod 数", dataIndex: "pod_count", width: 90, render: (v?: number) => (typeof v === "number" ? v : "-") },
    { title: "CPU(Req/Lim)", key: "cpu", width: 160, render: (_, r) => `${r.cpu_requests || "-"} / ${r.cpu_limits || "-"}` },
    { title: "内存(Req/Lim)", key: "mem", width: 170, render: (_, r) => `${r.mem_requests || "-"} / ${r.mem_limits || "-"}` },
    { title: "标签", key: "labels", width: 70, align: "center", render: (_, r) => renderMetaCell("标签", r.labels) },
    { title: "注解", key: "annotations", width: 70, align: "center", render: (_, r) => renderMetaCell("注解", r.annotations) },
    { title: "创建时间", dataIndex: "creation_time", width: 180, fixed: "right" },
  ];

  return (
    <>
    <YamlCrudPage<Item, Detail>
      title="命名空间管理"
      needNamespace={false}
      columns={columns}
      api={{
        list: async ({ clusterId, keyword }) => await listNamespaces(clusterId, keyword),
        detail: async ({ clusterId, name }) => await getNamespaceDetail(clusterId, name),
        apply: async ({ clusterId, manifest }) => await applyNamespace(clusterId, manifest),
        remove: async ({ clusterId, name }) => await deleteNamespace(clusterId, name),
      }}
      createTemplate={() => `apiVersion: v1
kind: Namespace
metadata:
  name: demo-namespace
`}
      detailExtra={(d) => (
        <div>
          <Descriptions size="small" bordered column={2} style={{ marginBottom: 10 }}>
            <Descriptions.Item label="命名空间">{d.item.name}</Descriptions.Item>
            <Descriptions.Item label="状态">
              <Tag color={d.item.status === "Active" ? "green" : "default"}>{d.item.status || "-"}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="标签">{renderMetaCell("标签", d.item.labels)}</Descriptions.Item>
            <Descriptions.Item label="注解">{renderMetaCell("注解", d.item.annotations)}</Descriptions.Item>
            <Descriptions.Item label="Finalizers" span={2}>
              {d.finalizers?.length ? d.finalizers.join(", ") : "-"}
            </Descriptions.Item>
          </Descriptions>
          <Divider style={{ margin: "10px 0" }} />
          <Typography.Text strong>ResourceQuota</Typography.Text>
          <Table
            size="small"
            pagination={false}
            style={{ marginTop: 8 }}
            rowKey={(r) => r.name}
            dataSource={d.resource_quotas || []}
            locale={{ emptyText: "无 ResourceQuota" }}
            columns={[
              { title: "名称", dataIndex: "name", width: 180 },
              {
                title: "Hard",
                render: (_, r) => Object.entries(r.hard || {}).map(([k, v]) => `${k}=${v}`).join(", ") || "-",
              },
              {
                title: "Used",
                render: (_, r) => Object.entries(r.used || {}).map(([k, v]) => `${k}=${v}`).join(", ") || "-",
              },
            ]}
          />
          <Divider style={{ margin: "10px 0" }} />
          <Typography.Text strong>LimitRange</Typography.Text>
          <Table
            size="small"
            pagination={false}
            style={{ marginTop: 8 }}
            rowKey={(r) => r.name}
            dataSource={d.limit_ranges || []}
            locale={{ emptyText: "无 LimitRange" }}
            columns={[
              { title: "名称", dataIndex: "name", width: 180 },
              {
                title: "限制项",
                render: (_, r) =>
                  (r.limits || [])
                    .map((x) => `${x.type || "Container"}: min(${Object.entries(x.min || {}).map(([k, v]) => `${k}=${v}`).join(" ") || "-"}) max(${Object.entries(x.max || {}).map(([k, v]) => `${k}=${v}`).join(" ") || "-"})`)
                    .join(" ; ") || "-",
              },
            ]}
          />
          <Divider style={{ margin: "10px 0" }} />
          <Typography.Text strong>最近事件</Typography.Text>
          <Table
            size="small"
            pagination={false}
            style={{ marginTop: 8 }}
            rowKey={(r) => `${r.last_time || ""}-${r.reason}-${r.message}`}
            dataSource={d.recent_events || []}
            locale={{ emptyText: "暂无事件" }}
            columns={[
              { title: "时间", dataIndex: "last_time", width: 170, render: (v?: string) => v || "-" },
              { title: "类型", dataIndex: "type", width: 90, render: (v: string) => <Tag color={v === "Warning" ? "red" : "green"}>{v || "-"}</Tag> },
              { title: "原因", dataIndex: "reason", width: 150 },
              { title: "次数", dataIndex: "count", width: 70 },
              { title: "消息", dataIndex: "message" },
            ]}
          />
        </div>
      )}
    />
    <Modal
      title={`${metaTitle}详情`}
      open={metaOpen}
      onCancel={() => setMetaOpen(false)}
      footer={null}
      width={760}
      destroyOnClose
    >
      <Table
        size="small"
        rowKey={(r) => r.key}
        pagination={{ pageSize: 10 }}
        dataSource={Object.entries(metaData).map(([key, value]) => ({ key, value }))}
        locale={{ emptyText: `该命名空间暂无${metaTitle}` }}
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

