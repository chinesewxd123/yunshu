import { DeleteOutlined, EditOutlined, PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Form, Input, Modal, Popconfirm, Select, Space, Table, Tag, message } from "antd";
import { useEffect, useState } from "react";
import { createProject, deleteProject, getProjects, updateProject, type ProjectCreatePayload, type ProjectItem, type ProjectUpdatePayload } from "../services/projects";
import { formatDateTime } from "../utils/format";

const defaultQuery = { keyword: "", page: 1, page_size: 10 };

export function ProjectsPage() {
  const [query, setQuery] = useState(defaultQuery);
  const [loading, setLoading] = useState(false);
  const [list, setList] = useState<ProjectItem[]>([]);
  const [total, setTotal] = useState(0);

  const [editorOpen, setEditorOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [current, setCurrent] = useState<ProjectItem | null>(null);
  const [form] = Form.useForm<ProjectCreatePayload & ProjectUpdatePayload>();

  useEffect(() => {
    void load();
  }, [query]);

  async function load() {
    setLoading(true);
    try {
      const data = await getProjects(query);
      setList(data.list);
      setTotal(data.total);
    } finally {
      setLoading(false);
    }
  }

  function openCreate() {
    setCurrent(null);
    form.resetFields();
    form.setFieldsValue({ status: 1 });
    setEditorOpen(true);
  }

  function openEdit(record: ProjectItem) {
    setCurrent(record);
    form.resetFields();
    form.setFieldsValue({ name: record.name, code: record.code, description: record.description ?? undefined, status: record.status });
    setEditorOpen(true);
  }

  async function onSubmit() {
    const values = await form.validateFields();
    setSubmitting(true);
    try {
      if (!current) {
        await createProject(values);
        message.success("已创建项目");
      } else {
        await updateProject(current.id, values);
        message.success("已更新项目");
      }
      setEditorOpen(false);
      void load();
    } finally {
      setSubmitting(false);
    }
  }

  async function onDelete(record: ProjectItem) {
    await deleteProject(record.id);
    message.success("已删除");
    void load();
  }

  return (
    <Card
      title="项目列表"
      extra={
        <Space>
          <Input.Search
            placeholder="搜索 name/code"
            allowClear
            onSearch={(keyword) => setQuery((q) => ({ ...q, keyword, page: 1 }))}
            style={{ width: 240 }}
          />
          <Button icon={<ReloadOutlined />} onClick={() => void load()} loading={loading}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
            新增
          </Button>
        </Space>
      }
    >
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
          { title: "ID", dataIndex: "id", width: 80 },
          { title: "名称", dataIndex: "name" },
          { title: "编码", dataIndex: "code", width: 180 },
          {
            title: "状态",
            dataIndex: "status",
            width: 120,
            render: (v: number) => (v === 1 ? <Tag color="green">启用</Tag> : <Tag color="red">停用</Tag>),
          },
          { title: "创建时间", dataIndex: "created_at", width: 200, render: (v: string) => formatDateTime(v) },
          {
            title: "操作",
            width: 180,
            render: (_: unknown, record: ProjectItem) => (
              <Space>
                <Button icon={<EditOutlined />} onClick={() => openEdit(record)}>
                  编辑
                </Button>
                <Popconfirm title="确定删除该项目？" onConfirm={() => void onDelete(record)}>
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
        title={current ? "编辑项目" : "新增项目"}
        open={editorOpen}
        onCancel={() => setEditorOpen(false)}
        onOk={() => void onSubmit()}
        confirmLoading={submitting}
        destroyOnClose
      >
        <Form layout="vertical" form={form}>
          <Form.Item label="名称" name="name" rules={[{ required: true, message: "请输入名称" }]}>
            <Input placeholder="例如：生产环境项目" />
          </Form.Item>
          <Form.Item label="编码" name="code" rules={[{ required: true, message: "请输入编码" }]}>
            <Input placeholder="例如：prod" />
          </Form.Item>
          <Form.Item label="描述" name="description">
            <Input.TextArea rows={3} placeholder="可选" />
          </Form.Item>
          <Form.Item label="状态" name="status" rules={[{ required: true }]}>
            <Select
              options={[
                { value: 1, label: "启用" },
                { value: 0, label: "停用" },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}

