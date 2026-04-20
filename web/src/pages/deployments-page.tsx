import { DownOutlined, EyeOutlined, FileTextOutlined, ReloadOutlined, ScissorOutlined, TagsOutlined } from "@ant-design/icons";
import { Button, Card, Dropdown, Form, Input, InputNumber, Modal, Progress, Space, Tag, Typography, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useRef, useState } from "react";
import { useKeyValueViewer } from "../components/k8s/key-value-viewer";
import { useRelatedPodsDrawer } from "../components/k8s/related-pods-drawer";
import { useWorkloadFormActions } from "../components/k8s/workload-form-actions";
import { YamlCrudPage } from "../components/k8s/yaml-crud-page";
import { InputNumber as AntdInputNumber } from "antd";
import { listNamespaces as listClusterNamespaces } from "../services/clusters";
import {
  applyDeployment,
  deleteDeployment,
  getDeploymentDetail,
  listDeployments,
  listDeploymentPods,
  restartDeployment,
  scaleDeployment,
  type WorkloadDetail,
  type WorkloadItem,
} from "../services/workloads";
import {
  WorkloadFormModal,
  NameNamespaceItems,
  ContainerCommonItems,
  WorkloadAdvancedItems,
  WorkloadPolicyItems,
  DeploymentHealthAndImagePullSecretsItems,
  buildDeploymentYaml,
  deploymentObjToForm,
  deploymentYamlToForm,
  qosFromResources,
  type DeploymentFormValues,
} from "../components/k8s/workload-forms";

function DeploymentDetailQuickEdit({
  detail,
  detailYaml,
  setDetailYaml,
}: {
  detail: WorkloadDetail;
  detailYaml: string;
  setDetailYaml: (next: string) => void;
}) {
  const [detailForm] = Form.useForm<DeploymentFormValues>();
  const values = deploymentYamlToForm(detailYaml || "") ?? deploymentObjToForm(detail.object) ?? deploymentYamlToForm(detail.yaml ?? "");
  const liveness = detail.object?.spec?.template?.spec?.containers?.[0]?.livenessProbe;
  const readiness = detail.object?.spec?.template?.spec?.containers?.[0]?.readinessProbe;
  const formatProbe = (probe: any): string => {
    if (!probe) return "-";
    if (probe.httpGet) {
      const hp = probe.httpGet;
      const port = typeof hp.port === "number" ? hp.port : String(hp.port || "");
      return `${hp.path || "/"} : ${port} ${hp.scheme || ""}`;
    }
    if (probe.tcpSocket) {
      const tp = probe.tcpSocket;
      const port = typeof tp.port === "number" ? tp.port : String(tp.port || "");
      return `TCP : ${port}`;
    }
    return "自定义探针";
  };
  return (
    <Form
      form={detailForm}
      layout="vertical"
      initialValues={values ?? undefined}
      onValuesChange={(_, allValues) => {
        try {
          setDetailYaml(buildDeploymentYaml(allValues as DeploymentFormValues));
        } catch {
          // ignore partial invalid values during typing
        }
      }}
    >
      <Card size="small" title="快速编辑">
        <Space style={{ width: "100%" }} align="start">
          <Form.Item name="name" label="名称" rules={[{ required: true }]} style={{ flex: 1 }}>
            <Input />
          </Form.Item>
          <Form.Item name="namespace" label="命名空间" rules={[{ required: true }]} style={{ width: 220 }}>
            <Input />
          </Form.Item>
          <Form.Item name="replicas" label="副本数" style={{ width: 160 }}>
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
        </Space>
        <Space style={{ width: "100%" }} align="start">
          <Form.Item name="container_name" label="容器名" style={{ width: 220 }}>
            <Input />
          </Form.Item>
          <Form.Item name="image" label="容器镜像" style={{ flex: 1 }}>
            <Input />
          </Form.Item>
          <Form.Item name="port" label="容器端口" style={{ width: 160 }}>
            <InputNumber min={1} max={65535} style={{ width: "100%" }} />
          </Form.Item>
        </Space>
        <Space style={{ width: "100%" }} align="start">
          <Form.Item name="requests_cpu" label="CPU Request" style={{ width: 180 }}>
            <Input />
          </Form.Item>
          <Form.Item name="limits_cpu" label="CPU Limit" style={{ width: 180 }}>
            <Input />
          </Form.Item>
          <Form.Item name="requests_memory" label="MEM Request" style={{ width: 180 }}>
            <Input />
          </Form.Item>
          <Form.Item name="limits_memory" label="MEM Limit" style={{ width: 180 }}>
            <Input />
          </Form.Item>
        </Space>
        <Typography.Text type="secondary">探针摘要：Liveness {formatProbe(liveness)}；Readiness {formatProbe(readiness)}</Typography.Text>
      </Card>
    </Form>
  );
}

export function DeploymentsPage() {
  const listReloadRef = useRef<() => void>(() => {});
  const [form] = Form.useForm<DeploymentFormValues>();
  const formActions = useWorkloadFormActions<DeploymentFormValues>({
    form,
    mode: true,
    defaultMode: "create",
    getDetail: async (clusterId, namespace, name) => await getDeploymentDetail(clusterId, namespace, name),
    toFormValues: (d) => deploymentObjToForm(d.object) ?? deploymentYamlToForm(d.yaml ?? ""),
    buildFallbackValues: ({ recordName, namespace, record }) => ({
      name: recordName,
      namespace,
      replicas: Number(record?.replicas ?? 1) || 1,
      container_name: recordName,
      image: "",
      env_pairs: [{ key: "", value: "" }],
    }),
  });

  const [scaleOpen, setScaleOpen] = useState(false);
  const [scaleValue, setScaleValue] = useState<number>(1);
  const [scaleTarget, setScaleTarget] = useState<{ clusterId: number; namespace: string; name: string } | null>(null);
  const { openPods, viewer: podsViewer } = useRelatedPodsDrawer(async ({ clusterId, namespace, name }) => await listDeploymentPods(clusterId, namespace, name));
  const { renderKVIcon, viewer } = useKeyValueViewer({
    width: 760,
    compact: true,
    pageSize: 10,
    destroyOnClose: true,
    emptyText: (title) => `暂无${title}`,
  });

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
    { title: "运行时长", dataIndex: "age", width: 90 },
    { title: "创建时间", dataIndex: "creation_time", width: 180, fixed: "right" },
  ];

  return (
    <>
      <YamlCrudPage<WorkloadItem, WorkloadDetail>
        title="Deployment 控制器管理"
        needNamespace
        onLoadNamespaces={async (cid) => {
          const res = await listClusterNamespaces(cid);
          return (res.list ?? []).map((n) => ({ label: n.name, value: n.name }));
        }}
        columns={columns}
        api={{
          list: async ({ clusterId, namespace, keyword }) => await listDeployments(clusterId, namespace ?? "default", keyword),
          detail: async ({ clusterId, namespace, name }) => await getDeploymentDetail(clusterId, namespace ?? "default", name),
          apply: async ({ clusterId, manifest }) => await applyDeployment(clusterId, manifest),
          remove: async ({ clusterId, namespace, name }) => await deleteDeployment(clusterId, namespace ?? "default", name),
        }}
        onEdit={(record, ctx) => formActions.openEdit({ clusterId: ctx.clusterId, namespace: ctx.namespace ?? "default", name: record.name }, record)}
        onToolbarReady={(ctx) => {
          listReloadRef.current = ctx.reload;
        }}
        onCreateDrawerOpen={(ctx) => {
          if (!ctx.clusterId) return;
          formActions.prepareCreate(
            { clusterId: ctx.clusterId, namespace: ctx.namespace ?? "default" },
            {
              namespace: ctx.namespace ?? "default",
              replicas: 1,
              env_pairs: [{ key: "", value: "" }],
              name: "",
              container_name: "",
              image: "nginx:latest",
            } as Partial<DeploymentFormValues>,
          );
        }}
        renderCreateFormTab={(ctx) => (
          <WorkloadFormModal<DeploymentFormValues>
            embedded
            title="Deployment 表单创建"
            open={false}
            loading={formActions.loading}
            form={form}
            onCancel={ctx.closeCreateDrawer}
            onSubmit={(values) => {
              if (!formActions.ctx) return;
              const fctx = formActions.ctx;
              formActions.setLoading(true);
              void (async () => {
                try {
                  const manifest = buildDeploymentYaml(values);
                  await applyDeployment(fctx.clusterId, manifest);
                  message.success("已应用 Deployment");
                  ctx.closeCreateDrawer();
                  listReloadRef.current();
                } finally {
                  formActions.setLoading(false);
                }
              })();
            }}
          >
            <NameNamespaceItems />
            <Space style={{ width: "100%" }} align="start">
              <Form.Item name="replicas" label="副本数" rules={[{ required: true, message: "请输入副本数" }]} style={{ width: 240 }}>
                <AntdInputNumber min={0} />
              </Form.Item>
            </Space>
            <ContainerCommonItems showPort />
            <WorkloadAdvancedItems />
            <WorkloadPolicyItems showDeployStrategy />
            <DeploymentHealthAndImagePullSecretsItems />
            <Form.Item noStyle shouldUpdate>
              {() => {
                const v = form.getFieldsValue();
                const qos = qosFromResources(v);
                return (
                  <Typography.Text type="secondary">
                    QoS 说明：Deployment 不能直接设置 QoS，QoS 由 resources 推导，当前预估为：{qos}
                  </Typography.Text>
                );
              }}
            </Form.Item>
          </WorkloadFormModal>
        )}
        createTemplate={({ namespace }) => `apiVersion: apps/v1
kind: Deployment
metadata:
  name: demo-deployment
  namespace: ${namespace ?? "default"}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: demo
  template:
    metadata:
      labels:
        app: demo
    spec:
      containers:
        - name: demo
          image: nginx:latest
          ports:
            - containerPort: 80
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
                    label: "重启工作负载",
                    icon: <ReloadOutlined />,
                    onClick: () => {
                      void (async () => {
                        await restartDeployment(ctx.clusterId, ctx.namespace ?? "default", record.name);
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

      <Modal
        title={`Deployment 扩缩容${scaleTarget ? `：${scaleTarget.name}` : ""}`}
        open={scaleOpen}
        onCancel={() => setScaleOpen(false)}
        onOk={() => {
          if (!scaleTarget) return;
          void (async () => {
            await scaleDeployment(scaleTarget.clusterId, scaleTarget.namespace, scaleTarget.name, scaleValue);
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

      <WorkloadFormModal<DeploymentFormValues>
        title="Deployment 表单编辑"
        open={formActions.open && formActions.mode === "edit"}
        loading={formActions.loading}
        form={form}
        onCancel={formActions.close}
        onSubmit={(values) => {
          if (!formActions.ctx) return;
          const wctx = formActions.ctx;
          formActions.setLoading(true);
          void (async () => {
            try {
              const manifest = buildDeploymentYaml(values);
              await applyDeployment(wctx.clusterId, manifest);
              message.success("已应用 Deployment");
              formActions.close();
              listReloadRef.current();
            } finally {
              formActions.setLoading(false);
            }
          })();
        }}
      >
        <NameNamespaceItems />
        <Space style={{ width: "100%" }} align="start">
          <Form.Item name="replicas" label="副本数" rules={[{ required: true, message: "请输入副本数" }]} style={{ width: 240 }}>
            <AntdInputNumber min={0} />
          </Form.Item>
        </Space>
        <ContainerCommonItems showPort />
        <WorkloadAdvancedItems />
        <WorkloadPolicyItems showDeployStrategy />
        <DeploymentHealthAndImagePullSecretsItems />
        <Form.Item noStyle shouldUpdate>
          {() => {
            const v = form.getFieldsValue();
            const qos = qosFromResources(v);
            return (
              <Typography.Text type="secondary">
                QoS 说明：Deployment 不能直接设置 QoS，QoS 由 resources 推导，当前预估为：{qos}
              </Typography.Text>
            );
          }}
        </Form.Item>
      </WorkloadFormModal>
      {viewer}
    </>
  );
}

