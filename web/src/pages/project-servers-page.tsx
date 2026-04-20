import {
  ApiOutlined,
  DeleteOutlined,
  DownloadOutlined,
  EditOutlined,
  LinkOutlined,
  PlusOutlined,
  ReloadOutlined,
  SyncOutlined,
  UploadOutlined,
} from "@ant-design/icons";
import { Button, Card, Col, Form, Input, InputNumber, Modal, Popconfirm, Row, Select, Space, Table, Tag, Tree, Typography, message } from "antd";
import type { DataNode } from "antd/es/tree";
import { useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  batchTestProjectServers,
  deleteProjectServer,
  deleteProjectServerGroup,
  downloadProjectServersImportTemplate,
  exportProjectServers,
  getProjectCloudAccounts,
  getProjectServerDetail,
  getProjectServerGroupTree,
  getProjectServers,
  getProjects,
  importProjectServers,
  syncProjectCloudAccount,
  testProjectServer,
  upsertProjectCloudAccount,
  upsertProjectServer,
  upsertProjectServerGroup,
  type CloudAccountItem,
  type ProjectItem,
  type ServerGroupItem,
  type ServerItem,
  type ServerUpsertPayload,
} from "../services/projects";
import { useDictOptions } from "../hooks/use-dict-options";
import { DictLabelFillSelect } from "../components/dict-fill-select";
import { formatDateTime } from "../utils/format";

const defaultQuery = { keyword: "", page: 1, page_size: 10 };

export function ProjectServersPage() {
  const navigate = useNavigate();
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [query, setQuery] = useState(defaultQuery);
  const [loading, setLoading] = useState(false);
  const [list, setList] = useState<ServerItem[]>([]);
  const [total, setTotal] = useState(0);
  const [selectedRowKeys, setSelectedRowKeys] = useState<number[]>([]);
  const [batchTesting, setBatchTesting] = useState(false);
  const [batchModal, setBatchModal] = useState<{ total: number; success: number; failed: number; results: { server_id: number; ok: boolean; message: string }[] } | null>(null);

  const [groups, setGroups] = useState<ServerGroupItem[]>([]);
  const [selectedGroupId, setSelectedGroupId] = useState<number>();
  const [selectedGroup, setSelectedGroup] = useState<ServerGroupItem | null>(null);
  const [groupEditorOpen, setGroupEditorOpen] = useState(false);
  const [editingGroup, setEditingGroup] = useState<ServerGroupItem | null>(null);
  const [groupForm] = Form.useForm<{ name: string; category: string; provider: string }>();

  const [cloudAccounts, setCloudAccounts] = useState<CloudAccountItem[]>([]);
  const [syncResultByAccount, setSyncResultByAccount] = useState<Record<number, { added: number; updated: number; disabled: number; unchanged: number }>>({});
  const [cloudSubmitting, setCloudSubmitting] = useState(false);
  const [cloudForm] = Form.useForm<{ account_name: string; region_scope: string; ak: string; sk: string }>();

  const [editorOpen, setEditorOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [current, setCurrent] = useState<ServerItem | null>(null);
  const [form] = Form.useForm<ServerUpsertPayload & { port_dict_label?: string }>();
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const [tableScrollY, setTableScrollY] = useState(360);

  const projectOptions = useMemo(() => projects.map((p) => ({ value: p.id, label: `${p.name} (${p.code})` })), [projects]);
  const serverGroupCategoryOptions = useDictOptions("server_group_category");
  const serverOsOptions = useDictOptions("server_os_type");
  const serverAuthOptions = useDictOptions("server_auth_type");
  const serverPortDict = useDictOptions("server_port");
  const sshUserDict = useDictOptions("server_ssh_username");
  const sshPwdDict = useDictOptions("server_ssh_password");
  const isCloudNode = selectedGroup?.category === "cloud" && selectedGroup?.provider !== "custom";

  useEffect(() => {
    void (async () => {
      const data = await getProjects({ page: 1, page_size: 1000 });
      setProjects(data.list);
      if (data.list[0]) setProjectId(data.list[0].id);
    })();
  }, []);

  useEffect(() => {
    if (!projectId) return;
    void loadGroups();
  }, [projectId]);

  useEffect(() => {
    if (!projectId || !selectedGroupId) return;
    void loadServers();
    if (isCloudNode) void loadCloudAccounts();
  }, [projectId, selectedGroupId, query.keyword, query.page, query.page_size, isCloudNode]);

  useEffect(() => {
    const recalc = () => {
      // Keep page filled while preserving toolbar/filter/pagination space.
      const next = Math.max(260, window.innerHeight - 430);
      setTableScrollY(next);
    };
    recalc();
    window.addEventListener("resize", recalc);
    return () => window.removeEventListener("resize", recalc);
  }, []);

  async function loadGroups() {
    if (!projectId) return;
    const tree = await getProjectServerGroupTree(projectId);
    setGroups(tree);
    const flat = flatten(tree);
    const picked = flat.find((it) => it.category === "self_hosted") || flat[0];
    if (picked) {
      setSelectedGroupId(picked.id);
      setSelectedGroup(picked);
    }
  }

  function flatten(items: ServerGroupItem[]): ServerGroupItem[] {
    const out: ServerGroupItem[] = [];
    const walk = (arr: ServerGroupItem[]) => {
      arr.forEach((it) => {
        out.push(it);
        if (it.children?.length) walk(it.children);
      });
    };
    walk(items);
    return out;
  }

  async function loadServers() {
    if (!projectId || !selectedGroupId) return;
    setLoading(true);
    try {
      const data = await getProjectServers(projectId, {
        ...query,
        group_id: selectedGroupId,
        source_type: selectedGroup?.category === "cloud" ? "cloud" : "self_hosted",
        provider: selectedGroup?.category === "cloud" ? (selectedGroup?.provider || undefined) : undefined,
      });
      setList(data.list || []);
      setTotal(data.total || 0);
      setSelectedRowKeys([]);
    } finally {
      setLoading(false);
    }
  }

  async function loadCloudAccounts() {
    if (!projectId || !selectedGroupId) return;
    const list = await getProjectCloudAccounts(projectId, selectedGroupId);
    setCloudAccounts(list);
  }

  function toTreeData(items: ServerGroupItem[]): DataNode[] {
    return items.map((it) => ({
      key: String(it.id),
      title: (
        <Space size={6}>
          <span>{it.name}</span>
          <Tag>{it.category === "cloud" ? "云" : "自建"}</Tag>
        </Space>
      ),
      children: it.children ? toTreeData(it.children) : undefined,
    }));
  }

  function removeNode(items: ServerGroupItem[], id: number): { tree: ServerGroupItem[]; removed: ServerGroupItem | null } {
    let removed: ServerGroupItem | null = null;
    const walk = (arr: ServerGroupItem[]): ServerGroupItem[] => {
      const out: ServerGroupItem[] = [];
      for (const it of arr) {
        if (it.id === id) {
          removed = { ...it, children: it.children ? [...it.children] : [] };
          continue;
        }
        out.push({ ...it, children: it.children ? walk(it.children) : [] });
      }
      return out;
    };
    return { tree: walk(items), removed };
  }

  function insertNode(items: ServerGroupItem[], targetID: number, node: ServerGroupItem, dropToGap: boolean, dropAfter: boolean): ServerGroupItem[] {
    const walk = (arr: ServerGroupItem[]): ServerGroupItem[] => {
      const out: ServerGroupItem[] = [];
      for (let i = 0; i < arr.length; i++) {
        const it = { ...arr[i], children: arr[i].children ? walk(arr[i].children!) : [] };
        if (it.id === targetID) {
          if (!dropToGap) {
            const children = [...(it.children || []), node];
            out.push({ ...it, children });
          } else if (dropAfter) {
            out.push(it, node);
          } else {
            out.push(node, it);
          }
        } else {
          out.push(it);
        }
      }
      return out;
    };
    return walk(items);
  }

  function normalizeSort(items: ServerGroupItem[], parentID?: number): ServerGroupItem[] {
    return items.map((it, idx) => ({
      ...it,
      parent_id: parentID ?? null,
      sort: (idx + 1) * 10,
      children: it.children ? normalizeSort(it.children, it.id) : [],
    }));
  }

  function flattenForSave(items: ServerGroupItem[]): ServerGroupItem[] {
    const out: ServerGroupItem[] = [];
    const walk = (arr: ServerGroupItem[]) => {
      arr.forEach((it) => {
        out.push(it);
        if (it.children?.length) walk(it.children);
      });
    };
    walk(items);
    return out;
  }

  function openCreate() {
    if (!projectId || !selectedGroupId) return;
    setCurrent(null);
    form.resetFields();
    form.setFieldsValue({
      project_id: projectId,
      group_id: selectedGroupId,
      port: 22,
      port_dict_label: undefined,
      os_type: "linux",
      status: 1,
      auth_type: "password",
      source_type: selectedGroup?.category === "cloud" ? "cloud" : "self_hosted",
      provider: selectedGroup?.provider || "",
      username_dict_label: undefined,
      password_dict_label: undefined,
    });
    setEditorOpen(true);
  }

  async function openEdit(record: ServerItem) {
    if (!projectId) return;
    setCurrent(record);
    form.resetFields();
    try {
      const detail = await getProjectServerDetail(projectId, record.id);
      form.setFieldsValue({
        ...detail,
        id: detail.id,
        project_id: detail.project_id,
        group_id: detail.group_id ?? selectedGroupId,
        auth_type: (detail.auth_type === "key" ? "key" : "password") as "password" | "key",
        username: detail.username ?? "",
        password: "",
        private_key: "",
        passphrase: "",
        port_dict_label: undefined,
        username_dict_label: detail.username_dict_label ?? undefined,
        password_dict_label: detail.password_dict_label ?? undefined,
      });
    } catch {
      form.setFieldsValue({
        ...record,
        id: record.id,
        project_id: record.project_id,
        group_id: record.group_id ?? selectedGroupId,
        auth_type: "password",
        port_dict_label: undefined,
        username_dict_label: undefined,
        password_dict_label: undefined,
      });
    }
    setEditorOpen(true);
  }

  async function onSubmit() {
    if (!projectId) return;
    const values = await form.validateFields();
    setSubmitting(true);
    try {
      const status = typeof values.status === "number" ? values.status : (current?.status ?? 1);
      const payload: ServerUpsertPayload = {
        id: values.id,
        project_id: values.project_id,
        group_id: values.group_id,
        name: values.name,
        host: values.host,
        port: values.port,
        os_type: values.os_type,
        tags: values.tags,
        status,
        source_type: values.source_type,
        provider: values.provider,
        cloud_instance_id: values.cloud_instance_id,
        cloud_region: values.cloud_region,
        auth_type: values.auth_type,
        username: (values.username || "").trim(),
        username_dict_label: (values.username_dict_label || "").trim() || undefined,
        password_dict_label: (values.password_dict_label || "").trim() || undefined,
      };
      const pw = (values.password || "").trim();
      if (pw) payload.password = pw;
      const pk = (values.private_key || "").trim();
      if (pk) payload.private_key = pk;
      const pp = (values.passphrase || "").trim();
      if (pp) payload.passphrase = pp;

      await upsertProjectServer(projectId, payload);
      message.success(current ? "已更新服务器" : "已新增服务器");
      setEditorOpen(false);
      void loadServers();
    } finally {
      setSubmitting(false);
    }
  }

  async function onDelete(record: ServerItem) {
    if (!projectId) return;
    await deleteProjectServer(projectId, record.id);
    message.success("已删除");
    void loadServers();
  }

  async function onBatchTest() {
    if (!projectId) return;
    if (selectedRowKeys.length === 0) return message.warning("请先勾选要测试的服务器");
    setBatchTesting(true);
    try {
      const res = await batchTestProjectServers(projectId, selectedRowKeys, 5);
      setBatchModal(res);
      message.success(`批量测试完成：成功 ${res.success}，失败 ${res.failed}`);
      void loadServers();
    } finally {
      setBatchTesting(false);
    }
  }

  async function onImport(file: File) {
    if (!projectId) return;
    const res = await importProjectServers(projectId, file);
    message.success(`已导入 ${res.imported} 条`);
    void loadServers();
  }

  async function onSaveGroup() {
    if (!projectId) return;
    const values = await groupForm.validateFields();
    await upsertProjectServerGroup(projectId, {
      id: editingGroup?.id,
      name: values.name,
      category: values.category,
      provider: values.provider,
      parent_id: editingGroup?.parent_id ?? (values.category === "cloud" && selectedGroup ? selectedGroup.id : undefined),
      sort: editingGroup?.sort,
      status: editingGroup?.status ?? 1,
    });
    message.success(editingGroup ? "分组已更新" : "分组已保存");
    setEditingGroup(null);
    setGroupEditorOpen(false);
    await loadGroups();
  }

  async function onDeleteGroup() {
    if (!projectId || !selectedGroupId) return;
    await deleteProjectServerGroup(projectId, selectedGroupId);
    message.success("分组已删除");
    await loadGroups();
  }

  async function onSaveCloudAccount() {
    if (!projectId || !selectedGroupId || !selectedGroup) return;
    const values = await cloudForm.validateFields();
    setCloudSubmitting(true);
    try {
      await upsertProjectCloudAccount(projectId, {
        group_id: selectedGroupId,
        provider: selectedGroup.provider || "alibaba",
        account_name: values.account_name,
        region_scope: values.region_scope,
        ak: values.ak,
        sk: values.sk,
        status: 1,
      });
      message.success("云账号已保存");
      cloudForm.resetFields(["ak", "sk"]);
      await loadCloudAccounts();
    } finally {
      setCloudSubmitting(false);
    }
  }

  async function onSync(accountId: number) {
    if (!projectId) return;
    const res = await syncProjectCloudAccount(projectId, accountId);
    setSyncResultByAccount((prev) => ({
      ...prev,
      [accountId]: { added: res.added, updated: res.updated, disabled: res.disabled, unchanged: res.unchanged },
    }));
    message.success(`同步完成：发现 ${res.total}，新增 ${res.added}，更新 ${res.updated}，停用 ${res.disabled}，不变 ${res.unchanged}`);
    await Promise.all([loadCloudAccounts(), loadServers()]);
  }

  return (
    <Card title="服务器管理" style={{ height: "calc(100vh - 130px)" }} bodyStyle={{ height: "100%", paddingBottom: 12 }}>
      <Row gutter={16} align="stretch" style={{ height: "100%" }}>
        <Col span={6} style={{ height: "100%", display: "flex" }}>
          <Card
            size="small"
            style={{ height: "100%", width: "100%" }}
            title="分组树"
            extra={<Space><Button size="small" icon={<PlusOutlined />} onClick={() => { setEditingGroup(null); groupForm.setFieldsValue({ name: "", category: "self_hosted", provider: "custom" }); setGroupEditorOpen(true); }}>新增</Button><Button size="small" icon={<ReloadOutlined />} onClick={() => void loadGroups()} /></Space>}
          >
            <div style={{ maxHeight: "calc(100vh - 310px)", overflow: "auto", paddingRight: 4 }}>
              <Tree
                draggable
                selectedKeys={selectedGroupId ? [String(selectedGroupId)] : []}
                treeData={toTreeData(groups)}
                onSelect={(keys) => {
                  const id = Number(keys[0]);
                  const item = flatten(groups).find((it) => it.id === id) || null;
                  setSelectedGroupId(id);
                  setSelectedGroup(item);
                  setQuery((q) => ({ ...q, page: 1 }));
                }}
                onDrop={async (info) => {
                  if (!projectId) return;
                  const dragID = Number((info.dragNode as DataNode).key);
                  const dropID = Number((info.node as DataNode).key);
                  const { tree, removed } = removeNode(groups, dragID);
                  if (!removed) return;
                  const nextTree = insertNode(tree, dropID, removed, info.dropToGap ?? false, info.dropPosition > 0);
                  const normalized = normalizeSort(nextTree);
                  setGroups(normalized);
                  const saveItems = flattenForSave(normalized);
                  for (const it of saveItems) {
                    await upsertProjectServerGroup(projectId, {
                      id: it.id,
                      name: it.name,
                      category: it.category,
                      provider: it.provider,
                      parent_id: it.parent_id ?? undefined,
                      sort: it.sort,
                      status: it.status,
                    });
                  }
                  message.success("分组顺序已更新");
                  await loadGroups();
                }}
              />
            </div>
            {selectedGroup ? (
              <Space style={{ marginTop: 12 }}>
                <Button
                  size="small"
                  icon={<EditOutlined />}
                  onClick={() => {
                    if (!selectedGroup) return;
                    setEditingGroup(selectedGroup);
                    groupForm.setFieldsValue({
                      name: selectedGroup.name,
                      category: selectedGroup.category,
                      provider: selectedGroup.provider || "custom",
                    });
                    setGroupEditorOpen(true);
                  }}
                >
                  编辑/重命名
                </Button>
                <Popconfirm title="确认删除该分组？" onConfirm={() => void onDeleteGroup()}>
                  <Button size="small" danger icon={<DeleteOutlined />}>删除分组</Button>
                </Popconfirm>
              </Space>
            ) : null}
          </Card>
        </Col>
        <Col span={18} style={{ height: "100%", display: "flex" }}>
          <div style={{ width: "100%", height: "100%", display: "flex", flexDirection: "column", gap: 12 }}>
            {isCloudNode ? (
              <Card size="small" title="云账号与同步">
                <Form form={cloudForm} layout="inline" onFinish={() => void onSaveCloudAccount()}>
                  <Form.Item name="account_name" rules={[{ required: true, message: "请输入账号名称" }]}>
                    <Input placeholder="账号名称" />
                  </Form.Item>
                  <Form.Item name="region_scope">
                    <Input placeholder="region，逗号分隔（如 cn-hangzhou,cn-beijing）" style={{ width: 260 }} />
                  </Form.Item>
                  <Form.Item name="ak" rules={[{ required: true, message: "请输入 AK" }]}>
                    <Input placeholder="AccessKey ID" style={{ width: 220 }} />
                  </Form.Item>
                  <Form.Item name="sk" rules={[{ required: true, message: "请输入 SK" }]}>
                    <Input.Password placeholder="AccessKey Secret" style={{ width: 220 }} />
                  </Form.Item>
                  <Form.Item>
                    <Button type="primary" htmlType="submit" loading={cloudSubmitting}>保存云账号</Button>
                  </Form.Item>
                </Form>
                <Table
                  style={{ marginTop: 12 }}
                  rowKey="id"
                  pagination={false}
                  dataSource={cloudAccounts}
                  columns={[
                    { title: "账号", dataIndex: "account_name" },
                    { title: "厂商", dataIndex: "provider", width: 100 },
                    { title: "区域", dataIndex: "region_scope", width: 220, render: (v: string) => v || "默认" },
                    { title: "最近同步", dataIndex: "last_sync_at", width: 180, render: (v?: string | null) => (v ? formatDateTime(v) : "-") },
                    {
                      title: "最近结果",
                      width: 250,
                      render: (_: unknown, r: CloudAccountItem) => {
                        const stat = syncResultByAccount[r.id];
                        if (!stat) return "-";
                        return (
                          <Space size={4} wrap>
                            <Tag color="success">新增 {stat.added}</Tag>
                            <Tag color="processing">更新 {stat.updated}</Tag>
                            <Tag color="warning">停用 {stat.disabled}</Tag>
                            <Tag>不变 {stat.unchanged}</Tag>
                          </Space>
                        );
                      },
                    },
                    { title: "同步错误", dataIndex: "last_sync_error", ellipsis: true, render: (v?: string | null) => v || "-" },
                    { title: "操作", width: 120, render: (_: unknown, r: CloudAccountItem) => <Button icon={<SyncOutlined />} onClick={() => void onSync(r.id)}>同步</Button> },
                  ]}
                />
              </Card>
            ) : null}
            <Card
              size="small"
              style={{ flex: 1, minHeight: 0 }}
              bodyStyle={{ height: "calc(100% - 57px)", display: "flex", flexDirection: "column", minHeight: 0 }}
              title={selectedGroup ? `当前分组：${selectedGroup.name}` : "服务器列表"}
              extra={
                <Space wrap>
                  <Input.Search allowClear placeholder="搜索 name/host/tags" onSearch={(keyword) => setQuery((q) => ({ ...q, keyword, page: 1 }))} style={{ width: 220 }} />
                  <Button icon={<ReloadOutlined />} onClick={() => void loadServers()} loading={loading}>刷新</Button>
                  <Button icon={<ApiOutlined />} onClick={() => void onBatchTest()} disabled={selectedRowKeys.length === 0} loading={batchTesting}>批量测试</Button>
                  {!isCloudNode ? (
                    <>
                      <Button icon={<UploadOutlined />} onClick={() => fileInputRef.current?.click()}>导入</Button>
                      <Button icon={<DownloadOutlined />} onClick={() => projectId && downloadProjectServersImportTemplate(projectId).then((blob) => { const u = URL.createObjectURL(blob); const a = document.createElement("a"); a.href = u; a.download = "servers-import-template.xlsx"; a.click(); URL.revokeObjectURL(u); })}>模板</Button>
                      <Button icon={<DownloadOutlined />} onClick={() => projectId && exportProjectServers(projectId, { keyword: query.keyword }).then((blob) => { const u = URL.createObjectURL(blob); const a = document.createElement("a"); a.href = u; a.download = `project-${projectId}-servers.xlsx`; a.click(); URL.revokeObjectURL(u); })}>导出</Button>
                    </>
                  ) : null}
                  <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>新增</Button>
                </Space>
              }
            >
              <Select style={{ width: 280, marginBottom: 12, flexShrink: 0 }} placeholder="选择项目" value={projectId} onChange={setProjectId} options={projectOptions} />
              <input ref={fileInputRef} type="file" accept=".xlsx,.xls" style={{ display: "none" }} onChange={(e) => { const f = e.target.files?.[0]; if (f) void onImport(f); e.target.value = ""; }} />
              <Table
                style={{ flex: 1, minHeight: 0 }}
                rowKey="id"
                dataSource={list}
                loading={loading}
                scroll={{ x: "max-content", y: tableScrollY }}
                rowSelection={{ selectedRowKeys, onChange: (keys) => setSelectedRowKeys(keys as number[]) }}
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
                  { title: "名称", dataIndex: "name", width: 150 },
                  { title: "Host", dataIndex: "host", width: 180 },
                  { title: "Port", dataIndex: "port", width: 80 },
                  { title: "来源", width: 120, render: (_: unknown, r: ServerItem) => <Tag>{r.source_type === "cloud" ? `云/${r.provider || "-"}` : "自建"}</Tag> },
                  { title: "区域", dataIndex: "cloud_region", width: 100, render: (v: string) => v || "-" },
                  { title: "OS / 架构", width: 150, render: (_: unknown, r: ServerItem) => `${r.os_type || "-"} / ${r.os_arch || "-"}` },
                  { title: "启用状态", dataIndex: "status", width: 100, render: (v: number) => (v === 1 ? <Tag color="green">启用</Tag> : <Tag color="red">停用</Tag>) },
                  {
                    title: "连通状态",
                    width: 120,
                    render: (_: unknown, r: ServerItem) => {
                      if (!r.last_test_at) return <Tag>未测试</Tag>;
                      return r.last_test_error ? <Tag color="error">异常</Tag> : <Tag color="success">正常</Tag>;
                    },
                  },
                  { title: "上次测试", dataIndex: "last_test_at", width: 180, render: (v?: string | null) => (v ? formatDateTime(v) : "-") },
                  { title: "测试错误", dataIndex: "last_test_error", ellipsis: true, render: (v?: string | null) => (v ? <span title={v}>{v}</span> : "-") },
                  {
                    title: "操作",
                    width: 420,
                    render: (_: unknown, r: ServerItem) => (
                      <Space wrap>
                        <Button
                          icon={<LinkOutlined />}
                          onClick={() => {
                            if (!projectId) return;
                            navigate(`/server-console?project_id=${projectId}&server_id=${r.id}`);
                          }}
                        >
                          连接
                        </Button>
                        <Button icon={<ApiOutlined />} onClick={() => projectId && testProjectServer(projectId, r.id).then((x) => { x.ok ? message.success(x.message || "连通性 OK") : message.error(x.message || "连通性失败"); void loadServers(); })}>测试</Button>
                        <Button icon={<EditOutlined />} onClick={() => openEdit(r)}>编辑</Button>
                        <Popconfirm title="确定删除该服务器？" onConfirm={() => void onDelete(r)}><Button danger icon={<DeleteOutlined />}>删除</Button></Popconfirm>
                      </Space>
                    ),
                  },
                ]}
              />
            </Card>
          </div>
        </Col>
      </Row>

      <Modal title={editingGroup ? "编辑分组" : "新增分组"} open={groupEditorOpen} onCancel={() => { setGroupEditorOpen(false); setEditingGroup(null); }} onOk={() => void onSaveGroup()} destroyOnClose>
        <Form form={groupForm} layout="vertical" initialValues={{ category: "self_hosted", provider: "custom" }}>
          <Form.Item label="名称" name="name" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item label="类型" name="category" rules={[{ required: true }]}>
            <Select options={serverGroupCategoryOptions} />
          </Form.Item>
          <Form.Item label="厂商标识" name="provider">
            <Input placeholder="custom / alibaba / tencent / jd" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal title={current ? "编辑服务器" : "新增服务器"} open={editorOpen} onCancel={() => setEditorOpen(false)} onOk={() => void onSubmit()} confirmLoading={submitting} destroyOnClose width={720}>
        <Form layout="vertical" form={form}>
          <Form.Item name="id" hidden><Input /></Form.Item>
          <Form.Item name="project_id" hidden><Input /></Form.Item>
          <Form.Item name="group_id" hidden><Input /></Form.Item>
          <Form.Item name="status" hidden><InputNumber /></Form.Item>
          <Form.Item name="source_type" hidden><Input /></Form.Item>
          <Form.Item name="provider" hidden><Input /></Form.Item>
          <Form.Item label="名称" name="name" rules={[{ required: true }]}><Input /></Form.Item>
          <Space style={{ width: "100%" }} size={16} align="start">
            <Form.Item label="Host" name="host" rules={[{ required: true }]} style={{ flex: 1 }}><Input /></Form.Item>
            <Form.Item label="Port" name="port" style={{ width: 160 }}>
              <InputNumber min={1} max={65535} style={{ width: "100%" }} />
            </Form.Item>
            <Form.Item label="OS" name="os_type" style={{ width: 160 }}><Select options={serverOsOptions} /></Form.Item>
          </Space>
          <Form.Item
            label="从数据字典填端口"
            name="port_dict_label"
            extra="按标签选择后自动写入上方 Port；手动修改 Port 不受影响。"
          >
            <DictLabelFillSelect
              form={form}
              labelFieldName="port_dict_label"
              targetFieldName="port"
              options={serverPortDict}
              placeholder="选择字典中的端口模板"
            />
          </Form.Item>
          <Form.Item label="区域" name="cloud_region"><Input placeholder="例如：cn-hangzhou / 华东-1 / IDC-A" /></Form.Item>
          <Form.Item label="Tags（逗号分隔）" name="tags"><Input /></Form.Item>
          <Card size="small" title="SSH 凭据（可选）">
            <Space style={{ width: "100%" }} size={16} align="start">
              <Form.Item label="认证方式" name="auth_type" style={{ width: 180 }}><Select options={serverAuthOptions} /></Form.Item>
            </Space>
            <Form.Item
              label="从数据字典填用户名"
              name="username_dict_label"
              extra="按标签选择，避免下拉展示过长内容；选后写入「用户名」。手改用户名会清空此处。"
            >
              <DictLabelFillSelect
                form={form}
                labelFieldName="username_dict_label"
                targetFieldName="username"
                options={sshUserDict}
                placeholder="选择字典中的用户名模板"
              />
            </Form.Item>
            <Form.Item label="用户名" name="username">
              <Input />
            </Form.Item>
            <Form.Item noStyle shouldUpdate={(a, b) => a.auth_type !== b.auth_type}>
              {({ getFieldValue }) =>
                getFieldValue("auth_type") === "key" ? (
                  <>
                    <Form.Item label="私钥（PEM）" name="private_key"><Input.TextArea rows={6} /></Form.Item>
                    <Form.Item label="私钥口令（可选）" name="passphrase"><Input.Password /></Form.Item>
                  </>
                ) : (
                  <>
                    <Form.Item
                      label="从数据字典填密码"
                      name="password_dict_label"
                      extra="按标签选择；选后写入下方密码框。编辑时密码可留空以保留原密码；手改密码会清空此处。"
                    >
                      <DictLabelFillSelect
                        form={form}
                        labelFieldName="password_dict_label"
                        targetFieldName="password"
                        options={sshPwdDict}
                        placeholder="选择字典中的密码模板"
                      />
                    </Form.Item>
                    <Form.Item label="密码" name="password">
                      <Input.Password placeholder={current ? "留空表示保留原密码" : undefined} />
                    </Form.Item>
                  </>
                )
              }
            </Form.Item>
          </Card>
        </Form>
      </Modal>

      <Modal title={batchModal ? `批量测试结果（成功 ${batchModal.success} / 失败 ${batchModal.failed}）` : "批量测试结果"} open={!!batchModal} onCancel={() => setBatchModal(null)} footer={null} width={720}>
        {batchModal ? (
          <Table
            size="small"
            rowKey="server_id"
            pagination={false}
            dataSource={batchModal.results}
            columns={[
              { title: "ServerID", dataIndex: "server_id", width: 90 },
              { title: "结果", dataIndex: "ok", width: 90, render: (v: boolean) => (v ? <Tag color="success">OK</Tag> : <Tag color="error">FAIL</Tag>) },
              { title: "说明", dataIndex: "message", render: (v: string) => <Typography.Text ellipsis={{ tooltip: v }}>{v || "-"}</Typography.Text> },
            ]}
          />
        ) : null}
      </Modal>
    </Card>
  );
}

