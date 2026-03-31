import { AuditOutlined, LockOutlined, SafetyOutlined, TeamOutlined } from "@ant-design/icons";
import { Card, List, Space, Statistic, Table, Tag, Typography } from "antd";
import { useEffect, useState } from "react";
import { PageHero } from "../components/page-hero";
import { getPermissions } from "../services/permissions";
import { getPolicies } from "../services/policies";
import { getRoles } from "../services/roles";
import { getUsers } from "../services/users";
import type { PolicyItem } from "../types/api";

interface DashboardMetrics {
  users: number;
  roles: number;
  permissions: number;
  policies: number;
}

const statItems = [
  { key: "users", title: "用户总数", icon: <TeamOutlined />, color: "#0f766e" },
  { key: "roles", title: "角色数量", icon: <SafetyOutlined />, color: "#2563eb" },
  { key: "permissions", title: "权限数量", icon: <LockOutlined />, color: "#ea580c" },
  { key: "policies", title: "策略绑定", icon: <AuditOutlined />, color: "#7c3aed" },
] as const;

export function DashboardPage() {
  const [metrics, setMetrics] = useState<DashboardMetrics>({ users: 0, roles: 0, permissions: 0, policies: 0 });
  const [policies, setPolicies] = useState<PolicyItem[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let active = true;

    async function load() {
      setLoading(true);
      try {
        const [users, roles, permissions, policyList] = await Promise.all([
          getUsers({ page: 1, page_size: 1 }),
          getRoles({ page: 1, page_size: 1 }),
          getPermissions({ page: 1, page_size: 1 }),
          getPolicies(),
        ]);
        if (!active) {
          return;
        }
        setMetrics({
          users: users.total,
          roles: roles.total,
          permissions: permissions.total,
          policies: policyList.length,
        });
        setPolicies(policyList.slice(0, 8));
      } finally {
        if (active) {
          setLoading(false);
        }
      }
    }

    void load();
    return () => {
      active = false;
    };
  }, []);

  return (
    <div>
      <PageHero
        title="系统概览"
        subtitle="从这里快速查看权限系统的运行基线，包括用户规模、角色编排、权限数量与策略绑定状态。"
        breadcrumbItems={[{ title: "控制台" }, { title: "概览" }]}
      />

      <div className="stats-grid">
        {statItems.map((item) => (
          <Card key={item.key} className="stats-card" loading={loading}>
            <Statistic
              title={item.title}
              value={metrics[item.key]}
              prefix={<span style={{ color: item.color }}>{item.icon}</span>}
            />
          </Card>
        ))}
      </div>

      <div className="metric-strip">
        <Card className="table-card" title="最近策略绑定" loading={loading}>
          <Table
            rowKey={(record) => `${record.role_id}-${record.permission_id}`}
            pagination={false}
            dataSource={policies}
            columns={[
              { title: "角色", dataIndex: "role_name" },
              { title: "角色编码", dataIndex: "role_code", render: (value: string) => <Tag>{value}</Tag> },
              { title: "权限", dataIndex: "permission_name" },
              { title: "资源", dataIndex: "resource" },
              { title: "动作", dataIndex: "action", render: (value: string) => <Tag color="processing">{value}</Tag> },
            ]}
          />
        </Card>
        <Space direction="vertical" size={20} style={{ width: "100%" }}>
          <Card className="glass-card">
            <Typography.Title level={4}>默认调试入口</Typography.Title>
            <List
              dataSource={[
                "Swagger UI: /swagger/index.html",
                "OpenAPI JSON: /swagger/doc.json",
                "API Base URL: /api/v1",
              ]}
              renderItem={(item) => <List.Item>{item}</List.Item>}
            />
          </Card>
          <Card className="glass-card">
            <Typography.Title level={4}>建议联调顺序</Typography.Title>
            <List
              dataSource={[
                "登录并确认当前用户信息",
                "创建角色和权限基础数据",
                "建立策略绑定后再分配用户角色",
                "使用 Swagger 或 APIpost 做接口回归",
              ]}
              renderItem={(item, index) => (
                <List.Item>
                  <span className="code-pill">0{index + 1}</span>
                  <span style={{ marginLeft: 12 }}>{item}</span>
                </List.Item>
              )}
            />
          </Card>
        </Space>
      </div>
    </div>
  );
}
