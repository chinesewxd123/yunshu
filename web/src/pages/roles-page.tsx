import { PlusOutlined, ReloadOutlined, UserOutlined, EyeOutlined, EditOutlined, DeleteOutlined } from "@ant-design/icons";
import { Button, Card, Descriptions, Form, Input, Modal, Popconfirm, Select, Space, Table, Tag, Tree, Typography, message } from "antd";
import { useEffect, useState } from "react";
import { StatusTag } from "../components/status-tag";
import { createRole, deleteRole, getRoles, getRole, updateRole } from "../services/roles";
import { getUsers } from "../services/users";
import { assignUserRoles } from "../services/users";
import type { RoleItem, RolePayload, UserItem } from "../types/api";
import { formatDateTime } from "../utils/format";

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
  const [assignOpen, setAssignOpen] = useState(false);
  const [assignTarget, setAssignTarget] = useState<RoleItem | null>(null);
  const [users, setUsers] = useState<UserItem[]>([]);
  const [checkedUserIds, setCheckedUserIds] = useState<number[]>([]);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailRecord, setDetailRecord] = useState<RoleItem | null>(null);

  useEffect(() => {
    void loadRoles(query);
  }, [query]);

  useEffect(() => {
    void loadUsers();
  }, []);

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
        message.success("角色模板已更新");
      } else {
        await createRole(values);
        message.success("角色模板创建成功");
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
    message.success(`已删除模板 ${record.name}`);
    void loadRoles();
  }

  async function loadUsers() {
    const result = await getUsers({ page: 1, page_size: 1000 });
    setUsers(result.list);
  }

  function openAssignUsers(record: RoleItem) {
    setAssignTarget(record);
    const userIds = users
      .filter((user) => user.roles.some((role) => role.id === record.id))
      .map((user) => user.id);
    setCheckedUserIds(userIds);
    setAssignOpen(true);
  }

  async function submitAssignUsers() {
    if (!assignTarget) return;
    setSubmitting(true);
    try {
      const promises = checkedUserIds.map((userId) => {
        const user = users.find((u) => u.id === userId);
        if (user) {
          const roleIds = user.roles.map((r) => r.id);
          if (!roleIds.includes(assignTarget.id)) {
            return assignUserRoles(userId, { role_ids: [...roleIds, assignTarget.id] });
          }
        }
        return Promise.resolve();
      });
      await Promise.all(promises);
      message.success("用户角色已更新");
      setAssignOpen(false);
      void loadUsers();
    } finally {
      setSubmitting(false);
    }
  }

  async function openDetail(record: RoleItem) {
    const detail = await getRole(record.id);
    setDetailRecord(detail);
    setDetailOpen(true);
  }

  return (
    <div>
      <Card className="table-card">
        <div className="toolbar">
          <Input.Search
            allowClear
            placeholder="搜索模板名称或编码"
            style={{ width: 280 }}
            onSearch={(keyword) => setQuery((prev) => ({ ...prev, keyword, page: 1 }))}
          />
          <div className="toolbar__actions">
            <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
              新建模板
            </Button>
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
            { title: "模板名称", dataIndex: "name" },
            { title: "模板编码", dataIndex: "code", render: (value: string) => <Tag color="blue">{value}</Tag> },
            { title: "说明", dataIndex: "description", render: (value?: string) => value || "-" },
            { title: "状态", dataIndex: "status", render: (value: number) => <StatusTag status={value} /> },
            {
              title: "操作",
              key: "action",
              render: (_: unknown, record: RoleItem) => (
                <Space>
                  <Button type="link" icon={<EyeOutlined />} onClick={() => openDetail(record)}>
                    详情
                  </Button>
                  <Button type="link" icon={<EditOutlined />} onClick={() => openEdit(record)}>
                    编辑
                  </Button>
                  <Button type="link" icon={<UserOutlined />} onClick={() => openAssignUsers(record)}>
                    分配用户
                  </Button>
                  <Popconfirm title="确认删除该模板吗？" onConfirm={() => handleDelete(record)}>
                    <Button type="link" danger icon={<DeleteOutlined />}>
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
        title={current ? `编辑模板 #${current.id}` : "新建角色模板"}
        open={open}
        onCancel={() => setOpen(false)}
        onOk={() => void handleSubmit()}
        confirmLoading={submitting}
        destroyOnClose
      >
        <Form form={form} layout="vertical" initialValues={{ status: 1 }}>
          <Form.Item label="模板名称" name="name" rules={[{ required: true, message: "请输入模板名称" }]}>
            <Input placeholder="例如：CMDB 运维值班" />
          </Form.Item>
          <Form.Item label="模板编码" name="code" rules={[{ required: true, message: "请输入模板编码" }]}>
            <Input placeholder="例如：cmdb-operator" />
          </Form.Item>
          <Form.Item label="说明" name="description">
            <Input.TextArea rows={3} placeholder="请输入模板说明" />
          </Form.Item>
          <Form.Item label="状态" name="status" rules={[{ required: true, message: "请选择状态" }]}>
            <Select options={[{ label: "启用", value: 1 }, { label: "停用", value: 0 }]} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={assignTarget ? `为角色 ${assignTarget.name} 分配用户` : "分配用户"}
        open={assignOpen}
        onCancel={() => {
          setAssignOpen(false);
          setCheckedUserIds([]);
        }}
        onOk={() => void submitAssignUsers()}
        confirmLoading={submitting}
        destroyOnClose
        width={600}
      >
        <Space direction="vertical" size={12} style={{ width: "100%" }}>
          <Typography.Text className="inline-muted">
            勾选需要分配该角色的用户，已选 {checkedUserIds.length} 个用户。
          </Typography.Text>
          <Table
            rowKey="id"
            dataSource={users}
            pagination={{ pageSize: 10 }}
            rowSelection={{
              selectedRowKeys: checkedUserIds,
              onChange: (keys) => setCheckedUserIds(keys as number[]),
            }}
            columns={[
              { title: "用户名", dataIndex: "username" },
              { title: "昵称", dataIndex: "nickname" },
              { title: "状态", dataIndex: "status", render: (status) => <StatusTag status={status} /> },
            ]}
            size="small"
          />
        </Space>
      </Modal>

      <Modal
        title="角色详情"
        open={detailOpen}
        onCancel={() => setDetailOpen(false)}
        footer={null}
        width={600}
      >
        {detailRecord && (
          <Descriptions bordered column={2} size="middle">
            <Descriptions.Item label="ID">{detailRecord.id}</Descriptions.Item>
            <Descriptions.Item label="模板名称">{detailRecord.name}</Descriptions.Item>
            <Descriptions.Item label="模板编码">
              <Tag color="blue">{detailRecord.code}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="状态" span={2}>
              <StatusTag status={detailRecord.status} />
            </Descriptions.Item>
            <Descriptions.Item label="说明" span={2}>{detailRecord.description || "-"}</Descriptions.Item>
            <Descriptions.Item label="创建时间">{formatDateTime(detailRecord.created_at)}</Descriptions.Item>

            <Descriptions.Item label="更新时间">{formatDateTime(detailRecord.updated_at)}</Descriptions.Item>
          </Descriptions>
        )}
      </Modal>
    </div>
  );
}