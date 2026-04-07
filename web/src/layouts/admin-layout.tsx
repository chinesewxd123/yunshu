import {
  ApiOutlined,
  ApartmentOutlined,
  AuditOutlined,
  CheckCircleOutlined,
  KubernetesOutlined,
  DatabaseOutlined,
  HistoryOutlined,
  LoginOutlined,
  MenuOutlined,
  TeamOutlined,
  LogoutOutlined,
  DownOutlined,
} from "@ant-design/icons";
import type { MenuProps } from "antd";
import { Avatar, Button, Dropdown, Layout, Menu, Spin, Tag, Typography } from "antd";
import { useEffect, useMemo, useState } from "react";
import { Link, Outlet, useLocation, useNavigate } from "react-router-dom";
import { BRAND_DESCRIPTION, BRAND_EN_NAME, BRAND_NAME, BRAND_SHORT, BRAND_SUBTITLE } from "../constants/brand";
import { useAuth } from "../contexts/auth-context";
import { getMenuTree } from "../services/menus";
import type { MenuItem } from "../services/menus";
import { buildSiderMenuItems, matchMenuSelectedKey, type AntdMenuItem } from "../utils/admin-menu";

const { Content, Header, Sider } = Layout;

const FALLBACK_MENU_ITEMS: MenuProps["items"] = [
  { key: "/", icon: <DatabaseOutlined />, label: <Link to="/">资产总览</Link> },
  { key: "/clusters", icon: <KubernetesOutlined />, label: <Link to="/clusters">集群管理</Link> },
  { key: "/pods", icon: <KubernetesOutlined />, label: <Link to="/pods">Pod 管理</Link> },
  { key: "/users", icon: <TeamOutlined />, label: <Link to="/users">账号管理</Link> },
  { key: "/roles", icon: <ApartmentOutlined />, label: <Link to="/roles">角色管理</Link> },
  { key: "/permissions", icon: <ApiOutlined />, label: <Link to="/permissions">API管理</Link> },
  { key: "/policies", icon: <AuditOutlined />, label: <Link to="/policies">授权管理</Link> },
  { key: "/registrations", icon: <CheckCircleOutlined />, label: <Link to="/registrations">注册审核</Link> },
  { key: "/menus", icon: <MenuOutlined />, label: <Link to="/menus">菜单管理</Link> },
  {
    key: "/system",
    icon: <MenuOutlined />,
    label: "系统管理",
    children: [
      { key: "/login-logs", icon: <LoginOutlined />, label: <Link to="/login-logs">登录日志</Link> },
      { key: "/operation-logs", icon: <HistoryOutlined />, label: <Link to="/operation-logs">操作历史</Link> },
      { key: "/banned-ips", icon: <ApiOutlined />, label: <Link to="/banned-ips">封禁 IP 管理</Link> },
    ],
  },
];

function defaultOpenKeysFor(items: AntdMenuItem[]): string[] {
  const keys: string[] = [];
  function walk(list: AntdMenuItem[]) {
    for (const it of list) {
      if (it && typeof it === "object" && "children" in it && Array.isArray(it.children) && it.children.length > 0) {
        keys.push(String(it.key));
        walk(it.children as AntdMenuItem[]);
      }
    }
  }
  walk(items);
  return keys;
}

export function AdminLayout() {
  const { pathname } = useLocation();
  const navigate = useNavigate();
  const { user, loading, logoutAction } = useAuth();
  const [siderItems, setSiderItems] = useState<MenuProps["items"]>(FALLBACK_MENU_ITEMS);
  const [menuEpoch, setMenuEpoch] = useState(0);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        const tree: MenuItem[] = await getMenuTree();
        if (cancelled || !tree?.length) return;
        const items = buildSiderMenuItems(tree);
        if (!items.length) return;
        setSiderItems(items);
        setMenuEpoch((n) => n + 1);
      } catch {
        setSiderItems(FALLBACK_MENU_ITEMS);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const selectedKey = useMemo(() => {
    const items = (siderItems ?? []) as AntdMenuItem[];
    return matchMenuSelectedKey(pathname, items);
  }, [pathname, siderItems]);

  const defaultOpenKeys = useMemo(
    () => defaultOpenKeysFor((siderItems ?? []) as AntdMenuItem[]),
    [siderItems],
  );

  async function handleLogout() {
    await logoutAction();
    navigate("/login", { replace: true });
  }

  const userMenuItems: MenuProps["items"] = [
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
            侧栏数据来自「菜单管理」与 /api/v1/menus/tree；隐藏或停用的项不会显示。页面访问仍受 Casbin 与 JWT 约束。
          </Typography.Paragraph>
        </div>

        <Menu
          key={menuEpoch}
          theme="dark"
          mode="inline"
          selectedKeys={[selectedKey]}
          defaultOpenKeys={defaultOpenKeys}
          items={siderItems}
          className="admin-menu"
        />
      </Sider>

      <Layout>
        <Header className="admin-header">
          <Dropdown
            trigger={["click"]}
            placement="bottomRight"
            menu={{ items: userMenuItems }}
            dropdownRender={(menu) => (
              <div className="user-header-dropdown-panel">
                <div className="user-header-dropdown-panel__head">
                  <Avatar className="user-header-dropdown-panel__avatar" size={40}>
                    {user?.nickname?.slice(0, 1) || user?.username?.slice(0, 1) || "Y"}
                  </Avatar>
                  <div className="user-header-dropdown-panel__text">
                    <div className="user-header-dropdown-panel__name">{user?.nickname || user?.username || "未登录"}</div>
                    {user?.username ? (
                      <div className="user-header-dropdown-panel__account">{user.username}</div>
                    ) : null}
                  </div>
                </div>
                {user?.roles?.length ? (
                  <div className="user-header-dropdown-panel__roles">
                    {user.roles.map((r) => (
                      <Tag key={r.id} color="blue" bordered={false}>
                        {r.name}
                      </Tag>
                    ))}
                  </div>
                ) : null}
                <div className="user-header-dropdown-panel__menu">{menu}</div>
              </div>
            )}
          >
            <Button type="text" className="user-header-trigger">
              <span className="user-header-trigger__inner">
                <Avatar className="user-header-trigger__avatar" size="small">
                  {user?.nickname?.slice(0, 1) || user?.username?.slice(0, 1) || "Y"}
                </Avatar>
                <span className="user-header-trigger__name">{user?.nickname || user?.username || "未登录"}</span>
                <DownOutlined className="user-header-trigger__caret" />
              </span>
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
