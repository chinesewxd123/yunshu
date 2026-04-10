import { DownOutlined, EditOutlined, EyeOutlined, FileTextOutlined, ReloadOutlined, TagsOutlined } from "@ant-design/icons";
import { Button, Descriptions, Drawer, Dropdown, Form, Progress, Space, Table, Tag, Tooltip, Typography, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { YamlCrudPage } from "../components/k8s/yaml-crud-page";
import { listNamespaces as listClusterNamespaces } from "../services/clusters";
import {
  applyDaemonSet,
  deleteDaemonSet,
  getDaemonSetDetail,
  listDaemonSets,
  listDaemonSetPods,
  restartDaemonSet,
  type RelatedPodItem,
  type WorkloadDetail,
  type WorkloadItem,
} from "../services/workloads";
import { useState } from "react";
import {
  ContainerCommonItems,
  DaemonSetFormValues,
  DeploymentHealthAndImagePullSecretsItems,
  NameNamespaceItems,
  WorkloadAdvancedItems,
  WorkloadFormModal,
  buildDaemonSetYaml,
  daemonSetObjToForm,
  daemonSetYamlToForm,
  qosFromResources,
} from "../components/k8s/workload-forms";

export function DaemonsetsPage() {
  const [formOpen, setFormOpen] = useState(false);
  const [formLoading, setFormLoading] = useState(false);
  const [formCtx, setFormCtx] = useState<{ clusterId: number; namespace: string; name?: string } | null>(null);
  const [form] = Form.useForm<DaemonSetFormValues>();

  const [podsOpen, setPodsOpen] = useState(false);
  const [podsLoading, setPodsLoading] = useState(false);
  const [podsTarget, setPodsTarget] = useState<{ clusterId: number; namespace: string; name: string } | null>(null);
  const [pods, setPods] = useState<RelatedPodItem[]>([]);
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

  const columns: ColumnsType<WorkloadItem> = [
    { title: "命名空间", dataIndex: "namespace", width: 110 },
    { title: "名称", dataIndex: "name", width: 200 },
    { title: "副本", dataIndex: "replicas", width: 80 },
    {
      title: "资源汇总",
      dataIndex: "resource_text",
      width: 180,
      render: (v?: string) => <Typography.Text style={{ whiteSpace: "pre-wrap", fontSize: 12 }}>{v || "-"}</Typography.Text>,
    },
    { title: "标签", key: "labels", width: 70, align: "center", render: (_, r) => renderKVIcon("标签", <TagsOutlined />, r.labels) },
    { title: "注解", key: "annotations", width: 70, align: "center", render: (_, r) => renderKVIcon("注解", <FileTextOutlined />, r.annotations) },
    {
      title: "容器",
      dataIndex: "containers_text",
      width: 220,
      render: (v?: string) => <Typography.Text style={{ whiteSpace: "pre-wrap", fontSize: 12 }}>{v || "-"}</Typography.Text>,
    },
    {
      title: "当前状态",
      key: "status",
      width: 180,
      render: (_, r) => {
        const p = typeof r.ready_percent === "number" ? r.ready_percent : 0;
        return <Progress percent={Math.max(0, Math.min(100, p))} size="small" format={() => r.ready || `${p}%`} />;
      },
    },
    { title: "条件", dataIndex: "conditions_text", width: 160, fixed: "right" },
    { title: "存在时长", dataIndex: "age", width: 90 },
    { title: "创建时间", dataIndex: "creation_time", width: 180, fixed: "right" },
  ];

  return (
    <>
      <YamlCrudPage<WorkloadItem, WorkloadDetail>
        title="DaemonSet 控制器管理"
        needNamespace
        onLoadNamespaces={async (cid) => {
          const res = await listClusterNamespaces(cid);
          return (res.list ?? []).map((n) => ({ label: n.name, value: n.name }));
        }}
        columns={columns}
        showEditButton={false}
        api={{
          list: async ({ clusterId, namespace, keyword }) => await listDaemonSets(clusterId, namespace ?? "default", keyword),
          detail: async ({ clusterId, namespace, name }) => await getDaemonSetDetail(clusterId, namespace ?? "default", name),
          apply: async ({ clusterId, manifest }) => await applyDaemonSet(clusterId, manifest),
          remove: async ({ clusterId, namespace, name }) => await deleteDaemonSet(clusterId, namespace ?? "default", name),
        }}
        renderDetail={(d) => {
          const fv = daemonSetObjToForm(d.object) ?? daemonSetYamlToForm(d.yaml ?? "");
          const qos = fv ? qosFromResources(fv) : "-";
          return (
            <Descriptions size="small" column={2} bordered>
              <Descriptions.Item label="名称">{fv?.name || "-"}</Descriptions.Item>
              <Descriptions.Item label="命名空间">{fv?.namespace || "-"}</Descriptions.Item>
              <Descriptions.Item label="QoS（推导）">{String(qos)}</Descriptions.Item>
              <Descriptions.Item label="容器名">{fv?.container_name || "-"}</Descriptions.Item>
              <Descriptions.Item label="镜像">{fv?.image || "-"}</Descriptions.Item>
              <Descriptions.Item label="拉取策略">{fv?.image_pull_policy || "默认"}</Descriptions.Item>
              <Descriptions.Item label="镜像拉取 Secret" span={2}>
                {(fv?.image_pull_secrets ?? []).filter(Boolean).join("\n") || "-"}
              </Descriptions.Item>
              <Descriptions.Item label="端口">{fv?.port ? String(fv.port) : "-"}</Descriptions.Item>
              <Descriptions.Item label="CPU Request">{fv?.requests_cpu || "-"}</Descriptions.Item>
              <Descriptions.Item label="CPU Limit">{fv?.limits_cpu || "-"}</Descriptions.Item>
              <Descriptions.Item label="Mem Request">{fv?.requests_memory || "-"}</Descriptions.Item>
              <Descriptions.Item label="Mem Limit">{fv?.limits_memory || "-"}</Descriptions.Item>
              <Descriptions.Item label="容忍" span={2}>
                {(fv?.tolerations ?? [])
                  .filter((t) => t.key || t.operator === "Exists")
                  .map((t) => `${t.key || "*"} ${t.operator || "Equal"} ${t.value || ""} ${t.effect || ""}`)
                  .join("\n") || "-"}
              </Descriptions.Item>
              <Descriptions.Item label="卷" span={2}>
                {(fv?.volumes ?? [])
                  .filter((v) => v.name)
                  .map((v) => `${v.name} (${v.type}) ${v.source_name || ""}`)
                  .join("\n") || "-"}
              </Descriptions.Item>
              <Descriptions.Item label="挂载" span={2}>
                {(fv?.volume_mounts ?? [])
                  .filter((m) => m.name && m.mount_path)
                  .map((m) => `${m.name} -> ${m.mount_path}${m.read_only ? " (ro)" : ""}${m.sub_path ? ` subPath=${m.sub_path}` : ""}`)
                  .join("\n") || "-"}
              </Descriptions.Item>
              <Descriptions.Item label="环境变量" span={2}>
                {(fv?.env_pairs ?? [])
                  .filter((p) => p.key)
                  .map((p) => `${p.key}=${p.value ?? ""}`)
                  .join("\n") || "-"}
              </Descriptions.Item>
              <Descriptions.Item label="命令" span={2}>
                {fv?.command || "-"}
              </Descriptions.Item>
            </Descriptions>
          );
        }}
        createTemplate={({ namespace }) => `apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: demo-daemonset
  namespace: ${namespace ?? "default"}
spec:
  selector:
    matchLabels:
      app: demo-ds
  template:
    metadata:
      labels:
        app: demo-ds
    spec:
      containers:
        - name: demo
          image: nginx:latest
`}
        extraRowActions={(record, ctx) => (
          <Space>
            <Dropdown
              menu={{
                items: [
                  {
                    key: "pods",
                    label: "关联 Pods",
                    icon: <EyeOutlined />,
                    onClick: () => {
                      setPodsTarget({ clusterId: ctx.clusterId, namespace: ctx.namespace ?? "default", name: record.name });
                      setPodsOpen(true);
                      setPodsLoading(true);
                      void (async () => {
                        try {
                          const items = await listDaemonSetPods(ctx.clusterId, ctx.namespace ?? "default", record.name);
                          setPods(items ?? []);
                        } finally {
                          setPodsLoading(false);
                        }
                      })();
                    },
                  },
                  {
                    key: "edit",
                    label: "编辑",
                    icon: <EditOutlined />,
                    onClick: () => {
                      setFormCtx({ clusterId: ctx.clusterId, namespace: ctx.namespace ?? "default", name: record.name });
                      setFormOpen(true);
                      setFormLoading(true);
                      void (async () => {
                        try {
                          const d = await getDaemonSetDetail(ctx.clusterId, ctx.namespace ?? "default", record.name);
                          const fv = daemonSetObjToForm(d.object) ?? daemonSetYamlToForm(d.yaml ?? "");
                          if (fv) {
                            form.setFieldsValue({ ...fv, namespace: ctx.namespace ?? fv.namespace } as any);
                          } else {
                            form.setFieldsValue({
                              name: record.name,
                              namespace: ctx.namespace ?? "default",
                              container_name: record.name,
                              image: "",
                              env_pairs: [{ key: "", value: "" }],
                            } as any);
                          }
                        } finally {
                          setFormLoading(false);
                        }
                      })();
                    },
                  },
                  {
                    key: "restart",
                    label: "重启",
                    icon: <ReloadOutlined />,
                    onClick: () => {
                      void (async () => {
                        await restartDaemonSet(ctx.clusterId, ctx.namespace ?? "default", record.name);
                        message.success("已触发滚动重启");
                        ctx.reload();
                      })();
                    },
                  },
                ],
              }}
            >
              <Button type="link">
                更多 <DownOutlined />
              </Button>
            </Dropdown>
          </Space>
        )}
      />

      <Drawer title={kvTitle} open={kvOpen} onClose={() => setKvOpen(false)} width={720}>
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
      </Drawer>

      <Drawer title={`关联 Pods${podsTarget ? `：${podsTarget.name}` : ""}`} open={podsOpen} onClose={() => setPodsOpen(false)} width={900}>
        <Table
          rowKey={(r) => `${r.namespace}/${r.name}`}
          loading={podsLoading}
          dataSource={pods}
          pagination={{ pageSize: 10 }}
          columns={[
            { title: "Pod 名称", dataIndex: "name" },
            {
              title: "状态",
              dataIndex: "phase",
              width: 120,
              render: (v: string) => <Tag color={v === "Running" ? "green" : "default"}>{v || "-"}</Tag>,
            },
            { title: "节点", dataIndex: "node_name", width: 160 },
            { title: "PodIP", dataIndex: "pod_ip", width: 140 },
            { title: "重启", dataIndex: "restart_count", width: 90 },
            { title: "启动时间", dataIndex: "start_time", width: 180 },
          ]}
        />
      </Drawer>

      <WorkloadFormModal<DaemonSetFormValues>
        title={`DaemonSet 编辑${formCtx?.name ? `：${formCtx.name}` : ""}`}
        open={formOpen}
        loading={formLoading}
        form={form}
        onCancel={() => setFormOpen(false)}
        onSubmit={(values) => {
          if (!formCtx) return;
          setFormLoading(true);
          void (async () => {
            try {
              const manifest = buildDaemonSetYaml(values);
              await applyDaemonSet(formCtx.clusterId, manifest);
              message.success("已应用 DaemonSet");
              setFormOpen(false);
            } finally {
              setFormLoading(false);
            }
          })();
        }}
      >
        <NameNamespaceItems />
        <ContainerCommonItems showPort />
        <WorkloadAdvancedItems />
        <DeploymentHealthAndImagePullSecretsItems />
      </WorkloadFormModal>
    </>
  );
}

