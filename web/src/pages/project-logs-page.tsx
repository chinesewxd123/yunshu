import { PauseOutlined, PlayCircleOutlined, PlusOutlined, ReloadOutlined, DownloadOutlined } from "@ant-design/icons";
import { Button, Card, Col, Form, Input, InputNumber, Row, Select, Space, Tag, message } from "antd";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Terminal } from "xterm";
import { FitAddon } from "xterm-addon-fit";
import "xterm/css/xterm.css";
import { useLogStream } from "../contexts/log-stream-context";
import { parseLogStreamFromSearch, syncLogStreamSearchParams } from "../lib/log-stream-session";
import {
  exportProjectLogs,
  getProjectAgentDiscovery,
  getProjectAgentStatus,
  getProjectLogSources,
  getProjectServers,
  getProjectServices,
  getProjects,
  upsertProjectLogSource,
  type AgentDiscoveryItem,
  type LogSourceItem,
  type ProjectItem,
  type ServerItem,
  type ServiceItem,
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
  const logStream = useLogStream();
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [servers, setServers] = useState<ServerItem[]>([]);
  const [services, setServices] = useState<ServiceItem[]>([]);
  const [sources, setSources] = useState<LogSourceItem[]>([]);
  const [discoveryFiles, setDiscoveryFiles] = useState<AgentDiscoveryItem[]>([]);
  const [creatingPath, setCreatingPath] = useState<string | null>(null);

  const [agentStatus, setAgentStatus] = useState({
    stateText: "未检测",
    online: false,
    recentPublishing: false,
    lastSeenText: "-",
    checked: false,
  });

  const termHostRef = useRef<HTMLDivElement | null>(null);
  const termWrapRef = useRef<HTMLDivElement | null>(null);
  const termRef = useRef<Terminal | null>(null);
  const fitRef = useRef<FitAddon | null>(null);
  const bootstrappedRef = useRef(false);

  const [form] = Form.useForm<StreamForm>();
  const watchProjectId = Form.useWatch("project_id", form);
  const watchServerId = Form.useWatch("server_id", form);
  const watchLogSourceId = Form.useWatch("log_source_id", form);

  const { streaming, streamModeHint, lineCount, linesPerSec, paused, start, stop, togglePause, attachWriter, registerTerminalClear, setForm } =
    logStream;

  const projectOptions = useMemo(() => projects.map((p) => ({ value: p.id, label: `${p.name} (${p.code})` })), [projects]);
  const serverOptions = useMemo(
    () => servers.map((s) => ({ value: s.id, label: `${s.name} ${s.host}:${s.port} (${s.os_type || "-"} ${s.os_arch || "-"})` })),
    [servers],
  );
  const serviceOptions = useMemo(() => services.map((s) => ({ value: s.id, label: s.name })), [services]);
  const sourceOptions = useMemo(() => sources.map((s) => ({ value: s.id, label: `${s.log_type}:${s.path}` })), [sources]);
  const selectedSource = useMemo(() => sources.find((s) => s.id === watchLogSourceId), [sources, watchLogSourceId]);
  const matchedFiles = useMemo(() => {
    if (!selectedSource || (selectedSource.log_type ?? "").toLowerCase() !== "file") return [];
    const sourcePath = (selectedSource.path ?? "").trim();
    if (!sourcePath) return [];
    return discoveryFiles
      .filter((it) => pathMatchesSource(it.value, sourcePath))
      .sort((a, b) => a.value.localeCompare(b.value));
  }, [discoveryFiles, selectedSource]);

  const discoveryOrphans = useMemo(() => {
    if (!watchServerId) return [];
    const list =
      sources.length === 0
        ? discoveryFiles
        : discoveryFiles.filter((it) => !sources.some((s) => pathMatchesSource(it.value, s.path)));
    return list.sort((a, b) => a.value.localeCompare(b.value)).slice(0, 12);
  }, [discoveryFiles, sources, watchServerId]);

  const fileOptions = useMemo(
    () => [{ value: "", label: "全部匹配文件" }, ...matchedFiles.map((it) => ({ value: it.value, label: it.value }))],
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
    const wrap = termWrapRef.current;
    const ro =
      wrap &&
      new ResizeObserver(() => {
        fitRef.current?.fit();
      });
    if (wrap && ro) ro.observe(wrap);
    return () => {
      window.removeEventListener("resize", onResize);
      ro?.disconnect();
      termRef.current?.dispose();
      termRef.current = null;
    };
  }, []);

  useEffect(() => {
    const detachWrite = attachWriter((line) => {
      termRef.current?.writeln(line);
    });
    const detachClear = registerTerminalClear(() => {
      termRef.current?.clear();
    });
    return () => {
      detachWrite();
      detachClear();
    };
  }, [attachWriter, registerTerminalClear]);

  const reloadServers = useCallback(async (projectId?: number) => {
    if (!projectId) return;
    const data = await getProjectServers(projectId, { page: 1, page_size: 1000 });
    setServers(data.list);
  }, []);

  const reloadServices = useCallback(async (projectId?: number, serverId?: number) => {
    if (!projectId) return;
    const data = await getProjectServices(projectId, { page: 1, page_size: 1000, server_id: serverId });
    setServices(data.list);
  }, []);

  const reloadSources = useCallback(async (projectId?: number, serviceId?: number) => {
    if (!projectId) return;
    const data = await getProjectLogSources(projectId, { page: 1, page_size: 1000, service_id: serviceId });
    setSources(data.list);
    return data.list;
  }, []);

  const reloadDiscoveryFiles = useCallback(async (projectId?: number, serverId?: number) => {
    setDiscoveryFiles([]);
    if (!projectId || !serverId) return;
    try {
      const data = await getProjectAgentDiscovery(projectId, {
        server_id: serverId,
        kind: "file",
        limit: 2000,
        fresh_hours: 24 * 7,
      });
      setDiscoveryFiles(data.list);
    } catch {
      setDiscoveryFiles([]);
    }
  }, []);

  useEffect(() => {
    void (async () => {
      const data = await getProjects({ page: 1, page_size: 1000 });
      setProjects(data.list);

      const fromUrl = parseLogStreamFromSearch();
      const merged: StreamForm = {
        project_id: fromUrl.project_id ?? data.list[0]?.id,
        server_id: fromUrl.server_id,
        service_id: fromUrl.service_id,
        log_source_id: fromUrl.log_source_id,
        file_path: fromUrl.file_path,
        tail_lines: fromUrl.tail_lines ?? 200,
        include: fromUrl.include,
        exclude: fromUrl.exclude,
        highlight: fromUrl.highlight,
      };
      form.setFieldsValue(merged);
      setForm(merged);

      if (merged.project_id) {
        await reloadServers(merged.project_id);
        if (merged.server_id) {
          await reloadServices(merged.project_id, merged.server_id);
          if (merged.service_id) {
            await reloadSources(merged.project_id, merged.service_id);
          }
          if (merged.server_id) {
            await reloadDiscoveryFiles(merged.project_id, merged.server_id);
          }
        }
      }

      if (!bootstrappedRef.current) {
        bootstrappedRef.current = true;
        if (fromUrl.autostart && merged.project_id && merged.server_id && merged.log_source_id) {
          try {
            await start(merged);
          } catch (e: unknown) {
            message.error(String((e as Error)?.message ?? e));
          }
        }
      }
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function handleStart() {
    try {
      const values = await form.validateFields();
      setForm(values);
      syncLogStreamSearchParams(values, true);
      await start(values);
    } catch (e: unknown) {
      const err = e as { errorFields?: unknown; message?: string; name?: string };
      if (err?.errorFields) return;
      if (err?.name === "AbortError") return;
      message.error(String(err?.message ?? e));
    }
  }

  function handleStop() {
    const values = form.getFieldsValue();
    syncLogStreamSearchParams(values, false);
    stop();
  }

  function handleFormChange(_: Partial<StreamForm>, all: StreamForm) {
    setForm(all);
    syncLogStreamSearchParams(all, streaming);
  }

  async function createSourceFromDiscovery(filePath: string) {
    const pid = form.getFieldValue("project_id");
    const svcId = form.getFieldValue("service_id");
    if (!pid || !svcId) {
      message.warning("请先选择项目与服务，再一键创建日志源");
      return;
    }
    setCreatingPath(filePath);
    try {
      await upsertProjectLogSource(pid, {
        service_id: svcId,
        log_type: "file",
        path: filePath,
        status: 1,
      });
      message.success("已创建日志源");
      await reloadSources(pid, svcId);
      await reloadDiscoveryFiles(pid, form.getFieldValue("server_id"));
    } catch (e: unknown) {
      message.error(String((e as Error)?.message ?? e));
    } finally {
      setCreatingPath(null);
    }
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
  }, [watchProjectId, watchServerId, watchLogSourceId]);

  useEffect(() => {
    form.setFieldValue("file_path", undefined);
  }, [watchProjectId, watchServerId, selectedSource?.id, form]);

  useEffect(() => {
    void reloadDiscoveryFiles(watchProjectId, watchServerId);
  }, [watchProjectId, watchServerId, reloadDiscoveryFiles]);

  return (
    <div className="project-logs-page">
      <Card
        className="table-card project-logs-card"
        title="日志平台"
        extra={
          <Space>
            <Button icon={<ReloadOutlined />} onClick={() => void reloadServers(form.getFieldValue("project_id"))}>
              刷新服务器
            </Button>
            <Button
              icon={<DownloadOutlined />}
              onClick={() => {
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
              }}
            >
              导出
            </Button>
            {streaming ? (
              <>
                <Button icon={paused ? <PlayCircleOutlined /> : <PauseOutlined />} onClick={togglePause}>
                  {paused ? "继续显示" : "暂停显示"}
                </Button>
                <Button danger onClick={handleStop}>
                  停止
                </Button>
              </>
            ) : (
              <Button type="primary" icon={<PlayCircleOutlined />} onClick={() => void handleStart()}>
                开始
              </Button>
            )}
          </Space>
        }
      >
        <div className="project-logs-status-panel">
          <Space size={12} wrap>
            <span style={{ fontWeight: 600 }}>采集状态面板</span>
            <Tag color={streaming ? "processing" : "default"}>流状态：{streamModeHint}</Tag>
            <Tag color={streaming ? "blue" : "default"}>
              {lineCount} 行 · {linesPerSec} 行/秒
            </Tag>
            <Tag color={!agentStatus.checked ? "default" : agentStatus.online ? "success" : "error"}>
              Agent：{agentStatus.stateText}
            </Tag>
            <Tag color={!agentStatus.checked ? "default" : agentStatus.recentPublishing ? "success" : "warning"}>
              上报：{!agentStatus.checked ? "未检测" : agentStatus.recentPublishing ? "最近有上报" : "最近无上报"}
            </Tag>
            <Tag>最后心跳：{agentStatus.lastSeenText}</Tag>
          </Space>
        </div>

        {discoveryOrphans.length > 0 ? (
          <div className="project-logs-discovery-panel">
            <Space size={8} wrap align="start">
              <span style={{ fontWeight: 600 }}>Agent 发现（未配置日志源）：</span>
              {discoveryOrphans.map((it) => (
                <Tag key={it.value}>
                  <span className="project-logs-discovery-path">{it.value}</span>
                  <Button
                    type="link"
                    size="small"
                    icon={<PlusOutlined />}
                    loading={creatingPath === it.value}
                    onClick={() => void createSourceFromDiscovery(it.value)}
                  >
                    创建日志源
                  </Button>
                </Tag>
              ))}
            </Space>
          </div>
        ) : null}

        <Form form={form} layout="vertical" onValuesChange={handleFormChange}>
          <Row gutter={12}>
            <Col span={5}>
              <Form.Item label="项目" name="project_id">
                <Select
                  options={projectOptions}
                  onChange={(pid) => {
                    form.setFieldsValue({
                      server_id: undefined,
                      service_id: undefined,
                      log_source_id: undefined,
                      file_path: undefined,
                    });
                    setServers([]);
                    setServices([]);
                    setSources([]);
                    setDiscoveryFiles([]);
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
                    void reloadServices(pid, sid);
                    void refreshAgentHint(pid, sid, undefined);
                    void reloadDiscoveryFiles(pid, sid);
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

        <div ref={termWrapRef} className="project-logs-terminal-wrap">
          <div ref={termHostRef} className="project-logs-terminal-host" />
        </div>
      </Card>
    </div>
  );
}
