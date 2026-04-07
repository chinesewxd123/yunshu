import { CheckCircleOutlined, ClockCircleOutlined, ClusterOutlined, InfoOutlined, TeamOutlined, WarningOutlined } from "@ant-design/icons";
import { Card, Col, Row, Space, Statistic, Tag, Typography, Progress } from "antd";
import { useEffect, useState } from "react";
import { getHealth } from "../services/auth";
import { getOverview } from "../services/overview";

interface DashboardMetrics {
  users: number;
  clusters: number;
  podNormal: number;
  podAbnormal: number;
  podClusterErrors: number;
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
    hint: "系统内已创建的账号数量",
    icon: <TeamOutlined />,
    color: "#0f766e",
    gradient: "linear-gradient(135deg, #0f766e 0%, #14804a 100%)",
  },
  {
    key: "clusters",
    title: "K8s 集群",
    hint: "已注册到系统的集群数量",
    icon: <ClusterOutlined />,
    color: "#0f6cbd",
    gradient: "linear-gradient(135deg, #0f6cbd 0%, #0077ea 100%)",
  },
  {
    key: "podNormal",
    title: "Pod 正常",
    hint: "Running 且所有容器 Ready",
    icon: <CheckCircleOutlined />,
    color: "#c96a11",
    gradient: "linear-gradient(135deg, #c96a11 0%, #fa8c16 100%)",
  },
  {
    key: "podAbnormal",
    title: "Pod 异常",
    hint: "非 Running 或容器未就绪",
    icon: <WarningOutlined />,
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
  const [metrics, setMetrics] = useState<DashboardMetrics>({
    users: 0,
    clusters: 0,
    podNormal: 0,
    podAbnormal: 0,
    podClusterErrors: 0,
  });
  const [health, setHealth] = useState<SystemHealth>({ status: "", version: "", uptime: 0, loading: true });
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let active = true;

    async function load() {
      setLoading(true);
      setHealth((prev) => ({ ...prev, loading: true }));
      try {
        const [overview, healthData] = await Promise.all([getOverview(), getHealth().catch(() => null)]);

        if (!active) {
          return;
        }

        setMetrics({
          users: overview.users_count,
          clusters: overview.clusters_count,
          podNormal: overview.pod_normal_count,
          podAbnormal: overview.pod_abnormal_count,
          podClusterErrors: overview.pod_cluster_errors,
        });

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

        <Card className="table-card" title="采集状态" loading={loading}>
          <Space direction="vertical" size={8} style={{ width: "100%" }}>
            <Typography.Text className="inline-muted">
              Pod 统计按「启用」集群跨命名空间聚合；若集群不可达会计入失败数。
            </Typography.Text>
            <Row gutter={16}>
              <Col span={12}>
                <Statistic title="正常 Pod" value={metrics.podNormal} valueStyle={{ color: "#14804a", fontWeight: 700 }} />
              </Col>
              <Col span={12}>
                <Statistic title="异常 Pod" value={metrics.podAbnormal} valueStyle={{ color: "#c23a2b", fontWeight: 700 }} />
              </Col>
            </Row>
            <div style={{ display: "flex", justifyContent: "space-between" }}>
              <Typography.Text type="secondary">采集失败集群数</Typography.Text>
              <Tag color={metrics.podClusterErrors > 0 ? "warning" : "success"}>{metrics.podClusterErrors}</Tag>
            </div>
          </Space>
        </Card>
      </div>
    </div>
  );
}
