import { DeleteOutlined, GiftOutlined, ReloadOutlined, ThunderboltOutlined } from "@ant-design/icons";
import {
  Alert,
  Button,
  Card,
  Divider,
  Empty,
  Form,
  Input,
  Popconfirm,
  Select,
  Space,
  Table,
  Tag,
  Typography,
  message,
} from "antd";
import { useEffect, useMemo, useState } from "react";
import { getClusters } from "../services/clusters";
import {
  createK8sNamespaceDenyRule,
  deleteK8sNamespaceDenyRule,
  listK8sNamespaceDenyRules,
  type K8sNamespaceDenyRule,
} from "../services/k8s-namespace-deny";
import {
  grantK8sScopedPolicies,
  grantK8sScopedPoliciesPreset,
  listK8sPoliciesByRole,
  listK8sPolicyActions,
  listK8sPolicyPaths,
} from "../services/k8s-policies";
import { listNamespaces } from "../services/namespaces";
import { getRoleOptions } from "../services/roles";
import type { RoleItem } from "../types/api";

export function K8sScopedPoliciesPage() {
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [presetSubmitting, setPresetSubmitting] = useState(false);
  const [roles, setRoles] = useState<RoleItem[]>([]);
  const [selectedRoleId, setSelectedRoleId] = useState<number>();
  const [k8sActions, setK8sActions] = useState<{ code: string; name: string; description: string }[]>([]);
  const [k8sPaths, setK8sPaths] = useState<string[]>([]);
  const [clusterOptions, setClusterOptions] = useState<{ id: number; name: string }[]>([]);
  const [namespaceOptions, setNamespaceOptions] = useState<string[]>([]);
  const [namespaceLoading, setNamespaceLoading] = useState(false);
  const [k8sPolicies, setK8sPolicies] = useState<
    { cluster_id: string; namespace: string; path: string; action: string; resource: string }[]
  >([]);
  const [denyRules, setDenyRules] = useState<K8sNamespaceDenyRule[]>([]);
  const [denyLoading, setDenyLoading] = useState(false);
  const [denySubmitting, setDenySubmitting] = useState(false);
  const [form] = Form.useForm<{
    cluster_ids: number[];
    namespaces: string[];
    actions: string[];
    paths: string[];
  }>();
  const [presetForm] = Form.useForm<{
    cluster_ids: number[];
    namespaces: string[];
    preset: "readonly" | "readonly_exec" | "admin";
    deny_namespaces?: string[];
  }>();
  const [denyForm] = Form.useForm<{ cluster_id?: number; namespace?: string }>();

  const selectedRole = useMemo(() => roles.find((r) => r.id === selectedRoleId) ?? null, [roles, selectedRoleId]);

  useEffect(() => {
    void bootstrap();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function bootstrap(preferredRoleId?: number) {
    setLoading(true);
    try {
      const [roleData, actionData, pathData, clusterData] = await Promise.all([
        getRoleOptions(),
        listK8sPolicyActions(),
        listK8sPolicyPaths(),
        getClusters({ page: 1, page_size: 200 }),
      ]);
      setRoles(roleData.list);
      setK8sActions(actionData.list);
      setK8sPaths(pathData.list);
      setClusterOptions(clusterData.list.map((c) => ({ id: c.id, name: c.name })));

      const nextRoleId = preferredRoleId ?? (roleData.list[0]?.id ?? undefined);
      setSelectedRoleId(nextRoleId);
      if (nextRoleId) {
        await refresh(nextRoleId);
        const rc = roleData.list.find((r) => r.id === nextRoleId)?.code ?? "";
        await refreshDenyRules(rc);
      } else {
        setK8sPolicies([]);
        setDenyRules([]);
      }
    } finally {
      setLoading(false);
    }
  }

  async function refresh(roleId: number) {
    const result = await listK8sPoliciesByRole(roleId);
    setK8sPolicies(result.list);
  }

  async function refreshDenyRules(roleCode: string) {
    if (!roleCode.trim()) {
      setDenyRules([]);
      return;
    }
    setDenyLoading(true);
    try {
      const data = await listK8sNamespaceDenyRules({ role_code: roleCode });
      setDenyRules(data.list ?? []);
    } catch {
      setDenyRules([]);
    } finally {
      setDenyLoading(false);
    }
  }

  async function loadNamespacesByClusters(clusterIds?: number[]) {
    const ids = (clusterIds ?? []).filter((v) => Number(v) > 0);
    if (ids.length !== 1) {
      setNamespaceOptions([]);
      return;
    }
    setNamespaceLoading(true);
    try {
      const ns = await listNamespaces(ids[0]);
      const names = Array.from(new Set((ns ?? []).map((it) => String(it.name ?? "").trim()).filter(Boolean))).sort();
      setNamespaceOptions(names);
    } catch {
      setNamespaceOptions([]);
    } finally {
      setNamespaceLoading(false);
    }
  }

  return (
    <div>
      <Card
        className="table-card"
        title="Kubernetes 三元授权（集群 + 命名空间 + 动作）"
        loading={loading}
        extra={
          <Space>
            <Button icon={<ReloadOutlined />} onClick={() => void bootstrap(selectedRoleId)}>
              刷新
            </Button>
          </Space>
        }
      >
        <Space direction="vertical" size={12} style={{ width: "100%" }}>
          <Space wrap style={{ width: "100%", justifyContent: "space-between" }}>
            <Space wrap>
              <Typography.Text className="inline-muted">角色模板</Typography.Text>
              <Select
                placeholder="请选择角色模板"
                style={{ minWidth: 320 }}
                value={selectedRoleId}
                onChange={(v) => {
                  setSelectedRoleId(v);
                  void refresh(v);
                  const rc = roles.find((r) => r.id === v)?.code ?? "";
                  void refreshDenyRules(rc);
                }}
                options={roles.map((role) => ({ label: `${role.name} (${role.code})`, value: role.id }))}
              />
            </Space>
          </Space>

          {selectedRole ? (
            <>
              <Typography.Text className="inline-muted">
                该页面下发的是 Casbin policy，但资源被编码为 <Tag>k8s:cluster:&lt;id&gt;:ns:&lt;ns&gt;:&lt;path&gt;</Tag>，
                用于约束高危动作（例如 exec/删除/scale）。
              </Typography.Text>

              <Alert
                type="info"
                showIcon
                style={{ width: "100%" }}
                message="档位下发（融合 k8m：只读 / 只读+Exec / 集群管理）"
                description={
                  <span>
                    按预设批量生成「路径 + 动作码」配对策略（非路径×动作全组合）。命名空间黑名单可选：须选择具体集群 ID（勿仅用「全部集群」），否则无法写入 deny 规则。
                  </span>
                }
              />

              <Form
                form={presetForm}
                layout="vertical"
                initialValues={{
                  cluster_ids: [],
                  namespaces: ["*"],
                  preset: "readonly" as const,
                  deny_namespaces: [],
                }}
                style={{ maxWidth: 960 }}
              >
                <Space wrap style={{ width: "100%", alignItems: "flex-start" }}>
                  <Form.Item label="档位" name="preset" rules={[{ required: true, message: "请选择档位" }]} style={{ minWidth: 240 }}>
                    <Select
                      style={{ minWidth: 220 }}
                      options={[
                        { value: "readonly", label: "只读（控制台资源 GET）" },
                        { value: "readonly_exec", label: "只读 + Pod Exec" },
                        { value: "admin", label: "集群管理（变更类 + 读）" },
                      ]}
                    />
                  </Form.Item>
                  <Form.Item label="集群" name="cluster_ids" style={{ minWidth: 260 }}>
                    <Select
                      mode="multiple"
                      allowClear
                      style={{ minWidth: 260 }}
                      placeholder="不选=全部（*）"
                      options={clusterOptions.map((c) => ({ label: `${c.name} (#${c.id})`, value: c.id }))}
                      onChange={(v) => {
                        const ids = Array.isArray(v) ? v : [];
                        void loadNamespacesByClusters(ids);
                      }}
                    />
                  </Form.Item>
                  <Form.Item label="命名空间" name="namespaces" style={{ minWidth: 260 }}>
                    <Select
                      mode="multiple"
                      allowClear
                      loading={namespaceLoading}
                      style={{ minWidth: 260 }}
                      placeholder='默认 "*"'
                      options={[
                        { label: "*（全部命名空间）", value: "*" },
                        ...namespaceOptions.map((name) => ({ label: name, value: name })),
                      ]}
                    />
                  </Form.Item>
                  <Form.Item
                    label="同步命名空间黑名单（可选）"
                    name="deny_namespaces"
                    tooltip="须在上栏选择具体集群；每条策略会对所选每个集群写入禁止访问该命名空间"
                  >
                    <Select mode="tags" style={{ minWidth: 320 }} placeholder="输入后回车添加，例如 kube-system" tokenSeparators={[","]} />
                  </Form.Item>
                  <Form.Item label=" ">
                    <Button
                      type="primary"
                      ghost
                      icon={<GiftOutlined />}
                      loading={presetSubmitting}
                      onClick={() => {
                        if (!selectedRoleId) return;
                        void (async () => {
                          const roleId = selectedRoleId;
                          const values = await presetForm.validateFields();
                          setPresetSubmitting(true);
                          try {
                            const denyRaw = values.deny_namespaces ?? [];
                            const denyList = (Array.isArray(denyRaw) ? denyRaw : []).map((s) => String(s).trim()).filter(Boolean);
                            const resp = await grantK8sScopedPoliciesPreset({
                              role_id: roleId,
                              cluster_ids: values.cluster_ids ?? [],
                              namespaces: (values.namespaces ?? []).filter((v) => String(v).trim() !== ""),
                              preset: values.preset,
                              deny_namespaces: denyList.length ? denyList : undefined,
                            });
                            message.success(
                              `档位下发完成：新增策略 ${resp.added} 条（跳过 ${resp.skipped}）；黑名单新增 ${resp.deny_rules_added}（跳过 ${resp.deny_rules_skipped}）`,
                            );
                            await refresh(roleId);
                            await refreshDenyRules(selectedRole.code);
                          } finally {
                            setPresetSubmitting(false);
                          }
                        })();
                      }}
                    >
                      按档位下发
                    </Button>
                  </Form.Item>
                </Space>
              </Form>

              <Form
                form={form}
                layout="inline"
                initialValues={{ cluster_ids: [], namespaces: ["*"], actions: [], paths: [] }}
                style={{ rowGap: 12 }}
              >
                <Form.Item label="集群" name="cluster_ids">
                  <Select
                    mode="multiple"
                    allowClear
                    style={{ minWidth: 260 }}
                    placeholder="不选=全部（*）"
                    options={clusterOptions.map((c) => ({ label: `${c.name} (#${c.id})`, value: c.id }))}
                    onChange={(v) => {
                      const ids = Array.isArray(v) ? v : [];
                      void loadNamespacesByClusters(ids);
                    }}
                  />
                </Form.Item>
                <Form.Item label="命名空间" name="namespaces">
                  <Select
                    mode="multiple"
                    allowClear
                    loading={namespaceLoading}
                    style={{ minWidth: 260 }}
                    placeholder='默认 "*"；选中单个集群可下拉选具体命名空间'
                    options={[
                      { label: "*（全部命名空间）", value: "*" },
                      ...namespaceOptions.map((name) => ({ label: name, value: name })),
                    ]}
                  />
                </Form.Item>
                <Form.Item label="资源路径" name="paths" rules={[{ required: true, message: "请选择资源路径" }]}>
                  <Select
                    mode="multiple"
                    allowClear
                    style={{ minWidth: 380 }}
                    placeholder="选择需要管控的 API 路径"
                    options={k8sPaths.map((p) => ({ label: p, value: p }))}
                  />
                </Form.Item>
                <Form.Item label="动作码" name="actions" rules={[{ required: true, message: "请选择动作码" }]}>
                  <Select
                    mode="multiple"
                    allowClear
                    style={{ minWidth: 420 }}
                    placeholder="例如 pods/exec、pods/delete、deployments/scale"
                    options={k8sActions.map((a) => ({ label: `${a.code}（${a.name}）`, value: a.code }))}
                    optionRender={(option) => {
                      const item = k8sActions.find((a) => a.code === option.value);
                      return (
                        <Space direction="vertical" size={0}>
                          <Typography.Text>{String(option.label)}</Typography.Text>
                          <Typography.Text className="inline-muted">{item?.description ?? "-"}</Typography.Text>
                        </Space>
                      );
                    }}
                  />
                </Form.Item>
                <Form.Item>
                  <Button
                    type="primary"
                    icon={<ThunderboltOutlined />}
                    loading={submitting}
                    onClick={() => {
                      if (!selectedRoleId) return;
                      void (async () => {
                      const roleId = selectedRoleId;
                        const values = await form.validateFields();
                        setSubmitting(true);
                        try {
                          const resp = await grantK8sScopedPolicies({
                          role_id: roleId,
                            cluster_ids: values.cluster_ids ?? [],
                            namespaces: (values.namespaces ?? []).filter((v) => String(v).trim() !== ""),
                            actions: values.actions ?? [],
                            paths: values.paths ?? [],
                          });
                          message.success(`已下发三元策略：新增 ${resp.added} 条，跳过 ${resp.skipped} 条`);
                        await refresh(roleId);
                          if (resp.added > 0) {
                            const latest = await listK8sPoliciesByRole(roleId);
                            setK8sPolicies(latest.list);
                          }
                        } finally {
                          setSubmitting(false);
                        }
                      })();
                    }}
                  >
                    快捷下发
                  </Button>
                </Form.Item>
              </Form>

              <Divider style={{ margin: "8px 0" }} />
              <Typography.Text strong>当前角色的三元策略（K8s Scope）</Typography.Text>
              <Table
                rowKey={(record) => `${record.resource}::${record.action}`}
                dataSource={k8sPolicies}
                pagination={{ pageSize: 10, showSizeChanger: true, pageSizeOptions: [10, 20, 50, 100], showQuickJumper: true }}
                size="small"
                scroll={{ x: "max-content" }}
                columns={[
                  { title: "集群", dataIndex: "cluster_id", render: (v: string) => <Tag>{v}</Tag> },
                  { title: "命名空间", dataIndex: "namespace", render: (v: string) => <Tag color="purple">{v}</Tag> },
                  { title: "路径", dataIndex: "path", render: (v: string) => <Tag>{v}</Tag> },
                  { title: "动作码", dataIndex: "action", render: (v: string) => <Tag color="processing">{v}</Tag> },
                ]}
              />
            </>
          ) : (
            <Empty description="暂无可配置角色模板" image={Empty.PRESENTED_IMAGE_SIMPLE} />
          )}
        </Space>
      </Card>

      <Card
        className="table-card"
        style={{ marginTop: 16 }}
        title="命名空间黑名单（对齐 k8m：黑名单优先）"
        loading={denyLoading}
      >
        <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
          若某角色在指定集群下配置了禁止的命名空间，则即使用户拥有该命名空间上的 K8s 三元策略，也会在请求进入集群前被拒绝（适用于保护 kube-system 等）。对已纳入三元校验的接口，含
          super-admin 也会被拦截。
        </Typography.Paragraph>
        {selectedRole ? (
          <Space direction="vertical" size={16} style={{ width: "100%" }}>
            <Form
              form={denyForm}
              layout="inline"
              onFinish={async (v) => {
                const cid = v.cluster_id;
                const ns = String(v.namespace ?? "").trim();
                if (!cid || !ns) {
                  message.warning("请选择集群并填写命名空间");
                  return;
                }
                setDenySubmitting(true);
                try {
                  await createK8sNamespaceDenyRule({ role_code: selectedRole.code, cluster_id: cid, namespace: ns });
                  message.success("已添加黑名单规则");
                  denyForm.resetFields();
                  await refreshDenyRules(selectedRole.code);
                } finally {
                  setDenySubmitting(false);
                }
              }}
            >
              <Typography.Text>角色：</Typography.Text>
              <Tag>{selectedRole.code}</Tag>
              <Form.Item name="cluster_id" rules={[{ required: true, message: "请选择集群" }]}>
                <Select
                  style={{ minWidth: 220 }}
                  placeholder="集群"
                  allowClear
                  options={clusterOptions.map((c) => ({ label: `${c.name} (#${c.id})`, value: c.id }))}
                />
              </Form.Item>
              <Form.Item name="namespace" rules={[{ required: true, message: "填写命名空间" }]}>
                <Input style={{ width: 200 }} placeholder="例如 kube-system" />
              </Form.Item>
              <Form.Item>
                <Button type="primary" htmlType="submit" loading={denySubmitting}>
                  添加禁止规则
                </Button>
              </Form.Item>
            </Form>
            <Table<K8sNamespaceDenyRule>
              rowKey="id"
              size="small"
              dataSource={denyRules}
              pagination={{ pageSize: 8 }}
              columns={[
                { title: "ID", dataIndex: "id", width: 70 },
                { title: "集群 ID", dataIndex: "cluster_id", width: 100 },
                { title: "命名空间", dataIndex: "namespace" },
                {
                  title: "操作",
                  key: "op",
                  width: 100,
                  render: (_, r) => (
                    <Popconfirm title="确定删除该黑名单规则？" onConfirm={() => void (async () => {
                      try {
                        await deleteK8sNamespaceDenyRule(r.id);
                        message.success("已删除");
                        await refreshDenyRules(selectedRole.code);
                      } catch {
                        /* http 拦截器已提示 */
                      }
                    })()}>
                      <Button type="link" danger size="small" icon={<DeleteOutlined />}>
                        删除
                      </Button>
                    </Popconfirm>
                  ),
                },
              ]}
            />
          </Space>
        ) : (
          <Empty description="请先选择角色" image={Empty.PRESENTED_IMAGE_SIMPLE} />
        )}
      </Card>
    </div>
  );
}

