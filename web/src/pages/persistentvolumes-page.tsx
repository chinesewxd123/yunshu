import { Button, Form, Input, Select, Space, Tag, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useRef, useState } from "react";
import { WorkloadFormModal } from "../components/k8s/workload-forms";
import { buildPVYaml, pvYamlToForm, type PersistentVolumeFormValues } from "../components/k8s/service-storage-forms";
import { YamlCrudPage } from "../components/k8s/yaml-crud-page";
import {
  applyPersistentVolume,
  deletePersistentVolume,
  getPersistentVolumeDetail,
  listPersistentVolumes,
  type PersistentVolumeItem,
  type StorageDetail,
} from "../services/storage";

export function PersistentVolumesPage() {
  const listReloadRef = useRef<() => void>(() => {});
  const [formOpen, setFormOpen] = useState(false);
  const [formLoading, setFormLoading] = useState(false);
  const [formCtx, setFormCtx] = useState<{ clusterId: number; name?: string } | null>(null);
  const [formMode, setFormMode] = useState<"create" | "edit">("create");
  const [form] = Form.useForm<PersistentVolumeFormValues>();

  const columns: ColumnsType<PersistentVolumeItem> = [
    { title: "名称", dataIndex: "name", width: 240 },
    { title: "状态", dataIndex: "status", width: 110, render: (v) => <Tag color={v === "Bound" ? "green" : "default"}>{v || "-"}</Tag> },
    { title: "容量", dataIndex: "capacity", width: 110 },
    { title: "访问模式", dataIndex: "access_modes", width: 150 },
    { title: "回收策略", dataIndex: "reclaim_policy", width: 140 },
    { title: "存储类", dataIndex: "storage_class", width: 140 },
    { title: "声明", dataIndex: "claim", width: 200 },
    { title: "创建时间", dataIndex: "creation_time", width: 180, fixed: "right" },
  ];

  return (
    <>
    <YamlCrudPage<PersistentVolumeItem, StorageDetail>
      title="存储 - PersistentVolume"
      needNamespace={false}
      columns={columns}
      showEditButton={false}
      api={{
        list: async ({ clusterId, keyword }) => await listPersistentVolumes(clusterId, keyword),
        detail: async ({ clusterId, name }) => await getPersistentVolumeDetail(clusterId, name),
        apply: async ({ clusterId, manifest }) => await applyPersistentVolume(clusterId, manifest),
        remove: async ({ clusterId, name }) => await deletePersistentVolume(clusterId, name),
      }}
      createTemplate={() => `apiVersion: v1
kind: PersistentVolume
metadata:
  name: demo-pv
spec:
  capacity:
    storage: 1Gi
  accessModes: ["ReadWriteOnce"]
  persistentVolumeReclaimPolicy: Delete
  storageClassName: manual
  hostPath:
    path: /tmp/demo-pv
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
          capacityStorage: "1Gi",
          accessModes: ["ReadWriteOnce"],
          reclaimPolicy: "Delete",
          storageClassName: "manual",
          volumeSourceType: "hostPath",
          hostPath: "/tmp/demo-pv",
        });
      }}
      renderCreateFormTab={(drawerCtx) => (
        <WorkloadFormModal<PersistentVolumeFormValues>
          embedded
          title="PV 表单创建"
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
                await applyPersistentVolume(cid, buildPVYaml(values));
                message.success("PV 已应用");
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
          <Space style={{ width: "100%" }} align="start">
            <Form.Item name="capacityStorage" label="容量" rules={[{ required: true, message: "请输入容量" }]} style={{ width: 220 }}>
              <Input placeholder="1Gi" />
            </Form.Item>
            <Form.Item name="storageClassName" label="StorageClass" style={{ flex: 1 }}>
              <Input />
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
          <Form.Item name="volumeSourceType" label="卷源类型" initialValue="hostPath">
            <Select
              options={[
                { label: "hostPath", value: "hostPath" },
                { label: "nfs", value: "nfs" },
                { label: "local", value: "local" },
              ]}
            />
          </Form.Item>
          <Form.Item noStyle shouldUpdate>
            {() => {
              const t = form.getFieldValue("volumeSourceType") || "hostPath";
              if (t === "nfs") {
                return (
                  <Space style={{ width: "100%" }} align="start">
                    <Form.Item name="nfsServer" label="NFS Server" rules={[{ required: true, message: "请输入 NFS server" }]} style={{ flex: 1 }}>
                      <Input placeholder="10.0.0.10" />
                    </Form.Item>
                    <Form.Item name="nfsPath" label="NFS Path" rules={[{ required: true, message: "请输入 NFS path" }]} style={{ flex: 1 }}>
                      <Input placeholder="/exports/data" />
                    </Form.Item>
                  </Space>
                );
              }
              if (t === "local") {
                return (
                  <Form.Item name="localPath" label="Local Path" rules={[{ required: true, message: "请输入本地路径" }]}>
                    <Input placeholder="/mnt/disks/vol1" />
                  </Form.Item>
                );
              }
              return (
                <Form.Item name="hostPath" label="HostPath" rules={[{ required: true, message: "请输入 hostPath" }]}>
                  <Input placeholder="/tmp/demo-pv" />
                </Form.Item>
              );
            }}
          </Form.Item>
          <Form.Item name="reclaimPolicy" label="回收策略" style={{ width: 220 }}>
            <Select options={[{ label: "Delete", value: "Delete" }, { label: "Retain", value: "Retain" }, { label: "Recycle", value: "Recycle" }]} />
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
                const d = await getPersistentVolumeDetail(ctx.clusterId, record.name);
                const fv = pvYamlToForm(d.yaml ?? "");
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
    <WorkloadFormModal<PersistentVolumeFormValues>
      title="PV 表单编辑"
      open={formOpen && formMode === "edit"}
      loading={formLoading}
      form={form}
      onCancel={() => setFormOpen(false)}
      onSubmit={(values) => {
        if (!formCtx) return;
        setFormLoading(true);
        void (async () => {
          try {
            await applyPersistentVolume(formCtx.clusterId, buildPVYaml(values));
            message.success("PV 已应用");
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
      <Space style={{ width: "100%" }} align="start">
        <Form.Item name="capacityStorage" label="容量" rules={[{ required: true, message: "请输入容量" }]} style={{ width: 220 }}>
          <Input placeholder="1Gi" />
        </Form.Item>
        <Form.Item name="storageClassName" label="StorageClass" style={{ flex: 1 }}>
          <Input />
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
      <Form.Item name="volumeSourceType" label="卷源类型" initialValue="hostPath">
        <Select
          options={[
            { label: "hostPath", value: "hostPath" },
            { label: "nfs", value: "nfs" },
            { label: "local", value: "local" },
          ]}
        />
      </Form.Item>
      <Form.Item noStyle shouldUpdate>
        {() => {
          const t = form.getFieldValue("volumeSourceType") || "hostPath";
          if (t === "nfs") {
            return (
              <Space style={{ width: "100%" }} align="start">
                <Form.Item name="nfsServer" label="NFS Server" rules={[{ required: true, message: "请输入 NFS server" }]} style={{ flex: 1 }}>
                  <Input placeholder="10.0.0.10" />
                </Form.Item>
                <Form.Item name="nfsPath" label="NFS Path" rules={[{ required: true, message: "请输入 NFS path" }]} style={{ flex: 1 }}>
                  <Input placeholder="/exports/data" />
                </Form.Item>
              </Space>
            );
          }
          if (t === "local") {
            return (
              <Form.Item name="localPath" label="Local Path" rules={[{ required: true, message: "请输入本地路径" }]}>
                <Input placeholder="/mnt/disks/vol1" />
              </Form.Item>
            );
          }
          return (
            <Form.Item name="hostPath" label="HostPath" rules={[{ required: true, message: "请输入 hostPath" }]}>
              <Input placeholder="/tmp/demo-pv" />
            </Form.Item>
          );
        }}
      </Form.Item>
      <Form.Item name="reclaimPolicy" label="回收策略" style={{ width: 220 }}>
        <Select options={[{ label: "Delete", value: "Delete" }, { label: "Retain", value: "Retain" }, { label: "Recycle", value: "Recycle" }]} />
      </Form.Item>
    </WorkloadFormModal>
    </>
  );
}

