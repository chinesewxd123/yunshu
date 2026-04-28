import { DeleteOutlined, EditOutlined, PlusOutlined, SendOutlined } from "@ant-design/icons";
import { Button, Card, Collapse, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Switch, Table, Tag, message } from "antd";
import { useEffect, useRef, useState } from "react";
import {
  createAlertChannel,
  deleteAlertChannel,
  listAlertChannels,
  previewAlertChannelTemplate,
  testAlertChannel,
  updateAlertChannel,
  type AlertChannelItem,
  type AlertTemplatePreviewResult,
} from "../services/alerts";
import { useDictOptions } from "../hooks/use-dict-options";
import { formatDateTime } from "../utils/format";
import { DictFillSelect } from "../components/dict-fill-select";
import { getProjects, type ProjectItem } from "../services/projects";

type TemplatePreviewStatus = "firing" | "resolved";
const DEFAULT_FIRING_TEMPLATE =
  "【{{.StatusText}}】**{{.Title}}**\n\n- 级别：`{{.Severity}}`\n\n- 项目：`{{.ProjectName}}`\n\n- 集群：`{{.Cluster}}`\n\n- 摘要：{{.Summary}}\n\n- 时间：{{.OccurredAt}}\n\n- 标签：{{.LabelsText}}";
const DEFAULT_RESOLVED_TEMPLATE =
  "【{{.StatusText}}】**{{.Title}}**\n\n- 级别：`{{.Severity}}`\n\n- 项目：`{{.ProjectName}}`\n\n- 集群：`{{.Cluster}}`\n\n- 恢复时间：{{.OccurredAt}}\n\n- 开始：{{.StartsAt}}\n\n- 结束：{{.EndsAt}}\n\n- 摘要：{{.Summary}}";
const SIMPLE_FIRING_TEMPLATE =
  "【告警】{{.Title}}\n级别：{{.Severity}}\n项目：{{.ProjectName}}\n摘要：{{.Summary}}";
const SIMPLE_RESOLVED_TEMPLATE =
  "【恢复】{{.Title}}\n开始：{{.StartsAt}}\n结束：{{.EndsAt}}\n项目：{{.ProjectName}}";
const DETAILED_FIRING_TEMPLATE =
  "【{{.StatusText}}】{{.Title}}\n级别：{{.Severity}}\n项目：{{.ProjectName}}\n集群：{{.Cluster}}\n摘要：{{.Summary}}\n描述：{{.Description}}\n时间：{{.OccurredAt}}\n标签：{{.LabelsText}}\n链接：{{.GeneratorURL}}";
const DETAILED_RESOLVED_TEMPLATE =
  "【{{.StatusText}}】{{.Title}}\n级别：{{.Severity}}\n项目：{{.ProjectName}}\n集群：{{.Cluster}}\n开始：{{.StartsAt}}\n结束：{{.EndsAt}}\n摘要：{{.Summary}}\n标签：{{.LabelsText}}";
const CHANNEL_PRESET_OPTIONS = [
  { label: "新手简版（推荐）", value: "simple" },
  { label: "标准版（默认）", value: "default" },
  { label: "详细排障版", value: "detailed" },
];

function presetTemplateByMode(mode?: string) {
  switch (String(mode || "").trim()) {
    case "simple":
      return { firing: SIMPLE_FIRING_TEMPLATE, resolved: SIMPLE_RESOLVED_TEMPLATE };
    case "detailed":
      return { firing: DETAILED_FIRING_TEMPLATE, resolved: DETAILED_RESOLVED_TEMPLATE };
    default:
      return { firing: DEFAULT_FIRING_TEMPLATE, resolved: DEFAULT_RESOLVED_TEMPLATE };
  }
}

export function AlertChannelsPage() {
  const [list, setList] = useState<AlertChannelItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [open, setOpen] = useState(false);
  const [testOpen, setTestOpen] = useState(false);
  const [testSending, setTestSending] = useState(false);
  const [testRow, setTestRow] = useState<AlertChannelItem | null>(null);
  const [templateOpen, setTemplateOpen] = useState(false);
  const [templateSaving, setTemplateSaving] = useState(false);
  const [templateChannelID, setTemplateChannelID] = useState<number | undefined>();
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [previewLoading, setPreviewLoading] = useState(false);
  const [previewError, setPreviewError] = useState("");
  const [previewResult, setPreviewResult] = useState<AlertTemplatePreviewResult | null>(null);
  const [editingTemplateTarget, setEditingTemplateTarget] = useState<TemplatePreviewStatus>("firing");
  const [firingPreset, setFiringPreset] = useState<string>("default");
  const [resolvedPreset, setResolvedPreset] = useState<string>("default");
  const [current, setCurrent] = useState<AlertChannelItem | null>(null);
  const [form] = Form.useForm();
  const [testForm] = Form.useForm();
  const [templateForm] = Form.useForm();
  const firingTemplateRef = useRef<any>(null);
  const resolvedTemplateRef = useRef<any>(null);
  const channelType = Form.useWatch("type", form);
  const wecomMode = Form.useWatch("wecom_mode", form);
  const dingMode = Form.useWatch("ding_mode", form);
  const templateFiring = Form.useWatch("template_firing", templateForm);
  const templateResolved = Form.useWatch("template_resolved", templateForm);
  const previewStatus = (Form.useWatch("preview_status", templateForm) || "firing") as TemplatePreviewStatus;
  const previewProjectID = Form.useWatch("preview_project_id", templateForm) as number | undefined;
  const previewRawPayloadJSON = Form.useWatch("preview_raw_payload_json", templateForm) as string | undefined;
  const webhookURLDictOptions = useDictOptions("alert_webhook_url");
  const wecomWebhookURLDictOptions = useDictOptions("wecom_webhook_url");
  const dingtalkWebhookURLDictOptions = useDictOptions("dingtalk_webhook_url");
  const channelTypeOptions = useDictOptions("alert_channel_type");
  const wecomModeOptions = useDictOptions("wecom_notify_mode");
  const dingModeOptions = useDictOptions("dingtalk_notify_mode");
  const wecomCorpIDOptions = useDictOptions("wecom_corp_id");
  const wecomCorpSecretOptions = useDictOptions("wecom_corp_secret");
  const wecomAgentIDOptions = useDictOptions("wecom_agent_id");
  const dingAppKeyOptions = useDictOptions("dingtalk_app_key");
  const dingAppSecretOptions = useDictOptions("dingtalk_app_secret");
  const dingChatIDOptions = useDictOptions("dingtalk_chat_id");
  const dingSignSecretOptions = useDictOptions("dingtalk_sign_secret");
  const urlFillOptions = channelType === "wechat_work" || channelType === "wechat"
    ? wecomWebhookURLDictOptions
    : channelType === "dingding"
      ? dingtalkWebhookURLDictOptions
      : webhookURLDictOptions;

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

  async function loadProjects() {
    const res = await getProjects({ page: 1, page_size: 500 });
    setProjects(res.list ?? []);
  }

  useEffect(() => {
    void load();
    void loadProjects();
  }, []);

  useEffect(() => {
    if (!templateOpen) return;
    const timer = window.setTimeout(async () => {
      setPreviewLoading(true);
      setPreviewError("");
      try {
        const res = await previewAlertChannelTemplate({
          template_firing: String(templateFiring || ""),
          template_resolved: String(templateResolved || ""),
          status: previewStatus,
          project_id: previewProjectID,
          raw_payload_json: String(previewRawPayloadJSON || ""),
        });
        setPreviewResult(res);
      } catch (err: any) {
        setPreviewResult(null);
        setPreviewError(String(err?.message || "模板预览失败"));
      } finally {
        setPreviewLoading(false);
      }
    }, 300);
    return () => window.clearTimeout(timer);
  }, [templateOpen, templateFiring, templateResolved, previewStatus, previewProjectID, previewRawPayloadJSON]);

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
    delete (extra as any).messageTemplateFiring;
    delete (extra as any).messageTemplateResolved;
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

  function insertTemplateToken(token: string) {
    const fieldName = editingTemplateTarget === "resolved" ? "template_resolved" : "template_firing";
    const currentValue = String(templateForm.getFieldValue(fieldName) || "");
    const inputRef = editingTemplateTarget === "resolved" ? resolvedTemplateRef.current : firingTemplateRef.current;
    const textarea = inputRef?.resizableTextArea?.textArea as HTMLTextAreaElement | undefined;
    if (textarea && Number.isFinite(textarea.selectionStart) && Number.isFinite(textarea.selectionEnd)) {
      const start = textarea.selectionStart;
      const end = textarea.selectionEnd;
      const next = `${currentValue.slice(0, start)}${token}${currentValue.slice(end)}`;
      templateForm.setFieldValue(fieldName, next);
      window.setTimeout(() => {
        textarea.focus();
        const pos = start + token.length;
        textarea.setSelectionRange(pos, pos);
      }, 0);
      return;
    }
    const sep = currentValue.trim() ? "\n" : "";
    templateForm.setFieldValue(fieldName, `${currentValue}${sep}${token}`);
  }

  function openTemplateConfig(channelID?: number) {
    const targetID = channelID ?? templateChannelID ?? list[0]?.id;
    if (!targetID) {
      message.warning("请先创建告警通道");
      return;
    }
    const row = list.find((it) => it.id === targetID);
    if (!row) {
      message.warning("未找到目标通道");
      return;
    }
    const settings = parseChannelSettings(row.headers_json || "");
    setTemplateChannelID(targetID);
    templateForm.setFieldsValue({
      template_firing: typeof settings.messageTemplateFiring === "string" ? settings.messageTemplateFiring : DEFAULT_FIRING_TEMPLATE,
      template_resolved: typeof settings.messageTemplateResolved === "string" ? settings.messageTemplateResolved : DEFAULT_RESOLVED_TEMPLATE,
      preview_status: "firing",
      preview_project_id: undefined,
      preview_raw_payload_json: "",
    });
    setFiringPreset("default");
    setResolvedPreset("default");
    setTemplateOpen(true);
  }

  async function submitTemplate() {
    if (!templateChannelID) {
      message.warning("请先选择要保存模板的告警通道");
      return;
    }
    const values = await templateForm.validateFields();
    const row = list.find((it) => it.id === templateChannelID);
    if (!row) {
      message.error("未找到目标通道");
      return;
    }
    const settings = parseChannelSettings(row.headers_json || "");
    const firing = String(values.template_firing || "").trim();
    const resolved = String(values.template_resolved || "").trim();
    if (firing) settings.messageTemplateFiring = firing;
    else delete (settings as any).messageTemplateFiring;
    if (resolved) settings.messageTemplateResolved = resolved;
    else delete (settings as any).messageTemplateResolved;
    setTemplateSaving(true);
    try {
      await updateAlertChannel(templateChannelID, {
        name: row.name,
        type: row.type,
        url: row.url,
        secret: row.secret,
        headers_json: stringifySettings(settings),
        enabled: row.enabled,
        timeout_ms: row.timeout_ms,
      });
      message.success("告警模板已保存");
      setTemplateOpen(false);
      await load();
    } finally {
      setTemplateSaving(false);
    }
  }

  function applyTemplatePreset(target: TemplatePreviewStatus, presetMode: string) {
    const tpl = presetTemplateByMode(presetMode);
    if (target === "firing") {
      setFiringPreset(presetMode);
      templateForm.setFieldValue("template_firing", tpl.firing);
      return;
    }
    setResolvedPreset(presetMode);
    templateForm.setFieldValue("template_resolved", tpl.resolved);
  }

  function openTest(row: AlertChannelItem) {
    setTestRow(row);
    testForm.setFieldsValue({
      status: "firing",
      severity: "info",
      title: "",
      content: "",
    });
    setTestOpen(true);
  }

  async function submitTest() {
    if (!testRow) return;
    const values = await testForm.validateFields();
    setTestSending(true);
    try {
      await testAlertChannel(testRow.id, {
        status: values.status,
        severity: values.severity,
        title: values.title,
        content: values.content,
      });
      message.success("测试发送成功");
      setTestOpen(false);
    } finally {
      setTestSending(false);
    }
  }

  const projectOptions = projects.map((p) => ({ label: `${p.name} (${p.code})`, value: p.id }));
  const availableFields = previewResult?.combined_fields ?? [];
  const fixedFields = previewResult?.available_fields ?? [];
  const rawPayloadFields = previewResult?.raw_payload_fields ?? [];
  const suggestedLabelKeys = previewResult?.suggested_label_keys ?? [];

  return (
    <Card className="table-card" title="Webhook 告警通道">
      <div style={{ display: "flex", justifyContent: "space-between", marginBottom: 12 }}>
        <Space>
          <Select
            style={{ width: 260 }}
            allowClear
            placeholder="选择一个通道配置告警模板"
            value={templateChannelID}
            onChange={(v) => setTemplateChannelID(v)}
            options={list.map((it) => ({ label: `${it.name} (${it.type})`, value: it.id }))}
          />
          <Button onClick={() => openTemplateConfig()}>告警模板</Button>
        </Space>
        <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>新建通道</Button>
      </div>
      <Table
        rowKey="id"
        loading={loading}
        dataSource={list}
        pagination={{ pageSize: 10, showSizeChanger: true, pageSizeOptions: [10, 20, 50, 100], showQuickJumper: true }}
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
            width: 220,
            render: (_: unknown, row: AlertChannelItem) => (
              <Space size={4} wrap>
                <Button type="link" icon={<SendOutlined />} onClick={() => openTest(row)}>
                  测试
                </Button>
                <Button type="link" onClick={() => openTemplateConfig(row.id)}>
                  模板
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
        title={current ? "编辑通道" : "新建告警通道"}
        open={open}
        onCancel={() => setOpen(false)}
        onOk={() => void submit()}
        confirmLoading={saving}
        destroyOnClose
        width={760}
      >
        <Form form={form} layout="vertical" autoComplete="off">
          <Form.Item name="name" label="通道名称" rules={[{ required: true, message: "请输入通道名称" }]}>
            <Input />
          </Form.Item>
          <Form.Item name="type" label="通道类型" rules={[{ required: true }]}>
            <Select options={channelTypeOptions} />
          </Form.Item>
          <Form.Item name="url" label="Webhook URL（email、钉钉app_chat 可留空）" rules={[{ type: "url", message: "URL 格式不正确" }]}>
            <Input />
          </Form.Item>
          <Form.Item label="从字典填充 URL">
            <DictFillSelect
              form={form}
              fieldName="url"
              options={urlFillOptions}
              placeholder={
                channelType === "dingding"
                  ? "可选：选择钉钉 Webhook URL"
                  : channelType === "wechat_work" || channelType === "wechat"
                    ? "可选：选择企业微信 Webhook URL"
                    : "可选：选择后自动填入 Webhook URL"
              }
            />
          </Form.Item>
          <Form.Item name="secret" label="签名密钥（可选）">
            <Input.Password allowClear autoComplete="new-password" />
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
              <Select options={wecomModeOptions} />
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
              <Form.Item label="从字典填充 corpID" hidden={wecomMode !== "app"}>
                <DictFillSelect form={form} fieldName="corp_id" options={wecomCorpIDOptions} placeholder="可选：字典中的 corpID" />
              </Form.Item>
              <Form.Item
                name="corp_secret"
                label="企业微信 corpSecret（app 模式：手机号查 userid 用）"
                hidden={wecomMode !== "app"}
              >
                <Input.Password allowClear autoComplete="new-password" />
              </Form.Item>
              <Form.Item label="从字典填充 corpSecret" hidden={wecomMode !== "app"}>
                <DictFillSelect form={form} fieldName="corp_secret" options={wecomCorpSecretOptions} placeholder="可选：字典中的企业密钥" />
              </Form.Item>
              <Form.Item name="agent_id" label="企业微信 agentId（app 模式必填）" hidden={wecomMode !== "app"}>
                <Input placeholder="1000002" />
              </Form.Item>
              <Form.Item label="从字典填充 agentId" hidden={wecomMode !== "app"}>
                <DictFillSelect form={form} fieldName="agent_id" options={wecomAgentIDOptions} placeholder="可选：字典中的 agentId" />
              </Form.Item>
            </>
          ) : null}
          {channelType === "dingding" ? (
            <Form.Item name="ding_mode" label="钉钉模式">
              <Select options={dingModeOptions} />
            </Form.Item>
          ) : null}
          {channelType === "dingding" ? (
            <>
              <Form.Item name="ding_app_key" label="钉钉 appKey（app_chat 模式）" hidden={dingMode !== "app_chat"}>
                <Input />
              </Form.Item>
              <Form.Item label="从字典填充 appKey" hidden={dingMode !== "app_chat"}>
                <DictFillSelect form={form} fieldName="ding_app_key" options={dingAppKeyOptions} placeholder="可选：字典中的应用账号" />
              </Form.Item>
              <Form.Item name="ding_app_secret" label="钉钉 appSecret（app_chat 模式）" hidden={dingMode !== "app_chat"}>
                <Input.Password allowClear autoComplete="new-password" />
              </Form.Item>
              <Form.Item label="从字典填充 appSecret" hidden={dingMode !== "app_chat"}>
                <DictFillSelect form={form} fieldName="ding_app_secret" options={dingAppSecretOptions} placeholder="可选：字典中的应用密钥" />
              </Form.Item>
              <Form.Item name="ding_chat_id" label="钉钉 chatId（app_chat 模式必填 / robot 可不填）">
                <Input placeholder="chatxxxxx" />
              </Form.Item>
              <Form.Item label="从字典填充 chatId" hidden={dingMode !== "app_chat"}>
                <DictFillSelect form={form} fieldName="ding_chat_id" options={dingChatIDOptions} placeholder="可选：字典中的 chatId" />
              </Form.Item>
              <Form.Item name="ding_sign_secret" label="钉钉 signSecret（robot 加签可选）" hidden={dingMode !== "robot"}>
                <Input.Password allowClear autoComplete="new-password" />
              </Form.Item>
              <Form.Item
                label="从字典填充 signSecret"
                hidden={dingMode !== "robot"}
                extra="仅展示数据字典中「启用」且类型为 dingtalk_sign_secret 的条目（与 app_chat 的 dingtalk_app_secret 不是同一配置）。若暂无数据，请到数据字典启用或新增该类型。"
              >
                <DictFillSelect form={form} fieldName="ding_sign_secret" options={dingSignSecretOptions} placeholder="可选：字典中的 signSecret" />
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
            rules={[
              {
                validator: async (_, value) => {
                  const s = String(value || "").trim();
                  if (!s) return;
                  try {
                    const obj = JSON.parse(s);
                    if (!obj || typeof obj !== "object" || Array.isArray(obj)) {
                      throw new Error("JSON 必须是对象");
                    }
                  } catch {
                    throw new Error("JSON 格式不正确");
                  }
                },
              },
            ]}
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
      <Modal
        title="告警模板配置"
        open={templateOpen}
        onCancel={() => setTemplateOpen(false)}
        onOk={() => void submitTemplate()}
        confirmLoading={templateSaving}
        destroyOnClose
        width={860}
      >
        <Form form={templateForm} layout="vertical">
          <Form.Item label="当前通道">
            <Select
              value={templateChannelID}
              onChange={(v) => {
                setTemplateChannelID(v);
                openTemplateConfig(v);
              }}
              options={list.map((it) => ({ label: `${it.name} (${it.type})`, value: it.id }))}
            />
          </Form.Item>
          <Form.Item
            name="template_firing"
            label="告警触发模板（可视化配置）"
            extra={'留空时使用系统默认模板；支持变量：{{.Title}} {{.Severity}} {{.StatusText}} {{.ProjectName}} {{.Cluster}} {{.Summary}} {{.Description}} {{.OccurredAt}} {{.StartsAt}} {{.EndsAt}} {{.Current}} {{.Fingerprint}} {{.GeneratorURL}} {{.LabelsText}}，也可按标签取值：{{index .Labels "alertname"}}'}
          >
            <Input.TextArea ref={firingTemplateRef} rows={6} placeholder="支持 Go Template 语法，例如 {{.Title}}" />
          </Form.Item>
          <Form.Item label="触发模板预设（下拉可选）" extra="选择后会自动生成模板示例，你可以继续微调。">
            <Select
              value={firingPreset}
              options={CHANNEL_PRESET_OPTIONS}
              onChange={(v) => applyTemplatePreset("firing", v)}
            />
          </Form.Item>
          <Form.Item
            name="template_resolved"
            label="告警恢复模板（可视化配置）"
            extra="留空时使用系统默认恢复模板；语法与触发模板一致。"
          >
            <Input.TextArea ref={resolvedTemplateRef} rows={6} placeholder="支持 Go Template 语法，例如 {{.StartsAt}} ~ {{.EndsAt}}" />
          </Form.Item>
          <Form.Item label="恢复模板预设（下拉可选）" extra="选择后会自动生成恢复模板示例，你可以继续微调。">
            <Select
              value={resolvedPreset}
              options={CHANNEL_PRESET_OPTIONS}
              onChange={(v) => applyTemplatePreset("resolved", v)}
            />
          </Form.Item>
          <Form.Item label="当前插入目标模板">
            <Select
              value={editingTemplateTarget}
              onChange={(v: TemplatePreviewStatus) => setEditingTemplateTarget(v)}
              options={[
                { label: "触发模板", value: "firing" },
                { label: "恢复模板", value: "resolved" },
              ]}
            />
          </Form.Item>
          <Form.Item name="preview_status" label="模板预览类型" initialValue="firing">
            <Select
              options={[
                { label: "触发模板预览", value: "firing" },
                { label: "恢复模板预览", value: "resolved" },
              ]}
            />
          </Form.Item>
          <Form.Item name="preview_project_id" label="预览项目上下文（可选）" extra="选择后将用该项目填充 ProjectName，并从近期该项目告警提取标签字段建议。">
            <Select allowClear options={projectOptions} placeholder="不选则使用默认示例项目" />
          </Form.Item>
          <Form.Item
            name="preview_raw_payload_json"
            label="预览原始 JSON（可选，实时合并）"
            extra="填写原始告警 JSON 对象后，会与系统示例 payload 合并参与真实后端渲染；同名字段以你填写的 JSON 为准。"
            rules={[
              {
                validator: async (_, value) => {
                  const s = String(value || "").trim();
                  if (!s) return;
                  try {
                    const obj = JSON.parse(s);
                    if (!obj || typeof obj !== "object" || Array.isArray(obj)) {
                      throw new Error("预览原始 JSON 必须是对象");
                    }
                  } catch {
                    throw new Error("预览原始 JSON 格式不正确");
                  }
                },
              },
            ]}
          >
            <Input.TextArea rows={6} placeholder='例如：{"labels":{"namespace":"prod"},"current":"9"}' />
          </Form.Item>
          <Form.Item label="模板预览（示例数据）">
            <Input.TextArea rows={8} value={previewResult?.rendered || ""} readOnly />
          </Form.Item>
          <Form.Item label="预览状态">
            <Input value={previewLoading ? "渲染中..." : previewError || "渲染成功"} readOnly status={previewError ? "error" : undefined} />
          </Form.Item>
          <Form.Item label="渲染上下文（sample_payload）" extra="展示后端本次预览实际使用的 payload，便于核对模板字段来源。">
            <Collapse
              size="small"
              items={[
                {
                  key: "sample_payload",
                  label: "展开查看 sample_payload JSON",
                  children: (
                    <Input.TextArea
                      rows={12}
                      readOnly
                      value={JSON.stringify(previewResult?.sample_payload || {}, null, 2)}
                    />
                  ),
                },
              ]}
            />
          </Form.Item>
          <Form.Item label="可用参考字段（组合）" extra="来自：后端固定模板字段 + 预览原始 JSON 顶层字段；可点击插入 {{.字段名}}">
            <Space wrap>
              {availableFields.map((v) => (
                <Tag
                  key={v}
                  color="blue"
                  style={{ cursor: "pointer", userSelect: "none" }}
                  onClick={() => insertTemplateToken(`{{.${v}}}`)}
                >
                  {v}
                </Tag>
              ))}
            </Space>
          </Form.Item>
          <Form.Item label="固定模板字段" extra="后端固定返回，任何预览场景都可用">
            <Space wrap>
              {fixedFields.map((v) => (
                <Tag key={v}>{v}</Tag>
              ))}
            </Space>
          </Form.Item>
          <Form.Item label="原始 JSON 字段" extra="来自你填写的预览原始 JSON 顶层 key">
            <Space wrap>
              {rawPayloadFields.length === 0 ? <Tag>（暂无）</Tag> : rawPayloadFields.map((v) => <Tag key={v}>{v}</Tag>)}
            </Space>
          </Form.Item>
          <Form.Item label="可用参考标签（近期告警提取）" extra='可直接用于模板：{{index .Labels "标签名"}}'>
            <Space wrap>
              {suggestedLabelKeys.map((v) => (
                <Tag
                  key={v}
                  color="purple"
                  style={{ cursor: "pointer", userSelect: "none" }}
                  onClick={() => insertTemplateToken(`{{index .Labels "${v}"}}`)}
                >
                  {v}
                </Tag>
              ))}
            </Space>
          </Form.Item>
        </Form>
      </Modal>
      <Modal
        title={testRow ? `测试发送 #${testRow.id} ${testRow.name}` : "测试发送"}
        open={testOpen}
        onCancel={() => setTestOpen(false)}
        onOk={() => void submitTest()}
        confirmLoading={testSending}
        destroyOnClose
      >
        <Form form={testForm} layout="vertical">
          <Form.Item name="status" label="测试状态" rules={[{ required: true, message: "请选择测试状态" }]}>
            <Select
              options={[
                { label: "触发（firing）", value: "firing" },
                { label: "恢复（resolved）", value: "resolved" },
              ]}
            />
          </Form.Item>
          <Form.Item name="severity" label="级别（可选）">
            <Input placeholder="默认 info" />
          </Form.Item>
          <Form.Item name="title" label="标题（可选）">
            <Input placeholder="留空则按状态自动生成测试标题" />
          </Form.Item>
          <Form.Item name="content" label="内容（可选）">
            <Input.TextArea rows={3} placeholder="留空则自动生成测试内容" />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}

