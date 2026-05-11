import { Progress, Tooltip, Typography } from "antd";

function barColor(pct: number): string {
  if (pct > 100) return "#dc2626";
  if (pct > 90) return "#ef4444";
  if (pct > 70) return "#f59e0b";
  return "#22c55e";
}

function MiniUsageBar({ label, pct }: { label: string; pct?: number }) {
  const raw = typeof pct === "number" && !Number.isNaN(pct) ? Math.max(0, pct) : 0;
  const clamped = Math.min(100, Math.round(raw));
  const text = raw < 10 && raw > 0 ? `${raw.toFixed(1)}%` : `${Math.round(raw)}%`;
  return (
    <div style={{ marginBottom: 4 }}>
      <div style={{ fontSize: 11, color: "#64748b", lineHeight: 1.2 }}>{label}</div>
      <Progress percent={clamped} size="small" strokeColor={barColor(raw)} showInfo format={() => text} />
    </div>
  );
}

export type PodUsageMetricsRow = {
  cpu_usage?: string;
  mem_usage?: string;
  cpu_pct_request?: number;
  cpu_pct_limit?: number;
  cpu_pct_node_alloc?: number;
  mem_pct_request?: number;
  mem_pct_limit?: number;
  mem_pct_node_alloc?: number;
};

export type WorkloadUsageMetricsRow = {
  cpu_usage?: string;
  mem_usage?: string;
  cpu_pct_request?: number;
  cpu_pct_limit?: number;
  mem_pct_request?: number;
  mem_pct_limit?: number;
};

/** Pod 列表：CPU 请求/上限/节点可分配 三维占比 */
export function PodCpuUsageBars({ row }: { row: PodUsageMetricsRow }) {
  const hasRt = Boolean(row.cpu_usage && row.cpu_usage !== "-");
  const reqZero = !row.cpu_pct_request || row.cpu_pct_request === 0;
  const hint =
    hasRt && reqZero
      ? "已有实时 CPU 用量；未设置 requests 或 requests 极小时「请求用量占比」可能为 0"
      : undefined;
  const inner = (
    <div style={{ minWidth: 152 }}>
      <MiniUsageBar label="请求用量占比" pct={row.cpu_pct_request} />
      <MiniUsageBar label="上限用量占比" pct={row.cpu_pct_limit} />
      <MiniUsageBar label="实时占节点" pct={row.cpu_pct_node_alloc} />
    </div>
  );
  return hint ? <Tooltip title={hint}>{inner}</Tooltip> : inner;
}

/** Pod 列表：内存 三维占比 */
export function PodMemUsageBars({ row }: { row: PodUsageMetricsRow }) {
  return (
    <div style={{ minWidth: 152 }}>
      <MiniUsageBar label="请求用量占比" pct={row.mem_pct_request} />
      <MiniUsageBar label="上限用量占比" pct={row.mem_pct_limit} />
      <MiniUsageBar label="实时占节点" pct={row.mem_pct_node_alloc} />
    </div>
  );
}

/** Deployment 等工作负载：相对模板×副本规模的请求/上限占比 */
export function WorkloadCpuUsageBars({ row }: { row: WorkloadUsageMetricsRow }) {
  return (
    <div style={{ minWidth: 140 }}>
      <MiniUsageBar label="请求用量占比" pct={row.cpu_pct_request} />
      <MiniUsageBar label="上限用量占比" pct={row.cpu_pct_limit} />
    </div>
  );
}

export function WorkloadMemUsageBars({ row }: { row: WorkloadUsageMetricsRow }) {
  return (
    <div style={{ minWidth: 140 }}>
      <MiniUsageBar label="请求用量占比" pct={row.mem_pct_request} />
      <MiniUsageBar label="上限用量占比" pct={row.mem_pct_limit} />
    </div>
  );
}

export function RealtimeUsageText({ cpu, mem }: { cpu?: string; mem?: string }) {
  return (
    <Typography.Text style={{ fontSize: 12, whiteSpace: "nowrap" }}>
      {(cpu && cpu !== "-" ? cpu : "-")} / {(mem && mem !== "-" ? mem : "-")}
    </Typography.Text>
  );
}
