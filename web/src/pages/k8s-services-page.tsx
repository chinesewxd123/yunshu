import { ApartmentOutlined, FileTextOutlined, TagsOutlined } from "@ant-design/icons";
import { Button, Form, Input, Modal, Select, Space, Table, Tag, Tooltip, Typography, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useState } from "react";
import { WorkloadFormModal } from "../components/k8s/workload-forms";
import { LabelsFormList, ServicePortsFormList, buildServiceYaml, serviceYamlToForm, type ServiceFormValues } from "../components/k8s/service-storage-forms";
import { YamlCrudPage } from "../components/k8s/yaml-crud-page";
import { listNamespaces as listClusterNamespaces } from "../services/clusters";
import { getDaemonSetDetail, getDeploymentDetail, getStatefulSetDetail, listDaemonSets, listDeployments, listStatefulSets } from "../services/workloads";
import {
  applyK8sService,
  deleteK8sService,
  getK8sServiceDetail,
  listK8sServices,
  type K8sServiceDetail,
  type K8sServiceItem,
} from "../services/k8s-services";

export function K8sServicesPage() {
  const [formOpen, setFormOpen] = useState(false);
  const [formLoading, setFormLoading] = useState(false);
  const [formCtx, setFormCtx] = useState<{ clusterId: number; namespace: string; name?: string } | null>(null);
  const [formMode, setFormMode] = useState<"create" | "edit">("create");
  const [workloadOptions, setWorkloadOptions] = useState<Array<{ label: string; value: string }>>([]);
  const [kvOpen, setKvOpen] = useState(false);
  const [kvTitle, setKvTitle] = useState("详情");
  const [kvData, setKvData] = useState<Record<string, string>>({});
  const [form] = Form.useForm<ServiceFormValues>();

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

  async function loadWorkloadsForSelector(clusterId: number, namespace: string) {
    const [deps, sts, dss] = await Promise.all([
      listDeployments(clusterId, namespace),
      listStatefulSets(clusterId, namespace),
      listDaemonSets(clusterId, namespace),
    ]);
    const opts = [
      ...(deps ?? []).map((x) => ({ label: `Deployment/${x.name}`, value: `deployment:${x.name}` })),
      ...(sts ?? []).map((x) => ({ label: `StatefulSet/${x.name}`, value: `statefulset:${x.name}` })),
      ...(dss ?? []).map((x) => ({ label: `DaemonSet/${x.name}`, value: `daemonset:${x.name}` })),
    ];
    setWorkloadOptions(opts);
  }

  async function backfillSelectorFromWorkload(clusterId: number, namespace: string, ref: string) {
    const [kind, name] = ref.split(":");
    if (!kind || !name) return;
    let detailObj: any = null;
    if (kind === "deployment") {
      detailObj = (await getDeploymentDetail(clusterId, namespace, name)).object;
    } else if (kind === "statefulset") {
      detailObj = (await getStatefulSetDetail(clusterId, namespace, name)).object;
    } else if (kind === "daemonset") {
      detailObj = (await getDaemonSetDetail(clusterId, namespace, name)).object;
    }
    const matchLabels = detailObj?.spec?.selector?.matchLabels ?? detailObj?.spec?.template?.metadata?.labels ?? {};
    const pairs = Object.entries(matchLabels).map(([k, v]) => ({ key: k, value: String(v ?? "") }));
    form.setFieldsValue({
      selector_pairs: pairs.length ? pairs : [{ key: "app", value: name }],
    });
  }

  const columns: ColumnsType<K8sServiceItem> = [
    { title: "命名空间", dataIndex: "namespace", width: 120 },
    { title: "名称", dataIndex: "name", width: 220 },
    { title: "访问类型", dataIndex: "type", width: 100, render: (v) => <Tag color="green">{v || "-"}</Tag> },
    { title: "内部流量策略", dataIndex: "internal_traffic", width: 120 },
    { title: "访问地址", dataIndex: "cluster_ip", width: 130 },
    { title: "外部地址", dataIndex: "external_ips", width: 160, ellipsis: true },
    { title: "访问端口", dataIndex: "ports", width: 220, ellipsis: true },
    { title: "IP协议族", dataIndex: "ip_families", width: 110 },
    { title: "IP协议族策略", dataIndex: "ip_family_policy", width: 120 },
    { title: "标签", key: "labels", width: 70, align: "center", render: (_, r) => renderKVIcon("标签", <TagsOutlined />, r.labels) },
    { title: "注解", key: "annotations", width: 70, align: "center", render: (_, r) => renderKVIcon("注解", <FileTextOutlined />, r.annotations) },
    { title: "POD选择器", key: "selectors", width: 90, align: "center", render: (_, r) => renderKVIcon("POD选择器", <ApartmentOutlined />, r.selectors) },
    { title: "会话亲和性", dataIndex: "session_affinity", width: 100 },
    { title: "存在时长", dataIndex: "age", width: 90, fixed: "right" },
    { title: "创建时间", dataIndex: "creation_time", width: 170, fixed: "right" },
  ];

  return (
    <>
    <YamlCrudPage<K8sServiceItem, K8sServiceDetail>
      title="Service 管理"
      needNamespace
      onLoadNamespaces={async (cid) => {
        const res = await listClusterNamespaces(cid);
        return (res.list ?? []).map((n) => ({ label: n.name, value: n.name }));
      }}
      columns={columns}
      showEditButton={false}
      api={{
        list: async ({ clusterId, namespace, keyword }) => await listK8sServices(clusterId, namespace ?? "default", keyword),
        detail: async ({ clusterId, namespace, name }) => await getK8sServiceDetail(clusterId, namespace ?? "default", name),
        apply: async ({ clusterId, manifest }) => await applyK8sService(clusterId, manifest),
        remove: async ({ clusterId, namespace, name }) => await deleteK8sService(clusterId, namespace ?? "default", name),
      }}
      createTemplate={({ namespace }) => `apiVersion: v1
kind: Service
metadata:
  name: demo-service
  namespace: ${namespace ?? "default"}
spec:
  selector:
    app: demo
  ports:
    - port: 80
      targetPort: 8080
      protocol: TCP
  type: ClusterIP
`}
      renderToolbarExtraRight={(ctx) => (
        <Button
          onClick={() => {
            if (!ctx.clusterId) return;
            setFormMode("create");
            setFormCtx({ clusterId: ctx.clusterId, namespace: ctx.namespace ?? "default" });
            void loadWorkloadsForSelector(ctx.clusterId, ctx.namespace ?? "default");
            form.setFieldsValue({
              name: "",
              namespace: ctx.namespace ?? "default",
              type: "ClusterIP",
              selector_pairs: [{ key: "app", value: "" }],
              ports: [{ protocol: "TCP", port: 80, targetPort: "80" }],
            });
            setFormOpen(true);
          }}
        >
          表单创建
        </Button>
      )}
      extraRowActions={(record, ctx) => (
        <Button
          type="link"
          onClick={() => {
            setFormMode("edit");
            setFormCtx({ clusterId: ctx.clusterId, namespace: ctx.namespace ?? "default", name: record.name });
            setFormOpen(true);
            setFormLoading(true);
            void (async () => {
              try {
                await loadWorkloadsForSelector(ctx.clusterId, ctx.namespace ?? "default");
                const d = await getK8sServiceDetail(ctx.clusterId, ctx.namespace ?? "default", record.name);
                const fv = serviceYamlToForm(d.yaml ?? "");
                if (fv) {
                  form.setFieldsValue(fv);
                }
              } finally {
                setFormLoading(false);
              }
            })();
          }}
        >
          表单编辑
        </Button>
      )}
    />
    <WorkloadFormModal<ServiceFormValues>
      title={formMode === "create" ? "Service 表单创建" : "Service 表单编辑"}
      open={formOpen}
      loading={formLoading}
      form={form}
      onCancel={() => setFormOpen(false)}
      onSubmit={(values) => {
        if (!formCtx) return;
        setFormLoading(true);
        void (async () => {
          try {
            await applyK8sService(formCtx.clusterId, buildServiceYaml(values));
            message.success("Service 已应用");
            setFormOpen(false);
          } finally {
            setFormLoading(false);
          }
        })();
      }}
    >
      <Space style={{ width: "100%" }} align="start">
        <Form.Item name="name" label="名称" rules={[{ required: true, message: "请输入名称" }]} style={{ flex: 1 }}>
          <Input />
        </Form.Item>
        <Form.Item name="namespace" label="命名空间" rules={[{ required: true, message: "请输入命名空间" }]} style={{ width: 220 }}>
          <Input />
        </Form.Item>
      </Space>
      <Space style={{ width: "100%" }} align="start">
        <Form.Item label="关联 Workload（自动回填 selector）" style={{ flex: 1 }}>
          <Select
            allowClear
            options={workloadOptions}
            placeholder="选择 Deployment/StatefulSet/DaemonSet"
            onChange={(val) => {
              if (!val || !formCtx) return;
              void backfillSelectorFromWorkload(formCtx.clusterId, form.getFieldValue("namespace") || formCtx.namespace, String(val));
            }}
          />
        </Form.Item>
      </Space>
      <Space style={{ width: "100%" }} align="start">
        <Form.Item name="type" label="类型" style={{ width: 220 }}>
          <Select
            options={[
              { label: "ClusterIP", value: "ClusterIP" },
              { label: "NodePort", value: "NodePort" },
              { label: "LoadBalancer", value: "LoadBalancer" },
              { label: "ExternalName", value: "ExternalName" },
            ]}
          />
        </Form.Item>
        <Form.Item name="externalName" label="ExternalName（可选）" style={{ flex: 1 }}>
          <Input placeholder="example.com" />
        </Form.Item>
      </Space>
      <Form.Item label="Selector">
        <LabelsFormList name="selector_pairs" addLabel="新增 Selector" />
      </Form.Item>
      <Form.Item label="Ports">
        <ServicePortsFormList />
      </Form.Item>
    </WorkloadFormModal>
    <Modal
      title={kvTitle}
      open={kvOpen}
      onCancel={() => setKvOpen(false)}
      footer={null}
      width={760}
      destroyOnClose
    >
      <Table
        size="small"
        rowKey={(r) => r.key}
        pagination={{ pageSize: 10 }}
        dataSource={Object.entries(kvData).map(([key, value]) => ({ key, value }))}
        locale={{ emptyText: `暂无${kvTitle}` }}
        columns={[
          {
            title: "Key",
            dataIndex: "key",
            width: 280,
            render: (v: string) => (
              <Typography.Text copyable style={{ whiteSpace: "pre-wrap" }}>
                {v || "-"}
              </Typography.Text>
            ),
          },
          {
            title: "Value",
            dataIndex: "value",
            render: (v: string) => (
              <Typography.Text copyable style={{ whiteSpace: "pre-wrap" }}>
                {v || "-"}
              </Typography.Text>
            ),
          },
        ]}
      />
    </Modal>
    </>
  );
}

