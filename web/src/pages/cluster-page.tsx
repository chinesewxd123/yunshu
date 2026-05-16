import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  DeleteOutlined,
  EditOutlined,
  PlusOutlined,
  ReloadOutlined,
  SettingOutlined,
  TeamOutlined,
} from "@ant-design/icons";
import { Button, Card, Drawer, Form, Input, Modal, Popconfirm, Select, Space, Switch, Table, Tag, Tooltip, Typography, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useEffect, useMemo, useState } from "react";
import { DictLabelFillSelect } from "../components/dict-fill-select";
import { useDictOptions } from "../hooks/use-dict-options";
import { formatDateTime } from "../utils/format";
import { batchDeleteK8sClusterGrants, deleteK8sClusterGrant, listClusterAuthMatrix, type K8sAuthMatrixRow } from "../services/k8s-policies";
import { createCluster, deleteCluster, getClusterDetail, getClusterStatus, getClusters, setClusterStatus, updateCluster, type ClusterItem, type ClusterCreatePayload, type ClusterUpdatePayload } from "../services/clusters";
import { getProjects, type ProjectItem } from "../services/projects";

type ClusterQuery = {
  keyword: string;
  page: number;
  page_size: number;
};

const defaultQuery: ClusterQuery = { keyword: "", page: 1, page_size: 10 };

function phaseColor(phase: string): string {
  const p = (phase || "").toLowerCase();
  if (p === "running") return "green";
  if (p === "pending") return "orange";
  if (p === "failed") return "red";
  if (p === "succeeded") return "blue";
  return "default";
}

export function ClusterPage() {
  const kubeTplDict = useDictOptions("k8s_kubeconfig_template");
  const directCfgDict = useDictOptions("k8s_direct_config");

  const [query, setQuery] = useState<ClusterQuery>(defaultQuery);
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [statusUpdatingID, setStatusUpdatingID] = useState<number | null>(null);

  const [list, setList] = useState<ClusterItem[]>([]);
  const [total, setTotal] = useState(0);

  const [statusByID, setStatusByID] = useState<Record<number, { server_version: string; connection_state?: string; last_error?: string }>>({});

  const [modalOpen, setModalOpen] = useState(false);
  const [authOpen, setAuthOpen] = useState(false);
  const [authCluster, setAuthCluster] = useState<ClusterItem | null>(null);
  const [authRows, setAuthRows] = useState<K8sAuthMatrixRow[]>([]);
  const [authLoading, setAuthLoading] = useState(false);
  const [authSelectedKeys, setAuthSelectedKeys] = useState<React.Key[]>([]);
  const [current, setCurrent] = useState<ClusterItem | null>(null);
  const [projectOptions, setProjectOptions] = useState<ProjectItem[]>([]);
  const [form] = Form.useForm<ClusterCreatePayload &
  Partial<ClusterUpdatePayload> & {
    kubeconfig_dict_label?: string;
  }>();
  const connectionMode = Form.useWatch("connection_mode", form) || "kubeconfig";
  const selectedDirectConfigKey = Form.useWatch(["direct_config", "dict_config_key"], form);

  const directConfigKeyOptions = useMemo(
    () =>
      directCfgDict.map((it) => {
        const rawValue = String(it.value ?? "").trim();
        const usesJSONAsValue = rawValue.startsWith("{") || rawValue.startsWith("[");
        return {
          label: String(it.label),
          value: usesJSONAsValue ? String(it.label) : rawValue,
        };
      }),
    [directCfgDict],
  );

  const directTemplateMap = useMemo(() => {
    const map = new Map<string, Record<string, unknown>>();
    for (const item of directCfgDict) {
      const raw = String(item.value ?? "").trim();
      if (!(raw.startsWith("{") || raw.startsWith("["))) continue;
      try {
        const parsed = JSON.parse(raw);
        if (parsed && typeof parsed === "object") {
          map.set(String(item.label), parsed as Record<string, unknown>);
        }
      } catch {
        // ignore invalid json template values
      }
    }
    return map;
  }, [directCfgDict]);

  useEffect(() => {
    void loadClusters();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [query]);

  useEffect(() => {
    void (async () => {
      try {
        const res = await getProjects({ page: 1, page_size: 500 });
        setProjectOptions(res.list);
      } catch {
        setProjectOptions([]);
      }
    })();
  }, []);

  async function loadClusters() {
    setLoading(true);
    try {
      const result = await getClusters(query);
      setList(result.list);
      setTotal(result.total);
    } finally {
      setLoading(false);
    }
  }

  const enabledClusters = useMemo(() => list.filter((c) => c.status === 1), [list]);

  useEffect(() => {
    let cancelled = false;
    async function loadStatuses() {
      const ids = enabledClusters.map((c) => c.id);
      if (ids.length === 0) return;

      // only fetch statuses we don't have yet
      const missing = ids.filter((id) => !statusByID[id]);
      if (missing.length === 0) return;

      const results = await Promise.allSettled(
        missing.map(async (id) => {
          const st = await getClusterStatus(id);
          return {
            id,
            status: {
              server_version: st.server_version || "-",
              connection_state: st.connection_state,
              last_error: st.last_error,
            },
          };
        }),
      );

      if (cancelled) return;
      setStatusByID((prev) => {
        const next = { ...prev };
        for (const r of results) {
          if (r.status === "fulfilled") {
            next[r.value.id] = r.value.status;
          } else {
            // keep existing value if any; otherwise fallback
            // eslint-disable-next-line @typescript-eslint/no-unused-expressions
            void 0;
          }
        }
        return next;
      });
    }
    void loadStatuses();
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [enabledClusters]);

  function openCreate() {
    setCurrent(null);
    setModalOpen(true);
  }

  async function openEdit(record: ClusterItem) {
    setCurrent(record);
    try {
      const detail = await getClusterDetail(record.id);
      setCurrent(detail);
    } catch {
      // Keep fallback record when detail request fails.
    }
    setModalOpen(true);
  }

  useEffect(() => {
    if (!modalOpen) return;
    // With Modal `destroyOnClose`, setFieldsValue may run before Form items mount.
    // Fill values after the modal becomes visible.
    if (current) {
      form.resetFields();
      form.setFieldsValue({
        name: current.name,
        owning_project_id:
          current.owning_project_id !== undefined && current.owning_project_id !== null && current.owning_project_id > 0
            ? current.owning_project_id
            : undefined,
        connection_mode: current.connection_mode || "kubeconfig",
        kubeconfig: current.kubeconfig || "",
        kubeconfig_dict_label: undefined,
        // 已配置时 kubeconfig 不回传，留空表示不修改
        direct_config: {
          server: current.direct_config?.server || "",
          dict_config_key: current.direct_config?.dict_config_key || undefined,
          token: current.direct_config?.token || "",
          username: current.direct_config?.username || "",
          password: current.direct_config?.password || "",
          client_cert_data: current.direct_config?.client_cert_data || "",
          client_key_data: current.direct_config?.client_key_data || "",
          ca_data: current.direct_config?.ca_data || "",
          insecure_skip_tls_verify: current.direct_config?.insecure_skip_tls_verify || false,
        },
      });
    } else {
      form.resetFields();
      form.setFieldsValue({
        name: "",
        owning_project_id: undefined,
        connection_mode: "kubeconfig",
        kubeconfig: "",
        kubeconfig_dict_label: undefined,
        direct_config: {
          server: "",
          dict_config_key: undefined,
          token: "",
          username: "",
          password: "",
          client_cert_data: "",
          client_key_data: "",
          ca_data: "",
          insecure_skip_tls_verify: false,
        },
      });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [modalOpen, current]);

  useEffect(() => {
    if (connectionMode !== "direct") return;
    const key = String(selectedDirectConfigKey || "").trim();
    if (!key) return;
    const template = directTemplateMap.get(key);
    if (!template) return;
    const patch: Record<string, unknown> = {};
    const server = String(template.server || "").trim();
    if (server) patch.server = server;
    const token = String(template.token || "").trim();
    if (token) patch.token = token;
    const username = String(template.username || "").trim();
    if (username) patch.username = username;
    const password = String(template.password || "").trim();
    if (password) patch.password = password;
    const client_cert_data = String(template.client_cert_data || "").trim();
    if (client_cert_data) patch.client_cert_data = client_cert_data;
    const client_key_data = String(template.client_key_data || "").trim();
    if (client_key_data) patch.client_key_data = client_key_data;
    const ca_data = String(template.ca_data || "").trim();
    if (ca_data) patch.ca_data = ca_data;
    if (typeof template.insecure_skip_tls_verify === "boolean") {
      patch.insecure_skip_tls_verify = template.insecure_skip_tls_verify;
    }
    if (Object.keys(patch).length === 0) return;
    const prev = (form.getFieldValue(["direct_config"]) || {}) as Record<string, unknown>;
    form.setFieldValue(["direct_config"], { ...prev, ...patch });
  }, [connectionMode, directTemplateMap, form, selectedDirectConfigKey]);

  async function handleSubmit() {
    const values = await form.validateFields();
    const name = (values.name || "").trim();
    if (!name) return;
    const connection_mode = values.connection_mode || "kubeconfig";

    const payload: ClusterCreatePayload & ClusterUpdatePayload = { name, connection_mode };
    const kubeconfig = (values.kubeconfig || "").trim();
    const isCreate = !current;

    if (connection_mode === "kubeconfig") {
      if (isCreate && !kubeconfig) {
        message.error("请填写 kubeconfig（创建集群必填）");
        return;
      }

      if (kubeconfig) {
        payload.kubeconfig = kubeconfig;
      } else {
        // Update without kubeconfig
        delete (payload as { kubeconfig?: string }).kubeconfig;
      }
    } else {
      const direct = values.direct_config || {};
      const server = (direct.server || "").trim();
      const dictConfigKey = (direct.dict_config_key || "").trim();
      if (!server && !dictConfigKey) {
        message.error("请填写 API Server 地址或选择直连模板");
        return;
      }
      payload.direct_config = {
        server: server || undefined,
        dict_config_key: dictConfigKey || undefined,
        token: (direct.token || "").trim() || undefined,
        username: (direct.username || "").trim() || undefined,
        password: (direct.password || "").trim() || undefined,
        client_cert_data: (direct.client_cert_data || "").trim() || undefined,
        client_key_data: (direct.client_key_data || "").trim() || undefined,
        ca_data: (direct.ca_data || "").trim() || undefined,
        insecure_skip_tls_verify: Boolean(direct.insecure_skip_tls_verify),
      };
      delete (payload as { kubeconfig?: string }).kubeconfig;
    }

    const ownPid = values.owning_project_id;
    if (current) {
      if (ownPid !== undefined && ownPid !== null && Number(ownPid) > 0) {
        (payload as ClusterUpdatePayload).owning_project_id = Number(ownPid);
      } else if (current.owning_project_id) {
        (payload as ClusterUpdatePayload).owning_project_id = 0;
      }
    } else if (ownPid !== undefined && ownPid !== null && Number(ownPid) > 0) {
      (payload as ClusterCreatePayload).owning_project_id = Number(ownPid);
    }

    setSubmitting(true);
    try {
      if (current) {
        await updateCluster(current.id, payload as ClusterUpdatePayload);
        message.success("集群已更新");
      } else {
        await createCluster(payload as ClusterCreatePayload);
        message.success("集群已创建");
      }
      setModalOpen(false);
      await loadClusters();
    } finally {
      setSubmitting(false);
    }
  }

  async function handleDelete(record: ClusterItem) {
    try {
      await deleteCluster(record.id);
      message.success("集群已删除");
      setStatusByID((prev) => {
        if (!prev[record.id]) return prev;
        const next = { ...prev };
        delete next[record.id];
        return next;
      });
      void loadClusters();
    } catch (error) {
      const msg = error instanceof Error ? error.message : "删除失败";
      message.error(msg || "删除失败");
    }
  }

  async function handleToggleStatus(record: ClusterItem) {
    const nextStatus: 0 | 1 = record.status === 1 ? 0 : 1;
    setStatusUpdatingID(record.id);
    try {
      await setClusterStatus(record.id, nextStatus);
      message.success(nextStatus === 1 ? "已启用集群" : "已停用集群");
      await loadClusters();
      if (nextStatus === 0) {
        setStatusByID((prev) => {
          if (!prev[record.id]) return prev;
          const next = { ...prev };
          delete next[record.id];
          return next;
        });
      }
    } catch (error) {
      const msg = error instanceof Error ? error.message : "操作失败";
      message.error(msg || "操作失败");
    } finally {
      setStatusUpdatingID(null);
    }
  }

  async function openAuthDrawer(record: ClusterItem) {
    setAuthCluster(record);
    setAuthOpen(true);
    setAuthSelectedKeys([]);
    setAuthLoading(true);
    try {
      const res = await listClusterAuthMatrix(record.id);
      setAuthRows(res.list ?? []);
    } catch (e) {
      message.error(e instanceof Error ? e.message : "加载已授权列表失败");
      setAuthRows([]);
    } finally {
      setAuthLoading(false);
    }
  }

  async function reloadAuthMatrix() {
    if (!authCluster) return;
    setAuthLoading(true);
    try {
      const res = await listClusterAuthMatrix(authCluster.id);
      setAuthRows(res.list ?? []);
      setAuthSelectedKeys([]);
    } finally {
      setAuthLoading(false);
    }
  }

  async function handleBatchRevokeGrants() {
    const idSet = new Set<number>();
    for (const k of authSelectedKeys) {
      const row = authRows.find((r) => r.row_key === k);
      if (row && row.grant_id > 0) idSet.add(row.grant_id);
    }
    const ids = [...idSet];
    if (ids.length === 0) {
      message.warning("请选择要撤销的授权行");
      return;
    }
    try {
      const res = await batchDeleteK8sClusterGrants(ids);
      message.success(`已撤销 ${res.deleted} 条集群档位授权`);
      await reloadAuthMatrix();
    } catch (e) {
      message.error(e instanceof Error ? e.message : "批量撤销失败");
    }
  }

  async function handleRevokeOneGrant(grantId: number) {
    try {
      await deleteK8sClusterGrant(grantId);
      message.success("已撤销");
      await reloadAuthMatrix();
    } catch (e) {
      message.error(e instanceof Error ? e.message : "撤销失败");
    }
  }

  async function handleConnectTest(record: ClusterItem) {
    try {
      const st = await getClusterStatus(record.id);
      setStatusByID((prev) => ({
        ...prev,
        [record.id]: {
          server_version: st.server_version || "-",
          connection_state: st.connection_state,
          last_error: st.last_error,
        },
      }));
      if (st.connection_state === "disabled") {
        message.info("集群已停用，未进行连通性检查");
        return;
      }
      if (st.server_version) message.success(`连接成功，K8s 版本：${st.server_version}`);
      else if (st.last_error) message.error(st.last_error || "连接测试失败");
      else message.warning("已请求状态，但未获取到版本信息");
    } catch (error) {
      const msg = error instanceof Error ? error.message : "连接测试失败";
      message.error(msg || "连接测试失败");
    }
  }

  const authColumns: ColumnsType<K8sAuthMatrixRow> = [
    { title: "用户名", dataIndex: "username", width: 140, render: (v: string, r) => (v === "-" ? <span className="inline-muted">{r.nickname || v}</span> : v) },
    {
      title: "集群",
      key: "cluster",
      width: 200,
      render: (_: unknown, r) => (
        <Space direction="vertical" size={0}>
          <Typography.Text>{r.cluster_name}</Typography.Text>
          {r.grant_scope_all ? <Tag color="blue">含「全部集群」档</Tag> : null}
        </Space>
      ),
    },
    { title: "档位", dataIndex: "preset_label", width: 120 },
    {
      title: "限制命名空间",
      dataIndex: "allow_namespaces",
      ellipsis: true,
      render: (v: string) => (v ? <Typography.Text ellipsis={{ tooltip: v }}>{v}</Typography.Text> : <span className="inline-muted">未限制（白名单未配置）</span>),
    },
    { title: "授权主体", dataIndex: "principal_show", width: 200, ellipsis: true },
    { title: "来源", dataIndex: "via", width: 130 },
    {
      title: "操作",
      key: "revoke",
      width: 100,
      fixed: "right",
      render: (_: unknown, r: K8sAuthMatrixRow) =>
        r.grant_id > 0 ? (
          <Popconfirm title="确认撤销该条集群档位授权？（同一档位多行会一并失效）" onConfirm={() => void handleRevokeOneGrant(r.grant_id)}>
            <Button type="link" size="small" danger>
              撤销
            </Button>
          </Popconfirm>
        ) : (
          <span className="inline-muted">-</span>
        ),
    },
  ];

  function renderConnection(record: ClusterItem) {
    if (record.status !== 1) return <Tag>disabled</Tag>;
    const st = statusByID[record.id];
    const state = (st?.connection_state || "unknown").toLowerCase();
    const color =
      state === "ready" ? "success" :
      state === "connecting" ? "processing" :
      state === "degraded" ? "error" :
      state === "disabled" ? "default" :
      "default";
    const label = st?.connection_state || "unknown";
    const err = (st?.last_error || "").trim();
    if (!err) return <Tag color={color}>{label}</Tag>;
    return (
      <Tooltip title={err}>
        <Tag color={color}>{label}</Tag>
      </Tooltip>
    );
  }

  return (
    <Card className="table-card">
      <Space direction="vertical" size={12} style={{ width: "100%" }}>
        <div className="toolbar">
          <Input.Search
            allowClear
            placeholder="搜索集群名称"
            style={{ width: 280 }}
            onSearch={(keyword) => setQuery((prev) => ({ ...prev, keyword, page: 1 }))}
          />
          <div className="toolbar__actions">
            <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
              新建集群
            </Button>
            <Button icon={<ReloadOutlined />} onClick={() => void loadClusters()}>
              刷新
            </Button>
          </div>
        </div>

        <Table
          rowKey="id"
          loading={loading}
          dataSource={list}
          size="small"
          pagination={{
            current: query.page,
            pageSize: query.page_size,
            total,
            showSizeChanger: true,
            pageSizeOptions: [10, 20, 50, 100],
            showQuickJumper: true,
            onChange: (page, pageSize) => setQuery((prev) => ({ ...prev, page, page_size: pageSize })),
          }}
          scroll={{ x: "max-content" }}
          columns={[
            { title: "ID", dataIndex: "id", width: 90 },
            { title: "集群名称", dataIndex: "name" },
            {
              title: "归属项目",
              key: "owning_project_id",
              width: 160,
              ellipsis: true,
              render: (_: unknown, record: ClusterItem) => {
                const pid = record.owning_project_id;
                if (!pid) return <span className="inline-muted">—</span>;
                const pn = projectOptions.find((p) => p.id === pid)?.name;
                return (
                  <Typography.Text ellipsis={{ tooltip: true }}>
                    {pn ?? `项目 #${pid}`}
                  </Typography.Text>
                );
              },
            },
            {
              title: "K8s 版本",
              key: "k8s_version",
              width: 140,
              render: (_: unknown, record: ClusterItem) => {
                if (record.status !== 1) return <span className="inline-muted">-</span>;
                const v = statusByID[record.id]?.server_version;
                return v ? v : <span className="inline-muted">获取中…</span>;
              },
            },
            {
              title: "连接状态",
              key: "conn_state",
              width: 140,
              render: (_: unknown, record: ClusterItem) => renderConnection(record),
            },
            {
              title: "状态",
              dataIndex: "status",
              width: 110,
              render: (v: number) => (v === 1 ? <Tag color="success">启用</Tag> : <Tag>停用</Tag>),
            },
            { title: "创建时间", dataIndex: "created_at", render: (v: string) => formatDateTime(v) },
            {
              title: "操作",
              key: "action",
              width: 420,
              render: (_: unknown, record: ClusterItem) => (
                <Space>
                  <Button type="link" icon={<TeamOutlined />} onClick={() => void openAuthDrawer(record)}>
                    已授权
                  </Button>
                  <Button
                    type="link"
                    icon={<SettingOutlined />}
                    onClick={() => {
                      void handleConnectTest(record);
                    }}
                  >
                    连接测试
                  </Button>

                  <Popconfirm
                    title={record.status === 1 ? "确认停用该集群吗？停用后将禁止访问该集群。" : "确认启用该集群吗？"}
                    onConfirm={() => {
                      void handleToggleStatus(record);
                    }}
                  >
                    <Button
                      type="link"
                      danger={record.status === 1}
                      loading={statusUpdatingID === record.id}
                      icon={record.status === 1 ? <CloseCircleOutlined /> : <CheckCircleOutlined />}
                    >
                      {record.status === 1 ? "停用" : "启用"}
                    </Button>
                  </Popconfirm>

                  <Button type="link" icon={<EditOutlined />} onClick={() => void openEdit(record)}>
                    编辑
                  </Button>
                  <Popconfirm title="确认删除该集群吗？" onConfirm={() => void handleDelete(record)}>
                    <Button type="link" danger icon={<DeleteOutlined />}>
                      删除
                    </Button>
                  </Popconfirm>
                </Space>
              ),
            },
          ]}
        />
      </Space>

      <Drawer
        title={authCluster ? `已授权用户 — ${authCluster.name}` : "已授权用户"}
        open={authOpen}
        onClose={() => {
          setAuthOpen(false);
          setAuthCluster(null);
          setAuthRows([]);
          setAuthSelectedKeys([]);
        }}
        width={980}
        destroyOnClose
      >
        <Space style={{ marginBottom: 12 }} wrap>
          <Typography.Text type="secondary">共 {authRows.length} 条（按用户展开；撤销档位后关联行会消失）</Typography.Text>
          <Button icon={<ReloadOutlined />} onClick={() => void reloadAuthMatrix()} disabled={!authCluster}>
            刷新
          </Button>
          <Popconfirm title="确认批量撤销选中行对应的集群档位？" onConfirm={() => void handleBatchRevokeGrants()}>
            <Button danger disabled={authSelectedKeys.length === 0}>
              批量撤销
            </Button>
          </Popconfirm>
        </Space>
        <Table<K8sAuthMatrixRow>
          rowKey="row_key"
          size="small"
          loading={authLoading}
          dataSource={authRows}
          columns={authColumns}
          scroll={{ x: 900 }}
          rowSelection={{
            selectedRowKeys: authSelectedKeys,
            onChange: (keys) => setAuthSelectedKeys(keys),
            getCheckboxProps: (r) => ({ disabled: !r.grant_id }),
          }}
          pagination={{ pageSize: 10, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
        />
      </Drawer>

      <Modal
        title={current ? "编辑集群" : "新建集群"}
        open={modalOpen}
        onCancel={() => setModalOpen(false)}
        onOk={() => void handleSubmit()}
        confirmLoading={submitting}
        destroyOnClose
        width={720}
      >
        <Form form={form} layout="vertical" initialValues={{ name: "", connection_mode: "kubeconfig", kubeconfig: "" }}>
          <Form.Item label="集群名称" name="name" rules={[{ required: true, message: "请输入集群名称" }]}>
            <Input placeholder="例如：prod-k8s" />
          </Form.Item>

          <Form.Item
            label="归属项目"
            name="owning_project_id"
            extra="可选；指定后仅归属项目成员可按策略访问该集群（与后端隔离规则一致）。"
          >
            <Select
              allowClear
              showSearch
              optionFilterProp="label"
              placeholder="不选则不限定项目（全局可见性由后端策略决定）"
              options={projectOptions.map((p) => ({ value: p.id, label: `${p.name} (${p.code})` }))}
            />
          </Form.Item>

          <Form.Item label="连接模式" name="connection_mode" rules={[{ required: true, message: "请选择连接模式" }]}>
            <Select
              options={[
                { label: "kubeconfig", value: "kubeconfig" },
                { label: "direct（直连）", value: "direct" },
              ]}
            />
          </Form.Item>

          {connectionMode === "kubeconfig" ? (
            <>
              <Form.Item
                label="从数据字典插入 kubeconfig 模板"
                name="kubeconfig_dict_label"
                extra="与下方 kubeconfig 内容一致时会自动选中对应模板；修改 yaml 后若不一致会清空此处。"
              >
                <DictLabelFillSelect
                  form={form}
                  labelFieldName="kubeconfig_dict_label"
                  targetFieldName="kubeconfig"
                  options={kubeTplDict}
                  placeholder="选择模板后填入下方文本框（按标签选，避免下拉展示整段 yaml）"
                  style={{ maxWidth: 480 }}
                />
              </Form.Item>

              <Form.Item
                label="kubeconfig"
                name="kubeconfig"
                rules={
                  current
                    ? []
                    : [{ required: true, message: "请填写 kubeconfig" }]
                }
                extra={
                  current?.kubeconfig_configured && !(current.kubeconfig || "").trim()
                    ? "已配置 kubeconfig（出于安全不回显）；留空表示不修改，重新粘贴可更新"
                    : current
                      ? "已预填当前 kubeconfig，可直接修改后保存"
                      : "用于 Kom SDK 注册并访问集群"
                }
              >
                <Input.TextArea
                  rows={8}
                  placeholder="粘贴 kubeconfig 内容（yaml）"
                  style={{ fontFamily: "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, Liberation Mono, Courier New, monospace" }}
                />
              </Form.Item>
            </>
          ) : (
            <>
              <Form.Item
                label="直连配置模板（数据字典）"
                name={["direct_config", "dict_config_key"]}
                extra="字典类型为 k8s_direct_config，value 应为配置键。"
              >
                <Select
                  allowClear
                  showSearch
                  optionFilterProp="label"
                  options={directConfigKeyOptions}
                  placeholder="可选：先选择模板键，再按需覆盖下面字段"
                />
              </Form.Item>

              <Form.Item
                label="API Server 地址"
                name={["direct_config", "server"]}
                rules={[
                  ({ getFieldValue }) => ({
                    validator(_, value) {
                      const server = String(value || "").trim();
                      const dictKey = String(getFieldValue(["direct_config", "dict_config_key"]) || "").trim();
                      if (!server && !dictKey) {
                        return Promise.reject(new Error("请输入 API Server 地址或选择直连模板"));
                      }
                      if (!server) {
                        return Promise.resolve();
                      }
                      try {
                        const u = new URL(server);
                        if (!u.protocol || !u.host) {
                          return Promise.reject(new Error("请输入合法 URL，例如 https://10.0.0.1:6443"));
                        }
                        return Promise.resolve();
                      } catch {
                        return Promise.reject(new Error("请输入合法 URL，例如 https://10.0.0.1:6443"));
                      }
                    },
                  }),
                ]}
              >
                <Input placeholder="https://x.x.x.x:6443" />
              </Form.Item>

              <Form.Item label="Token" name={["direct_config", "token"]}>
                <Input.Password placeholder="Service Account Token（可选）" autoComplete="off" />
              </Form.Item>

              <Form.Item label="用户名" name={["direct_config", "username"]}>
                <Input placeholder="Basic Auth 用户名（可选）" />
              </Form.Item>
              <Form.Item label="密码" name={["direct_config", "password"]}>
                <Input.Password placeholder="Basic Auth 密码（可选）" autoComplete="off" />
              </Form.Item>

              <Form.Item label="客户端证书（base64）" name={["direct_config", "client_cert_data"]}>
                <Input.TextArea rows={3} placeholder="client_cert_data（可选）" />
              </Form.Item>
              <Form.Item label="客户端私钥（base64）" name={["direct_config", "client_key_data"]}>
                <Input.TextArea rows={3} placeholder="client_key_data（可选）" />
              </Form.Item>
              <Form.Item label="CA 证书（base64）" name={["direct_config", "ca_data"]}>
                <Input.TextArea rows={3} placeholder="ca_data（可选）" />
              </Form.Item>

              <Form.Item label="跳过 TLS 验证" name={["direct_config", "insecure_skip_tls_verify"]} valuePropName="checked">
                <Switch />
              </Form.Item>
            </>
          )}
        </Form>
      </Modal>
    </Card>
  );
}

// keep exported icons reference used elsewhere

