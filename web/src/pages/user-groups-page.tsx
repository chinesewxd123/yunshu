import { DeleteOutlined, PlusOutlined, ReloadOutlined, TeamOutlined, UserOutlined } from "@ant-design/icons";
import { Button, Card, Drawer, Form, Input, Modal, Popconfirm, Select, Space, Table, Tag, Typography, message } from "antd";
import { useEffect, useState } from "react";
import { StatusTag } from "../components/status-tag";
import {
  assignUserGroupMembers,
  createUserGroup,
  deleteUserGroup,
  getUserGroup,
  listUserGroups,
  updateUserGroup,
  type UserGroupDetail,
  type UserGroupItem,
} from "../services/user-groups";
import { getUsers } from "../services/users";
import type { UserItem } from "../types/api";
import { formatDateTime } from "../utils/format";

const defaultQuery = { keyword: "", page: 1, page_size: 10 };

export function UserGroupsPage() {
  const [list, setList] = useState<UserGroupItem[]>([]);
  const [total, setTotal] = useState(0);
  const [query, setQuery] = useState(defaultQuery);
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [open, setOpen] = useState(false);
  const [form] = Form.useForm<{ name: string; code: string; description?: string; status: number }>();
  const [detailOpen, setDetailOpen] = useState(false);
  const [detail, setDetail] = useState<UserGroupDetail | null>(null);
  const [detailForm] = Form.useForm<{ name: string; description?: string; status: number }>();
  const [membersOpen, setMembersOpen] = useState(false);
  const [memberGroup, setMemberGroup] = useState<UserGroupItem | null>(null);
  const [users, setUsers] = useState<UserItem[]>([]);
  const [checkedUserIds, setCheckedUserIds] = useState<number[]>([]);
  useEffect(() => {
    void load();
  }, [query]);

  useEffect(() => {
    void loadUsers();
  }, []);

  async function loadUsers() {
    const res = await getUsers({ page: 1, page_size: 2000 });
    setUsers(res.list);
  }

  async function load(next = query) {
    setLoading(true);
    try {
      const res = await listUserGroups(next);
      setList(res.list);
      setTotal(res.total);
    } finally {
      setLoading(false);
    }
  }

  function openCreate() {
    form.resetFields();
    form.setFieldsValue({ status: 1 });
    setOpen(true);
  }

  async function submitCreate() {
    const v = await form.validateFields();
    setSubmitting(true);
    try {
      await createUserGroup({
        name: v.name.trim(),
        code: v.code.trim(),
        description: v.description?.trim(),
        status: v.status,
      });
      message.success("用户组已创建");
      setOpen(false);
      form.resetFields();
      void load();
    } finally {
      setSubmitting(false);
    }
  }

  async function openDetail(record: UserGroupItem) {
    setSubmitting(true);
    try {
      const d = await getUserGroup(record.id);
      setDetail(d);
      detailForm.setFieldsValue({
        name: d.name,
        description: d.description,
        status: d.status,
      });
      setDetailOpen(true);
    } finally {
      setSubmitting(false);
    }
  }

  async function saveDetail() {
    if (!detail) return;
    const v = await detailForm.validateFields();
    setSubmitting(true);
    try {
      await updateUserGroup(detail.id, {
        name: v.name.trim(),
        description: v.description?.trim(),
        status: v.status,
      });
      message.success("已保存");
      setDetailOpen(false);
      setDetail(null);
      void load();
    } finally {
      setSubmitting(false);
    }
  }

  function openMembers(g: UserGroupItem) {
    setMemberGroup(g);
    setCheckedUserIds([]);
    setMembersOpen(true);
    void (async () => {
      try {
        const d = await getUserGroup(g.id);
        setCheckedUserIds(d.members.map((m) => m.user_id));
      } catch {
        setCheckedUserIds([]);
      }
    })();
  }

  async function saveMembers() {
    if (!memberGroup) return;
    setSubmitting(true);
    try {
      await assignUserGroupMembers(memberGroup.id, { user_ids: checkedUserIds });
      message.success("成员已更新");
      setMembersOpen(false);
      setMemberGroup(null);
      void load();
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div>
      <Card
        className="table-card"
        title={
          <Space>
            <TeamOutlined />
            <span>用户组管理</span>
          </Space>
        }
        loading={loading}
        extra={
          <Space>
            <Button icon={<ReloadOutlined />} onClick={() => void load()}>
              刷新
            </Button>
            <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
              新建用户组
            </Button>
          </Space>
        }
      >
        <Typography.Paragraph type="secondary">
          用户组编码用于 K8s 集群权限主体（<Typography.Text code>principal_kind=group</Typography.Text>）。成员在登录后通过 JWT 上下文参与鉴权；请在「K8s
          集群访问档位」中按组编码下发策略。
        </Typography.Paragraph>
        <Space style={{ marginBottom: 12 }} wrap>
          <Input.Search
            allowClear
            placeholder="按名称或编码搜索"
            style={{ width: 260 }}
            onSearch={(keyword) => setQuery((q) => ({ ...q, keyword: keyword ?? "", page: 1 }))}
          />
        </Space>
        <Table<UserGroupItem>
          rowKey="id"
          size="small"
          dataSource={list}
          pagination={{
            current: query.page,
            pageSize: query.page_size,
            total,
            showSizeChanger: true,
            onChange: (page, pageSize) => setQuery((q) => ({ ...q, page, page_size: pageSize ?? q.page_size })),
          }}
          columns={[
            { title: "ID", dataIndex: "id", width: 70 },
            { title: "名称", dataIndex: "name" },
            { title: "编码", dataIndex: "code", render: (c: string) => <Typography.Text code>{c}</Typography.Text> },
            { title: "成员数", dataIndex: "member_count", width: 90 },
            {
              title: "状态",
              dataIndex: "status",
              width: 100,
              render: (s: number) => <StatusTag status={s} />,
            },
            { title: "创建时间", dataIndex: "created_at", width: 180, render: (t: string) => formatDateTime(t) },
            {
              title: "操作",
              key: "op",
              width: 280,
              render: (_, r) => (
                <Space size="small" wrap>
                  <Button type="link" size="small" onClick={() => void openDetail(r)}>
                    编辑
                  </Button>
                  <Button type="link" size="small" icon={<UserOutlined />} onClick={() => openMembers(r)}>
                    成员
                  </Button>
                  <Popconfirm
                    title="确定删除该用户组？"
                    onConfirm={() =>
                      void (async () => {
                        try {
                          await deleteUserGroup(r.id);
                          message.success("已删除");
                          void load();
                        } catch {
                          /* http 拦截器已提示 */
                        }
                      })()
                    }
                  >
                    <Button type="link" danger size="small" icon={<DeleteOutlined />}>
                      删除
                    </Button>
                  </Popconfirm>
                </Space>
              ),
            },
          ]}
        />
      </Card>

      <Modal title="新建用户组" open={open} onCancel={() => setOpen(false)} onOk={() => void submitCreate()} confirmLoading={submitting} destroyOnClose>
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="名称" rules={[{ required: true, message: "请输入名称" }]}>
            <Input placeholder="显示名称" />
          </Form.Item>
          <Form.Item
            name="code"
            label="编码"
            rules={[{ required: true, message: "请输入编码" }]}
            extra="创建后不可修改；用于 K8s principal_ref，建议英文、数字、连字符"
          >
            <Input placeholder="例如 platform-devs" />
          </Form.Item>
          <Form.Item name="description" label="说明">
            <Input.TextArea rows={2} />
          </Form.Item>
          <Form.Item name="status" label="状态" rules={[{ required: true }]}>
            <Select
              options={[
                { value: 1, label: "启用" },
                { value: 0, label: "停用" },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>

      <Drawer
        title={detail ? `编辑：${detail.name}` : "编辑"}
        width={480}
        open={detailOpen}
        onClose={() => {
          setDetailOpen(false);
          setDetail(null);
        }}
        destroyOnClose
        extra={
          <Space>
            <Button onClick={() => setDetailOpen(false)}>取消</Button>
            <Button type="primary" loading={submitting} onClick={() => void saveDetail()}>
              保存
            </Button>
          </Space>
        }
      >
        {detail && (
          <Form form={detailForm} layout="vertical">
            <Typography.Paragraph type="secondary">
              编码 <Typography.Text code>{detail.code}</Typography.Text> 不可修改。
            </Typography.Paragraph>
            <Form.Item name="name" label="名称" rules={[{ required: true }]}>
              <Input />
            </Form.Item>
            <Form.Item name="description" label="说明">
              <Input.TextArea rows={2} />
            </Form.Item>
            <Form.Item name="status" label="状态" rules={[{ required: true }]}>
              <Select
                options={[
                  { value: 1, label: "启用" },
                  { value: 0, label: "停用" },
                ]}
              />
            </Form.Item>
          </Form>
        )}
      </Drawer>

      <Drawer
        title={memberGroup ? `成员：${memberGroup.name}` : "成员"}
        width={560}
        open={membersOpen}
        onClose={() => {
          setMembersOpen(false);
          setMemberGroup(null);
        }}
        destroyOnClose
        extra={
          <Space>
            <Button onClick={() => setMembersOpen(false)}>取消</Button>
            <Button type="primary" loading={submitting} onClick={() => void saveMembers()}>
              保存
            </Button>
          </Space>
        }
      >
        <Typography.Paragraph type="secondary">选择属于该组的用户（全量覆盖）。</Typography.Paragraph>
        <Select
          mode="multiple"
          allowClear
          showSearch
          optionFilterProp="label"
          style={{ width: "100%" }}
          placeholder="搜索并选择用户"
          value={checkedUserIds}
          onChange={(v) => setCheckedUserIds(v as number[])}
          options={users.map((u) => ({
            value: u.id,
            label: `${u.nickname} (${u.username})`,
          }))}
        />
      </Drawer>
    </div>
  );
}
