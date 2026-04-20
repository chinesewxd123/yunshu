import { DownOutlined, EyeOutlined, FileTextOutlined, PlayCircleOutlined, TagsOutlined } from "@ant-design/icons";
import { Button, Dropdown, Form, Input, Progress, Space, Switch, Tag, Typography, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useRef } from "react";
import { useKeyValueViewer } from "../components/k8s/key-value-viewer";
import { useRelatedPodsDrawer } from "../components/k8s/related-pods-drawer";
import { useWorkloadFormActions } from "../components/k8s/workload-form-actions";
import { YamlCrudPage } from "../components/k8s/yaml-crud-page";
import { listNamespaces as listClusterNamespaces } from "../services/clusters";
import {
  applyCronJob,
  deleteCronJob,
  getCronJobDetail,
  listCronJobPods,
  listCronJobsV2,
  suspendCronJob,
  triggerCronJob,
  type CronJobItemV2,
  type WorkloadDetail,
} from "../services/workloads";
import { Select } from "antd";
import {
  CronJobFormValues,
  DeploymentHealthAndImagePullSecretsItems,
  EnvPairsFormItem,
  NameNamespaceItems,
  WorkloadAdvancedItems,
  WorkloadPolicyItems,
  WorkloadFormModal,
  buildCronJobYaml,
  cronJobObjToForm,
  cronJobYamlToForm,
  qosFromResources,
} from "../components/k8s/workload-forms";

function CronJobDetailQuickEdit({
  detail,
  detailYaml,
  setDetailYaml,
}: {
  detail: WorkloadDetail;
  detailYaml: string;
  setDetailYaml: (next: string) => void;
}) {
  const [detailForm] = Form.useForm<CronJobFormValues>();
  const values = cronJobYamlToForm(detailYaml || "") ?? cronJobObjToForm(detail.object) ?? cronJobYamlToForm(detail.yaml ?? "");
  return (
    <Form
      form={detailForm}
      layout="vertical"
      initialValues={values ?? undefined}
      onValuesChange={(_, allValues) => {
        try {
          setDetailYaml(buildCronJobYaml(allValues as CronJobFormValues));
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
        <Form.Item name="schedule" label="Schedule" style={{ flex: 1 }}>
          <Input />
        </Form.Item>
        <Form.Item name="restart_policy" label="RestartPolicy" style={{ width: 220 }}>
          <Select options={[{ label: "Never", value: "Never" }, { label: "OnFailure", value: "OnFailure" }]} />
        </Form.Item>
      </Space>
      <Form.Item name="suspend" label="Suspend" valuePropName="checked">
        <Switch checkedChildren="暂停" unCheckedChildren="运行" />
      </Form.Item>
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

export function CronjobsPage() {
  const listReloadRef = useRef<() => void>(() => {});
  const [form] = Form.useForm<CronJobFormValues>();
  const formActions = useWorkloadFormActions<CronJobFormValues>({
    form,
    mode: true,
    defaultMode: "create",
    getDetail: async (clusterId, namespace, name) => await getCronJobDetail(clusterId, namespace, name),
    toFormValues: (d) => cronJobObjToForm(d.object) ?? cronJobYamlToForm(d.yaml ?? ""),
    buildFallbackValues: ({ recordName, namespace, record }) => ({
      name: recordName,
      namespace,
      schedule: record?.schedule,
      suspend: record?.suspend,
      restart_policy: "Never",
      container_name: recordName,
      image: "",
      env_pairs: [{ key: "", value: "" }],
    }),
  });

  const { openPods, viewer: podsViewer } = useRelatedPodsDrawer(async ({ clusterId, namespace, name }) => await listCronJobPods(clusterId, namespace, name));
  const { renderKVIcon, viewer } = useKeyValueViewer({ mode: "drawer" });

  const columns: ColumnsType<CronJobItemV2> = [
    { title: "命名空间", dataIndex: "namespace", width: 110 },
    { title: "名称", dataIndex: "name", width: 180 },
    { title: "Schedule", dataIndex: "schedule", width: 220 },
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
      title: "状态",
      dataIndex: "suspend",
      width: 120,
      render: (v: boolean) => <Tag color={v ? "default" : "green"}>{v ? "暂停" : "运行"}</Tag>,
    },
    {
      title: "任务活跃度",
      key: "activeProgress",
      width: 150,
      render: (_, r) => {
        const n = Number(r.active_count ?? 0);
        return <Progress percent={Math.min(100, n > 0 ? 100 : 0)} size="small" format={() => `active=${n}`} />;
      },
    },
    { title: "上次调度", dataIndex: "last_schedule_time", width: 180, render: (v?: string) => v || "-" },
    { title: "最近成功", dataIndex: "last_successful_time", width: 180, render: (v?: string) => v || "-" },
    { title: "Active", dataIndex: "active_count", width: 120, render: (v?: string) => <Tag color={v && Number(v) > 0 ? "red" : "default"}>{v || 0}</Tag> },
    { title: "运行时长", dataIndex: "age", width: 90, fixed: "right" },
    { title: "创建时间", dataIndex: "creation_time", width: 180, fixed: "right" },
  ];

  return (
    <>
      <YamlCrudPage<CronJobItemV2, WorkloadDetail>
        title="CronJob 控制器管理"
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
              schedule: "*/5 * * * *",
              suspend: false,
              restart_policy: "Never",
              container_name: "",
              image: "busybox:1.36",
              env_pairs: [{ key: "", value: "" }],
            } as Partial<CronJobFormValues>,
          );
        }}
        renderCreateFormTab={(ctx) => (
          <WorkloadFormModal<CronJobFormValues>
            embedded
            title="CronJob 表单创建"
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
                  const manifest = buildCronJobYaml(values);
                  await applyCronJob(fctx.clusterId, manifest);
                  message.success("已应用 CronJob");
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
              <Form.Item name="schedule" label="Schedule" rules={[{ required: true, message: "请输入 schedule" }]} style={{ flex: 1 }}>
                <Input placeholder='例如：*/5 * * * *' />
              </Form.Item>
              <Form.Item name="restart_policy" label="RestartPolicy" rules={[{ required: true, message: "请选择" }]} style={{ width: 240 }}>
                <Select options={[{ label: "Never", value: "Never" }, { label: "OnFailure", value: "OnFailure" }]} />
              </Form.Item>
            </Space>
            <Form.Item name="suspend" label="Suspend" valuePropName="checked">
              <Switch checkedChildren="暂停" unCheckedChildren="运行" />
            </Form.Item>
            <Space style={{ width: "100%" }} align="start">
              <Form.Item name="container_name" label="容器名" rules={[{ required: true, message: "请输入容器名" }]} style={{ flex: 1 }}>
                <Input />
              </Form.Item>
              <Form.Item name="image" label="镜像" rules={[{ required: true, message: "请输入镜像" }]} style={{ flex: 2 }}>
                <Input placeholder="busybox:1.36" />
              </Form.Item>
            </Space>
            <Form.Item name="command" label="启动命令（可选，sh -c）">
              <Input placeholder='例如：date; echo cron-run' />
            </Form.Item>
            <Form.Item label="环境变量">
              <EnvPairsFormItem name="env_pairs" />
            </Form.Item>
            <WorkloadAdvancedItems />
            <WorkloadPolicyItems showCronJobPolicy />
            <DeploymentHealthAndImagePullSecretsItems />
          </WorkloadFormModal>
        )}
        api={{
          list: async ({ clusterId, namespace, keyword }) => await listCronJobsV2(clusterId, namespace ?? "default", keyword),
          detail: async ({ clusterId, namespace, name }) => await getCronJobDetail(clusterId, namespace ?? "default", name),
          apply: async ({ clusterId, manifest }) => await applyCronJob(clusterId, manifest),
          remove: async ({ clusterId, namespace, name }) => await deleteCronJob(clusterId, namespace ?? "default", name),
        }}
        createTemplate={({ namespace }) => `apiVersion: batch/v1
kind: CronJob
metadata:
  name: demo-cronjob
  namespace: ${namespace ?? "default"}
spec:
  schedule: "*/5 * * * *"
  suspend: false
  jobTemplate:
    spec:
      template:
        spec:
          restartPolicy: Never
          containers:
            - name: demo
              image: busybox:1.36
              command: ["sh", "-c", "date; echo cron-run"]
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
                    key: "trigger",
                    label: "触发",
                    icon: <PlayCircleOutlined />,
                    onClick: () => {
                      void (async () => {
                        const res = await triggerCronJob(ctx.clusterId, ctx.namespace ?? "default", record.name);
                        message.success(`已触发执行，创建 Job：${res.job_name}`);
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
            <Switch
              checkedChildren="运行"
              unCheckedChildren="暂停"
              checked={!record.suspend}
              onChange={(checked) => {
                void (async () => {
                  await suspendCronJob(ctx.clusterId, ctx.namespace ?? "default", record.name, !checked);
                  message.success(checked ? "已恢复 CronJob" : "已暂停 CronJob");
                  ctx.reload();
                })();
              }}
            />
          </Space>
        )}
      />

      {viewer}

      {podsViewer}

      <WorkloadFormModal<CronJobFormValues>
        title={`CronJob 表单编辑${formActions.ctx?.name ? `：${formActions.ctx.name}` : ""}`}
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
              const manifest = buildCronJobYaml(values);
              await applyCronJob(fctx.clusterId, manifest);
              message.success("已应用 CronJob");
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
          <Form.Item name="schedule" label="Schedule" rules={[{ required: true, message: "请输入 schedule" }]} style={{ flex: 1 }}>
            <Input placeholder='例如：*/5 * * * *' />
          </Form.Item>
          <Form.Item name="restart_policy" label="RestartPolicy" rules={[{ required: true, message: "请选择" }]} style={{ width: 240 }}>
            <Select options={[{ label: "Never", value: "Never" }, { label: "OnFailure", value: "OnFailure" }]} />
          </Form.Item>
        </Space>
        <Form.Item name="suspend" label="Suspend" valuePropName="checked">
          <Switch checkedChildren="暂停" unCheckedChildren="运行" />
        </Form.Item>
        <Space style={{ width: "100%" }} align="start">
          <Form.Item name="container_name" label="容器名" rules={[{ required: true, message: "请输入容器名" }]} style={{ flex: 1 }}>
            <Input />
          </Form.Item>
          <Form.Item name="image" label="镜像" rules={[{ required: true, message: "请输入镜像" }]} style={{ flex: 2 }}>
            <Input placeholder="busybox:1.36" />
          </Form.Item>
        </Space>
        <Form.Item name="command" label="启动命令（可选，sh -c）">
          <Input placeholder='例如：date; echo cron-run' />
        </Form.Item>
        <Form.Item label="环境变量">
          <EnvPairsFormItem name="env_pairs" />
        </Form.Item>
        <WorkloadAdvancedItems />
        <WorkloadPolicyItems showCronJobPolicy />
        <DeploymentHealthAndImagePullSecretsItems />
      </WorkloadFormModal>
    </>
  );
}

