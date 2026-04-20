import { PlayCircleOutlined, ReloadOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Form, Input, Space, Tabs, Tag, Typography, message } from "antd";
import { useEffect, useMemo, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import { execProjectServerCommand, getProjectServerDetail, type ServerItem } from "../services/projects";
import { useAuth } from "../contexts/auth-context";
import { Terminal } from "xterm";
import { FitAddon } from "xterm-addon-fit";
import "xterm/css/xterm.css";

type ExecForm = {
  command: string;
  timeout_sec?: number;
};

export function ServerConsolePage() {
  const { token } = useAuth();
  const [searchParams] = useSearchParams();
  const projectId = Number(searchParams.get("project_id") || 0);
  const serverId = Number(searchParams.get("server_id") || 0);
  const [server, setServer] = useState<ServerItem | null>(null);
  const [loading, setLoading] = useState(false);
  const [running, setRunning] = useState(false);
  const [result, setResult] = useState<{ stdout: string; stderr: string; exit_code: number; duration_ms: number; truncated: boolean } | null>(null);
  const [terminalConnected, setTerminalConnected] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);
  const termBoxRef = useRef<HTMLDivElement | null>(null);
  const xtermRef = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const dataDisposableRef = useRef<{ dispose: () => void } | null>(null);
  const [form] = Form.useForm<ExecForm>();

  const validParams = useMemo(() => projectId > 0 && serverId > 0, [projectId, serverId]);

  useEffect(() => {
    if (!validParams) return;
    void loadDetail();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [validParams, projectId, serverId]);

  async function loadDetail() {
    setLoading(true);
    try {
      const data = await getProjectServerDetail(projectId, serverId);
      setServer(data);
    } finally {
      setLoading(false);
    }
  }

  async function runCommand() {
    if (!validParams) return;
    const values = await form.validateFields();
    setRunning(true);
    try {
      const res = await execProjectServerCommand(projectId, serverId, {
        command: values.command,
        timeout_sec: values.timeout_sec ?? 20,
      });
      setResult(res);
      if (res.exit_code === 0) {
        message.success("命令执行成功");
      } else {
        message.warning(`命令执行完成，退出码 ${res.exit_code}`);
      }
    } finally {
      setRunning(false);
    }
  }

  function appendTerminalText(text: string) {
    if (!xtermRef.current) return;
    xtermRef.current.write(text);
  }

  function buildTerminalWSURL() {
    const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
    const host = window.location.host;
    const q = new URLSearchParams();
    if (token) q.set("token", token);
    return `${proto}//${host}/api/v1/projects/${projectId}/servers/${serverId}/terminal/ws?${q.toString()}`;
  }

  function openTerminal() {
    if (!validParams) return;
    if (!token) {
      message.error("未获取到登录 token，无法建立终端连接");
      return;
    }
    if (wsRef.current && wsRef.current.readyState === WebSocket.OPEN) return;
    const ws = new WebSocket(buildTerminalWSURL());
    wsRef.current = ws;
    ws.onopen = () => {
      setTerminalConnected(true);
      appendTerminalText("\r\n[connected]\r\n");
      fitAddonRef.current?.fit();
      const cols = Math.max(80, xtermRef.current?.cols ?? 120);
      const rows = Math.max(24, xtermRef.current?.rows ?? 40);
      ws.send(JSON.stringify({ type: "resize", cols, rows }));
    };
    ws.onmessage = (ev) => {
      try {
        const payload = JSON.parse(String(ev.data)) as { type?: string; data?: string };
        if (payload.type === "stdout" && typeof payload.data === "string") {
          appendTerminalText(payload.data);
          return;
        }
        if (payload.type === "error" && payload.data) {
          appendTerminalText(`\r\n[error] ${payload.data}\r\n`);
          return;
        }
        if (payload.type === "exit") {
          appendTerminalText("\r\n[session exited]\r\n");
          setTerminalConnected(false);
          return;
        }
      } catch {
        appendTerminalText(String(ev.data));
      }
    };
    ws.onerror = () => {
      appendTerminalText("\r\n[websocket error]\r\n");
    };
    ws.onclose = (ev) => {
      setTerminalConnected(false);
      wsRef.current = null;
      appendTerminalText(`\r\n[disconnected code=${ev.code}${ev.reason ? ` reason=${ev.reason}` : ""}]\r\n`);
    };
  }

  function closeTerminal() {
    const ws = wsRef.current;
    if (!ws) return;
    try {
      ws.send(JSON.stringify({ type: "close" }));
    } catch {
      // ignore
    }
    ws.close();
    wsRef.current = null;
    setTerminalConnected(false);
  }

  function sendTerminalInput(text: string) {
    const ws = wsRef.current;
    if (!ws || ws.readyState !== WebSocket.OPEN) return;
    ws.send(JSON.stringify({ type: "input", data: text }));
  }

  useEffect(() => {
    if (xtermRef.current || !termBoxRef.current) return;
    const term = new Terminal({
      cursorBlink: true,
      convertEol: true,
      fontFamily: "Consolas, Menlo, Monaco, monospace",
      fontSize: 13,
      lineHeight: 1.25,
      theme: {
        background: "#0b1220",
        foreground: "#d7e3ff",
      },
      scrollback: 5000,
    });
    const fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    term.open(termBoxRef.current);
    fitAddon.fit();
    term.writeln("Ready. Click '连接终端' to start.");

    dataDisposableRef.current = term.onData((data) => {
      sendTerminalInput(data);
    });
    term.attachCustomKeyEventHandler((ev) => {
      if ((ev.ctrlKey || ev.metaKey) && ev.shiftKey && ev.type === "keydown") {
        const key = ev.key.toLowerCase();
        if (key === "c") {
          const selected = term.getSelection();
          if (selected) {
            void navigator.clipboard?.writeText(selected);
          } else {
            sendTerminalInput("\u0003");
          }
          return false;
        }
        if (key === "v") {
          void navigator.clipboard?.readText().then((txt) => {
            if (txt) {
              sendTerminalInput(txt);
            }
          });
          return false;
        }
      }
      return true;
    });

    xtermRef.current = term;
    fitAddonRef.current = fitAddon;

    const resizeObs = new ResizeObserver(() => {
      fitAddon.fit();
      const ws = wsRef.current;
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({ type: "resize", cols: term.cols, rows: term.rows }));
      }
    });
    resizeObs.observe(termBoxRef.current);

    return () => {
      resizeObs.disconnect();
      dataDisposableRef.current?.dispose();
      dataDisposableRef.current = null;
      xtermRef.current?.dispose();
      xtermRef.current = null;
      fitAddonRef.current = null;
    };
  });

  useEffect(() => {
    return () => {
      closeTerminal();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  if (!validParams) {
    return (
      <Card className="table-card" title="服务器操作">
        <Alert type="error" showIcon message="参数不完整" description="请从服务器管理页面点击“连接”进入。" />
      </Card>
    );
  }

  return (
    <Space direction="vertical" size={12} style={{ width: "100%" }}>
      <Card
        className="table-card"
        loading={loading}
        title="服务器操作台"
        extra={<Link to="/project-servers">返回服务器管理</Link>}
      >
        <Space wrap>
          <Tag color="blue">Project #{projectId}</Tag>
          <Tag>Server #{serverId}</Tag>
          <Tag>{server?.source_type === "cloud" ? `云/${server?.provider || "-"}` : "自建"}</Tag>
          <Tag>{server?.host || "-"}</Tag>
        </Space>
      </Card>

      <Card className="table-card" bodyStyle={{ paddingTop: 8 }}>
        <Tabs
          items={[
            {
              key: "terminal",
              label: "交互终端（WebSocket）",
              children: (
                <Space direction="vertical" size={12} style={{ width: "100%" }}>
                  <Space wrap>
                    <Button type="primary" onClick={openTerminal} disabled={terminalConnected}>连接终端</Button>
                    <Button onClick={closeTerminal} disabled={!terminalConnected}>断开</Button>
                    <Button onClick={() => sendTerminalInput("\u0003")} disabled={!terminalConnected}>发送 Ctrl+C</Button>
                    <Button onClick={() => xtermRef.current?.clear()}>清屏</Button>
                    <Tag color={terminalConnected ? "success" : "default"}>{terminalConnected ? "已连接" : "未连接"}</Tag>
                    <Tag>快捷键: Ctrl+Shift+C 复制/中断, Ctrl+Shift+V 粘贴</Tag>
                  </Space>
                  <div
                    ref={termBoxRef}
                    style={{ minHeight: 340, maxHeight: 520, overflow: "hidden", borderRadius: 10, padding: 8, background: "#0b1220" }}
                  />
                </Space>
              ),
            },
            {
              key: "oneshot",
              label: "单次命令执行",
              children: (
                <Space direction="vertical" size={12} style={{ width: "100%" }}>
                  <Form
                    form={form}
                    layout="vertical"
                    initialValues={{ command: "uname -a", timeout_sec: 20 }}
                    onFinish={() => void runCommand()}
                  >
                    <Form.Item
                      label="命令"
                      name="command"
                      rules={[{ required: true, message: "请输入要执行的命令" }]}
                    >
                      <Input.TextArea rows={4} placeholder="例如：uname -a && whoami" />
                    </Form.Item>
                    <Form.Item label="超时时间（秒）" name="timeout_sec">
                      <Input type="number" min={1} max={120} style={{ width: 180 }} />
                    </Form.Item>
                    <Space>
                      <Button type="primary" icon={<PlayCircleOutlined />} onClick={() => void runCommand()} loading={running}>
                        执行
                      </Button>
                      <Button icon={<ReloadOutlined />} onClick={() => setResult(null)}>
                        清空结果
                      </Button>
                    </Space>
                  </Form>

                  <Card size="small" title="执行结果">
                    {result ? (
                      <Space direction="vertical" size={10} style={{ width: "100%" }}>
                        <Space>
                          <Tag color={result.exit_code === 0 ? "success" : "error"}>退出码 {result.exit_code}</Tag>
                          <Tag>耗时 {result.duration_ms} ms</Tag>
                          {result.truncated ? <Tag color="warning">输出已截断</Tag> : null}
                        </Space>
                        <Typography.Text strong>STDOUT</Typography.Text>
                        <pre style={{ margin: 0, padding: 12, borderRadius: 10, background: "#0b1220", color: "#d7e3ff", maxHeight: 280, overflow: "auto" }}>{result.stdout || "(empty)"}</pre>
                        <Typography.Text strong>STDERR</Typography.Text>
                        <pre style={{ margin: 0, padding: 12, borderRadius: 10, background: "#0b1220", color: "#ffd5d5", maxHeight: 220, overflow: "auto" }}>{result.stderr || "(empty)"}</pre>
                      </Space>
                    ) : (
                      <Typography.Text type="secondary">暂无执行结果。</Typography.Text>
                    )}
                  </Card>
                </Space>
              ),
            },
          ]}
        />
      </Card>
    </Space>
  );
}
