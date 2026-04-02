import { ApiOutlined, ApartmentOutlined, AuditOutlined, TeamOutlined, CheckCircleOutlined, CloseCircleOutlined } from "@ant-design/icons";
import { Card, Collapse, List, Space, Table, Tag, Typography } from "antd";
import { useEffect, useState } from "react";
import { PageHero } from "../components/page-hero";
import { API_CATALOG_GROUPS } from "../constants/api-catalog";
import { BRAND_NAME } from "../constants/brand";
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
    hint: "对应 GET/POST/PUT/DELETE /api/v1/users*",
    icon: <TeamOutlined />,
    color: "#0f766e",
  },
  {
    key: "roles",
    title: "角色模板",
    hint: "对应 /api/v1/roles*，用于 Casbin 分组",
    icon: <ApartmentOutlined />,
    color: "#0f6cbd",
  },
  {
    key: "permissions",
    title: "接口能力",
    hint: "对应 /api/v1/permissions*，定义可授权资源",
    icon: <ApiOutlined />,
    color: "#c96a11",
  },
  {
    key: "policies",
    title: "授权编排",
    hint: "对应 /api/v1/policies，角色与能力绑定条数",
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
        subtitle={`${BRAND_NAME} 控制台与后端 /api/v1 路由一一对应：下方「接口目录」汇总全部已暴露接口及前端入口；能力项以 Gin 路由模板（含 :id）为准，须与 Casbin 校验路径一致。`}
        breadcrumbItems={[{ title: "控制台" }, { title: "资产总览" }]}
        extra={
          <Space wrap>
            <Tag color="processing">JWT + Redis 会话</Tag>
            <Tag color="gold">Casbin 授权</Tag>
            <Tag color="green">操作审计</Tag>
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

      <Card className="table-card" title="后端接口目录（与控制台映射）" style={{ marginTop: 24 }}>
        <Typography.Paragraph type="secondary" style={{ marginBottom: 16 }}>
          与 Gin 路由一致；Casbin 使用 FullPath 模板匹配。撤销授权为 <Typography.Text code>DELETE /api/v1/policies</Typography.Text>{" "}
          且请求体为 JSON，资源路径不要写成 <Typography.Text code>/api/v1/policies/:id</Typography.Text>。
        </Typography.Paragraph>
        <Collapse
          defaultActiveKey={API_CATALOG_GROUPS[0] ? [API_CATALOG_GROUPS[0].title] : []}
          items={API_CATALOG_GROUPS.map((group) => ({
            key: group.title,
            label: `${group.title}（${group.routes.length}）`,
            children: (
              <Table
                size="small"
                pagination={false}
                rowKey={(row) => `${row.method}-${row.path}`}
                dataSource={group.routes}
                columns={[
                  { title: "方法", dataIndex: "method", width: 72 },
                  {
                    title: "路径",
                    dataIndex: "path",
                    render: (value: string) => <Typography.Text code>{value}</Typography.Text>,
                  },
                  { title: "说明", dataIndex: "summary" },
                  { title: "前端入口 / 场景", dataIndex: "ui" },
                  {
                    title: "鉴权",
                    dataIndex: "auth",
                    width: 88,
                    render: (auth: boolean) =>
                      auth ? <Tag color="blue">Bearer</Tag> : <Tag color="default">公开</Tag>,
                  },
                ]}
              />
            ),
          }))}
        />
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
            <Typography.Title level={4}>文档与基路径</Typography.Title>
            <List
              dataSource={[
                "Swagger UI：/swagger/index.html",
                "OpenAPI JSON：/swagger/doc.json",
                "业务 API 前缀：/api/v1",
                "图形验证码：POST /api/v1/auth/password-login-code",
                "邮箱验证码：POST /api/v1/auth/verification-code",
              ]}
              renderItem={(item) => <List.Item>{item}</List.Item>}
            />
          </Card>

          <Card className="glass-card">
            <Typography.Title level={4}>治理建议</Typography.Title>
            <List
              dataSource={[
                "先维护「角色」模板，再在「账号管理」里绑定角色；无角色用户无法通过 Casbin 访问业务接口。",
                "「API 管理」中的资源路径须与后端路由模板一致（例如撤销授权为 DELETE /api/v1/policies，而非带 :id 的路径）。",
                "在「授权管理」中把能力项授予角色后策略立即生效；变更后可通过操作历史追溯。",
                "联调与回归可优先走 Swagger，再对照本页接口目录核对前端页面。",
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
