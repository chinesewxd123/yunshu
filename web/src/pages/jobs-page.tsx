import { ColumnHeightOutlined, DownOutlined, EyeOutlined, FileTextOutlined, PlayCircleOutlined, TagsOutlined } from "@ant-design/icons";
import { Alert, Button, Dropdown, Form, Input, InputNumber, Modal, Progress, Select, Space, Tag, Typography, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useRef, useState } from "react";
import { useKeyValueViewer } from "../components/k8s/key-value-viewer";
import { useRelatedPodsDrawer } from "../components/k8s/related-pods-drawer";
import { RealtimeUsageText, WorkloadCpuUsageBars, WorkloadMemUsageBars } from "../components/k8s/k8s-resource-usage-cells";
import { useWorkloadFormActions } from "../components/k8s/workload-form-actions";
import { YamlCrudPage } from "../components/k8s/yaml-crud-page";
import { listNamespaces as listClusterNamespaces } from "../services/clusters";
import {
  applyJob,
  buildCpuMemoryResourceMaps,
  deleteJob,
  getJobDetail,
  listJobPods,
  listJobs,
  patchJobContainerResources,
  rerunJob,
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
  buildJobYaml,
  jobObjToForm,
  jobYamlToForm,
  qosFromResources,
  type JobFormValues,
} from "../components/k8s/workload-forms";

function JobDetailQuickEdit({
  detail,
  detailYaml,
  setDetailYaml,
}: {
  detail: WorkloadDetail;
  detailYaml: string;
  setDetailYaml: (next: string) => void;
}) {
  const [detailForm] = Form.useForm<JobFormValues>();
  const values = jobYamlToForm(detailYaml || "") ?? jobObjToForm(detail.object) ?? jobYamlToForm(detail.yaml ?? "");
  return (
    <Form
      form={detailForm}
      layout="vertical"
      initialValues={values ?? undefined}
      onValuesChange={(_, allValues) => {
        try {
          setDetailYaml(buildJobYaml(allValues as JobFormValues));
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
        <Form.Item name="restart_policy" label="RestartPolicy" style={{ width: 220 }}>
          <Select options={[{ label: "Never", value: "Never" }, { label: "OnFailure", value: "OnFailure" }]} />
        </Form.Item>
        <Form.Item name="parallelism" label="并行数" style={{ width: 160 }}>
          <InputNumber min={0} style={{ width: "100%" }} />
        </Form.Item>
        <Form.Item name="completions" label="完成数" style={{ width: 160 }}>
          <InputNumber min={0} style={{ width: "100%" }} />
        </Form.Item>
      </Space>
      <Space style={{ width: "100%" }} align="start">
        <Form.Item name="container_name" label="容器名" style={{ width: 220 }}>
          <Input />
        </Form.Item>
        <Form.Item name="image" label="镜像" style={{ flex: 1 }}>
          <Input />
        </Form.Item>
      </Space>
      <Form.Item name="command" label="命令（可选，sh -c）">
        <Input />
      </Form.Item>
    </Form>
  );
}

export function JobsPage() {
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
  const [form] = Form.useForm<JobFormValues>();
  const formActions = useWorkloadFormActions<JobFormValues>({
    form,
    mode: true,
    defaultMode: "create",
    getDetail: async (clusterId, namespace, name) => await getJobDetail(clusterId, namespace, name),
    toFormValues: (d) => jobObjToForm(d.object) ?? jobYamlToForm(d.yaml ?? ""),
    buildFallbackValues: ({ recordName, namespace }) => ({
      name: recordName,
      namespace,
      restart_policy: "Never",
      container_name: recordName,
      image: "",
      env_pairs: [{ key: "", value: "" }],
    }),
  });

  const { openPods, viewer: podsViewer } = useRelatedPodsDrawer(async ({ clusterId, namespace, name }) => await listJobPods(clusterId, namespace, name));
  const { renderKVIcon, viewer } = useKeyValueViewer({ mode: "drawer" });

  const columns: ColumnsType<WorkloadItem> = [
    { title: "命名空间", dataIndex: "namespace", width: 110 },
    { title: "名称", dataIndex: "name", width: 200 },
    { title: "Completions", dataIndex: "replicas", width: 120 },
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
      title: "实时用量",
      key: "usage_rt",
      width: 130,
      render: (_, r) => <RealtimeUsageText cpu={r.cpu_usage} mem={r.mem_usage} />,
    },
    {
      title: "CPU 资源",
      key: "cpu_bars",
      width: 152,
      render: (_, r) => <WorkloadCpuUsageBars row={r} />,
    },
    {
      title: "内存资源",
      key: "mem_bars",
      width: 152,
      render: (_, r) => <WorkloadMemUsageBars row={r} />,
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
    { title: "失败", dataIndex: "failed", width: 120, render: (v?: string) => v ?? "-" },
    { title: "进行中", dataIndex: "active", width: 120, render: (v?: string) => v ?? "-" },
    { title: "开始时间", dataIndex: "start_time", width: 180, render: (v?: string) => v ?? "-" },
    { title: "完成时间", dataIndex: "completion_time", width: 180, render: (v?: string) => v ?? "-" },
    { title: "运行时长", dataIndex: "age", width: 90 },
    { title: "创建时间", dataIndex: "creation_time", width: 180, fixed: "right" },
  ];

  return (
    <>
      <YamlCrudPage<WorkloadItem, WorkloadDetail>
        title="Job 控制器管理"
        needNamespace
        onLoadNamespaces={async (cid) => {
          const res = await listClusterNamespaces(cid);
          return (res.list ?? []).map((n) => ({ label: n.name, value: n.name }));
        }}
        columns={columns}
        onEdit={(record, ctx) => formActions.openEdit({ clusterId: ctx.clusterId, namespace: ctx.namespace ?? "default", name: record.name }, record)}
        api={{
          list: async ({ clusterId, namespace, keyword }) => await listJobs(clusterId, namespace ?? "default", keyword),
          detail: async ({ clusterId, namespace, name }) => await getJobDetail(clusterId, namespace ?? "default", name),
          apply: async ({ clusterId, manifest }) => await applyJob(clusterId, manifest),
          remove: async ({ clusterId, namespace, name }) => await deleteJob(clusterId, namespace ?? "default", name),
        }}
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
              restart_policy: "Never",
              container_name: "",
              image: "busybox:1.36",
              env_pairs: [{ key: "", value: "" }],
            } as Partial<JobFormValues>,
          );
        }}
        renderCreateFormTab={(ctx) => (
          <WorkloadFormModal<JobFormValues>
            embedded
            title="Job 表单创建"
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
                  const manifest = buildJobYaml(values);
                  await applyJob(fctx.clusterId, manifest);
                  message.success("已应用 Job");
                  ctx.closeCreateDrawer();
                  listReloadRef.current();
                } finally {
                  formActions.setLoading(false);
                }
              })();
            }}
          >
            <NameNamespaceItems />
            <ContainerCommonItems showRestartPolicy />
            <WorkloadAdvancedItems />
            <WorkloadPolicyItems showJobPolicy />
            <DeploymentHealthAndImagePullSecretsItems />
            <Form.Item noStyle shouldUpdate>
              {() => {
                const v = form.getFieldsValue();
                const qos = qosFromResources(v);
                return (
                  <Typography.Text type="secondary">
                    QoS 说明：Job 不能直接设置 QoS，QoS 由 resources 推导，当前预估为：{qos}
                  </Typography.Text>
                );
              }}
            </Form.Item>
          </WorkloadFormModal>
        )}
        createTemplate={({ namespace }) => `apiVersion: batch/v1
kind: Job
metadata:
  name: demo-job
  namespace: ${namespace ?? "default"}
spec:
  template:
    spec:
      restartPolicy: Never
      containers:
        - name: demo
          image: busybox:1.36
          command: ["sh", "-c", "echo hello && sleep 5"]
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
                    key: "rerun",
                    label: "重新执行",
                    icon: <PlayCircleOutlined />,
                    onClick: () => {
                      void (async () => {
                        const res = await rerunJob(ctx.clusterId, ctx.namespace ?? "default", record.name);
                        message.success(`已重新执行，创建新 Job：${res.job_name}`);
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
        title={`Job 垂直扩缩（Pod 模板 resources · VPA 类）${verticalTarget ? `：${verticalTarget.name}` : ""}`}
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
            await patchJobContainerResources(verticalTarget.clusterId, verticalTarget.namespace, verticalTarget.name, {
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
          type="info"
          showIcon
          style={{ marginBottom: 12 }}
          message="Job 不属于 HPA「scale 副本」类工作负载；此处仅修改模板 resources。若集群使用 VPA 纳管 Job/CronJob，通常仅在 Initial / Off 等模式下对新建 Pod 更安全，避免运行中任务被不当干扰。"
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

      <WorkloadFormModal<JobFormValues>
        title="Job 表单编辑"
        open={formActions.open && formActions.mode === "edit"}
        loading={formActions.loading}
        form={form}
        onCancel={formActions.close}
        onSubmit={(values) => {
          if (!formActions.ctx) return;
          const fctx = formActions.ctx;
          formActions.setLoading(true);
          void (async () => {
            try {
              const manifest = buildJobYaml(values);
              await applyJob(fctx.clusterId, manifest);
              message.success("已应用 Job");
              formActions.close();
              listReloadRef.current();
            } finally {
              formActions.setLoading(false);
            }
          })();
        }}
      >
        <NameNamespaceItems />
        <ContainerCommonItems showRestartPolicy />
        <WorkloadAdvancedItems />
        <WorkloadPolicyItems showJobPolicy />
        <DeploymentHealthAndImagePullSecretsItems />
        <Form.Item noStyle shouldUpdate>
          {() => {
            const v = form.getFieldsValue();
            const qos = qosFromResources(v);
            return (
              <Typography.Text type="secondary">
                QoS 说明：Job 不能直接设置 QoS，QoS 由 resources 推导，当前预估为：{qos}
              </Typography.Text>
            );
          }}
        </Form.Item>
      </WorkloadFormModal>
    </>
  );
}

