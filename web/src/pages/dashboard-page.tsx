import { ApiOutlined, ApartmentOutlined, AuditOutlined, TeamOutlined, CheckCircleOutlined, ClockCircleOutlined, InfoOutlined } from "@ant-design/icons";
import { Card, Col, Row, Space, Statistic, Tag, Typography, Progress } from "antd";
import { useEffect, useState } from "react";
import { getPermissions } from "../services/permissions";
import { getPolicies } from "../services/policies";
import { getRoles } from "../services/roles";
import { getUsers } from "../services/users";
import { getHealth } from "../services/auth";
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
    hint: "可登录与分配角色的账号数量",
    icon: <TeamOutlined />,
    color: "#0f766e",
    gradient: "linear-gradient(135deg, #0f766e 0%, #14804a 100%)",
  },
  {
    key: "roles",
    title: "角色模板",
    hint: "用于批量挂载权限的角色定义",
    icon: <ApartmentOutlined />,
    color: "#0f6cbd",
    gradient: "linear-gradient(135deg, #0f6cbd 0%, #0077ea 100%)",
  },
  {
    key: "permissions",
    title: "接口能力",
    hint: "已注册的 API 与资源点",
    icon: <ApiOutlined />,
    color: "#c96a11",
    gradient: "linear-gradient(135deg, #c96a11 0%, #fa8c16 100%)",
  },
  {
    key: "policies",
    title: "授权编排",
    hint: "角色与权限的绑定关系",
    icon: <AuditOutlined />,
    color: "#c23a2b",
    gradient: "linear-gradient(135deg, #c23a2b 0%, #f5222d 100%)",
  },
] as const;

function formatUptime(seconds: number): string {
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  return `${hours}小时 ${minutes}分钟`;
}

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

  const uptimePercentage = Math.min(100, (health.uptime / 86400) * 100);

  return (
    <div>
      <Row gutter={[20, 20]} style={{ marginBottom: 24 }}>
        {statItems.map((item) => (
          <Col xs={24} sm={12} lg={6} key={item.key}>
            <Card
              className="stats-card"
              loading={loading}
              hoverable
              style={{
                height: "100%",
                borderLeft: `4px solid ${item.color}`,
              }}
            >
              <Statistic
                title={<span style={{ color: "#5a6d89", fontSize: 14 }}>{item.title}</span>}
                value={metrics[item.key]}
                valueStyle={{ color: item.color, fontSize: 32, fontWeight: 700 }}
                prefix={
                  <span
                    style={{
                      display: "inline-flex",
                      alignItems: "center",
                      justifyContent: "center",
                      width: 42,
                      height: 42,
                      borderRadius: 12,
                      background: item.gradient,
                      color: "#fff",
                      fontSize: 20,
                      marginRight: 12,
                    }}
                  >
                    {item.icon}
                  </span>
                }
              />
              <Typography.Paragraph
                className="stats-card__hint"
                style={{ marginTop: 12, marginBottom: 0, fontSize: 12, color: "#6f819d" }}
              >
                {item.hint}
              </Typography.Paragraph>
            </Card>
          </Col>
        ))}
      </Row>

      <div className="dashboard-pair-grid">
        <Card
          className="table-card"
          title={
            <Space>
              <CheckCircleOutlined style={{ color: "#52c41a" }} />
              系统状态
            </Space>
          }
          loading={health.loading}
        >
            <Space direction="vertical" size="large" style={{ width: "100%" }}>
              <Row gutter={16}>
                <Col span={8}>
                  <div style={{ textAlign: "center", padding: "16px 0" }}>
                    <Typography.Text type="secondary" style={{ display: "block", marginBottom: 8 }}>
                      服务状态
                    </Typography.Text>
                    {health.status === "ok" || health.status === "healthy" ? (
                      <Tag color="success" style={{ fontSize: 14, padding: "4px 12px" }}>
                        运行正常
                      </Tag>
                    ) : health.status === "error" ? (
                      <Tag color="error" style={{ fontSize: 14, padding: "4px 12px" }}>
                        连接异常
                      </Tag>
                    ) : (
                      <Tag color="warning" style={{ fontSize: 14, padding: "4px 12px" }}>
                        {health.status}
                      </Tag>
                    )}
                  </div>
                </Col>
                <Col span={8}>
                  <div style={{ textAlign: "center", padding: "16px 0" }}>
                    <Typography.Text type="secondary" style={{ display: "block", marginBottom: 8 }}>
                      版本信息
                    </Typography.Text>
                    <Tag style={{ fontSize: 14, padding: "4px 12px" }}>
                      <InfoOutlined style={{ marginRight: 4 }} />
                      {health.version}
                    </Tag>
                  </div>
                </Col>
                <Col span={8}>
                  <div style={{ textAlign: "center", padding: "16px 0" }}>
                    <Typography.Text type="secondary" style={{ display: "block", marginBottom: 8 }}>
                      运行时间
                    </Typography.Text>
                    <div style={{ color: "#0f6cbd", fontWeight: 600 }}>
                      <ClockCircleOutlined style={{ marginRight: 4 }} />
                      {formatUptime(health.uptime)}
                    </div>
                  </div>
                </Col>
              </Row>
              <div style={{ borderTop: "1px solid #f0f0f0", paddingTop: 16 }}>
                <Typography.Text type="secondary" style={{ display: "block", marginBottom: 8 }}>
                  运行时长进度（以 24 小时为基准）
                </Typography.Text>
                <Progress
                  percent={uptimePercentage}
                  strokeColor={{
                    "0%": "#108ee9",
                    "100%": "#87d068",
                  }}
                  trailColor="#f5f5f5"
                  format={(percent) => `${Math.floor(((percent ?? 0) / 100) * 24)}小时`}
                />
              </div>
            </Space>
          </Card>

        <Card
          className="table-card"
          title={
            <Space>
              <ApiOutlined style={{ color: "#0f6cbd" }} />
              最近授权
            </Space>
          }
          loading={loading}
        >
            {policies.length > 0 ? (
              <Space direction="vertical" size="small" style={{ width: "100%" }}>
                {policies.map((policy, idx) => (
                  <div
                    key={idx}
                    style={{
                      display: "flex",
                      justifyContent: "space-between",
                      alignItems: "center",
                      padding: "10px 12px",
                      background: "#fafafa",
                      borderRadius: 8,
                      marginBottom: idx < policies.length - 1 ? 8 : 0,
                    }}
                  >
                    <Space>
                      <Tag color="blue">{policy.role_name}</Tag>
                      <Typography.Text style={{ fontSize: 13 }}>
                        {policy.permission_name}
                      </Typography.Text>
                    </Space>
                    <Tag color="green" style={{ fontSize: 12 }}>
                      {policy.action} {policy.resource}
                    </Tag>
                  </div>
                ))}
              </Space>
            ) : (
              <Typography.Text type="secondary">暂无授权记录</Typography.Text>
            )}
          </Card>
      </div>
    </div>
  );
}
