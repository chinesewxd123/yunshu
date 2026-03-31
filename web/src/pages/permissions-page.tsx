import { PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Form, Input, Modal, Popconfirm, Select, Space, Table, Tag, message } from "antd";
import { useEffect, useState } from "react";
import { PageHero } from "../components/page-hero";
import { createPermission, deletePermission, getPermissions, updatePermission } from "../services/permissions";
import type { PermissionItem, PermissionPayload } from "../types/api";

const defaultQuery = { keyword: "", page: 1, page_size: 10 };
const actionOptions = ["GET", "POST", "PUT", "DELETE", "PATCH"];

export function PermissionsPage() {
  const [list, setList] = useState<PermissionItem[]>([]);
  const [total, setTotal] = useState(0);
  const [query, setQuery] = useState(defaultQuery);
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [current, setCurrent] = useState<PermissionItem | null>(null);
  const [open, setOpen] = useState(false);
  const [form] = Form.useForm<PermissionPayload>();

  useEffect(() => {
    void loadPermissions(query);
  }, [query]);

  async function loadPermissions(nextQuery = query) {
    setLoading(true);
    try {
      const result = await getPermissions(nextQuery);
      setList(result.list);
      setTotal(result.total);
    } finally {
      setLoading(false);
    }
  }

  function openCreate() {
    setCurrent(null);
    form.resetFields();
    form.setFieldsValue({ action: "GET" });
    setOpen(true);
  }

  function openEdit(record: PermissionItem) {
    setCurrent(record);
    form.setFieldsValue(record);
    setOpen(true);
  }

  async function handleSubmit() {
    const values = await form.validateFields();
    setSubmitting(true);
    try {
      if (current) {
        await updatePermission(current.id, values);
        message.success("权限已更新");
      } else {
        await createPermission(values);
        message.success("权限创建成功");
      }
      setOpen(false);
      form.resetFields();
      void loadPermissions();
    } finally {
      setSubmitting(false);
    }
  }

  async function handleDelete(record: PermissionItem) {
    await deletePermission(record.id);
    message.success(`已删除权限 ${record.name}`);
    void loadPermissions();
  }

  return (
    <div>
      <PageHero
        title="权限管理"
        subtitle="每条权限记录对应一个资源与动作组合，可直接用于 Casbin 策略绑定。"
        breadcrumbItems={[{ title: "控制台" }, { title: "权限管理" }]}
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
            新建权限
          </Button>
        }
      />

      <Card className="table-card">
        <div className="toolbar">
          <Input.Search
            allowClear
            placeholder="搜索权限名称或资源"
            style={{ width: 280 }}
            onSearch={(keyword) => setQuery((prev) => ({ ...prev, keyword, page: 1 }))}
          />
          <div className="toolbar__actions">
            <Button icon={<ReloadOutlined />} onClick={() => void loadPermissions()}>
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
            { title: "权限名称", dataIndex: "name" },
            { title: "资源", dataIndex: "resource", render: (value: string) => <Tag>{value}</Tag> },
            { title: "动作", dataIndex: "action", render: (value: string) => <Tag color="processing">{value}</Tag> },
            { title: "描述", dataIndex: "description", render: (value?: string) => value || "-" },
            {
              title: "操作",
              key: "action",
              render: (_: unknown, record: PermissionItem) => (
                <Space>
                  <Button type="link" onClick={() => openEdit(record)}>
                    编辑
                  </Button>
                  <Popconfirm title="确认删除该权限吗？" onConfirm={() => handleDelete(record)}>
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
        title={current ? `编辑权限 #${current.id}` : "新建权限"}
        open={open}
        onCancel={() => setOpen(false)}
        onOk={() => void handleSubmit()}
        confirmLoading={submitting}
        destroyOnClose
      >
        <Form form={form} layout="vertical" initialValues={{ action: "GET" }}>
          <Form.Item label="权限名称" name="name" rules={[{ required: true, message: "请输入权限名称" }]}>
            <Input placeholder="例如：查询用户列表" />
          </Form.Item>
          <Form.Item label="资源路径" name="resource" rules={[{ required: true, message: "请输入资源路径" }]}>
            <Input placeholder="例如：/api/v1/users" />
          </Form.Item>
          <Form.Item label="HTTP 动作" name="action" rules={[{ required: true, message: "请选择动作" }]}>
            <Select options={actionOptions.map((item) => ({ label: item, value: item }))} />
          </Form.Item>
          <Form.Item label="描述" name="description">
            <Input.TextArea rows={3} placeholder="请输入权限描述" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
