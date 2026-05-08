import {
  AlertOutlined,
  ApiOutlined,
  CalendarOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  CloudOutlined,
  ClusterOutlined,
  DashboardOutlined,
  DesktopOutlined,
  DisconnectOutlined,
  InfoOutlined,
  LoginOutlined,
  ProfileOutlined,
  SafetyCertificateOutlined,
  TeamOutlined,
  ThunderboltOutlined,
  WarningOutlined,
} from "@ant-design/icons";
import { Card, Col, Divider, Progress, Row, Space, Statistic, Table, Tag, Typography } from "antd";
import { useEffect, useMemo, useState } from "react";
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
  alertFiring: number;
  alertEventsToday: number;
  logAgentsOnline: number;
  logAgentsOffline: number;
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
  alertFiring: 0,
  alertEventsToday: 0,
  logAgentsOnline: 0,
  logAgentsOffline: 0,
};

const assetStats = [
  {
    key: "users",
    title: "账号主体",
    hint: "系统内已激活账号数量",
    icon: <TeamOutlined />,
    accent: "#22d3ee",
  },
  {
    key: "clusters",
    title: "K8s 集群",
    hint: "已注册且启用的集群",
    icon: <ClusterOutlined />,
    accent: "#38bdf8",
  },
  {
    key: "servers",
    title: "服务器",
    hint: "纳管日志源服务器",
    icon: <DesktopOutlined />,
    accent: "#a78bfa",
  },
  {
    key: "pendingRegistrations",
    title: "待审核注册",
    hint: "待审批的注册申请",
    icon: <SafetyCertificateOutlined />,
    accent: "#fbbf24",
  },
] as const;

const k8sStats = [
  {
    key: "podNormal",
    title: "正常 Pod",
    hint: "Running 且容器就绪",
    icon: <CheckCircleOutlined />,
    accent: "#34d399",
  },
  {
    key: "podAbnormal",
    title: "异常 Pod",
    hint: "非 Running 或未就绪",
    icon: <WarningOutlined />,
    accent: "#fb7185",
  },
  {
    key: "eventTotal",
    title: "Events 采样",
    hint: "各集群最近 500 条合计",
    icon: <CloudOutlined />,
    accent: "#818cf8",
  },
  {
    key: "eventWarning",
    title: "Warning",
    hint: "采样中的 Warning 类型",
    icon: <ThunderboltOutlined />,
    accent: "#f97316",
  },
] as const;

const alertAndAgentStats = [
  {
    key: "alertFiring",
    title: "告警 Firing",
    hint: "当前仍未恢复的告警事件",
    icon: <AlertOutlined />,
    accent: "#f87171",
  },
  {
    key: "alertEventsToday",
    title: "今日告警事件",
    hint: "本自然日新建事件条数",
    icon: <CalendarOutlined />,
    accent: "#fbbf24",
  },
  {
    key: "logAgentsOnline",
    title: "日志 Agent 在线",
    hint: "最近 90s 内有心跳",
    icon: <ApiOutlined />,
    accent: "#4ade80",
  },
  {
    key: "logAgentsOffline",
    title: "日志 Agent 离线",
    hint: "已启用但未满足在线条件",
    icon: <DisconnectOutlined />,
    accent: "#94a3b8",
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
            alertFiring: overview.alert_firing_count ?? 0,
            alertEventsToday: overview.alert_events_today_count ?? 0,
            logAgentsOnline: overview.log_agents_online_count ?? 0,
            logAgentsOffline: overview.log_agents_offline_count ?? 0,
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

  const podTotal = metrics.podNormal + metrics.podAbnormal;
  const podHealthyPct = useMemo(() => {
    if (podTotal <= 0) return 0;
    return Math.round((metrics.podNormal / podTotal) * 100);
  }, [metrics.podNormal, podTotal]);

  const warnRatio = useMemo(() => {
    if (metrics.eventTotal <= 0) return 0;
    return Math.round((metrics.eventWarning / metrics.eventTotal) * 100);
  }, [metrics.eventTotal, metrics.eventWarning]);

  const agentOnlinePct = useMemo(() => {
    const t = metrics.logAgentsOnline + metrics.logAgentsOffline;
    if (t <= 0) return 0;
    return Math.round((metrics.logAgentsOnline / t) * 100);
  }, [metrics.logAgentsOnline, metrics.logAgentsOffline]);

  const uptimePct = Math.min(100, (health.uptime / 86400) * 100);

  return (
    <div className="overview-big-screen">
      <div className="overview-big-screen__hero">
        <div>
          <Typography.Title level={3} className="overview-big-screen__title">
            <DashboardOutlined /> 运维运行总览
          </Typography.Title>
          <Typography.Text className="overview-big-screen__subtitle">
            资产底座 · Kubernetes · 告警与日志采集 · 活跃趋势 · 健康状态
          </Typography.Text>
        </div>
      </div>

      <Typography.Text className="overview-big-screen__section-label">
        <TeamOutlined /> 资产底座
      </Typography.Text>
      <Row gutter={[16, 16]} className="overview-big-screen__metrics">
        {assetStats.map((item) => (
          <Col xs={24} sm={12} xl={6} key={item.key}>
            <Card className="overview-big-screen__stat-card" loading={loading} bordered={false}>
              <div className="overview-big-screen__stat-head">
                <span className="overview-big-screen__stat-icon" style={{ boxShadow: `0 0 24px ${item.accent}44` }}>
                  {item.icon}
                </span>
                <Statistic
                  title={<span className="overview-big-screen__stat-title">{item.title}</span>}
                  value={metrics[item.key as keyof DashboardMetrics] as number}
                  valueStyle={{
                    color: item.accent,
                    fontSize: 36,
                    fontWeight: 700,
                    fontFamily: "var(--overview-num-font, ui-monospace, monospace)",
                  }}
                />
              </div>
              <Typography.Paragraph className="overview-big-screen__stat-hint">{item.hint}</Typography.Paragraph>
            </Card>
          </Col>
        ))}
      </Row>

      <Typography.Text className="overview-big-screen__section-label">
        <ClusterOutlined /> Kubernetes 运行态势
      </Typography.Text>
      <Row gutter={[16, 16]} className="overview-big-screen__metrics">
        {k8sStats.map((item) => (
          <Col xs={24} sm={12} xl={6} key={item.key}>
            <Card className="overview-big-screen__stat-card overview-big-screen__stat-card--k8s" loading={loading} bordered={false}>
              <div className="overview-big-screen__stat-head">
                <span className="overview-big-screen__stat-icon" style={{ boxShadow: `0 0 28px ${item.accent}55` }}>
                  {item.icon}
                </span>
                <Statistic
                  title={<span className="overview-big-screen__stat-title">{item.title}</span>}
                  value={metrics[item.key as keyof DashboardMetrics] as number}
                  valueStyle={{
                    color: item.accent,
                    fontSize: 34,
                    fontWeight: 700,
                    fontFamily: "var(--overview-num-font, ui-monospace, monospace)",
                  }}
                />
              </div>
              <Typography.Paragraph className="overview-big-screen__stat-hint">{item.hint}</Typography.Paragraph>
            </Card>
          </Col>
        ))}
      </Row>

      <Typography.Text className="overview-big-screen__section-label">
        <AlertOutlined /> 告警与日志采集
      </Typography.Text>
      <Row gutter={[16, 16]} className="overview-big-screen__metrics">
        {alertAndAgentStats.map((item) => (
          <Col xs={24} sm={12} xl={6} key={item.key}>
            <Card className="overview-big-screen__stat-card overview-big-screen__stat-card--alert" loading={loading} bordered={false}>
              <div className="overview-big-screen__stat-head">
                <span className="overview-big-screen__stat-icon" style={{ boxShadow: `0 0 28px ${item.accent}55` }}>
                  {item.icon}
                </span>
                <Statistic
                  title={<span className="overview-big-screen__stat-title">{item.title}</span>}
                  value={metrics[item.key as keyof DashboardMetrics] as number}
                  valueStyle={{
                    color: item.accent,
                    fontSize: 34,
                    fontWeight: 700,
                    fontFamily: "var(--overview-num-font, ui-monospace, monospace)",
                  }}
                />
              </div>
              <Typography.Paragraph className="overview-big-screen__stat-hint">{item.hint}</Typography.Paragraph>
            </Card>
          </Col>
        ))}
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 8 }}>
        <Col xs={24} xl={15}>
          <Card
            className="overview-big-screen__panel"
            title={
              <Space>
                <InfoOutlined style={{ color: "#38bdf8" }} />
                <span>平台活跃趋势（近 7 天）</span>
              </Space>
            }
            loading={loading && !trends}
            bordered={false}
          >
            {trends ? (
              <LineChart
                darkMode
                labels={trends.days}
                series={[
                  { name: "登录成功", data: trends.login_success, color: "#38bdf8" },
                  { name: "操作量", data: trends.operation_total, color: "#34d399" },
                  { name: "登录失败", data: trends.login_fail, color: "#f87171" },
                ]}
                height={300}
              />
            ) : (
              <Typography.Text type="secondary" style={{ color: "rgba(186, 214, 238, 0.65)" }}>
                暂无趋势数据（未产生登录/操作日志或服务未启用统计）。
              </Typography.Text>
            )}
          </Card>
        </Col>
        <Col xs={24} xl={9}>
          <Space direction="vertical" size={16} style={{ width: "100%" }}>
            <Card className="overview-big-screen__panel" title={<Space><CheckCircleOutlined /> 系统状态</Space>} loading={health.loading} bordered={false}>
              <Row gutter={[12, 12]}>
                <Col span={24}>
                  <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", flexWrap: "wrap", gap: 8 }}>
                    <Typography.Text style={{ color: "rgba(186, 214, 238, 0.85)" }}>服务</Typography.Text>
                    {health.status === "ok" || health.status === "healthy" ? (
                      <Tag color="success">运行正常</Tag>
                    ) : health.status === "error" ? (
                      <Tag color="error">连接异常</Tag>
                    ) : (
                      <Tag color="warning">{health.status}</Tag>
                    )}
                  </div>
                </Col>
                <Col span={24}>
                  <div style={{ display: "flex", justifyContent: "space-between", gap: 8, flexWrap: "wrap" }}>
                    <Typography.Text style={{ color: "rgba(186, 214, 238, 0.85)" }}>版本</Typography.Text>
                    <Tag style={{ marginInlineEnd: 0 }}>
                      <InfoOutlined /> {health.version}
                    </Tag>
                  </div>
                </Col>
                <Col span={24}>
                  <Typography.Text style={{ color: "rgba(186, 214, 238, 0.85)", display: "block", marginBottom: 8 }}>
                    运行时间 · {formatUptime(health.uptime)}
                  </Typography.Text>
                  <Progress
                    percent={uptimePct}
                    strokeColor={{ "0%": "#38bdf8", "100%": "#34d399" }}
                    trailColor="rgba(148, 163, 184, 0.15)"
                    format={(p) => `${Math.floor(((p ?? 0) / 100) * 24)}h / 24h`}
                  />
                </Col>
              </Row>
            </Card>

            <Card className="overview-big-screen__panel" title={<Space><ThunderboltOutlined /> Pod / Event 摘要</Space>} loading={loading} bordered={false}>
              <Space direction="vertical" size={12} style={{ width: "100%" }}>
                <div>
                  <Typography.Text style={{ color: "rgba(186, 214, 238, 0.85)" }}>Pod 健康占比</Typography.Text>
                  <Progress
                    percent={metrics.clusters === 0 ? 0 : podHealthyPct}
                    strokeColor="#34d399"
                    trailColor="rgba(251, 113, 133, 0.35)"
                    format={() => (metrics.clusters === 0 ? "未接入集群" : `${podHealthyPct}%`)}
                  />
                </div>
                <div>
                  <Typography.Text style={{ color: "rgba(186, 214, 238, 0.85)" }}>Warning 占采样 Events</Typography.Text>
                  <Progress
                    percent={metrics.eventTotal === 0 ? 0 : warnRatio}
                    strokeColor="#f97316"
                    trailColor="rgba(148, 163, 184, 0.2)"
                    format={() => (metrics.eventTotal === 0 ? "无采样" : `${warnRatio}%`)}
                  />
                </div>
                <div>
                  <Typography.Text style={{ color: "rgba(186, 214, 238, 0.85)" }}>日志 Agent 在线占比</Typography.Text>
                  <Progress
                    percent={metrics.logAgentsOnline + metrics.logAgentsOffline === 0 ? 0 : agentOnlinePct}
                    strokeColor="#4ade80"
                    trailColor="rgba(148, 163, 184, 0.2)"
                    format={() =>
                      metrics.logAgentsOnline + metrics.logAgentsOffline === 0 ? "无启用 Agent" : `${agentOnlinePct}%`
                    }
                  />
                </div>
                <Divider style={{ borderColor: "rgba(56, 189, 248, 0.15)", margin: "8px 0" }} />
                {metrics.clusters > 0 && (metrics.podClusterErrors > 0 || metrics.eventClusterErrors > 0) ? (
                  <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
                    <Typography.Text style={{ color: "rgba(248, 250, 252, 0.75)" }}>采集失败集群（Pod / Event）</Typography.Text>
                    <Tag color="warning">
                      {metrics.podClusterErrors} / {metrics.eventClusterErrors}
                    </Tag>
                  </div>
                ) : (
                  <Typography.Text style={{ color: "rgba(186, 214, 238, 0.55)", fontSize: 12 }}>
                    Events 为每集群最近 500 条采样；未接入集群时上方 K8s 指标可能为 0。
                  </Typography.Text>
                )}
              </Space>
            </Card>
          </Space>
        </Col>
      </Row>

      <Row gutter={[16, 16]} style={{ marginTop: 16 }}>
        <Col xs={24} lg={12}>
          <Card
            className="overview-big-screen__panel overview-big-screen__table-card"
            title={
              <Space>
                <ProfileOutlined style={{ color: "#34d399" }} />
                最近操作
              </Space>
            }
            loading={loading}
            bordered={false}
          >
            <Table<OperationLogItem>
              size="small"
              rowKey="id"
              dataSource={recentOps}
              pagination={false}
              tableLayout="fixed"
              columns={[
                { title: "用户", dataIndex: "username", width: 88, render: (v: string) => v || "-" },
                { title: "方法", dataIndex: "method", width: 72, render: (v: string) => <Tag bordered={false}>{v}</Tag> },
                {
                  title: "路径",
                  dataIndex: "path",
                  ellipsis: true,
                  render: (v: string) => (
                    <Typography.Text ellipsis={{ tooltip: v }} style={{ maxWidth: "100%", color: "rgba(241, 245, 249, 0.92)" }}>
                      {v}
                    </Typography.Text>
                  ),
                },
                { title: "时间", dataIndex: "created_at", width: 152, render: formatDateTime },
              ]}
            />
          </Card>
        </Col>
        <Col xs={24} lg={12}>
          <Card
            className="overview-big-screen__panel overview-big-screen__table-card"
            title={
              <Space>
                <LoginOutlined style={{ color: "#38bdf8" }} />
                最近登录
              </Space>
            }
            loading={loading}
            bordered={false}
          >
            <Table<LoginLogItem>
              size="small"
              rowKey="id"
              dataSource={recentLogins}
              pagination={false}
              tableLayout="fixed"
              columns={[
                { title: "用户", dataIndex: "username", width: 120, render: (v: string) => v || "-" },
                {
                  title: "状态",
                  dataIndex: "status",
                  width: 72,
                  render: (v: number) => (v === 1 ? <Tag color="success">成功</Tag> : <Tag color="error">失败</Tag>),
                },
                { title: "来源", dataIndex: "source", width: 96, render: (v: string) => <Tag bordered={false}>{v || "-"}</Tag> },
                { title: "时间", dataIndex: "created_at", width: 152, render: formatDateTime },
              ]}
            />
          </Card>
        </Col>
      </Row>
    </div>
  );
}
