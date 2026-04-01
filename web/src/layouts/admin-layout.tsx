import {
  ApiOutlined,
  ApartmentOutlined,
  AuditOutlined,
  DatabaseOutlined,
  LogoutOutlined,
  TeamOutlined,
} from "@ant-design/icons";
import { Avatar, Button, Dropdown, Layout, Menu, Space, Spin, Tag, Typography } from "antd";
import { Link, Outlet, useLocation, useNavigate } from "react-router-dom";
import {
  BRAND_DESCRIPTION,
  BRAND_EN_NAME,
  BRAND_NAME,
  BRAND_SHORT,
  BRAND_SUBTITLE,
} from "../constants/brand";
import { useAuth } from "../contexts/auth-context";

const { Content, Header, Sider } = Layout;

const menuItems = [
  {
    key: "/",
    icon: <DatabaseOutlined />,
    label: <Link to="/">资产总览</Link>,
  },
  {
    key: "/users",
    icon: <TeamOutlined />,
    label: <Link to="/users">账号治理</Link>,
  },
  {
    key: "/roles",
    icon: <ApartmentOutlined />,
    label: <Link to="/roles">角色模板</Link>,
  },
  {
    key: "/permissions",
    icon: <ApiOutlined />,
    label: <Link to="/permissions">接口能力</Link>,
  },
  {
    key: "/policies",
    icon: <AuditOutlined />,
    label: <Link to="/policies">授权编排</Link>,
  },
];

const titleMap: Record<string, string> = {
  "/": "资产总览",
  "/users": "账号治理",
  "/roles": "角色模板",
  "/permissions": "接口能力",
  "/policies": "授权编排",
};

const capabilityTags = ["Redis 验证码登录", "Casbin 授权编排", "运维资产治理"];

export function AdminLayout() {
  const { pathname } = useLocation();
  const navigate = useNavigate();
  const { user, loading, logoutAction } = useAuth();

  let selectedKey = "/";
  if (pathname !== "/") {
    const matched = menuItems.find((item) => item.key !== "/" && pathname.startsWith(item.key));
    selectedKey = matched?.key ?? "/";
  }

  async function handleLogout() {
    await logoutAction();
    navigate("/login", { replace: true });
  }

  const dropdownItems = [
    {
      key: "logout",
      icon: <LogoutOutlined />,
      label: "退出登录",
      onClick: handleLogout,
    },
  ];

  return (
    <Layout className="admin-shell">
      <Sider width={288} className="admin-sider" breakpoint="lg" collapsedWidth={0}>
        <div className="brand-block">
          <div className="brand-block__mark">{BRAND_SHORT}</div>
          <div>
            <Typography.Text className="brand-block__eyebrow">{BRAND_SUBTITLE}</Typography.Text>
            <Typography.Title level={4} className="brand-block__title">
              {BRAND_NAME}
            </Typography.Title>
            <Typography.Text className="brand-block__subtitle">{BRAND_DESCRIPTION}</Typography.Text>
          </div>
        </div>

        <div className="brand-block__hint">
          <Typography.Text>{BRAND_EN_NAME}</Typography.Text>
          <Typography.Paragraph className="brand-block__hint-copy">
            统一承接账号主体、角色模板、接口能力和授权策略，适合做运维 CMDB 的权限治理底座。
          </Typography.Paragraph>
        </div>

        <div className="brand-pills">
          <div className="brand-pill">
            <span className="brand-pill__label">登录方式</span>
            <strong>6 位验证码</strong>
          </div>
          <div className="brand-pill">
            <span className="brand-pill__label">授权引擎</span>
            <strong>JWT + Casbin</strong>
          </div>
        </div>

        <Menu theme="dark" mode="inline" selectedKeys={[selectedKey]} items={menuItems} className="admin-menu" />
      </Sider>

      <Layout>
        <Header className="admin-header">
          <div>
            <Typography.Text className="admin-header__eyebrow">{BRAND_SUBTITLE}</Typography.Text>
            <Typography.Title level={3} className="admin-header__title">
              {titleMap[selectedKey] ?? BRAND_NAME}
            </Typography.Title>
            <Typography.Text className="admin-header__desc">
              当前工作台聚焦账号、授权与接口能力的统一治理，适合运维侧持续扩展。
            </Typography.Text>
          </div>

          <div className="admin-header__actions">
            <Space wrap className="admin-header__meta">
              {capabilityTags.map((item) => (
                <Tag key={item} bordered={false} className="admin-header__status">
                  {item}
                </Tag>
              ))}
            </Space>

            <Dropdown menu={{ items: dropdownItems }} trigger={["click"]}>
              <Button size="large" className="user-chip">
                <Space>
                  <Avatar>{user?.nickname?.slice(0, 1) || user?.username?.slice(0, 1) || "Y"}</Avatar>
                  <span>{user?.nickname || user?.username || "未登录"}</span>
                  {user?.roles?.[0] ? <Tag color="blue">{user.roles[0].name}</Tag> : null}
                </Space>
              </Button>
            </Dropdown>
          </div>
        </Header>

        <Content className="admin-content">
          {loading ? (
            <div className="page-loading">
              <Spin size="large" />
            </div>
          ) : (
            <Outlet />
          )}
        </Content>
      </Layout>
    </Layout>
  );
}
