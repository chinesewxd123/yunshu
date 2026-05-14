import {
  PlusOutlined,
  ReloadOutlined,
  EyeOutlined,
  DeleteOutlined,
  UserSwitchOutlined,
  LockOutlined,
  ClusterOutlined,
  AppstoreOutlined,
} from "@ant-design/icons";
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
import type { ColumnsType } from "antd/es/table";
import { useEffect, useMemo, useState, useRef } from "react";
import { useNavigate } from "react-router-dom";
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
import { listUserClusterAuth, type K8sUserClusterAuthRow } from "../services/k8s-policies";
import { formatDateTime } from "../utils/format";
import { buildRoleTreeData, normalizeCheckedKeys } from "../utils/tree";
import { useDictOptions } from "../hooks/use-dict-options";
import { useAuth } from "../contexts/auth-context";

const defaultQuery = { keyword: "", department_id: undefined as number | undefined, page: 1, page_size: 10 };

export function UsersPage() {
  const navigate = useNavigate();
  const { user: currentUser } = useAuth();
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
  const [resetPwdOpen, setResetPwdOpen] = useState(false);
  const [resetPwdTarget, setResetPwdTarget] = useState<UserItem | null>(null);
  const [resetPwdSubmitting, setResetPwdSubmitting] = useState(false);
  const [resetPwdForm] = Form.useForm<{ password: string; confirm: string }>();
  const statusOptions = useDictOptions("common_status");
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const [k8sAuthOpen, setK8sAuthOpen] = useState(false);
  const [k8sAuthTarget, setK8sAuthTarget] = useState<UserItem | null>(null);
  const [k8sAuthRows, setK8sAuthRows] = useState<K8sUserClusterAuthRow[]>([]);
  const [k8sAuthLoading, setK8sAuthLoading] = useState(false);
  const [permOpen, setPermOpen] = useState(false);
  const [permTarget, setPermTarget] = useState<UserItem | null>(null);

  const roleTreeData = useMemo(() => buildRoleTreeData(roles), [roles]);
  const roleIdSet = useMemo(() => new Set(roles.map((role) => role.id)), [roles]);
  const isSuperAdmin = useMemo(
    () => currentUser?.roles?.some((r) => r.code === "super-admin") ?? false,
    [currentUser?.roles],
  );

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
      phone: detail.phone || "",
      status: detail.status,
      department_id: detail.department_id,
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
      if (values.phone !== undefined) payload.phone = String(values.phone || "").trim();
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
        phone: String(values.phone || "").trim() || undefined,
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

  function openResetPassword(record: UserItem) {
    if (!isSuperAdmin) {
      message.warning("仅管理员（超级管理员）可修改其他用户的登录密码");
      return;
    }
    if (currentUser && record.id === currentUser.id) {
      message.warning("不能在此修改当前登录账号的密码");
      return;
    }
    setResetPwdTarget(record);
    resetPwdForm.resetFields();
    setResetPwdOpen(true);
  }

  async function submitResetPassword() {
    if (!resetPwdTarget) {
      return;
    }
    const values = await resetPwdForm.validateFields();
    if (values.password !== values.confirm) {
      message.error("两次输入的新密码不一致");
      return;
    }
    setResetPwdSubmitting(true);
    try {
      await updateUser(resetPwdTarget.id, { password: values.password });
      message.success("已更新该账号的登录密码");
      setResetPwdOpen(false);
      setResetPwdTarget(null);
    } finally {
      setResetPwdSubmitting(false);
    }
  }

  async function openK8sAuthClusters(record: UserItem) {
    setK8sAuthTarget(record);
    setK8sAuthOpen(true);
    setK8sAuthLoading(true);
    try {
      const res = await listUserClusterAuth(record.id);
      setK8sAuthRows(res.list ?? []);
    } catch (e) {
      message.error(e instanceof Error ? e.message : "加载 K8s 授权失败");
      setK8sAuthRows([]);
    } finally {
      setK8sAuthLoading(false);
    }
  }

  function openPermissionView(record: UserItem) {
    setPermTarget(record);
    setPermOpen(true);
  }

  function openK8sFromPermissionView() {
    if (!permTarget) return;
    setPermOpen(false);
    void openK8sAuthClusters(permTarget);
  }

  const k8sAuthColumns: ColumnsType<K8sUserClusterAuthRow> = [
    {
      title: "集群",
      dataIndex: "cluster_name",
      width: 220,
      render: (v: string, r: K8sUserClusterAuthRow) => (
        <Space direction="vertical" size={0}>
          <Typography.Text>{v}</Typography.Text>
          {r.grant_scope_all ? <Tag color="blue">全部集群</Tag> : null}
        </Space>
      ),
    },
    { title: "档位", dataIndex: "preset_label", width: 130 },
    {
      title: "限制命名空间",
      dataIndex: "allow_namespaces",
      ellipsis: true,
      render: (v: string) =>
        v ? <Typography.Text ellipsis={{ tooltip: v }}>{v}</Typography.Text> : <span className="inline-muted">未配置白名单</span>,
    },
    { title: "来源", dataIndex: "via", width: 200, ellipsis: true },
  ];

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
            {
              title: "用户组",
              dataIndex: "groups",
              width: 220,
              render: (_: unknown, record: UserItem) => {
                const gs = record.groups ?? [];
                if (gs.length === 0) return <span className="inline-muted">未加入</span>;
                return (
                  <Space size={[4, 4]} wrap>
                    {gs.map((g) => (
                      <Tag key={g.id} color="geekblue">
                        {g.name}
                      </Tag>
                    ))}
                  </Space>
                );
              },
            },
            { title: "所属部门", dataIndex: "department_name", render: (v: string) => v || <span className="inline-muted">未设置</span> },
            {
              title: "手机号",
              dataIndex: "phone",
              width: 130,
              render: (v: string) => (v ? String(v) : <span className="inline-muted">—</span>),
            },
            { title: "状态", dataIndex: "status", render: (value: number) => <StatusTag status={value} /> },
            { title: "创建时间", dataIndex: "created_at", render: formatDateTime },
            {
              title: "操作",
              key: "action",
              width: 420,
              render: (_: unknown, record: UserItem) => (
                <Space size={4} wrap>
                  <Button type="link" size="small" icon={<EyeOutlined />} onClick={() => openDetail(record)}>
                    详情
                  </Button>
                  <Button type="link" size="small" icon={<AppstoreOutlined />} onClick={() => openPermissionView(record)}>
                    权限查看
                  </Button>
                  <Button type="link" size="small" icon={<ClusterOutlined />} onClick={() => void openK8sAuthClusters(record)}>
                    授权集群
                  </Button>
                  {isSuperAdmin && currentUser && record.id !== currentUser.id ? (
                    <Button type="link" size="small" icon={<LockOutlined />} onClick={() => openResetPassword(record)}>
                      修改密码
                    </Button>
                  ) : null}
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
          <Form.Item label="手机号" name="phone" extra="选填；与钉钉/企微账号一致时可被告警 @ 提及">
            <Input placeholder="11 位手机号" maxLength={20} autoComplete="off" />
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
          <Space>
            {isSuperAdmin && detailRecord && currentUser && detailRecord.id !== currentUser.id ? (
              <Button icon={<LockOutlined />} onClick={() => openResetPassword(detailRecord)}>
                修改密码
              </Button>
            ) : null}
            <Button type="primary" loading={detailSubmitting} onClick={() => void submitDetailEdit()}>
              保存修改
            </Button>
          </Space>
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
            <Form.Item label="手机号" name="phone" extra="与钉钉/企微一致时可被监控规则处理人在 IM 通道 @">
              <Input placeholder="选填" maxLength={20} autoComplete="off" />
            </Form.Item>
            <Form.Item label="状态" name="status" rules={[{ required: true, message: "请选择状态" }]}>
              <Select options={statusOptions} />
            </Form.Item>
            <Form.Item label="所属部门" name="department_id">
              <Select allowClear options={departmentOptions} />
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
            <Form.Item label="用户组">
              <Input.TextArea
                rows={3}
                value={
                  (detailRecord.groups ?? []).length > 0
                    ? (detailRecord.groups ?? []).map((g) => `${g.name}（${g.code}）`).join("，")
                    : "未加入任何用户组；可在「用户组管理」中将用户加入组以继承 K8s 集群档位等授权。"
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

      <Drawer
        title={permTarget ? `权限概览 — ${permTarget.nickname}` : "权限概览"}
        open={permOpen}
        onClose={() => {
          setPermOpen(false);
          setPermTarget(null);
        }}
        width={520}
        destroyOnClose
      >
        {permTarget ? (
          <Space direction="vertical" size={16} style={{ width: "100%" }}>
            <div>
              <Typography.Text strong>责任域角色</Typography.Text>
              <div style={{ marginTop: 8 }}>
                {permTarget.roles.length > 0 ? (
                  <Space wrap>{permTarget.roles.map((r) => <Tag key={r.id}>{r.name}</Tag>)}</Space>
                ) : (
                  <Typography.Text type="secondary">未分配角色</Typography.Text>
                )}
              </div>
            </div>
            <div>
              <Typography.Text strong>用户组</Typography.Text>
              <Typography.Paragraph type="secondary" style={{ marginBottom: 8, fontSize: 12 }}>
                用户组与账号为多对多；成员关系在「用户组管理」中维护。组编码可作为 K8s 授权主体。
              </Typography.Paragraph>
              <div>
                {(permTarget.groups ?? []).length > 0 ? (
                  <Space wrap>
                    {(permTarget.groups ?? []).map((g) => (
                      <Tag key={g.id} color="geekblue">
                        {g.name}{" "}
                        <Typography.Text type="secondary" style={{ fontSize: 11 }}>
                          ({g.code})
                        </Typography.Text>
                      </Tag>
                    ))}
                  </Space>
                ) : (
                  <Typography.Text type="secondary">未加入用户组</Typography.Text>
                )}
              </div>
              <Button type="link" style={{ paddingLeft: 0 }} onClick={() => navigate("/user-groups")}>
                打开用户组管理
              </Button>
            </div>
            <div>
              <Typography.Text strong>Kubernetes 集群档位</Typography.Text>
              <Typography.Paragraph type="secondary" style={{ marginBottom: 8, fontSize: 12 }}>
                汇总直授、责任域角色、用户组在集群策略中的档位；与「授权集群」入口相同。
              </Typography.Paragraph>
              <Button type="primary" icon={<ClusterOutlined />} onClick={() => openK8sFromPermissionView()}>
                查看集群授权明细
              </Button>
            </div>
          </Space>
        ) : null}
      </Drawer>

      <Modal
        title={resetPwdTarget ? `修改登录密码：${resetPwdTarget.username}` : "修改密码"}
        open={resetPwdOpen}
        onCancel={() => {
          setResetPwdOpen(false);
          setResetPwdTarget(null);
        }}
        onOk={() => void submitResetPassword()}
        confirmLoading={resetPwdSubmitting}
        destroyOnClose
      >
        <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
          仅管理员（超级管理员）可为其他账号设置新密码；普通用户不能在个人中心自行改密。完成后请通知对方使用新密码登录。
        </Typography.Paragraph>
        <Form form={resetPwdForm} layout="vertical" autoComplete="off">
          <Form.Item
            label="新密码"
            name="password"
            rules={[
              { required: true, message: "请输入新密码" },
              { min: 6, message: "至少 6 位" },
            ]}
          >
            <Input.Password placeholder="新密码" autoComplete="new-password" />
          </Form.Item>
          <Form.Item
            label="确认新密码"
            name="confirm"
            rules={[
              { required: true, message: "请再次输入新密码" },
              { min: 6, message: "至少 6 位" },
            ]}
          >
            <Input.Password placeholder="再次输入" autoComplete="new-password" />
          </Form.Item>
        </Form>
      </Modal>

      <Drawer
        title={k8sAuthTarget ? `K8s 已授权集群 — ${k8sAuthTarget.username}` : "K8s 已授权集群"}
        open={k8sAuthOpen}
        onClose={() => {
          setK8sAuthOpen(false);
          setK8sAuthTarget(null);
          setK8sAuthRows([]);
        }}
        width={900}
        destroyOnClose
      >
        <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
          汇总用户直授、责任域角色、用户组在「集群档位」表中的授权；命名空间白名单来自 k8s_namespace_allow_rules。撤销请至「K8s 集群策略」或集群列表「已授权」中操作。
        </Typography.Paragraph>
        <Table<K8sUserClusterAuthRow>
          rowKey="row_key"
          size="small"
          loading={k8sAuthLoading}
          dataSource={k8sAuthRows}
          columns={k8sAuthColumns}
          scroll={{ x: 720 }}
          pagination={{ pageSize: 10, showTotal: (t) => `共 ${t} 条` }}
        />
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
