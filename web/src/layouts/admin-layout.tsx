import {
  ApiOutlined,
  ApartmentOutlined,
  AuditOutlined,
  CheckCircleOutlined,
  DatabaseOutlined,
  HistoryOutlined,
  LoginOutlined,
  MenuOutlined,
  TeamOutlined,
  LogoutOutlined,
} from "@ant-design/icons";
import type { MenuProps } from "antd";
import { Avatar, Button, Dropdown, Layout, Menu, Space, Spin, Tag, Typography } from "antd";
import { useEffect, useMemo, useState } from "react";
import { Link, Outlet, useLocation, useNavigate } from "react-router-dom";
import {
  BRAND_DESCRIPTION,
  BRAND_EN_NAME,
  BRAND_NAME,
  BRAND_SHORT,
  BRAND_SUBTITLE,
} from "../constants/brand";
import { useAuth } from "../contexts/auth-context";
import { getMenuTree } from "../services/menus";
import type { MenuItem } from "../services/menus";
import {
  buildSiderMenuItems,
  flattenMenuTitles,
  matchMenuSelectedKey,
  type AntdMenuItem,
} from "../utils/admin-menu";

const { Content, Header, Sider } = Layout;

const FALLBACK_MENU_ITEMS: MenuProps["items"] = [
  { key: "/", icon: <DatabaseOutlined />, label: <Link to="/">资产总览</Link> },
  { key: "/users", icon: <TeamOutlined />, label: <Link to="/users">账号管理</Link> },
  { key: "/roles", icon: <ApartmentOutlined />, label: <Link to="/roles">角色管理</Link> },
  { key: "/permissions", icon: <ApiOutlined />, label: <Link to="/permissions">API管理</Link> },
  { key: "/policies", icon: <AuditOutlined />, label: <Link to="/policies">授权管理</Link> },
  { key: "/registrations", icon: <CheckCircleOutlined />, label: <Link to="/registrations">注册审核</Link> },
  { key: "/menus", icon: <MenuOutlined />, label: <Link to="/menus">菜单管理</Link> },
  { key: "/login-logs", icon: <LoginOutlined />, label: <Link to="/login-logs">登录日志</Link> },
  { key: "/operation-logs", icon: <HistoryOutlined />, label: <Link to="/operation-logs">操作历史</Link> },
];

const FALLBACK_TITLE_MAP: Record<string, string> = {
  "/": "资产总览",
  "/users": "账号管理",
  "/roles": "角色管理",
  "/permissions": "API管理",
  "/policies": "授权管理",
  "/registrations": "注册审核",
  "/menus": "菜单管理",
  "/login-logs": "登录日志",
  "/operation-logs": "操作历史",
};

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

const capabilityTags = ["JWT + Casbin", "动态菜单 · /menus/tree", "登录/操作审计"];

export function AdminLayout() {
  const { pathname } = useLocation();
  const navigate = useNavigate();
  const { user, loading, logoutAction } = useAuth();
  const [siderItems, setSiderItems] = useState<MenuProps["items"]>(FALLBACK_MENU_ITEMS);
  const [titleMap, setTitleMap] = useState<Record<string, string>>(FALLBACK_TITLE_MAP);
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
        setTitleMap({ ...FALLBACK_TITLE_MAP, ...flattenMenuTitles(tree) });
        setMenuEpoch((n) => n + 1);
      } catch {
        setSiderItems(FALLBACK_MENU_ITEMS);
        setTitleMap(FALLBACK_TITLE_MAP);
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
            侧栏数据来自「菜单管理」与 /api/v1/menus/tree；隐藏或停用的项不会显示。页面访问仍受 Casbin 与 JWT 约束。
          </Typography.Paragraph>
        </div>

        <div className="brand-pills">
          <div className="brand-pill">
            <span className="brand-pill__label">登录</span>
            <strong>邮箱 6 位 / 密码+图形码</strong>
          </div>
          <div className="brand-pill">
            <span className="brand-pill__label">授权</span>
            <strong>角色 → 能力项</strong>
          </div>
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
          <div>
            <Typography.Text className="admin-header__eyebrow">{BRAND_SUBTITLE}</Typography.Text>
            <Typography.Title level={3} className="admin-header__title">
              {titleMap[selectedKey] ?? titleMap[pathname] ?? BRAND_NAME}
            </Typography.Title>
            <Typography.Text className="admin-header__desc">
              页面操作即调用后端接口；权限不足时请在「授权管理」中为当前角色勾选对应 API 能力项。
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
                  {user?.roles?.length ? (
                    <Space size={4} wrap>
                      {user.roles.slice(0, 3).map((r) => (
                        <Tag key={r.id} color="blue">
                          {r.name}
                        </Tag>
                      ))}
                      {user.roles.length > 3 ? <Tag>+{user.roles.length - 3}</Tag> : null}
                    </Space>
                  ) : null}
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
