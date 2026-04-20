import { DownOutlined, EditOutlined, EyeOutlined, FileTextOutlined, ReloadOutlined, TagsOutlined } from "@ant-design/icons";
import { Button, Dropdown, Form, Input, InputNumber, Progress, Space, Tag, Typography, message } from "antd";
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
        showEditButton={false}
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
        renderDetail={(d, { detailYaml, setDetailYaml }) => <DaemonSetDetailQuickEdit detail={d} detailYaml={detailYaml} setDetailYaml={setDetailYaml} />}
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
                    key: "edit",
                    label: "编辑",
                    icon: <EditOutlined />,
                    onClick: () => formActions.openEdit({ clusterId: ctx.clusterId, namespace: ctx.namespace ?? "default", name: record.name }, record),
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

