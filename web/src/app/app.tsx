import { App as AntdApp, ConfigProvider, Spin, theme } from "antd";
import zhCN from "antd/locale/zh_CN";
import { Suspense, lazy, useEffect, useMemo, useState } from "react";
import { BrowserRouter, Navigate, Route, Routes, useLocation } from "react-router-dom";
import { AuthProvider, useAuth } from "../contexts/auth-context";
import { ErrorBoundary } from "../components/error-boundary";

const AdminLayout = lazy(() => import("../layouts/admin-layout").then((module) => ({ default: module.AdminLayout })));
const DashboardPage = lazy(() => import("../pages/dashboard-page").then((module) => ({ default: module.DashboardPage })));
const LoginPage = lazy(() => import("../pages/login-page").then((module) => ({ default: module.LoginPage })));
const PermissionsPage = lazy(() =>
  import("../pages/permissions-page").then((module) => ({ default: module.PermissionsPage })),
);
const PoliciesPage = lazy(() => import("../pages/policies-page").then((module) => ({ default: module.PoliciesPage })));
const RolesPage = lazy(() => import("../pages/roles-page").then((module) => ({ default: module.RolesPage })));
const UsersPage = lazy(() => import("../pages/users-page").then((module) => ({ default: module.UsersPage })));
const DepartmentsPage = lazy(() => import("../pages/departments-page").then((module) => ({ default: module.DepartmentsPage })));
const RegistrationsPage = lazy(() => import("../pages/registrations-page").then((module) => ({ default: module.RegistrationsPage })));
const MenusPage = lazy(() => import("../pages/menus-page").then((module) => ({ default: module.MenusPage })));
const LoginLogsPage = lazy(() => import("../pages/login-logs-page").then((module) => ({ default: module.LoginLogsPage })));
const OperationLogsPage = lazy(() =>
  import("../pages/operation-logs-page").then((module) => ({ default: module.OperationLogsPage })),
);
const BannedIPsPage = lazy(() => import("../pages/banned-ips-page").then((module) => ({ default: module.BannedIPsPage })));
const DynamicMenuPage = lazy(() => import("../pages/dynamic-menu-page").then((module) => ({ default: module.DynamicMenuPage })));
const ClusterPage = lazy(() => import("../pages/cluster-page").then((module) => ({ default: module.ClusterPage })));
const PodPage = lazy(() => import("../pages/pod-page").then((module) => ({ default: module.PodPage })));
const ServerConsolePage = lazy(() => import("../pages/server-console-page").then((module) => ({ default: module.ServerConsolePage })));
const PersonalSettingsPage = lazy(() =>
  import("../pages/personal-settings-page").then((module) => ({ default: module.PersonalSettingsPage })),
);

function RouteFallback() {
  return (
    <div className="full-screen-loading">
      <Spin size="large" />
    </div>
  );
}

function ProtectedRoutes() {
  const { isAuthenticated, loading } = useAuth();
  const location = useLocation();

  if (loading) {
    return <RouteFallback />;
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />;
  }

  return <AdminLayout />;
}

function AuthRoutes() {
  const { isAuthenticated, loading } = useAuth();

  if (loading) {
    return <RouteFallback />;
  }

  if (isAuthenticated) {
    return <Navigate to="/" replace />;
  }

  return <LoginPage />;
}

export function App() {
  const [mode, setMode] = useState<"dark" | "light">(() => {
    const saved = window.localStorage.getItem("admin-theme-mode");
    return saved === "light" ? "light" : "dark";
  });
  const [accent, setAccent] = useState<string>(() => {
    return window.localStorage.getItem("admin-theme-accent") ?? "#3f6dff";
  });

  useEffect(() => {
    const onModeChange = (event: Event) => {
      const detail = (event as CustomEvent<{ mode?: "dark" | "light" }>).detail;
      const next = detail?.mode === "light" ? "light" : "dark";
      setMode(next);
    };
    const onStorage = (event: StorageEvent) => {
      if (event.key === "admin-theme-mode") {
        setMode(event.newValue === "light" ? "light" : "dark");
      }
      if (event.key === "admin-theme-accent" && event.newValue) {
        setAccent(event.newValue);
      }
    };
    const onAccentChange = (event: Event) => {
      const detail = (event as CustomEvent<{ accent?: string }>).detail;
      if (!detail?.accent) return;
      setAccent(detail.accent);
    };
    window.addEventListener("admin-theme-mode-change", onModeChange as EventListener);
    window.addEventListener("admin-theme-accent-change", onAccentChange as EventListener);
    window.addEventListener("storage", onStorage);
    return () => {
      window.removeEventListener("admin-theme-mode-change", onModeChange as EventListener);
      window.removeEventListener("admin-theme-accent-change", onAccentChange as EventListener);
      window.removeEventListener("storage", onStorage);
    };
  }, []);

  const isDark = mode === "dark";
  const algorithm = useMemo(() => (isDark ? theme.darkAlgorithm : theme.defaultAlgorithm), [isDark]);

  return (
    <ConfigProvider
      locale={zhCN}
      theme={{
        algorithm,
        token: {
          colorPrimary: accent,
          colorSuccess: "#22c55e",
          colorWarning: "#f59e0b",
          colorError: "#ef4444",
          borderRadius: 10,
          fontFamily:
            '"Inter", "Avenir Next", "SF Pro Text", "HarmonyOS Sans SC", "PingFang SC", "Microsoft YaHei", sans-serif',
          colorBgLayout: isDark ? "#02071a" : "#eff4ff",
        },
        components: {
          Layout: {
            headerBg: "transparent",
            siderBg: isDark ? "#050b20" : "#f7f9ff",
            bodyBg: isDark ? "#02071a" : "#eff4ff",
          },
          Menu: {
            darkItemBg: "#050b20",
            darkSubMenuItemBg: "#050b20",
            darkItemSelectedBg: "rgba(79, 107, 255, 0.24)",
            darkItemHoverBg: "rgba(79, 107, 255, 0.12)",
          },
          Card: {
            boxShadow: isDark ? "0 14px 32px rgba(0, 0, 0, 0.36)" : "0 10px 24px rgba(15, 23, 42, 0.08)",
          },
          Button: {
            controlHeightLG: 46,
          },
        },
      }}
    >
      <AntdApp>
        <AuthProvider>
          <BrowserRouter>
            <ErrorBoundary>
            <Suspense fallback={<RouteFallback />}>
              <Routes>
                <Route path="/login" element={<AuthRoutes />} />
                <Route path="/" element={<ProtectedRoutes />}>
                  <Route index element={<DashboardPage />} />
                  <Route path="users" element={<UsersPage />} />
                  <Route path="departments" element={<DepartmentsPage />} />
                  <Route path="roles" element={<RolesPage />} />
                  <Route path="permissions" element={<PermissionsPage />} />
                  <Route path="policies" element={<PoliciesPage />} />
                  <Route path="registrations" element={<RegistrationsPage />} />
                  <Route path="menus" element={<MenusPage />} />
                  <Route path="login-logs" element={<LoginLogsPage />} />
                  <Route path="operation-logs" element={<OperationLogsPage />} />
                  <Route path="banned-ips" element={<BannedIPsPage />} />
                  <Route path="cluster" element={<Navigate to="/clusters" replace />} />
                  <Route path="clusters" element={<ClusterPage />} />
                  <Route path="pods" element={<PodPage />} />
                  <Route path="server-console" element={<ServerConsolePage />} />
                  <Route path="personal-settings" element={<PersonalSettingsPage />} />
                  <Route path="*" element={<DynamicMenuPage />} />
                </Route>
              </Routes>
            </Suspense>
            </ErrorBoundary>
          </BrowserRouter>
        </AuthProvider>
      </AntdApp>
    </ConfigProvider>
  );
}
