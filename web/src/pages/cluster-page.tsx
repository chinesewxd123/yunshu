import {
  CheckCircleOutlined,
  ClusterOutlined,
  CloseCircleOutlined,
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
  ReloadOutlined,
  SettingOutlined,
} from "@ant-design/icons";
import { Button, Card, Form, Input, Modal, Popconfirm, Space, Table, Tag, Typography, message } from "antd";
import { useEffect, useMemo, useState } from "react";
import { formatDateTime } from "../utils/format";
import { createCluster, deleteCluster, getClusterStatus, getClusters, setClusterStatus, updateCluster, type ClusterItem, type ClusterCreatePayload, type ClusterUpdatePayload } from "../services/clusters";

type ClusterQuery = {
  keyword: string;
  page: number;
  page_size: number;
};

const defaultQuery: ClusterQuery = { keyword: "", page: 1, page_size: 10 };

function phaseColor(phase: string): string {
  const p = (phase || "").toLowerCase();
  if (p === "running") return "green";
  if (p === "pending") return "orange";
  if (p === "failed") return "red";
  if (p === "succeeded") return "blue";
  return "default";
}

export function ClusterPage() {
  const [query, setQuery] = useState<ClusterQuery>(defaultQuery);
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [statusUpdatingID, setStatusUpdatingID] = useState<number | null>(null);

  const [list, setList] = useState<ClusterItem[]>([]);
  const [total, setTotal] = useState(0);

  const [serverVersionByID, setServerVersionByID] = useState<Record<number, string>>({});

  const [modalOpen, setModalOpen] = useState(false);
  const [current, setCurrent] = useState<ClusterItem | null>(null);
  const [form] = Form.useForm<ClusterCreatePayload & Partial<ClusterUpdatePayload>>();

  useEffect(() => {
    void loadClusters();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [query]);

  async function loadClusters() {
    setLoading(true);
    try {
      const result = await getClusters(query);
      setList(result.list);
      setTotal(result.total);
    } finally {
      setLoading(false);
    }
  }

  const enabledClusters = useMemo(() => list.filter((c) => c.status === 1), [list]);

  useEffect(() => {
    let cancelled = false;
    async function loadVersions() {
      const ids = enabledClusters.map((c) => c.id);
      if (ids.length === 0) return;

      // only fetch versions we don't have yet
      const missing = ids.filter((id) => !serverVersionByID[id]);
      if (missing.length === 0) return;

      const results = await Promise.allSettled(
        missing.map(async (id) => {
          const st = await getClusterStatus(id);
          return { id, version: st.server_version || "-" };
        }),
      );

      if (cancelled) return;
      setServerVersionByID((prev) => {
        const next = { ...prev };
        for (const r of results) {
          if (r.status === "fulfilled") {
            next[r.value.id] = r.value.version;
          } else {
            // keep existing value if any; otherwise fallback
            // eslint-disable-next-line @typescript-eslint/no-unused-expressions
            void 0;
          }
        }
        return next;
      });
    }
    void loadVersions();
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [enabledClusters]);

  function openCreate() {
    setCurrent(null);
    form.resetFields();
    setModalOpen(true);
  }

  function openEdit(record: ClusterItem) {
    setCurrent(record);
    form.setFieldsValue({
      name: record.name,
      // kubeconfig intentionally not prefilled (sensitive, not returned by backend)
      kubeconfig: "",
    });
    setModalOpen(true);
  }

  async function handleSubmit() {
    const values = await form.validateFields();
    const name = (values.name || "").trim();
    if (!name) return;

    const payload: ClusterCreatePayload & ClusterUpdatePayload = { name, kubeconfig: "" };
    const kubeconfig = (values.kubeconfig || "").trim();
    const isCreate = !current;

    if (isCreate && !kubeconfig) {
      message.error("请填写 kubeconfig（创建集群必填）");
      return;
    }

    if (kubeconfig) {
      payload.kubeconfig = kubeconfig;
    } else {
      // Update without kubeconfig
      delete (payload as any).kubeconfig;
    }

    setSubmitting(true);
    try {
      if (current) {
        await updateCluster(current.id, payload as ClusterUpdatePayload);
        message.success("集群已更新");
      } else {
        await createCluster(payload as ClusterCreatePayload);
        message.success("集群已创建");
      }
      setModalOpen(false);
      await loadClusters();
    } finally {
      setSubmitting(false);
    }
  }

  async function handleDelete(record: ClusterItem) {
    await deleteCluster(record.id);
    message.success("集群已删除");
    void loadClusters();
  }

  async function handleToggleStatus(record: ClusterItem) {
    const nextStatus: 0 | 1 = record.status === 1 ? 0 : 1;
    setStatusUpdatingID(record.id);
    try {
      await setClusterStatus(record.id, nextStatus);
      message.success(nextStatus === 1 ? "已启用集群" : "已停用集群");
      await loadClusters();
      if (nextStatus === 0) {
        setServerVersionByID((prev) => {
          if (!prev[record.id]) return prev;
          const next = { ...prev };
          delete next[record.id];
          return next;
        });
      }
    } finally {
      setStatusUpdatingID(null);
    }
  }

  async function handleConnectTest(record: ClusterItem) {
    try {
      const st = await getClusterStatus(record.id);
      const versionText = st.server_version || "-";
      setServerVersionByID((prev) => ({ ...prev, [record.id]: versionText }));
      if (st.server_version) {
        message.success(`连接成功，K8s 版本：${st.server_version}`);
      } else {
        message.warning("连接已建立，但未获取到 K8s 版本信息");
      }
    } catch (error) {
      const msg = error instanceof Error ? error.message : "连接测试失败";
      message.error(msg || "连接测试失败");
    }
  }

  return (
    <Card className="table-card">
      <Space direction="vertical" size={12} style={{ width: "100%" }}>
        <div className="toolbar">
          <Input.Search
            allowClear
            placeholder="搜索集群名称"
            style={{ width: 280 }}
            onSearch={(keyword) => setQuery((prev) => ({ ...prev, keyword, page: 1 }))}
          />
          <div className="toolbar__actions">
            <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
              新建集群
            </Button>
            <Button icon={<ReloadOutlined />} onClick={() => void loadClusters()}>
              刷新
            </Button>
          </div>
        </div>

        <Table
          rowKey="id"
          loading={loading}
          dataSource={list}
          size="small"
          pagination={{
            current: query.page,
            pageSize: query.page_size,
            total,
            showSizeChanger: true,
            onChange: (page, pageSize) => setQuery((prev) => ({ ...prev, page, page_size: pageSize })),
          }}
          scroll={{ x: "max-content" }}
          columns={[
            { title: "ID", dataIndex: "id", width: 90 },
            { title: "集群名称", dataIndex: "name" },
            {
              title: "K8s 版本",
              key: "k8s_version",
              width: 140,
              render: (_: unknown, record: ClusterItem) => {
                if (record.status !== 1) return <span className="inline-muted">-</span>;
                const v = serverVersionByID[record.id];
                return v ? v : <span className="inline-muted">获取中…</span>;
              },
            },
            {
              title: "状态",
              dataIndex: "status",
              width: 110,
              render: (v: number) => (v === 1 ? <Tag color="success">启用</Tag> : <Tag>停用</Tag>),
            },
            { title: "创建时间", dataIndex: "created_at", render: (v: string) => formatDateTime(v) },
            {
              title: "操作",
              key: "action",
              width: 340,
              render: (_: unknown, record: ClusterItem) => (
                <Space>
                  <Button
                    type="link"
                    icon={<SettingOutlined />}
                    onClick={() => {
                      void handleConnectTest(record);
                    }}
                  >
                    连接测试
                  </Button>

                  <Popconfirm
                    title={record.status === 1 ? "确认停用该集群吗？停用后将禁止访问该集群。" : "确认启用该集群吗？"}
                    onConfirm={() => {
                      void handleToggleStatus(record);
                    }}
                  >
                    <Button
                      type="link"
                      danger={record.status === 1}
                      loading={statusUpdatingID === record.id}
                      icon={record.status === 1 ? <CloseCircleOutlined /> : <CheckCircleOutlined />}
                    >
                      {record.status === 1 ? "停用" : "启用"}
                    </Button>
                  </Popconfirm>

                  <Button type="link" icon={<EditOutlined />} onClick={() => openEdit(record)}>
                    编辑
                  </Button>
                  <Popconfirm title="确认删除该集群吗？" onConfirm={() => void handleDelete(record)}>
                    <Button type="link" danger icon={<DeleteOutlined />}>
                      删除
                    </Button>
                  </Popconfirm>
                </Space>
              ),
            },
          ]}
        />
      </Space>
      <Modal
        title={current ? `编辑集群 #${current.id}` : "新建集群"}
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={() => void handleSubmit()}
        confirmLoading={submitting}
        destroyOnClose
        width={720}
      >
        <Form form={form} layout="vertical" initialValues={{ name: "", kubeconfig: "" }}>
          <Form.Item label="集群名称" name="name" rules={[{ required: true, message: "请输入集群名称" }]}>
            <Input placeholder="例如：prod-k8s" />
          </Form.Item>

          <Form.Item
            label="kubeconfig"
            name="kubeconfig"
            rules={
              current
                ? []
                : [{ required: true, message: "请填写 kubeconfig" }]
            }
            extra={current ? "留空则不更新 kubeconfig（不会预填旧值）" : "用于 Kom SDK 注册并访问集群"}
          >
            <Input.TextArea
              rows={8}
              placeholder="粘贴 kubeconfig 内容（yaml）"
              style={{ fontFamily: "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, Liberation Mono, Courier New, monospace" }}
            />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}

// keep exported icons reference used elsewhere

