import { ApiOutlined, ApartmentOutlined, AuditOutlined, TeamOutlined } from "@ant-design/icons";
import { Card, List, Space, Table, Tag, Typography } from "antd";
import { useEffect, useState } from "react";
import { PageHero } from "../components/page-hero";
import { BRAND_NAME } from "../constants/brand";
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
  {
    key: "users",
    title: "账号主体",
    hint: "当前纳管的运维成员与系统账号",
    icon: <TeamOutlined />,
    color: "#0f766e",
  },
  {
    key: "roles",
    title: "角色模板",
    hint: "岗位职责与责任域模板数量",
    icon: <ApartmentOutlined />,
    color: "#0f6cbd",
  },
  {
    key: "permissions",
    title: "接口能力",
    hint: "已沉淀的资源动作能力项",
    icon: <ApiOutlined />,
    color: "#c96a11",
  },
  {
    key: "policies",
    title: "授权编排",
    hint: "角色与能力之间的最终绑定关系",
    icon: <AuditOutlined />,
    color: "#c23a2b",
  },
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
        title="资产总览"
        subtitle={`${BRAND_NAME} 以账号、角色模板、接口能力和授权编排为核心，帮助运维团队先把权限治理做成统一底座，再继续向 CMDB 资产模块扩展。`}
        breadcrumbItems={[{ title: "控制台" }, { title: "资产总览" }]}
        extra={
          <Space wrap>
            <Tag color="processing">Redis 验证码登录</Tag>
            <Tag color="gold">Casbin 策略同步</Tag>
            <Tag color="green">运维控制台</Tag>
          </Space>
        }
      />

      <div className="stats-grid">
        {statItems.map((item) => (
          <Card key={item.key} className="stats-card" loading={loading}>
            <div className="stats-card__icon" style={{ color: item.color }}>
              {item.icon}
            </div>
            <Typography.Text className="stats-card__label">{item.title}</Typography.Text>
            <Typography.Title level={2} className="stats-card__value">
              {metrics[item.key]}
            </Typography.Title>
            <Typography.Paragraph className="stats-card__hint">{item.hint}</Typography.Paragraph>
          </Card>
        ))}
      </div>

      <div className="metric-strip">
        <Card className="table-card" title="最新授权编排" loading={loading}>
          <Table
            rowKey={(record) => `${record.role_id}-${record.permission_id}`}
            pagination={false}
            dataSource={policies}
            columns={[
              { title: "角色模板", dataIndex: "role_name" },
              { title: "模板编码", dataIndex: "role_code", render: (value: string) => <Tag>{value}</Tag> },
              { title: "能力项", dataIndex: "permission_name" },
              { title: "资源路径", dataIndex: "resource" },
              { title: "动作", dataIndex: "action", render: (value: string) => <Tag color="processing">{value}</Tag> },
            ]}
          />
        </Card>

        <Space direction="vertical" size={20} style={{ width: "100%" }}>
          <Card className="glass-card">
            <Typography.Title level={4}>标准接入口</Typography.Title>
            <List
              dataSource={[
                "Swagger UI: /swagger/index.html",
                "OpenAPI JSON: /swagger/doc.json",
                "API Base URL: /api/v1",
                "登录验证码: /api/v1/auth/captcha",
              ]}
              renderItem={(item) => <List.Item>{item}</List.Item>}
            />
          </Card>

          <Card className="glass-card">
            <Typography.Title level={4}>治理建议</Typography.Title>
            <List
              dataSource={[
                "先沉淀角色模板，再批量分配账号责任域。",
                "把常用接口拆成能力项，便于后续接入更多 CMDB 模块。",
                "策略编排完成后再做联调，可减少重复授权操作。",
                "保留 Swagger 作为回归入口，方便前后端一起验收。",
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
