import { DeleteOutlined, EditOutlined, PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Form, Input, Modal, Popconfirm, Select, Space, Table, Tag, message } from "antd";
import { useEffect, useMemo, useState } from "react";
import { getProjects, getProjectServers, getProjectServices, upsertProjectService, deleteProjectService, type ProjectItem, type ServerItem, type ServiceItem } from "../services/projects";

export function ProjectServicesPage() {
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [servers, setServers] = useState<ServerItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [serverId, setServerId] = useState<number>();
  const [loading, setLoading] = useState(false);
  const [list, setList] = useState<ServiceItem[]>([]);
  const [editorOpen, setEditorOpen] = useState(false);
  const [current, setCurrent] = useState<ServiceItem | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [form] = Form.useForm<{ id?: number; server_id: number; name: string; env?: string; labels?: string; remark?: string; status: number }>();

  const projectOptions = useMemo(() => projects.map((p) => ({ value: p.id, label: `${p.name} (${p.code})` })), [projects]);
  const serverOptions = useMemo(() => servers.map((s) => ({ value: s.id, label: `${s.name} ${s.host}:${s.port}` })), [servers]);

  useEffect(() => {
    void (async () => {
      const p = await getProjects({ page: 1, page_size: 1000 });
      setProjects(p.list);
      if (p.list[0]) setProjectId(p.list[0].id);
    })();
  }, []);

  useEffect(() => {
    if (!projectId) return;
    void (async () => {
      const sv = await getProjectServers(projectId, { page: 1, page_size: 1000 });
      setServers(sv.list);
      if (!serverId && sv.list[0]) setServerId(sv.list[0].id);
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId]);

  useEffect(() => {
    if (!projectId) return;
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId, serverId]);

  async function load() {
    if (!projectId) return;
    setLoading(true);
    try {
      const res = await getProjectServices(projectId, { page: 1, page_size: 1000, server_id: serverId });
      setList(res.list);
    } finally {
      setLoading(false);
    }
  }

  function openCreate() {
    if (!serverId) {
      message.warning("请先选择服务器");
      return;
    }
    setCurrent(null);
    form.resetFields();
    form.setFieldsValue({ server_id: serverId, status: 1 });
    setEditorOpen(true);
  }

  function openEdit(record: ServiceItem) {
    setCurrent(record);
    form.setFieldsValue({
      id: record.id,
      server_id: record.server_id,
      name: record.name,
      env: record.env ?? undefined,
      labels: record.labels ?? undefined,
      remark: record.remark ?? undefined,
      status: record.status,
    });
    setEditorOpen(true);
  }

  async function onSubmit() {
    if (!projectId) return;
    const v = await form.validateFields();
    setSubmitting(true);
    try {
      await upsertProjectService(projectId, v);
      message.success(current ? "已更新服务" : "已创建服务");
      setEditorOpen(false);
      void load();
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Card
      title="服务配置"
      extra={
        <Space>
          <Select style={{ width: 260 }} value={projectId} onChange={setProjectId} options={projectOptions} placeholder="选择项目" />
          <Select style={{ width: 260 }} value={serverId} onChange={setServerId} options={serverOptions} placeholder="选择服务器" allowClear />
          <Button icon={<ReloadOutlined />} onClick={() => void load()} loading={loading}>刷新</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>新增服务</Button>
        </Space>
      }
    >
      <Table
        rowKey="id"
        dataSource={list}
        loading={loading}
        pagination={false}
        columns={[
          { title: "服务名", dataIndex: "name" },
          { title: "环境", dataIndex: "env", width: 120, render: (v?: string | null) => v || "-" },
          { title: "标签", dataIndex: "labels", render: (v?: string | null) => v || "-" },
          { title: "备注", dataIndex: "remark", render: (v?: string | null) => v || "-" },
          { title: "状态", dataIndex: "status", width: 100, render: (v: number) => (v === 1 ? <Tag color="green">启用</Tag> : <Tag color="red">停用</Tag>) },
          {
            title: "操作",
            width: 180,
            render: (_: unknown, record: ServiceItem) => (
              <Space>
                <Button icon={<EditOutlined />} onClick={() => openEdit(record)}>编辑</Button>
                <Popconfirm title="确认删除服务？" onConfirm={() => projectId && deleteProjectService(projectId, record.id).then(() => { message.success("已删除"); void load(); })}>
                  <Button danger icon={<DeleteOutlined />}>删除</Button>
                </Popconfirm>
              </Space>
            ),
          },
        ]}
      />
      <Modal open={editorOpen} title={current ? "编辑服务" : "新增服务"} onCancel={() => setEditorOpen(false)} onOk={() => void onSubmit()} confirmLoading={submitting}>
        <Form form={form} layout="vertical">
          <Form.Item name="id" hidden><Input /></Form.Item>
          <Form.Item label="服务器" name="server_id" rules={[{ required: true }]}><Select options={serverOptions} /></Form.Item>
          <Form.Item label="服务名" name="name" rules={[{ required: true }]}><Input /></Form.Item>
          <Form.Item label="环境" name="env"><Input /></Form.Item>
          <Form.Item label="标签" name="labels"><Input /></Form.Item>
          <Form.Item label="备注" name="remark"><Input.TextArea rows={3} /></Form.Item>
          <Form.Item label="状态" name="status" rules={[{ required: true }]}><Select options={[{ value: 1, label: "启用" }, { value: 0, label: "停用" }]} /></Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}

