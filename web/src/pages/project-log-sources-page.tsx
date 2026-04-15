import { DeleteOutlined, EditOutlined, PlusOutlined, ReloadOutlined, DeploymentUnitOutlined, CopyOutlined } from "@ant-design/icons";
import { AutoComplete, Button, Card, Col, Form, Input, Modal, Popconfirm, Row, Select, Space, Table, Tag, message } from "antd";
import { useEffect, useMemo, useState } from "react";
import {
  getProjects,
  getProjectServers,
  getProjectServices,
  getProjectLogSources,
  upsertProjectLogSource,
  deleteProjectLogSource,
  bootstrapProjectAgent,
  rotateProjectAgentToken,
  type ProjectItem,
  type ServerItem,
  type ServiceItem,
  type LogSourceItem,
} from "../services/projects";

export function ProjectLogSourcesPage() {
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [servers, setServers] = useState<ServerItem[]>([]);
  const [services, setServices] = useState<ServiceItem[]>([]);
  const [sources, setSources] = useState<LogSourceItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [serverId, setServerId] = useState<number>();
  const [serviceId, setServiceId] = useState<number>();
  const [loading, setLoading] = useState(false);
  const [editorOpen, setEditorOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [bootstrapOpen, setBootstrapOpen] = useState(false);
  const [bootstrapping, setBootstrapping] = useState(false);
  const [rotateConfirmOpen, setRotateConfirmOpen] = useState(false);
  const [rotateConfirmInput, setRotateConfirmInput] = useState("");
  const [bootstrapResult, setBootstrapResult] = useState<{ run_command: string; systemd_service: string; token: string } | null>(null);
  const [fileOptions, setFileOptions] = useState<Array<{ value: string; label: string }>>([]);
  const [current, setCurrent] = useState<LogSourceItem | null>(null);
  const [bootstrapServerId, setBootstrapServerId] = useState<number | null>(null);
  const [form] = Form.useForm<{ id?: number; service_id: number; log_type: string; path: string; log_dir?: string; include_regex?: string; exclude_regex?: string; status: number }>();
  const [bootstrapForm] = Form.useForm<{ platform_url: string; agent_name?: string; agent_version?: string }>();

  const projectOptions = useMemo(() => projects.map((p) => ({ value: p.id, label: `${p.name} (${p.code})` })), [projects]);
  const serverOptions = useMemo(() => servers.map((s) => ({ value: s.id, label: `${s.name} ${s.host}:${s.port} (${s.os_type}/${s.os_arch || "-"})` })), [servers]);
  const serviceOptions = useMemo(() => services.map((s) => ({ value: s.id, label: s.name })), [services]);

  useEffect(() => {
    void (async () => {
      const p = await getProjects({ page: 1, page_size: 1000 });
      setProjects(p.list);
      if (p.list[0]) setProjectId(p.list[0].id);
    })();
  }, []);

  useEffect(() => {
    if (!projectId) return;
    void (async () => {
      const sv = await getProjectServers(projectId, { page: 1, page_size: 1000 });
      setServers(sv.list);
      setServerId(undefined);
      setServiceId(undefined);
      setServices([]);
    })();
  }, [projectId]);

  useEffect(() => {
    if (!projectId) return;
    void (async () => {
      const list = await getProjectServices(projectId, { page: 1, page_size: 1000, server_id: serverId });
      setServices(list.list);
      setServiceId(undefined);
    })();
  }, [projectId, serverId]);

  useEffect(() => {
    if (!projectId) return;
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId, serviceId]);

  function splitFilePathForEditor(rawPath: string): { logDir?: string; filePart: string } {
    const p = String(rawPath || "").trim();
    if (!p) return { filePart: "" };
    const normalized = p.replace(/\\/g, "/");
    const idx = normalized.lastIndexOf("/");
    if (idx <= 0) return { filePart: p };
    return {
      logDir: normalized.slice(0, idx),
      filePart: normalized.slice(idx + 1),
    };
  }

  async function load() {
    if (!projectId) return;
    setLoading(true);
    try {
      const res = await getProjectLogSources(projectId, { page: 1, page_size: 1000, service_id: serviceId });
      setSources(res.list);
    } finally {
      setLoading(false);
    }
  }

  function openCreate() {
    if (!serviceId) {
      message.warning("请先选择服务");
      return;
    }
    setCurrent(null);
    setFileOptions([]);
    form.resetFields();
    form.setFieldsValue({ service_id: serviceId, log_type: "file", status: 1 });
    setEditorOpen(true);
  }

  function openEdit(record: LogSourceItem) {
    setCurrent(record);
    setFileOptions(record.path ? [{ value: record.path, label: record.path }] : []);
    const split = record.log_type === "file" ? splitFilePathForEditor(record.path) : { filePart: record.path };
    form.setFieldsValue({
      id: record.id,
      service_id: record.service_id,
      log_type: record.log_type,
      path: split.filePart || record.path,
      log_dir: split.logDir,
      status: record.status,
      include_regex: record.include_regex ?? undefined,
      exclude_regex: record.exclude_regex ?? undefined,
    });
    setEditorOpen(true);
  }

  async function onSubmit() {
    if (!projectId) return;
    const v = await form.validateFields();
    setSubmitting(true);
    try {
      let finalPath = String(v.path || "").trim();
      if (v.log_type === "file") {
        const logDir = String(v.log_dir || "").trim();
        const isAbsWin = /^[a-zA-Z]:[\\/]/.test(finalPath);
        const isAbsUnix = finalPath.startsWith("/");
        if (logDir && !isAbsWin && !isAbsUnix) {
          const d = logDir.replace(/[\\/]+$/, "");
          const p = finalPath.replace(/^[\\/]+/, "");
          finalPath = `${d}/${p}`;
        }
      }
      await upsertProjectLogSource(projectId, {
        id: v.id,
        service_id: v.service_id,
        log_type: v.log_type,
        path: finalPath,
        include_regex: v.include_regex,
        exclude_regex: v.exclude_regex,
        status: v.status,
      });
      message.success(current ? "已更新日志源" : "已创建日志源");
      setEditorOpen(false);
      void load();
    } finally {
      setSubmitting(false);
    }
  }

  function openBootstrapForServer() {
    if (!projectId) return;
    if (!serverId) {
      message.warning("请先选择服务器");
      return;
    }
    setBootstrapServerId(serverId);
    setBootstrapResult(null);
    bootstrapForm.setFieldsValue({
      platform_url: window.location.origin.replace(/\/$/, ""),
      agent_name: `log-agent-s${serverId}`,
      agent_version: "v0.1.0",
    });
    setBootstrapOpen(true);
  }

  async function doBootstrap() {
    if (!projectId || !bootstrapServerId) return;
    const values = await bootstrapForm.validateFields();
    setBootstrapping(true);
    try {
      const res = await bootstrapProjectAgent(projectId, {
        server_id: bootstrapServerId,
        platform_url: values.platform_url,
        agent_name: values.agent_name,
        agent_version: values.agent_version,
      });
      setBootstrapResult({
        run_command: res.run_command,
        systemd_service: res.systemd_service,
        token: res.token,
      });
      message.success("已生成部署命令");
    } finally {
      setBootstrapping(false);
    }
  }

  async function doRotateToken() {
    if (!projectId || !bootstrapServerId) return;
    const values = await bootstrapForm.validateFields();
    setBootstrapping(true);
    try {
      const res = await rotateProjectAgentToken(projectId, {
        server_id: bootstrapServerId,
        platform_url: values.platform_url,
        agent_name: values.agent_name,
        agent_version: values.agent_version,
      });
      setBootstrapResult({
        run_command: res.run_command,
        systemd_service: res.systemd_service,
        token: res.token,
      });
      setRotateConfirmOpen(false);
      message.success("Token已轮换，旧Token立即失效");
    } finally {
      setBootstrapping(false);
    }
  }

  function openRotateConfirm() {
    setRotateConfirmInput("");
    setRotateConfirmOpen(true);
  }

  return (
    <Card
      title="日志源配置"
      extra={
        <Space>
          <Select style={{ width: 240 }} value={projectId} onChange={setProjectId} options={projectOptions} placeholder="项目" />
          <Select style={{ width: 300 }} value={serverId} onChange={setServerId} options={serverOptions} placeholder="服务器" allowClear />
          <Select style={{ width: 220 }} value={serviceId} onChange={setServiceId} options={serviceOptions} placeholder="服务" allowClear />
          <Button icon={<ReloadOutlined />} onClick={() => void load()} loading={loading}>刷新</Button>
          <Button icon={<DeploymentUnitOutlined />} onClick={openBootstrapForServer} disabled={!serverId}>
            部署本机Agent
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>新增日志源</Button>
        </Space>
      }
    >
      <Table
        rowKey="id"
        dataSource={sources}
        loading={loading}
        pagination={false}
        columns={[
          { title: "类型", dataIndex: "log_type", width: 100 },
          { title: "路径/Unit", dataIndex: "path" },
          { title: "include", dataIndex: "include_regex", render: (v?: string | null) => v || "-" },
          { title: "exclude", dataIndex: "exclude_regex", render: (v?: string | null) => v || "-" },
          { title: "状态", dataIndex: "status", width: 90, render: (v: number) => (v === 1 ? <Tag color="green">启用</Tag> : <Tag color="red">停用</Tag>) },
          {
            title: "操作",
            width: 200,
            render: (_: unknown, record: LogSourceItem) => (
              <Space>
                <Button icon={<EditOutlined />} onClick={() => openEdit(record)}>编辑</Button>
                <Popconfirm title="确认删除日志源？" onConfirm={() => projectId && deleteProjectLogSource(projectId, record.id).then(() => { message.success("已删除"); void load(); })}>
                  <Button danger icon={<DeleteOutlined />}>删除</Button>
                </Popconfirm>
              </Space>
            ),
          },
        ]}
      />

      <Modal open={editorOpen} title={current ? "编辑日志源" : "新增日志源"} onCancel={() => setEditorOpen(false)} onOk={() => void onSubmit()} confirmLoading={submitting} width={920}>
        <Form form={form} layout="vertical">
          <Form.Item name="id" hidden>
            <Input />
          </Form.Item>
          <Row gutter={12}>
            <Col span={8}><Form.Item label="服务" name="service_id" rules={[{ required: true }]}><Select options={serviceOptions} /></Form.Item></Col>
            <Col span={4}><Form.Item label="类型" name="log_type" rules={[{ required: true }]}><Select options={[{ value: "file", label: "file" }, { value: "journal", label: "journal" }]} /></Form.Item></Col>
            <Col span={4}><Form.Item label="状态" name="status" rules={[{ required: true }]}><Select options={[{ value: 1, label: "启用" }, { value: 0, label: "停用" }]} /></Form.Item></Col>
            <Col span={8}><Form.Item label="日志目录（file 类型）" name="log_dir"><Input placeholder="/var/log/app" /></Form.Item></Col>
          </Row>
          <Row gutter={12}>
            <Col span={24}>
              <Space>
                <span style={{ color: "#999" }}>agent-only 模式：请手动填写日志文件路径或 systemd unit</span>
              </Space>
            </Col>
          </Row>
          <Row gutter={12} style={{ marginTop: 12 }}>
            <Col span={24}>
              <Form.Item label="路径/Unit" name="path" rules={[{ required: true }]}>
                <AutoComplete
                  allowClear
                  options={fileOptions}
                  placeholder="可从扫描结果选择，也可手动输入"
                  filterOption={(input, option) => (String(option?.label ?? "")).toLowerCase().includes(input.toLowerCase())}
                />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={12}>
            <Col span={12}><Form.Item label="include regex" name="include_regex"><Input /></Form.Item></Col>
            <Col span={12}><Form.Item label="exclude regex" name="exclude_regex"><Input /></Form.Item></Col>
          </Row>
        </Form>
      </Modal>

      <Modal
        open={bootstrapOpen}
        title="部署本机 Agent（覆盖该服务器所有日志源）"
        onCancel={() => setBootstrapOpen(false)}
        onOk={() => void doBootstrap()}
        confirmLoading={bootstrapping}
        width={980}
      >
        <Form form={bootstrapForm} layout="vertical">
          <Row gutter={12}>
            <Col span={12}>
              <Form.Item label="平台地址" name="platform_url" rules={[{ required: true, message: "请输入平台地址" }]}>
                <Input placeholder="例如：http://10.10.10.10:8080" />
              </Form.Item>
            </Col>
            <Col span={6}>
              <Form.Item label="Agent 名称" name="agent_name">
                <Input placeholder="log-agent" />
              </Form.Item>
            </Col>
            <Col span={6}>
              <Form.Item label="Agent 版本" name="agent_version">
                <Input placeholder="v0.1.0" />
              </Form.Item>
            </Col>
          </Row>
        </Form>
        {!bootstrapResult ? null : (
          <Space direction="vertical" style={{ width: "100%" }} size={10}>
            <div style={{ color: "#999" }}>Token（请妥善保管）</div>
            <Input.TextArea rows={2} value={bootstrapResult.token} readOnly />
            <Button
              icon={<CopyOutlined />}
              onClick={() => navigator.clipboard.writeText(bootstrapResult.token).then(() => message.success("Token已复制"))}
              style={{ width: 140 }}
            >
              复制Token
            </Button>
            <Button onClick={openRotateConfirm} loading={bootstrapping} danger style={{ width: 160 }}>
              轮换Token
            </Button>
            <div style={{ color: "#999" }}>一次性运行命令（Linux）</div>
            <Input.TextArea rows={4} value={bootstrapResult.run_command} readOnly />
            <Button
              icon={<CopyOutlined />}
              onClick={() => navigator.clipboard.writeText(bootstrapResult.run_command).then(() => message.success("命令已复制"))}
              style={{ width: 140 }}
            >
              复制命令
            </Button>
            <div style={{ color: "#999" }}>systemd 服务文件（推荐）</div>
            <Input.TextArea rows={12} value={bootstrapResult.systemd_service} readOnly />
            <Button
              icon={<CopyOutlined />}
              onClick={() => navigator.clipboard.writeText(bootstrapResult.systemd_service).then(() => message.success("systemd配置已复制"))}
              style={{ width: 180 }}
            >
              复制systemd配置
            </Button>
          </Space>
        )}
      </Modal>

      <Modal
        open={rotateConfirmOpen}
        title="确认轮换 Token"
        onCancel={() => setRotateConfirmOpen(false)}
        onOk={() => void doRotateToken()}
        okText="确认轮换"
        okButtonProps={{
          danger: true,
          disabled: (() => {
            const target = (bootstrapForm.getFieldValue("agent_name") || "").trim();
            return !target || rotateConfirmInput.trim() !== target;
          })(),
          loading: bootstrapping,
        }}
      >
        <Space direction="vertical" style={{ width: "100%" }}>
          <div style={{ color: "#999" }}>
            此操作会让旧 Token 立即失效。请输入当前 Agent 名称进行二次确认：
            <Tag color="orange" style={{ marginLeft: 8 }}>{(bootstrapForm.getFieldValue("agent_name") || "-") as string}</Tag>
          </div>
          <Input
            placeholder="请输入上面的 Agent 名称"
            value={rotateConfirmInput}
            onChange={(e) => setRotateConfirmInput(e.target.value)}
          />
        </Space>
      </Modal>
    </Card>
  );
}

