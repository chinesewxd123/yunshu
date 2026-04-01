import { ApiOutlined, ApartmentOutlined, AuditOutlined, TeamOutlined, CheckCircleOutlined, CloseCircleOutlined } from "@ant-design/icons";
import { Card, List, Space, Table, Tag, Typography, message } from "antd";
import { useEffect, useState } from "react";
import { PageHero } from "../components/page-hero";
import { BRAND_NAME } from "../constants/brand";
import { getPermissions } from "../services/permissions";
import { getPolicies } from "../services/policies";
import { getRoles } from "../services/roles";
import { getUsers } from "../services/users";
import { getHealth, type HealthData } from "../services/auth";
import type { PolicyItem } from "../types/api";

interface DashboardMetrics {
  users: number;
  roles: number;
  permissions: number;
  policies: number;
}

interface SystemHealth {
  status: string;
  version: string;
  uptime: number;
  loading: boolean;
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
  const [health, setHealth] = useState<SystemHealth>({ status: "", version: "", uptime: 0, loading: true });
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let active = true;

    async function load() {
      setLoading(true);
      setHealth((prev) => ({ ...prev, loading: true }));
      try {
        const [users, roles, permissions, policyList, healthData] = await Promise.all([
          getUsers({ page: 1, page_size: 1 }),
          getRoles({ page: 1, page_size: 1 }),
          getPermissions({ page: 1, page_size: 1 }),
          getPolicies(),
          getHealth().catch(() => null),
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

        if (healthData) {
          setHealth({
            status: healthData.status || "unknown",
            version: healthData.version || "-",
            uptime: healthData.uptime || 0,
            loading: false,
          });
        } else {
          setHealth((prev) => ({ ...prev, status: "error", loading: false }));
        }
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

      <Card className="table-card" title="系统状态" style={{ marginTop: 24 }} loading={health.loading}>
        <Space size="large">
          <div>
            <Typography.Text type="secondary">服务状态</Typography.Text>
            <div style={{ display: "flex", alignItems: "center", gap: 8, marginTop: 4 }}>
              {health.status === "ok" || health.status === "healthy" ? (
                <>
                  <CheckCircleOutlined style={{ color: "#52c41a", fontSize: 20 }} />
                  <Tag color="success">运行正常</Tag>
                </>
              ) : health.status === "error" ? (
                <>
                  <CloseCircleOutlined style={{ color: "#ff4d4f", fontSize: 20 }} />
                  <Tag color="error">连接异常</Tag>
                </>
              ) : (
                <>
                  <Tag color="warning">{health.status}</Tag>
                </>
              )}
            </div>
          </div>
          <div>
            <Typography.Text type="secondary">版本</Typography.Text>
            <div style={{ marginTop: 4 }}>
              <Tag>{health.version}</Tag>
            </div>
          </div>
          <div>
            <Typography.Text type="secondary">运行时间</Typography.Text>
            <div style={{ marginTop: 4 }}>
              <Typography.Text strong>{Math.floor(health.uptime / 3600)} 小时 {Math.floor((health.uptime % 3600) / 60)} 分钟</Typography.Text>
            </div>
          </div>
        </Space>
      </Card>

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
