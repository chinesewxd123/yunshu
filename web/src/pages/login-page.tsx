import {
  DatabaseOutlined,
  DeploymentUnitOutlined,
  LockOutlined,
  SafetyCertificateOutlined,
  UserOutlined,
  MailOutlined,
} from "@ant-design/icons";
import { Button, Card, Form, Input, Space, Tabs, Typography, message } from "antd";
import { useEffect, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import {
  BRAND_DESCRIPTION,
  BRAND_EN_NAME,
  BRAND_NAME,
  BRAND_SUBTITLE,
} from "../constants/brand";
import { useAuth } from "../contexts/auth-context";
import { sendEmailCode, sendPasswordLoginCode, passwordLogin as passwordLoginRequest, emailLogin as emailLoginRequest, registerByEmail } from "../services/auth";
import type { PasswordLoginPayload, EmailLoginPayload, SendEmailCodePayload, RegisterPayload } from "../types/api";

interface LocationState {
  from?: string;
}

const featureList = [
  {
    icon: <DatabaseOutlined />,
    title: "权限与页面一致",
    description: "控制台每个菜单对应一组后端 /api/v1 接口；资产总览提供完整「接口目录」便于核对。",
  },
  {
    icon: <DeploymentUnitOutlined />,
    title: "角色驱动授权",
    description: "用户须绑定角色，角色通过 Casbin 绑定「资源路径 + HTTP 方法」能力项后方可访问管理接口。",
  },
  {
    icon: <SafetyCertificateOutlined />,
    title: "双通道登录",
    description: "管理员可用「用户名 + 密码 + 图形验证码」；日常推荐「邮箱 + 6 位验证码」。注册提交后需管理员审核。",
  },
];

export function LoginPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const { passwordLoginAction, emailLoginAction } = useAuth();
  const [submitting, setSubmitting] = useState(false);
  const [sendingCode, setSendingCode] = useState(false);
  const [codeCountdown, setCodeCountdown] = useState(0);
  const [passwordCodeCountdown, setPasswordCodeCountdown] = useState(0);
  const [captchaKey, setCaptchaKey] = useState<string>("");
  const [captchaImage, setCaptchaImage] = useState<string>("");
  const [passwordForm] = Form.useForm<PasswordLoginPayload>();
  const [emailForm] = Form.useForm<EmailLoginPayload>();
  const [registerForm] = Form.useForm<RegisterPayload>();
  const [registerCodeCountdown, setRegisterCodeCountdown] = useState(0);

  useEffect(() => {
    let timer: ReturnType<typeof setTimeout>;
    if (codeCountdown > 0) {
      timer = setTimeout(() => setCodeCountdown(codeCountdown - 1), 1000);
    }
    return () => {
      if (timer) clearTimeout(timer);
    };
  }, [codeCountdown]);

  useEffect(() => {
    let timer: ReturnType<typeof setTimeout>;
    if (passwordCodeCountdown > 0) {
      timer = setTimeout(() => setPasswordCodeCountdown(passwordCodeCountdown - 1), 1000);
    }
    return () => {
      if (timer) clearTimeout(timer);
    };
  }, [passwordCodeCountdown]);

  useEffect(() => {
    let timer: ReturnType<typeof setTimeout>;
    if (registerCodeCountdown > 0) {
      timer = setTimeout(() => setRegisterCodeCountdown(registerCodeCountdown - 1), 1000);
    }
    return () => {
      if (timer) clearTimeout(timer);
    };
  }, [registerCodeCountdown]);

  async function handlePasswordLogin(values: PasswordLoginPayload) {
    setSubmitting(true);
    try {
      await passwordLoginAction(values);
      message.success("登录成功");
      const state = location.state as LocationState | null;
      navigate(state?.from || "/", { replace: true });
    } finally {
      setSubmitting(false);
    }
  }

  async function handleSendPasswordCode() {
    try {
      const username = passwordForm.getFieldValue("username");
      if (!username) {
        message.warning("请先输入用户名");
        return;
      }

      setSendingCode(true);
      const result = await sendPasswordLoginCode({ username });
      setCaptchaKey(result.captcha_key);
      setCaptchaImage(result.image);
      passwordForm.setFieldValue("captcha_key", result.captcha_key);
      message.success("验证码已生成");
      setPasswordCodeCountdown(60);
    } catch (error) {
      message.error("生成验证码失败");
    } finally {
      setSendingCode(false);
    }
  }

  async function handleSendEmailCode() {
    try {
      const email = emailForm.getFieldValue("email");
      if (!email) {
        message.warning("请先输入邮箱地址");
        return;
      }

      setSendingCode(true);
      const payload: SendEmailCodePayload = { email, scene: "login" };
      await sendEmailCode(payload);
      message.success("验证码已发送到您的邮箱，请查收");
      setCodeCountdown(60);
    } finally {
      setSendingCode(false);
    }
  }

  async function handleSendEmailOrUsernameCode() {
    await handleSendEmailCode();
  }

  async function handleEmailLogin(values: EmailLoginPayload) {
    setSubmitting(true);
    try {
      const { email, code } = values;
      await emailLoginAction({ email, code });
      message.success("登录成功");
      const state = location.state as LocationState | null;
      navigate(state?.from || "/", { replace: true });
    } finally {
      setSubmitting(false);
    }
  }

  async function handleSendRegisterCode() {
    try {
      const email = registerForm.getFieldValue("email");
      if (!email) {
        message.warning("请先输入邮箱地址");
        return;
      }

      setSendingCode(true);
      const payload: SendEmailCodePayload = { email, scene: "register" };
      await sendEmailCode(payload);
      message.success("验证码已发送到您的邮箱，请查收");
      setRegisterCodeCountdown(60);
    } finally {
      setSendingCode(false);
    }
  }

  async function handleRegister(values: RegisterPayload) {
    setSubmitting(true);
    try {
      const result = await registerByEmail(values);
      message.success(result.message || "注册申请已提交，请等待管理员审核");
      registerForm.resetFields();
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div className="login-shell">
      <section className="login-promo">
        <div className="login-badge">
          <SafetyCertificateOutlined />
          {BRAND_SUBTITLE}
        </div>
        <Typography.Text className="login-brand-code">{BRAND_EN_NAME}</Typography.Text>
        <h1 className="login-promo__title">{BRAND_NAME}</h1>
        <p className="login-promo__desc">{BRAND_DESCRIPTION}</p>

        <div className="login-points">
          {featureList.map((item) => (
            <div className="login-point" key={item.title}>
              <div className="login-point__icon">{item.icon}</div>
              <div>
                <Typography.Title level={4} className="login-point__title">
                  {item.title}
                </Typography.Title>
                <Typography.Paragraph className="login-point__desc">{item.description}</Typography.Paragraph>
              </div>
            </div>
          ))}
        </div>
      </section>

      <section className="login-panel">
        <Card className="login-card" bordered={false}>
          <Space direction="vertical" size={8} style={{ width: "100%", marginBottom: 28 }}>
            <Typography.Text className="code-pill">安全登录</Typography.Text>
          </Space>

          <Tabs
            defaultActiveKey="password"
            items={[
              {
                key: "password",
                label: "用户名密码登录",
                children: (
                  <Form<PasswordLoginPayload>
                    form={passwordForm}
                    layout="vertical"
                    onFinish={handlePasswordLogin}
                    size="large"
                  >
                    <Form.Item label="用户名" name="username" rules={[{ required: true, message: "请输入用户名" }]}>
                      <Input prefix={<UserOutlined />} placeholder="请输入用户名" autoComplete="off" />
                    </Form.Item>
                    <Form.Item label="密码" name="password" rules={[{ required: true, message: "请输入密码" }]}>
                      <Input.Password prefix={<LockOutlined />} placeholder="请输入密码" autoComplete="new-password" />
                    </Form.Item>
                    <div style={{ marginBottom: 24 }}>
                      <Typography.Text style={{ display: "block", marginBottom: 8 }}>验证码</Typography.Text>
                      <Form.Item name="code" rules={[{ required: true, message: "请输入验证码" }, { len: 4, message: "验证码为4位数字" }, { pattern: /^\d+$/, message: "验证码必须为数字" }]} style={{ margin: 0, marginBottom: 8 }}>
                        <Input 
                          prefix={<SafetyCertificateOutlined />} 
                          placeholder="请输入验证码" 
                          maxLength={4}
                          suffix={
                            <Button 
                              type="link" 
                              size="small"
                              onClick={() => void handleSendPasswordCode()}
                              loading={sendingCode}
                              disabled={passwordCodeCountdown > 0}
                              style={{ padding: 0, height: "auto" }}
                            >
                              {passwordCodeCountdown > 0 ? `${passwordCodeCountdown}s` : "刷新"}
                            </Button>
                          }
                        />
                      </Form.Item>
                      <Form.Item name="captcha_key" style={{ margin: 0, display: "none" }}>
                        <Input type="hidden" />
                      </Form.Item>
                      {captchaImage && (
                        <div style={{ border: "1px solid #d9d9d9", borderRadius: 4, padding: 4, backgroundColor: "#fff", textAlign: "center" }}>
                          <img 
                            src={`data:image/png;base64,${captchaImage}`} 
                            alt="验证码" 
                            style={{ width: 200, height: 80, cursor: "pointer" }}
                            onClick={() => void handleSendPasswordCode()}
                            title="点击刷新验证码"
                          />
                        </div>
                      )}
                    </div>
                    <Button type="primary" htmlType="submit" block loading={submitting}>
                      登录 {BRAND_NAME}
                    </Button>
                  </Form>
                ),
              },
              {
                key: "email",
                label: "邮箱验证码登录",
                children: (
                  <Form<EmailLoginPayload>
                    form={emailForm}
                    layout="vertical"
                    onFinish={handleEmailLogin}
                    size="large"
                  >
                    <Form.Item label="邮箱" name="email" rules={[{ required: true, type: "email", message: "请输入正确的邮箱地址" }]}>
                      <Input prefix={<MailOutlined />} placeholder="请输入邮箱地址" />
                    </Form.Item>
                    <div style={{ marginBottom: 24 }}>
                      <Typography.Text style={{ display: "block", marginBottom: 8 }}>验证码</Typography.Text>
                      <Space.Compact style={{ width: "100%" }}>
                        <Form.Item name="code" rules={[{ required: true, message: "请输入验证码" }]} style={{ margin: 0, flex: 1 }}>
                          <Input prefix={<SafetyCertificateOutlined />} placeholder="请输入验证码" />
                        </Form.Item>
                        <Button
                          onClick={() => void handleSendEmailOrUsernameCode()}
                          loading={sendingCode}
                          disabled={codeCountdown > 0}
                        >
                          {codeCountdown > 0 ? `${codeCountdown}s` : "发送验证码"}
                        </Button>
                      </Space.Compact>
                    </div>
                    <Button type="primary" htmlType="submit" block loading={submitting}>
                      登录 {BRAND_NAME}
                    </Button>
                  </Form>
                ),
              },
              {
                key: "register",
                label: "注册账号",
                children: (
                  <Form<RegisterPayload>
                    form={registerForm}
                    layout="vertical"
                    onFinish={handleRegister}
                    size="large"
                  >
                    <Form.Item label="用户名" name="username" rules={[{ required: true, min: 3, max: 64, message: "用户名长度为3-64个字符" }]}>
                      <Input prefix={<UserOutlined />} placeholder="请输入用户名" />
                    </Form.Item>
                    <Form.Item label="邮箱" name="email" rules={[{ required: true, type: "email", message: "请输入正确的邮箱地址" }]}>
                      <Input prefix={<MailOutlined />} placeholder="请输入邮箱地址" />
                    </Form.Item>
                    <Form.Item label="昵称" name="nickname" rules={[{ required: true, max: 128, message: "请输入昵称" }]}>
                      <Input prefix={<UserOutlined />} placeholder="请输入昵称" />
                    </Form.Item>
                    <Form.Item label="密码" name="password" rules={[{ required: true, min: 6, max: 64, message: "密码长度为6-64个字符" }]}>
                      <Input.Password prefix={<LockOutlined />} placeholder="请输入密码" />
                    </Form.Item>
                    <div style={{ marginBottom: 24 }}>
                      <Typography.Text style={{ display: "block", marginBottom: 8 }}>验证码</Typography.Text>
                      <Space.Compact style={{ width: "100%" }}>
                        <Form.Item name="code" rules={[{ required: true, len: 6, message: "验证码为6位数字" }]} style={{ margin: 0, flex: 1 }}>
                          <Input prefix={<SafetyCertificateOutlined />} placeholder="请输入验证码" maxLength={6} />
                        </Form.Item>
                        <Button
                          onClick={() => void handleSendRegisterCode()}
                          loading={sendingCode}
                          disabled={registerCodeCountdown > 0}
                        >
                          {registerCodeCountdown > 0 ? `${registerCodeCountdown}s` : "发送验证码"}
                        </Button>
                      </Space.Compact>
                    </div>
                    <Button type="primary" htmlType="submit" block loading={submitting}>
                      注册账号
                    </Button>
                  </Form>
                ),
              },
            ]}
          />
        </Card>
      </section>
    </div>
  );
}