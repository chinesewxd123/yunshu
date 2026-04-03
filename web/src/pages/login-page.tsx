import { LockOutlined, MailOutlined, SafetyCertificateOutlined, UserOutlined } from "@ant-design/icons";
import { Button, Form, Input, Modal, message } from "antd";
import { useEffect, useMemo, useRef, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { sendEmailCode, sendPasswordLoginCode, registerByEmail } from "../services/auth";
import type {
  EmailLoginPayload,
  PasswordLoginPayload,
  RegisterPayload,
  SendEmailCodePayload,
  SendPasswordLoginCodeResult,
} from "../types/api";
import { BRAND_EN_NAME, BRAND_NAME, BRAND_SUBTITLE } from "../constants/brand";
import { useAuth } from "../contexts/auth-context";
import loginAtomBgUrl from "../assets/login-atom-bg.svg";

type AuthTabKey = "account" | "email";
type ButtonFxState = "idle" | "loading" | "success";

interface LocationState {
  from?: string;
}

function useCountdown(seconds: number, onTick: (next: number) => void) {
  useEffect(() => {
    if (seconds <= 0) return;
    const t = window.setTimeout(() => onTick(seconds - 1), 1000);
    return () => window.clearTimeout(t);
  }, [seconds, onTick]);
}

export function LoginPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const { passwordLoginAction, emailLoginAction } = useAuth();

  const [tab, setTab] = useState<AuthTabKey>("account");
  const [registerOpen, setRegisterOpen] = useState(false);

  const [submitting, setSubmitting] = useState(false);
  const [sendingCode, setSendingCode] = useState(false);

  const [passwordCodeCountdown, setPasswordCodeCountdown] = useState(0);
  const [emailCodeCountdown, setEmailCodeCountdown] = useState(0);
  const [registerCodeCountdown, setRegisterCodeCountdown] = useState(0);

  const [captchaKey, setCaptchaKey] = useState("");
  const [captchaImage, setCaptchaImage] = useState<string | null>(null);

  const [buttonFx, setButtonFx] = useState<ButtonFxState>("idle");

  const [passwordForm] = Form.useForm<PasswordLoginPayload>();
  const [emailForm] = Form.useForm<EmailLoginPayload>();
  const [registerForm] = Form.useForm<RegisterPayload>();

  const bgRef = useRef<HTMLDivElement | null>(null);
  const rafRef = useRef<number>(0);

  useEffect(() => {
    const bg = bgRef.current;
    if (!bg) return;
    const mq = window.matchMedia("(prefers-reduced-motion: reduce)");

    const onMove = (e: MouseEvent) => {
      if (mq.matches) return;
      cancelAnimationFrame(rafRef.current);
      rafRef.current = requestAnimationFrame(() => {
        const nx = (e.clientX / window.innerWidth - 0.5) * 14;
        const ny = (e.clientY / window.innerHeight - 0.5) * 9;
        bg.style.transform = `translate3d(${nx}px, ${ny}px, 0) scale(1.01)`;
      });
    };

    window.addEventListener("mousemove", onMove, { passive: true });
    return () => {
      cancelAnimationFrame(rafRef.current);
      window.removeEventListener("mousemove", onMove);
    };
  }, []);

  useCountdown(passwordCodeCountdown, setPasswordCodeCountdown);
  useCountdown(emailCodeCountdown, setEmailCodeCountdown);
  useCountdown(registerCodeCountdown, setRegisterCodeCountdown);

  const fromPath = (location.state as LocationState | null)?.from ?? "/";

  async function handleSendPasswordCode() {
    try {
      const username = passwordForm.getFieldValue("username");
      if (!username) {
        message.warning("请先输入用户名");
        return;
      }

      setSendingCode(true);
      const result: SendPasswordLoginCodeResult = await sendPasswordLoginCode({ username });
      setCaptchaKey(result.captcha_key);
      setCaptchaImage(result.image);
      passwordForm.setFieldValue("captcha_key", result.captcha_key);
      message.success("验证码已生成");
      setPasswordCodeCountdown(60);
    } catch {
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
      setEmailCodeCountdown(60);
    } finally {
      setSendingCode(false);
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

  async function runLogin<TPayload>(
    action: (payload: TPayload) => Promise<unknown>,
    payload: TPayload,
  ) {
    setSubmitting(true);
    setButtonFx("loading");

    try {
      await action(payload);
      message.success("登录成功");
      setButtonFx("success");
      window.setTimeout(() => navigate(fromPath, { replace: true }), 520);
    } catch (e) {
      setButtonFx("idle");
      message.error(e instanceof Error ? e.message : "登录失败");
    } finally {
      window.setTimeout(() => setButtonFx("idle"), 1200);
      setSubmitting(false);
    }
  }

  async function handlePasswordLogin(values: PasswordLoginPayload) {
    const payload: PasswordLoginPayload = {
      ...values,
      captcha_key: values.captcha_key || captchaKey,
    };
    void runLogin(passwordLoginAction as (p: PasswordLoginPayload) => Promise<unknown>, payload);
  }

  async function handleEmailLogin(values: EmailLoginPayload) {
    void runLogin(emailLoginAction as (p: EmailLoginPayload) => Promise<unknown>, values);
  }

  async function handleRegister(values: RegisterPayload) {
    setSubmitting(true);
    try {
      const result = await registerByEmail(values);
      message.success(result.message || "注册申请已提交，请等待管理员审核");
      setRegisterOpen(false);
      registerForm.resetFields();
    } catch (e) {
      message.error(e instanceof Error ? e.message : "注册失败");
    } finally {
      setSubmitting(false);
    }
  }

  const submitButtonLabel = useMemo(() => "登录", []);

  return (
    <div className="login-shell login-shell--auth">
      <div className="login-bg-atom" ref={bgRef} aria-hidden="true">
        <img className="login-bg-atom__img" src={loginAtomBgUrl} alt="" />
      </div>

      <div className="login-card-glass" role="region" aria-label="登录">
        <div className="login-card-glass__top">
          <div className="login-brand">
            <div className="login-brand__badge">
              <SafetyCertificateOutlined />
            </div>
            <div className="login-brand__text">
              <div className="login-brand__sub">{BRAND_SUBTITLE}</div>
              <div className="login-brand__en">{BRAND_EN_NAME}</div>
              <div className="login-brand__name">{BRAND_NAME}</div>
            </div>
          </div>
        </div>

        <div className="login-auth-tabs" role="tablist" aria-label="登录方式">
          <button
            type="button"
            className={`login-auth-tabs__tab ${tab === "account" ? "is-active" : ""}`}
            onClick={() => setTab("account")}
            role="tab"
            aria-selected={tab === "account"}
          >
            账号登录
          </button>
          <button
            type="button"
            className={`login-auth-tabs__tab ${tab === "email" ? "is-active" : ""}`}
            onClick={() => setTab("email")}
            role="tab"
            aria-selected={tab === "email"}
          >
            邮箱登录
          </button>
          <div
            className="login-auth-tabs__indicator"
            style={{ transform: `translateX(${tab === "account" ? 0 : 100}%)` }}
            aria-hidden="true"
          />
        </div>

        <div className="login-auth-panels">
          <div className={`login-auth-pane ${tab === "account" ? "is-active" : ""}`} aria-hidden={tab !== "account"}>
            <Form<PasswordLoginPayload> form={passwordForm} layout="vertical" onFinish={handlePasswordLogin} size="large">
              <Form.Item label="用户名" name="username" rules={[{ required: true, message: "请输入用户名" }]}>
                <Input prefix={<UserOutlined />} placeholder="请输入用户名" autoComplete="off" />
              </Form.Item>

              <Form.Item label="密码" name="password" rules={[{ required: true, message: "请输入密码" }]}>
                <Input.Password prefix={<LockOutlined />} placeholder="请输入密码" autoComplete="new-password" />
              </Form.Item>

              <Form.Item label="验证码">
                <div className="login-captchaRow">
                  <Form.Item
                    name="code"
                    rules={[
                      { required: true, message: "请输入验证码" },
                      { len: 4, message: "验证码为4位数字" },
                      { pattern: /^\d+$/, message: "验证码必须为数字" },
                    ]}
                    style={{ margin: 0, flex: 1 }}
                  >
                    <Input
                      prefix={<SafetyCertificateOutlined />}
                      placeholder="请输入验证码"
                      maxLength={4}
                    />
                  </Form.Item>

                  {captchaImage ? (
                    <div
                      className="login-captchaWave"
                      onClick={() => void handleSendPasswordCode()}
                      role="button"
                      tabIndex={0}
                      aria-label="刷新验证码"
                    >
                      <img
                        className="login-captchaWave__img"
                        src={`data:image/png;base64,${captchaImage}`}
                        alt="验证码"
                      />
                      <div className="login-captchaWave__noise" />
                      <div className="login-captchaWave__wave" />
                    </div>
                  ) : (
                    <div className="login-captchaEmpty login-captchaEmpty--inline">
                      <Button type="link" onClick={() => void handleSendPasswordCode()} loading={sendingCode}>
                        刷新
                      </Button>
                    </div>
                  )}
                </div>
              </Form.Item>

              <Form.Item
                name="captcha_key"
                style={{ display: "none" }}
                rules={[{ required: true, message: "请先刷新验证码" }]}
              >
                <Input type="hidden" />
              </Form.Item>

              <div className="login-submitRow">
                <Button
                  htmlType="submit"
                  className="login-submitBtn"
                  disabled={submitting}
                  data-fx={buttonFx}
                >
                  <span className="login-submitBtn__label">{submitButtonLabel}</span>
                  <span className="login-submitBtn__spinner" aria-hidden="true" />
                </Button>
              </div>
            </Form>
          </div>

          <div className={`login-auth-pane ${tab === "email" ? "is-active" : ""}`} aria-hidden={tab !== "email"}>
            <Form<EmailLoginPayload> form={emailForm} layout="vertical" onFinish={handleEmailLogin} size="large">
              <Form.Item label="邮箱" name="email" rules={[{ required: true, type: "email", message: "请输入正确的邮箱地址" }]}>
                <Input prefix={<MailOutlined />} placeholder="请输入邮箱地址" autoComplete="off" />
              </Form.Item>

              <Form.Item label="验证码" name="code" rules={[{ required: true, message: "请输入验证码" }]}>
                <Input prefix={<SafetyCertificateOutlined />} placeholder="请输入验证码" style={{ marginBottom: 10 }} />
                <div className="login-emailActions">
                  <Button
                    type="primary"
                    ghost
                    className="login-sendBtn"
                    onClick={() => void handleSendEmailCode()}
                    loading={sendingCode}
                    disabled={emailCodeCountdown > 0}
                  >
                    {emailCodeCountdown > 0 ? `${emailCodeCountdown}s` : "发送验证码"}
                  </Button>
                </div>
              </Form.Item>

              <div className="login-submitRow">
                <Button htmlType="submit" className="login-submitBtn" disabled={submitting} data-fx={buttonFx}>
                  <span className="login-submitBtn__label">{submitButtonLabel}</span>
                  <span className="login-submitBtn__spinner" aria-hidden="true" />
                </Button>
              </div>
            </Form>
          </div>
        </div>

        <div className="login-bottom">
          <button type="button" className="login-registerLink" onClick={() => setRegisterOpen(true)}>
            注册新用户
          </button>
        </div>
      </div>

      <Modal
        open={registerOpen}
        title="注册账号"
        footer={null}
        onCancel={() => setRegisterOpen(false)}
        destroyOnClose
        centered
        width={520}
        className="login-registerModal"
      >
        <Form<RegisterPayload> form={registerForm} layout="vertical" onFinish={handleRegister} size="large">
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

          <Form.Item label="验证码" name="code" rules={[{ required: true, len: 6, message: "验证码为6位数字" }]}>
            <Input
              prefix={<SafetyCertificateOutlined />}
              placeholder="请输入验证码"
              maxLength={6}
              style={{ marginBottom: 10 }}
            />
            <Button
              type="primary"
              ghost
              className="login-sendBtn"
              onClick={() => void handleSendRegisterCode()}
              loading={sendingCode}
              disabled={registerCodeCountdown > 0}
            >
              {registerCodeCountdown > 0 ? `${registerCodeCountdown}s` : "发送验证码"}
            </Button>
          </Form.Item>

          <Button htmlType="submit" type="primary" block disabled={submitting}>
            提交注册
          </Button>
        </Form>
      </Modal>
    </div>
  );
}

