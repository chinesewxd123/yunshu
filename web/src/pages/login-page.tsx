import {
  AppstoreOutlined,
  BgColorsOutlined,
  BulbFilled,
  BulbOutlined,
  TranslationOutlined,
  LockOutlined,
  MailOutlined,
  SafetyCertificateOutlined,
  UserOutlined,
} from "@ant-design/icons";
import { Button, Form, Input, Modal, message } from "antd";
import { useEffect, useState } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { sendEmailCode, sendPasswordLoginCode, registerByEmail } from "../services/auth";
import type {
  EmailLoginPayload,
  PasswordLoginPayload,
  RegisterPayload,
  SendEmailCodePayload,
  SendPasswordLoginCodeResult,
} from "../types/api";
import { useAuth } from "../contexts/auth-context";
import loginHeroImage from "../assets/login-hero.svg";

type AuthTabKey = "account" | "email";
type ButtonFxState = "idle" | "loading" | "success";

interface LocationState {
  from?: string;
}

function getErrorMessage(error: unknown, fallback: string) {
  const responseMessage = (error as any)?.response?.data?.message;
  if (typeof responseMessage === "string" && responseMessage.trim()) {
    return responseMessage;
  }
  if (error instanceof Error && error.message.trim()) {
    return error.message;
  }
  return fallback;
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
  const [darkMode, setDarkMode] = useState<boolean>(() => window.localStorage.getItem("admin-theme-mode") !== "light");
  const [langMode, setLangMode] = useState<"zh" | "en">("zh");
  const [panelAlign, setPanelAlign] = useState<"left" | "center" | "right">("right");
  const [paletteOpen, setPaletteOpen] = useState(false);
  const [layoutOpen, setLayoutOpen] = useState(false);
  const [accent, setAccent] = useState<"blue" | "violet" | "emerald" | "amber">("blue");

  const [passwordForm] = Form.useForm<PasswordLoginPayload>();
  const [emailForm] = Form.useForm<EmailLoginPayload>();
  const [registerForm] = Form.useForm<RegisterPayload>();

  useCountdown(passwordCodeCountdown, setPasswordCodeCountdown);
  useCountdown(emailCodeCountdown, setEmailCodeCountdown);
  useCountdown(registerCodeCountdown, setRegisterCodeCountdown);

  useEffect(() => {
    const mode = darkMode ? "dark" : "light";
    window.localStorage.setItem("admin-theme-mode", mode);
    window.dispatchEvent(new CustomEvent("admin-theme-mode-change", { detail: { mode } }));
  }, [darkMode]);

  useEffect(() => {
    const onModeChange = (event: Event) => {
      const detail = (event as CustomEvent<{ mode?: "dark" | "light" }>).detail;
      if (!detail?.mode) return;
      setDarkMode(detail.mode !== "light");
    };
    window.addEventListener("admin-theme-mode-change", onModeChange as EventListener);
    return () => {
      window.removeEventListener("admin-theme-mode-change", onModeChange as EventListener);
    };
  }, []);

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
      const payload: RegisterPayload = {
        ...values,
        code: String(values.code ?? "").trim().replace(/[^\d]/g, ""),
      };
      const result = await registerByEmail(payload);
      message.success(result.message || "注册申请已提交，请等待管理员审核");
      setRegisterOpen(false);
      registerForm.resetFields();
    } catch (e) {
      message.error(getErrorMessage(e, "注册失败"));
    } finally {
      setSubmitting(false);
    }
  }

  const isZh = langMode === "zh";
  const submitButtonLabel = isZh ? "登录" : "Login";
  const cardTitle = isZh ? "欢迎回来 👋🏻" : "Welcome Back 👋🏻";
  const cardSubTitle = isZh ? "请输入您的账户信息以开始管理您的系统" : "Please enter your account information to continue.";
  const appTitle = isZh ? "云枢运维平台" : "Yunshu Ops Platform";
  const introTitle = isZh ? "云枢运维平台" : "Yunshu Ops Platform";
  const introDesc = isZh ? "开箱即用的企业级运维管理系统" : "Out-of-the-box enterprise operations platform";

  function renderFormCard() {
    return (
      <section className="gw-auth-card" role="region" aria-label="登录面板">
        <div className="gw-auth-card__header">
          <div className="gw-auth-card__title">{cardTitle}</div>
          <div className="gw-auth-card__sub">{cardSubTitle}</div>
        </div>

        <div className="login-light-switch" role="tablist" aria-label="登录方式">
          <button
            type="button"
            className={`login-light-switch__item ${tab === "account" ? "is-active" : ""}`}
            onClick={() => setTab("account")}
            role="tab"
            aria-selected={tab === "account"}
          >
            {isZh ? "用户名密码登录" : "Account Login"}
          </button>
          <button
            type="button"
            className={`login-light-switch__item ${tab === "email" ? "is-active" : ""}`}
            onClick={() => setTab("email")}
            role="tab"
            aria-selected={tab === "email"}
          >
            {isZh ? "邮箱验证码登录" : "Email Login"}
          </button>
        </div>

        <div className="login-light-card__hint">
          {tab === "account"
            ? isZh
              ? "适用于平台账号直接登录"
              : "Direct platform account login"
            : isZh
              ? "适用于通过邮箱验证码快速登录"
              : "Quick login with email code"}
        </div>

        {tab === "account" ? (
          <Form<PasswordLoginPayload> form={passwordForm} layout="vertical" onFinish={handlePasswordLogin} size="large">
            <Form.Item label={isZh ? "用户名" : "Username"} name="username" rules={[{ required: true, message: isZh ? "请输入用户名" : "Please enter username" }]}>
              <Input prefix={<UserOutlined />} placeholder={isZh ? "请输入用户名" : "Please enter username"} autoComplete="off" />
            </Form.Item>

            <Form.Item label={isZh ? "密码" : "Password"} name="password" rules={[{ required: true, message: isZh ? "请输入密码" : "Please enter password" }]}>
              <Input.Password prefix={<LockOutlined />} placeholder={isZh ? "请输入密码" : "Please enter password"} autoComplete="new-password" />
            </Form.Item>

            <Form.Item label={isZh ? "验证码" : "Code"}>
              <div className="login-captchaRow">
                <Form.Item
                  name="code"
                  rules={[
                    { required: true, message: isZh ? "请输入验证码" : "Please enter code" },
                    { len: 4, message: isZh ? "验证码为4位数字" : "4 digits required" },
                    { pattern: /^\d+$/, message: isZh ? "验证码必须为数字" : "Digits only" },
                  ]}
                  style={{ margin: 0, flex: 1 }}
                >
                  <Input prefix={<SafetyCertificateOutlined />} placeholder={isZh ? "验证码" : "Code"} maxLength={4} />
                </Form.Item>

                <div
                  className={`login-captchaWave ${sendingCode ? "is-loading" : ""}`}
                  onClick={() => void handleSendPasswordCode()}
                  role="button"
                  tabIndex={0}
                  aria-label={isZh ? "点击刷新验证码" : "Refresh captcha"}
                >
                  {captchaImage ? (
                    <img className="login-captchaWave__img" src={`data:image/png;base64,${captchaImage}`} alt={isZh ? "验证码图片" : "Captcha image"} />
                  ) : (
                    <span className="login-captchaWave__placeholder">{isZh ? "生成验证码" : "Generate"}</span>
                  )}
                </div>
              </div>
            </Form.Item>

            <Form.Item
              name="captcha_key"
              style={{ display: "none" }}
              rules={[{ required: true, message: isZh ? "请先生成验证码" : "Please generate code first" }]}
            >
              <Input type="hidden" />
            </Form.Item>

            <div className="login-submitRow">
              <Button htmlType="submit" className="login-submitBtn" disabled={submitting || !captchaKey} data-fx={buttonFx}>
                <span className="login-submitBtn__label">{submitButtonLabel}</span>
                <span className="login-submitBtn__spinner" aria-hidden="true" />
              </Button>
            </div>
          </Form>
        ) : (
          <Form<EmailLoginPayload> form={emailForm} layout="vertical" onFinish={handleEmailLogin} size="large">
            <Form.Item label={isZh ? "邮箱" : "Email"} name="email" rules={[{ required: true, type: "email", message: isZh ? "请输入正确的邮箱地址" : "Please enter valid email" }]}>
              <Input prefix={<MailOutlined />} placeholder={isZh ? "请输入邮箱地址" : "Please enter email"} autoComplete="off" />
            </Form.Item>

            <Form.Item label={isZh ? "验证码" : "Code"} name="code" rules={[{ required: true, message: isZh ? "请输入验证码" : "Please enter code" }]}>
              <Input prefix={<SafetyCertificateOutlined />} placeholder={isZh ? "邮箱验证码" : "Email code"} />
            </Form.Item>

            <div className="login-light-inlineAction">
              <Button
                type="default"
                className="login-light-secondaryBtn"
                onClick={() => void handleSendEmailCode()}
                loading={sendingCode}
                disabled={emailCodeCountdown > 0}
              >
                {emailCodeCountdown > 0 ? `${emailCodeCountdown}s ${isZh ? "后重发" : "to resend"}` : isZh ? "发送邮箱验证码" : "Send Email Code"}
              </Button>
            </div>

            <div className="login-submitRow">
              <Button htmlType="submit" className="login-submitBtn" disabled={submitting} data-fx={buttonFx}>
                <span className="login-submitBtn__label">{submitButtonLabel}</span>
                <span className="login-submitBtn__spinner" aria-hidden="true" />
              </Button>
            </div>
          </Form>
        )}

        <div className="login-light-card__footer">
          <button type="button" className="login-light-registerLink" onClick={() => setRegisterOpen(true)}>
            {isZh ? "注册用户" : "Register User"}
          </button>
        </div>
      </section>
    );
  }

  return (
    <div className={`gw-auth-shell ${darkMode ? "is-dark" : "is-light"} gw-accent-${accent}`}>
      <div className="gw-auth-brand">
        <span className="gw-auth-brand__logoDot" />
        <span>{appTitle}</span>
      </div>
      <div className="gw-auth-toolbar">
        <button
          type="button"
          className={`gw-auth-toolbar__btn ${paletteOpen ? "is-active" : ""}`}
          onClick={() => {
            setPaletteOpen((v) => !v);
            setLayoutOpen(false);
          }}
        >
          <BgColorsOutlined />
        </button>
        {paletteOpen ? (
          <div className="gw-auth-toolbar__panel gw-auth-toolbar__panel--palette">
            <button type="button" className={`gw-auth-dot gw-auth-dot--blue ${accent === "blue" ? "is-active" : ""}`} onClick={() => setAccent("blue")} />
            <button type="button" className={`gw-auth-dot gw-auth-dot--violet ${accent === "violet" ? "is-active" : ""}`} onClick={() => setAccent("violet")} />
            <button type="button" className={`gw-auth-dot gw-auth-dot--emerald ${accent === "emerald" ? "is-active" : ""}`} onClick={() => setAccent("emerald")} />
            <button type="button" className={`gw-auth-dot gw-auth-dot--amber ${accent === "amber" ? "is-active" : ""}`} onClick={() => setAccent("amber")} />
          </div>
        ) : null}
        <button
          type="button"
          className={`gw-auth-toolbar__btn ${layoutOpen ? "is-active" : ""}`}
          onClick={() => {
            setLayoutOpen((v) => !v);
            setPaletteOpen(false);
          }}
        >
          <AppstoreOutlined />
        </button>
        {layoutOpen ? (
          <div className="gw-auth-toolbar__panel gw-auth-toolbar__panel--layout">
            <button type="button" className={panelAlign === "left" ? "is-active" : ""} onClick={() => setPanelAlign("left")}>{isZh ? "居左" : "Left"}</button>
            <button type="button" className={panelAlign === "center" ? "is-active" : ""} onClick={() => setPanelAlign("center")}>{isZh ? "居中" : "Center"}</button>
            <button type="button" className={panelAlign === "right" ? "is-active" : ""} onClick={() => setPanelAlign("right")}>{isZh ? "居右" : "Right"}</button>
          </div>
        ) : null}
        <button type="button" className="gw-auth-toolbar__btn" onClick={() => setLangMode((v) => (v === "zh" ? "en" : "zh"))}>
          <TranslationOutlined />
        </button>
        <button type="button" className="gw-auth-toolbar__btn" onClick={() => setDarkMode((v) => !v)}>
          {darkMode ? <BulbOutlined /> : <BulbFilled />}
        </button>
      </div>

      <div className={`gw-auth-main gw-auth-main--${panelAlign}`}>
        {panelAlign === "left" ? renderFormCard() : null}
        {panelAlign !== "center" ? (
          <aside className="gw-auth-slogan">
            <img className="gw-auth-slogan__image" src={loginHeroImage} alt="云枢运维平台插画" />
            <div className="gw-auth-slogan__title">{introTitle}</div>
            <div className="gw-auth-slogan__desc">{introDesc}</div>
          </aside>
        ) : null}
        {panelAlign !== "left" ? renderFormCard() : null}
      </div>

      <Modal
        open={registerOpen}
        title={isZh ? "注册账号" : "Register Account"}
        footer={null}
        onCancel={() => setRegisterOpen(false)}
        destroyOnClose
        centered
        width={520}
        className="login-registerModal"
      >
        <Form<RegisterPayload> form={registerForm} layout="vertical" onFinish={handleRegister} size="large" autoComplete="off">
          <Form.Item label={isZh ? "用户名" : "Username"} name="username" rules={[{ required: true, min: 3, max: 64, message: isZh ? "用户名长度为3-64个字符" : "3-64 characters" }]}>
            <Input prefix={<UserOutlined />} placeholder={isZh ? "请输入用户名" : "Please enter username"} autoComplete="off" />
          </Form.Item>
          <Form.Item label={isZh ? "邮箱" : "Email"} name="email" rules={[{ required: true, type: "email", message: isZh ? "请输入正确的邮箱地址" : "Please enter valid email" }]}>
            <Input prefix={<MailOutlined />} placeholder={isZh ? "请输入邮箱地址" : "Please enter email"} autoComplete="off" />
          </Form.Item>
          <Form.Item label={isZh ? "昵称" : "Nickname"} name="nickname" rules={[{ required: true, max: 128, message: isZh ? "请输入昵称" : "Please enter nickname" }]}>
            <Input prefix={<UserOutlined />} placeholder={isZh ? "请输入昵称" : "Please enter nickname"} autoComplete="off" />
          </Form.Item>
          <Form.Item label={isZh ? "密码" : "Password"} name="password" rules={[{ required: true, min: 6, max: 64, message: isZh ? "密码长度为6-64个字符" : "6-64 characters" }]}>
            <Input.Password prefix={<LockOutlined />} placeholder={isZh ? "请输入密码" : "Please enter password"} autoComplete="new-password" />
          </Form.Item>

          <Form.Item
            label={isZh ? "验证码" : "Code"}
            name="code"
            getValueFromEvent={(event) => String(event?.target?.value ?? "").replace(/[^\d]/g, "").slice(0, 6)}
            rules={[
              { required: true, message: isZh ? "请输入验证码" : "Please enter code" },
              { pattern: /^\d{6}$/, message: isZh ? "验证码为6位数字" : "Code must be 6 digits" },
            ]}
          >
            <Input
              prefix={<SafetyCertificateOutlined />}
              placeholder={isZh ? "请输入验证码" : "Please enter code"}
              maxLength={6}
              autoComplete="off"
            />
          </Form.Item>
          <Button
            type="primary"
            ghost
            className="login-sendBtn"
            onClick={() => void handleSendRegisterCode()}
            loading={sendingCode}
            disabled={registerCodeCountdown > 0}
            style={{ marginTop: -10, marginBottom: 16 }}
          >
            {registerCodeCountdown > 0 ? `${registerCodeCountdown}s` : isZh ? "发送验证码" : "Send Code"}
          </Button>

          <Button htmlType="submit" type="primary" block disabled={submitting}>
            {isZh ? "提交注册" : "Submit Registration"}
          </Button>
        </Form>
      </Modal>
    </div>
  );
}

