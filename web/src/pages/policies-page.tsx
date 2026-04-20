import { ReloadOutlined, SaveOutlined } from "@ant-design/icons";
import { Button, Card, Empty, Input, Select, Space, Table, Tag, Tree, Typography, message } from "antd";
import { useEffect, useMemo, useState } from "react";
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
  const [roleKeyword, setRoleKeyword] = useState("");
  const [permissionKeyword, setPermissionKeyword] = useState("");
  const [roleStatus, setRoleStatus] = useState<number | undefined>();

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
  const filteredRoles = useMemo(() => {
    const key = roleKeyword.trim().toLowerCase();
    return roles.filter((role) => {
      const matchKeyword = !key || role.name.toLowerCase().includes(key) || role.code.toLowerCase().includes(key);
      const matchStatus = roleStatus === undefined || role.status === roleStatus;
      return matchKeyword && matchStatus;
    });
  }, [roles, roleKeyword, roleStatus]);
  const filteredPermissionTree = useMemo(() => {
    const key = permissionKeyword.trim().toLowerCase();
    if (!key) return permissionTreeData;
    const walk = (nodes: any[]): any[] => {
      const next: any[] = [];
      for (const node of nodes) {
        const titleText = String(node.title ?? "").toLowerCase();
        const children = Array.isArray(node.children) ? walk(node.children) : [];
        if (titleText.includes(key) || children.length > 0) {
          next.push({ ...node, children });
        }
      }
      return next;
    };
    return walk(permissionTreeData as any[]);
  }, [permissionTreeData, permissionKeyword]);

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
      setCheckedPermissionIds(nextRoleId ? getRolePermissionIds(policyList, nextRoleId).filter((id) => id > 0) : []);
    } finally {
      setLoading(false);
    }
  }

  function handleRoleChange(value: number) {
    setSelectedRoleId(value);
    setCheckedPermissionIds(getRolePermissionIds(list, value).filter((id) => id > 0));
  }

  async function handleSave() {
    if (!selectedRoleId) {
      message.warning("请先选择一个角色模板");
      return;
    }

    const currentIds = getRolePermissionIds(list, selectedRoleId).filter((id) => id > 0);
    const desiredIds = checkedPermissionIds.filter((id) => id > 0 && permissionIdSet.has(id));
    const currentIdSet = new Set(currentIds);
    const desiredIdSet = new Set(desiredIds);
    const toGrant = desiredIds.filter((id) => !currentIdSet.has(id));
    const toRevoke = currentIds.filter((id) => !desiredIdSet.has(id));

    if (toGrant.length === 0 && toRevoke.length === 0) {
      message.info("授权编排没有变化");
      return;
    }

    setSubmitting(true);
    try {
      await Promise.all([
        ...toGrant.map((permissionId) => grantPolicy({ role_id: selectedRoleId, permission_id: permissionId })),
        ...toRevoke.map((permissionId) => revokePolicy({ role_id: selectedRoleId, permission_id: permissionId })),
      ]);
      message.success("授权编排已同步");
      await bootstrap(selectedRoleId);
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div>
      <Card className="table-card" loading={loading}>
        <div className="toolbar auth-toolbar">
          <Space>
            <Input
              allowClear
              value={roleKeyword}
              onChange={(e) => setRoleKeyword(e.target.value)}
              placeholder="分组名称/编码"
              style={{ width: 180 }}
            />
            <Input
              allowClear
              value={permissionKeyword}
              onChange={(e) => setPermissionKeyword(e.target.value)}
              placeholder="权限名称/资源路径"
              style={{ width: 220 }}
            />
            <Select
              allowClear
              placeholder="角色状态"
              style={{ width: 130 }}
              value={roleStatus}
              onChange={(v) => setRoleStatus(v)}
              options={[
                { value: 1, label: "启用" },
                { value: 0, label: "停用" },
              ]}
            />
          </Space>
          <div className="toolbar__actions">
            <Button
              onClick={() => {
                setRoleKeyword("");
                setPermissionKeyword("");
                setRoleStatus(undefined);
              }}
            >
              重置
            </Button>
            <Button icon={<ReloadOutlined />} onClick={() => void bootstrap(selectedRoleId)}>
              刷新
            </Button>
            <Button type="primary" icon={<SaveOutlined />} loading={submitting} onClick={() => void handleSave()}>
              同步权限
            </Button>
          </div>
        </div>

        <div className="auth-split">
          <Card
            className="glass-card auth-split__left"
            title="权限分组"
            extra={<Tag className="status-chip status-chip--ok">共 {filteredRoles.length} 项</Tag>}
          >
            <Table
              rowKey="id"
              dataSource={filteredRoles}
              pagination={false}
              size="small"
              scroll={{ y: 560 }}
              rowClassName={(record) => (record.id === selectedRoleId ? "is-selected-row" : "")}
              onRow={(record) => ({
                onClick: () => handleRoleChange(record.id),
              })}
              columns={[
                { title: "分组名称", dataIndex: "name" },
                { title: "业务标识", dataIndex: "code" },
                {
                  title: "状态",
                  dataIndex: "status",
                  width: 90,
                  render: (status: number) =>
                    status === 1 ? <Tag className="status-chip status-chip--ok">正常</Tag> : <Tag className="status-chip status-chip--off">停用</Tag>,
                },
              ]}
            />
          </Card>

          <Card
            className="glass-card auth-split__right"
            title="授权管理"
            extra={
              selectedRole ? (
                <Typography.Text className="inline-muted">
                  当前角色：{selectedRole.name}（已选 {checkedPermissionIds.length} 项）
                </Typography.Text>
              ) : null
            }
          >
            {selectedRole ? (
              <div className="auth-right-stack">
                <div className="auth-right-tree">
                  <div className="auth-right-tree__head">权限配置树</div>
                  <div className="tree-shell auth-tree-shell">
                    <Tree
                      checkable
                      defaultExpandAll
                      checkedKeys={checkedPermissionIds}
                      treeData={filteredPermissionTree}
                      onCheck={(checkedKeys) => {
                        const nextIds = normalizeCheckedKeys(checkedKeys).filter((id) => permissionIdSet.has(id));
                        setCheckedPermissionIds(nextIds);
                      }}
                    />
                  </div>
                </div>
                <div className="auth-right-result">
                  <div className="auth-right-result__head">已授权权限</div>
                  <Table
                    rowKey={(record) => `${record.role_id}-${record.permission_id}`}
                    dataSource={currentRolePolicies}
                    pagination={{ pageSize: 8 }}
                    size="small"
                    scroll={{ y: 260 }}
                    columns={[
                      { title: "权限名称", dataIndex: "permission_name" },
                      { title: "权限编码", dataIndex: "resource", render: (value: string) => <Tag>{value}</Tag> },
                      { title: "分组名称", dataIndex: "role_name" },
                      { title: "状态", dataIndex: "action", width: 80, render: () => <Tag className="status-chip status-chip--ok">正常</Tag> },
                    ]}
                  />
                </div>
              </div>
            ) : (
              <Empty description="请选择左侧分组后进行授权" image={Empty.PRESENTED_IMAGE_SIMPLE} />
            )}
          </Card>
        </div>
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
