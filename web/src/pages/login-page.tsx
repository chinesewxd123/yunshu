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
import { sendEmailCode, sendPasswordLoginCode, passwordLogin as passwordLoginRequest, emailLogin as emailLoginRequest } from "../services/auth";
import type { PasswordLoginPayload, EmailLoginPayload, SendEmailCodePayload } from "../types/api";

interface LocationState {
  from?: string;
}

const featureList = [
  {
    icon: <DatabaseOutlined />,
    title: "统一资产权限底座",
    description: "账号、角色模板、接口能力和授权编排放在同一套治理视图里。",
  },
  {
    icon: <DeploymentUnitOutlined />,
    title: "适合运维 CMDB 扩展",
    description: "当前页面结构已经为主机、服务树、环境分层等模块预留了后台风格。",
  },
  {
    icon: <SafetyCertificateOutlined />,
    title: "邮箱验证码登录",
    description: "支持用户名密码登录和邮箱验证码登录两种方式，安全灵活。",
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
  const [emailForm] = Form.useForm<EmailLoginPayload & { email: string }>();

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

  async function handleEmailLogin(values: EmailLoginPayload & { email: string }) {
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
            <Typography.Title level={2} style={{ margin: 0 }}>
              进入运维控制台
            </Typography.Title>
            <Typography.Paragraph className="inline-muted" style={{ marginBottom: 0 }}>
              使用用户名密码或邮箱验证码进行登录。
            </Typography.Paragraph>
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
                  <Form<EmailLoginPayload & { email: string }>
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
                          onClick={() => void handleSendEmailCode()}
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
            ]}
          />
        </Card>
      </section>
    </div>
  );
}