import {
  AuditOutlined,
  DashboardOutlined,
  LockOutlined,
  LogoutOutlined,
  SafetyOutlined,
  TeamOutlined,
} from "@ant-design/icons";
import { Avatar, Button, Dropdown, Layout, Menu, Space, Spin, Tag, Typography } from "antd";
import { Link, Outlet, useLocation, useNavigate } from "react-router-dom";
import { useAuth } from "../contexts/auth-context";

const { Content, Header, Sider } = Layout;

const menuItems = [
  {
    key: "/",
    icon: <DashboardOutlined />,
    label: <Link to="/">概览</Link>,
  },
  {
    key: "/users",
    icon: <TeamOutlined />,
    label: <Link to="/users">用户管理</Link>,
  },
  {
    key: "/roles",
    icon: <SafetyOutlined />,
    label: <Link to="/roles">角色管理</Link>,
  },
  {
    key: "/permissions",
    icon: <LockOutlined />,
    label: <Link to="/permissions">权限管理</Link>,
  },
  {
    key: "/policies",
    icon: <AuditOutlined />,
    label: <Link to="/policies">策略绑定</Link>,
  },
];

const titleMap: Record<string, string> = {
  "/": "概览",
  "/users": "用户管理",
  "/roles": "角色管理",
  "/permissions": "权限管理",
  "/policies": "策略绑定",
};

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
      <Sider width={264} className="admin-sider" breakpoint="lg" collapsedWidth={0}>
        <div className="brand-block">
          <div className="brand-block__mark">PS</div>
          <div>
            <Typography.Title level={4} className="brand-block__title">
              Permission System
            </Typography.Title>
            <Typography.Text className="brand-block__subtitle">React + Gin 管理后台</Typography.Text>
          </div>
        </div>
        <div className="brand-block__hint">统一管理用户、角色、权限与 Casbin 策略。</div>
        <Menu theme="dark" mode="inline" selectedKeys={[selectedKey]} items={menuItems} className="admin-menu" />
      </Sider>
      <Layout>
        <Header className="admin-header">
          <div>
            <Typography.Text className="admin-header__eyebrow">控制台</Typography.Text>
            <Typography.Title level={4} className="admin-header__title">
              {titleMap[selectedKey] ?? "权限管理系统"}
            </Typography.Title>
          </div>
          <Dropdown menu={{ items: dropdownItems }} trigger={["click"]}>
            <Button size="large" className="user-chip">
              <Space>
                <Avatar>{user?.nickname?.slice(0, 1) || user?.username?.slice(0, 1) || "A"}</Avatar>
                <span>{user?.nickname || user?.username || "未登录"}</span>
                {user?.roles?.[0] ? <Tag color="gold">{user.roles[0].name}</Tag> : null}
              </Space>
            </Button>
          </Dropdown>
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
