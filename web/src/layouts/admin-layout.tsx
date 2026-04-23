import {
  ApiOutlined,
  CloudServerOutlined,
  ApartmentOutlined,
  AuditOutlined,
  BgColorsOutlined,
  BulbFilled,
  BulbOutlined,
  CheckCircleOutlined,
  CloseOutlined,
  FullscreenOutlined,
  KubernetesOutlined,
  DatabaseOutlined,
  HistoryOutlined,
  LoginOutlined,
  MenuOutlined,
  ReloadOutlined,
  SettingOutlined,
  TeamOutlined,
  UserOutlined,
  LogoutOutlined,
  DownOutlined,
} from "@ant-design/icons";
import type { MenuProps } from "antd";
import { Avatar, Button, Drawer, Dropdown, Layout, Menu, Space, Spin, Switch, Tabs, Tag, Typography } from "antd";
import { useEffect, useMemo, useState } from "react";
import { Link, Outlet, useLocation, useNavigate } from "react-router-dom";
import { BRAND_DESCRIPTION, BRAND_NAME, BRAND_SUBTITLE } from "../constants/brand";
import { useAuth } from "../contexts/auth-context";
import { getMenuTree } from "../services/menus";
import type { MenuItem } from "../services/menus";
import { buildSiderMenuItems, matchMenuSelectedKey, type AntdMenuItem } from "../utils/admin-menu";

const { Content, Header, Sider } = Layout;
const UI_PREFS_KEY = "admin-ui-preferences";

type UIPreferences = {
  showRefresh: boolean;
  showFullscreen: boolean;
  showThemeToggle: boolean;
  compactContent: boolean;
  darkSider: boolean;
  darkHeader: boolean;
};

const defaultUIPreferences: UIPreferences = {
  showRefresh: true,
  showFullscreen: true,
  showThemeToggle: true,
  compactContent: false,
  darkSider: true,
  darkHeader: true,
};

function loadUIPreferences(): UIPreferences {
  try {
    const raw = window.localStorage.getItem(UI_PREFS_KEY);
    if (!raw) return defaultUIPreferences;
    const parsed = JSON.parse(raw) as Partial<UIPreferences>;
    return { ...defaultUIPreferences, ...parsed };
  } catch {
    return defaultUIPreferences;
  }
}

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
      { key: "/departments", icon: <ApartmentOutlined />, label: <Link to="/departments">组织架构</Link> },
      { key: "/runtime-config", icon: <DatabaseOutlined />, label: <Link to="/runtime-config">配置中心</Link> },
      { key: "/dict-entries", icon: <DatabaseOutlined />, label: <Link to="/dict-entries">数据字典</Link> },
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
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [settingsTab, setSettingsTab] = useState("appearance");
  const [uiPreferences, setUIPreferences] = useState<UIPreferences>(() => loadUIPreferences());
  const [themeMode, setThemeMode] = useState<"dark" | "light">(() => {
    const saved = window.localStorage.getItem("admin-theme-mode");
    return saved === "light" ? "light" : "dark";
  });
  const [accent, setAccent] = useState<string>(() => {
    return window.localStorage.getItem("admin-theme-accent") ?? "#3f6dff";
  });

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
  const menuTheme = themeMode === "dark" ? "dark" : "light";
  const layoutClassName = [
    "admin-shell",
    themeMode === "dark" ? "theme-dark" : "theme-light",
    uiPreferences.compactContent ? "layout-compact" : "",
    uiPreferences.darkSider ? "layout-dark-sider" : "layout-soft-sider",
    uiPreferences.darkHeader ? "layout-dark-header" : "layout-soft-header",
  ]
    .filter(Boolean)
    .join(" ");

  const pageTitle = useMemo(() => {
    const titleMap: Record<string, string> = {
      "/": "分析页",
      "/users": "租户列表",
      "/departments": "组织架构",
      "/dict-entries": "数据字典",
      "/runtime-config": "配置中心",
      "/roles": "角色列表",
      "/permissions": "API 列表",
      "/policies": "策略列表",
      "/registrations": "注册审核",
      "/menus": "菜单列表",
      "/clusters": "集群列表",
      "/pods": "Pod 列表",
      "/login-logs": "登录日志",
      "/operation-logs": "操作日志",
      "/banned-ips": "黑名单列表",
      "/personal-settings": "个人设置",
    };
    return titleMap[pathname] ?? "";
  }, [pathname]);

  async function handleLogout() {
    await logoutAction();
    navigate("/login", { replace: true });
  }

  function handleRefresh() {
    window.location.reload();
  }

  async function handleToggleFullscreen() {
    try {
      if (document.fullscreenElement) {
        await document.exitFullscreen();
        return;
      }
      await document.documentElement.requestFullscreen();
    } catch {
      // Ignore unsupported fullscreen APIs.
    }
  }

  function applyThemeMode(mode: "dark" | "light") {
    setThemeMode(mode);
    window.localStorage.setItem("admin-theme-mode", mode);
    window.dispatchEvent(new CustomEvent("admin-theme-mode-change", { detail: { mode } }));
  }

  function handleToggleMode() {
    applyThemeMode(themeMode === "dark" ? "light" : "dark");
  }

  function applyAccent(next: string) {
    setAccent(next);
    window.localStorage.setItem("admin-theme-accent", next);
    document.documentElement.style.setProperty("--admin-accent", next);
    window.dispatchEvent(new CustomEvent("admin-theme-accent-change", { detail: { accent: next } }));
  }

  function patchUIPreferences(patch: Partial<UIPreferences>) {
    setUIPreferences((prev) => {
      const next = { ...prev, ...patch };
      window.localStorage.setItem(UI_PREFS_KEY, JSON.stringify(next));
      return next;
    });
  }

  async function handleClearCacheAndLogout() {
    const mode = window.localStorage.getItem("admin-theme-mode");
    const accent = window.localStorage.getItem("admin-theme-accent");
    window.localStorage.clear();
    if (mode) window.localStorage.setItem("admin-theme-mode", mode);
    if (accent) window.localStorage.setItem("admin-theme-accent", accent);
    await handleLogout();
  }

  async function handleCopyPreference() {
    const payload = JSON.stringify({ themeMode, accent, uiPreferences }, null, 2);
    try {
      await navigator.clipboard.writeText(payload);
    } catch {
      // Ignore clipboard permission issues.
    }
  }

  useEffect(() => {
    document.documentElement.style.setProperty("--admin-accent", accent);
  }, [accent]);

  const userMenuItems: MenuProps["items"] = [
    {
      key: "personal-settings",
      icon: <UserOutlined />,
      label: "个人设置",
      onClick: () => navigate("/personal-settings"),
    },
    {
      key: "logout",
      icon: <LogoutOutlined />,
      label: "退出登录",
      onClick: handleLogout,
    },
  ];

  return (
    <Layout className={layoutClassName}>
      <Sider width={288} className="admin-sider" breakpoint="lg" collapsedWidth={0}>
        <div className="brand-block">
          <div className="brand-block__mark">
            <CloudServerOutlined />
          </div>
          <div>
            <Typography.Text className="brand-block__eyebrow">{BRAND_SUBTITLE}</Typography.Text>
            <Typography.Title level={4} className="brand-block__title">
              {BRAND_NAME}
            </Typography.Title>
            <Typography.Text className="brand-block__subtitle">{BRAND_DESCRIPTION}</Typography.Text>
          </div>
        </div>

        <Menu
          key={menuEpoch}
          theme={menuTheme}
          mode="inline"
          selectedKeys={[selectedKey]}
          defaultOpenKeys={defaultOpenKeys}
          items={siderItems}
          className="admin-menu"
        />
      </Sider>

      <Layout>
        <Header className="admin-header">
          <div className="admin-header__left">
            <div className="admin-header__title">{pageTitle}</div>
          </div>
          <div className="admin-header__quick-actions">
            {uiPreferences.showRefresh ? (
              <Button type="text" className="admin-icon-btn" onClick={handleRefresh}>
                <ReloadOutlined />
              </Button>
            ) : null}
            {uiPreferences.showFullscreen ? (
              <Button type="text" className="admin-icon-btn" onClick={() => void handleToggleFullscreen()}>
                <FullscreenOutlined />
              </Button>
            ) : null}
            {uiPreferences.showThemeToggle ? (
              <Button type="text" className="admin-icon-btn" onClick={handleToggleMode} title="模式切换">
                {themeMode === "dark" ? <BulbOutlined /> : <BulbFilled />}
              </Button>
            ) : null}
            <Button type="text" className="admin-icon-btn" onClick={() => setSettingsOpen(true)} title="偏好设置">
              <SettingOutlined />
            </Button>
          </div>
          <div className="admin-header__user-wrap">
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
                  <Avatar className="user-header-trigger__avatar" size={30}>
                    {user?.nickname?.slice(0, 1) || user?.username?.slice(0, 1) || "Y"}
                  </Avatar>
                  <span className="user-header-trigger__name">{user?.nickname || user?.username || "未登录"}</span>
                  <DownOutlined className="user-header-trigger__caret" />
                </span>
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

        <Drawer
          width={360}
          open={settingsOpen}
          onClose={() => setSettingsOpen(false)}
          className={`admin-settings-drawer ${themeMode === "dark" ? "is-dark" : "is-light"}`}
          closeIcon={<CloseOutlined />}
          title="偏好设置"
          extra={
            <Button
              type="text"
              size="small"
              icon={themeMode === "dark" ? <BulbOutlined /> : <BulbFilled />}
              onClick={handleToggleMode}
            >
              {themeMode === "dark" ? "深色" : "浅色"}
            </Button>
          }
        >
          <Tabs
            activeKey={settingsTab}
            onChange={setSettingsTab}
            className="admin-settings-tabs"
            items={[
              {
                key: "appearance",
                label: "外观",
                children: (
                  <>
                    <div className="admin-settings-section">
                      <Typography.Text className="admin-settings-label">主题模式</Typography.Text>
                      <Space size={8} style={{ marginTop: 10 }}>
                        <Button
                          className={themeMode === "light" ? "is-active" : ""}
                          onClick={() => applyThemeMode("light")}
                          icon={<BulbFilled />}
                        >
                          浅色
                        </Button>
                        <Button
                          className={themeMode === "dark" ? "is-active" : ""}
                          onClick={() => applyThemeMode("dark")}
                          icon={<BulbOutlined />}
                        >
                          深色
                        </Button>
                      </Space>
                    </div>

                    <div className="admin-settings-section">
                      <Typography.Text className="admin-settings-label">内置主题色</Typography.Text>
                      <div className="admin-accent-grid">
                        {["#3f6dff", "#7c3aed", "#f43f5e", "#eab308", "#14b8a6", "#0ea5e9", "#6366f1", "#10b981", "#1d4ed8", "#f97316", "#dc2626", "#4b5563"].map((item) => (
                          <button
                            key={item}
                            type="button"
                            className={`admin-accent-dot ${accent === item ? "is-active" : ""}`}
                            style={{ background: item }}
                            onClick={() => applyAccent(item)}
                            aria-label={`切换主题色 ${item}`}
                          />
                        ))}
                      </div>
                    </div>
                  </>
                ),
              },
              {
                key: "layout",
                label: "布局",
                children: (
                  <div className="admin-settings-section">
                    <Typography.Text className="admin-settings-label">顶部快捷区</Typography.Text>
                    <div className="admin-setting-row">
                      <span>显示刷新按钮</span>
                      <Switch size="small" checked={uiPreferences.showRefresh} onChange={(checked) => patchUIPreferences({ showRefresh: checked })} />
                    </div>
                    <div className="admin-setting-row">
                      <span>显示全屏按钮</span>
                      <Switch size="small" checked={uiPreferences.showFullscreen} onChange={(checked) => patchUIPreferences({ showFullscreen: checked })} />
                    </div>
                    <div className="admin-setting-row">
                      <span>显示主题切换按钮</span>
                      <Switch size="small" checked={uiPreferences.showThemeToggle} onChange={(checked) => patchUIPreferences({ showThemeToggle: checked })} />
                    </div>
                    <div className="admin-setting-row">
                      <span>内容区紧凑模式</span>
                      <Switch size="small" checked={uiPreferences.compactContent} onChange={(checked) => patchUIPreferences({ compactContent: checked })} />
                    </div>
                    <div className="admin-setting-row">
                      <span>深色侧边栏</span>
                      <Switch size="small" checked={uiPreferences.darkSider} onChange={(checked) => patchUIPreferences({ darkSider: checked })} />
                    </div>
                    <div className="admin-setting-row">
                      <span>深色顶栏</span>
                      <Switch size="small" checked={uiPreferences.darkHeader} onChange={(checked) => patchUIPreferences({ darkHeader: checked })} />
                    </div>
                  </div>
                ),
              },
              {
                key: "shortcuts",
                label: "快捷键",
                children: (
                  <div className="admin-settings-section">
                    <div className="admin-setting-row">
                      <span>开启全局搜索快捷键</span>
                      <Switch size="small" checked disabled />
                    </div>
                    <div className="admin-setting-row">
                      <span>按键提示浮层</span>
                      <Switch size="small" checked disabled />
                    </div>
                  </div>
                ),
              },
              {
                key: "general",
                label: "通用",
                children: (
                  <div className="admin-settings-section">
                    <div className="admin-setting-row">
                      <span>自动保存主题偏好</span>
                      <Switch size="small" checked disabled />
                    </div>
                    <Space style={{ marginTop: 10 }}>
                      <Button type="primary" onClick={() => void handleCopyPreference()}>
                        复制偏好设置
                      </Button>
                      <Button danger onClick={() => void handleClearCacheAndLogout()}>
                        清空缓存 & 退出登录
                      </Button>
                    </Space>
                  </div>
                ),
              },
            ]}
          />

          <div className="admin-settings-tip">
            <BgColorsOutlined />
            <span>设置会自动保存，下次打开自动恢复。</span>
          </div>
        </Drawer>
      </Layout>
    </Layout>
  );
}
