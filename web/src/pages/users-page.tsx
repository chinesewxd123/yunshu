import { PlusOutlined, ReloadOutlined, EyeOutlined, DeleteOutlined, UserSwitchOutlined } from "@ant-design/icons";
import {
  Button,
  Card,
  Drawer,
  Form,
  Input,
  Modal,
  Popconfirm,
  Select,
  Space,
  Table,
  Tag,
  Tree,
  TreeSelect,
  Typography,
  message,
} from "antd";
import { useEffect, useMemo, useState, useRef } from "react";
import { StatusTag } from "../components/status-tag";
import { getDepartmentTree } from "../services/departments";
import { getRoleOptions } from "../services/roles";
import {
  assignUserRoles,
  createUser,
  deleteUser,
  downloadUsersImportTemplate,
  getUsers,
  getUser,
  updateUser,
  exportUsers,
  importUsers,
} from "../services/users";
import type { DepartmentItem, RoleItem, UserCreatePayload, UserItem, UserUpdatePayload } from "../types/api";
import { formatDateTime } from "../utils/format";
import { buildRoleTreeData, normalizeCheckedKeys } from "../utils/tree";
import { useDictOptions } from "../hooks/use-dict-options";

const defaultQuery = { keyword: "", department_id: undefined as number | undefined, page: 1, page_size: 10 };

export function UsersPage() {
  const [list, setList] = useState<UserItem[]>([]);
  const [total, setTotal] = useState(0);
  const [query, setQuery] = useState(defaultQuery);
  const [loading, setLoading] = useState(false);
  const [roles, setRoles] = useState<RoleItem[]>([]);
  const [departments, setDepartments] = useState<DepartmentItem[]>([]);
  const [editorOpen, setEditorOpen] = useState(false);
  const [assignOpen, setAssignOpen] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailRecord, setDetailRecord] = useState<UserItem | null>(null);
  const [detailSubmitting, setDetailSubmitting] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [roleTarget, setRoleTarget] = useState<UserItem | null>(null);
  const [checkedRoleIds, setCheckedRoleIds] = useState<number[]>([]);
  const [form] = Form.useForm<UserCreatePayload & UserUpdatePayload>();
  const [detailForm] = Form.useForm<UserUpdatePayload>();
  const statusOptions = useDictOptions("common_status");
  const fileInputRef = useRef<HTMLInputElement | null>(null);

  const roleTreeData = useMemo(() => buildRoleTreeData(roles), [roles]);
  const roleIdSet = useMemo(() => new Set(roles.map((role) => role.id)), [roles]);

  useEffect(() => {
    void loadUsers(query);
  }, [query]);

  useEffect(() => {
    void loadRoles();
    void loadDepartments();
  }, []);

  async function loadUsers(nextQuery = query) {
    setLoading(true);
    try {
      const result = await getUsers(nextQuery);
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

  async function loadDepartments() {
    const data = await getDepartmentTree();
    setDepartments(data);
  }

  function openCreate() {
    form.resetFields();
    form.setFieldsValue({ status: 1, role_ids: [], department_id: undefined });
    setEditorOpen(true);
  }

  function openAssign(record: UserItem) {
    setRoleTarget(record);
    setCheckedRoleIds(record.roles.map((role) => role.id));
    setAssignOpen(true);
  }

  async function openDetail(record: UserItem) {
    const detail = await getUser(record.id);
    setDetailRecord(detail);
    detailForm.setFieldsValue({
      nickname: detail.nickname,
      email: detail.email || "",
      status: detail.status,
      department_id: detail.department_id,
      password: "",
    });
    setDetailOpen(true);
  }

  async function submitDetailEdit() {
    if (!detailRecord) return;
    const values = await detailForm.validateFields();
    setDetailSubmitting(true);
    try {
      const payload: UserUpdatePayload = {
        nickname: values.nickname,
        status: values.status,
        department_id: values.department_id,
      };
      if (values.email) payload.email = values.email;
      if (values.password) payload.password = values.password;
      await updateUser(detailRecord.id, payload);
      message.success("用户详情已更新");
      setDetailOpen(false);
      setDetailRecord(null);
      await loadUsers();
    } finally {
      setDetailSubmitting(false);
    }
  }

  async function handleDelete(record: UserItem) {
    await deleteUser(record.id);
    message.success(`已删除账号 ${record.username}`);
    void loadUsers();
  }

  async function submitEditor() {
    const values = await form.validateFields();
    setSubmitting(true);
    try {
      await createUser({
        username: values.username,
        email: values.email,
        password: values.password,
        nickname: values.nickname,
        status: values.status ?? 1,
        department_id: values.department_id,
        role_ids: values.role_ids ?? [],
      });
      message.success("账号创建成功");
      setEditorOpen(false);
      form.resetFields();
      void loadUsers();
    } finally {
      setSubmitting(false);
    }
  }

  async function submitAssign() {
    if (!roleTarget) {
      return;
    }

    setSubmitting(true);
    try {
      await assignUserRoles(roleTarget.id, { role_ids: checkedRoleIds.filter((id) => roleIdSet.has(id)) });
      message.success("责任域角色已同步");
      setAssignOpen(false);
      setCheckedRoleIds([]);
      void loadUsers();
    } finally {
      setSubmitting(false);
    }
  }

  async function handleExport() {
    try {
      const res = await exportUsers({ keyword: query.keyword });
      const blob = res;
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = "users.xlsx";
      a.click();
      window.URL.revokeObjectURL(url);
    } catch (err) {
      message.error("导出失败");
    }
  }

  async function handleImportChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    try {
      await importUsers(file);
      message.success("导入完成");
      void loadUsers();
    } catch (err) {
      message.error("导入失败");
    }
    // reset
    if (fileInputRef.current) fileInputRef.current.value = "";
  }

  async function handleDownloadTemplate() {
    try {
      const blob = await downloadUsersImportTemplate();
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = "users-import-template.xlsx";
      a.click();
      window.URL.revokeObjectURL(url);
    } catch {
      message.error("模板下载失败");
    }
  }

  const departmentOptions = useMemo(
    () => toDepartmentOptions(departments).map((item) => ({ value: item.value, label: item.label })),
    [departments],
  );

  return (
    <div>
      <Card className="table-card">
        <div className="toolbar">
          <Space wrap>
            <Input.Search
              allowClear
              placeholder="搜索账号、昵称或责任人"
              style={{ width: 280 }}
              onSearch={(keyword) => setQuery((prev) => ({ ...prev, keyword, page: 1 }))}
            />
            <Select
              allowClear
              placeholder="按部门筛选"
              style={{ width: 220 }}
              options={departmentOptions}
              value={query.department_id}
              onChange={(departmentID) => setQuery((prev) => ({ ...prev, department_id: departmentID, page: 1 }))}
            />
          </Space>
          <div className="toolbar__actions">
            <input ref={(el) => (fileInputRef.current = el)} type="file" accept=".xlsx" style={{ display: "none" }} onChange={handleImportChange} />
            <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
              新建账号
            </Button>
            <Button onClick={() => void handleDownloadTemplate()}>模板</Button>
            <Button onClick={() => fileInputRef.current?.click()}>导入</Button>
            <Button onClick={() => void handleExport()}>导出</Button>
            <Button icon={<ReloadOutlined />} onClick={() => void loadUsers()}>
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
            pageSizeOptions: [10, 20, 50, 100],
            showQuickJumper: true,
            onChange: (page, pageSize) => setQuery((prev) => ({ ...prev, page, page_size: pageSize })),
          }}
          columns={[
            { title: "ID", dataIndex: "id", width: 70 },
            {
              title: "账号标识",
              dataIndex: "username",
              render: (_: string, record: UserItem) => (
                <Space direction="vertical" size={0}>
                  <Typography.Text strong>{record.nickname}</Typography.Text>
                  <Typography.Text className="inline-muted">{record.username}</Typography.Text>
                </Space>
              ),
            },
            {
              title: "责任域角色",
              dataIndex: "roles",
              render: (value: RoleItem[]) =>
                value.length > 0 ? value.map((role) => <Tag key={role.id}>{role.name}</Tag>) : <span className="inline-muted">未分配</span>,
            },
            { title: "所属部门", dataIndex: "department_name", render: (v: string) => v || <span className="inline-muted">未设置</span> },
            { title: "状态", dataIndex: "status", render: (value: number) => <StatusTag status={value} /> },
            { title: "创建时间", dataIndex: "created_at", render: formatDateTime },
            {
              title: "操作",
              key: "action",
              width: 260,
              render: (_: unknown, record: UserItem) => (
                <Space size={4}>
                  <Button type="link" size="small" icon={<EyeOutlined />} onClick={() => openDetail(record)}>
                    详情
                  </Button>
                  <Button type="link" size="small" icon={<UserSwitchOutlined />} onClick={() => openAssign(record)}>
                    分配角色
                  </Button>
                  <Popconfirm title="确认删除该账号吗？" onConfirm={() => handleDelete(record)}>
                    <Button type="link" size="small" danger icon={<DeleteOutlined />}>
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
        title="新建账号"
        open={editorOpen}
        onCancel={() => setEditorOpen(false)}
        onOk={() => void submitEditor()}
        confirmLoading={submitting}
        destroyOnClose
      >
        <Form form={form} layout="vertical" initialValues={{ status: 1, role_ids: [] }} autoComplete="off">
          <Form.Item label="账号名" name="username" rules={[{ required: true, message: "请输入账号名" }]}>
            <Input placeholder="例如：admin01" autoComplete="off" />
          </Form.Item>
          <Form.Item label="邮箱" name="email" rules={[{ required: true, type: "email", message: "请输入正确的邮箱地址" }]}>
            <Input placeholder="例如：admin@example.com" autoComplete="off" />
          </Form.Item>
          <Form.Item label="初始责任域" name="role_ids">
            <TreeSelect
              treeCheckable
              treeDefaultExpandAll
              showCheckedStrategy={TreeSelect.SHOW_CHILD}
              placeholder="可选，创建时直接绑定角色模板"
              treeData={roleTreeData}
              maxTagCount="responsive"
              style={{ width: "100%" }}
            />
          </Form.Item>
          <Form.Item label="显示名称" name="nickname" rules={[{ required: true, message: "请输入显示名称" }]}>
            <Input placeholder="请输入显示名称" />
          </Form.Item>
          <Form.Item label="所属部门" name="department_id">
            <Select allowClear placeholder="可选，选择用户所属部门" options={departmentOptions} />
          </Form.Item>
          <Form.Item
            label="密码"
            name="password"
            rules={[{ required: true, message: "请输入密码" }]}
          >
            <Input.Password placeholder="请输入密码" autoComplete="new-password" />
          </Form.Item>
          <Form.Item label="状态" name="status" rules={[{ required: true, message: "请选择状态" }]}>
            <Select options={statusOptions} />
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={roleTarget ? `为 ${roleTarget.nickname} 分配责任域角色` : "分配角色"}
        open={assignOpen}
        onCancel={() => {
          setAssignOpen(false);
          setCheckedRoleIds([]);
        }}
        onOk={() => void submitAssign()}
        confirmLoading={submitting}
        destroyOnClose
      >
        <Space direction="vertical" size={12} style={{ width: "100%" }}>
          <Typography.Text className="inline-muted">
            使用树形勾选为当前账号分配角色模板，已选 {checkedRoleIds.length} 个角色。
          </Typography.Text>
          <div className="tree-shell">
            <Tree
              checkable
              defaultExpandAll
              checkedKeys={checkedRoleIds}
              treeData={roleTreeData}
              onCheck={(checkedKeys) => {
                const nextIds = normalizeCheckedKeys(checkedKeys).filter((id) => roleIdSet.has(id));
                setCheckedRoleIds(nextIds);
              }}
            />
          </div>
        </Space>
      </Modal>

      <Drawer
        title="用户详情"
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
          <Form form={detailForm} layout="vertical" className="detail-edit-form" autoComplete="off">
            <Form.Item label="ID">
              <Input value={String(detailRecord.id)} readOnly />
            </Form.Item>
            <Form.Item label="用户名">
              <Input value={detailRecord.username} readOnly />
            </Form.Item>
            <Form.Item label="昵称" name="nickname" rules={[{ required: true, message: "请输入昵称" }]}>
              <Input autoComplete="off" />
            </Form.Item>
            <Form.Item label="邮箱" name="email" rules={[{ type: "email", message: "请输入正确邮箱" }]}>
              <Input autoComplete="off" />
            </Form.Item>
            <Form.Item label="状态" name="status" rules={[{ required: true, message: "请选择状态" }]}>
              <Select options={statusOptions} />
            </Form.Item>
            <Form.Item label="所属部门" name="department_id">
              <Select allowClear options={departmentOptions} />
            </Form.Item>
            <Form.Item label="新密码" name="password">
              <Input.Password placeholder="留空则不修改密码" autoComplete="new-password" />
            </Form.Item>
            <Form.Item label="角色">
              <Input.TextArea
                rows={3}
                value={
                  detailRecord.roles.length > 0
                    ? detailRecord.roles.map((role) => `${role.name} (${role.code})`).join("，")
                    : "暂无角色"
                }
                readOnly
              />
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

function toDepartmentOptions(tree: DepartmentItem[], prefix = ""): Array<{ value: number; label: string }> {
  const result: Array<{ value: number; label: string }> = [];
  for (const item of tree) {
    const label = prefix ? `${prefix} / ${item.name}` : item.name;
    result.push({ value: item.id, label });
    if (item.children?.length) {
      result.push(...toDepartmentOptions(item.children, label));
    }
  }
  return result;
}
