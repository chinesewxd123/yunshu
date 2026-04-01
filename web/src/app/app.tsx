import { App as AntdApp, ConfigProvider, Spin, theme } from "antd";
import { Suspense, lazy } from "react";
import { BrowserRouter, Navigate, Route, Routes, useLocation } from "react-router-dom";
import { AuthProvider, useAuth } from "../contexts/auth-context";

const AdminLayout = lazy(() => import("../layouts/admin-layout").then((module) => ({ default: module.AdminLayout })));
const DashboardPage = lazy(() => import("../pages/dashboard-page").then((module) => ({ default: module.DashboardPage })));
const LoginPage = lazy(() => import("../pages/login-page").then((module) => ({ default: module.LoginPage })));
const PermissionsPage = lazy(() =>
  import("../pages/permissions-page").then((module) => ({ default: module.PermissionsPage })),
);
const PoliciesPage = lazy(() => import("../pages/policies-page").then((module) => ({ default: module.PoliciesPage })));
const RolesPage = lazy(() => import("../pages/roles-page").then((module) => ({ default: module.RolesPage })));
const UsersPage = lazy(() => import("../pages/users-page").then((module) => ({ default: module.UsersPage })));
const RegistrationsPage = lazy(() => import("../pages/registrations-page").then((module) => ({ default: module.RegistrationsPage })));
const MenusPage = lazy(() => import("../pages/menus-page").then((module) => ({ default: module.MenusPage })));

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
  return (
    <ConfigProvider
      theme={{
        algorithm: theme.defaultAlgorithm,
        token: {
          colorPrimary: "#0f6cbd",
          colorSuccess: "#14804a",
          colorWarning: "#c96a11",
          colorError: "#c23a2b",
          borderRadius: 18,
          fontFamily:
            'Bahnschrift, Aptos, "HarmonyOS Sans SC", "PingFang SC", "Microsoft YaHei", sans-serif',
          colorBgLayout: "#edf3fb",
        },
        components: {
          Layout: {
            headerBg: "transparent",
            siderBg: "#081d3a",
            bodyBg: "#edf3fb",
          },
          Menu: {
            darkItemBg: "#081d3a",
            darkSubMenuItemBg: "#081d3a",
            darkItemSelectedBg: "rgba(15, 108, 189, 0.16)",
            darkItemHoverBg: "rgba(255, 255, 255, 0.08)",
          },
          Card: {
            boxShadow: "0 20px 54px rgba(8, 27, 52, 0.08)",
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
            <Suspense fallback={<RouteFallback />}>
              <Routes>
                <Route path="/login" element={<AuthRoutes />} />
                <Route path="/" element={<ProtectedRoutes />}>
                  <Route index element={<DashboardPage />} />
                  <Route path="users" element={<UsersPage />} />
                  <Route path="roles" element={<RolesPage />} />
                  <Route path="permissions" element={<PermissionsPage />} />
                  <Route path="policies" element={<PoliciesPage />} />
                  <Route path="registrations" element={<RegistrationsPage />} />
                  <Route path="menus" element={<MenusPage />} />
                </Route>
                <Route path="*" element={<Navigate to="/" replace />} />
              </Routes>
            </Suspense>
          </BrowserRouter>
        </AuthProvider>
      </AntdApp>
    </ConfigProvider>
  );
}
