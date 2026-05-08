import { ColumnHeightOutlined, DownOutlined, EyeOutlined, FileTextOutlined, ReloadOutlined, TagsOutlined } from "@ant-design/icons";
import { Alert, Button, Dropdown, Form, Input, InputNumber, Modal, Progress, Space, Tag, Typography, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { YamlCrudPage } from "../components/k8s/yaml-crud-page";
import { listNamespaces as listClusterNamespaces } from "../services/clusters";
import {
  applyDaemonSet,
  buildCpuMemoryResourceMaps,
  deleteDaemonSet,
  getDaemonSetDetail,
  listDaemonSets,
  listDaemonSetPods,
  patchDaemonSetContainerResources,
  restartDaemonSet,
  type WorkloadDetail,
  type WorkloadItem,
} from "../services/workloads";
import { useRef, useState } from "react";
import { useKeyValueViewer } from "../components/k8s/key-value-viewer";
import { useRelatedPodsDrawer } from "../components/k8s/related-pods-drawer";
import { useWorkloadFormActions } from "../components/k8s/workload-form-actions";
import {
  ContainerCommonItems,
  DaemonSetFormValues,
  DeploymentHealthAndImagePullSecretsItems,
  NameNamespaceItems,
  WorkloadAdvancedItems,
  WorkloadPolicyItems,
  WorkloadFormModal,
  buildDaemonSetYaml,
  daemonSetObjToForm,
  daemonSetYamlToForm,
  qosFromResources,
} from "../components/k8s/workload-forms";

function DaemonSetDetailQuickEdit({
  detail,
  detailYaml,
  setDetailYaml,
}: {
  detail: WorkloadDetail;
  detailYaml: string;
  setDetailYaml: (next: string) => void;
}) {
  const [detailForm] = Form.useForm<DaemonSetFormValues>();
  const values = daemonSetYamlToForm(detailYaml || "") ?? daemonSetObjToForm(detail.object) ?? daemonSetYamlToForm(detail.yaml ?? "");
  return (
    <Form
      form={detailForm}
      layout="vertical"
      initialValues={values ?? undefined}
      onValuesChange={(_, allValues) => {
        try {
          setDetailYaml(buildDaemonSetYaml(allValues as DaemonSetFormValues));
        } catch {
          // ignore partial invalid values during typing
        }
      }}
    >
      <Space style={{ width: "100%" }} align="start">
        <Form.Item name="name" label="名称" rules={[{ required: true }]} style={{ flex: 1 }}>
          <Input />
        </Form.Item>
        <Form.Item name="namespace" label="命名空间" rules={[{ required: true }]} style={{ width: 220 }}>
          <Input />
        </Form.Item>
      </Space>
      <Space style={{ width: "100%" }} align="start">
        <Form.Item name="container_name" label="容器名" style={{ width: 220 }}>
          <Input />
        </Form.Item>
        <Form.Item name="image" label="镜像" style={{ flex: 1 }}>
          <Input />
        </Form.Item>
        <Form.Item name="port" label="端口" style={{ width: 160 }}>
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
    </Form>
  );
}

export function DaemonsetsPage() {
  const listReloadRef = useRef<() => void>(() => {});
  const [verticalOpen, setVerticalOpen] = useState(false);
  const [verticalTarget, setVerticalTarget] = useState<{ clusterId: number; namespace: string; name: string } | null>(null);
  const [verticalForm] = Form.useForm<{
    container_name?: string;
    requests_cpu?: string;
    requests_memory?: string;
    limits_cpu?: string;
    limits_memory?: string;
  }>();
  const [form] = Form.useForm<DaemonSetFormValues>();
  const formActions = useWorkloadFormActions<DaemonSetFormValues>({
    form,
    mode: true,
    defaultMode: "create",
    getDetail: async (clusterId, namespace, name) => await getDaemonSetDetail(clusterId, namespace, name),
    toFormValues: (d) => daemonSetObjToForm(d.object) ?? daemonSetYamlToForm(d.yaml ?? ""),
    buildFallbackValues: ({ recordName, namespace }) => ({
      name: recordName,
      namespace,
      container_name: recordName,
      image: "",
      env_pairs: [{ key: "", value: "" }],
    }),
  });

  const { openPods, viewer: podsViewer } = useRelatedPodsDrawer(async ({ clusterId, namespace, name }) => await listDaemonSetPods(clusterId, namespace, name));
  const { renderKVIcon, viewer } = useKeyValueViewer({ mode: "drawer" });

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
        onEdit={(record, ctx) => formActions.openEdit({ clusterId: ctx.clusterId, namespace: ctx.namespace ?? "default", name: record.name }, record)}
        onToolbarReady={(ctx) => {
          listReloadRef.current = ctx.reload;
        }}
        onCreateDrawerOpen={(ctx) => {
          if (!ctx.clusterId) return;
          const ns = ctx.namespace ?? "default";
          formActions.prepareCreate(
            { clusterId: ctx.clusterId, namespace: ns },
            {
              namespace: ns,
              name: "",
              container_name: "",
              image: "nginx:latest",
              env_pairs: [{ key: "", value: "" }],
            } as Partial<DaemonSetFormValues>,
          );
        }}
        renderCreateFormTab={(ctx) => (
          <WorkloadFormModal<DaemonSetFormValues>
            embedded
            title="DaemonSet 表单创建"
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
                  const manifest = buildDaemonSetYaml(values);
                  await applyDaemonSet(fctx.clusterId, manifest);
                  message.success("已应用 DaemonSet");
                  ctx.closeCreateDrawer();
                  listReloadRef.current();
                } finally {
                  formActions.setLoading(false);
                }
              })();
            }}
          >
            <NameNamespaceItems />
            <ContainerCommonItems showPort />
            <WorkloadAdvancedItems />
            <WorkloadPolicyItems showDaemonSetStrategy />
            <DeploymentHealthAndImagePullSecretsItems />
          </WorkloadFormModal>
        )}
        api={{
          list: async ({ clusterId, namespace, keyword }) => await listDaemonSets(clusterId, namespace ?? "default", keyword),
          detail: async ({ clusterId, namespace, name }) => await getDaemonSetDetail(clusterId, namespace ?? "default", name),
          apply: async ({ clusterId, manifest }) => await applyDaemonSet(clusterId, manifest),
          remove: async ({ clusterId, namespace, name }) => await deleteDaemonSet(clusterId, namespace ?? "default", name),
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
                    onClick: () => openPods({ clusterId: ctx.clusterId, namespace: ctx.namespace ?? "default", name: record.name }),
                  },
                  {
                    key: "vertical",
                    label: "垂直扩缩（resources · VPA 类）",
                    icon: <ColumnHeightOutlined />,
                    onClick: () => {
                      setVerticalTarget({ clusterId: ctx.clusterId, namespace: ctx.namespace ?? "default", name: record.name });
                      verticalForm.resetFields();
                      setVerticalOpen(true);
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

      <Modal
        title={`DaemonSet 垂直扩缩（Pod 模板 resources · VPA 类）${verticalTarget ? `：${verticalTarget.name}` : ""}`}
        open={verticalOpen}
        onCancel={() => setVerticalOpen(false)}
        destroyOnClose
        width={560}
        onOk={() => {
          if (!verticalTarget) return;
          void verticalForm.validateFields().then(async (values) => {
            const { requests, limits } = buildCpuMemoryResourceMaps(values);
            if (Object.keys(requests).length === 0 && Object.keys(limits).length === 0) {
              message.warning("请至少填写一项 requests 或 limits（如 cpu/memory）");
              return;
            }
            await patchDaemonSetContainerResources(verticalTarget.clusterId, verticalTarget.namespace, verticalTarget.name, {
              container_name: values.container_name,
              requests,
              limits,
            });
            message.success("已更新容器资源");
            setVerticalOpen(false);
            listReloadRef.current();
          });
        }}
      >
        <Alert
          type="warning"
          showIcon
          style={{ marginBottom: 12 }}
          message="DaemonSet 在每节点运行副本，不属于 HPA「scale 副本」类控制器；仅支持修改 Pod 模板 resources（与 VPA 纳管范围一致）。上调 requests 可能导致节点容量紧张，请谨慎评估。"
        />
        <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
          留空容器名则修改第一个容器。示例：CPU <Typography.Text code>100m</Typography.Text>，内存{" "}
          <Typography.Text code>256Mi</Typography.Text>。
        </Typography.Paragraph>
        <Form form={verticalForm} layout="vertical">
          <Form.Item label="容器名（可选）" name="container_name">
            <Input placeholder="默认第一个容器" allowClear />
          </Form.Item>
          <Form.Item label="requests.cpu" name="requests_cpu">
            <Input placeholder="如 100m" allowClear />
          </Form.Item>
          <Form.Item label="requests.memory" name="requests_memory">
            <Input placeholder="如 256Mi" allowClear />
          </Form.Item>
          <Form.Item label="limits.cpu" name="limits_cpu">
            <Input placeholder="如 500m" allowClear />
          </Form.Item>
          <Form.Item label="limits.memory" name="limits_memory">
            <Input placeholder="如 512Mi" allowClear />
          </Form.Item>
        </Form>
      </Modal>

      {viewer}

      {podsViewer}

      <WorkloadFormModal<DaemonSetFormValues>
        title={`DaemonSet 表单编辑${formActions.ctx?.name ? `：${formActions.ctx.name}` : ""}`}
        open={formActions.open && formActions.mode === "edit"}
        loading={formActions.loading}
        form={form}
        onCancel={formActions.close}
        onSubmit={(values) => {
          if (!formActions.ctx) return;
          const ctx = formActions.ctx;
          formActions.setLoading(true);
          void (async () => {
            try {
              const manifest = buildDaemonSetYaml(values);
              await applyDaemonSet(ctx.clusterId, manifest);
              message.success("已应用 DaemonSet");
              formActions.close();
              listReloadRef.current();
            } finally {
              formActions.setLoading(false);
            }
          })();
        }}
      >
        <NameNamespaceItems />
        <ContainerCommonItems showPort />
        <WorkloadAdvancedItems />
        <WorkloadPolicyItems showDaemonSetStrategy />
        <DeploymentHealthAndImagePullSecretsItems />
      </WorkloadFormModal>
    </>
  );
}

