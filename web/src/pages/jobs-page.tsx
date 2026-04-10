import { DownOutlined, EditOutlined, EyeOutlined, FileTextOutlined, PlayCircleOutlined, TagsOutlined } from "@ant-design/icons";
import { Button, Drawer, Dropdown, Form, Progress, Space, Table, Tag, Tooltip, Typography, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useState } from "react";
import { YamlCrudPage } from "../components/k8s/yaml-crud-page";
import { listNamespaces as listClusterNamespaces } from "../services/clusters";
import {
  applyJob,
  deleteJob,
  getJobDetail,
  listJobPods,
  listJobs,
  rerunJob,
  type RelatedPodItem,
  type WorkloadDetail,
  type WorkloadItem,
} from "../services/workloads";
import {
  WorkloadFormModal,
  NameNamespaceItems,
  ContainerCommonItems,
  WorkloadAdvancedItems,
  buildJobYaml,
  jobObjToForm,
  jobYamlToForm,
  qosFromResources,
  type JobFormValues,
} from "../components/k8s/workload-forms";

export function JobsPage() {
  const [formOpen, setFormOpen] = useState(false);
  const [formLoading, setFormLoading] = useState(false);
  const [formMode, setFormMode] = useState<"create" | "edit">("create");
  const [formCtx, setFormCtx] = useState<{ clusterId: number; namespace: string; name?: string } | null>(null);
  const [form] = Form.useForm<JobFormValues>();

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
        showEditButton={false}
        api={{
          list: async ({ clusterId, namespace, keyword }) => await listJobs(clusterId, namespace ?? "default", keyword),
          detail: async ({ clusterId, namespace, name }) => await getJobDetail(clusterId, namespace ?? "default", name),
          apply: async ({ clusterId, manifest }) => await applyJob(clusterId, manifest),
          remove: async ({ clusterId, namespace, name }) => await deleteJob(clusterId, namespace ?? "default", name),
        }}
        renderToolbarExtraRight={undefined}
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
                    onClick: () => {
                      setPodsTarget({ clusterId: ctx.clusterId, namespace: ctx.namespace ?? "default", name: record.name });
                      setPodsOpen(true);
                      setPodsLoading(true);
                      void (async () => {
                        try {
                          const items = await listJobPods(ctx.clusterId, ctx.namespace ?? "default", record.name);
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
                      setFormMode("edit");
                      setFormCtx({ clusterId: ctx.clusterId, namespace: ctx.namespace ?? "default", name: record.name });
                      setFormOpen(true);
                      setFormLoading(true);
                      void (async () => {
                        try {
                          const d = await getJobDetail(ctx.clusterId, ctx.namespace ?? "default", record.name);
                          const fv = jobObjToForm(d.object) ?? jobYamlToForm(d.yaml ?? "");
                          if (fv) {
                            form.setFieldsValue({ ...fv, namespace: ctx.namespace ?? fv.namespace } as any);
                          } else {
                            form.setFieldsValue({
                              name: record.name,
                              namespace: ctx.namespace ?? "default",
                              restart_policy: "Never",
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
            { title: "状态", dataIndex: "phase", width: 120, render: (v: string) => <Tag color={v === "Running" ? "green" : "default"}>{v || "-"}</Tag> },
            { title: "节点", dataIndex: "node_name", width: 160 },
            { title: "PodIP", dataIndex: "pod_ip", width: 140 },
            { title: "重启", dataIndex: "restart_count", width: 90 },
            { title: "启动时间", dataIndex: "start_time", width: 180 },
          ]}
        />
      </Drawer>

      <WorkloadFormModal<JobFormValues>
        title={formMode === "create" ? "Job 表单创建" : "Job 表单编辑"}
        open={formOpen}
        loading={formLoading}
        form={form}
        onCancel={() => setFormOpen(false)}
        onSubmit={(values) => {
          if (!formCtx) return;
          setFormLoading(true);
          void (async () => {
            try {
              const manifest = buildJobYaml(values);
              await applyJob(formCtx.clusterId, manifest);
              message.success("已应用 Job");
              setFormOpen(false);
            } finally {
              setFormLoading(false);
            }
          })();
        }}
      >
        <NameNamespaceItems />
        <ContainerCommonItems showRestartPolicy />
        <WorkloadAdvancedItems />
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

