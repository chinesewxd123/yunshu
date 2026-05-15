import { DeleteOutlined, EditOutlined, PlusOutlined, ReloadOutlined, TeamOutlined } from "@ant-design/icons";
import { Button, Card, Drawer, Form, Input, Modal, Popconfirm, Select, Space, Table, Tag, Tooltip, message } from "antd";
import { useEffect, useMemo, useState } from "react";
import { useAuth } from "../contexts/auth-context";
import { ProjectMembersPanel } from "../components/project-members-panel";
import { getDepartmentTree } from "../services/departments";
import {
  createProject,
  deleteProject,
  getProjects,
  updateProject,
  type ProjectCreatePayload,
  type ProjectItem,
  type ProjectUpdatePayload,
} from "../services/projects";
import type { DepartmentItem, UserItem } from "../types/api";
import { formatDateTime } from "../utils/format";

const defaultQuery = { keyword: "", page: 1, page_size: 10 };

function flattenDepartments(items: DepartmentItem[], prefix = ""): { value: number; label: string }[] {
  const out: { value: number; label: string }[] = [];
  for (const n of items) {
    const label = prefix ? `${prefix} / ${n.name}` : n.name;
    out.push({ value: n.id, label });
    if (n.children?.length) out.push(...flattenDepartments(n.children, label));
  }
  return out;
}

function isSuperAdminUser(u: UserItem | null | undefined): boolean {
  return Boolean(u?.roles?.some((r) => r.code === "super-admin"));
}

/** 可编辑项目元数据（PUT/DELETE /projects/:id），与网关中间件一致 */
function canEditProjectMeta(isSuper: boolean, myProjectRole: string | null | undefined): boolean {
  if (isSuper) return true;
  const r = String(myProjectRole || "").toLowerCase();
  return r === "owner" || r === "admin";
}

const MY_ROLE_LABEL: Record<string, string> = {
  owner: "负责人",
  admin: "管理员",
  member: "成员",
  readonly: "只读",
};

export function ProjectsPage() {
  const { user } = useAuth();
  const isSuper = useMemo(() => isSuperAdminUser(user), [user]);
  const [query, setQuery] = useState(defaultQuery);
  const [loading, setLoading] = useState(false);
  const [list, setList] = useState<ProjectItem[]>([]);
  const [total, setTotal] = useState(0);

  const [editorOpen, setEditorOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [current, setCurrent] = useState<ProjectItem | null>(null);
  const [form] = Form.useForm<ProjectCreatePayload & ProjectUpdatePayload>();

  const [memberProject, setMemberProject] = useState<ProjectItem | null>(null);
  const [deptTree, setDeptTree] = useState<DepartmentItem[]>([]);

  const departmentOptions = useMemo(() => flattenDepartments(deptTree), [deptTree]);

  useEffect(() => {
    void (async () => {
      try {
        const tree = await getDepartmentTree();
        setDeptTree(tree);
      } catch {
        setDeptTree([]);
      }
    })();
  }, []);

  useEffect(() => {
    void load();
  }, [query]);

  async function load() {
    setLoading(true);
    try {
      const data = await getProjects(query);
      setList(data.list);
      setTotal(data.total);
    } finally {
      setLoading(false);
    }
  }

  function openCreate() {
    setCurrent(null);
    form.resetFields();
    form.setFieldsValue({ status: 1 });
    setEditorOpen(true);
  }

  function openEdit(record: ProjectItem) {
    setCurrent(record);
    form.resetFields();
    form.setFieldsValue({
      name: record.name,
      code: record.code,
      description: record.description ?? undefined,
      status: record.status,
      owner_department_id: record.owner_department_id && record.owner_department_id > 0 ? record.owner_department_id : undefined,
    });
    setEditorOpen(true);
  }

  async function onSubmit() {
    const values = await form.validateFields();
    setSubmitting(true);
    try {
      if (!current) {
        const payload: ProjectCreatePayload = {
          name: values.name,
          code: values.code,
          description: values.description,
          status: values.status,
        };
        const od = values.owner_department_id;
        if (od !== undefined && od !== null && Number(od) > 0) {
          payload.owner_department_id = Number(od);
        }
        await createProject(payload);
        message.success("已创建项目（你已自动加入项目成员）");
      } else {
        const payload: ProjectUpdatePayload = {
          name: values.name,
          code: values.code,
          description: values.description,
          status: values.status,
        };
        const od = values.owner_department_id;
        if (od !== undefined && od !== null && Number(od) > 0) {
          payload.owner_department_id = Number(od);
        } else if (current.owner_department_id) {
          payload.owner_department_id = 0;
        }
        await updateProject(current.id, payload);
        message.success("已更新项目");
      }
      setEditorOpen(false);
      void load();
    } finally {
      setSubmitting(false);
    }
  }

  async function onDelete(record: ProjectItem) {
    await deleteProject(record.id);
    message.success("已删除");
    void load();
  }

  return (
    <Card
      title="项目列表"
      extra={
        <Space>
          <Input.Search
            placeholder="搜索 name/code"
            allowClear
            onSearch={(keyword) => setQuery((q) => ({ ...q, keyword, page: 1 }))}
            style={{ width: 240 }}
          />
          <Button icon={<ReloadOutlined />} onClick={() => void load()} loading={loading}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
            新增
          </Button>
        </Space>
      }
    >
      <Table
        rowKey="id"
        dataSource={list}
        loading={loading}
        pagination={{
          current: query.page,
          pageSize: query.page_size,
          total,
          showSizeChanger: true,
          pageSizeOptions: [10, 20, 50, 100],
          showQuickJumper: true,
          onChange: (page, pageSize) => setQuery((q) => ({ ...q, page, page_size: pageSize })),
        }}
        columns={[
          { title: "ID", dataIndex: "id", width: 80 },
          { title: "名称", dataIndex: "name" },
          { title: "编码", dataIndex: "code", width: 180 },
          {
            title: "归属部门",
            key: "owner_department_id",
            width: 220,
            ellipsis: true,
            render: (_: unknown, record: ProjectItem) => {
              const id = record.owner_department_id;
              if (!id) return <span className="inline-muted">—</span>;
              const label = departmentOptions.find((o) => o.value === id)?.label;
              return <span title={label}>{label ?? `ID ${id}`}</span>;
            },
          },
          {
            title: "状态",
            dataIndex: "status",
            width: 120,
            render: (v: number) => (v === 1 ? <Tag color="green">启用</Tag> : <Tag color="red">停用</Tag>),
          },
          {
            title: "我的角色",
            key: "my_project_role",
            width: 100,
            render: (_: unknown, r: ProjectItem) => {
              if (isSuper) return <Tag>超管</Tag>;
              const k = String(r.my_project_role || "").toLowerCase();
              if (!k) return <span className="inline-muted">—</span>;
              return <Tag>{MY_ROLE_LABEL[k] ?? r.my_project_role}</Tag>;
            },
          },
          { title: "创建时间", dataIndex: "created_at", width: 200, render: (v: string) => formatDateTime(v) },
          {
            title: "操作",
            width: 360,
            render: (_: unknown, record: ProjectItem) => {
              const canMeta = canEditProjectMeta(isSuper, record.my_project_role);
              const needAdminTip = "需要项目负责人或管理员权限（超级管理员不受限）";
              return (
                <Space size={6} wrap={false}>
                  <Button type="link" icon={<TeamOutlined />} onClick={() => setMemberProject(record)}>
                    成员
                  </Button>
                  <Tooltip title={!canMeta ? needAdminTip : undefined}>
                    <span>
                      <Button type="link" icon={<EditOutlined />} disabled={!canMeta} onClick={() => canMeta && openEdit(record)}>
                        编辑
                      </Button>
                    </span>
                  </Tooltip>
                  <Tooltip title={!canMeta ? needAdminTip : undefined}>
                    <span>
                      <Popconfirm title="确定删除该项目？" disabled={!canMeta} onConfirm={() => void onDelete(record)}>
                        <Button type="link" danger icon={<DeleteOutlined />} disabled={!canMeta}>
                          删除
                        </Button>
                      </Popconfirm>
                    </span>
                  </Tooltip>
                </Space>
              );
            },
          },
        ]}
      />

      <Drawer
        title={memberProject ? `项目成员 — ${memberProject.name}` : "项目成员"}
        placement="right"
        width={720}
        open={memberProject !== null}
        onClose={() => setMemberProject(null)}
        destroyOnClose
      >
        {memberProject ? <ProjectMembersPanel projectId={memberProject.id} /> : null}
      </Drawer>

      <Modal
        title={current ? "编辑项目" : "新增项目"}
        open={editorOpen}
        onCancel={() => setEditorOpen(false)}
        onOk={() => void onSubmit()}
        confirmLoading={submitting}
        destroyOnClose
      >
        <Form layout="vertical" form={form}>
          <Form.Item label="名称" name="name" rules={[{ required: true, message: "请输入名称" }]}>
            <Input placeholder="例如：生产环境项目" />
          </Form.Item>
          <Form.Item label="编码" name="code" rules={[{ required: true, message: "请输入编码" }]}>
            <Input placeholder="例如：prod" />
          </Form.Item>
          <Form.Item label="描述" name="description">
            <Input.TextArea rows={3} placeholder="可选" />
          </Form.Item>
          <Form.Item label="状态" name="status" rules={[{ required: true }]}>
            <Select
              options={[
                { value: 1, label: "启用" },
                { value: 0, label: "停用" },
              ]}
            />
          </Form.Item>
          <Form.Item
            label="归属部门"
            name="owner_department_id"
            extra="可选；用于组织维度归属。清空并保存将移除归属。"
          >
            <Select
              allowClear
              showSearch
              optionFilterProp="label"
              placeholder="不选则无归属部门"
              options={departmentOptions}
            />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
