import { ExclamationCircleOutlined, ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Drawer, Progress, Select, Space, Table, Tag, message } from "antd";
import { useDictOptions } from "../hooks/use-dict-options";
import { useEffect, useMemo, useState } from "react";
import {
  batchRefreshProjectAgentHeartbeat,
  getProjects,
  listProjectAgents,
  type ProjectAgentListItem,
  type ProjectItem,
} from "../services/projects";
import { formatDateTime } from "../utils/format";

export function AgentListPage() {
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [onlineFilter, setOnlineFilter] = useState<"all" | "online" | "offline">("all");
  const [healthStatusFilter, setHealthStatusFilter] = useState<string>("all");
  const [loading, setLoading] = useState(false);
  const [batchRefreshing, setBatchRefreshing] = useState(false);
  const [list, setList] = useState<ProjectAgentListItem[]>([]);
  const [selectedRowKeys, setSelectedRowKeys] = useState<Array<string | number>>([]);
  const [errorDrawerOpen, setErrorDrawerOpen] = useState(false);
  const [errorDetailRow, setErrorDetailRow] = useState<ProjectAgentListItem | null>(null);

  const projectOptions = useMemo(() => projects.map((p) => ({ value: p.id, label: `${p.name} (${p.code})` })), [projects]);
  const healthDictOptions = useDictOptions("log_agent_health_status");
  const healthFilterOptions = useMemo(
    () => [
      { label: "健康状态: 全部", value: "all" },
      ...healthDictOptions.map((o) => ({ label: o.label, value: String(o.value) })),
    ],
    [healthDictOptions],
  );

  useEffect(() => {
    void (async () => {
      const data = await getProjects({ page: 1, page_size: 1000 });
      setProjects(data.list ?? []);
      if (data.list?.[0]) {
        setProjectId(data.list[0].id);
      }
    })();
  }, []);

  async function load() {
    if (!projectId) return;
    setLoading(true);
    try {
      const data = await listProjectAgents(projectId, {
        online: onlineFilter === "all" ? undefined : onlineFilter === "online",
        health_status: healthStatusFilter === "all" ? undefined : healthStatusFilter,
      });
      setList(data.list ?? []);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [projectId, onlineFilter, healthStatusFilter]);

  async function batchRefreshHeartbeat() {
    if (!projectId) return;
    setBatchRefreshing(true);
    try {
      const serverIDs = selectedRowKeys.map((k) => Number(k)).filter((n) => Number.isFinite(n) && n > 0);
      const data = await batchRefreshProjectAgentHeartbeat(projectId, { server_ids: serverIDs.length ? serverIDs : undefined });
      setList(data.list ?? []);
      message.success(`已刷新 ${data.refreshed ?? 0} 台服务器心跳`);
    } finally {
      setBatchRefreshing(false);
    }
  }

  return (
    <Card
      className="table-card"
      title="Agent 列表"
      extra={
        <Space>
          <Select style={{ width: 260 }} value={projectId} onChange={setProjectId} options={projectOptions} placeholder="选择项目" />
          <Select
            style={{ width: 140 }}
            value={onlineFilter}
            onChange={setOnlineFilter}
            options={[
              { label: "在线状态: 全部", value: "all" },
              { label: "在线", value: "online" },
              { label: "离线", value: "offline" },
            ]}
          />
          <Select style={{ width: 170 }} value={healthStatusFilter} onChange={setHealthStatusFilter} options={healthFilterOptions} />
          <Button onClick={() => void batchRefreshHeartbeat()} loading={batchRefreshing} disabled={!projectId}>
            批量刷新心跳
          </Button>
          <Button icon={<ReloadOutlined />} onClick={() => void load()} loading={loading}>
            刷新
          </Button>
        </Space>
      }
    >
      <Table
        rowKey="server_id"
        loading={loading}
        dataSource={list}
        rowSelection={{
          selectedRowKeys,
          onChange: (keys) => setSelectedRowKeys(keys.map((k) => (typeof k === "bigint" ? Number(k) : k))),
        }}
        pagination={{ pageSize: 10, showSizeChanger: true, pageSizeOptions: [10, 20, 50, 100], showQuickJumper: true }}
        columns={[
          { title: "服务器", dataIndex: "server_name", width: 180, render: (v: string, r) => `${v || "-"} (${r.server_host || "-"})` },
          {
            title: "项目名称",
            dataIndex: "project_name",
            width: 160,
            ellipsis: true,
            render: (v: string) => v || projects.find((p) => p.id === projectId)?.name || "-",
          },
          { title: "Agent", dataIndex: "name", width: 170, render: (v: string) => v || "-" },
          { title: "版本", dataIndex: "version", width: 120, render: (v: string) => v || "-" },
          {
            title: "健康状态",
            dataIndex: "health_status",
            width: 130,
            render: (v: string) => {
              if (v === "running") return <Tag color="success">running</Tag>;
              if (v === "starting") return <Tag color="processing">starting</Tag>;
              if (v === "stopped" || v === "error") return <Tag color="error">{v || "error"}</Tag>;
              return <Tag>{v || "unknown"}</Tag>;
            },
          },
          {
            title: "在线",
            dataIndex: "online",
            width: 90,
            render: (v: boolean) => (v ? <Tag color="success">在线</Tag> : <Tag color="error">离线</Tag>),
          },
          {
            title: "上报",
            dataIndex: "recent_publishing",
            width: 90,
            render: (v?: boolean) => (v ? <Tag color="success">最近有</Tag> : <Tag>无</Tag>),
          },
          {
            title: "安装进度",
            dataIndex: "install_progress",
            width: 180,
            render: (v: number) => <Progress percent={Math.max(0, Math.min(100, Number(v || 0)))} size="small" />,
          },
          {
            title: "本机端口",
            dataIndex: "listen_port",
            width: 120,
            render: (v: number) =>
              v > 0 ? (
                v
              ) : (
                <span title="当前 Agent 不监听本地端口，仅主动连接平台 gRPC（出站）">无（仅出站）</span>
              ),
          },
          { title: "最近心跳", dataIndex: "last_seen_at", width: 170, render: (v: string) => (v ? formatDateTime(v) : "-") },
          {
            title: "最近错误",
            dataIndex: "last_error",
            ellipsis: true,
            render: (v: string, row: ProjectAgentListItem) =>
              v ? (
                <Button
                  type="link"
                  icon={<ExclamationCircleOutlined />}
                  onClick={() => {
                    setErrorDetailRow(row);
                    setErrorDrawerOpen(true);
                  }}
                >
                  查看错误
                </Button>
              ) : (
                "-"
              ),
          },
        ]}
      />
      <Drawer
        title={`错误详情 - ${errorDetailRow?.server_name || "-"}`}
        open={errorDrawerOpen}
        onClose={() => setErrorDrawerOpen(false)}
        width={640}
      >
        <p><strong>服务器:</strong> {errorDetailRow?.server_name || "-"} ({errorDetailRow?.server_host || "-"})</p>
        <p><strong>Agent:</strong> {errorDetailRow?.name || "-"} / {errorDetailRow?.version || "-"}</p>
        <p><strong>健康状态:</strong> {errorDetailRow?.health_status || "-"}</p>
        <p><strong>最近心跳:</strong> {errorDetailRow?.last_seen_at ? formatDateTime(errorDetailRow.last_seen_at) : "-"}</p>
        <p><strong>错误内容:</strong></p>
        <pre style={{ whiteSpace: "pre-wrap", wordBreak: "break-word", background: "#fafafa", padding: 12, borderRadius: 8 }}>
          {errorDetailRow?.last_error || "-"}
        </pre>
      </Drawer>
    </Card>
  );
}
