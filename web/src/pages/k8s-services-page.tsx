import { ApartmentOutlined, FileTextOutlined, TagsOutlined } from "@ant-design/icons";
import { Button, Form, Input, Modal, Select, Space, Table, Tag, Typography, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useRef, useState } from "react";
import { useKeyValueViewer } from "../components/k8s/key-value-viewer";
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
  const listReloadRef = useRef<() => void>(() => {});
  const [formOpen, setFormOpen] = useState(false);
  const [formLoading, setFormLoading] = useState(false);
  const [formCtx, setFormCtx] = useState<{ clusterId: number; namespace: string; name?: string } | null>(null);
  const [formMode, setFormMode] = useState<"create" | "edit">("create");
  const [workloadOptions, setWorkloadOptions] = useState<Array<{ label: string; value: string }>>([]);
  const [recommendedPortNames, setRecommendedPortNames] = useState<string[]>([]);
  const { renderKVIcon, viewer } = useKeyValueViewer({
    width: 760,
    compact: true,
    pageSize: 10,
    destroyOnClose: true,
    emptyText: (title) => `暂无${title}`,
  });
  const [form] = Form.useForm<ServiceFormValues>();

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

  function extractRecommendedPortNamesFromWorkload(detailObj: any): string[] {
    const containerPorts = detailObj?.spec?.template?.spec?.containers?.[0]?.ports;
    if (!Array.isArray(containerPorts)) return [];
    return containerPorts
      .map((p: any) => String(p?.name ?? "").trim())
      .filter((x: string) => !!x);
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
    const recommended = extractRecommendedPortNamesFromWorkload(detailObj);
    setRecommendedPortNames(recommended);
    const fallbackTargetPort = recommended[0];
    const currentPorts = (form.getFieldValue("ports") as ServiceFormValues["ports"]) ?? [];
    const nextPorts =
      fallbackTargetPort && currentPorts.length
        ? currentPorts.map((p) => {
            const current = String(p?.targetPort ?? "").trim();
            if (current) return p;
            return { ...p, targetPort: fallbackTargetPort };
          })
        : currentPorts;
    form.setFieldsValue({
      selector_pairs: pairs.length ? pairs : [{ key: "app", value: name }],
      ports: nextPorts,
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
      onToolbarReady={(ctx) => {
        listReloadRef.current = ctx.reload;
      }}
      onCreateDrawerOpen={(ctx) => {
        if (!ctx.clusterId) return;
        setFormMode("create");
        setFormCtx({ clusterId: ctx.clusterId, namespace: ctx.namespace ?? "default" });
        setRecommendedPortNames([]);
        void loadWorkloadsForSelector(ctx.clusterId, ctx.namespace ?? "default");
        form.setFieldsValue({
          name: "",
          namespace: ctx.namespace ?? "default",
          type: "ClusterIP",
          selector_pairs: [{ key: "app", value: "" }],
          ports: [{ protocol: "TCP", port: 80, targetPort: "80" }],
        });
      }}
      renderCreateFormTab={(drawerCtx) => (
        <WorkloadFormModal<ServiceFormValues>
          embedded
          title="Service 表单创建"
          open={false}
          loading={formLoading}
          form={form}
          onCancel={drawerCtx.closeCreateDrawer}
          onSubmit={(values) => {
            const cid = drawerCtx.clusterId;
            if (!cid) return;
            setFormLoading(true);
            void (async () => {
              try {
                await applyK8sService(cid, buildServiceYaml(values));
                message.success("Service 已应用");
                drawerCtx.closeCreateDrawer();
                listReloadRef.current();
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
                  if (!val || !drawerCtx.clusterId) return;
                  void backfillSelectorFromWorkload(
                    drawerCtx.clusterId,
                    form.getFieldValue("namespace") || drawerCtx.namespace || "default",
                    String(val),
                  );
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
          <Space style={{ width: "100%" }} align="start">
            <Form.Item name="sessionAffinity" label="会话亲和性" style={{ width: 220 }}>
              <Select allowClear options={[{ label: "None", value: "None" }, { label: "ClientIP", value: "ClientIP" }]} />
            </Form.Item>
            <Form.Item name="externalTrafficPolicy" label="外部流量策略" style={{ width: 220 }}>
              <Select allowClear options={[{ label: "Cluster", value: "Cluster" }, { label: "Local", value: "Local" }]} />
            </Form.Item>
            <Form.Item name="internalTrafficPolicy" label="内部流量策略" style={{ width: 220 }}>
              <Select allowClear options={[{ label: "Cluster", value: "Cluster" }, { label: "Local", value: "Local" }]} />
            </Form.Item>
          </Space>
          <Space style={{ width: "100%" }} align="start">
            <Form.Item name="ipFamilyPolicy" label="IP Family Policy" style={{ width: 220 }}>
              <Select
                allowClear
                options={[
                  { label: "SingleStack", value: "SingleStack" },
                  { label: "PreferDualStack", value: "PreferDualStack" },
                  { label: "RequireDualStack", value: "RequireDualStack" },
                ]}
              />
            </Form.Item>
            <Form.Item name="ipFamilies" label="IP Families" style={{ width: 280 }}>
              <Select mode="multiple" allowClear options={[{ label: "IPv4", value: "IPv4" }, { label: "IPv6", value: "IPv6" }]} />
            </Form.Item>
            <Form.Item name="healthCheckNodePort" label="HealthCheck NodePort" style={{ width: 220 }}>
              <Input type="number" min={1} max={65535} />
            </Form.Item>
          </Space>
          <Form.Item name="loadBalancerSourceRanges" label="LoadBalancer Source Ranges">
            <Select mode="tags" tokenSeparators={[",", " "]} placeholder="例如 10.0.0.0/24" />
          </Form.Item>
          <Form.Item label="Selector">
            <LabelsFormList name="selector_pairs" addLabel="新增 Selector" />
          </Form.Item>
          <Form.Item label="Ports">
            <ServicePortsFormList recommendedPortNames={recommendedPortNames} />
          </Form.Item>
        </WorkloadFormModal>
      )}
      extraRowActions={(record, ctx) => (
        <Button
          type="link"
          onClick={() => {
            setFormMode("edit");
            setFormCtx({ clusterId: ctx.clusterId, namespace: ctx.namespace ?? "default", name: record.name });
            setFormOpen(true);
            setFormLoading(true);
            setRecommendedPortNames([]);
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
      title="Service 表单编辑"
      open={formOpen && formMode === "edit"}
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
            listReloadRef.current();
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
      <Space style={{ width: "100%" }} align="start">
        <Form.Item name="sessionAffinity" label="会话亲和性" style={{ width: 220 }}>
          <Select allowClear options={[{ label: "None", value: "None" }, { label: "ClientIP", value: "ClientIP" }]} />
        </Form.Item>
        <Form.Item name="externalTrafficPolicy" label="外部流量策略" style={{ width: 220 }}>
          <Select allowClear options={[{ label: "Cluster", value: "Cluster" }, { label: "Local", value: "Local" }]} />
        </Form.Item>
        <Form.Item name="internalTrafficPolicy" label="内部流量策略" style={{ width: 220 }}>
          <Select allowClear options={[{ label: "Cluster", value: "Cluster" }, { label: "Local", value: "Local" }]} />
        </Form.Item>
      </Space>
      <Space style={{ width: "100%" }} align="start">
        <Form.Item name="ipFamilyPolicy" label="IP Family Policy" style={{ width: 220 }}>
          <Select
            allowClear
            options={[
              { label: "SingleStack", value: "SingleStack" },
              { label: "PreferDualStack", value: "PreferDualStack" },
              { label: "RequireDualStack", value: "RequireDualStack" },
            ]}
          />
        </Form.Item>
        <Form.Item name="ipFamilies" label="IP Families" style={{ width: 280 }}>
          <Select mode="multiple" allowClear options={[{ label: "IPv4", value: "IPv4" }, { label: "IPv6", value: "IPv6" }]} />
        </Form.Item>
        <Form.Item name="healthCheckNodePort" label="HealthCheck NodePort" style={{ width: 220 }}>
          <Input type="number" min={1} max={65535} />
        </Form.Item>
      </Space>
      <Form.Item name="loadBalancerSourceRanges" label="LoadBalancer Source Ranges">
        <Select mode="tags" tokenSeparators={[",", " "]} placeholder="例如 10.0.0.0/24" />
      </Form.Item>
      <Form.Item label="Selector">
        <LabelsFormList name="selector_pairs" addLabel="新增 Selector" />
      </Form.Item>
      <Form.Item label="Ports">
        <ServicePortsFormList recommendedPortNames={recommendedPortNames} />
      </Form.Item>
    </WorkloadFormModal>
    {viewer}
    </>
  );
}

