import { Alert, Button, Card, Form, Input, InputNumber, Space, Switch, Tabs, Typography, message } from "antd";
import { useEffect, useMemo, useState } from "react";
import { createDictEntry, getDictEntries, updateDictEntry, type DictEntryItem } from "../services/dict";

type DictSingleton = {
  dict_type: string;
  label: string;
  value: string;
  status: number | boolean; // 0/1 or checked
  remark?: string;
  sort?: number;
  id?: number;
};

async function loadSingleton(dictType: string, fallback: Omit<DictSingleton, "dict_type">): Promise<DictSingleton> {
  const res = await getDictEntries({ dict_type: dictType, page: 1, page_size: 20 });
  const list = res.list ?? [];
  const enabled = list.find((x) => x.status === 1);
  const first = enabled ?? list[0];
  if (!first) {
    return { dict_type: dictType, ...fallback };
  }
  return {
    id: first.id,
    dict_type: first.dict_type,
    label: first.label,
    value: first.value ?? "",
    status: first.status,
    remark: first.remark,
    sort: first.sort,
  };
}

async function upsertSingleton(payload: DictSingleton): Promise<DictEntryItem> {
  const status =
    typeof payload.status === "boolean" ? (payload.status ? 1 : 0) : payload.status == null ? 1 : Number(payload.status) ? 1 : 0;
  const body = {
    dict_type: payload.dict_type,
    label: String(payload.label || "").trim() || payload.dict_type,
    value: String(payload.value ?? ""),
    status,
    sort: payload.sort ?? 0,
    remark: String(payload.remark || "").trim(),
  };
  if (payload.id) {
    return updateDictEntry(payload.id, body);
  }
  return createDictEntry(body);
}

export function RuntimeConfigPage() {
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);

  const [alertForm] = Form.useForm();
  const [mailForm] = Form.useForm();

  const dictTypes = useMemo(
    () => ({
      alertWebhookToken: "alert_webhook_token",
      alertPromURL: "alert_enrich_prometheus_url",
      alertPromToken: "alert_enrich_prometheus_token",

      mailHost: "mail_host",
      mailPort: "mail_port",
      mailUsername: "mail_username",
      mailPassword: "mail_password",
      mailFromEmail: "mail_from_email",
      mailFromName: "mail_from_name",
      mailUseTLS: "mail_use_tls",
    }),
    [],
  );

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      setLoading(true);
      try {
        const [webhook, promURL, promToken] = await Promise.all([
          loadSingleton(dictTypes.alertWebhookToken, { label: "Webhook Token", value: "", status: 1, remark: "覆盖 alert.webhook_token（重启服务生效）", sort: 0 }),
          loadSingleton(dictTypes.alertPromURL, { label: "Prometheus URL", value: "", status: 1, remark: "覆盖 alert.prometheus_url（重启服务生效）", sort: 0 }),
          loadSingleton(dictTypes.alertPromToken, { label: "Prometheus Token", value: "", status: 0, remark: "覆盖 alert.prometheus_token（重启服务生效）", sort: 0 }),
        ]);
        const [mailHost, mailPort, mailUsername, mailPassword, mailFromEmail, mailFromName, mailUseTLS] = await Promise.all([
          loadSingleton(dictTypes.mailHost, { label: "SMTP Host", value: "smtp.163.com", status: 1, remark: "覆盖 mail.host（重启服务生效）", sort: 0 }),
          loadSingleton(dictTypes.mailPort, { label: "SMTP Port", value: "465", status: 1, remark: "覆盖 mail.port（重启服务生效）", sort: 0 }),
          loadSingleton(dictTypes.mailUsername, { label: "SMTP Username", value: "", status: 0, remark: "覆盖 mail.username（重启服务生效）", sort: 0 }),
          loadSingleton(dictTypes.mailPassword, { label: "SMTP Password", value: "", status: 0, remark: "覆盖 mail.password（重启服务生效）", sort: 0 }),
          loadSingleton(dictTypes.mailFromEmail, { label: "From Email", value: "", status: 0, remark: "覆盖 mail.from_email（重启服务生效）", sort: 0 }),
          loadSingleton(dictTypes.mailFromName, { label: "From Name", value: "", status: 0, remark: "覆盖 mail.from_name（重启服务生效）", sort: 0 }),
          loadSingleton(dictTypes.mailUseTLS, { label: "Use TLS", value: "true", status: 1, remark: "覆盖 mail.use_tls（true/false，重启服务生效）", sort: 0 }),
        ]);

        if (cancelled) return;
        alertForm.setFieldsValue({
          webhook: { ...webhook, status: webhook.status === 1 },
          promURL: { ...promURL, status: promURL.status === 1 },
          promToken: { ...promToken, status: promToken.status === 1 },
        });
        mailForm.setFieldsValue({
          mailHost: { ...mailHost, status: mailHost.status === 1 },
          mailPort: { ...mailPort, status: mailPort.status === 1, value: Number(mailPort.value || 0) || 465 },
          mailUsername: { ...mailUsername, status: mailUsername.status === 1 },
          mailPassword: { ...mailPassword, status: mailPassword.status === 1 },
          mailFromEmail: { ...mailFromEmail, status: mailFromEmail.status === 1 },
          mailFromName: { ...mailFromName, status: mailFromName.status === 1 },
          mailUseTLS: {
            ...mailUseTLS,
            status: mailUseTLS.status === 1,
            value: String(mailUseTLS.value || "").toLowerCase() === "true",
          },
        });
      } catch (e) {
        if (!cancelled) message.error(`加载配置失败：${e instanceof Error ? e.message : String(e)}`);
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [alertForm, mailForm, dictTypes]);

  async function saveAlert() {
    const v = await alertForm.validateFields();
    setSaving(true);
    try {
      await Promise.all([
        upsertSingleton(v.webhook),
        upsertSingleton(v.promURL),
        upsertSingleton(v.promToken),
      ]);
      message.success("告警配置已保存（重启服务生效）");
    } finally {
      setSaving(false);
    }
  }

  async function saveMail() {
    const v = await mailForm.validateFields();
    setSaving(true);
    try {
      const mailPort = { ...v.mailPort, value: String(Number(v.mailPort?.value ?? 0) || 0) };
      const mailUseTLS = { ...v.mailUseTLS, value: v.mailUseTLS?.value ? "true" : "false" };
      await Promise.all([
        upsertSingleton(v.mailHost),
        upsertSingleton(mailPort),
        upsertSingleton(v.mailUsername),
        upsertSingleton(v.mailPassword),
        upsertSingleton(v.mailFromEmail),
        upsertSingleton(v.mailFromName),
        upsertSingleton(mailUseTLS),
      ]);
      message.success("邮件配置已保存（重启服务生效）");
    } finally {
      setSaving(false);
    }
  }

  function DictItemEditor({ name, title, placeholder, password }: { name: string; title: string; placeholder?: string; password?: boolean }) {
    return (
      <Card size="small" title={title} styles={{ body: { paddingTop: 8 } }}>
        <Space direction="vertical" style={{ width: "100%" }} size="small">
          <Form.Item name={[name, "id"]} hidden>
            <Input />
          </Form.Item>
          <Form.Item name={[name, "dict_type"]} hidden>
            <Input />
          </Form.Item>
          <Space wrap style={{ width: "100%" }}>
            <Form.Item name={[name, "status"]} label="启用覆盖" style={{ marginBottom: 0 }} valuePropName="checked">
              <Switch checkedChildren="启用" unCheckedChildren="停用" />
            </Form.Item>
            <Form.Item name={[name, "label"]} label="显示名" style={{ marginBottom: 0, minWidth: 220 }}>
              <Input placeholder="用于数据字典列表展示" />
            </Form.Item>
          </Space>
          <Form.Item name={[name, "value"]} label="值" style={{ marginBottom: 0 }}>
            {password ? <Input.Password placeholder={placeholder} /> : <Input placeholder={placeholder} />}
          </Form.Item>
          <Form.Item name={[name, "remark"]} label="说明" style={{ marginBottom: 0 }}>
            <Input placeholder="可选" />
          </Form.Item>
        </Space>
      </Card>
    );
  }

  return (
    <Card className="table-card" title="运行期配置中心（数据字典）" loading={loading}>
      <Alert
        type="info"
        showIcon
        message="说明"
        description={
          <div>
            <div>这里直接读写数据字典中的配置项，服务端启动时会从字典覆盖 `config.yaml` 中的告警/邮件配置。</div>
            <div>
              <Typography.Text strong>保存后需要重启服务</Typography.Text> 才会生效（当前策略为启动读取一次，稳定优先）。
            </div>
          </div>
        }
        style={{ marginBottom: 12 }}
      />

      <Tabs
        items={[
          {
            key: "alert",
            label: "告警配置",
            children: (
              <Form form={alertForm} layout="vertical">
                <Space direction="vertical" style={{ width: "100%" }} size="middle">
                  <DictItemEditor name="webhook" title="Webhook Token（alert.webhook_token）" placeholder="用于 /alerts/webhook/alertmanager 鉴权" password />
                  <DictItemEditor name="promURL" title="Prometheus URL（alert.prometheus_url）" placeholder="例如 http://prometheus:9090" />
                  <DictItemEditor name="promToken" title="Prometheus Token（alert.prometheus_token，可选）" placeholder="Bearer Token（可留空）" password />
                  <Button type="primary" loading={saving} onClick={() => void saveAlert()}>
                    保存告警配置
                  </Button>
                </Space>
              </Form>
            ),
          },
          {
            key: "mail",
            label: "邮件配置",
            children: (
              <Form form={mailForm} layout="vertical">
                <Space direction="vertical" style={{ width: "100%" }} size="middle">
                  <DictItemEditor name="mailHost" title="SMTP Host（mail.host）" placeholder="例如 smtp.163.com" />
                  <Card size="small" title="SMTP Port（mail.port）">
                    <Space direction="vertical" style={{ width: "100%" }} size="small">
                      <Form.Item name={["mailPort", "id"]} hidden>
                        <Input />
                      </Form.Item>
                      <Form.Item name={["mailPort", "dict_type"]} hidden>
                        <Input />
                      </Form.Item>
                      <Space wrap>
                        <Form.Item name={["mailPort", "status"]} label="启用覆盖" style={{ marginBottom: 0 }} valuePropName="checked">
                          <Switch checkedChildren="启用" unCheckedChildren="停用" />
                        </Form.Item>
                        <Form.Item name={["mailPort", "label"]} label="显示名" style={{ marginBottom: 0, minWidth: 220 }}>
                          <Input />
                        </Form.Item>
                      </Space>
                      <Form.Item name={["mailPort", "value"]} label="端口" rules={[{ required: true, message: "请输入端口" }]}>
                        <InputNumber min={1} max={65535} style={{ width: "100%" }} />
                      </Form.Item>
                      <Form.Item name={["mailPort", "remark"]} label="说明" style={{ marginBottom: 0 }}>
                        <Input placeholder="可选" />
                      </Form.Item>
                    </Space>
                  </Card>
                  <Card size="small" title="Use TLS（mail.use_tls）">
                    <Space direction="vertical" style={{ width: "100%" }} size="small">
                      <Form.Item name={["mailUseTLS", "id"]} hidden>
                        <Input />
                      </Form.Item>
                      <Form.Item name={["mailUseTLS", "dict_type"]} hidden>
                        <Input />
                      </Form.Item>
                      <Space wrap>
                        <Form.Item name={["mailUseTLS", "status"]} label="启用覆盖" style={{ marginBottom: 0 }} valuePropName="checked">
                          <Switch checkedChildren="启用" unCheckedChildren="停用" />
                        </Form.Item>
                        <Form.Item name={["mailUseTLS", "label"]} label="显示名" style={{ marginBottom: 0, minWidth: 220 }}>
                          <Input />
                        </Form.Item>
                      </Space>
                      <Form.Item name={["mailUseTLS", "value"]} label="是否启用 TLS" valuePropName="checked">
                        <Switch checkedChildren="是" unCheckedChildren="否" />
                      </Form.Item>
                      <Form.Item name={["mailUseTLS", "remark"]} label="说明" style={{ marginBottom: 0 }}>
                        <Input placeholder="可选" />
                      </Form.Item>
                    </Space>
                  </Card>
                  <DictItemEditor name="mailUsername" title="SMTP Username（mail.username）" placeholder="例如 user@example.com" />
                  <DictItemEditor name="mailPassword" title="SMTP Password（mail.password）" placeholder="密码/授权码" password />
                  <DictItemEditor name="mailFromEmail" title="From Email（mail.from_email）" placeholder="例如 noreply@example.com" />
                  <DictItemEditor name="mailFromName" title="From Name（mail.from_name）" placeholder="例如 YunShu" />
                  <Button type="primary" loading={saving} onClick={() => void saveMail()}>
                    保存邮件配置
                  </Button>
                </Space>
              </Form>
            ),
          },
        ]}
      />
    </Card>
  );
}

