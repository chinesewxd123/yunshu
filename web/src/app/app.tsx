import { App as AntdApp, ConfigProvider, Spin, theme } from "antd";
import { BrowserRouter, Navigate, Route, Routes, useLocation } from "react-router-dom";
import { AuthProvider, useAuth } from "../contexts/auth-context";
import { AdminLayout } from "../layouts/admin-layout";
import { DashboardPage } from "../pages/dashboard-page";
import { LoginPage } from "../pages/login-page";
import { PermissionsPage } from "../pages/permissions-page";
import { PoliciesPage } from "../pages/policies-page";
import { RolesPage } from "../pages/roles-page";
import { UsersPage } from "../pages/users-page";

function ProtectedRoutes() {
  const { isAuthenticated, loading } = useAuth();
  const location = useLocation();

  if (loading) {
    return (
      <div className="full-screen-loading">
        <Spin size="large" />
      </div>
    );
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace state={{ from: location.pathname }} />;
  }

  return <AdminLayout />;
}

function AuthRoutes() {
  const { isAuthenticated, loading } = useAuth();

  if (loading) {
    return (
      <div className="full-screen-loading">
        <Spin size="large" />
      </div>
    );
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
          colorPrimary: "#0f766e",
          colorSuccess: "#16a34a",
          colorWarning: "#ea580c",
          colorError: "#dc2626",
          borderRadius: 18,
          fontFamily:
            'Aptos, "HarmonyOS Sans SC", "PingFang SC", "Microsoft YaHei", sans-serif',
          colorBgLayout: "#eef6f5",
        },
        components: {
          Layout: {
            headerBg: "rgba(255,255,255,0.78)",
            siderBg: "#0f172a",
            bodyBg: "#eef6f5",
          },
          Menu: {
            darkItemBg: "#0f172a",
            darkSubMenuItemBg: "#0f172a",
            darkItemSelectedBg: "rgba(20,184,166,0.22)",
            darkItemHoverBg: "rgba(255,255,255,0.08)",
          },
          Card: {
            boxShadow: "0 18px 48px rgba(15, 23, 42, 0.08)",
          },
        },
      }}
    >
      <AntdApp>
        <AuthProvider>
          <BrowserRouter>
            <Routes>
              <Route path="/login" element={<AuthRoutes />} />
              <Route path="/" element={<ProtectedRoutes />}>
                <Route index element={<DashboardPage />} />
                <Route path="users" element={<UsersPage />} />
                <Route path="roles" element={<RolesPage />} />
                <Route path="permissions" element={<PermissionsPage />} />
                <Route path="policies" element={<PoliciesPage />} />
              </Route>
              <Route path="*" element={<Navigate to="/" replace />} />
            </Routes>
          </BrowserRouter>
        </AuthProvider>
      </AntdApp>
    </ConfigProvider>
  );
}
