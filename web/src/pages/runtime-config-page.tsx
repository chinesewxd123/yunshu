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
  const [k8sEventForm] = Form.useForm();
  const [minioForm] = Form.useForm();

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

      k8sEnabled: "k8s_event_forward_enabled",
      k8sBuffer: "k8s_event_forward_watcher_buffer_size",
      k8sInterval: "k8s_event_forward_worker_interval_seconds",
      k8sBatch: "k8s_event_forward_worker_batch_size",
      k8sRetries: "k8s_event_forward_worker_max_retries",

      minioEndpoint: "minio_endpoint",
      minioAccessKey: "minio_access_key",
      minioSecretKey: "minio_secret_key",
      minioBucket: "minio_bucket",
      minioUseSSL: "minio_use_ssl",
      minioPrefix: "minio_backup_prefix",
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
        const [k8sEnabled, k8sBuffer, k8sInterval, k8sBatch, k8sRetries] = await Promise.all([
          loadSingleton(dictTypes.k8sEnabled, { label: "启用 Event 转发", value: "false", status: 0, remark: "覆盖 k8s_event_forward.enabled（Worker 批处理周期内可热更新）", sort: 0 }),
          loadSingleton(dictTypes.k8sBuffer, { label: "监听缓冲", value: "1000", status: 0, remark: "覆盖 watcher_buffer_size（重启 watcher 生效）", sort: 0 }),
          loadSingleton(dictTypes.k8sInterval, { label: "批处理周期(秒)", value: "10", status: 0, remark: "覆盖 worker_interval_seconds", sort: 0 }),
          loadSingleton(dictTypes.k8sBatch, { label: "批大小", value: "50", status: 0, remark: "覆盖 worker_batch_size", sort: 0 }),
          loadSingleton(dictTypes.k8sRetries, { label: "最大重试", value: "3", status: 0, remark: "覆盖 worker_max_retries", sort: 0 }),
        ]);
        const [minioEndpoint, minioAccessKey, minioSecretKey, minioBucket, minioUseSSL, minioPrefix] = await Promise.all([
          loadSingleton(dictTypes.minioEndpoint, { label: "MinIO Endpoint", value: "127.0.0.1:9000", status: 0, remark: "MySQL 备份归档", sort: 0 }),
          loadSingleton(dictTypes.minioAccessKey, { label: "Access Key", value: "", status: 0, remark: "", sort: 0 }),
          loadSingleton(dictTypes.minioSecretKey, { label: "Secret Key", value: "", status: 0, remark: "", sort: 0 }),
          loadSingleton(dictTypes.minioBucket, { label: "Bucket", value: "yunshu-mysql-backup", status: 0, remark: "", sort: 0 }),
          loadSingleton(dictTypes.minioUseSSL, { label: "Use SSL", value: "false", status: 0, remark: "true/false", sort: 0 }),
          loadSingleton(dictTypes.minioPrefix, { label: "对象前缀", value: "mysql-backups", status: 0, remark: "", sort: 0 }),
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
        k8sEventForm.setFieldsValue({
          k8sEnabled: {
            ...k8sEnabled,
            status: k8sEnabled.status === 1,
            value: String(k8sEnabled.value || "").toLowerCase() === "true",
          },
          k8sBuffer: { ...k8sBuffer, status: k8sBuffer.status === 1, value: Number(k8sBuffer.value || 0) || 1000 },
          k8sInterval: { ...k8sInterval, status: k8sInterval.status === 1, value: Number(k8sInterval.value || 0) || 10 },
          k8sBatch: { ...k8sBatch, status: k8sBatch.status === 1, value: Number(k8sBatch.value || 0) || 50 },
          k8sRetries: { ...k8sRetries, status: k8sRetries.status === 1, value: Number(k8sRetries.value || 0) || 3 },
        });
        minioForm.setFieldsValue({
          minioEndpoint: { ...minioEndpoint, status: minioEndpoint.status === 1 },
          minioAccessKey: { ...minioAccessKey, status: minioAccessKey.status === 1 },
          minioSecretKey: { ...minioSecretKey, status: minioSecretKey.status === 1 },
          minioBucket: { ...minioBucket, status: minioBucket.status === 1 },
          minioUseSSL: {
            ...minioUseSSL,
            status: minioUseSSL.status === 1,
            value: String(minioUseSSL.value || "").toLowerCase() === "true",
          },
          minioPrefix: { ...minioPrefix, status: minioPrefix.status === 1 },
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
  }, [alertForm, k8sEventForm, mailForm, minioForm, dictTypes]);

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

  async function saveK8sEventForward() {
    const v = await k8sEventForm.validateFields();
    setSaving(true);
    try {
      const k8sEnabled = { ...v.k8sEnabled, value: v.k8sEnabled?.value ? "true" : "false" };
      await Promise.all([
        upsertSingleton(k8sEnabled),
        upsertSingleton({ ...v.k8sBuffer, value: String(Number(v.k8sBuffer?.value ?? 0) || 0) }),
        upsertSingleton({ ...v.k8sInterval, value: String(Number(v.k8sInterval?.value ?? 0) || 0) }),
        upsertSingleton({ ...v.k8sBatch, value: String(Number(v.k8sBatch?.value ?? 0) || 0) }),
        upsertSingleton({ ...v.k8sRetries, value: String(Number(v.k8sRetries?.value ?? 0) || 0) }),
      ]);
      message.success("K8s Event 转发配置已保存（开关与 Worker 参数下一批处理周期内生效；缓冲变更建议重启服务）");
    } finally {
      setSaving(false);
    }
  }

  async function saveMinio() {
    const v = await minioForm.validateFields();
    setSaving(true);
    try {
      const minioUseSSL = { ...v.minioUseSSL, value: v.minioUseSSL?.value ? "true" : "false" };
      await Promise.all([
        upsertSingleton(v.minioEndpoint),
        upsertSingleton(v.minioAccessKey),
        upsertSingleton(v.minioSecretKey),
        upsertSingleton(v.minioBucket),
        upsertSingleton(minioUseSSL),
        upsertSingleton(v.minioPrefix),
      ]);
      message.success("MinIO 配置已保存（立即生效）");
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
            {password ? <Input.Password placeholder={placeholder} autoComplete="new-password" /> : <Input placeholder={placeholder} autoComplete="off" />}
          </Form.Item>
          <Form.Item name={[name, "remark"]} label="说明" style={{ marginBottom: 0 }}>
            <Input placeholder="可选" autoComplete="off" />
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
            <div>这里直接读写数据字典中的配置项，服务端启动时会从字典覆盖 `config.yaml` 中的告警/邮件/K8s Event 转发配置。</div>
            <div>
              <Typography.Text strong>告警/邮件</Typography.Text> 保存后需重启服务；<Typography.Text strong>K8s Event 转发</Typography.Text> 开关与 Worker 参数在下一批处理周期内热更新。
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
              <Form form={alertForm} layout="vertical" autoComplete="off">
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
              <Form form={mailForm} layout="vertical" autoComplete="off">
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
          {
            key: "minio",
            label: "MinIO（MySQL 备份）",
            children: (
              <Form form={minioForm} layout="vertical" autoComplete="off">
                <Space direction="vertical" style={{ width: "100%" }} size="middle">
                  <DictItemEditor name="minioEndpoint" title="Endpoint（minio_endpoint）" placeholder="127.0.0.1:9000" />
                  <DictItemEditor name="minioAccessKey" title="Access Key" placeholder="minioadmin" />
                  <DictItemEditor name="minioSecretKey" title="Secret Key" placeholder="密钥" password />
                  <DictItemEditor name="minioBucket" title="Bucket" placeholder="yunshu-mysql-backup" />
                  <Card size="small" title="Use SSL（minio_use_ssl）">
                    <Form.Item name={["minioUseSSL", "id"]} hidden>
                      <Input />
                    </Form.Item>
                    <Form.Item name={["minioUseSSL", "dict_type"]} hidden>
                      <Input />
                    </Form.Item>
                    <Form.Item name={["minioUseSSL", "status"]} label="启用覆盖" valuePropName="checked">
                      <Switch />
                    </Form.Item>
                    <Form.Item name={["minioUseSSL", "value"]} label="SSL" valuePropName="checked">
                      <Switch />
                    </Form.Item>
                  </Card>
                  <DictItemEditor name="minioPrefix" title="对象前缀（minio_backup_prefix）" placeholder="mysql-backups" />
                  <Button type="primary" loading={saving} onClick={() => void saveMinio()}>
                    保存 MinIO 配置
                  </Button>
                </Space>
              </Form>
            ),
          },
          {
            key: "k8s-event-forward",
            label: "K8s Event 转发",
            children: (
              <Form form={k8sEventForm} layout="vertical" autoComplete="off">
                <Space direction="vertical" style={{ width: "100%" }} size="middle">
                  <Alert
                    type="info"
                    showIcon
                    message="入站复用告警平台 Webhook"
                    description="规则 webhook_url 留空或填 internal/alertmanager 时，Worker 将 POST 到 /api/v1/alerts/webhook/alertmanager，鉴权使用 alert_webhook_token。"
                  />
                  <Card size="small" title="启用（k8s_event_forward_enabled）">
                    <Space direction="vertical" style={{ width: "100%" }} size="small">
                      <Form.Item name={["k8sEnabled", "id"]} hidden>
                        <Input />
                      </Form.Item>
                      <Form.Item name={["k8sEnabled", "dict_type"]} hidden>
                        <Input />
                      </Form.Item>
                      <Space wrap>
                        <Form.Item name={["k8sEnabled", "status"]} label="启用覆盖" style={{ marginBottom: 0 }} valuePropName="checked">
                          <Switch checkedChildren="启用" unCheckedChildren="停用" />
                        </Form.Item>
                        <Form.Item name={["k8sEnabled", "label"]} label="显示名" style={{ marginBottom: 0, minWidth: 220 }}>
                          <Input />
                        </Form.Item>
                      </Space>
                      <Form.Item name={["k8sEnabled", "value"]} label="是否启用转发" valuePropName="checked">
                        <Switch checkedChildren="是" unCheckedChildren="否" />
                      </Form.Item>
                      <Form.Item name={["k8sEnabled", "remark"]} label="说明" style={{ marginBottom: 0 }}>
                        <Input placeholder="可选" />
                      </Form.Item>
                    </Space>
                  </Card>
                  <Card size="small" title="监听缓冲（k8s_event_forward_watcher_buffer_size）">
                    <Space direction="vertical" style={{ width: "100%" }} size="small">
                      <Form.Item name={["k8sBuffer", "id"]} hidden>
                        <Input />
                      </Form.Item>
                      <Form.Item name={["k8sBuffer", "dict_type"]} hidden>
                        <Input />
                      </Form.Item>
                      <Space wrap>
                        <Form.Item name={["k8sBuffer", "status"]} label="启用覆盖" style={{ marginBottom: 0 }} valuePropName="checked">
                          <Switch checkedChildren="启用" unCheckedChildren="停用" />
                        </Form.Item>
                      </Space>
                      <Form.Item name={["k8sBuffer", "value"]} label="缓冲大小" rules={[{ required: true }]}>
                        <InputNumber min={100} max={100000} style={{ width: "100%" }} />
                      </Form.Item>
                    </Space>
                  </Card>
                  <Card size="small" title="Worker 参数">
                    <Space direction="vertical" style={{ width: "100%" }} size="small">
                      {(
                        [
                          ["k8sInterval", "批处理周期(秒)", 1, 3600],
                          ["k8sBatch", "批大小", 1, 500],
                          ["k8sRetries", "最大重试", 0, 20],
                        ] as const
                      ).map(([name, title, min, max]) => (
                        <Card key={name} size="small" type="inner" title={title}>
                          <Form.Item name={[name, "id"]} hidden>
                            <Input />
                          </Form.Item>
                          <Form.Item name={[name, "dict_type"]} hidden>
                            <Input />
                          </Form.Item>
                          <Form.Item name={[name, "status"]} label="启用覆盖" valuePropName="checked">
                            <Switch />
                          </Form.Item>
                          <Form.Item name={[name, "value"]} label="值" rules={[{ required: true }]}>
                            <InputNumber min={min} max={max} style={{ width: "100%" }} />
                          </Form.Item>
                        </Card>
                      ))}
                    </Space>
                  </Card>
                  <Button type="primary" loading={saving} onClick={() => void saveK8sEventForward()}>
                    保存 K8s Event 转发配置
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

