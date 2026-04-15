import { PauseOutlined, PlayCircleOutlined, ReloadOutlined, DownloadOutlined } from "@ant-design/icons";
import { Button, Card, Col, Form, Input, InputNumber, Row, Select, Space, Tag, message } from "antd";
import { useEffect, useMemo, useRef, useState } from "react";
import { Terminal } from "xterm";
import { FitAddon } from "xterm-addon-fit";
import "xterm/css/xterm.css";
import { getToken } from "../services/storage";
import {
  getProjects,
  getProjectServers,
  getProjectServices,
  getProjectLogSources,
  getProjectAgentDiscovery,
  getProjectAgentStatus,
  exportProjectLogs,
  type ProjectItem, type ServerItem, type ServiceItem, type LogSourceItem, type AgentDiscoveryItem,
} from "../services/projects";
import { formatDateTime } from "../utils/format";

type StreamForm = {
  project_id?: number;
  server_id?: number;
  service_id?: number;
  log_source_id?: number;
  file_path?: string;
  tail_lines?: number;
  include?: string;
  exclude?: string;
  highlight?: string;
};

function buildSseUrl(projectId: number, params: Record<string, string>) {
  const qs = new URLSearchParams(params);
  return `/api/v1/projects/${projectId}/logs/stream?${qs.toString()}`;
}

function globToRegExp(pattern: string): RegExp | null {
  const src = pattern.trim();
  if (!src) return null;
  let re = "^";
  let i = 0;
  while (i < src.length) {
    const ch = src[i];
    if (ch === "*") {
      if (src[i + 1] === "*") {
        re += ".*";
        i += 2;
      } else {
        re += "[^/]*";
        i += 1;
      }
      continue;
    }
    if (ch === "?") {
      re += ".";
      i += 1;
      continue;
    }
    if (ch === ".") {
      re += "\\.";
      i += 1;
      continue;
    }
    if ("+^$(){}|[]\\".includes(ch)) {
      re += `\\${ch}`;
      i += 1;
      continue;
    }
    re += ch;
    i += 1;
  }
  re += "$";
  try {
    return new RegExp(re);
  } catch {
    return null;
  }
}

function pathMatchesSource(filePath: string, sourcePath: string): boolean {
  const file = filePath.trim();
  const src = sourcePath.trim();
  if (!file || !src) return false;
  if (!src.includes("*") && !src.includes("?")) return file === src;
  const re = globToRegExp(src);
  if (!re) return false;
  return re.test(file);
}

export function ProjectLogsPage() {
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [servers, setServers] = useState<ServerItem[]>([]);
  const [services, setServices] = useState<ServiceItem[]>([]);
  const [sources, setSources] = useState<LogSourceItem[]>([]);
  const [matchedFiles, setMatchedFiles] = useState<AgentDiscoveryItem[]>([]);

  const [streaming, setStreaming] = useState(false);
  const [streamModeHint, setStreamModeHint] = useState("未开始");
  const [agentStatus, setAgentStatus] = useState<{
    stateText: string;
    online: boolean;
    recentPublishing: boolean;
    lastSeenText: string;
    checked: boolean;
  }>({
    stateText: "未检测",
    online: false,
    recentPublishing: false,
    lastSeenText: "-",
    checked: false,
  });
  const abortRef = useRef<AbortController | null>(null);

  const termHostRef = useRef<HTMLDivElement | null>(null);
  const termRef = useRef<Terminal | null>(null);
  const fitRef = useRef<FitAddon | null>(null);

  const [form] = Form.useForm<StreamForm>();
  const watchProjectId = Form.useWatch("project_id", form);
  const watchServerId = Form.useWatch("server_id", form);
  const watchLogSourceId = Form.useWatch("log_source_id", form);

  const projectOptions = useMemo(() => projects.map((p) => ({ value: p.id, label: `${p.name} (${p.code})` })), [projects]);
  const serverOptions = useMemo(
    () => servers.map((s) => ({ value: s.id, label: `${s.name} ${s.host}:${s.port} (${s.os_type || "-"} ${s.os_arch || "-"})` })),
    [servers],
  );
  const serviceOptions = useMemo(() => services.map((s) => ({ value: s.id, label: s.name })), [services]);
  const sourceOptions = useMemo(() => sources.map((s) => ({ value: s.id, label: `${s.log_type}:${s.path}` })), [sources]);
  const selectedSource = useMemo(
    () => sources.find((s) => s.id === watchLogSourceId),
    [sources, watchLogSourceId],
  );
  const fileOptions = useMemo(
    () => [
      { value: "", label: "全部匹配文件" },
      ...matchedFiles.map((it) => ({ value: it.value, label: it.value })),
    ],
    [matchedFiles],
  );

  useEffect(() => {
    termRef.current = new Terminal({ convertEol: true, fontSize: 12, scrollback: 5000 });
    fitRef.current = new FitAddon();
    termRef.current.loadAddon(fitRef.current);
    if (termHostRef.current) {
      termRef.current.open(termHostRef.current);
      fitRef.current.fit();
    }
    const onResize = () => fitRef.current?.fit();
    window.addEventListener("resize", onResize);
    return () => {
      window.removeEventListener("resize", onResize);
      termRef.current?.dispose();
      termRef.current = null;
    };
  }, []);

  useEffect(() => {
    void (async () => {
      const data = await getProjects({ page: 1, page_size: 1000 });
      setProjects(data.list);
      if (data.list[0]) {
        const pid = data.list[0].id;
        form.setFieldsValue({ project_id: pid, tail_lines: 200 });
        void reloadServers(pid);
      }
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function reloadServers(projectId?: number) {
    if (!projectId) return;
    const data = await getProjectServers(projectId, { page: 1, page_size: 1000 });
    setServers(data.list);
  }

  async function reloadServices(projectId?: number, serverId?: number) {
    if (!projectId) return;
    const data = await getProjectServices(projectId, { page: 1, page_size: 1000, server_id: serverId });
    setServices(data.list);
  }

  async function reloadSources(projectId?: number, serviceId?: number) {
    if (!projectId) return;
    const data = await getProjectLogSources(projectId, { page: 1, page_size: 1000, service_id: serviceId });
    setSources(data.list);
  }

  async function reloadMatchedFiles(projectId?: number, serverId?: number, source?: LogSourceItem) {
    setMatchedFiles([]);
    if (!projectId || !serverId || !source) return;
    if ((source.log_type ?? "").toLowerCase() !== "file") return;
    const sourcePath = (source.path ?? "").trim();
    if (!sourcePath) return;
    try {
      const data = await getProjectAgentDiscovery(projectId, { server_id: serverId, kind: "file", limit: 2000 });
      const list = data.list
        .filter((it) => pathMatchesSource(it.value, sourcePath))
        .sort((a, b) => a.value.localeCompare(b.value));
      setMatchedFiles(list);
    } catch {
      setMatchedFiles([]);
    }
  }

  async function start() {
    const values = await form.validateFields();
    const projectId = values.project_id;
    const serverId = values.server_id;
    const sourceId = values.log_source_id;
    if (!projectId || !serverId || !sourceId) {
      message.error("请选择 project/server/log source");
      return;
    }

    stop();
    termRef.current?.clear();
    termRef.current?.writeln(`Streaming logs... project=${projectId} server=${serverId} source=${sourceId}`);

    const params: Record<string, string> = {
      project_id: String(projectId),
      server_id: String(serverId),
      log_source_id: String(sourceId),
      tail_lines: String(values.tail_lines ?? 200),
    };
    if (values.include) params.include = values.include;
    if (values.exclude) params.exclude = values.exclude;
    if (values.highlight) params.highlight = values.highlight;
    if (values.file_path && values.file_path.trim() !== "") params.file_path = values.file_path.trim();
    params.source = "agent";

    const url = buildSseUrl(projectId, params);
    const token = getToken();
    const ac = new AbortController();
    abortRef.current = ac;
    setStreaming(true);
    setStreamModeHint("运行中：agent");

    try {
      const resp = await fetch(url, { headers: { Authorization: token ? `Bearer ${token}` : "" }, signal: ac.signal });
      if (!resp.ok || !resp.body) {
        throw new Error(`stream failed: ${resp.status}`);
      }
      const reader = resp.body.getReader();
      const decoder = new TextDecoder("utf-8");
      let buf = "";
      while (true) {
        const { done, value } = await reader.read();
        if (done) break;
        buf += decoder.decode(value, { stream: true });
        let idx: number;
        while ((idx = buf.indexOf("\n\n")) >= 0) {
          const rawEvent = buf.slice(0, idx);
          buf = buf.slice(idx + 2);
          const lines = rawEvent.split("\n");
          const eventName = lines.find((l) => l.startsWith("event:"))?.slice("event:".length).trim();
          const dataLine = lines.find((l) => l.startsWith("data:"))?.slice("data:".length).trim();
          if (!dataLine) continue;
          if (eventName === "ping") continue;
          if (eventName === "log") {
            try {
              const payload = JSON.parse(dataLine) as { line?: string };
              if (payload.line != null) termRef.current?.writeln(payload.line);
            } catch {
              termRef.current?.writeln(dataLine);
            }
          }
        }
      }
    } catch (e: any) {
      if (e?.name !== "AbortError") {
        message.error(String(e?.message ?? e));
      }
    } finally {
      setStreaming(false);
      abortRef.current = null;
      setStreamModeHint("已停止");
    }
  }

  function stop() {
    abortRef.current?.abort();
    abortRef.current = null;
    setStreaming(false);
    setStreamModeHint("已停止");
  }

  async function refreshAgentHint(projectId?: number, serverId?: number, logSourceId?: number) {
    if (!projectId || !serverId) {
      setAgentStatus({
        stateText: "未检测",
        online: false,
        recentPublishing: false,
        lastSeenText: "-",
        checked: false,
      });
      return;
    }
    try {
      const st = await getProjectAgentStatus(projectId, { server_id: serverId, log_source_id: logSourceId });
      setAgentStatus({
        stateText: st.online ? "在线" : "离线",
        online: st.online,
        recentPublishing: st.recent_publishing,
        lastSeenText: st.last_seen_at ? formatDateTime(st.last_seen_at) : "-",
        checked: true,
      });
    } catch {
      setAgentStatus({
        stateText: "检测失败",
        online: false,
        recentPublishing: false,
        lastSeenText: "-",
        checked: true,
      });
    }
  }

  useEffect(() => {
    void refreshAgentHint(watchProjectId, watchServerId, watchLogSourceId);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [watchProjectId, watchServerId, watchLogSourceId]);

  useEffect(() => {
    form.setFieldValue("file_path", undefined);
    void reloadMatchedFiles(watchProjectId, watchServerId, selectedSource);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [watchProjectId, watchServerId, selectedSource?.id, selectedSource?.path, selectedSource?.log_type]);

  return (
    <Card
      title="日志平台"
      extra={
        <Space>
          <Button icon={<ReloadOutlined />} onClick={() => void reloadServers(form.getFieldValue("project_id"))}>
            刷新服务器
          </Button>
          <Button icon={<DownloadOutlined />} onClick={() => {
            const v = form.getFieldsValue();
            if (!v.project_id || !v.server_id || !v.log_source_id) return;
            void (async () => {
              const blob = await exportProjectLogs(v.project_id!, {
                server_id: v.server_id!,
                log_source_id: v.log_source_id!,
                max_lines: v.tail_lines ?? 2000,
                include: v.include,
                exclude: v.exclude,
              });
              const url = window.URL.createObjectURL(blob);
              const a = document.createElement("a");
              a.href = url;
              a.download = `project-${v.project_id}-server-${v.server_id}-logs.txt`;
              a.click();
              window.URL.revokeObjectURL(url);
            })();
          }}>
            导出
          </Button>
          {streaming ? (
            <Button danger icon={<PauseOutlined />} onClick={stop}>
              停止
            </Button>
          ) : (
            <Button type="primary" icon={<PlayCircleOutlined />} onClick={() => void start()}>
              开始
            </Button>
          )}
        </Space>
      }
    >
      <div
        style={{
          marginBottom: 12,
          padding: "10px 12px",
          border: "1px solid #e6f4ff",
          background: "#f6ffed",
          borderRadius: 8,
        }}
      >
        <Space size={12} wrap>
          <span style={{ fontWeight: 600 }}>采集状态面板</span>
          <Tag color={streaming ? "processing" : "default"}>流状态：{streamModeHint}</Tag>
          <Tag color={!agentStatus.checked ? "default" : agentStatus.online ? "success" : "error"}>
            Agent：{agentStatus.stateText}
          </Tag>
          <Tag color={!agentStatus.checked ? "default" : agentStatus.recentPublishing ? "success" : "warning"}>
            上报：{!agentStatus.checked ? "未检测" : agentStatus.recentPublishing ? "最近有上报" : "最近无上报"}
          </Tag>
          <Tag>最后心跳：{agentStatus.lastSeenText}</Tag>
        </Space>
      </div>

      <Form form={form} layout="vertical">
        <Row gutter={12}>
          <Col span={5}>
            <Form.Item label="项目" name="project_id">
              <Select
                options={projectOptions}
                onChange={(pid) => {
                  form.setFieldsValue({ server_id: undefined, service_id: undefined, log_source_id: undefined, file_path: undefined });
                  setServers([]);
                  setServices([]);
                  setSources([]);
                  setMatchedFiles([]);
                  void reloadServers(pid);
                }}
              />
            </Form.Item>
          </Col>
          <Col span={5}>
            <Form.Item label="服务器" name="server_id">
              <Select
                options={serverOptions}
                placeholder="先选择服务器"
                onChange={(sid) => {
                  const pid = form.getFieldValue("project_id");
                  form.setFieldsValue({ service_id: undefined, log_source_id: undefined, file_path: undefined });
                  setServices([]);
                  setSources([]);
                  setMatchedFiles([]);
                  void reloadServices(pid, sid);
                  void refreshAgentHint(pid, sid, undefined);
                }}
              />
            </Form.Item>
          </Col>
          <Col span={5}>
            <Form.Item label="服务" name="service_id">
              <Select
                options={serviceOptions}
                placeholder="服务配置请到“服务配置”页面"
                allowClear
                onChange={(svcId) => {
                  const pid = form.getFieldValue("project_id");
                  form.setFieldsValue({ log_source_id: undefined, file_path: undefined });
                  setSources([]);
                  setMatchedFiles([]);
                  void reloadSources(pid, svcId);
                  void refreshAgentHint(pid, form.getFieldValue("server_id"), undefined);
                }}
              />
            </Form.Item>
          </Col>
          <Col span={5}>
            <Form.Item label="日志源" name="log_source_id">
              <Select
                options={sourceOptions}
                placeholder="日志源配置请到“日志源配置”页面"
                onChange={(logSourceId) => {
                  form.setFieldValue("file_path", undefined);
                  void refreshAgentHint(form.getFieldValue("project_id"), form.getFieldValue("server_id"), logSourceId);
                }}
              />
            </Form.Item>
          </Col>
          <Col span={4}>
            <Form.Item label="日志文件" name="file_path">
              <Select
                options={fileOptions}
                placeholder="默认全部匹配文件"
                allowClear
                disabled={!selectedSource || (selectedSource.log_type ?? "").toLowerCase() !== "file"}
              />
            </Form.Item>
          </Col>
        </Row>

        <Row gutter={12}>
          <Col span={4}>
            <Form.Item label="Tail 行数" name="tail_lines">
              <InputNumber min={1} max={5000} style={{ width: "100%" }} />
            </Form.Item>
          </Col>
          <Col span={6}>
            <Form.Item label="include（regex）" name="include">
              <Input placeholder="可选" />
            </Form.Item>
          </Col>
          <Col span={6}>
            <Form.Item label="exclude（regex）" name="exclude">
              <Input placeholder="可选" />
            </Form.Item>
          </Col>
          <Col span={8}>
            <Form.Item label="highlight（关键字）" name="highlight">
              <Input placeholder="可选" />
            </Form.Item>
          </Col>
        </Row>
      </Form>

      <div style={{ height: 520, border: "1px solid #f0f0f0", borderRadius: 6, overflow: "hidden" }}>
        <div ref={termHostRef} style={{ height: "100%", width: "100%" }} />
      </div>
    </Card>
  );
}

