import { Card, Form, Input, Menu, Button, message } from "antd";
import type { MenuProps } from "antd";
import { useEffect, useMemo, useState } from "react";
import { useAuth } from "../contexts/auth-context";
import { changePassword, updateProfile } from "../services/auth";

type SettingsTab = "basic" | "password";

export function PersonalSettingsPage() {
  const { user, refreshUser } = useAuth();
  const [tab, setTab] = useState<SettingsTab>("basic");
  const [profileLoading, setProfileLoading] = useState(false);
  const [passwordLoading, setPasswordLoading] = useState(false);
  const [profileForm] = Form.useForm<{ nickname: string; email?: string }>();
  const [passwordForm] = Form.useForm<{ old_password: string; new_password: string; confirm_password: string }>();

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

  async function submitPassword() {
    const values = await passwordForm.validateFields();
    setPasswordLoading(true);
    try {
      await changePassword({
        old_password: values.old_password,
        new_password: values.new_password,
      });
      passwordForm.resetFields();
      message.success("密码修改成功");
    } finally {
      setPasswordLoading(false);
    }
  }

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
              >
                <Form.Item label="昵称" name="nickname" rules={[{ required: true, message: "请输入昵称" }]}>
                  <Input placeholder="请输入昵称" />
                </Form.Item>
                <Form.Item label="账号">
                  <Input value={user?.username ?? ""} disabled />
                </Form.Item>
                <Form.Item label="邮箱" name="email" rules={[{ type: "email", message: "请输入正确邮箱地址" }]}>
                  <Input placeholder="请输入邮箱地址" />
                </Form.Item>
                <Button type="primary" loading={profileLoading} onClick={() => void submitProfile()}>
                  更新基本信息
                </Button>
              </Form>
            </div>
          ) : (
            <div>
              <h3 className="personal-settings__title">修改密码</h3>
              <Form form={passwordForm} layout="vertical" className="personal-settings__form">
                <Form.Item label="旧密码" name="old_password" rules={[{ required: true, message: "请输入旧密码" }]}>
                  <Input.Password placeholder="请输入旧密码" />
                </Form.Item>
                <Form.Item
                  label="新密码"
                  name="new_password"
                  rules={[{ required: true, message: "请输入新密码" }, { min: 6, message: "新密码至少 6 位" }]}
                >
                  <Input.Password placeholder="请输入新密码" />
                </Form.Item>
                <Form.Item
                  label="确认密码"
                  name="confirm_password"
                  dependencies={["new_password"]}
                  rules={[
                    { required: true, message: "请再次输入新密码" },
                    ({ getFieldValue }) => ({
                      validator(_, value) {
                        if (!value || getFieldValue("new_password") === value) {
                          return Promise.resolve();
                        }
                        return Promise.reject(new Error("两次输入的新密码不一致"));
                      },
                    }),
                  ]}
                >
                  <Input.Password placeholder="请再次输入新密码" />
                </Form.Item>
                <Button type="primary" loading={passwordLoading} onClick={() => void submitPassword()}>
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
