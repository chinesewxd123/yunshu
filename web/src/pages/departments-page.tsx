import { DeleteOutlined, EditOutlined, PlusOutlined, PlusSquareOutlined, ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Table, Tag, TreeSelect, Typography, message } from "antd";
import { useEffect, useMemo, useState } from "react";
import { createDepartment, deleteDepartment, getDepartmentTree, updateDepartment } from "../services/departments";
import { getUsers } from "../services/users";
import type { DepartmentItem, UserItem } from "../types/api";
import { useDictOptions } from "../hooks/use-dict-options";

type DepartmentFormValues = {
  parent_id?: number;
  name: string;
  code: string;
  sort?: number;
  status: number;
  leader_id?: number;
  phone?: string;
  email?: string;
  remark?: string;
};

export function DepartmentsPage() {
  const [list, setList] = useState<DepartmentItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [open, setOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [current, setCurrent] = useState<DepartmentItem | null>(null);
  const [users, setUsers] = useState<UserItem[]>([]);
  const [expandedRowKeys, setExpandedRowKeys] = useState<number[]>([]);
  const [form] = Form.useForm<DepartmentFormValues>();
  const statusOptions = useDictOptions("common_status");

  useEffect(() => {
    void Promise.all([loadTree(), loadUsers()]);
  }, []);

  async function loadTree() {
    setLoading(true);
    try {
      const data = await getDepartmentTree();
      setList(data);
      setExpandedRowKeys(collectDepartmentIDs(data));
    } finally {
      setLoading(false);
    }
  }

  async function loadUsers() {
    const res = await getUsers({ page: 1, page_size: 500 });
    setUsers(res.list);
  }

  const departmentTreeSelectData = useMemo(
    () =>
      buildDepartmentTreeSelectData(list, (item) => ({
        value: item.id,
        title: `${item.name} (${item.code})`,
      })),
    [list],
  );

  const leaderOptions = useMemo(
    () =>
      users.map((u) => ({
        value: u.id,
        label: `${u.nickname} (${u.username})`,
      })),
    [users],
  );

  function openCreate(parentID?: number) {
    setCurrent(null);
    form.resetFields();
    form.setFieldsValue({ parent_id: parentID, status: 1, sort: 0 });
    setOpen(true);
  }

  function openEdit(record: DepartmentItem) {
    setCurrent(record);
    form.setFieldsValue({
      parent_id: record.parent_id,
      name: record.name,
      code: record.code,
      sort: record.sort,
      status: record.status,
      leader_id: record.leader_id,
      phone: record.phone,
      email: record.email,
      remark: record.remark,
    });
    setOpen(true);
  }

  async function handleSubmit() {
    const values = await form.validateFields();
    setSubmitting(true);
    try {
      if (current) {
        await updateDepartment(current.id, values);
        message.success("部门信息已更新");
      } else {
        await createDepartment(values);
        message.success("部门创建成功");
      }
      setOpen(false);
      void loadTree();
    } finally {
      setSubmitting(false);
    }
  }

  async function handleDelete(id: number) {
    await deleteDepartment(id);
    message.success("部门已删除");
    void loadTree();
  }

  return (
    <Card className="table-card">
      <div className="toolbar">
        <Typography.Text type="secondary">组织架构支持无限层级、子树迁移与负责人绑定。</Typography.Text>
        <div className="toolbar__actions">
          <Button type="primary" icon={<PlusOutlined />} onClick={() => openCreate()}>
            新建部门
          </Button>
          <Button icon={<ReloadOutlined />} onClick={() => void loadTree()}>
            刷新
          </Button>
        </div>
      </div>

      <Table
        rowKey="id"
        loading={loading}
        dataSource={list}
        pagination={false}
        expandable={{
          expandedRowKeys,
          onExpandedRowsChange: (keys) => setExpandedRowKeys(keys as number[]),
        }}
        columns={[
          {
            title: "部门名称",
            dataIndex: "name",
            render: (_: string, record: DepartmentItem) => (
              <Space>
                <Typography.Text strong>{record.name}</Typography.Text>
                <Typography.Text className="inline-muted">({record.code})</Typography.Text>
              </Space>
            ),
          },
          { title: "负责人", dataIndex: "leader_name", render: (v: string) => v || "-" },
          { title: "成员数", dataIndex: "user_count", width: 100, align: "center" as const, render: (v: number) => v ?? 0 },
          { title: "层级", dataIndex: "level", width: 80, align: "center" as const },
          { title: "排序", dataIndex: "sort", width: 80, align: "center" as const },
          {
            title: "状态",
            dataIndex: "status",
            width: 90,
            render: (v: number) => (v === 1 ? <Tag className="status-chip status-chip--ok">启用</Tag> : <Tag className="status-chip status-chip--off">停用</Tag>),
          },
          {
            title: "操作",
            width: 250,
            render: (_: unknown, record: DepartmentItem) => (
              <Space size={4}>
                <Button type="link" size="small" icon={<PlusSquareOutlined />} onClick={() => openCreate(record.id)}>
                  子部门
                </Button>
                <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openEdit(record)}>
                  编辑
                </Button>
                <Popconfirm title="确认删除该部门吗？" onConfirm={() => void handleDelete(record.id)}>
                  <Button type="link" danger size="small" icon={<DeleteOutlined />}>
                    删除
                  </Button>
                </Popconfirm>
              </Space>
            ),
          },
        ]}
      />

      <Modal
        title={current ? `编辑部门 #${current.id}` : "新建部门"}
        open={open}
        onCancel={() => setOpen(false)}
        onOk={() => void handleSubmit()}
        confirmLoading={submitting}
        width={700}
        destroyOnClose
      >
        <Form form={form} layout="vertical" initialValues={{ status: 1, sort: 0 }}>
          <Space style={{ width: "100%" }} size="middle">
            <Form.Item label="上级部门" name="parent_id" style={{ flex: 1, marginBottom: 0 }}>
              <TreeSelect
                allowClear
                treeDefaultExpandAll
                placeholder="不选则为根部门"
                treeData={departmentTreeSelectData}
                style={{ width: "100%" }}
              />
            </Form.Item>
            <Form.Item label="负责人" name="leader_id" style={{ flex: 1, marginBottom: 0 }}>
              <Select allowClear showSearch optionFilterProp="label" options={leaderOptions} placeholder="请选择负责人" />
            </Form.Item>
          </Space>

          <Space style={{ width: "100%" }} size="middle">
            <Form.Item label="部门名称" name="name" rules={[{ required: true, message: "请输入部门名称" }]} style={{ flex: 1, marginBottom: 0 }}>
              <Input placeholder="例如：研发中心" />
            </Form.Item>
            <Form.Item label="部门编码" name="code" rules={[{ required: true, message: "请输入部门编码" }]} style={{ flex: 1, marginBottom: 0 }}>
              <Input placeholder="例如：RND-CENTER" />
            </Form.Item>
          </Space>

          <Space style={{ width: "100%" }} size="middle">
            <Form.Item label="联系邮箱" name="email" rules={[{ type: "email", message: "请输入正确邮箱地址" }]} style={{ flex: 1, marginBottom: 0 }}>
              <Input placeholder="例如：rd@example.com" />
            </Form.Item>
            <Form.Item label="联系电话" name="phone" style={{ flex: 1, marginBottom: 0 }}>
              <Input placeholder="例如：010-12345678" />
            </Form.Item>
          </Space>

          <Space style={{ width: "100%" }} size="middle">
            <Form.Item label="排序" name="sort" style={{ width: 140, marginBottom: 0 }}>
              <InputNumber min={0} style={{ width: "100%" }} />
            </Form.Item>
            <Form.Item label="状态" name="status" style={{ width: 160, marginBottom: 0 }}>
              <Select options={statusOptions} />
            </Form.Item>
          </Space>

          <Form.Item label="备注" name="remark" style={{ marginTop: 12 }}>
            <Input.TextArea rows={3} placeholder="部门职责、说明等" />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}

function collectDepartmentIDs(tree: DepartmentItem[]): number[] {
  const result: number[] = [];
  for (const item of tree) {
    result.push(item.id);
    if (item.children?.length) {
      result.push(...collectDepartmentIDs(item.children));
    }
  }
  return result;
}

function buildDepartmentTreeSelectData(
  nodes: DepartmentItem[],
  mapper: (item: DepartmentItem) => { value: number; title: string },
): Array<{ value: number; title: string; children?: any[] }> {
  return nodes.map((item) => {
    const base = mapper(item);
    return {
      ...base,
      children: item.children?.length ? buildDepartmentTreeSelectData(item.children, mapper) : undefined,
    };
  });
}
