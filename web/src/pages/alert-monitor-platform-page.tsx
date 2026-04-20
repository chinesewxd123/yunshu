import {
  DeleteOutlined,
  EditOutlined,
  CalendarOutlined,
  MinusCircleOutlined,
  PlusOutlined,
  ReloadOutlined,
  TeamOutlined,
} from "@ant-design/icons";
import type { TreeSelectProps } from "antd";
import {
  Alert,
  AutoComplete,
  Badge,
  Button,
  Calendar,
  Card,
  Col,
  Collapse,
  DatePicker,
  Drawer,
  Form,
  Input,
  InputNumber,
  Popconfirm,
  Radio,
  Row,
  Segmented,
  Select,
  Space,
  Switch,
  Table,
  Tabs,
  Tag,
  TreeSelect,
  Typography,
  message,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import type { Dayjs } from "dayjs";
import dayjs from "dayjs";
import "dayjs/locale/zh-cn";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "react-router-dom";
import type { DepartmentItem } from "../types/api";
import { getDepartmentTree } from "../services/departments";
import { getProjects, type ProjectItem } from "../services/projects";
import { getUsers } from "../services/users";
import {
  createAlertDatasource,
  createAlertMonitorRule,
  createAlertSilence,
  createDutyBlock,
  deleteAlertDatasource,
  deleteAlertMonitorRule,
  deleteAlertSilence,
  deleteDutyBlock,
  getMonitorRuleAssignees,
  listAlertDatasources,
  listAlertMonitorRules,
  listAlertSilences,
  listDutyBlocks,
  promActiveAlerts,
  promInstantQuery,
  promRangeQuery,
  updateAlertDatasource,
  updateAlertMonitorRule,
  updateAlertSilence,
  updateDutyBlock,
  upsertMonitorRuleAssignees,
  type AlertDatasourceItem,
  type AlertDutyBlockItem,
  type AlertMonitorRuleItem,
  type AlertSilenceItem,
} from "../services/alert-platform";
import { useDictOptions } from "../hooks/use-dict-options";
import type { UserUpdatePayload } from "../types/api";
import { getUser, updateUser } from "../services/users";
import { formatDateTime } from "../utils/format";
import { AlertConfigCenterPanel, type AlertConfigTab } from "./alert-config-center-panel";

dayjs.locale("zh-cn");

type TabKey = "datasources" | "config" | "silences" | "rules" | "promql";

type SilenceMatcherForm = { name: string; value: string; is_regex: boolean };

type PromNativeAlertRow = {
  key: string;
  alertname: string;
  state: string;
  labelsShort: string;
  activeAt?: string;
  labels: Record<string, string>;
};
type QuickSilenceTarget = {
  key: string;
  name: string;
  labels: Record<string, string>;
  startsAt: Dayjs;
  endsAt: Dayjs;
};
type RuleComparator = ">" | ">=" | "<" | "<=" | "==" | "!=";
type RuleBuilderLogic = "and" | "or";
type RuleBuilderCondition = { metric: string; comparator: RuleComparator; threshold: number | null };
type MetricLabelFilter = { key: string; op: "=" | "!=" | "=~" | "!~"; value: string };

function parseSilenceMatchersForForm(raw?: string): SilenceMatcherForm[] {
  const s = raw?.trim();
  if (!s) return [{ name: "alertname", value: "", is_regex: false }];
  try {
    const v = JSON.parse(s) as unknown;
    if (!Array.isArray(v)) return [{ name: "alertname", value: "", is_regex: false }];
    return v.map((row: unknown) => {
      const o = row as Record<string, unknown>;
      return {
        name: String(o?.name ?? "").trim(),
        value: String(o?.value ?? "").trim(),
        is_regex: Boolean(o?.is_regex),
      };
    });
  } catch {
    return [{ name: "alertname", value: "", is_regex: false }];
  }
}

function parsePrometheusActiveAlertsTable(body: unknown): PromNativeAlertRow[] {
  if (!body || typeof body !== "object") return [];
  const root = body as { data?: { alerts?: unknown[] } };
  const alerts = root.data?.alerts;
  if (!Array.isArray(alerts)) return [];
  return alerts.map((a, i) => {
    const row = (a ?? {}) as { labels?: Record<string, string>; state?: string; activeAt?: string };
    const labels = row.labels ?? {};
    const name = labels.alertname ?? "";
    const short = JSON.stringify(labels);
    return {
      key: String(i),
      alertname: String(name),
      state: String(row.state ?? ""),
      labelsShort: short.length > 140 ? `${short.slice(0, 140)}…` : short,
      activeAt: row.activeAt,
      labels,
    };
  });
}

function parseUintArrayJSON(raw?: string): number[] {
  const s = raw?.trim();
  if (!s) return [];
  try {
    const v = JSON.parse(s) as unknown;
    if (!Array.isArray(v)) return [];
    return v
      .map((x) => (typeof x === "number" ? x : typeof x === "string" && /^\d+$/.test(x) ? Number(x) : NaN))
      .filter((n) => !Number.isNaN(n));
  } catch {
    return [];
  }
}

function deptToTreeData(nodes: DepartmentItem[]): TreeSelectProps["treeData"] {
  return nodes.map((n) => ({
    title: n.name,
    value: n.id,
    children: n.children?.length ? deptToTreeData(n.children) : undefined,
  }));
}

function sortMetricKeys(a: string, b: string): number {
  if (a === "__name__") return -1;
  if (b === "__name__") return 1;
  return a.localeCompare(b);
}

function formatPromTimestampLocal(raw: string): string {
  const n = Number(raw);
  if (!Number.isFinite(n)) return raw;
  const ms = n > 1e12 ? n : n * 1000;
  return dayjs(ms).format("YYYY-MM-DD HH:mm:ss");
}

function isValidPromLabelKey(s: string): boolean {
  return /^[a-zA-Z_][a-zA-Z0-9_]*$/.test(String(s || "").trim());
}

function buildPromSelectorExpr(metric: string, filters: MetricLabelFilter[]): string {
  const m = String(metric || "").trim();
  if (!m) return "";
  const parts = filters
    .map((f) => ({
      key: String(f.key || "").trim(),
      op: f.op,
      value: String(f.value || "").trim(),
    }))
    .filter((f) => isValidPromLabelKey(f.key) && f.value !== "")
    .map((f) => `${f.key}${f.op}"${f.value.replace(/"/g, '\\"')}"`);
  if (!parts.length) return m;
  return `${m}{${parts.join(",")}}`;
}

function parsePromSelectorExpr(raw: string): { metric: string; filters: MetricLabelFilter[] } | null {
  const s = String(raw || "").trim();
  if (!s) return null;
  const m = s.match(/^([a-zA-Z_:][a-zA-Z0-9_:]*)(?:\{([\s\S]*)\})?$/);
  if (!m) return null;
  const metric = String(m[1] || "").trim();
  if (!metric) return null;
  const body = String(m[2] || "").trim();
  if (!body) return { metric, filters: [{ key: "instance", op: "=", value: "" }] };
  const filters: MetricLabelFilter[] = [];
  const re = /([a-zA-Z_][a-zA-Z0-9_]*)\s*(=~|!~|!=|=)\s*"((?:\\.|[^"\\])*)"\s*(?:,|$)/g;
  let match: RegExpExecArray | null;
  while ((match = re.exec(body)) !== null) {
    const key = String(match[1] || "").trim();
    const op = (match[2] as MetricLabelFilter["op"]) || "=";
    const value = String(match[3] || "").replace(/\\"/g, '"').trim();
    filters.push({ key, op, value });
  }
  return { metric, filters: filters.length ? filters : [{ key: "instance", op: "=", value: "" }] };
}

function detectPromFunctionKeyFromExpr(exprRaw: string): string | null {
  const s = String(exprRaw || "").trim().toLowerCase();
  if (!s) return null;
  if (/^histogram_quantile\s*\(/.test(s)) return "histogram_quantile";
  if (/^sum\s+by\s*\(/.test(s)) return "sum_by";
  if (/^avg_over_time\s*\(/.test(s)) return "avg_over_time";
  if (/^max_over_time\s*\(/.test(s)) return "max_over_time";
  if (/^min_over_time\s*\(/.test(s)) return "min_over_time";
  if (/^increase\s*\(/.test(s)) return "increase";
  if (/^irate\s*\(/.test(s)) return "irate";
  if (/^rate\s*\(/.test(s)) return "rate";
  if (/^ceil\s*\(/.test(s)) return "ceil";
  if (/^floor\s*\(/.test(s)) return "floor";
  if (/^round\s*\(/.test(s)) return "round";
  return null;
}

type PromTableView = {
  columns: ColumnsType<Record<string, string>>;
  dataSource: Record<string, string>[];
};

/** 将 Prometheus instant/range 的 data 段解析为表格（vector / matrix）。 */
function buildPromTableView(data: unknown): PromTableView | null {
  if (!data || typeof data !== "object") return null;
  const obj = data as Record<string, unknown>;
  const rt = String(obj.resultType ?? "");
  const result = obj.result;
  if (!Array.isArray(result) || result.length === 0) return null;

  if (rt === "vector") {
    const rows: Record<string, string>[] = [];
    const keySet = new Set<string>();
    let k = 0;
    for (const item of result as Array<{ metric?: Record<string, string>; value?: [string, string] }>) {
      const m = item.metric ?? {};
      const val = item.value;
      const row: Record<string, string> = { key: String(k++) };
      for (const [mk, mv] of Object.entries(m)) {
        keySet.add(mk);
        row[mk] = mv;
      }
      row.__timestamp__ = val?.[0] ?? "";
      row.__time_local__ = formatPromTimestampLocal(val?.[0] ?? "");
      row.__value__ = val?.[1] ?? "";
      keySet.add("__timestamp__");
      keySet.add("__time_local__");
      keySet.add("__value__");
      rows.push(row);
    }
    const metricKeys = [...keySet]
      .filter((x) => x !== "__timestamp__" && x !== "__time_local__" && x !== "__value__")
      .sort(sortMetricKeys);
    const columns: ColumnsType<Record<string, string>> = [
      { title: "时间", dataIndex: "__time_local__", width: 180, ellipsis: true },
      { title: "时间戳", dataIndex: "__timestamp__", width: 150, ellipsis: true },
      ...metricKeys.map((name) => ({ title: name, dataIndex: name, ellipsis: true })),
      { title: "Value", dataIndex: "__value__", width: 120 },
    ];
    return { columns, dataSource: rows };
  }

  if (rt === "matrix") {
    const rows: Record<string, string>[] = [];
    const keySet = new Set<string>();
    let k = 0;
    for (const item of result as Array<{ metric?: Record<string, string>; values?: [string, string][] }>) {
      const m = item.metric ?? {};
      const vals = item.values ?? [];
      for (const pair of vals) {
        const row: Record<string, string> = { key: String(k++) };
        for (const [mk, mv] of Object.entries(m)) {
          keySet.add(mk);
          row[mk] = mv;
        }
        row.__timestamp__ = pair?.[0] ?? "";
        row.__time_local__ = formatPromTimestampLocal(pair?.[0] ?? "");
        row.__value__ = pair?.[1] ?? "";
        keySet.add("__timestamp__");
        keySet.add("__time_local__");
        keySet.add("__value__");
        rows.push(row);
      }
    }
    const metricKeys = [...keySet]
      .filter((x) => x !== "__timestamp__" && x !== "__time_local__" && x !== "__value__")
      .sort(sortMetricKeys);
    const columns: ColumnsType<Record<string, string>> = [
      { title: "时间", dataIndex: "__time_local__", width: 180, ellipsis: true },
      { title: "时间戳", dataIndex: "__timestamp__", width: 150, ellipsis: true },
      ...metricKeys.map((name) => ({ title: name, dataIndex: name, ellipsis: true })),
      { title: "Value", dataIndex: "__value__", width: 120 },
    ];
    return { columns, dataSource: rows };
  }

  return null;
}

function formatPromScalarSummary(data: unknown): string | null {
  if (!data || typeof data !== "object") return null;
  const o = data as Record<string, unknown>;
  if (String(o.resultType) !== "string") return null;
  const r = o.result;
  if (Array.isArray(r) && r.length >= 2) return `结果值：${String(r[1])}（时间戳 ${r[0]}）`;
  return null;
}

/** 后端返回的 Prometheus JSON 可能为 { status, data:{ resultType, ... } }，表格解析取内层 data。 */
function unwrapPrometheusQueryData(body: unknown): unknown {
  if (!body || typeof body !== "object") return body;
  const o = body as Record<string, unknown>;
  if (o.data && typeof o.data === "object") {
    const d = o.data as Record<string, unknown>;
    if (typeof d.resultType === "string" || Array.isArray(d.result)) return o.data;
  }
  return body;
}

export function AlertMonitorPlatformPage() {
  const [searchParams, setSearchParams] = useSearchParams();
  const projectContextId = useMemo(() => {
    const raw = String(searchParams.get("project_id") || "").trim();
    if (!raw) return undefined;
    const n = Number(raw);
    if (!Number.isFinite(n) || n <= 0) return undefined;
    return n;
  }, [searchParams]);
  const tab: TabKey = useMemo(() => {
    const t = searchParams.get("tab");
    if (t === "config" || t === "silences" || t === "rules" || t === "promql") return t;
    return "datasources";
  }, [searchParams]);

  const configTab: AlertConfigTab = useMemo(() => {
    const c = searchParams.get("cfg");
    if (c === "history") return c;
    return "policies";
  }, [searchParams]);

  function setTab(key: TabKey) {
    setSearchParams(
      (prev) => {
        const p = new URLSearchParams(prev);
        if (key === "datasources") p.delete("tab");
        else p.set("tab", key);
        if (key !== "config") p.delete("cfg");
        return p;
      },
      { replace: true },
    );
  }

  function setConfigTab(key: AlertConfigTab) {
    setSearchParams(
      (prev) => {
        const p = new URLSearchParams(prev);
        p.set("tab", "config");
        if (key === "policies") p.delete("cfg");
        else p.set("cfg", key);
        return p;
      },
      { replace: true },
    );
  }

  function setProjectContext(projectID?: number) {
    setSearchParams(
      (prev) => {
        const p = new URLSearchParams(prev);
        if (projectID && Number.isFinite(projectID) && projectID > 0) p.set("project_id", String(projectID));
        else p.delete("project_id");
        return p;
      },
      { replace: true },
    );
  }

  const [dsList, setDsList] = useState<AlertDatasourceItem[]>([]);
  const [silenceList, setSilenceList] = useState<AlertSilenceItem[]>([]);
  const [ruleList, setRuleList] = useState<AlertMonitorRuleItem[]>([]);
  const [blockList, setBlockList] = useState<AlertDutyBlockItem[]>([]);
  const [dutyRuleId, setDutyRuleId] = useState<number | null>(null);
  const [dutyModalOpen, setDutyModalOpen] = useState(false);
  /** 规则值班弹窗：从其他规则复制班次时的来源规则 ID */
  const [copySourceRuleId, setCopySourceRuleId] = useState<number | undefined>();
  const [copyDutyLoading, setCopyDutyLoading] = useState(false);

  const [loading, setLoading] = useState(false);
  const [users, setUsers] = useState<Array<{ label: string; value: number }>>([]);
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [deptTree, setDeptTree] = useState<TreeSelectProps["treeData"]>([]);

  const [dsModalOpen, setDsModalOpen] = useState(false);
  const [dsCurrent, setDsCurrent] = useState<AlertDatasourceItem | null>(null);
  const [dsForm] = Form.useForm();
  const [dsSubmitting, setDsSubmitting] = useState(false);

  const [silModalOpen, setSilModalOpen] = useState(false);
  const [silCurrent, setSilCurrent] = useState<AlertSilenceItem | null>(null);
  const [silForm] = Form.useForm();
  const [silSubmitting, setSilSubmitting] = useState(false);

  const [ruleModalOpen, setRuleModalOpen] = useState(false);
  const [ruleCurrent, setRuleCurrent] = useState<AlertMonitorRuleItem | null>(null);
  const [ruleForm] = Form.useForm();
  const [ruleSubmitting, setRuleSubmitting] = useState(false);
  const [ruleLogic, setRuleLogic] = useState<RuleBuilderLogic>("and");
  const [ruleConditions, setRuleConditions] = useState<RuleBuilderCondition[]>([{ metric: "", comparator: ">", threshold: null }]);

  const [assignOpen, setAssignOpen] = useState(false);
  const [assignRuleId, setAssignRuleId] = useState<number | null>(null);
  const [assignForm] = Form.useForm();
  const [assignSubmitting, setAssignSubmitting] = useState(false);
  const assignUserIds = Form.useWatch("user_ids", assignForm) as number[] | undefined;
  const assignSyncedKeyRef = useRef("");
  const [assignProfileOriginal, setAssignProfileOriginal] = useState<{ email: string; department_id?: number } | null>(null);
  const [assignUsersHint, setAssignUsersHint] = useState("");

  const [blkModalOpen, setBlkModalOpen] = useState(false);
  const [blkCurrent, setBlkCurrent] = useState<AlertDutyBlockItem | null>(null);
  const [blkForm] = Form.useForm();
  const [blkSubmitting, setBlkSubmitting] = useState(false);
  const blkUserIds = Form.useWatch("user_ids", blkForm) as number[] | undefined;
  const dutySyncedKeyRef = useRef<string>("");
  const [dutyProfileOriginal, setDutyProfileOriginal] = useState<{ email: string; department_id?: number } | null>(null);
  const [dutyUsersHint, setDutyUsersHint] = useState<string>("");

  const alertSeverityOpts = useDictOptions("alert_severity");
  const dsUrlDictOpts = useDictOptions("alert_datasource_base_url");
  const dsBasicUserDictOpts = useDictOptions("alert_datasource_basic_user");
  const promqlLabelKeyOpts = useDictOptions("alert_promql_label_key");
  const thresholdUnitDictOpts = useDictOptions("alert_threshold_unit");
  const dsUrlAutoOpts = useMemo(
    () => dsUrlDictOpts.map((o) => ({ label: o.label, value: String(o.value) })),
    [dsUrlDictOpts],
  );
  const dsBasicUserAutoOpts = useMemo(
    () => dsBasicUserDictOpts.map((o) => ({ label: o.label, value: String(o.value) })),
    [dsBasicUserDictOpts],
  );
  const silMatcherNameOpts = useDictOptions("alert_silence_matcher_name");
  const ruleComparatorOptions = useMemo(
    () => [
      { label: "大于 (>)", value: ">" },
      { label: "大于等于 (>=)", value: ">=" },
      { label: "小于 (<)", value: "<" },
      { label: "小于等于 (<=)", value: "<=" },
      { label: "等于 (==)", value: "==" },
      { label: "不等于 (!=)", value: "!=" },
    ],
    [],
  );
  const ruleLogicOptions = useMemo(
    () => [
      { label: "AND（且）", value: "and" },
      { label: "OR（或）", value: "or" },
    ],
    [],
  );

  const [promDsId, setPromDsId] = useState<number | undefined>();
  const [promMode, setPromMode] = useState<"instant" | "range">("instant");
  const [promQuery, setPromQuery] = useState("up");
  const [promTime, setPromTime] = useState("");
  const [promStart, setPromStart] = useState("");
  const [promEnd, setPromEnd] = useState("");
  const [promStep, setPromStep] = useState("30s");
  const [promResult, setPromResult] = useState<string>("");
  const [promDataInner, setPromDataInner] = useState<unknown>(null);
  const [promViewMode, setPromViewMode] = useState<"table" | "json">("table");
  const [promLoading, setPromLoading] = useState(false);
  const [metricKeyword, setMetricKeyword] = useState("");
  const [metricLoading, setMetricLoading] = useState(false);
  const [metricOptions, setMetricOptions] = useState<string[]>([]);
  const [selectedMetric, setSelectedMetric] = useState("");
  const [metricLabelFilters, setMetricLabelFilters] = useState<MetricLabelFilter[]>([{ key: "instance", op: "=", value: "" }]);
  const [labelValueLoading, setLabelValueLoading] = useState(false);
  const [labelValueOptions, setLabelValueOptions] = useState<string[]>([]);
  const [selectedPromFunc, setSelectedPromFunc] = useState<string>("none");

  const [silNativeDsId, setSilNativeDsId] = useState<number | undefined>();
  const [nativeAlertsLoading, setNativeAlertsLoading] = useState(false);
  const [nativeAlertsRows, setNativeAlertsRows] = useState<PromNativeAlertRow[]>([]);
  const [selectedNativeAlertKeys, setSelectedNativeAlertKeys] = useState<string[]>([]);
  const [selectedSilenceIds, setSelectedSilenceIds] = useState<number[]>([]);
  const [quickSilenceOpen, setQuickSilenceOpen] = useState(false);
  const [quickSilenceSubmitting, setQuickSilenceSubmitting] = useState(false);
  const [quickSilenceTargets, setQuickSilenceTargets] = useState<QuickSilenceTarget[]>([]);
  /** 批量静默（从活跃告警勾选）时共用的说明，写入每条 alert_silences.comment */
  const [quickSilenceComment, setQuickSilenceComment] = useState("");
  const projectOptions = useMemo(() => projects.map((p) => ({ label: `${p.name} (${p.code})`, value: p.id })), [projects]);
  const activeProjectName = useMemo(() => {
    if (!projectContextId) return "";
    const p = projects.find((it) => it.id === projectContextId);
    return p ? `${p.name} (${p.code})` : `项目 ${projectContextId}`;
  }, [projects, projectContextId]);

  const promTableView = useMemo(() => buildPromTableView(promDataInner), [promDataInner]);
  const promScalarText = useMemo(() => formatPromScalarSummary(promDataInner), [promDataInner]);
  const ruleSeverityOptions = useMemo(() => {
    const s = ruleCurrent?.severity?.trim();
    const base = alertSeverityOpts;
    if (!s || base.some((o) => String(o.value) === s)) return base;
    return [...base, { label: `${s}（当前规则）`, value: s }];
  }, [alertSeverityOpts, ruleCurrent?.severity]);
  const commonLabelKeyOptions = useMemo(() => {
    const defaults = ["instance", "job", "cluster", "namespace", "pod", "service", "node", "severity", "alertname", "path", "device", "fstype", "mountpoint"];
    const merged = new Set<string>(defaults);
    promqlLabelKeyOpts.forEach((o) => merged.add(String(o.value || "").trim()));
    return Array.from(merged)
      .filter(Boolean)
      .sort((a, b) => a.localeCompare(b))
      .map((k) => ({ label: k, value: k }));
  }, [promqlLabelKeyOpts]);
  const thresholdUnitOptions = useMemo(() => {
    const defaults = [
      { label: "原始值", value: "raw" },
      { label: "百分比 (%)", value: "percent" },
      { label: "字节 (bytes)", value: "bytes" },
      { label: "毫秒 (ms)", value: "ms" },
      { label: "计数 (count)", value: "count" },
    ];
    const merged = [...defaults];
    thresholdUnitDictOpts.forEach((o) => {
      const v = String(o.value || "").trim();
      if (!v) return;
      if (!merged.some((it) => it.value === v)) {
        merged.push({ label: String(o.label || v), value: v });
      }
    });
    if (!merged.some((it) => it.value === "precent")) {
      merged.push({ label: "百分比（兼容旧拼写 precent）", value: "precent" });
    }
    return merged;
  }, [thresholdUnitDictOpts]);
  const thresholdUnit = Form.useWatch("threshold_unit", ruleForm) as string | undefined;
  const promFunctionTemplates = useMemo(
    () => [
      { key: "none", label: "不使用函数", template: "__METRIC__", desc: "直接使用指标与标签过滤，不包裹 Prometheus 函数。" },
      { key: "rate", label: "rate()", template: "rate(__METRIC__[5m])", desc: "计算窗口内每秒增长率，常用于 counter 指标。" },
      { key: "irate", label: "irate()", template: "irate(__METRIC__[5m])", desc: "基于最近两点计算瞬时速率，波动更灵敏。" },
      { key: "increase", label: "increase()", template: "increase(__METRIC__[5m])", desc: "计算窗口内增长总量。" },
      { key: "ceil", label: "ceil()", template: "ceil(__METRIC__)", desc: "向上取整。" },
      { key: "floor", label: "floor()", template: "floor(__METRIC__)", desc: "向下取整。" },
      { key: "round", label: "round()", template: "round(__METRIC__, 0.1)", desc: "按给定精度四舍五入。" },
      { key: "avg_over_time", label: "avg_over_time()", template: "avg_over_time(__METRIC__[5m])", desc: "时间窗口平均值，适合平滑抖动。" },
      { key: "max_over_time", label: "max_over_time()", template: "max_over_time(__METRIC__[5m])", desc: "时间窗口最大值。" },
      { key: "min_over_time", label: "min_over_time()", template: "min_over_time(__METRIC__[5m])", desc: "时间窗口最小值。" },
      { key: "sum_by", label: "sum by()", template: "sum by (instance) (__METRIC__)", desc: "按标签聚合求和。" },
      { key: "histogram_quantile", label: "histogram_quantile()", template: "histogram_quantile(0.95, sum by (le) (__METRIC__))", desc: "直方图分位数计算（如 P95）。" },
    ],
    [],
  );
  const selectedPromFuncMeta = useMemo(
    () => promFunctionTemplates.find((it) => it.key === selectedPromFunc) ?? promFunctionTemplates[0],
    [promFunctionTemplates, selectedPromFunc],
  );
  function parseRuleBuilderExpr(exprRaw: string): { conditions: RuleBuilderCondition[]; logic: RuleBuilderLogic } | null {
    const expr = String(exprRaw || "").trim();
    if (!expr) return null;
    const hasOr = /\s+or\s+/i.test(expr);
    const hasAnd = /\s+and\s+/i.test(expr);
    if (hasOr && hasAnd) return null;
    const logic: RuleBuilderLogic = hasOr ? "or" : "and";
    const parts = (hasOr ? expr.split(/\s+or\s+/i) : expr.split(/\s+and\s+/i)).map((p) => p.trim()).filter(Boolean);
    const parsed: RuleBuilderCondition[] = [];
    for (const p0 of parts) {
      const p = p0.replace(/^\((.*)\)$/, "$1").trim();
      const m = p.match(/^(.+?)\s*(>=|<=|==|!=|>|<)\s*(-?\d+(?:\.\d+)?)\s*$/);
      if (!m) return null;
      parsed.push({
        metric: String(m[1] || "").trim(),
        comparator: (m[2] as RuleComparator) || ">",
        threshold: Number(m[3]),
      });
    }
    if (!parsed.length) return null;
    return { conditions: parsed, logic };
  }

  function tryFillRuleBuilderFromExpr(exprRaw: string) {
    const parsed = parseRuleBuilderExpr(exprRaw);
    if (!parsed) {
      setRuleLogic("and");
      setRuleConditions([{ metric: "", comparator: ">", threshold: null }]);
      return;
    }
    setRuleLogic(parsed.logic);
    setRuleConditions(parsed.conditions);
  }

  function buildRuleExprByConditions(conditions: RuleBuilderCondition[], logic: RuleBuilderLogic): string {
    const valid = conditions
      .map((c) => ({
        metric: String(c.metric || "").trim(),
        comparator: c.comparator,
        threshold: c.threshold,
      }))
      .filter((c) => c.metric && c.threshold !== null && !Number.isNaN(c.threshold));
    if (!valid.length) return "";
    if (valid.length === 1) {
      return `${valid[0].metric} ${valid[0].comparator} ${valid[0].threshold}`;
    }
    const joiner = logic === "or" ? " or " : " and ";
    return valid.map((c) => `(${c.metric} ${c.comparator} ${c.threshold})`).join(joiner);
  }

  function applyRuleBuilderToExpr() {
    if (!ruleConditions.length) {
      message.warning("请至少添加一个条件");
      return;
    }
    if (ruleConditions.some((c) => !String(c.metric || "").trim() || c.threshold === null || Number.isNaN(c.threshold))) {
      message.warning("请完善每个条件的指标表达式和阈值");
      return;
    }
    ruleForm.setFieldValue("expr", buildRuleExprByConditions(ruleConditions, ruleLogic));
  }

  const fillAssignFromUserIds = useCallback(
    async (ids: number[] | undefined) => {
      if (!ids?.length) {
        assignForm.setFieldsValue({ department_ids: [], profile_email: undefined });
        setAssignProfileOriginal(null);
        setAssignUsersHint("");
        return;
      }
      try {
        const details = await Promise.all(ids.map((id) => getUser(id)));
        const deptSet = new Set<number>();
        details.forEach((u) => {
          if (u.department_id) deptSet.add(u.department_id);
        });
        assignForm.setFieldsValue({ department_ids: [...deptSet] });
        setAssignUsersHint(details.map((u) => `${u.nickname || u.username}：${u.email || "（无邮箱）"}`).join("；"));
        if (ids.length === 1) {
          const u = details[0];
          const em = (u.email ?? "").trim();
          assignForm.setFieldsValue({ profile_email: em });
          setAssignProfileOriginal({ email: em, department_id: u.department_id });
        } else {
          assignForm.setFieldsValue({ profile_email: undefined });
          setAssignProfileOriginal(null);
        }
      } catch {
        setAssignUsersHint("");
      }
    },
    [assignForm],
  );

  useEffect(() => {
    if (!assignOpen) {
      assignSyncedKeyRef.current = "";
      return;
    }
    const key = (assignUserIds ?? []).join(",");
    if (key === assignSyncedKeyRef.current) return;
    assignSyncedKeyRef.current = key;
    void fillAssignFromUserIds(assignUserIds);
  }, [assignOpen, assignUserIds, fillAssignFromUserIds]);

  const fillDutyFromUserIds = useCallback(
    async (ids: number[] | undefined) => {
      if (!ids?.length) {
        blkForm.setFieldsValue({ department_ids: [], profile_email: undefined });
        setDutyProfileOriginal(null);
        setDutyUsersHint("");
        return;
      }
      try {
        const details = await Promise.all(ids.map((id) => getUser(id)));
        const deptSet = new Set<number>();
        details.forEach((u) => {
          if (u.department_id) deptSet.add(u.department_id);
        });
        blkForm.setFieldsValue({ department_ids: [...deptSet] });
        setDutyUsersHint(details.map((u) => `${u.nickname || u.username}：${u.email || "（无邮箱）"}`).join("；"));
        if (ids.length === 1) {
          const u = details[0];
          const em = (u.email ?? "").trim();
          blkForm.setFieldsValue({ profile_email: em });
          setDutyProfileOriginal({ email: em, department_id: u.department_id });
        } else {
          blkForm.setFieldsValue({ profile_email: undefined });
          setDutyProfileOriginal(null);
        }
      } catch {
        setDutyUsersHint("");
      }
    },
    [blkForm],
  );

  useEffect(() => {
    if (!blkModalOpen) {
      dutySyncedKeyRef.current = "";
      return;
    }
    const key = (blkUserIds ?? []).join(",");
    if (key === dutySyncedKeyRef.current) return;
    dutySyncedKeyRef.current = key;
    void fillDutyFromUserIds(blkUserIds);
  }, [blkModalOpen, blkUserIds, fillDutyFromUserIds]);

  const loadDatasources = useCallback(async (projectID?: number) => {
    const r = await listAlertDatasources({ project_id: projectID, page: 1, page_size: 200 });
    setDsList(r.list ?? []);
    setPromDsId((prev) => prev ?? r.list?.[0]?.id);
  }, []);

  const loadSilences = useCallback(async () => {
    const r = await listAlertSilences({ page: 1, page_size: 200 });
    setSilenceList(r.list ?? []);
  }, []);

  const loadNativeSilAlerts = useCallback(async () => {
    if (!silNativeDsId) {
      message.warning("请先选择 Prometheus 数据源");
      return;
    }
    setNativeAlertsLoading(true);
    try {
      const raw = await promActiveAlerts(silNativeDsId);
      const rows = parsePrometheusActiveAlertsTable(raw);
      setNativeAlertsRows(rows);
      setSelectedNativeAlertKeys((prev) => prev.filter((k) => rows.some((r) => r.key === k)));
    } catch {
      setNativeAlertsRows([]);
      setSelectedNativeAlertKeys([]);
    } finally {
      setNativeAlertsLoading(false);
    }
  }, [silNativeDsId]);

  const loadRules = useCallback(async (projectID?: number) => {
    const r = await listAlertMonitorRules({ project_id: projectID, page: 1, page_size: 200 });
    setRuleList(r.list ?? []);
  }, []);
  useEffect(() => {
    void (async () => {
      try {
        const [tree, u, projRes] = await Promise.all([getDepartmentTree(), getUsers({ page: 1, page_size: 500 }), getProjects({ page: 1, page_size: 500 })]);
        setDeptTree(deptToTreeData(tree ?? []));
        setUsers(
          (u.list ?? []).map((it) => ({
            value: it.id,
            label: `${it.nickname || it.username} (${it.email || "-"})`,
          })),
        );
        setProjects(projRes.list ?? []);
      } catch {
        /* ignore */
      }
    })();
  }, []);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      setLoading(true);
      try {
        if (tab === "datasources") await loadDatasources(projectContextId);
        if (tab === "promql") await loadDatasources(projectContextId);
        if (tab === "silences") await Promise.all([loadSilences(), loadDatasources(projectContextId)]);
        if (tab === "rules") {
          await Promise.all([loadDatasources(projectContextId), loadRules(projectContextId)]);
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [tab, projectContextId, loadDatasources, loadSilences, loadRules]);

  useEffect(() => {
    if (tab !== "silences") return;
    if (silNativeDsId != null && dsList.some((d) => d.id === silNativeDsId)) return;
    const first = dsList.find((d) => d.enabled)?.id ?? dsList[0]?.id;
    setSilNativeDsId(first);
  }, [tab, dsList, silNativeDsId]);
  useEffect(() => {
    if (tab !== "promql") return;
    if (promDsId != null && dsList.some((d) => d.id === promDsId)) return;
    const first = dsList.find((d) => d.enabled)?.id ?? dsList[0]?.id;
    setPromDsId(first);
  }, [tab, dsList, promDsId]);

  async function runProm() {
    if (!promDsId) {
      message.warning("请选择数据源");
      return;
    }
    setPromLoading(true);
    setPromResult("");
    setPromDataInner(null);
    try {
      if (promMode === "instant") {
        const r = await promInstantQuery(promDsId, { query: promQuery, time: promTime.trim() || undefined });
        const outer = (r as { data?: unknown }).data ?? r;
        const inner = unwrapPrometheusQueryData(outer);
        setPromDataInner(inner);
        setPromResult(JSON.stringify(outer, null, 2));
      } else {
        const r = await promRangeQuery(promDsId, {
          query: promQuery,
          start: promStart.trim(),
          end: promEnd.trim(),
          step: promStep.trim() || "30s",
        });
        const outer = (r as { data?: unknown }).data ?? r;
        const inner = unwrapPrometheusQueryData(outer);
        setPromDataInner(inner);
        setPromResult(JSON.stringify(outer, null, 2));
      }
      setPromViewMode("table");
    } catch (e) {
      setPromResult(e instanceof Error ? e.message : String(e));
      setPromDataInner(null);
    } finally {
      setPromLoading(false);
    }
  }

  function fillPromTimeNow() {
    setPromTime(dayjs().toISOString());
  }

  function fillPromRangeLastHour() {
    const end = dayjs();
    const start = end.subtract(1, "hour");
    setPromStart(start.toISOString());
    setPromEnd(end.toISOString());
    setPromStep("30s");
  }

  const dsColumns = [
    { title: "ID", dataIndex: "id", width: 70 },
    { title: "项目", dataIndex: "project_name", width: 160, render: (v: string, r: AlertDatasourceItem) => v || String(r.project_id || "-") },
    { title: "名称", dataIndex: "name" },
    { title: "地址", dataIndex: "base_url", ellipsis: true },
    { title: "启用", dataIndex: "enabled", width: 80, render: (v: boolean) => (v ? <Tag color="green">是</Tag> : <Tag>否</Tag>) },
    {
      title: "操作",
      width: 160,
      render: (_: unknown, r: AlertDatasourceItem) => (
        <Space>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openDsEdit(r)}>
            编辑
          </Button>
          <Popconfirm title="删除数据源？" onConfirm={() => void removeDs(r.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  function openDsCreate() {
    setDsCurrent(null);
    dsForm.resetFields();
    const fallbackProjectID = projectContextId ?? projects[0]?.id;
    dsForm.setFieldsValue({ project_id: fallbackProjectID, type: "prometheus", skip_tls_verify: false, enabled: true });
    setDsModalOpen(true);
  }

  function openDsEdit(r: AlertDatasourceItem) {
    setDsCurrent(r);
    dsForm.setFieldsValue({
      project_id: r.project_id,
      name: r.name,
      type: r.type,
      base_url: r.base_url,
      basic_user: r.basic_user ?? "",
      skip_tls_verify: r.skip_tls_verify,
      enabled: r.enabled,
      remark: r.remark,
    });
    setDsModalOpen(true);
  }

  async function submitDs() {
    setDsSubmitting(true);
    try {
      const v = await dsForm.validateFields();
      if (dsCurrent) {
        await updateAlertDatasource(dsCurrent.id, v);
        message.success("已更新");
      } else {
        await createAlertDatasource(v);
        message.success("已创建");
      }
      setDsModalOpen(false);
      await loadDatasources();
    } finally {
      setDsSubmitting(false);
    }
  }

  async function removeDs(id: number) {
    await deleteAlertDatasource(id);
    message.success("已删除");
    await loadDatasources();
  }

  const nativeAlertsColumns: ColumnsType<PromNativeAlertRow> = useMemo(
    () => [
      { title: "告警名", dataIndex: "alertname", width: 160, ellipsis: true },
      {
        title: "状态",
        dataIndex: "state",
        width: 120,
        render: (v: string) => {
          const s = String(v || "").toLowerCase();
          const firing = s === "firing";
          const resolved = s === "resolved";
          return (
            <Space size={6}>
              <Badge status={firing ? "error" : resolved ? "success" : "default"} />
              <Typography.Text>{v || "-"}</Typography.Text>
            </Space>
          );
        },
      },
      { title: "Labels", dataIndex: "labelsShort", ellipsis: true },
      { title: "activeAt", dataIndex: "activeAt", width: 180, ellipsis: true },
      {
        title: "操作",
        width: 110,
        render: (_: unknown, r: PromNativeAlertRow) => (
          <Button type="link" size="small" onClick={() => openQuickSilence([r])}>
            静默
          </Button>
        ),
      },
    ],
    [],
  );

  const silColumns = [
    { title: "ID", dataIndex: "id", width: 70 },
    { title: "名称", dataIndex: "name" },
    {
      title: "说明",
      dataIndex: "comment",
      width: 140,
      ellipsis: true,
      render: (c: string) => (c && String(c).trim() ? c : "—"),
    },
    {
      title: "匹配摘要",
      key: "m",
      width: 200,
      ellipsis: true,
      render: (_: unknown, r: AlertSilenceItem) => {
        try {
          const arr = JSON.parse(r.matchers_json || "[]") as Array<{ name?: string; value?: string }>;
          if (!Array.isArray(arr)) return "—";
          return arr.map((x) => `${x.name ?? ""}=${x.value ?? ""}`).join(", ");
        } catch {
          return r.matchers_json?.slice(0, 80) ?? "—";
        }
      },
    },
    { title: "开始", dataIndex: "starts_at", width: 170, render: (t: string) => formatDateTime(t) },
    { title: "结束", dataIndex: "ends_at", width: 170, render: (t: string) => formatDateTime(t) },
    {
      title: "状态",
      key: "status",
      width: 100,
      render: (_: unknown, r: AlertSilenceItem) => {
        const expired = dayjs(r.ends_at).isBefore(dayjs());
        if (expired) return <Tag color="red">已过期</Tag>;
        return r.enabled ? <Tag color="green">启用</Tag> : <Tag>停用</Tag>;
      },
    },
    {
      title: "操作",
      width: 230,
      render: (_: unknown, r: AlertSilenceItem) => (
        <Space>
          <Button type="link" size="small" disabled={!r.enabled} onClick={() => void releaseSingleSilence(r)}>
            解除静默
          </Button>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openSilEdit(r)}>
            编辑
          </Button>
          <Popconfirm title="删除静默？" onConfirm={() => void removeSil(r.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  function openSilCreate() {
    setSilCurrent(null);
    silForm.resetFields();
    silForm.setFieldsValue({
      name: "",
      matchers: [{ name: "alertname", value: "", is_regex: false }],
      comment: "",
      enabled: true,
      starts_at: dayjs(),
      ends_at: dayjs().add(2, "hour"),
    });
    setSilModalOpen(true);
  }

  function toQuickSilenceTarget(row: PromNativeAlertRow): QuickSilenceTarget {
    const now = dayjs();
    const n = String(row.alertname || "").trim() || "未命名告警";
    return {
      key: row.key,
      name: `静默 ${n}`,
      labels: row.labels ?? {},
      startsAt: now,
      endsAt: now.add(2, "hour"),
    };
  }

  function openQuickSilence(rows: PromNativeAlertRow[]) {
    setQuickSilenceComment("");
    const targets = rows.map(toQuickSilenceTarget);
    if (!targets.length) {
      message.warning("请先选择需要静默的告警");
      return;
    }
    setQuickSilenceTargets(targets);
    setQuickSilenceOpen(true);
  }

  function buildMatchersByLabels(labels: Record<string, string>): SilenceMatcherForm[] {
    return Object.entries(labels ?? {})
      .map(([name, value]) => ({ name: String(name || "").trim(), value: String(value || "").trim(), is_regex: false }))
      .filter((m) => m.name && m.value);
  }

  async function submitQuickSilence() {
    if (!quickSilenceTargets.length) {
      setQuickSilenceOpen(false);
      return;
    }
    for (const it of quickSilenceTargets) {
      if (!it.endsAt.isAfter(it.startsAt)) {
        message.error(`「${it.name}」结束时间必须晚于开始时间`);
        return;
      }
    }
    setQuickSilenceSubmitting(true);
    try {
      const comment = quickSilenceComment.trim();
      const results = await Promise.allSettled(
        quickSilenceTargets.map((it) =>
          createAlertSilence({
            name: it.name,
            matchers_json: JSON.stringify(buildMatchersByLabels(it.labels)),
            comment,
            enabled: true,
            starts_at: it.startsAt.toISOString(),
            ends_at: it.endsAt.toISOString(),
          }),
        ),
      );
      const ok = results.filter((r) => r.status === "fulfilled").length;
      const fail = results.length - ok;
      if (ok > 0) message.success(`已创建 ${ok} 条静默`);
      if (fail > 0) message.warning(`${fail} 条静默创建失败，请重试`);
      setQuickSilenceOpen(false);
      await loadSilences();
    } finally {
      setQuickSilenceSubmitting(false);
    }
  }

  function openSilEdit(r: AlertSilenceItem) {
    setSilCurrent(r);
    silForm.setFieldsValue({
      name: r.name,
      matchers: parseSilenceMatchersForForm(r.matchers_json),
      comment: r.comment,
      enabled: r.enabled,
      starts_at: dayjs(r.starts_at),
      ends_at: dayjs(r.ends_at),
    });
    setSilModalOpen(true);
  }

  async function submitSil() {
    setSilSubmitting(true);
    try {
      const v = await silForm.validateFields();
      const rawMatchers = (v.matchers ?? []) as SilenceMatcherForm[];
      const matchers = rawMatchers
        .map((m) => ({
          name: String(m?.name ?? "").trim(),
          value: String(m?.value ?? "").trim(),
          is_regex: Boolean(m?.is_regex),
        }))
        .filter((m) => m.name !== "");
      if (matchers.length === 0) {
        message.error("至少添加一条匹配器，并填写名称（如 alertname）");
        return;
      }
      const payload = {
        name: v.name,
        matchers_json: JSON.stringify(matchers),
        comment: v.comment,
        enabled: v.enabled,
        starts_at: (v.starts_at as Dayjs).toISOString(),
        ends_at: (v.ends_at as Dayjs).toISOString(),
      };
      if (silCurrent) {
        await updateAlertSilence(silCurrent.id, payload);
        message.success("已更新");
      } else {
        await createAlertSilence(payload);
        message.success("已创建");
      }
      setSilModalOpen(false);
      await loadSilences();
    } finally {
      setSilSubmitting(false);
    }
  }

  async function removeSil(id: number) {
    await deleteAlertSilence(id);
    message.success("已删除");
    await loadSilences();
  }

  async function releaseSilenceNow(row: AlertSilenceItem) {
    await updateAlertSilence(row.id, {
      name: row.name,
      matchers_json: row.matchers_json,
      comment: row.comment ?? "",
      enabled: false,
      starts_at: row.starts_at,
      ends_at: row.ends_at,
    });
  }

  async function releaseSingleSilence(row: AlertSilenceItem) {
    await releaseSilenceNow(row);
    message.success("已解除静默");
    await loadSilences();
  }

  async function releaseSelectedSilences() {
    const rows = silenceList.filter((it) => selectedSilenceIds.includes(it.id) && it.enabled);
    if (!rows.length) {
      message.warning("请选择需要解除的启用静默");
      return;
    }
    const results = await Promise.allSettled(rows.map((r) => releaseSilenceNow(r)));
    const ok = results.filter((r) => r.status === "fulfilled").length;
    const fail = rows.length - ok;
    if (ok > 0) message.success(`已解除 ${ok} 条静默`);
    if (fail > 0) message.warning(`${fail} 条静默解除失败`);
    setSelectedSilenceIds([]);
    await loadSilences();
  }

  const ruleColumns = [
    { title: "ID", dataIndex: "id", width: 70 },
    { title: "项目", dataIndex: "project_name", width: 160, render: (v: string, r: AlertMonitorRuleItem) => v || (r.project_id ? String(r.project_id) : "—") },
    { title: "名称", dataIndex: "name", width: 160 },
    {
      title: "数据源",
      key: "ds",
      width: 200,
      render: (_: unknown, r: AlertMonitorRuleItem) => {
        const name = String(r.datasource_name || "").trim();
        if (name) return name;
        const ds = dsList.find((d) => d.id === r.datasource_id);
        return ds ? (ds.project_name ? `${ds.project_name} / ${ds.name}` : ds.name) : String(r.datasource_id);
      },
    },
    { title: "级别", dataIndex: "severity", width: 90 },
    { title: "for(s)", dataIndex: "for_seconds", width: 80 },
    { title: "间隔(s)", dataIndex: "eval_interval_seconds", width: 90 },
    { title: "启用", dataIndex: "enabled", width: 70, render: (v: boolean) => (v ? <Tag color="green">是</Tag> : <Tag>否</Tag>) },
    {
      title: "操作",
      width: 260,
      fixed: "right" as const,
      render: (_: unknown, r: AlertMonitorRuleItem) => (
        <Space wrap>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openRuleEdit(r)}>
            规则
          </Button>
          <Button type="link" size="small" icon={<TeamOutlined />} onClick={() => void openAssign(r.id)}>
            处理人
          </Button>
          <Button type="link" size="small" icon={<CalendarOutlined />} onClick={() => void openDuty(r.id)}>
            值班
          </Button>
          <Popconfirm title="删除规则？" onConfirm={() => void removeRule(r.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  function openRuleCreate() {
    setRuleCurrent(null);
    setRuleLogic("and");
    setRuleConditions([{ metric: "", comparator: ">", threshold: null }]);
    ruleForm.resetFields();
    const firstDs = dsList[0]?.id;
    ruleForm.setFieldsValue({
      datasource_id: firstDs,
      for_seconds: 0,
      eval_interval_seconds: 30,
      severity: "warning",
      threshold_unit: "percent",
      enabled: true,
    });
    setMetricKeyword("");
    setMetricOptions([]);
    setSelectedMetric("");
    setMetricLabelFilters([{ key: "instance", op: "=", value: "" }]);
    setSelectedPromFunc("none");
    setLabelValueOptions([]);
    setRuleModalOpen(true);
  }

  function openRuleEdit(r: AlertMonitorRuleItem) {
    setRuleCurrent(r);
    ruleForm.setFieldsValue({
      datasource_id: r.datasource_id,
      name: r.name,
      expr: r.expr,
      for_seconds: r.for_seconds,
      eval_interval_seconds: r.eval_interval_seconds,
      severity: r.severity,
      threshold_unit: r.threshold_unit || "raw",
      enabled: r.enabled,
    });
    tryFillRuleBuilderFromExpr(r.expr);
    setMetricKeyword("");
    setMetricOptions([]);
    const parsedByExpr = parseRuleBuilderExpr(r.expr);
    const selectorSrc = parsedByExpr?.conditions?.[0]?.metric || r.expr;
    const selector = parsePromSelectorExpr(selectorSrc);
    setSelectedMetric(selector?.metric || "");
    setMetricLabelFilters(selector?.filters || [{ key: "instance", op: "=", value: "" }]);
    const funcKey = detectPromFunctionKeyFromExpr(selectorSrc);
    setSelectedPromFunc(funcKey || "none");
    setLabelValueOptions([]);
    setRuleModalOpen(true);
  }

  async function loadMetricOptionsForRule() {
    const dsID = Number(ruleForm.getFieldValue("datasource_id"));
    if (!dsID) {
      message.warning("请先选择数据源");
      return;
    }
    const kw = String(metricKeyword || "").trim();
    const re = kw ? `.*${kw.replace(/\//g, "\\/")}.*` : ".+";
    const query = `topk(300, count by (__name__)({__name__=~"${re}"}))`;
    setMetricLoading(true);
    try {
      const r = await promInstantQuery(dsID, { query });
      const outer = (r as { data?: unknown }).data ?? r;
      const inner = unwrapPrometheusQueryData(outer) as { result?: Array<{ metric?: Record<string, string> }> };
      const names = Array.from(
        new Set(
          (inner?.result ?? [])
            .map((it) => String(it?.metric?.__name__ ?? "").trim())
            .filter(Boolean),
        ),
      ).sort((a, b) => a.localeCompare(b));
      setMetricOptions(names);
      if (names.length === 0) message.warning("未检索到指标，请调整关键字");
    } catch (e) {
      message.error(`加载指标失败：${e instanceof Error ? e.message : String(e)}`);
    } finally {
      setMetricLoading(false);
    }
  }

  async function loadLabelValuesForRule(idx: number) {
    const dsID = Number(ruleForm.getFieldValue("datasource_id"));
    if (!dsID) {
      message.warning("请先选择数据源");
      return;
    }
    const metric = String(selectedMetric || "").trim();
    if (!metric) {
      message.warning("请先选择指标");
      return;
    }
    const key = String(metricLabelFilters[idx]?.key || "").trim();
    if (!isValidPromLabelKey(key)) {
      message.warning("标签名不合法");
      return;
    }
    const selector = buildPromSelectorExpr(
      metric,
      metricLabelFilters.filter((_, i) => i !== idx),
    );
    const query = `topk(200, count by (${key}) (${selector}))`;
    setLabelValueLoading(true);
    try {
      const r = await promInstantQuery(dsID, { query });
      const outer = (r as { data?: unknown }).data ?? r;
      const inner = unwrapPrometheusQueryData(outer) as { result?: Array<{ metric?: Record<string, string> }> };
      const vals = Array.from(
        new Set(
          (inner?.result ?? [])
            .map((it) => String(it?.metric?.[key] ?? "").trim())
            .filter(Boolean),
        ),
      ).sort((a, b) => a.localeCompare(b));
      setLabelValueOptions(vals);
      if (!vals.length) message.warning("未检索到可用标签值");
    } catch (e) {
      message.error(`加载标签值失败：${e instanceof Error ? e.message : String(e)}`);
    } finally {
      setLabelValueLoading(false);
    }
  }

  function applyMetricSelectorToRuleExpr() {
    const metric = String(selectedMetric || "").trim();
    if (!metric) {
      message.warning("请先选择指标");
      return;
    }
    const selector = buildPromSelectorExpr(metric, metricLabelFilters);
    ruleForm.setFieldValue("expr", selector);
    setRuleConditions((prev) => {
      if (!prev.length) return [{ metric: selector, comparator: ">", threshold: null }];
      return prev.map((it, i) => (i === 0 ? { ...it, metric: selector } : it));
    });
    message.success("已生成并带入 PromQL");
  }

  function materializePromFunctionTemplate(raw: string): string {
    const selector = buildPromSelectorExpr(String(selectedMetric || "").trim(), metricLabelFilters);
    const baseMetric = selector || String(ruleConditions[0]?.metric || "").trim() || "your_metric";
    return raw.split("__METRIC__").join(baseMetric);
  }

  function buildMetricExprBySteps(): string {
    const selector = buildPromSelectorExpr(String(selectedMetric || "").trim(), metricLabelFilters);
    const baseMetric = selector || String(ruleConditions[0]?.metric || "").trim() || "";
    if (!baseMetric) return "";
    if (selectedPromFuncMeta.key === "none") return baseMetric;
    return selectedPromFuncMeta.template.split("__METRIC__").join(baseMetric);
  }

  function insertPromFunctionToExpr() {
    const nextExpr = materializePromFunctionTemplate(selectedPromFuncMeta.template);
    const prev = String(ruleForm.getFieldValue("expr") || "").trim();
    ruleForm.setFieldValue("expr", prev ? `${prev}\n${nextExpr}` : nextExpr);
  }

  function usePromFunctionAsConditionMetric() {
    const nextExpr = materializePromFunctionTemplate(selectedPromFuncMeta.template);
    setRuleConditions((prev) => {
      if (!prev.length) return [{ metric: nextExpr, comparator: ">", threshold: null }];
      return prev.map((it, i) => (i === 0 ? { ...it, metric: nextExpr } : it));
    });
  }

  function applyStepwisePromQL() {
    const metricExpr = buildMetricExprBySteps();
    if (!metricExpr) {
      message.warning("请先完成第1步：选择指标与标签过滤（或手填条件表达式）");
      return;
    }
    const nextConditions: RuleBuilderCondition[] =
      ruleConditions.length > 0
        ? ruleConditions.map((it, i) => (i === 0 ? { ...it, metric: metricExpr } : it))
        : [{ metric: metricExpr, comparator: ">" as RuleComparator, threshold: null as number | null }];
    if (nextConditions.some((c) => !String(c.metric || "").trim() || c.threshold === null || Number.isNaN(c.threshold))) {
      setRuleConditions(nextConditions);
      message.warning("请完成第3步：填写阈值后再生成最终 PromQL");
      return;
    }
    setRuleConditions(nextConditions);
    ruleForm.setFieldValue("expr", buildRuleExprByConditions(nextConditions, ruleLogic));
  }

  async function submitRule() {
    setRuleSubmitting(true);
    try {
      const v = await ruleForm.validateFields();
      const normalizedUnit = String(v.threshold_unit || "raw").trim().toLowerCase() === "precent" ? "percent" : v.threshold_unit;
      const payload = {
        ...v,
        threshold_unit: normalizedUnit,
        labels_json: ruleCurrent?.labels_json ?? "{}",
        annotations_json: ruleCurrent?.annotations_json ?? "{}",
      };
      if (ruleCurrent) {
        await updateAlertMonitorRule(ruleCurrent.id, payload);
        message.success("已更新");
      } else {
        await createAlertMonitorRule(payload);
        message.success("已创建");
      }
      setRuleModalOpen(false);
      await loadRules();
    } finally {
      setRuleSubmitting(false);
    }
  }

  async function removeRule(id: number) {
    await deleteAlertMonitorRule(id);
    message.success("已删除");
    await loadRules();
  }

  async function openAssign(ruleId: number) {
    assignSyncedKeyRef.current = "";
    setAssignRuleId(ruleId);
    assignForm.resetFields();
    try {
      const { list } = await getMonitorRuleAssignees(ruleId);
      const row = list?.[0];
      assignForm.setFieldsValue({
        user_ids: row ? parseUintArrayJSON(row.user_ids_json) : [],
        department_ids: row ? parseUintArrayJSON(row.department_ids_json) : [],
        notify_on_resolved: row?.notify_on_resolved ?? false,
        remark: row?.remark ?? "",
      });
    } catch {
      assignForm.setFieldsValue({ user_ids: [], department_ids: [], notify_on_resolved: false, remark: "" });
    }
    setAssignOpen(true);
  }

  async function submitAssign() {
    if (!assignRuleId) return;
    setAssignSubmitting(true);
    try {
      const v = await assignForm.validateFields();
      const userIds = (v.user_ids ?? []) as number[];
      const deptIds = (v.department_ids ?? []) as number[];
      if (userIds.length === 1 && assignProfileOriginal) {
        const patch: UserUpdatePayload = {};
        const emailNew = String(v.profile_email ?? "").trim();
        if (emailNew && emailNew !== String(assignProfileOriginal.email ?? "").trim()) {
          patch.email = emailNew;
        }
        if (deptIds.length === 1 && deptIds[0] !== assignProfileOriginal.department_id) {
          patch.department_id = deptIds[0];
        }
        if (Object.keys(patch).length) {
          try {
            await updateUser(userIds[0], patch);
            message.success("已同步更新用户资料中的邮箱或部门");
          } catch (e) {
            message.warning(`处理人已保存，但写回用户资料失败：${e instanceof Error ? e.message : String(e)}`);
          }
        }
      }
      await upsertMonitorRuleAssignees(assignRuleId, {
        user_ids_json: JSON.stringify(userIds),
        department_ids_json: JSON.stringify(deptIds),
        extra_emails_json: "[]",
        notify_on_resolved: v.notify_on_resolved,
        remark: v.remark ?? "",
      });
      message.success("处理人已保存（部门按子树展开到启用用户）");
      setAssignOpen(false);
    } finally {
      setAssignSubmitting(false);
    }
  }

  const copyDutyRuleOptions = useMemo(() => {
    if (!dutyRuleId) return [];
    return ruleList
      .filter((r) => r.id !== dutyRuleId)
      .map((r) => ({ label: r.name, value: r.id }));
  }, [ruleList, dutyRuleId]);

  async function openDuty(ruleId: number) {
    setDutyRuleId(ruleId);
    setCopySourceRuleId(undefined);
    setBlockList([]);
    try {
      const r = await listDutyBlocks({ monitor_rule_id: ruleId, page: 1, page_size: 500 });
      setBlockList(r.list ?? []);
    } catch {
      setBlockList([]);
    }
    setDutyModalOpen(true);
  }

  /** 将「来源规则」下的全部班次复制为当前规则的班次（新建记录，互不影响原规则）。 */
  async function copyDutyBlocksFromSelectedRule() {
    if (!dutyRuleId) {
      message.warning("未识别当前规则");
      return;
    }
    if (!copySourceRuleId || copySourceRuleId === dutyRuleId) {
      message.warning("请选择一条其他监控规则作为班次来源");
      return;
    }
    setCopyDutyLoading(true);
    try {
      const r = await listDutyBlocks({ monitor_rule_id: copySourceRuleId, page: 1, page_size: 500 });
      const src = r.list ?? [];
      if (src.length === 0) {
        message.info("所选规则下暂无班次，无法复制");
        return;
      }
      for (const b of src) {
        await createDutyBlock({
          monitor_rule_id: dutyRuleId,
          starts_at: b.starts_at,
          ends_at: b.ends_at,
          title: b.title,
          user_ids_json: b.user_ids_json || "[]",
          department_ids_json: b.department_ids_json || "[]",
          extra_emails_json: b.extra_emails_json && String(b.extra_emails_json).trim() ? b.extra_emails_json : "[]",
          remark: b.remark ?? "",
        });
      }
      message.success(`已复制 ${src.length} 条班次到当前规则`);
      const refreshed = await listDutyBlocks({ monitor_rule_id: dutyRuleId, page: 1, page_size: 500 });
      setBlockList(refreshed.list ?? []);
    } catch (e) {
      message.error(e instanceof Error ? e.message : String(e));
    } finally {
      setCopyDutyLoading(false);
    }
  }

  const blkColumns = [
    { title: "ID", dataIndex: "id", width: 70 },
    { title: "标题", dataIndex: "title", width: 120 },
    { title: "开始", dataIndex: "starts_at", width: 160, render: (t: string) => formatDateTime(t) },
    { title: "结束", dataIndex: "ends_at", width: 160, render: (t: string) => formatDateTime(t) },
    {
      title: "操作",
      width: 120,
      render: (_: unknown, r: AlertDutyBlockItem) => (
        <Space>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openBlkEdit(r)}>
            编辑
          </Button>
          <Popconfirm title="删除班次？" onConfirm={() => void removeBlk(r.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  function openBlkCreate() {
    if (!dutyRuleId) {
      message.warning("请先选择一条监控规则");
      return;
    }
    dutySyncedKeyRef.current = "";
    setBlkCurrent(null);
    blkForm.resetFields();
    blkForm.setFieldsValue({
      monitor_rule_id: dutyRuleId,
      range: [dayjs(), dayjs().add(8, "hour")],
      title: "",
      user_ids: [],
      department_ids: [],
      profile_email: undefined,
      remark: "",
    });
    setBlkModalOpen(true);
  }

  function openBlkEdit(r: AlertDutyBlockItem) {
    dutySyncedKeyRef.current = "";
    setBlkCurrent(r);
    blkForm.setFieldsValue({
      monitor_rule_id: r.monitor_rule_id,
      range: [dayjs(r.starts_at), dayjs(r.ends_at)],
      title: r.title,
      user_ids: parseUintArrayJSON(r.user_ids_json),
      department_ids: parseUintArrayJSON(r.department_ids_json),
      profile_email: undefined,
      remark: r.remark,
    });
    setBlkModalOpen(true);
  }

  async function submitBlk() {
    setBlkSubmitting(true);
    try {
      const v = await blkForm.validateFields();
      const range = v.range as [Dayjs, Dayjs];
      const monitorRuleID = Number(v.monitor_rule_id || dutyRuleId || 0);
      if (!monitorRuleID) {
        message.error("未识别到监控规则，请关闭侧栏后重新从规则行进入值班");
        return;
      }
      if (!Array.isArray(range) || range.length !== 2 || !range[0] || !range[1]) {
        message.error("请选择完整的起止时间");
        return;
      }
      if (!range[1].isAfter(range[0])) {
        message.error("结束时间必须晚于开始时间");
        return;
      }
      const userIds = (v.user_ids ?? []) as number[];
      const deptIds = (v.department_ids ?? []) as number[];
      if (userIds.length === 1 && dutyProfileOriginal) {
        const patch: UserUpdatePayload = {};
        const emailNew = String(v.profile_email ?? "").trim();
        if (emailNew && emailNew !== String(dutyProfileOriginal.email ?? "").trim()) {
          patch.email = emailNew;
        }
        if (deptIds.length === 1 && deptIds[0] !== dutyProfileOriginal.department_id) {
          patch.department_id = deptIds[0];
        }
        if (Object.keys(patch).length) {
          try {
            await updateUser(userIds[0], patch);
            message.success("已同步更新用户资料中的邮箱或部门");
          } catch (e) {
            message.warning(`班次已保存，但写回用户资料失败：${e instanceof Error ? e.message : String(e)}`);
          }
        }
      }
      const payload = {
        monitor_rule_id: monitorRuleID,
        starts_at: range[0].toISOString(),
        ends_at: range[1].toISOString(),
        title: v.title,
        user_ids_json: JSON.stringify(userIds),
        department_ids_json: JSON.stringify(deptIds),
        extra_emails_json: "[]",
        remark: v.remark ?? "",
      };
      if (blkCurrent) {
        await updateDutyBlock(blkCurrent.id, payload);
        message.success("已更新");
      } else {
        await createDutyBlock(payload);
        message.success("已创建");
      }
      setBlkModalOpen(false);
      if (dutyRuleId) {
        const r = await listDutyBlocks({ monitor_rule_id: dutyRuleId, page: 1, page_size: 500 });
        setBlockList(r.list ?? []);
      }
    } finally {
      setBlkSubmitting(false);
    }
  }

  async function removeBlk(id: number) {
    await deleteDutyBlock(id);
    message.success("已删除");
    if (dutyRuleId) {
      const r = await listDutyBlocks({ monitor_rule_id: dutyRuleId, page: 1, page_size: 500 });
      setBlockList(r.list ?? []);
    }
  }

  return (
    <Card
      className="table-card"
      title={
        <Space size={8}>
          <span>告警监控平台</span>
          {projectContextId ? <Tag color="default">当前项目：{activeProjectName}</Tag> : null}
        </Space>
      }
      loading={loading}
    >
      <Space style={{ marginBottom: 12 }} wrap>
        <Typography.Text type="secondary">全局项目上下文</Typography.Text>
        <Select
          style={{ minWidth: 280 }}
          allowClear
          value={projectContextId}
          onChange={(v) => setProjectContext(v)}
          options={projectOptions}
          placeholder="全部项目（可选）"
        />
      </Space>
      <Tabs
        activeKey={tab}
        onChange={(k) => setTab(k as TabKey)}
        items={[
          {
            key: "datasources",
            label: "数据源",
            children: (
              <Space direction="vertical" style={{ width: "100%" }} size="middle">
                <Space>
                  <Button type="primary" icon={<PlusOutlined />} onClick={openDsCreate}>
                    新建数据源
                  </Button>
                  <Button icon={<ReloadOutlined />} onClick={() => void loadDatasources(projectContextId)}>
                    刷新
                  </Button>
                </Space>
                <Table rowKey="id" columns={dsColumns} dataSource={dsList} pagination={false} scroll={{ x: 900 }} />
              </Space>
            ),
          },
          {
            key: "config",
            label: "策略与联调",
            children: (
              <Space direction="vertical" style={{ width: "100%" }} size="middle">
                <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
                  Webhook 入站、告警策略、历史记录与规则模板与左侧「数据源 / 监控规则 / 值班」共用同一告警链路：前者决定命中与通知策略，后者负责 Prometheus 侧规则与处理人、值班邮箱合并。
                </Typography.Paragraph>
                <Collapse
                  size="small"
                  items={[
                    {
                      key: "roles",
                      label: "功能边界：策略 / 规则模板 / 监控规则",
                      children: (
                        <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
                          <ul style={{ margin: 0, paddingLeft: 18 }}>
                            <li>
                              <strong>告警策略</strong>：仅作用于经 Alertmanager Webhook 进入平台的告警，按 match_labels 等筛选并绑定通道；与 Prometheus 页面里的「规则文件」无自动同步关系。
                            </li>
                            <li>
                              <strong>告警规则模板</strong>：可复用的 PromQL / labels / annotations 片段库，便于人工拷贝到 Prometheus 或本平台「监控规则」。
                            </li>
                            <li>
                              <strong>规则与值班绑定</strong>：平台定时向已登记数据源执行 PromQL，命中后走同一套通知与处理人逻辑；若与 Alertmanager 双发，请自行控制重复告警。
                            </li>
                          </ul>
                        </Typography.Paragraph>
                      ),
                    },
                  ]}
                  style={{ marginBottom: 8 }}
                />
                <AlertConfigCenterPanel embedded activeTab={configTab} onTabChange={setConfigTab} />
              </Space>
            ),
          },
          {
            key: "silences",
            label: "静默",
            children: (
              <Space direction="vertical" style={{ width: "100%" }} size="middle">
                <Alert
                  type="info"
                  showIcon
                  message="平台静默 ≠ Alertmanager 静默"
                  description={
                    <span>
                      下方「平台静默规则」在<strong>服务端处理 Webhook</strong>时按 matchers 与告警 labels 比对，命中则<strong>不再向通道发送</strong>（并可能落一条「silence suppressed」类记录）。不会调用 Alertmanager API
                      去创建静默；Prometheus / AM UI 里的告警状态不会因本列表而变化。若要与 AM 一致，请在 Alertmanager 侧配置 silences。
                    </span>
                  }
                />
                <Typography.Title level={5} style={{ margin: 0 }}>
                  Prometheus 活跃告警（只读快照）
                </Typography.Title>
                <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
                  与「历史告警记录」不同：此处直接查询已选数据源的 <Typography.Text code>/api/v1/alerts</Typography.Text>，用于对照 Prometheus UI 中 Firing 的条目是否已进 Webhook 链路。
                </Typography.Paragraph>
                <Space wrap>
                  <Select
                    style={{ minWidth: 240 }}
                    placeholder="数据源"
                    value={silNativeDsId}
                    onChange={(v) => setSilNativeDsId(v)}
                    options={dsList.map((d) => ({ label: d.project_name ? `${d.project_name} / ${d.name}` : d.name, value: d.id }))}
                  />
                  <Button type="primary" loading={nativeAlertsLoading} onClick={() => void loadNativeSilAlerts()}>
                    拉取活跃告警
                  </Button>
                  <Button
                    onClick={() => {
                      const rows = nativeAlertsRows.filter((r) => selectedNativeAlertKeys.includes(r.key));
                      openQuickSilence(rows);
                    }}
                    disabled={selectedNativeAlertKeys.length === 0}
                  >
                    批量静默
                  </Button>
                </Space>
                <Table
                  rowKey="key"
                  size="small"
                  loading={nativeAlertsLoading}
                  columns={nativeAlertsColumns}
                  dataSource={nativeAlertsRows}
                  rowSelection={{
                    type: "checkbox",
                    selectedRowKeys: selectedNativeAlertKeys,
                    onChange: (keys) => setSelectedNativeAlertKeys(keys.map((k) => String(k))),
                  }}
                  pagination={{ pageSize: 8 }}
                  locale={{ emptyText: "暂无数据，请选择数据源后点击「拉取活跃告警」" }}
                />
                <Typography.Title level={5} style={{ margin: 0 }}>
                  静默列表
                </Typography.Title>
                <Space>
                  <Button type="primary" onClick={() => void releaseSelectedSilences()} disabled={selectedSilenceIds.length === 0}>
                    批量解除静默
                  </Button>
                  <Button icon={<ReloadOutlined />} onClick={() => void loadSilences()}>
                    刷新
                  </Button>
                </Space>
                <Table
                  rowKey="id"
                  rowSelection={{
                    type: "checkbox",
                    selectedRowKeys: selectedSilenceIds,
                    onChange: (keys) => setSelectedSilenceIds(keys.map((k) => Number(k)).filter((n) => Number.isFinite(n))),
                  }}
                  columns={silColumns}
                  dataSource={silenceList}
                  pagination={false}
                  scroll={{ x: 960 }}
                />
              </Space>
            ),
          },
          {
            key: "rules",
            label: "规则与值班绑定",
            children: (
              <Space direction="vertical" style={{ width: "100%" }} size="middle">
                <Space>
                  <Button type="primary" icon={<PlusOutlined />} onClick={openRuleCreate}>
                    新建规则
                  </Button>
                  <Button icon={<ReloadOutlined />} onClick={() => void Promise.all([loadRules(projectContextId), loadDatasources(projectContextId)])}>
                    刷新
                  </Button>
                </Space>
                <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
                  规则级「处理人」与所选「值班表」当前班次通知邮箱会在告警 outgoing 中合并去重；部门选择为根部门子树全员。
                </Typography.Paragraph>
                <Alert
                  type="info"
                  showIcon
                  message="规则配置建议"
                  description={
                    <span>
                      建议先确认四项：1) <Typography.Text code>datasource</Typography.Text> 选正确集群；2){" "}
                      <Typography.Text code>severity</Typography.Text> 与策略匹配（critical/warning/info）；3){" "}
                      <Typography.Text code>for_seconds</Typography.Text> 防抖时长；4) <Typography.Text code>eval_interval_seconds</Typography.Text>{" "}
                      评估频率（常用 30s/60s）。
                    </span>
                  }
                />
                <Table rowKey="id" columns={ruleColumns} dataSource={ruleList} pagination={false} scroll={{ x: 1100 }} />
              </Space>
            ),
          },
          {
            key: "promql",
            label: "PromQL 查询",
            children: (
              <Space direction="vertical" style={{ width: "100%" }} size="middle">
                <Space wrap>
                  <Select
                    style={{ minWidth: 220 }}
                    placeholder="数据源"
                    value={promDsId}
                    onChange={(v) => setPromDsId(v)}
                    options={dsList.map((d) => ({ label: d.name, value: d.id }))}
                  />
                  <Radio.Group value={promMode} onChange={(e) => setPromMode(e.target.value)}>
                    <Radio.Button value="instant">即时</Radio.Button>
                    <Radio.Button value="range">范围</Radio.Button>
                  </Radio.Group>
                </Space>
                <Input.TextArea rows={4} value={promQuery} onChange={(e) => setPromQuery(e.target.value)} placeholder="PromQL" />
                {promMode === "instant" ? (
                  <Space wrap>
                    <Input
                      style={{ maxWidth: 420 }}
                      placeholder="评估时间（可选，RFC3339，例如 2026-04-18T13:30:00+08:00）"
                      value={promTime}
                      onChange={(e) => setPromTime(e.target.value)}
                    />
                    <Button onClick={fillPromTimeNow}>当前时间</Button>
                    <Button onClick={() => setPromTime("")}>清空</Button>
                  </Space>
                ) : (
                  <Space wrap>
                    <Input
                      style={{ width: 280 }}
                      placeholder="start RFC3339，如 2026-04-18T12:00:00+08:00"
                      value={promStart}
                      onChange={(e) => setPromStart(e.target.value)}
                    />
                    <Input
                      style={{ width: 280 }}
                      placeholder="end RFC3339，如 2026-04-18T13:00:00+08:00"
                      value={promEnd}
                      onChange={(e) => setPromEnd(e.target.value)}
                    />
                    <Input style={{ width: 100 }} placeholder="step" value={promStep} onChange={(e) => setPromStep(e.target.value)} />
                    <Button onClick={fillPromRangeLastHour}>最近1小时</Button>
                  </Space>
                )}
                <Typography.Text type="secondary">
                  说明：评估时间是“在哪个时刻执行这条 PromQL”。留空默认当前时间；范围查询需填写 start/end（RFC3339），step 可填 30s/1m/5m。
                </Typography.Text>
                <Button type="primary" loading={promLoading} onClick={() => void runProm()}>
                  执行
                </Button>
                <Segmented
                  value={promViewMode}
                  onChange={(v) => setPromViewMode(v as "table" | "json")}
                  options={[
                    { label: "表格结果", value: "table" },
                    { label: "JSON 原文", value: "json" },
                  ]}
                />
                {promViewMode === "json" ? (
                  <Input.TextArea rows={14} readOnly value={promResult} placeholder="查询结果 JSON" />
                ) : promTableView ? (
                  <Table
                    rowKey="key"
                    size="small"
                    bordered
                    pagination={{ pageSize: 20, showSizeChanger: true }}
                    scroll={{ x: "max-content", y: 420 }}
                    columns={promTableView.columns}
                    dataSource={promTableView.dataSource}
                  />
                ) : promScalarText ? (
                  <Typography.Paragraph>{promScalarText}</Typography.Paragraph>
                ) : (
                  <Typography.Paragraph type="secondary">
                    执行查询后在此展示与 Prometheus 页面类似的表格；当前返回类型可能为标量或空结果，可切换到「JSON 原文」查看。
                  </Typography.Paragraph>
                )}
              </Space>
            ),
          },
        ]}
      />

      <Drawer
        title={dsCurrent ? "编辑数据源" : "新建数据源"}
        placement="right"
        width={640}
        open={dsModalOpen}
        onClose={() => setDsModalOpen(false)}
        destroyOnClose
        styles={{ body: { paddingBottom: 24 } }}
        extra={
          <Space>
            <Button onClick={() => setDsModalOpen(false)}>取消</Button>
            <Button type="primary" loading={dsSubmitting} onClick={() => void submitDs()}>
              确定
            </Button>
          </Space>
        }
      >
        <Form form={dsForm} layout="vertical">
          <Form.Item name="project_id" label="所属项目" rules={[{ required: true, message: "请选择项目" }]}>
            <Select options={projectOptions} placeholder="请选择项目" />
          </Form.Item>
          <Form.Item name="name" label="名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="type" label="类型">
            <Input placeholder="prometheus" />
          </Form.Item>
          <Form.Item
            name="base_url"
            label="Base URL"
            rules={[{ required: true, message: "请输入或从下拉选择 Base URL" }]}
            extra={
              <Typography.Text type="secondary">
                可直接输入；亦可从下拉选字典项（类型 <Typography.Text code>alert_datasource_base_url</Typography.Text>，「值」存完整 URL）。
              </Typography.Text>
            }
          >
            <AutoComplete
              style={{ width: "100%" }}
              allowClear
              placeholder="输入 URL 或点击选择字典项"
              options={dsUrlAutoOpts}
              filterOption={(input, option) =>
                (option?.label ?? "").toString().toLowerCase().includes(input.toLowerCase()) ||
                (option?.value ?? "").toString().toLowerCase().includes(input.toLowerCase())
              }
            />
          </Form.Item>
          <Form.Item name="bearer_token" label="Bearer Token">
            <Input.Password placeholder="留空表示不改" />
          </Form.Item>
          <Form.Item
            name="basic_user"
            label="Basic 用户"
            extra={
              <Typography.Text type="secondary">
                可直接输入；亦可从下拉选字典项（<Typography.Text code>alert_datasource_basic_user</Typography.Text>）；密码勿入字典。
              </Typography.Text>
            }
          >
            <AutoComplete
              style={{ width: "100%" }}
              allowClear
              placeholder="输入用户名或从字典选择"
              options={dsBasicUserAutoOpts}
              filterOption={(input, option) =>
                (option?.label ?? "").toString().toLowerCase().includes(input.toLowerCase()) ||
                (option?.value ?? "").toString().toLowerCase().includes(input.toLowerCase())
              }
            />
          </Form.Item>
          <Form.Item name="basic_password" label="Basic 密码">
            <Input.Password placeholder="留空表示不改" />
          </Form.Item>
          <Form.Item name="skip_tls_verify" label="跳过 TLS 校验" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="enabled" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <Input />
          </Form.Item>
        </Form>
      </Drawer>

      <Drawer
        title={silCurrent ? "编辑静默" : "新建静默"}
        placement="right"
        width={720}
        open={silModalOpen}
        onClose={() => setSilModalOpen(false)}
        destroyOnClose
        styles={{ body: { paddingBottom: 24 } }}
        extra={
          <Space>
            <Button onClick={() => setSilModalOpen(false)}>取消</Button>
            <Button type="primary" loading={silSubmitting} onClick={() => void submitSil()}>
              确定
            </Button>
          </Space>
        }
      >
        <Form form={silForm} layout="vertical">
          <Form.Item name="name" label="名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Typography.Paragraph type="secondary" style={{ marginBottom: 8 }}>
            匹配器：名称通常选 <Typography.Text code>alertname</Typography.Text> / <Typography.Text code>cluster</Typography.Text> 等；值支持精确匹配；勾选「正则」时按 Alertmanager matcher 语义使用正则。
          </Typography.Paragraph>
          <Form.List name="matchers">
            {(fields, { add, remove }) => (
              <>
                {fields.map((field) => (
                  <Space key={field.key} align="baseline" style={{ display: "flex", marginBottom: 8 }} wrap>
                    <Form.Item
                      name={[field.name, "name"]}
                      rules={[{ required: true, message: "填写 label 名称" }]}
                      style={{ marginBottom: 0, minWidth: 200 }}
                    >
                      <AutoComplete
                        allowClear
                        placeholder="label 名（可输入或选字典）"
                        options={silMatcherNameOpts.map((o) => ({ label: o.label, value: String(o.value) }))}
                        filterOption={(input, option) =>
                          (option?.label ?? "").toString().toLowerCase().includes(input.toLowerCase()) ||
                          (option?.value ?? "").toString().toLowerCase().includes(input.toLowerCase())
                        }
                      />
                    </Form.Item>
                    <Form.Item name={[field.name, "value"]} style={{ marginBottom: 0, flex: 1, minWidth: 160 }}>
                      <Input placeholder="匹配值" />
                    </Form.Item>
                    <Form.Item
                      name={[field.name, "is_regex"]}
                      valuePropName="checked"
                      initialValue={false}
                      style={{ marginBottom: 0 }}
                    >
                      <Switch checkedChildren="正则" unCheckedChildren="精确" />
                    </Form.Item>
                    <MinusCircleOutlined onClick={() => remove(field.name)} />
                  </Space>
                ))}
                <Form.Item>
                  <Button type="dashed" onClick={() => add({ name: "alertname", value: "", is_regex: false })} block icon={<PlusOutlined />}>
                    添加匹配条件
                  </Button>
                </Form.Item>
              </>
            )}
          </Form.List>
          <Form.Item name="starts_at" label="开始时间" rules={[{ required: true }]}>
            <DatePicker showTime style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item name="ends_at" label="结束时间" rules={[{ required: true }]}>
            <DatePicker showTime style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item name="comment" label="说明">
            <Input />
          </Form.Item>
          <Form.Item name="enabled" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Drawer>

      <Drawer
        title="批量静默（可分别设置起止时间）"
        placement="right"
        width={1000}
        open={quickSilenceOpen}
        onClose={() => setQuickSilenceOpen(false)}
        destroyOnClose
        styles={{ body: { paddingBottom: 24 } }}
        extra={
          <Space>
            <Button onClick={() => setQuickSilenceOpen(false)}>取消</Button>
            <Button type="primary" loading={quickSilenceSubmitting} onClick={() => void submitQuickSilence()}>
              确定
            </Button>
          </Space>
        }
      >
        <Typography.Paragraph type="secondary" style={{ marginBottom: 8 }}>
          静默说明（可选）：将写入每条静默记录的 <Typography.Text code>comment</Typography.Text> 字段，便于审计。
        </Typography.Paragraph>
        <Input.TextArea
          rows={2}
          value={quickSilenceComment}
          onChange={(e) => setQuickSilenceComment(e.target.value)}
          placeholder="例如：发布窗口临时静默、误报告警排查中…"
          maxLength={512}
          showCount
          style={{ marginBottom: 12 }}
        />
        <Table
          rowKey="key"
          size="small"
          pagination={false}
          dataSource={quickSilenceTargets}
          scroll={{ x: 920 }}
          columns={[
            { title: "名称", dataIndex: "name", width: 200 },
            {
              title: "匹配器摘要",
              width: 360,
              ellipsis: true,
              render: (_: unknown, r: QuickSilenceTarget) =>
                Object.entries(r.labels || {})
                  .map(([k, v]) => `${k}=${v}`)
                  .join(", "),
            },
            {
              title: "开始",
              width: 170,
              render: (_: unknown, r: QuickSilenceTarget) => (
                <DatePicker
                  showTime
                  value={r.startsAt}
                  onChange={(v) =>
                    setQuickSilenceTargets((prev) => prev.map((it) => (it.key === r.key ? { ...it, startsAt: v ?? it.startsAt } : it)))
                  }
                />
              ),
            },
            {
              title: "结束",
              width: 170,
              render: (_: unknown, r: QuickSilenceTarget) => (
                <DatePicker
                  showTime
                  value={r.endsAt}
                  onChange={(v) =>
                    setQuickSilenceTargets((prev) => prev.map((it) => (it.key === r.key ? { ...it, endsAt: v ?? it.endsAt } : it)))
                  }
                />
              ),
            },
          ]}
        />
      </Drawer>

      <Drawer
        title={ruleCurrent ? "编辑监控规则" : "新建监控规则"}
        placement="right"
        width={920}
        open={ruleModalOpen}
        onClose={() => setRuleModalOpen(false)}
        destroyOnClose
        styles={{ body: { paddingBottom: 24 } }}
        extra={
          <Space>
            <Button onClick={() => setRuleModalOpen(false)}>取消</Button>
            <Button type="primary" loading={ruleSubmitting} onClick={() => void submitRule()}>
              确定
            </Button>
          </Space>
        }
      >
        <Form form={ruleForm} layout="vertical">
          <Form.Item name="datasource_id" label="数据源" rules={[{ required: true }]}>
            <Select
              options={(projectContextId ? dsList.filter((d) => d.project_id === projectContextId) : dsList).map((d) => ({
                label: d.project_name ? `${d.project_name} / ${d.name}` : d.name,
                value: d.id,
              }))}
            />
          </Form.Item>
          <Form.Item name="name" label="规则名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Card size="small" title="PromQL 辅助生成（推荐）" style={{ marginBottom: 12 }}>
            <Space direction="vertical" style={{ width: "100%" }} size={8}>
              <Space wrap>
                <Input
                  style={{ width: 260 }}
                  value={metricKeyword}
                  onChange={(e) => setMetricKeyword(e.target.value)}
                  placeholder="按指标名关键字检索，如 cpu/memory/http"
                />
                <Button loading={metricLoading} onClick={() => void loadMetricOptionsForRule()}>
                  拉取指标
                </Button>
                <Select
                  showSearch
                  style={{ minWidth: 320 }}
                  placeholder="选择指标名"
                  value={selectedMetric || undefined}
                  options={metricOptions.map((m) => ({ label: m, value: m }))}
                  onChange={(v) => setSelectedMetric(String(v || ""))}
                  filterOption={(input, option) => String(option?.value ?? "").toLowerCase().includes(input.toLowerCase())}
                />
              </Space>
              {metricLabelFilters.map((f, idx) => (
                <Space key={`metric-filter-${idx}`} wrap>
                  <Select
                    mode="tags"
                    style={{ width: 180 }}
                    value={f.key ? [f.key] : []}
                    placeholder="标签名"
                    options={commonLabelKeyOptions}
                    onChange={(v) =>
                      setMetricLabelFilters((prev) =>
                        prev.map((it, i) => {
                          if (i !== idx) return it;
                          const val = Array.isArray(v) ? String(v[0] || "") : "";
                          return { ...it, key: val };
                        }),
                      )
                    }
                  />
                  <Select
                    style={{ width: 110 }}
                    value={f.op}
                    options={[
                      { label: "等于 (=)", value: "=" },
                      { label: "不等于 (!=)", value: "!=" },
                      { label: "正则 (=~)", value: "=~" },
                      { label: "反正则 (!~)", value: "!~" },
                    ]}
                    onChange={(v) =>
                      setMetricLabelFilters((prev) => prev.map((it, i) => (i === idx ? { ...it, op: v as MetricLabelFilter["op"] } : it)))
                    }
                  />
                  <AutoComplete
                    style={{ width: 260 }}
                    value={f.value}
                    options={labelValueOptions.map((v) => ({ value: v }))}
                    onChange={(v) =>
                      setMetricLabelFilters((prev) => prev.map((it, i) => (i === idx ? { ...it, value: String(v || "") } : it)))
                    }
                    placeholder="标签值，可手填或拉取候选"
                  />
                  <Button loading={labelValueLoading} onClick={() => void loadLabelValuesForRule(idx)}>
                    拉取值
                  </Button>
                  <Button
                    danger
                    disabled={metricLabelFilters.length <= 1}
                    onClick={() => setMetricLabelFilters((prev) => prev.filter((_, i) => i !== idx))}
                  >
                    删除
                  </Button>
                </Space>
              ))}
              <Space wrap>
                <Button onClick={() => setMetricLabelFilters((prev) => [...prev, { key: "", op: "=", value: "" }])}>新增标签过滤</Button>
                <Button type="primary" onClick={applyMetricSelectorToRuleExpr}>
                  生成并带入 PromQL
                </Button>
              </Space>
              <Typography.Text type="secondary">
                先选指标，再按标签过滤，最后一键带入到上方 PromQL；不会覆盖你后续用“条件构建器”生成的比较表达式。
              </Typography.Text>
              <Card size="small" title="Prometheus 函数助手（内置）">
                <Space direction="vertical" style={{ width: "100%" }} size={8}>
                  <Space wrap>
                    <Select
                      style={{ minWidth: 280 }}
                      value={selectedPromFunc}
                      options={promFunctionTemplates.map((it) => ({ label: it.label, value: it.key }))}
                      onChange={(v) => setSelectedPromFunc(String(v || "rate"))}
                    />
                    <Button onClick={insertPromFunctionToExpr}>插入到 PromQL</Button>
                    <Button onClick={usePromFunctionAsConditionMetric}>带入条件构造器首条指标</Button>
                  </Space>
                  <Typography.Text type="secondary">
                    {selectedPromFuncMeta.desc}
                    <br />
                    模板：<Typography.Text code>{selectedPromFuncMeta.template}</Typography.Text>
                    <br />
                    推荐顺序：第1步标签过滤 {"->"} 第2步函数（可选） {"->"} 第3步阈值比较。
                  </Typography.Text>
                </Space>
              </Card>
            </Space>
          </Card>
          <Form.Item name="expr" label="PromQL" rules={[{ required: true }]}>
            <Input.TextArea rows={4} />
          </Form.Item>
          <Card size="small" title="条件构建器（可选）" style={{ marginBottom: 12 }}>
            <Space direction="vertical" style={{ width: "100%" }} size={8}>
              <Space wrap>
                <Typography.Text type="secondary">组合逻辑</Typography.Text>
                <Select style={{ width: 180 }} value={ruleLogic} options={ruleLogicOptions} onChange={(v) => setRuleLogic(v as RuleBuilderLogic)} />
              </Space>
              {ruleConditions.map((cond, idx) => (
                <Space key={`rule-cond-${idx}`} wrap style={{ width: "100%" }}>
                  <Input
                    style={{ minWidth: 320 }}
                    value={cond.metric}
                    onChange={(e) =>
                      setRuleConditions((prev) => prev.map((it, i) => (i === idx ? { ...it, metric: e.target.value } : it)))
                    }
                    placeholder="指标表达式，如 rate(http_requests_total[5m])"
                  />
                  <Select
                    style={{ width: 160 }}
                    value={cond.comparator}
                    options={ruleComparatorOptions}
                    onChange={(v) =>
                      setRuleConditions((prev) => prev.map((it, i) => (i === idx ? { ...it, comparator: v as RuleComparator } : it)))
                    }
                  />
                  <InputNumber
                    style={{ width: 160 }}
                    value={cond.threshold}
                    onChange={(v) =>
                      setRuleConditions((prev) => prev.map((it, i) => (i === idx ? { ...it, threshold: v ?? null } : it)))
                    }
                    placeholder="阈值"
                  />
                  <Tag>{thresholdUnit || "raw"}</Tag>
                  <Button
                    danger
                    disabled={ruleConditions.length <= 1}
                    onClick={() => setRuleConditions((prev) => prev.filter((_, i) => i !== idx))}
                  >
                    删除条件
                  </Button>
                </Space>
              ))}
              <Space wrap>
                <Button onClick={() => setRuleConditions((prev) => [...prev, { metric: "", comparator: ">", threshold: null }])}>新增条件</Button>
                <Button type="primary" onClick={applyRuleBuilderToExpr}>
                  生成 PromQL
                </Button>
                <Button onClick={applyStepwisePromQL}>按步骤一键生成（推荐）</Button>
              </Space>
            </Space>
          </Card>
          <Form.Item name="for_seconds" label="持续满足秒数 (for)">
            <InputNumber min={0} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item name="threshold_unit" label="阈值单位">
            <Select options={thresholdUnitOptions} />
          </Form.Item>
          <Form.Item name="eval_interval_seconds" label="评估间隔秒">
            <InputNumber min={5} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item name="severity" label="级别" rules={[{ required: true, message: "请选择级别" }]}>
            <Select placeholder="选择级别" options={ruleSeverityOptions} />
          </Form.Item>
          <Form.Item name="enabled" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
        </Form>
      </Drawer>

      <Drawer
        title="规则处理人"
        placement="right"
        width={640}
        open={assignOpen}
        onClose={() => setAssignOpen(false)}
        destroyOnClose
        styles={{ body: { paddingBottom: 24 } }}
        extra={
          <Space>
            <Button onClick={() => setAssignOpen(false)}>取消</Button>
            <Button type="primary" loading={assignSubmitting} onClick={() => void submitAssign()}>
              保存
            </Button>
          </Space>
        }
      >
        <Typography.Paragraph type="secondary">部门为根节点时包含子部门内全部启用用户。</Typography.Paragraph>
        <Form form={assignForm} layout="vertical">
          <Form.Item name="user_ids" label="用户">
            <Select mode="multiple" options={users} optionFilterProp="label" placeholder="选择用户" />
          </Form.Item>
          {assignUsersHint ? (
            <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
              用户资料邮箱：{assignUsersHint}
            </Typography.Paragraph>
          ) : null}
          <Form.Item name="department_ids" label="部门（子树）">
            <TreeSelect treeData={deptTree} treeCheckable showSearch allowClear treeDefaultExpandAll style={{ width: "100%" }} placeholder="随用户带出，可改" />
          </Form.Item>
          {assignUserIds?.length === 1 ? (
            <Form.Item name="profile_email" label="邮箱（可改，保存时写回该用户资料）">
              <Input placeholder="无邮箱时请填写，保存后写入用户表" />
            </Form.Item>
          ) : (
            <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
              多人时通知按各用户资料邮箱合并；仅选择一名用户时可在此编辑邮箱并写回用户资料。额外邮箱字段已废弃，统一走用户资料。
            </Typography.Paragraph>
          )}
          <Form.Item name="notify_on_resolved" label="恢复时通知" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <Input />
          </Form.Item>
        </Form>
      </Drawer>

      <Drawer
        title="规则值班（按时间段生效）"
        placement="right"
        width={720}
        open={dutyModalOpen}
        onClose={() => setDutyModalOpen(false)}
        destroyOnClose
        styles={{ body: { paddingBottom: 24 } }}
        extra={
          <Button type="primary" onClick={() => setDutyModalOpen(false)}>
            关闭
          </Button>
        }
      >
        <Space direction="vertical" style={{ width: "100%" }} size="small">
          <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
            当前规则 ID：{dutyRuleId ?? "-"}。班次命中时会与“处理人”邮箱合并去重后写入 <Typography.Text code>assignee_emails</Typography.Text>。
          </Typography.Paragraph>
          <Typography.Paragraph type="secondary" style={{ marginBottom: 0, fontSize: 12 }}>
            若其他规则上已配好相同时间段与值班人，可从该规则「复制班次」到本规则（会新增独立记录，两条规则各自生效、互不影响）。
          </Typography.Paragraph>
          <Space wrap align="start">
            <Select
              allowClear
              showSearch
              placeholder="选择已有班次的来源规则"
              style={{ minWidth: 280 }}
              options={copyDutyRuleOptions}
              value={copySourceRuleId}
              onChange={(v) => setCopySourceRuleId(v)}
              optionFilterProp="label"
              disabled={!dutyRuleId || copyDutyRuleOptions.length === 0}
            />
            <Button
              loading={copyDutyLoading}
              disabled={!dutyRuleId || !copySourceRuleId}
              onClick={() => void copyDutyBlocksFromSelectedRule()}
            >
              复制班次到当前规则
            </Button>
          </Space>
          <Button type="primary" icon={<PlusOutlined />} disabled={!dutyRuleId} onClick={openBlkCreate}>
            新建班次
          </Button>
          <Table rowKey="id" columns={blkColumns} dataSource={blockList} pagination={false} size="small" scroll={{ x: 720 }} />
        </Space>
      </Drawer>

      <Drawer
        title={blkCurrent ? "编辑班次" : "新建班次"}
        placement="right"
        width={640}
        open={blkModalOpen}
        onClose={() => setBlkModalOpen(false)}
        destroyOnClose
        styles={{ body: { paddingBottom: 24 } }}
        extra={
          <Space>
            <Button onClick={() => setBlkModalOpen(false)}>取消</Button>
            <Button type="primary" loading={blkSubmitting} onClick={() => void submitBlk()}>
              确定
            </Button>
          </Space>
        }
      >
        <Form form={blkForm} layout="vertical">
          <Form.Item name="monitor_rule_id" hidden>
            <InputNumber />
          </Form.Item>
          <Form.Item name="range" label="起止时间" rules={[{ required: true }]}>
            <DatePicker.RangePicker showTime={{ format: "HH:mm" }} format="YYYY-MM-DD HH:mm" style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item name="title" label="标题">
            <Input />
          </Form.Item>
          <Form.Item name="user_ids" label="用户">
            <Select mode="multiple" options={users} optionFilterProp="label" placeholder="选择值班人员" />
          </Form.Item>
          {dutyUsersHint ? (
            <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
              用户资料邮箱：{dutyUsersHint}
            </Typography.Paragraph>
          ) : null}
          <Form.Item name="department_ids" label="部门（子树）">
            <TreeSelect treeData={deptTree} treeCheckable showSearch allowClear treeDefaultExpandAll style={{ width: "100%" }} placeholder="随用户带出，可改" />
          </Form.Item>
          {blkUserIds?.length === 1 ? (
            <Form.Item name="profile_email" label="邮箱（可改，保存班次时写回该用户资料）">
              <Input placeholder="无邮箱时请填写，保存后写入用户表" />
            </Form.Item>
          ) : (
            <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
              多人值班时通知仍按各用户资料邮箱合并；仅选择一名用户时可在此编辑邮箱并写回用户资料。
            </Typography.Paragraph>
          )}
          <Form.Item name="remark" label="备注">
            <Input />
          </Form.Item>
        </Form>
      </Drawer>
    </Card>
  );
}
