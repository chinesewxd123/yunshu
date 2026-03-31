import { ReloadOutlined, SaveOutlined } from "@ant-design/icons";
import { Button, Card, Empty, Select, Space, Table, Tag, Tree, Typography, message } from "antd";
import { useEffect, useMemo, useState } from "react";
import { PageHero } from "../components/page-hero";
import { getPermissionOptions } from "../services/permissions";
import { getPolicies, grantPolicy, revokePolicy } from "../services/policies";
import { getRoleOptions } from "../services/roles";
import type { PermissionItem, PolicyItem, RoleItem } from "../types/api";
import { buildPermissionTreeData, normalizeCheckedKeys } from "../utils/tree";

export function PoliciesPage() {
  const [list, setList] = useState<PolicyItem[]>([]);
  const [roles, setRoles] = useState<RoleItem[]>([]);
  const [permissions, setPermissions] = useState<PermissionItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [selectedRoleId, setSelectedRoleId] = useState<number>();
  const [checkedPermissionIds, setCheckedPermissionIds] = useState<number[]>([]);

  const permissionTreeData = useMemo(() => buildPermissionTreeData(permissions), [permissions]);
  const permissionIdSet = useMemo(() => new Set(permissions.map((permission) => permission.id)), [permissions]);
  const selectedRole = useMemo(
    () => roles.find((role) => role.id === selectedRoleId) ?? null,
    [roles, selectedRoleId],
  );
  const currentRolePolicies = useMemo(
    () => (selectedRoleId ? list.filter((policy) => policy.role_id === selectedRoleId) : []),
    [list, selectedRoleId],
  );

  useEffect(() => {
    void bootstrap();
  }, []);

  async function bootstrap(preferredRoleId?: number) {
    setLoading(true);
    try {
      const [policyList, roleData, permissionData] = await Promise.all([
        getPolicies(),
        getRoleOptions(),
        getPermissionOptions(),
      ]);

      setList(policyList);
      setRoles(roleData.list);
      setPermissions(permissionData.list);

      const nextRoleId = preferredRoleId ?? selectRoleId(roleData.list, selectedRoleId);
      setSelectedRoleId(nextRoleId);
      setCheckedPermissionIds(nextRoleId ? getRolePermissionIds(policyList, nextRoleId) : []);
    } finally {
      setLoading(false);
    }
  }

  function handleRoleChange(value: number) {
    setSelectedRoleId(value);
    setCheckedPermissionIds(getRolePermissionIds(list, value));
  }

  async function handleSave() {
    if (!selectedRoleId) {
      message.warning("请先选择一个角色");
      return;
    }

    const currentIds = getRolePermissionIds(list, selectedRoleId);
    const desiredIds = checkedPermissionIds.filter((id) => permissionIdSet.has(id));
    const currentIdSet = new Set(currentIds);
    const desiredIdSet = new Set(desiredIds);
    const toGrant = desiredIds.filter((id) => !currentIdSet.has(id));
    const toRevoke = currentIds.filter((id) => !desiredIdSet.has(id));

    if (toGrant.length === 0 && toRevoke.length === 0) {
      message.info("权限树没有变化");
      return;
    }

    setSubmitting(true);
    try {
      await Promise.all([
        ...toGrant.map((permissionId) => grantPolicy({ role_id: selectedRoleId, permission_id: permissionId })),
        ...toRevoke.map((permissionId) => revokePolicy({ role_id: selectedRoleId, permission_id: permissionId })),
      ]);
      message.success("权限树已同步");
      await bootstrap(selectedRoleId);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div>
      <PageHero
        title="策略绑定"
        subtitle="按角色维度配置权限树，勾选后保存即可批量同步 Casbin 策略。"
        breadcrumbItems={[{ title: "控制台" }, { title: "策略绑定" }]}
        extra={
          <Space wrap>
            <Button icon={<ReloadOutlined />} onClick={() => void bootstrap(selectedRoleId)}>
              刷新
            </Button>
            <Button type="primary" icon={<SaveOutlined />} loading={submitting} onClick={() => void handleSave()}>
              保存权限树
            </Button>
          </Space>
        }
      />

      <div className="section-grid">
        <Card className="glass-card" title="角色维度">
          <Space direction="vertical" size={16} style={{ width: "100%" }}>
            <Select
              placeholder="请选择角色"
              value={selectedRoleId}
              onChange={handleRoleChange}
              options={roles.map((role) => ({ label: `${role.name} (${role.code})`, value: role.id }))}
            />
            {selectedRole ? (
              <>
                <Typography.Text strong>{selectedRole.name}</Typography.Text>
                <Typography.Text className="inline-muted">角色编码：{selectedRole.code}</Typography.Text>
                <Typography.Text className="inline-muted">已绑定 {currentRolePolicies.length} 条策略</Typography.Text>
              </>
            ) : (
              <Empty description="暂无可配置角色" image={Empty.PRESENTED_IMAGE_SIMPLE} />
            )}
          </Space>
        </Card>

        <Card className="table-card" title="权限树" loading={loading}>
          {selectedRole ? (
            <Space direction="vertical" size={12} style={{ width: "100%" }}>
              <Typography.Text className="inline-muted">
                以资源路径为层级组织权限，已勾选 {checkedPermissionIds.length} 项。
              </Typography.Text>
              <div className="tree-shell tree-shell--tall">
                <Tree
                  checkable
                  defaultExpandAll
                  checkedKeys={checkedPermissionIds}
                  treeData={permissionTreeData}
                  onCheck={(checkedKeys) => {
                    const nextIds = normalizeCheckedKeys(checkedKeys).filter((id) => permissionIdSet.has(id));
                    setCheckedPermissionIds(nextIds);
                  }}
                />
              </div>
            </Space>
          ) : (
            <Empty description="请选择角色后配置权限树" image={Empty.PRESENTED_IMAGE_SIMPLE} />
          )}
        </Card>
      </div>

      <Card className="table-card" title={selectedRole ? `${selectedRole.name} 的当前策略` : "当前策略列表"}>
        <Table
          rowKey={(record) => `${record.role_id}-${record.permission_id}`}
          loading={loading}
          dataSource={selectedRole ? currentRolePolicies : list}
          pagination={{ pageSize: 10 }}
          columns={[
            { title: "角色", dataIndex: "role_name" },
            { title: "角色编码", dataIndex: "role_code", render: (value: string) => <Tag color="blue">{value}</Tag> },
            { title: "权限", dataIndex: "permission_name" },
            { title: "资源", dataIndex: "resource", render: (value: string) => <Tag>{value}</Tag> },
            { title: "动作", dataIndex: "action", render: (value: string) => <Tag color="processing">{value}</Tag> },
          ]}
        />
      </Card>
    </div>
  );
}

function selectRoleId(roles: RoleItem[], currentRoleId?: number) {
  if (currentRoleId && roles.some((role) => role.id === currentRoleId)) {
    return currentRoleId;
  }
  return roles[0]?.id;
}

function getRolePermissionIds(policies: PolicyItem[], roleId: number) {
  return policies.filter((policy) => policy.role_id === roleId).map((policy) => policy.permission_id);
}
