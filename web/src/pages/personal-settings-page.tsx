import { Card, Form, Input, Menu, Button, message } from "antd";
import type { MenuProps } from "antd";
import { useEffect, useMemo, useState, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../contexts/auth-context";
import { changePassword, updateProfile } from "../services/auth";
import { clearAuthStorage } from "../services/storage";

type SettingsTab = "basic" | "password";

export function PersonalSettingsPage() {
  const { user, refreshUser } = useAuth();
  const navigate = useNavigate();
  const [tab, setTab] = useState<SettingsTab>("basic");
  const [profileLoading, setProfileLoading] = useState(false);
  const [passwordLoading, setPasswordLoading] = useState(false);
  const [profileForm] = Form.useForm<{ nickname: string; email?: string }>();
  const [passwordForm] = Form.useForm<{ old_password: string; new_password: string; confirm_password: string }>();

  // 使用受控组件状态，直接管理输入值
  const [oldPassword, setOldPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");

  const menuItems = useMemo<MenuProps["items"]>(
    () => [
      { key: "basic", label: "基本设置" },
      { key: "password", label: "修改密码" },
    ],
    [],
  );

  useEffect(() => {
    profileForm.setFieldsValue({
      nickname: user?.nickname ?? "",
      email: user?.email ?? "",
    });
  }, [profileForm, user?.nickname, user?.email]);

  // 切换到修改密码页面时，清空输入
  useEffect(() => {
    if (tab === "password") {
      setOldPassword("");
      setNewPassword("");
      setConfirmPassword("");
      passwordForm.resetFields();
    }
  }, [tab, passwordForm]);

  async function submitProfile() {
    const values = await profileForm.validateFields();
    setProfileLoading(true);
    try {
      await updateProfile({
        nickname: values.nickname,
        email: values.email?.trim() || undefined,
      });
      await refreshUser();
      message.success("基本信息已更新");
    } finally {
      setProfileLoading(false);
    }
  }

  const handleSubmitPassword = useCallback(async () => {
    console.log("当前状态值:", { oldPassword, newPassword, confirmPassword });

    if (!oldPassword) {
      message.error("请输入旧密码");
      return;
    }
    if (!newPassword) {
      message.error("请输入新密码");
      return;
    }
    if (newPassword !== confirmPassword) {
      message.error("两次输入的新密码不一致");
      return;
    }
    if (newPassword.length < 6) {
      message.error("新密码至少 6 位");
      return;
    }

    const payload = {
      old_password: oldPassword,
      new_password: newPassword,
    };
    console.log("发送的 payload:", JSON.stringify(payload));

    setPasswordLoading(true);
    try {
      await changePassword(payload);
      message.success("密码修改成功，请重新登录");
      // 清除登录状态并跳转到登录页
      clearAuthStorage();
      setTimeout(() => {
        navigate("/login");
      }, 1500);
    } catch (err: any) {
      console.error("密码修改失败:", err);
      const errorMessage = err?.response?.data?.message || err?.message || "密码修改失败";
      message.error(errorMessage);
    } finally {
      setPasswordLoading(false);
    }
  }, [oldPassword, newPassword, confirmPassword, navigate]);

  return (
    <Card className="table-card personal-settings-card">
      <div className="personal-settings">
        <aside className="personal-settings__sidebar">
          <Menu
            mode="inline"
            selectedKeys={[tab]}
            items={menuItems}
            onClick={(info) => setTab(info.key as SettingsTab)}
            className="personal-settings__menu"
          />
        </aside>
        <section className="personal-settings__content">
          {tab === "basic" ? (
            <div>
              <h3 className="personal-settings__title">基本设置</h3>
              <Form
                form={profileForm}
                layout="vertical"
                initialValues={{ nickname: user?.nickname ?? "", email: user?.email ?? "" }}
                className="personal-settings__form"
                autoComplete="off"
              >
                <Form.Item label="昵称" name="nickname" rules={[{ required: true, message: "请输入昵称" }]}>
                  <Input placeholder="请输入昵称" autoComplete="off" />
                </Form.Item>
                <Form.Item label="账号">
                  <Input value={user?.username ?? ""} disabled />
                </Form.Item>
                <Form.Item label="邮箱" name="email" rules={[{ type: "email", message: "请输入正确邮箱地址" }]}>
                  <Input placeholder="请输入邮箱地址" autoComplete="off" />
                </Form.Item>
                <Button type="primary" loading={profileLoading} onClick={() => void submitProfile()}>
                  更新基本信息
                </Button>
              </Form>
            </div>
          ) : (
            <div>
              <h3 className="personal-settings__title">修改密码</h3>
              <Form form={passwordForm} layout="vertical" className="personal-settings__form" autoComplete="off">
                <Form.Item label="旧密码" rules={[{ required: true, message: "请输入旧密码" }]}>
                  <Input.Password
                    value={oldPassword}
                    onChange={(e) => setOldPassword(e.target.value)}
                    placeholder="请输入旧密码"
                    autoComplete="off"
                  />
                </Form.Item>
                <Form.Item
                  label="新密码"
                  rules={[{ required: true, message: "请输入新密码" }, { min: 6, message: "新密码至少 6 位" }]}
                >
                  <Input.Password
                    value={newPassword}
                    onChange={(e) => setNewPassword(e.target.value)}
                    placeholder="请输入新密码"
                    autoComplete="off"
                  />
                </Form.Item>
                <Form.Item label="确认密码" rules={[{ required: true, message: "请再次输入新密码" }]}>
                  <Input.Password
                    value={confirmPassword}
                    onChange={(e) => setConfirmPassword(e.target.value)}
                    placeholder="请再次输入新密码"
                    autoComplete="off"
                  />
                </Form.Item>
                <Button type="primary" loading={passwordLoading} onClick={handleSubmitPassword}>
                  更新密码
                </Button>
              </Form>
            </div>
          )}
        </section>
      </div>
    </Card>
  );
}
