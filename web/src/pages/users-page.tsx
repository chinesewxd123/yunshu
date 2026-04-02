import { PlusOutlined, ReloadOutlined, EyeOutlined } from "@ant-design/icons";
import {
  Button,
  Card,
  Descriptions,
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
import { PageHero } from "../components/page-hero";
import { StatusTag } from "../components/status-tag";
import { getRoleOptions } from "../services/roles";
import { assignUserRoles, createUser, deleteUser, getUsers, getUser, updateUser, exportUsers, importUsers } from "../services/users";
import type { RoleItem, UserCreatePayload, UserItem, UserUpdatePayload } from "../types/api";
import { formatDateTime } from "../utils/format";
import { buildRoleTreeData, normalizeCheckedKeys } from "../utils/tree";

const defaultQuery = { keyword: "", page: 1, page_size: 10 };

export function UsersPage() {
  const [list, setList] = useState<UserItem[]>([]);
  const [total, setTotal] = useState(0);
  const [query, setQuery] = useState(defaultQuery);
  const [loading, setLoading] = useState(false);
  const [roles, setRoles] = useState<RoleItem[]>([]);
  const [editorOpen, setEditorOpen] = useState(false);
  const [assignOpen, setAssignOpen] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [detailRecord, setDetailRecord] = useState<UserItem | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [current, setCurrent] = useState<UserItem | null>(null);
  const [roleTarget, setRoleTarget] = useState<UserItem | null>(null);
  const [checkedRoleIds, setCheckedRoleIds] = useState<number[]>([]);
  const [form] = Form.useForm<UserCreatePayload & UserUpdatePayload>();
  const fileInputRef = useRef<HTMLInputElement | null>(null);

  const roleTreeData = useMemo(() => buildRoleTreeData(roles), [roles]);
  const roleIdSet = useMemo(() => new Set(roles.map((role) => role.id)), [roles]);

  useEffect(() => {
    void loadUsers(query);
  }, [query]);

  useEffect(() => {
    void loadRoles();
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

  function openCreate() {
    setCurrent(null);
    form.resetFields();
    form.setFieldsValue({ status: 1, role_ids: [] });
    setEditorOpen(true);
  }

  function openEdit(record: UserItem) {
    setCurrent(record);
    form.setFieldsValue({
      nickname: record.nickname,
      email: record.email || "",
      status: record.status,
      password: "",
    });
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
    setDetailOpen(true);
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
      if (current) {
        const payload: UserUpdatePayload = {
          nickname: values.nickname,
          status: values.status,
        };
        if (values.email) {
          payload.email = values.email;
        }
        if (values.password) {
          payload.password = values.password;
        }
        await updateUser(current.id, payload);
        message.success("账号信息已更新");
      } else {
        await createUser({
          username: values.username,
          email: values.email,
          password: values.password,
          nickname: values.nickname,
          status: values.status ?? 1,
          role_ids: values.role_ids ?? [],
        });
        message.success("账号创建成功");
      }
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
      const blob = new Blob([res], { type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" });
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
    if (fileInputRef) fileInputRef.value = "";
  }

  return (
    <div>
      <PageHero
        title="账号治理"
        subtitle="维护用户与角色的绑定；保存角色会同步 Casbin。无角色用户无法访问需授权的管理接口。"
        breadcrumbItems={[{ title: "控制台" }, { title: "账号治理" }]}
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
            新建账号
          </Button>
        }
      />

      <Card className="table-card">
        <div className="toolbar">
          <Space wrap>
            <Input.Search
              allowClear
              placeholder="搜索账号、昵称或责任人"
              style={{ width: 280 }}
              onSearch={(keyword) => setQuery((prev) => ({ ...prev, keyword, page: 1 }))}
            />
          </Space>
          <div className="toolbar__actions">
            <input ref={(el) => (fileInputRef.current = el)} type="file" accept=".xlsx" style={{ display: "none" }} onChange={handleImportChange} />
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
                  <Button type="link" size="small" onClick={() => openEdit(record)}>
                    编辑
                  </Button>
                  <Button type="link" size="small" onClick={() => openAssign(record)}>
                    分配角色
                  </Button>
                  <Popconfirm title="确认删除该账号吗？" onConfirm={() => handleDelete(record)}>
                    <Button type="link" size="small" danger>
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
        title={current ? `编辑账号 #${current.id}` : "新建账号"}
        open={editorOpen}
        onCancel={() => setEditorOpen(false)}
        onOk={() => void submitEditor()}
        confirmLoading={submitting}
        destroyOnClose
      >
        <Form form={form} layout="vertical" initialValues={{ status: 1, role_ids: [] }}>
          {!current ? (
            <>
              <Form.Item label="账号名" name="username" rules={[{ required: true, message: "请输入账号名" }]}>
                <Input placeholder="例如：admin01" />
              </Form.Item>
              <Form.Item label="邮箱" name="email" rules={[{ required: true, type: "email", message: "请输入正确的邮箱地址" }]}>
                <Input placeholder="例如：admin@example.com" />
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
            </>
          ) : (
            <Form.Item label="邮箱" name="email" rules={[{ type: "email", message: "请输入正确的邮箱地址" }]}>
              <Input placeholder="留空则不修改邮箱" />
            </Form.Item>
          )}
          <Form.Item label="显示名称" name="nickname" rules={[{ required: true, message: "请输入显示名称" }]}>
            <Input placeholder="请输入显示名称" />
          </Form.Item>
          <Form.Item
            label={current ? "新密码" : "密码"}
            name="password"
            rules={current ? [] : [{ required: true, message: "请输入密码" }]}
          >
            <Input.Password placeholder={current ? "留空则不修改密码" : "请输入密码"} />
          </Form.Item>
          <Form.Item label="状态" name="status" rules={[{ required: true, message: "请选择状态" }]}>
            <Select options={[{ label: "启用", value: 1 }, { label: "停用", value: 0 }]} />
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

      <Modal
        title="用户详情"
        open={detailOpen}
        onCancel={() => setDetailOpen(false)}
        footer={null}
        width={650}
      >
        {detailRecord && (
          <Descriptions bordered column={2} size="middle">
            <Descriptions.Item label="ID">{detailRecord.id}</Descriptions.Item>
            <Descriptions.Item label="用户名">{detailRecord.username}</Descriptions.Item>
            <Descriptions.Item label="昵称">{detailRecord.nickname}</Descriptions.Item>
            <Descriptions.Item label="邮箱">{detailRecord.email || "-"}</Descriptions.Item>
            <Descriptions.Item label="状态" span={2}>
              <StatusTag status={detailRecord.status} />
            </Descriptions.Item>
            <Descriptions.Item label="角色" span={2}>
              {detailRecord.roles.length > 0 ? (
                detailRecord.roles.map((role) => (
                  <Tag color="blue" key={role.id} style={{ marginBottom: 4 }}>
                    {role.name} ({role.code})
                  </Tag>
                ))
              ) : (
                <Typography.Text type="secondary">暂无角色</Typography.Text>
              )}
            </Descriptions.Item>
            <Descriptions.Item label="创建时间">{formatDateTime(detailRecord.created_at)}</Descriptions.Item>
            <Descriptions.Item label="更新时间">{formatDateTime(detailRecord.updated_at)}</Descriptions.Item>
          </Descriptions>
        )}
      </Modal>
    </div>
  );
}
