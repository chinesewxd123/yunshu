import { PlusOutlined, ReloadOutlined, SafetyCertificateOutlined, EyeOutlined } from "@ant-design/icons";
import { Button, Card, Descriptions, Form, Input, Modal, Popconfirm, Select, Space, Table, Tag, Typography, message } from "antd";
import { useEffect, useState } from "react";
import { PageHero } from "../components/page-hero";
import { StatusTag } from "../components/status-tag";
import { createPermission, deletePermission, getPermissions, getPermission, updatePermission } from "../services/permissions";
import { getRoleOptions } from "../services/roles";
import { grantPolicy } from "../services/policies";
import type { PermissionItem, PermissionPayload, RoleItem } from "../types/api";
import { formatDateTime } from "../utils/format";

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
  const [assignOpen, setAssignOpen] = useState(false);
  const [assignTarget, setAssignTarget] = useState<PermissionItem | null>(null);
  const [roles, setRoles] = useState<RoleItem[]>([]);
  const [checkedRoleIds, setCheckedRoleIds] = useState<number[]>([]);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailRecord, setDetailRecord] = useState<PermissionItem | null>(null);

  useEffect(() => {
    void loadPermissions(query);
  }, [query]);

  useEffect(() => {
    void loadRoles();
  }, []);

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

  async function loadRoles() {
    const result = await getRoleOptions();
    setRoles(result.list);
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
        message.success("接口能力已更新");
      } else {
        await createPermission(values);
        message.success("接口能力创建成功");
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
    message.success(`已删除能力项 ${record.name}`);
    void loadPermissions();
  }

  async function openDetail(record: PermissionItem) {
    const detail = await getPermission(record.id);
    setDetailRecord(detail);
    setDetailOpen(true);
  }

  function openAssignRoles(record: PermissionItem) {
    setAssignTarget(record);
    setCheckedRoleIds([]);
    setAssignOpen(true);
  }

  async function submitAssignRoles() {
    if (!assignTarget) return;
    setSubmitting(true);
    try {
      const promises = checkedRoleIds.map((roleId) =>
        grantPolicy({ role_id: roleId, permission_id: assignTarget.id })
      );
      await Promise.all(promises);
      message.success("角色权限已更新");
      setAssignOpen(false);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div>
      <PageHero
        title="接口能力"
        subtitle="把 CMDB 后端可调用能力整理成资源路径与动作矩阵，为授权编排提供统一能力目录。"
        breadcrumbItems={[{ title: "控制台" }, { title: "接口能力" }]}
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
            新建能力项
          </Button>
        }
      />

      <Card className="table-card">
        <div className="toolbar">
          <Input.Search
            allowClear
            placeholder="搜索能力名称或资源路径"
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
            { title: "能力名称", dataIndex: "name" },
            { title: "资源路径", dataIndex: "resource", render: (value: string) => <Tag>{value}</Tag> },
            { title: "动作", dataIndex: "action", render: (value: string) => <Tag color="processing">{value}</Tag> },
            { title: "说明", dataIndex: "description", render: (value?: string) => value || "-" },
            {
              title: "操作",
              key: "action",
              render: (_: unknown, record: PermissionItem) => (
                <Space>
                  <Button type="link" icon={<EyeOutlined />} onClick={() => openDetail(record)}>
                    详情
                  </Button>
                  <Button type="link" onClick={() => openEdit(record)}>
                    编辑
                  </Button>
                  <Button type="link" icon={<SafetyCertificateOutlined />} onClick={() => openAssignRoles(record)}>
                    分配角色
                  </Button>
                  <Popconfirm title="确认删除该能力项吗？" onConfirm={() => void handleDelete(record)}>
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
        title={current ? `编辑能力项 #${current.id}` : "新建接口能力"}
        open={open}
        onCancel={() => setOpen(false)}
        onOk={() => void handleSubmit()}
        confirmLoading={submitting}
        destroyOnClose
      >
        <Form form={form} layout="vertical" initialValues={{ action: "GET" }}>
          <Form.Item label="能力名称" name="name" rules={[{ required: true, message: "请输入能力名称" }]}>
            <Input placeholder="例如：查询主机列表" />
          </Form.Item>
          <Form.Item label="资源路径" name="resource" rules={[{ required: true, message: "请输入资源路径" }]}>
            <Input placeholder="例如：/api/v1/users" />
          </Form.Item>
          <Form.Item label="HTTP 动作" name="action" rules={[{ required: true, message: "请选择动作" }]}>
            <Select options={actionOptions.map((item) => ({ label: item, value: item }))} />
          </Form.Item>
          <Form.Item label="说明" name="description">
            <Input.TextArea rows={3} placeholder="请输入能力说明" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={assignTarget ? `为权限 ${assignTarget.name} 分配角色` : "分配角色"}
        open={assignOpen}
        onCancel={() => {
          setAssignOpen(false);
          setCheckedRoleIds([]);
        }}
        onOk={() => void submitAssignRoles()}
        confirmLoading={submitting}
        destroyOnClose
        width={600}
      >
        <Space direction="vertical" size={12} style={{ width: "100%" }}>
          <Typography.Text className="inline-muted">
            勾选需要分配该权限的角色，已选 {checkedRoleIds.length} 个角色。
          </Typography.Text>
          <Table
            rowKey="id"
            dataSource={roles}
            pagination={{ pageSize: 10 }}
            rowSelection={{
              selectedRowKeys: checkedRoleIds,
              onChange: (keys) => setCheckedRoleIds(keys as number[]),
            }}
            columns={[
              { title: "角色名称", dataIndex: "name" },
              { title: "角色编码", dataIndex: "code", render: (code) => <Tag color="blue">{code}</Tag> },
              { title: "状态", dataIndex: "status", render: (status) => <StatusTag status={status} /> },
            ]}
            size="small"
          />
        </Space>
      </Modal>

      <Modal
        title="权限详情"
        open={detailOpen}
        onCancel={() => setDetailOpen(false)}
        footer={null}
        width={650}
      >
        {detailRecord && (
          <Descriptions bordered column={2} size="middle">
            <Descriptions.Item label="ID">{detailRecord.id}</Descriptions.Item>
            <Descriptions.Item label="能力名称">{detailRecord.name}</Descriptions.Item>
            <Descriptions.Item label="资源路径" span={2}>
              <Tag>{detailRecord.resource}</Tag>
            </Descriptions.Item>
            <Descriptions.Item label="HTTP 动作">
              <Tag color="processing">{detailRecord.action}</Tag>
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