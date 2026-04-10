import { DownOutlined, EyeOutlined, FileTextOutlined, ReloadOutlined, ScissorOutlined, TagsOutlined } from "@ant-design/icons";
import { Button, Drawer, Form, Input, InputNumber, Modal, Progress, Space, Table, Tag, Tooltip, Typography, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useState } from "react";
import { YamlCrudPage } from "../components/k8s/yaml-crud-page";
import { listNamespaces as listClusterNamespaces } from "../services/clusters";
import {
  applyStatefulSet,
  deleteStatefulSet,
  getStatefulSetDetail,
  listStatefulSets,
  listStatefulSetPods,
  restartStatefulSet,
  scaleStatefulSet,
  type RelatedPodItem,
  type WorkloadDetail,
  type WorkloadItem,
} from "../services/workloads";
import { Dropdown } from "antd";
import {
  WorkloadFormModal,
  NameNamespaceItems,
  ContainerCommonItems,
  WorkloadAdvancedItems,
  buildStatefulSetYaml,
  statefulSetObjToForm,
  statefulSetYamlToForm,
  qosFromResources,
  type StatefulSetFormValues,
} from "../components/k8s/workload-forms";

export function StatefulsetsPage() {
  const [formOpen, setFormOpen] = useState(false);
  const [formLoading, setFormLoading] = useState(false);
  const [formMode, setFormMode] = useState<"create" | "edit">("create");
  const [formCtx, setFormCtx] = useState<{ clusterId: number; namespace: string; name?: string } | null>(null);
  const [form] = Form.useForm<StatefulSetFormValues>();

  const [scaleOpen, setScaleOpen] = useState(false);
  const [scaleValue, setScaleValue] = useState<number>(1);
  const [scaleTarget, setScaleTarget] = useState<{ clusterId: number; namespace: string; name: string } | null>(null);
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
        title="StatefulSet 控制器管理"
        needNamespace
        onLoadNamespaces={async (cid) => {
          const res = await listClusterNamespaces(cid);
          return (res.list ?? []).map((n) => ({ label: n.name, value: n.name }));
        }}
        columns={columns}
        showEditButton={false}
        api={{
          list: async ({ clusterId, namespace, keyword }) => await listStatefulSets(clusterId, namespace ?? "default", keyword),
          detail: async ({ clusterId, namespace, name }) => await getStatefulSetDetail(clusterId, namespace ?? "default", name),
          apply: async ({ clusterId, manifest }) => await applyStatefulSet(clusterId, manifest),
          remove: async ({ clusterId, namespace, name }) => await deleteStatefulSet(clusterId, namespace ?? "default", name),
        }}
        renderToolbarExtraRight={undefined}
        createTemplate={({ namespace }) => `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: demo-statefulset
  namespace: ${namespace ?? "default"}
spec:
  serviceName: demo-headless
  replicas: 1
  selector:
    matchLabels:
      app: demo-sts
  template:
    metadata:
      labels:
        app: demo-sts
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
                          const items = await listStatefulSetPods(ctx.clusterId, ctx.namespace ?? "default", record.name);
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
                    onClick: () => {
                      setFormMode("edit");
                      setFormCtx({ clusterId: ctx.clusterId, namespace: ctx.namespace ?? "default", name: record.name });
                      setFormOpen(true);
                      setFormLoading(true);
                      void (async () => {
                        try {
                          const d = await getStatefulSetDetail(ctx.clusterId, ctx.namespace ?? "default", record.name);
                          const fv = statefulSetObjToForm(d.object) ?? statefulSetYamlToForm(d.yaml ?? "");
                          if (fv) {
                            form.setFieldsValue({ ...fv, namespace: ctx.namespace ?? fv.namespace } as any);
                          } else {
                            form.setFieldsValue({
                              name: record.name,
                              namespace: ctx.namespace ?? "default",
                              service_name: `${record.name}-headless`,
                              replicas: Number(record.replicas ?? 1) || 1,
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
                    key: "scale",
                    label: "扩缩容",
                    icon: <ScissorOutlined />,
                    onClick: () => {
                      setScaleTarget({ clusterId: ctx.clusterId, namespace: ctx.namespace ?? "default", name: record.name });
                      setScaleValue(Number(record.replicas ?? 1) || 1);
                      setScaleOpen(true);
                    },
                  },
                  {
                    key: "restart",
                    label: "重启",
                    icon: <ReloadOutlined />,
                    onClick: () => {
                      void (async () => {
                        await restartStatefulSet(ctx.clusterId, ctx.namespace ?? "default", record.name);
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

      <Modal
        title={`StatefulSet 扩缩容${scaleTarget ? `：${scaleTarget.name}` : ""}`}
        open={scaleOpen}
        onCancel={() => setScaleOpen(false)}
        onOk={() => {
          if (!scaleTarget) return;
          void (async () => {
            await scaleStatefulSet(scaleTarget.clusterId, scaleTarget.namespace, scaleTarget.name, scaleValue);
            message.success("扩缩容成功");
            setScaleOpen(false);
          })();
        }}
      >
        <Space>
          <span>副本数</span>
          <InputNumber min={0} value={scaleValue} onChange={(v) => setScaleValue(Number(v ?? 0))} />
        </Space>
      </Modal>

      <Drawer title={`关联 Pods${podsTarget ? `：${podsTarget.name}` : ""}`} open={podsOpen} onClose={() => setPodsOpen(false)} width={900}>
        <Table
          rowKey={(r) => `${r.namespace}/${r.name}`}
          loading={podsLoading}
          dataSource={pods}
          pagination={{ pageSize: 10 }}
          columns={[
            { title: "Pod 名称", dataIndex: "name" },
            { title: "状态", dataIndex: "phase", width: 120, render: (v: string) => <Tag color={v === "Running" ? "green" : "default"}>{v || "-"}</Tag> },
            { title: "节点", dataIndex: "node_name", width: 160 },
            { title: "PodIP", dataIndex: "pod_ip", width: 140 },
            { title: "重启", dataIndex: "restart_count", width: 90 },
            { title: "启动时间", dataIndex: "start_time", width: 180 },
          ]}
        />
      </Drawer>

      <WorkloadFormModal<StatefulSetFormValues>
        title={formMode === "create" ? "StatefulSet 表单创建" : "StatefulSet 表单编辑"}
        open={formOpen}
        loading={formLoading}
        form={form}
        onCancel={() => setFormOpen(false)}
        onSubmit={(values) => {
          if (!formCtx) return;
          setFormLoading(true);
          void (async () => {
            try {
              const manifest = buildStatefulSetYaml(values);
              await applyStatefulSet(formCtx.clusterId, manifest);
              message.success("已应用 StatefulSet");
              setFormOpen(false);
            } finally {
              setFormLoading(false);
            }
          })();
        }}
      >
        <NameNamespaceItems />
        <Form.Item name="service_name" label="ServiceName（Headless）" rules={[{ required: true, message: "请输入 serviceName" }]}>
          <Input placeholder="demo-headless" />
        </Form.Item>
        <Form.Item name="replicas" label="副本数" rules={[{ required: true, message: "请输入副本数" }]} style={{ width: 240 }}>
          <InputNumber min={0} />
        </Form.Item>
        <ContainerCommonItems showPort />
        <WorkloadAdvancedItems />
        <Form.Item noStyle shouldUpdate>
          {() => {
            const v = form.getFieldsValue();
            const qos = qosFromResources(v);
            return (
              <Typography.Text type="secondary">
                QoS 说明：StatefulSet 不能直接设置 QoS，QoS 由 resources 推导，当前预估为：{qos}
              </Typography.Text>
            );
          }}
        </Form.Item>
      </WorkloadFormModal>
    </>
  );
}

