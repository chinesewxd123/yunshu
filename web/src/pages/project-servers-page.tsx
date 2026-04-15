import { DeleteOutlined, EditOutlined, PlusOutlined, ReloadOutlined, UploadOutlined, DownloadOutlined, ApiOutlined } from "@ant-design/icons";
import { Button, Card, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Table, Tag, message } from "antd";
import { useEffect, useMemo, useRef, useState } from "react";
import {
  exportProjectServers,
  getProjects,
  getProjectServers,
  importProjectServers,
  testProjectServer,
  upsertProjectServer,
  deleteProjectServer,
  type ProjectItem,
  type ServerItem,
  type ServerUpsertPayload,
} from "../services/projects";
import { formatDateTime } from "../utils/format";

const defaultQuery = { keyword: "", page: 1, page_size: 10 };

export function ProjectServersPage() {
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [query, setQuery] = useState(defaultQuery);
  const [loading, setLoading] = useState(false);
  const [list, setList] = useState<ServerItem[]>([]);
  const [total, setTotal] = useState(0);

  const [editorOpen, setEditorOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [current, setCurrent] = useState<ServerItem | null>(null);
  const [form] = Form.useForm<ServerUpsertPayload>();
  const fileInputRef = useRef<HTMLInputElement | null>(null);

  const projectOptions = useMemo(() => projects.map((p) => ({ value: p.id, label: `${p.name} (${p.code})` })), [projects]);

  useEffect(() => {
    void (async () => {
      const data = await getProjects({ page: 1, page_size: 1000 });
      setProjects(data.list);
      if (!projectId && data.list[0]) {
        setProjectId(data.list[0].id);
      }
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (!projectId) return;
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId, query]);

  async function load() {
    if (!projectId) return;
    setLoading(true);
    try {
      const data = await getProjectServers(projectId, query);
      setList(data.list);
      setTotal(data.total);
    } finally {
      setLoading(false);
    }
  }

  function openCreate() {
    if (!projectId) return;
    setCurrent(null);
    form.resetFields();
    form.setFieldsValue({ project_id: projectId, port: 22, os_type: "linux", status: 1, auth_type: "password" });
    setEditorOpen(true);
  }

  function openEdit(record: ServerItem) {
    setCurrent(record);
    form.resetFields();
    form.setFieldsValue({
      id: record.id,
      project_id: record.project_id,
      name: record.name,
      host: record.host,
      port: record.port,
      os_type: record.os_type,
      tags: record.tags,
      status: record.status,
    });
    setEditorOpen(true);
  }

  async function onSubmit() {
    if (!projectId) return;
    const values = await form.validateFields();
    setSubmitting(true);
    try {
      await upsertProjectServer(projectId, values);
      message.success(current ? "已更新服务器" : "已新增服务器");
      setEditorOpen(false);
      void load();
    } finally {
      setSubmitting(false);
    }
  }

  async function onDelete(record: ServerItem) {
    if (!projectId) return;
    await deleteProjectServer(projectId, record.id);
    message.success("已删除");
    void load();
  }

  async function onTest(record: ServerItem) {
    if (!projectId) return;
    const res = await testProjectServer(projectId, record.id);
    if (res.ok) message.success(res.message || "连通性 OK");
    else message.error(res.message || "连通性失败");
    void load();
  }

  async function onImport(file: File) {
    if (!projectId) return;
    const res = await importProjectServers(projectId, file);
    message.success(`已导入 ${res.imported} 条`);
    void load();
  }

  return (
    <Card
      title="服务器管理"
      extra={
        <Space>
          <Select style={{ width: 260 }} placeholder="选择项目" value={projectId} onChange={setProjectId} options={projectOptions} />
          <Input.Search
            placeholder="搜索 name/host/tags"
            allowClear
            onSearch={(keyword) => setQuery((q) => ({ ...q, keyword, page: 1 }))}
            style={{ width: 240 }}
          />
          <Button icon={<ReloadOutlined />} onClick={() => void load()} loading={loading}>
            刷新
          </Button>
          <Button icon={<UploadOutlined />} onClick={() => fileInputRef.current?.click()} disabled={!projectId}>
            导入
          </Button>
          <Button
            icon={<DownloadOutlined />}
            disabled={!projectId}
            onClick={() => {
              if (!projectId) return;
              void (async () => {
                const blob = await exportProjectServers(projectId, { keyword: query.keyword });
                const url = window.URL.createObjectURL(blob);
                const a = document.createElement("a");
                a.href = url;
                a.download = `project-${projectId}-servers.xlsx`;
                a.click();
                window.URL.revokeObjectURL(url);
              })();
            }}
          >
            导出
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate} disabled={!projectId}>
            新增
          </Button>
        </Space>
      }
    >
      <input
        ref={fileInputRef}
        type="file"
        accept=".xlsx,.xls"
        style={{ display: "none" }}
        onChange={(e) => {
          const f = e.target.files?.[0];
          if (f) void onImport(f);
          e.target.value = "";
        }}
      />

      <Table
        rowKey="id"
        dataSource={list}
        loading={loading}
        pagination={{
          current: query.page,
          pageSize: query.page_size,
          total,
          showSizeChanger: true,
          onChange: (page, pageSize) => setQuery((q) => ({ ...q, page, page_size: pageSize })),
        }}
        columns={[
          { title: "名称", dataIndex: "name", width: 180 },
          { title: "Host", dataIndex: "host", width: 180 },
          { title: "Port", dataIndex: "port", width: 90 },
          {
            title: "OS / 架构",
            width: 180,
            render: (_: unknown, record: ServerItem) => `${record.os_type || "-"} / ${record.os_arch || "-"}`,
          },
          { title: "Tags", dataIndex: "tags" },
          {
            title: "状态",
            dataIndex: "status",
            width: 100,
            render: (v: number) => (v === 1 ? <Tag color="green">启用</Tag> : <Tag color="red">停用</Tag>),
          },
          { title: "上次测试", dataIndex: "last_test_at", width: 180, render: (v?: string | null) => (v ? formatDateTime(v) : "-") },
          {
            title: "测试错误",
            dataIndex: "last_test_error",
            width: 220,
            ellipsis: true,
            render: (v?: string | null) => (v ? <span title={v}>{v}</span> : "-"),
          },
          {
            title: "操作",
            width: 280,
            render: (_: unknown, record: ServerItem) => (
              <Space>
                <Button icon={<ApiOutlined />} onClick={() => void onTest(record)}>
                  测试
                </Button>
                <Button icon={<EditOutlined />} onClick={() => openEdit(record)}>
                  编辑
                </Button>
                <Popconfirm title="确定删除该服务器？" onConfirm={() => void onDelete(record)}>
                  <Button danger icon={<DeleteOutlined />}>
                    删除
                  </Button>
                </Popconfirm>
              </Space>
            ),
          },
        ]}
      />

      <Modal
        title={current ? "编辑服务器" : "新增服务器"}
        open={editorOpen}
        onCancel={() => setEditorOpen(false)}
        onOk={() => void onSubmit()}
        confirmLoading={submitting}
        destroyOnClose
        width={720}
      >
        <Form layout="vertical" form={form}>
          <Form.Item name="id" hidden>
            <Input />
          </Form.Item>
          <Form.Item name="project_id" hidden>
            <Input />
          </Form.Item>
          <Space style={{ width: "100%" }} size={16} align="start">
            <Form.Item label="名称" name="name" rules={[{ required: true }]} style={{ flex: 1 }}>
              <Input />
            </Form.Item>
            <Form.Item label="OS" name="os_type" style={{ width: 160 }}>
              <Select options={[{ value: "linux", label: "linux" }, { value: "windows", label: "windows" }]} />
            </Form.Item>
            <Form.Item label="状态" name="status" rules={[{ required: true }]} style={{ width: 160 }}>
              <Select options={[{ value: 1, label: "启用" }, { value: 0, label: "停用" }]} />
            </Form.Item>
          </Space>
          <Space style={{ width: "100%" }} size={16} align="start">
            <Form.Item label="Host" name="host" rules={[{ required: true }]} style={{ flex: 1 }}>
              <Input placeholder="例如：10.0.0.12" />
            </Form.Item>
            <Form.Item label="Port" name="port" style={{ width: 160 }}>
              <InputNumber min={1} max={65535} style={{ width: "100%" }} />
            </Form.Item>
          </Space>
          <Form.Item label="Tags（逗号分隔）" name="tags">
            <Input />
          </Form.Item>

          <Card size="small" title="SSH 凭据（可选：不填则不更新凭据）">
            <Space style={{ width: "100%" }} size={16} align="start">
              <Form.Item label="认证方式" name="auth_type" style={{ width: 180 }}>
                <Select options={[{ value: "password", label: "密码" }, { value: "key", label: "私钥" }]} />
              </Form.Item>
              <Form.Item label="用户名" name="username" style={{ flex: 1 }}>
                <Input placeholder="例如：root" />
              </Form.Item>
            </Space>
            <Form.Item noStyle shouldUpdate={(a, b) => a.auth_type !== b.auth_type}>
              {({ getFieldValue }) => {
                const authType = getFieldValue("auth_type");
                if (authType === "key") {
                  return (
                    <>
                      <Form.Item label="私钥（PEM）" name="private_key">
                        <Input.TextArea rows={6} placeholder="-----BEGIN OPENSSH PRIVATE KEY-----" />
                      </Form.Item>
                      <Form.Item label="私钥口令（可选）" name="passphrase">
                        <Input.Password />
                      </Form.Item>
                    </>
                  );
                }
                return (
                  <Form.Item label="密码" name="password">
                    <Input.Password />
                  </Form.Item>
                );
              }}
            </Form.Item>
          </Card>
        </Form>
      </Modal>
    </Card>
  );
}

