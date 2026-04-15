import { DeleteOutlined, EditOutlined, PlusOutlined, SendOutlined } from "@ant-design/icons";
import { Button, Card, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Switch, Table, Tag, message } from "antd";
import { useEffect, useState } from "react";
import {
  createAlertChannel,
  deleteAlertChannel,
  listAlertChannels,
  testAlertChannel,
  updateAlertChannel,
  type AlertChannelItem,
} from "../services/alerts";
import { formatDateTime } from "../utils/format";

export function AlertChannelsPage() {
  const [list, setList] = useState<AlertChannelItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [open, setOpen] = useState(false);
  const [current, setCurrent] = useState<AlertChannelItem | null>(null);
  const [form] = Form.useForm();
  const channelType = Form.useWatch("type", form);
  const wecomMode = Form.useWatch("wecom_mode", form);
  const dingMode = Form.useWatch("ding_mode", form);

  function parseChannelSettings(raw?: string) {
    if (!raw?.trim()) return {};
    try {
      const obj = JSON.parse(raw);
      if (obj && typeof obj === "object" && !Array.isArray(obj)) return obj as Record<string, unknown>;
      return {};
    } catch {
      return {};
    }
  }

  function stringifySettings(v: Record<string, unknown>) {
    const cleaned: Record<string, unknown> = {};
    for (const [k, val] of Object.entries(v)) {
      if (val === undefined || val === null) continue;
      if (typeof val === "string" && !val.trim()) continue;
      if (Array.isArray(val) && val.length === 0) continue;
      cleaned[k] = val;
    }
    return JSON.stringify(cleaned);
  }

  async function load() {
    setLoading(true);
    try {
      const res = await listAlertChannels();
      setList(res.list ?? []);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, []);

  function openCreate() {
    setCurrent(null);
    form.setFieldsValue({
      name: "",
      type: "generic_webhook",
      url: "",
      secret: "",
      headers_json: "",
      extra_headers_json: "{}",
      at_mobiles: [],
      at_user_ids: [],
      is_at_all: false,
      wecom_mode: "robot",
      corp_id: "",
      corp_secret: "",
      agent_id: "",
      ding_mode: "robot",
      ding_app_key: "",
      ding_app_secret: "",
      ding_chat_id: "",
      ding_sign_secret: "",
      email_to: [],
      enabled: true,
      timeout_ms: 5000,
    });
    setOpen(true);
  }

  function openEdit(row: AlertChannelItem) {
    setCurrent(row);
    const settings = parseChannelSettings(row.headers_json || "");
    const atMobiles = Array.isArray(settings.atMobiles) ? (settings.atMobiles as unknown[]).map((v) => String(v)) : [];
    const atUserIds = Array.isArray(settings.atUserIds) ? (settings.atUserIds as unknown[]).map((v) => String(v)) : [];
    const emailTo =
      Array.isArray(settings.to)
        ? (settings.to as unknown[]).map((v) => String(v))
        : typeof settings.to === "string"
          ? String(settings.to).split(",").map((s) => s.trim()).filter(Boolean)
          : [];
    const extra = { ...settings };
    delete (extra as any).atMobiles;
    delete (extra as any).atUserIds;
    delete (extra as any).isAtAll;
    delete (extra as any).corpID;
    delete (extra as any).corpSecret;
    delete (extra as any).chatId;
    delete (extra as any).signSecret;
    delete (extra as any).to;
    form.setFieldsValue({
      name: row.name,
      type: row.type || "generic_webhook",
      url: row.url,
      secret: row.secret || "",
      headers_json: row.headers_json || "",
      extra_headers_json: stringifySettings(extra),
      at_mobiles: atMobiles,
      at_user_ids: atUserIds,
      is_at_all: !!settings.isAtAll,
      wecom_mode: typeof settings.wecomMode === "string" ? settings.wecomMode : "robot",
      corp_id: typeof settings.corpID === "string" ? settings.corpID : "",
      corp_secret: typeof settings.corpSecret === "string" ? settings.corpSecret : "",
      agent_id: typeof settings.agentId === "string" ? settings.agentId : "",
      ding_mode: typeof settings.dingMode === "string" ? settings.dingMode : "robot",
      ding_app_key: typeof settings.appKey === "string" ? settings.appKey : "",
      ding_app_secret: typeof settings.appSecret === "string" ? settings.appSecret : "",
      ding_chat_id: typeof settings.chatId === "string" ? settings.chatId : "",
      ding_sign_secret: typeof settings.signSecret === "string" ? settings.signSecret : "",
      email_to: emailTo,
      enabled: row.enabled,
      timeout_ms: row.timeout_ms || 5000,
    });
    setOpen(true);
  }

  async function submit() {
    const values = await form.validateFields();
    setSaving(true);
    try {
      const extraSettings = parseChannelSettings(values.extra_headers_json || "{}");
      const settings: Record<string, unknown> = { ...extraSettings };
      const atMobiles = (values.at_mobiles ?? []).map((v: string) => v.trim()).filter(Boolean);
      const atUserIds = (values.at_user_ids ?? []).map((v: string) => v.trim()).filter(Boolean);
      const emailTo = (values.email_to ?? []).map((v: string) => v.trim()).filter(Boolean);
      if (atMobiles.length > 0) settings.atMobiles = atMobiles;
      if (atUserIds.length > 0) settings.atUserIds = atUserIds;
      if (values.is_at_all) settings.isAtAll = true;
      if ((values.wecom_mode || "").trim()) settings.wecomMode = values.wecom_mode.trim();
      if ((values.corp_id || "").trim()) settings.corpID = values.corp_id.trim();
      if ((values.corp_secret || "").trim()) settings.corpSecret = values.corp_secret.trim();
      if ((values.agent_id || "").trim()) settings.agentId = values.agent_id.trim();
      if ((values.ding_mode || "").trim()) settings.dingMode = values.ding_mode.trim();
      if ((values.ding_app_key || "").trim()) settings.appKey = values.ding_app_key.trim();
      if ((values.ding_app_secret || "").trim()) settings.appSecret = values.ding_app_secret.trim();
      if ((values.ding_chat_id || "").trim()) settings.chatId = values.ding_chat_id.trim();
      if ((values.ding_sign_secret || "").trim()) settings.signSecret = values.ding_sign_secret.trim();
      if (values.type === "email" && emailTo.length > 0) settings.to = emailTo;
      const payload = {
        name: values.name,
        type: values.type,
        url: values.url,
        secret: values.secret,
        headers_json: stringifySettings(settings),
        enabled: !!values.enabled,
        timeout_ms: Number(values.timeout_ms || 5000),
      };
      if (current) {
        await updateAlertChannel(current.id, payload);
        message.success("告警通道已更新");
      } else {
        await createAlertChannel(payload);
        message.success("告警通道已创建");
      }
      setOpen(false);
      await load();
    } finally {
      setSaving(false);
    }
  }

  return (
    <Card className="table-card" title="Webhook 告警通道">
      <div style={{ display: "flex", justifyContent: "space-between", marginBottom: 12 }}>
        <Space />
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
          新建通道
        </Button>
      </div>
      <Table
        rowKey="id"
        loading={loading}
        dataSource={list}
        pagination={{ pageSize: 10 }}
        columns={[
          { title: "名称", dataIndex: "name", width: 180 },
          { title: "类型", dataIndex: "type", width: 140 },
          { title: "Webhook URL", dataIndex: "url", ellipsis: true },
          { title: "超时(ms)", dataIndex: "timeout_ms", width: 110 },
          {
            title: "状态",
            dataIndex: "enabled",
            width: 90,
            render: (v: boolean) => (v ? <Tag color="success">启用</Tag> : <Tag>停用</Tag>),
          },
          { title: "更新时间", dataIndex: "updated_at", width: 170, render: (v: string) => formatDateTime(v) },
          {
            title: "操作",
            key: "action",
            width: 280,
            render: (_: unknown, row: AlertChannelItem) => (
              <Space>
                <Button type="link" icon={<SendOutlined />} onClick={() => void testAlertChannel(row.id).then(() => message.success("测试发送成功"))}>
                  测试
                </Button>
                <Button type="link" icon={<EditOutlined />} onClick={() => openEdit(row)}>
                  编辑
                </Button>
                <Popconfirm
                  title="确认删除该通道吗？"
                  onConfirm={() =>
                    void (async () => {
                      await deleteAlertChannel(row.id);
                      message.success("已删除");
                      await load();
                    })()
                  }
                >
                  <Button type="link" danger icon={<DeleteOutlined />}>
                    删除
                  </Button>
                </Popconfirm>
              </Space>
            ),
          },
        ]}
      />

      <Modal
        title={current ? `编辑通道 #${current.id}` : "新建告警通道"}
        open={open}
        onCancel={() => setOpen(false)}
        onOk={() => void submit()}
        confirmLoading={saving}
        destroyOnClose
        width={760}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="通道名称" rules={[{ required: true, message: "请输入通道名称" }]}>
            <Input />
          </Form.Item>
          <Form.Item name="type" label="通道类型" rules={[{ required: true }]}>
            <Select
              options={[
                { label: "generic_webhook", value: "generic_webhook" },
                { label: "wechat_work", value: "wechat_work" },
                { label: "dingding", value: "dingding" },
                { label: "email", value: "email" },
              ]}
            />
          </Form.Item>
          <Form.Item name="url" label="Webhook URL（email、钉钉app_chat 可留空）" rules={[{ type: "url", message: "URL 格式不正确" }]}>
            <Input />
          </Form.Item>
          <Form.Item name="secret" label="签名密钥（可选）">
            <Input.Password allowClear />
          </Form.Item>
          <Form.Item name="at_mobiles" label="@手机号（钉钉/企业微信）">
            <Select mode="tags" tokenSeparators={[",", " "]} placeholder="例如：13800000000" />
          </Form.Item>
          <Form.Item name="at_user_ids" label="@用户ID（钉钉/企业微信，可选）">
            <Select mode="tags" tokenSeparators={[",", " "]} placeholder="例如：user001" />
          </Form.Item>
          {channelType === "dingding" ? (
            <Form.Item name="is_at_all" label="@所有人（钉钉）" valuePropName="checked">
              <Switch />
            </Form.Item>
          ) : null}

          {channelType === "wechat_work" || channelType === "wechat" ? (
            <Form.Item name="wecom_mode" label="企业微信模式">
              <Select options={[{ label: "robot（群机器人）", value: "robot" }, { label: "app（企业应用消息）", value: "app" }]} />
            </Form.Item>
          ) : null}
          {channelType === "wechat_work" || channelType === "wechat" ? (
            <>
              <Form.Item
                name="corp_id"
                label="企业微信 corpID（app 模式：手机号查 userid 用）"
                hidden={wecomMode !== "app"}
              >
                <Input placeholder="wxcorp..." />
              </Form.Item>
              <Form.Item
                name="corp_secret"
                label="企业微信 corpSecret（app 模式：手机号查 userid 用）"
                hidden={wecomMode !== "app"}
              >
                <Input.Password allowClear />
              </Form.Item>
              <Form.Item name="agent_id" label="企业微信 agentId（app 模式必填）" hidden={wecomMode !== "app"}>
                <Input placeholder="1000002" />
              </Form.Item>
            </>
          ) : null}
          {channelType === "dingding" ? (
            <Form.Item name="ding_mode" label="钉钉模式">
              <Select options={[{ label: "robot（群机器人）", value: "robot" }, { label: "app_chat（官方会话API）", value: "app_chat" }]} />
            </Form.Item>
          ) : null}
          {channelType === "dingding" ? (
            <>
              <Form.Item name="ding_app_key" label="钉钉 appKey（app_chat 模式）" hidden={dingMode !== "app_chat"}>
                <Input />
              </Form.Item>
              <Form.Item name="ding_app_secret" label="钉钉 appSecret（app_chat 模式）" hidden={dingMode !== "app_chat"}>
                <Input.Password allowClear />
              </Form.Item>
              <Form.Item name="ding_chat_id" label="钉钉 chatId（app_chat 模式必填 / robot 可不填）">
                <Input placeholder="chatxxxxx" />
              </Form.Item>
              <Form.Item name="ding_sign_secret" label="钉钉 signSecret（robot 加签可选）" hidden={dingMode !== "robot"}>
                <Input.Password allowClear />
              </Form.Item>
            </>
          ) : null}

          {channelType === "email" ? (
            <Form.Item
              name="email_to"
              label="邮件接收人（email 通道）"
              rules={[
                { required: true, message: "请填写至少一个收件邮箱" },
                {
                  validator: async (_, value) => {
                    const arr = (value ?? []).map((v: string) => String(v).trim()).filter(Boolean);
                    if (arr.length === 0) {
                      throw new Error("请填写至少一个收件邮箱");
                    }
                    const re = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
                    for (const e of arr) {
                      if (!re.test(e)) {
                        throw new Error(`邮箱格式不正确：${e}`);
                      }
                    }
                  },
                },
              ]}
            >
              <Select mode="tags" tokenSeparators={[",", " ", ";"]} placeholder="输入后回车，或粘贴 a@xx.com,b@yy.com" />
            </Form.Item>
          ) : null}
          <Form.Item
            name="extra_headers_json"
            label="额外配置 JSON（可选）"
            extra={'用于自定义请求头。示例：{"headers":{"Authorization":"Bearer xxxxx"}}'}
          >
            <Input.TextArea rows={4} />
          </Form.Item>
          <Space style={{ width: "100%" }} size="large">
            <Form.Item name="timeout_ms" label="超时(ms)" style={{ width: 200 }}>
              <InputNumber min={1000} max={60000} style={{ width: "100%" }} />
            </Form.Item>
            <Form.Item name="enabled" label="启用" valuePropName="checked">
              <Switch />
            </Form.Item>
          </Space>
        </Form>
      </Modal>
    </Card>
  );
}

