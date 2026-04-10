import { FileTextOutlined, TagsOutlined } from "@ant-design/icons";
import { Button, Modal, Table, Tag, Tooltip, Typography } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useState } from "react";
import { YamlCrudPage } from "../components/k8s/yaml-crud-page";
import { applyIngressClass, deleteIngressClass, getIngressClassDetail, listIngressClasses, type IngressClassDetail, type IngressClassItem } from "../services/ingresses";

export function IngressClassesPage() {
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

  const columns: ColumnsType<IngressClassItem> = [
    { title: "名称", dataIndex: "name", width: 180 },
    { title: "Ingress数量", dataIndex: "ingress_count", width: 120 },
    { title: "控制器名称", dataIndex: "controller", width: 260, render: (v?: string) => v || "-" },
    {
      title: "是否默认",
      dataIndex: "is_default",
      width: 110,
      render: (v: boolean) => <Tag color={v ? "green" : "default"}>{v ? "是" : "否"}</Tag>,
    },
    { title: "标签", key: "labels", width: 70, align: "center", render: (_, r) => renderKVIcon("标签", <TagsOutlined />, r.labels) },
    { title: "注解", key: "annotations", width: 70, align: "center", render: (_, r) => renderKVIcon("注解", <FileTextOutlined />, r.annotations) },
    { title: "存在时长", dataIndex: "age", width: 100, fixed: "right", render: (v?: string) => v || "-" },
    { title: "创建时间", dataIndex: "creation_time", width: 180, fixed: "right" },
  ];

  return (
    <>
      <YamlCrudPage<IngressClassItem, IngressClassDetail>
        title="IngressClass 入口类管理"
        columns={columns}
        api={{
          list: async ({ clusterId, keyword }) => await listIngressClasses(clusterId, keyword),
          detail: async ({ clusterId, name }) => await getIngressClassDetail(clusterId, name),
          apply: async ({ clusterId, manifest }) => await applyIngressClass(clusterId, manifest),
          remove: async ({ clusterId, name }) => await deleteIngressClass(clusterId, name),
        }}
        createTemplate={() => `apiVersion: networking.k8s.io/v1
kind: IngressClass
metadata:
  name: nginx
  annotations:
    ingressclass.kubernetes.io/is-default-class: "false"
spec:
  controller: k8s.io/ingress-nginx
`}
      />

      <Modal title={kvTitle} open={kvOpen} onCancel={() => setKvOpen(false)} footer={null} width={720}>
        <Table
          rowKey={(r) => r.key}
          pagination={false}
          dataSource={Object.entries(kvData).map(([key, value]) => ({ key, value }))}
          locale={{ emptyText: "暂无数据" }}
          columns={[
            { title: "Key", dataIndex: "key", width: 260, render: (v: string) => <Typography.Text copyable>{v}</Typography.Text> },
            { title: "Value", dataIndex: "value", render: (v: string) => <Typography.Text copyable style={{ whiteSpace: "pre-wrap" }}>{v}</Typography.Text> },
          ]}
        />
      </Modal>
    </>
  );
}

