import { ReloadOutlined, ThunderboltOutlined } from "@ant-design/icons";
import { Button, Card, Divider, Empty, Form, Select, Space, Table, Tag, Typography, message } from "antd";
import { useEffect, useMemo, useState } from "react";
import { getClusters } from "../services/clusters";
import { grantK8sScopedPolicies, listK8sPoliciesByRole, listK8sPolicyActions, listK8sPolicyPaths } from "../services/k8s-policies";
import { listNamespaces } from "../services/namespaces";
import { getRoleOptions } from "../services/roles";
import type { RoleItem } from "../types/api";

export function K8sScopedPoliciesPage() {
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
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
  const [form] = Form.useForm<{
    cluster_ids: number[];
    namespaces: string[];
    actions: string[];
    paths: string[];
  }>();

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
      } else {
        setK8sPolicies([]);
      }
    } finally {
      setLoading(false);
    }
  }

  async function refresh(roleId: number) {
    const result = await listK8sPoliciesByRole(roleId);
    setK8sPolicies(result.list);
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
                pagination={{ pageSize: 10 }}
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
    </div>
  );
}

