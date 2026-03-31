import { PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Form, Input, Modal, Popconfirm, Select, Space, Table, Tag, message } from "antd";
import { useEffect, useState } from "react";
import { PageHero } from "../components/page-hero";
import { StatusTag } from "../components/status-tag";
import { createRole, deleteRole, getRoles, updateRole } from "../services/roles";
import type { RoleItem, RolePayload } from "../types/api";

const defaultQuery = { keyword: "", page: 1, page_size: 10 };

export function RolesPage() {
  const [list, setList] = useState<RoleItem[]>([]);
  const [total, setTotal] = useState(0);
  const [query, setQuery] = useState(defaultQuery);
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [current, setCurrent] = useState<RoleItem | null>(null);
  const [open, setOpen] = useState(false);
  const [form] = Form.useForm<RolePayload>();

  useEffect(() => {
    void loadRoles(query);
  }, [query]);

  async function loadRoles(nextQuery = query) {
    setLoading(true);
    try {
      const result = await getRoles(nextQuery);
      setList(result.list);
      setTotal(result.total);
    } finally {
      setLoading(false);
    }
  }

  function openCreate() {
    setCurrent(null);
    form.resetFields();
    form.setFieldsValue({ status: 1 });
    setOpen(true);
  }

  function openEdit(record: RoleItem) {
    setCurrent(record);
    form.setFieldsValue(record);
    setOpen(true);
  }

  async function handleSubmit() {
    const values = await form.validateFields();
    setSubmitting(true);
    try {
      if (current) {
        await updateRole(current.id, values);
        message.success("角色已更新");
      } else {
        await createRole(values);
        message.success("角色创建成功");
      }
      setOpen(false);
      form.resetFields();
      void loadRoles();
    } finally {
      setSubmitting(false);
    }
  }

  async function handleDelete(record: RoleItem) {
    await deleteRole(record.id);
    message.success(`已删除角色 ${record.name}`);
    void loadRoles();
  }

  return (
    <div>
      <PageHero
        title="角色管理"
        subtitle="角色是 Casbin 授权链路中的核心主体，这里可以维护角色基础信息和启停状态。"
        breadcrumbItems={[{ title: "控制台" }, { title: "角色管理" }]}
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
            新建角色
          </Button>
        }
      />

      <Card className="table-card">
        <div className="toolbar">
          <Input.Search
            allowClear
            placeholder="搜索角色名称或编码"
            style={{ width: 280 }}
            onSearch={(keyword) => setQuery((prev) => ({ ...prev, keyword, page: 1 }))}
          />
          <div className="toolbar__actions">
            <Button icon={<ReloadOutlined />} onClick={() => void loadRoles()}>
              刷新
            </Button>
          </div>
        </div>
        <Table
          rowKey="id"
          loading={loading}
          dataSource={list}
          pagination={{
            current: query.page,
            pageSize: query.page_size,
            total,
            showSizeChanger: true,
            onChange: (page, pageSize) => setQuery((prev) => ({ ...prev, page, page_size: pageSize })),
          }}
          columns={[
            { title: "ID", dataIndex: "id", width: 70 },
            { title: "角色名称", dataIndex: "name" },
            { title: "编码", dataIndex: "code", render: (value: string) => <Tag color="blue">{value}</Tag> },
            { title: "描述", dataIndex: "description", render: (value?: string) => value || "-" },
            { title: "状态", dataIndex: "status", render: (value: number) => <StatusTag status={value} /> },
            {
              title: "操作",
              key: "action",
              render: (_: unknown, record: RoleItem) => (
                <Space>
                  <Button type="link" onClick={() => openEdit(record)}>
                    编辑
                  </Button>
                  <Popconfirm title="确认删除该角色吗？" onConfirm={() => handleDelete(record)}>
                    <Button type="link" danger>
                      删除
                    </Button>
                  </Popconfirm>
                </Space>
              ),
            },
          ]}
        />
      </Card>

      <Modal
        title={current ? `编辑角色 #${current.id}` : "新建角色"}
        open={open}
        onCancel={() => setOpen(false)}
        onOk={() => void handleSubmit()}
        confirmLoading={submitting}
        destroyOnClose
      >
        <Form form={form} layout="vertical" initialValues={{ status: 1 }}>
          <Form.Item label="角色名称" name="name" rules={[{ required: true, message: "请输入角色名称" }]}>
            <Input placeholder="例如：运营经理" />
          </Form.Item>
          <Form.Item label="角色编码" name="code" rules={[{ required: true, message: "请输入角色编码" }]}>
            <Input placeholder="例如：operator-manager" />
          </Form.Item>
          <Form.Item label="描述" name="description">
            <Input.TextArea rows={3} placeholder="请输入角色描述" />
          </Form.Item>
          <Form.Item label="状态" name="status" rules={[{ required: true, message: "请选择状态" }]}>
            <Select options={[{ label: "启用", value: 1 }, { label: "停用", value: 0 }]} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
