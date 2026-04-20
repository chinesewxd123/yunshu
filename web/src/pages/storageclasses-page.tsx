import { Button, Form, Input, Select, Space, Switch, Tag, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useRef, useState } from "react";
import { LabelsFormList, buildStorageClassYaml, storageClassYamlToForm, type StorageClassFormValues } from "../components/k8s/service-storage-forms";
import { WorkloadFormModal } from "../components/k8s/workload-forms";
import { YamlCrudPage } from "../components/k8s/yaml-crud-page";
import {
  applyStorageClass,
  deleteStorageClass,
  getStorageClassDetail,
  listStorageClasses,
  type StorageClassItem,
  type StorageDetail,
} from "../services/storage";

export function StorageClassesPage() {
  const listReloadRef = useRef<() => void>(() => {});
  const [formOpen, setFormOpen] = useState(false);
  const [formLoading, setFormLoading] = useState(false);
  const [formCtx, setFormCtx] = useState<{ clusterId: number; name?: string } | null>(null);
  const [formMode, setFormMode] = useState<"create" | "edit">("create");
  const [form] = Form.useForm<StorageClassFormValues>();

  const columns: ColumnsType<StorageClassItem> = [
    { title: "名称", dataIndex: "name", width: 240 },
    { title: "Provisioner", dataIndex: "provisioner", width: 260 },
    { title: "回收策略", dataIndex: "reclaim_policy", width: 130 },
    { title: "绑定模式", dataIndex: "volume_binding_mode", width: 130 },
    {
      title: "扩容",
      dataIndex: "allow_volume_expansion",
      width: 90,
      render: (v: boolean) => <Tag color={v ? "green" : "default"}>{v ? "允许" : "否"}</Tag>,
    },
    { title: "创建时间", dataIndex: "creation_time", width: 180, fixed: "right" },
  ];

  return (
    <>
    <YamlCrudPage<StorageClassItem, StorageDetail>
      title="存储 - StorageClass"
      needNamespace={false}
      columns={columns}
      showEditButton={false}
      api={{
        list: async ({ clusterId, keyword }) => await listStorageClasses(clusterId, keyword),
        detail: async ({ clusterId, name }) => await getStorageClassDetail(clusterId, name),
        apply: async ({ clusterId, manifest }) => await applyStorageClass(clusterId, manifest),
        remove: async ({ clusterId, name }) => await deleteStorageClass(clusterId, name),
      }}
      createTemplate={() => `apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: demo-sc
provisioner: kubernetes.io/no-provisioner
volumeBindingMode: WaitForFirstConsumer
reclaimPolicy: Delete
allowVolumeExpansion: true
`}
      onToolbarReady={(ctx) => {
        listReloadRef.current = ctx.reload;
      }}
      onCreateDrawerOpen={(ctx) => {
        if (!ctx.clusterId) return;
        setFormMode("create");
        setFormCtx({ clusterId: ctx.clusterId });
        form.setFieldsValue({
          name: "",
          provisioner: "kubernetes.io/no-provisioner",
          reclaimPolicy: "Delete",
          volumeBindingMode: "WaitForFirstConsumer",
          allowVolumeExpansion: true,
          mountOptions: [],
          params: [],
        });
      }}
      renderCreateFormTab={(drawerCtx) => (
        <WorkloadFormModal<StorageClassFormValues>
          embedded
          title="StorageClass 表单创建"
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
                await applyStorageClass(cid, buildStorageClassYaml(values));
                message.success("StorageClass 已应用");
                drawerCtx.closeCreateDrawer();
                listReloadRef.current();
              } finally {
                setFormLoading(false);
              }
            })();
          }}
        >
          <Form.Item name="name" label="名称" rules={[{ required: true, message: "请输入名称" }]}>
            <Input />
          </Form.Item>
          <Form.Item name="provisioner" label="Provisioner" rules={[{ required: true, message: "请输入 provisioner" }]}>
            <Input />
          </Form.Item>
          <Space style={{ width: "100%" }} align="start">
            <Form.Item name="reclaimPolicy" label="回收策略" style={{ width: 220 }}>
              <Select options={[{ label: "Delete", value: "Delete" }, { label: "Retain", value: "Retain" }]} />
            </Form.Item>
            <Form.Item name="volumeBindingMode" label="绑定模式" style={{ width: 260 }}>
              <Select options={[{ label: "Immediate", value: "Immediate" }, { label: "WaitForFirstConsumer", value: "WaitForFirstConsumer" }]} />
            </Form.Item>
            <Form.Item name="allowVolumeExpansion" label="允许扩容" valuePropName="checked">
              <Switch />
            </Form.Item>
          </Space>
          <Form.Item name="mountOptions" label="MountOptions">
            <Select mode="tags" tokenSeparators={[",", " "]} placeholder="例如 noatime,nodiratime" />
          </Form.Item>
          <Form.Item label="Parameters">
            <LabelsFormList name="params" addLabel="新增参数" />
          </Form.Item>
        </WorkloadFormModal>
      )}
      extraRowActions={(record, ctx) => (
        <Button
          type="link"
          onClick={() => {
            setFormMode("edit");
            setFormCtx({ clusterId: ctx.clusterId, name: record.name });
            setFormOpen(true);
            setFormLoading(true);
            void (async () => {
              try {
                const d = await getStorageClassDetail(ctx.clusterId, record.name);
                const fv = storageClassYamlToForm(d.yaml ?? "");
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
    <WorkloadFormModal<StorageClassFormValues>
      title="StorageClass 表单编辑"
      open={formOpen && formMode === "edit"}
      loading={formLoading}
      form={form}
      onCancel={() => setFormOpen(false)}
      onSubmit={(values) => {
        if (!formCtx) return;
        setFormLoading(true);
        void (async () => {
          try {
            await applyStorageClass(formCtx.clusterId, buildStorageClassYaml(values));
            message.success("StorageClass 已应用");
            setFormOpen(false);
            listReloadRef.current();
          } finally {
            setFormLoading(false);
          }
        })();
      }}
    >
      <Form.Item name="name" label="名称" rules={[{ required: true, message: "请输入名称" }]}>
        <Input />
      </Form.Item>
      <Form.Item name="provisioner" label="Provisioner" rules={[{ required: true, message: "请输入 provisioner" }]}>
        <Input />
      </Form.Item>
      <Space style={{ width: "100%" }} align="start">
        <Form.Item name="reclaimPolicy" label="回收策略" style={{ width: 220 }}>
          <Select options={[{ label: "Delete", value: "Delete" }, { label: "Retain", value: "Retain" }]} />
        </Form.Item>
        <Form.Item name="volumeBindingMode" label="绑定模式" style={{ width: 260 }}>
          <Select options={[{ label: "Immediate", value: "Immediate" }, { label: "WaitForFirstConsumer", value: "WaitForFirstConsumer" }]} />
        </Form.Item>
        <Form.Item name="allowVolumeExpansion" label="允许扩容" valuePropName="checked">
          <Switch />
        </Form.Item>
      </Space>
      <Form.Item name="mountOptions" label="MountOptions">
        <Select mode="tags" tokenSeparators={[",", " "]} placeholder="例如 noatime,nodiratime" />
      </Form.Item>
      <Form.Item label="Parameters">
        <LabelsFormList name="params" addLabel="新增参数" />
      </Form.Item>
    </WorkloadFormModal>
    </>
  );
}

