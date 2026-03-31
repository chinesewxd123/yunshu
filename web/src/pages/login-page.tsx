import { LockOutlined, SafetyCertificateOutlined, UserOutlined } from "@ant-design/icons";
import { Button, Card, Form, Input, Space, Typography, message } from "antd";
import { useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { useAuth } from "../contexts/auth-context";
import type { LoginPayload } from "../types/api";

interface LocationState {
  from?: string;
}

export function LoginPage() {
  const [form] = Form.useForm<LoginPayload>();
  const navigate = useNavigate();
  const location = useLocation();
  const { loginAction } = useAuth();
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit(values: LoginPayload) {
    setSubmitting(true);
    try {
      await loginAction(values);
      message.success("登录成功");
      const state = location.state as LocationState | null;
      navigate(state?.from || "/", { replace: true });
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="login-shell">
      <section className="login-promo">
        <div className="login-badge">
          <SafetyCertificateOutlined />
          基于 Gin + Casbin 的权限中枢
        </div>
        <h1 className="login-promo__title">统一权限治理，从登录开始。</h1>
        <p className="login-promo__desc">
          这套基础版后台已经连通用户、角色、权限和策略绑定能力，适合直接用于二次开发与日常调试。
        </p>
        <div className="login-points">
          <div className="login-point">
            <Typography.Title level={4} style={{ color: "#fff", marginTop: 0 }}>
              清晰的后台布局
            </Typography.Title>
            <div>左侧导航、顶部状态、面包屑与卡片化内容区一起工作，方便持续扩展。</div>
          </div>
          <div className="login-point">
            <Typography.Title level={4} style={{ color: "#fff", marginTop: 0 }}>
              完整的基础权限流
            </Typography.Title>
            <div>前端直接调用现有 JWT 鉴权与 Casbin 策略接口，页面行为与后端状态保持一致。</div>
          </div>
        </div>
      </section>
      <section className="login-panel">
        <Card className="login-card" bordered={false}>
          <Space direction="vertical" size={8} style={{ width: "100%", marginBottom: 28 }}>
            <Typography.Text className="code-pill">欢迎回来</Typography.Text>
            <Typography.Title level={2} style={{ margin: 0 }}>
              登录管理系统
            </Typography.Title>
            <Typography.Paragraph className="inline-muted" style={{ marginBottom: 0 }}>
              默认种子账号可直接使用 `admin / Admin@123` 进行联调。
            </Typography.Paragraph>
          </Space>
          <Form<LoginPayload>
            form={form}
            layout="vertical"
            initialValues={{ username: "admin", password: "Admin@123" }}
            onFinish={handleSubmit}
            size="large"
          >
            <Form.Item label="用户名" name="username" rules={[{ required: true, message: "请输入用户名" }]}>
              <Input prefix={<UserOutlined />} placeholder="请输入用户名" />
            </Form.Item>
            <Form.Item label="密码" name="password" rules={[{ required: true, message: "请输入密码" }]}>
              <Input.Password prefix={<LockOutlined />} placeholder="请输入密码" />
            </Form.Item>
            <Button type="primary" htmlType="submit" block loading={submitting}>
              进入系统
            </Button>
          </Form>
        </Card>
      </section>
    </div>
  );
}
