import {
  CheckCircleOutlined,
  ClockCircleOutlined,
  ClusterOutlined,
  DesktopOutlined,
  InfoOutlined,
  LoginOutlined,
  ProfileOutlined,
  SafetyCertificateOutlined,
  TeamOutlined,
  WarningOutlined,
} from "@ant-design/icons";
import { Card, Col, Row, Space, Statistic, Tag, Typography, Progress, Table } from "antd";
import { useEffect, useState } from "react";
import { getHealth } from "../services/auth";
import { getOverview, getOverviewTrends } from "../services/overview";
import type { OverviewTrendsResponse } from "../services/overview";
import { LineChart } from "../components/line-chart";
import { getOperationLogs, type OperationLogItem } from "../services/operation-logs";
import { getLoginLogs, type LoginLogItem } from "../services/login-logs";
import { formatDateTime } from "../utils/format";

interface DashboardMetrics {
  users: number;
  clusters: number;
  pendingRegistrations: number;
  servers: number;
  podNormal: number;
  podAbnormal: number;
  podClusterErrors: number;
  eventTotal: number;
  eventWarning: number;
  eventClusterErrors: number;
}

interface SystemHealth {
  status: string;
  version: string;
  uptime: number;
  loading: boolean;
}

const defaultMetrics: DashboardMetrics = {
  users: 0,
  clusters: 0,
  pendingRegistrations: 0,
  servers: 0,
  podNormal: 0,
  podAbnormal: 0,
  podClusterErrors: 0,
  eventTotal: 0,
  eventWarning: 0,
  eventClusterErrors: 0,
};

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
    key: "pendingRegistrations",
    title: "待审核",
    hint: "待审批的注册申请数量",
    icon: <SafetyCertificateOutlined />,
    color: "#c96a11",
    gradient: "linear-gradient(135deg, #c96a11 0%, #fa8c16 100%)",
  },
  {
    key: "servers",
    title: "服务器",
    hint: "纳管的服务器数量",
    icon: <DesktopOutlined />,
    color: "#7a4dd8",
    gradient: "linear-gradient(135deg, #7a4dd8 0%, #a855f7 100%)",
  },
] as const;

function formatUptime(seconds: number): string {
  const hours = Math.floor(seconds / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);
  return `${hours}小时 ${minutes}分钟`;
}

export function DashboardPage() {
  const [metrics, setMetrics] = useState<DashboardMetrics>(defaultMetrics);
  const [health, setHealth] = useState<SystemHealth>({ status: "", version: "", uptime: 0, loading: true });
  const [loading, setLoading] = useState(true);
  const [trends, setTrends] = useState<OverviewTrendsResponse | null>(null);
  const [recentOps, setRecentOps] = useState<OperationLogItem[]>([]);
  const [recentLogins, setRecentLogins] = useState<LoginLogItem[]>([]);

  useEffect(() => {
    let active = true;

    async function load() {
      setLoading(true);
      setHealth((prev) => ({ ...prev, loading: true }));
      try {
        const [overview, healthData, trendsData, ops, logins] = await Promise.all([
          getOverview().catch(() => null),
          getHealth().catch(() => null),
          getOverviewTrends().catch(() => null),
          getOperationLogs({ page: 1, page_size: 5 }).catch(() => null),
          getLoginLogs({ page: 1, page_size: 5 }).catch(() => null),
        ]);

        if (!active) {
          return;
        }

        if (overview) {
          setMetrics({
            users: overview.users_count,
            clusters: overview.clusters_count,
            pendingRegistrations: overview.pending_registrations_count,
            servers: overview.servers_count,
            podNormal: overview.pod_normal_count,
            podAbnormal: overview.pod_abnormal_count,
            podClusterErrors: overview.pod_cluster_errors,
            eventTotal: overview.event_total_count,
            eventWarning: overview.event_warning_count,
            eventClusterErrors: overview.event_cluster_errors,
          });
        } else {
          setMetrics(defaultMetrics);
        }

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

        setTrends(trendsData);
        setRecentOps(ops?.list || []);
        setRecentLogins(logins?.list || []);
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
          <Col xs={24} sm={12} lg={8} xl={6} key={item.key}>
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

      <Row gutter={[20, 20]} style={{ marginBottom: 24 }}>
        <Col xs={24}>
          <Card
            className="table-card"
            title={
              <Space>
                <InfoOutlined style={{ color: "#0f6cbd" }} />
                平台活跃趋势（近 7 天）
              </Space>
            }
            loading={loading && !trends}
          >
            {trends ? (
              <LineChart
                labels={trends.days}
                series={[
                  { name: "登录成功", data: trends.login_success, color: "#2563eb" },
                  { name: "操作量", data: trends.operation_total, color: "#10b981" },
                  { name: "登录失败", data: trends.login_fail, color: "#ef4444" },
                ]}
                height={240}
              />
            ) : (
              <Typography.Text type="secondary">暂无趋势数据（未产生登录/操作日志或服务未启用统计）。</Typography.Text>
            )}
          </Card>
        </Col>
      </Row>

      <Row gutter={[20, 20]} style={{ marginBottom: 24 }}>
        <Col xs={24} lg={12}>
          <Card
            className="table-card"
            title={
              <Space>
                <ProfileOutlined style={{ color: "#10b981" }} />
                最近操作（Top 5）
              </Space>
            }
            loading={loading}
            bodyStyle={{ paddingTop: 12 }}
          >
            <Table<OperationLogItem>
              size="small"
              rowKey="id"
              dataSource={recentOps}
              pagination={false}
              tableLayout="fixed"
              columns={[
                { title: "用户", dataIndex: "username", width: 100, render: (v: string) => v || "-" },
                { title: "方法", dataIndex: "method", width: 80, render: (v: string) => <Tag>{v}</Tag> },
                {
                  title: "路径",
                  dataIndex: "path",
                  ellipsis: true,
                  render: (v: string) => (
                    <Typography.Text ellipsis={{ tooltip: v }} style={{ maxWidth: "100%" }}>
                      {v}
                    </Typography.Text>
                  ),
                },
                { title: "时间", dataIndex: "created_at", width: 160, render: formatDateTime },
              ]}
            />
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card
            className="table-card"
            title={
              <Space>
                <LoginOutlined style={{ color: "#2563eb" }} />
                最近登录（Top 5）
              </Space>
            }
            loading={loading}
            bodyStyle={{ paddingTop: 12 }}
          >
            <Table<LoginLogItem>
              size="small"
              rowKey="id"
              dataSource={recentLogins}
              pagination={false}
              tableLayout="fixed"
              columns={[
                { title: "用户", dataIndex: "username", width: 130, render: (v: string) => v || "-" },
                {
                  title: "状态",
                  dataIndex: "status",
                  width: 80,
                  render: (v: number) => (v === 1 ? <Tag color="success">成功</Tag> : <Tag color="error">失败</Tag>),
                },
                { title: "来源", dataIndex: "source", width: 110, render: (v: string) => <Tag>{v || "-"}</Tag> },
                { title: "时间", dataIndex: "created_at", width: 160, render: formatDateTime },
              ]}
            />
          </Card>
        </Col>
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
              Pod 和 Events 均按“已启用集群”聚合；Events 为每集群最近 500 条采样。未接入/未启用集群时将显示为 0，不代表系统异常。
            </Typography.Text>
            <Row gutter={16}>
              <Col span={12}>
                <Statistic title="正常 Pod" value={metrics.podNormal} valueStyle={{ color: "#14804a", fontWeight: 700 }} />
              </Col>
              <Col span={12}>
                <Statistic title="异常 Pod" value={metrics.podAbnormal} valueStyle={{ color: "#c23a2b", fontWeight: 700 }} />
              </Col>
            </Row>
            {metrics.clusters > 0 && (metrics.podClusterErrors > 0 || metrics.eventClusterErrors > 0) ? (
              <div style={{ display: "flex", justifyContent: "space-between" }}>
                <Typography.Text type="secondary">采集失败集群数（Pod / Event）</Typography.Text>
                <Tag color="warning">
                  {metrics.podClusterErrors} / {metrics.eventClusterErrors}
                </Tag>
              </div>
            ) : null}
            <Row gutter={16}>
              <Col span={12}>
                <Statistic title="Events 总数" value={metrics.eventTotal} valueStyle={{ color: "#7a4dd8", fontWeight: 700 }} />
              </Col>
              <Col span={12}>
                <Statistic title="Warning Events" value={metrics.eventWarning} valueStyle={{ color: "#d4380d", fontWeight: 700 }} />
              </Col>
            </Row>
            {metrics.clusters === 0 ? (
              <Tag color="default">未接入集群</Tag>
            ) : null}
          </Space>
        </Card>
      </div>
    </div>
  );
}
