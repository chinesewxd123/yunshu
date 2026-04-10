import { Button, Form, Input, Select, Space, Tag, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useState } from "react";
import { WorkloadFormModal } from "../components/k8s/workload-forms";
import { buildPVCYaml, pvcYamlToForm, type PersistentVolumeClaimFormValues } from "../components/k8s/service-storage-forms";
import { YamlCrudPage } from "../components/k8s/yaml-crud-page";
import { listNamespaces as listClusterNamespaces } from "../services/clusters";
import {
  applyPersistentVolumeClaim,
  deletePersistentVolumeClaim,
  getPersistentVolumeClaimDetail,
  listPersistentVolumeClaims,
  listStorageClasses,
  type PersistentVolumeClaimItem,
  type StorageDetail,
} from "../services/storage";

export function PersistentVolumeClaimsPage() {
  const [formOpen, setFormOpen] = useState(false);
  const [formLoading, setFormLoading] = useState(false);
  const [formCtx, setFormCtx] = useState<{ clusterId: number; namespace: string; name?: string } | null>(null);
  const [formMode, setFormMode] = useState<"create" | "edit">("create");
  const [storageClassOptions, setStorageClassOptions] = useState<Array<{ label: string; value: string }>>([]);
  const [form] = Form.useForm<PersistentVolumeClaimFormValues>();

  async function loadStorageClassOptions(clusterId: number) {
    const list = await listStorageClasses(clusterId);
    setStorageClassOptions((list ?? []).map((x) => ({ label: x.name, value: x.name })));
  }

  const columns: ColumnsType<PersistentVolumeClaimItem> = [
    { title: "名称", dataIndex: "name", width: 220 },
    { title: "状态", dataIndex: "status", width: 110, render: (v) => <Tag color={v === "Bound" ? "green" : "default"}>{v || "-"}</Tag> },
    { title: "容量", dataIndex: "capacity", width: 110 },
    { title: "访问模式", dataIndex: "access_modes", width: 150 },
    { title: "卷", dataIndex: "volume", width: 180 },
    { title: "存储类", dataIndex: "storage_class", width: 140 },
    { title: "创建时间", dataIndex: "creation_time", width: 180, fixed: "right" },
  ];

  return (
    <>
    <YamlCrudPage<PersistentVolumeClaimItem, StorageDetail>
      title="存储 - PersistentVolumeClaim"
      needNamespace
      onLoadNamespaces={async (cid) => {
        const res = await listClusterNamespaces(cid);
        return (res.list ?? []).map((n) => ({ label: n.name, value: n.name }));
      }}
      columns={columns}
      showEditButton={false}
      api={{
        list: async ({ clusterId, namespace, keyword }) => await listPersistentVolumeClaims(clusterId, namespace ?? "default", keyword),
        detail: async ({ clusterId, namespace, name }) => await getPersistentVolumeClaimDetail(clusterId, namespace ?? "default", name),
        apply: async ({ clusterId, manifest }) => await applyPersistentVolumeClaim(clusterId, manifest),
        remove: async ({ clusterId, namespace, name }) => await deletePersistentVolumeClaim(clusterId, namespace ?? "default", name),
      }}
      createTemplate={({ namespace }) => `apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: demo-pvc
  namespace: ${namespace ?? "default"}
spec:
  accessModes: ["ReadWriteOnce"]
  resources:
    requests:
      storage: 1Gi
`}
      renderToolbarExtraRight={(ctx) => (
        <Button
          onClick={() => {
            if (!ctx.clusterId) return;
            setFormMode("create");
            setFormCtx({ clusterId: ctx.clusterId, namespace: ctx.namespace ?? "default" });
            void loadStorageClassOptions(ctx.clusterId);
            form.setFieldsValue({
              name: "",
              namespace: ctx.namespace ?? "default",
              accessModes: ["ReadWriteOnce"],
              requestStorage: "1Gi",
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
                await loadStorageClassOptions(ctx.clusterId);
                const d = await getPersistentVolumeClaimDetail(ctx.clusterId, ctx.namespace ?? "default", record.name);
                const fv = pvcYamlToForm(d.yaml ?? "");
                if (fv) form.setFieldsValue(fv);
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
    <WorkloadFormModal<PersistentVolumeClaimFormValues>
      title={formMode === "create" ? "PVC 表单创建" : "PVC 表单编辑"}
      open={formOpen}
      loading={formLoading}
      form={form}
      onCancel={() => setFormOpen(false)}
      onSubmit={(values) => {
        if (!formCtx) return;
        setFormLoading(true);
        void (async () => {
          try {
            await applyPersistentVolumeClaim(formCtx.clusterId, buildPVCYaml(values));
            message.success("PVC 已应用");
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
        <Form.Item name="storageClassName" label="StorageClass" style={{ flex: 1 }}>
          <Select
            allowClear
            showSearch
            optionFilterProp="label"
            options={storageClassOptions}
            placeholder="选择 StorageClass"
          />
        </Form.Item>
        <Form.Item name="requestStorage" label="请求容量" rules={[{ required: true, message: "请输入容量" }]} style={{ width: 220 }}>
          <Input placeholder="1Gi" />
        </Form.Item>
      </Space>
      <Form.Item name="accessModes" label="访问模式">
        <Select
          mode="multiple"
          options={[
            { label: "ReadWriteOnce", value: "ReadWriteOnce" },
            { label: "ReadOnlyMany", value: "ReadOnlyMany" },
            { label: "ReadWriteMany", value: "ReadWriteMany" },
            { label: "ReadWriteOncePod", value: "ReadWriteOncePod" },
          ]}
        />
      </Form.Item>
      <Form.Item name="volumeName" label="绑定 PV（可选）">
        <Input />
      </Form.Item>
    </WorkloadFormModal>
    </>
  );
}

