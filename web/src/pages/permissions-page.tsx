import { PlusOutlined, ReloadOutlined, SafetyCertificateOutlined, EyeOutlined, DeleteOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Drawer, Form, Input, Modal, Popconfirm, Select, Space, Switch, Table, Tag, Tooltip, Typography, message } from "antd";
import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import { StatusTag } from "../components/status-tag";
import { createPermission, deletePermission, getPermissions, getPermission, updatePermission } from "../services/permissions";
import { getRoleOptions } from "../services/roles";
import { grantPolicy } from "../services/policies";
import { API_CATALOG_GROUPS } from "../constants/api-catalog";
import type { PermissionItem, PermissionPayload, RoleItem } from "../types/api";
import { formatDateTime } from "../utils/format";

const defaultQuery = { keyword: "", page: 1, page_size: 10 };

const HTTP_METHOD_OPTIONS = ["GET", "POST", "PUT", "DELETE", "PATCH"].map((m) => ({ label: m, value: m }));

export function PermissionsPage() {
  const [list, setList] = useState<PermissionItem[]>([]);
  const [total, setTotal] = useState(0);
  const [query, setQuery] = useState(defaultQuery);
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [open, setOpen] = useState(false);
  const [form] = Form.useForm<PermissionPayload>();
  const [assignOpen, setAssignOpen] = useState(false);
  const [assignTarget, setAssignTarget] = useState<PermissionItem | null>(null);
  const [roles, setRoles] = useState<RoleItem[]>([]);
  const [checkedRoleIds, setCheckedRoleIds] = useState<number[]>([]);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailRecord, setDetailRecord] = useState<PermissionItem | null>(null);
  const [detailSubmitting, setDetailSubmitting] = useState(false);
  const [detailForm] = Form.useForm<PermissionPayload>();
  const [syncingCatalog, setSyncingCatalog] = useState(false);

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
    form.resetFields();
    form.setFieldsValue({ action: "GET", k8s_scope_enabled: false });
    setOpen(true);
  }

  async function handleSubmit() {
    const values = await form.validateFields();
    setSubmitting(true);
    try {
      await createPermission(values);
      message.success("接口能力创建成功");
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

  async function handleToggleK8sScope(record: PermissionItem, enabled: boolean) {
    await updatePermission(record.id, { k8s_scope_enabled: enabled });
    message.success(enabled ? "已纳入 K8s 范围校验目录" : "已取消 K8s 范围校验目录");
    setList((prev) => prev.map((item) => (item.id === record.id ? { ...item, k8s_scope_enabled: enabled } : item)));
    if (detailRecord?.id === record.id) {
      setDetailRecord((prev) => (prev ? { ...prev, k8s_scope_enabled: enabled } : prev));
    }
  }

  async function openDetail(record: PermissionItem) {
    const detail = await getPermission(record.id);
    setDetailRecord(detail);
    detailForm.setFieldsValue({
      name: detail.name,
      resource: detail.resource,
      action: detail.action,
      description: detail.description,
      k8s_scope_enabled: detail.k8s_scope_enabled,
    });
    setDetailOpen(true);
  }

  async function submitDetailEdit() {
    if (!detailRecord) return;
    const values = await detailForm.validateFields();
    setDetailSubmitting(true);
    try {
      await updatePermission(detailRecord.id, values);
      message.success("权限详情已更新");
      setDetailOpen(false);
      setDetailRecord(null);
      await loadPermissions();
    } finally {
      setDetailSubmitting(false);
    }
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

  async function handleSyncCatalog() {
    setSyncingCatalog(true);
    try {
      const existing = new Set<string>();
      let page = 1;
      const pageSize = 100;
      while (true) {
        const res = await getPermissions({ page, page_size: pageSize });
        for (const it of res.list) {
          existing.add(`${it.action.toUpperCase()} ${it.resource}`);
        }
        if (!res.list.length || page * res.page_size >= res.total) {
          break;
        }
        page++;
      }
      const missing: { name: string; resource: string; action: string; description: string }[] = [];
      for (const group of API_CATALOG_GROUPS) {
        for (const route of group.routes) {
          const action = route.method.toUpperCase();
          const resource = route.path.trim();
          const key = `${action} ${resource}`;
          if (existing.has(key)) continue;
          missing.push({
            name: route.summary,
            resource,
            action,
            description: `${group.title} · ${route.ui}`,
          });
        }
      }
      if (missing.length === 0) {
        message.info("接口能力记录已是最新，无需补全");
        return;
      }
      for (const it of missing) {
        await createPermission({
          name: it.name,
          resource: it.resource,
          action: it.action,
          description: it.description,
          k8s_scope_enabled: false,
        });
      }
      message.success(`已补全 ${missing.length} 条接口能力记录`);
      await loadPermissions();
    } finally {
      setSyncingCatalog(false);
    }
  }

  const totalCount = Number(total);

  return (
    <div className="permissions-admin-page">
      <Card className="table-card">
        <div className="toolbar">
          <Input.Search
            allowClear
            placeholder="搜索能力名称或资源路径"
            style={{ width: 280 }}
            onSearch={(keyword) => setQuery((prev) => ({ ...prev, keyword, page: 1 }))}
          />
          <div className="toolbar__actions">
            <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
              新建能力项
            </Button>
            <Button onClick={() => void handleSyncCatalog()} loading={syncingCatalog}>
              一键补全接口
            </Button>
            <Button icon={<ReloadOutlined />} onClick={() => void loadPermissions()}>
              刷新
            </Button>
          </div>
        </div>

        <Alert
          type="info"
          showIcon
          style={{ marginBottom: 16 }}
          message="接口目录与前端入口"
          description={
            <span>
              「一键补全接口」按 <Typography.Text code>constants/api-catalog.ts</Typography.Text> 中「告警中心」等分组补全缺失的权限记录（能力名称取各行的{" "}
              <Typography.Text code>summary</Typography.Text>，须与 <Typography.Text code>cmd/seed.go</Typography.Text> 中 Casbin 的{" "}
              <Typography.Text code>Name</Typography.Text> 一致）。数据源、静默、监控规则、处理人、值班、PromQL 与「策略与联调」（Webhook、策略、历史、模板）均在{" "}
              <Link to="/alert-monitor-platform">告警监控平台</Link>
              （<Link to="/alert-monitor-platform?tab=config">策略与联调</Link>）。
              <br />
              <Typography.Text type="secondary">
                列表中的「K8s 范围校验」开关用于把该接口纳入 K8s 三元中间件的<strong>校验目录</strong>（见权限设计文档 §0）；不等于给角色授权，角色授权在「授权管理」中配置。
              </Typography.Text>
            </span>
          }
        />

        <Table
          rowKey="id"
          loading={loading}
          dataSource={list}
          pagination={{
            current: query.page,
            pageSize: query.page_size,
            total: Number.isFinite(totalCount) ? totalCount : 0,
            showSizeChanger: true,
            pageSizeOptions: [10, 20, 50, 100],
            showQuickJumper: true,
            showTotal: (t, range) => `${range[0]}-${range[1]} / 共 ${t} 条`,
            onChange: (page, pageSize) => {
              setQuery((prev) => ({
                ...prev,
                page,
                page_size: pageSize ?? prev.page_size,
              }));
            },
            onShowSizeChange: (_page, size) => {
              setQuery((prev) => ({ ...prev, page: 1, page_size: size }));
            },
          }}
          columns={[
            { title: "ID", dataIndex: "id", width: 70 },
            { title: "能力名称", dataIndex: "name" },
            { title: "资源路径", dataIndex: "resource", render: (value: string) => <Tag>{value}</Tag> },
            { title: "动作", dataIndex: "action", render: (value: string) => <Tag color="processing">{value}</Tag> },
            {
              title: "K8s 范围校验",
              dataIndex: "k8s_scope_enabled",
              width: 120,
              render: (v?: boolean) => <Tag color={v ? "purple" : "default"}>{v ? "已纳入目录" : "未纳入"}</Tag>,
            },
            { title: "说明", dataIndex: "description", render: (value?: string) => value || "-" },
            {
              title: "操作",
              key: "action",
              render: (_: unknown, record: PermissionItem) => (
                <Space>
                  <Button type="link" icon={<EyeOutlined />} onClick={() => openDetail(record)}>
                    详情
                  </Button>
                  <Button type="link" icon={<SafetyCertificateOutlined />} onClick={() => openAssignRoles(record)}>
                    分配角色
                  </Button>
                  <Tooltip
                    title={
                      "打开后：该接口在请求带集群上下文时将进入 K8s 范围校验中间件。仅标记「是否参与校验」；集群侧能力由「K8s 集群访问档位」页配置，API 能否调用仍由「授权管理」决定。"
                    }
                  >
                    <Switch
                      size="small"
                      checked={Boolean(record.k8s_scope_enabled)}
                      checkedChildren="开"
                      unCheckedChildren="关"
                      onChange={(checked) => {
                        void handleToggleK8sScope(record, checked);
                      }}
                    />
                  </Tooltip>
                  <Popconfirm title="确认删除该能力项吗？" onConfirm={() => void handleDelete(record)}>
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
        title="新建接口能力"
        open={open}
        onCancel={() => setOpen(false)}
        onOk={() => void handleSubmit()}
        confirmLoading={submitting}
        destroyOnClose
      >
        <Form form={form} layout="vertical" initialValues={{ action: "GET", k8s_scope_enabled: false }}>
          <Form.Item label="能力名称" name="name" rules={[{ required: true, message: "请输入能力名称" }]}>
            <Input placeholder="例如：查询主机列表" />
          </Form.Item>
          <Form.Item label="资源路径" name="resource" rules={[{ required: true, message: "请输入资源路径" }]}>
            <Input placeholder="须与后端一致，例如 /api/v1/users 或 /api/v1/users/:id；撤销策略为 DELETE /api/v1/policies（勿写 :id）" />
          </Form.Item>
          <Form.Item label="HTTP 动作" name="action" rules={[{ required: true, message: "请选择动作" }]}>
            <Select options={HTTP_METHOD_OPTIONS} />
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
            pagination={{ pageSize: 10, showSizeChanger: true, pageSizeOptions: [10, 20, 50, 100], showQuickJumper: true }}
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

      <Drawer
        title="权限详情"
        open={detailOpen}
        onClose={() => {
          setDetailOpen(false);
          setDetailRecord(null);
        }}
        width={680}
        className="detail-edit-drawer"
        extra={
          <Button type="primary" loading={detailSubmitting} onClick={() => void submitDetailEdit()}>
            保存修改
          </Button>
        }
      >
        {detailRecord && (
          <Form form={detailForm} layout="vertical" className="detail-edit-form">
            <Form.Item label="ID">
              <Input value={String(detailRecord.id)} readOnly />
            </Form.Item>
            <Form.Item label="能力名称" name="name" rules={[{ required: true, message: "请输入能力名称" }]}>
              <Input />
            </Form.Item>
            <Form.Item label="资源路径" name="resource" rules={[{ required: true, message: "请输入资源路径" }]}>
              <Input />
            </Form.Item>
            <Form.Item label="HTTP 动作" name="action" rules={[{ required: true, message: "请选择动作" }]}>
              <Select options={HTTP_METHOD_OPTIONS} />
            </Form.Item>
            <Form.Item label="说明" name="description">
              <Input.TextArea rows={4} />
            </Form.Item>
            <Form.Item
              label="K8s 范围校验"
              name="k8s_scope_enabled"
              valuePropName="checked"
              extra="打开后该接口纳入 K8s 三元中间件目录（permissions.k8s_scope_enabled）；与「授权管理」中的 API 勾选相互独立。"
            >
              <Switch checkedChildren="已纳入" unCheckedChildren="未纳入" />
            </Form.Item>
            <Form.Item label="创建时间">
              <Input value={formatDateTime(detailRecord.created_at)} readOnly />
            </Form.Item>
            <Form.Item label="更新时间">
              <Input value={formatDateTime(detailRecord.updated_at)} readOnly />
            </Form.Item>
          </Form>
        )}
      </Drawer>
    </div>
  );
}