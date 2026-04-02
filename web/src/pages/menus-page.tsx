import { PlusOutlined, ReloadOutlined, EditOutlined, DeleteOutlined, PlusSquareOutlined } from "@ant-design/icons";
import { Button, Card, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Switch, Table, Tag, Typography, message } from "antd";
import { useEffect, useState } from "react";
import { PageHero } from "../components/page-hero";
import { getMenuTree, createMenu, updateMenu, deleteMenu } from "../services/menus";
import type { MenuItem, MenuCreatePayload, MenuUpdatePayload } from "../services/menus";

export function MenusPage() {
  const [treeData, setTreeData] = useState<MenuItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [open, setOpen] = useState(false);
  const [current, setCurrent] = useState<MenuItem | null>(null);
  const [parentID, setParentID] = useState<number | undefined>();
  const [form] = Form.useForm<MenuCreatePayload>();

  useEffect(() => {
    void loadTree();
  }, []);

  async function loadTree() {
    setLoading(true);
    try {
      const data = await getMenuTree();
      setTreeData(data);
    } finally {
      setLoading(false);
    }
  }

  function openCreate(parentId?: number) {
    setCurrent(null);
    setParentID(parentId);
    form.resetFields();
    form.setFieldsValue({ status: 1, sort: 0 });
    setOpen(true);
  }

  function openEdit(record: MenuItem) {
    setCurrent(record);
    setParentID(record.parent_id);
    form.setFieldsValue({
      name: record.name,
      path: record.path,
      icon: record.icon,
      sort: record.sort,
      hidden: record.hidden,
      component: record.component,
      redirect: record.redirect,
      status: record.status,
    });
    setOpen(true);
  }

  async function handleSubmit() {
    const values = await form.validateFields();
    setSubmitting(true);
    try {
      if (current) {
        await updateMenu(current.id, values as MenuUpdatePayload);
        message.success("菜单已更新");
      } else {
        await createMenu({ ...values, parent_id: parentID } as MenuCreatePayload);
        message.success("菜单已创建");
      }
      setOpen(false);
      void loadTree();
    } finally {
      setSubmitting(false);
    }
  }

  async function handleDelete(id: number) {
    await deleteMenu(id);
    message.success("菜单已删除");
    void loadTree();
  }

  return (
    <div>
      <PageHero
        title="菜单管理"
        subtitle="维护 /api/v1/menus* 菜单元数据；保存后刷新即可驱动左侧导航（启用且未隐藏的项）。路由 path 需与 React Router 中已注册路径一致。"
        breadcrumbItems={[{ title: "控制台" }, { title: "菜单管理" }]}
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={() => openCreate()}>
            新增根菜单
          </Button>
        }
      />

      <Card className="table-card">
        <div className="toolbar">
          <Typography.Text type="secondary">点击节点可操作子级</Typography.Text>
          <div className="toolbar__actions">
            <Button icon={<ReloadOutlined />} onClick={() => void loadTree()}>
              刷新
            </Button>
          </div>
        </div>

        <Table
          loading={loading}
          pagination={false}
          expandable={{
            defaultExpandAllRows: true,
          }}
          dataSource={treeData}
          rowKey="id"
          columns={[
            {
              title: "菜单名称",
              dataIndex: "name",
              key: "name",
              render: (name: string, record: MenuItem) => (
                <Space>
                  {record.icon && <span style={{ fontSize: 16 }}>{record.icon}</span>}
                  <span>{name}</span>
                  {record.hidden && <Tag color="default">隐藏</Tag>}
                </Space>
              ),
            },
            { title: "路由路径", dataIndex: "path", render: (v: string) => v || "-" },
            { title: "组件路径", dataIndex: "component", render: (v: string) => v || "-" },
            { title: "重定向", dataIndex: "redirect", render: (v: string) => v || "-" },
            { title: "排序", dataIndex: "sort", width: 70 },
            { title: "状态", dataIndex: "status", width: 80, render: (s: number) => s === 1 ? <Tag color="success">启用</Tag> : <Tag color="default">停用</Tag> },
            {
              title: "操作",
              key: "action",
              width: 220,
              render: (_: unknown, record: MenuItem) => (
                <Space wrap>
                  <Button
                    type="link"
                    size="small"
                    icon={<PlusSquareOutlined />}
                    onClick={() => openCreate(record.id)}
                  >
                    添加子菜单
                  </Button>
                  <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openEdit(record)}>
                    编辑
                  </Button>
                  <Popconfirm title="确认删除该菜单吗？子菜单需先删除。" onConfirm={() => void handleDelete(record.id)}>
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
        title={current ? `编辑菜单 #${current.id}` : parentID ? `新增子菜单` : "新增根菜单"}
        open={open}
        onCancel={() => setOpen(false)}
        onOk={() => void handleSubmit()}
        confirmLoading={submitting}
        destroyOnClose
        width={600}
      >
        <Form form={form} layout="vertical" initialValues={{ status: 1, sort: 0 }}>
          <Form.Item label="菜单名称" name="name" rules={[{ required: true, message: "请输入菜单名称" }]}>
            <Input placeholder="例如：系统管理" />
          </Form.Item>
          <Space style={{ width: "100%" }} size="middle">
            <Form.Item label="路由路径" name="path" style={{ flex: 1, marginBottom: 0 }}>
              <Input placeholder="/system" />
            </Form.Item>
            <Form.Item label="组件路径" name="component" style={{ flex: 1, marginBottom: 0 }}>
              <Input placeholder="system/index" />
            </Form.Item>
          </Space>
          <Space style={{ width: "100%" }} size="middle">
            <Form.Item label="图标" name="icon" style={{ flex: 1, marginBottom: 0 }}>
              <Input placeholder="SettingOutlined" />
            </Form.Item>
            <Form.Item label="排序" name="sort" style={{ width: 120, marginBottom: 0 }}>
              <InputNumber min={0} style={{ width: "100%" }} />
            </Form.Item>
          </Space>
          <Form.Item label="重定向" name="redirect">
            <Input placeholder="/redirect/path" />
          </Form.Item>
          <Space style={{ width: "100%" }} size="middle">
            <Form.Item label="状态" name="status" style={{ flex: 1, marginBottom: 0 }}>
              <Select options={[{ label: "启用", value: 1 }, { label: "停用", value: 0 }]} />
            </Form.Item>
            <Form.Item label="是否隐藏" name="hidden" valuePropName="checked" style={{ flex: 1, marginBottom: 0 }} initialValue={false}>
              <Switch />
            </Form.Item>
          </Space>
        </Form>
      </Modal>
    </div>
  );
}
