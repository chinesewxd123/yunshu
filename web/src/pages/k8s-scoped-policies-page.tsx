import { DeleteOutlined, GiftOutlined, ReloadOutlined } from "@ant-design/icons";
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
  deleteK8sClusterGrant,
  grantK8sScopedPoliciesPreset,
  listK8sPoliciesByRole,
  type K8sClusterAccessItem,
} from "../services/k8s-policies";
import { getRoleOptions } from "../services/roles";
import type { RoleItem } from "../types/api";

export function K8sScopedPoliciesPage() {
  const [loading, setLoading] = useState(false);
  const [presetSubmitting, setPresetSubmitting] = useState(false);
  const [roles, setRoles] = useState<RoleItem[]>([]);
  const [selectedRoleId, setSelectedRoleId] = useState<number>();
  const [clusterOptions, setClusterOptions] = useState<{ id: number; name: string }[]>([]);
  const [accessGrants, setAccessGrants] = useState<K8sClusterAccessItem[]>([]);
  const [denyRules, setDenyRules] = useState<K8sNamespaceDenyRule[]>([]);
  const [denyLoading, setDenyLoading] = useState(false);
  const [denySubmitting, setDenySubmitting] = useState(false);
  const [presetForm] = Form.useForm<{
    cluster_ids: number[];
    preset: "readonly" | "readonly_exec" | "admin";
    deny_namespaces?: string[];
    allow_namespaces?: string[];
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
      const [roleData, clusterData] = await Promise.all([getRoleOptions(), getClusters({ page: 1, page_size: 200 })]);
      setRoles(roleData.list);
      setClusterOptions(clusterData.list.map((c) => ({ id: c.id, name: c.name })));

      const nextRoleId = preferredRoleId ?? (roleData.list[0]?.id ?? undefined);
      setSelectedRoleId(nextRoleId);
      if (nextRoleId) {
        await refresh(nextRoleId);
        const rc = roleData.list.find((r) => r.id === nextRoleId)?.code ?? "";
        await refreshDenyRules(rc);
      } else {
        setAccessGrants([]);
        setDenyRules([]);
      }
    } finally {
      setLoading(false);
    }
  }

  async function refresh(roleId: number) {
    const result = await listK8sPoliciesByRole(roleId);
    setAccessGrants(result.list);
  }

  async function refreshDenyRules(roleCode: string) {
    if (!roleCode.trim()) {
      setDenyRules([]);
      return;
    }
    setDenyLoading(true);
    try {
      const data = await listK8sNamespaceDenyRules({ principal_kind: "role", principal_ref: roleCode });
      setDenyRules(data.list ?? []);
    } catch {
      setDenyRules([]);
    } finally {
      setDenyLoading(false);
    }
  }

  function presetLabel(p: string) {
    switch (p) {
      case "readonly":
        return "只读";
      case "readonly_exec":
        return "只读+Exec";
      case "admin":
        return "集群管理";
      default:
        return p;
    }
  }

  return (
    <div>
      <Card
        className="table-card"
        title="Kubernetes 集群访问档位（数据库维护，不经 Casbin）"
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
              <Alert
                type="info"
                showIcon
                style={{ width: "100%" }}
                message="与 API / Casbin 的关系"
                description={
                  <span>
                    此处为<strong>主体</strong>（本页为角色模板；亦可对用户 / 用户组）配置<strong>集群维度档位</strong>（只读 / 只读+Exec / 管理），数据在表{" "}
                    <Typography.Text code>k8s_cluster_access_grants</Typography.Text>。HTTP 接口能否调用仍由<strong>授权管理</strong>中的 Casbin
                    API 权限决定；带 <Typography.Text code>cluster_id</Typography.Text> 的 K8s 类请求在通过 API 鉴权后，再按此处档位与<strong>命名空间黑/白名单</strong>校验。详见{" "}
                    <Typography.Text code>docs/handbook/permissions/casbin-and-k8s-triple-policy.md</Typography.Text>。
                  </span>
                }
              />

              <Alert
                type="info"
                showIcon
                style={{ width: "100%" }}
                message="档位下发（对齐 k8m 语义）"
                description={
                  <span>
                    按主体（当前页为<strong>角色模板</strong>）+ 集群写入档位；不选集群表示 <Tag>全部集群（ID=0）</Tag>。命名空间黑/白名单可选：须选择<strong>具体集群</strong>（勿仅用「全部集群」），否则无法写入规则。若某主体在某集群存在白名单规则，则仅允许名单内命名空间（黑名单仍优先）。
                  </span>
                }
              />

              <Form
                form={presetForm}
                layout="vertical"
                initialValues={{
                  cluster_ids: [],
                  preset: "readonly" as const,
                  deny_namespaces: [],
                  allow_namespaces: [],
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
                      placeholder="不选 = 全部集群"
                      options={clusterOptions.map((c) => ({ label: `${c.name} (#${c.id})`, value: c.id }))}
                    />
                  </Form.Item>
                  <Form.Item
                    label="同步命名空间黑名单（可选）"
                    name="deny_namespaces"
                    tooltip="须在上栏选择具体集群；对每个所选集群写入禁止访问该命名空间"
                  >
                    <Select mode="tags" style={{ minWidth: 320 }} placeholder="输入后回车，例如 kube-system" tokenSeparators={[","]} />
                  </Form.Item>
                  <Form.Item
                    label="同步命名空间白名单（可选）"
                    name="allow_namespaces"
                    tooltip="须选择具体集群；写入后该主体在该集群仅允许访问所列命名空间（黑名单优先）"
                  >
                    <Select mode="tags" style={{ minWidth: 320 }} placeholder="输入后回车" tokenSeparators={[","]} />
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
                            const allowRaw = values.allow_namespaces ?? [];
                            const allowList = (Array.isArray(allowRaw) ? allowRaw : []).map((s) => String(s).trim()).filter(Boolean);
                            const resp = await grantK8sScopedPoliciesPreset({
                              principal_kind: "role",
                              role_id: roleId,
                              cluster_ids: values.cluster_ids ?? [],
                              preset: values.preset,
                              deny_namespaces: denyList.length ? denyList : undefined,
                              allow_namespaces: allowList.length ? allowList : undefined,
                            });
                            message.success(
                              `档位已保存：新增 ${resp.added}，更新跳过 ${resp.skipped}；黑名单新增 ${resp.deny_rules_added}（跳过 ${resp.deny_rules_skipped}）；白名单新增 ${resp.allow_rules_added}（跳过 ${resp.allow_rules_skipped}）`,
                            );
                            await refresh(roleId);
                            await refreshDenyRules(selectedRole.code);
                          } finally {
                            setPresetSubmitting(false);
                          }
                        })();
                      }}
                    >
                      按档位保存
                    </Button>
                  </Form.Item>
                </Space>
              </Form>

              <Divider style={{ margin: "8px 0" }} />
              <Typography.Text strong>当前角色的集群档位</Typography.Text>
              <Table<K8sClusterAccessItem>
                rowKey="id"
                dataSource={accessGrants}
                pagination={{ pageSize: 10, showSizeChanger: true, pageSizeOptions: [10, 20, 50, 100], showQuickJumper: true }}
                size="small"
                scroll={{ x: "max-content" }}
                columns={[
                  {
                    title: "主体",
                    key: "principal",
                    render: (_: unknown, r: K8sClusterAccessItem) => (
                      <span>
                        <Tag>{r.principal_kind || (r.role_code ? "role" : "")}</Tag>{" "}
                        <Typography.Text code>{r.principal_ref || r.role_code}</Typography.Text>
                      </span>
                    ),
                  },
                  {
                    title: "集群",
                    dataIndex: "cluster_id",
                    render: (v: number) => (v === 0 ? <Tag color="blue">全部集群</Tag> : <Tag>#{v}</Tag>),
                  },
                  {
                    title: "档位",
                    dataIndex: "preset",
                    render: (v: string) => <Tag color="processing">{presetLabel(v)}</Tag>,
                  },
                  {
                    title: "操作",
                    key: "op",
                    width: 100,
                    render: (_, r) => (
                      <Popconfirm
                        title="确定删除该集群档位？"
                        onConfirm={() =>
                          void (async () => {
                            try {
                              await deleteK8sClusterGrant(r.id);
                              message.success("已删除");
                              if (selectedRoleId) await refresh(selectedRoleId);
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
                    ),
                  },
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
        title="命名空间黑名单（对齐 k8m：黑名单优先于白名单与档位）"
        loading={denyLoading}
      >
        <Typography.Paragraph type="secondary" style={{ marginBottom: 12 }}>
          若某<strong>主体</strong>（角色 / 用户 / 组）在指定集群下配置了禁止的命名空间，则即使用户拥有该集群档位，也会在请求进入集群前被拒绝。对已纳入 K8s 范围校验的接口，含
          super-admin 也会被拦截。白名单规则见接口 <Typography.Text code>/api/v1/k8s-namespace-allow-rules</Typography.Text>。
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
                  await createK8sNamespaceDenyRule({
                    principal_kind: "role",
                    principal_ref: selectedRole.code,
                    cluster_id: cid,
                    namespace: ns,
                  });
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
                {
                  title: "主体",
                  key: "p",
                  width: 160,
                  render: (_: unknown, r: K8sNamespaceDenyRule) => (
                    <span>
                      <Tag>{r.principal_kind}</Tag> <Typography.Text code>{r.principal_ref}</Typography.Text>
                    </span>
                  ),
                },
                { title: "集群 ID", dataIndex: "cluster_id", width: 100 },
                { title: "命名空间", dataIndex: "namespace" },
                {
                  title: "操作",
                  key: "op",
                  width: 100,
                  render: (_, r) => (
                    <Popconfirm
                      title="确定删除该黑名单规则？"
                      onConfirm={() =>
                        void (async () => {
                          try {
                            await deleteK8sNamespaceDenyRule(r.id);
                            message.success("已删除");
                            await refreshDenyRules(selectedRole.code);
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
