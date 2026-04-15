import { DownOutlined, EyeOutlined, FileTextOutlined, ReloadOutlined, ScissorOutlined, TagsOutlined } from "@ant-design/icons";
import { Button, Form, Input, InputNumber, Modal, Progress, Space, Tag, Typography, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useState } from "react";
import { useKeyValueViewer } from "../components/k8s/key-value-viewer";
import { useRelatedPodsDrawer } from "../components/k8s/related-pods-drawer";
import { useWorkloadFormActions } from "../components/k8s/workload-form-actions";
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
  const [form] = Form.useForm<StatefulSetFormValues>();
  const formActions = useWorkloadFormActions<StatefulSetFormValues>({
    form,
    mode: true,
    defaultMode: "create",
    getDetail: async (clusterId, namespace, name) => await getStatefulSetDetail(clusterId, namespace, name),
    toFormValues: (d) => statefulSetObjToForm(d.object) ?? statefulSetYamlToForm(d.yaml ?? ""),
    buildFallbackValues: ({ recordName, namespace, record }) => ({
      name: recordName,
      namespace,
      service_name: `${recordName}-headless`,
      replicas: Number(record?.replicas ?? 1) || 1,
      container_name: recordName,
      image: "",
      env_pairs: [{ key: "", value: "" }],
    }),
  });

  const [scaleOpen, setScaleOpen] = useState(false);
  const [scaleValue, setScaleValue] = useState<number>(1);
  const [scaleTarget, setScaleTarget] = useState<{ clusterId: number; namespace: string; name: string } | null>(null);
  const { openPods, viewer: podsViewer } = useRelatedPodsDrawer(async ({ clusterId, namespace, name }) => await listStatefulSetPods(clusterId, namespace, name));
  const { renderKVIcon, viewer } = useKeyValueViewer();

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
                    onClick: () => openPods({ clusterId: ctx.clusterId, namespace: ctx.namespace ?? "default", name: record.name }),
                  },
                  {
                    key: "edit",
                    label: "编辑",
                    onClick: () => formActions.openEdit({ clusterId: ctx.clusterId, namespace: ctx.namespace ?? "default", name: record.name }, record),
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

      {viewer}

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

      {podsViewer}

      <WorkloadFormModal<StatefulSetFormValues>
        title={formActions.mode === "create" ? "StatefulSet 表单创建" : "StatefulSet 表单编辑"}
        open={formActions.open}
        loading={formActions.loading}
        form={form}
        onCancel={formActions.close}
        onSubmit={(values) => {
          if (!formActions.ctx) return;
          const ctx = formActions.ctx;
          formActions.setLoading(true);
          void (async () => {
            try {
              const manifest = buildStatefulSetYaml(values);
              await applyStatefulSet(ctx.clusterId, manifest);
              message.success("已应用 StatefulSet");
              formActions.close();
            } finally {
              formActions.setLoading(false);
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

