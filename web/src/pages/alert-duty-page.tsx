import { DeleteOutlined, EditOutlined, PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import type { TreeSelectProps } from "antd";
import { Button, Card, DatePicker, Form, Input, Modal, Popconfirm, Segmented, Select, Space, Table, Tag, TreeSelect, Typography, message } from "antd";
import dayjs from "dayjs";
import { useEffect, useMemo, useState } from "react";
import { getDepartmentTree } from "../services/departments";
import {
  createDutyBlock,
  deleteDutyBlock,
  listAlertMonitorRules,
  listDutyBlocks,
  type AlertDutyBlockItem,
  type AlertMonitorRuleItem,
  updateDutyBlock,
} from "../services/alert-platform";
import { getProjects, type ProjectItem } from "../services/projects";
import { getUsers } from "../services/users";
import { formatDateTime } from "../utils/format";

export function AlertDutyPage() {
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [rules, setRules] = useState<AlertMonitorRuleItem[]>([]);
  const [ruleId, setRuleId] = useState<number>();
  const [list, setList] = useState<AlertDutyBlockItem[]>([]);
  const [viewMode, setViewMode] = useState<"day" | "week">("week");
  const [anchorDate, setAnchorDate] = useState(dayjs());
  const [users, setUsers] = useState<Array<{ label: string; value: number }>>([]);
  const [deptTree, setDeptTree] = useState<TreeSelectProps["treeData"]>([]);
  const [loading, setLoading] = useState(false);
  const [open, setOpen] = useState(false);
  const [saving, setSaving] = useState(false);
  const [current, setCurrent] = useState<AlertDutyBlockItem | null>(null);
  const [form] = Form.useForm();

  const projectOptions = useMemo(() => projects.map((p) => ({ label: `${p.name} (${p.code})`, value: p.id })), [projects]);
  const ruleOptions = useMemo(() => rules.map((r) => ({ label: r.name, value: r.id })), [rules]);
  const ruleNameMap = useMemo(() => {
    const m = new Map<number, string>();
    rules.forEach((r) => m.set(r.id, r.name));
    return m;
  }, [rules]);
  const userNameMap = useMemo(() => {
    const m = new Map<number, string>();
    users.forEach((u) => m.set(u.value, String(u.label || "")));
    return m;
  }, [users]);
  const deptNameMap = useMemo(() => {
    const m = new Map<number, string>();
    const walk = (nodes?: TreeSelectProps["treeData"]) => {
      (nodes || []).forEach((n) => {
        const id = Number((n as { value?: unknown }).value || 0);
        const title = String((n as { title?: unknown }).title || "");
        if (id) m.set(id, title);
        if ((n as { children?: TreeSelectProps["treeData"] }).children?.length) {
          walk((n as { children?: TreeSelectProps["treeData"] }).children);
        }
      });
    };
    walk(deptTree);
    return m;
  }, [deptTree]);
  const rangeStart = useMemo(() => (viewMode === "day" ? anchorDate.startOf("day") : anchorDate.startOf("week")), [anchorDate, viewMode]);
  const rangeEnd = useMemo(() => (viewMode === "day" ? rangeStart.add(1, "day") : rangeStart.add(1, "week")), [rangeStart, viewMode]);
  const totalRangeMs = Math.max(1, rangeEnd.valueOf() - rangeStart.valueOf());
  const timeAxisTicks = useMemo(() => {
    if (viewMode === "day") {
      return Array.from({ length: 25 }).map((_, i) => rangeStart.add(i, "hour"));
    }
    return Array.from({ length: 8 }).map((_, i) => rangeStart.add(i, "day"));
  }, [rangeStart, viewMode]);
  const nowRatio = (() => {
    const now = dayjs().valueOf();
    if (now <= rangeStart.valueOf()) return 0;
    if (now >= rangeEnd.valueOf()) return 100;
    return ((now - rangeStart.valueOf()) / totalRangeMs) * 100;
  })();

  const groupedBars = useMemo(() => {
    const m = new Map<number, AlertDutyBlockItem[]>();
    for (const item of list) {
      const s = dayjs(item.starts_at).valueOf();
      const e = dayjs(item.ends_at).valueOf();
      if (e <= rangeStart.valueOf() || s >= rangeEnd.valueOf()) continue;
      if (!m.has(item.monitor_rule_id)) m.set(item.monitor_rule_id, []);
      m.get(item.monitor_rule_id)!.push(item);
    }
    return Array.from(m.entries()).map(([rid, blocks]) => ({ rid, blocks: blocks.sort((a, b) => dayjs(a.starts_at).valueOf() - dayjs(b.starts_at).valueOf()) }));
  }, [list, rangeEnd, rangeStart]);

  async function loadProjects() {
    const res = await getProjects({ page: 1, page_size: 500 });
    setProjects(res.list ?? []);
  }

  async function loadPeopleMeta() {
    const [tree, userRes] = await Promise.all([getDepartmentTree(), getUsers({ page: 1, page_size: 500 })]);
    const toTree = (nodes: Array<{ id: number; name: string; children?: unknown[] }>): TreeSelectProps["treeData"] =>
      nodes.map((n) => ({
        title: n.name,
        value: n.id,
        key: n.id,
        children: toTree((n.children as Array<{ id: number; name: string; children?: unknown[] }>) || []),
      }));
    setDeptTree(toTree((tree || []) as Array<{ id: number; name: string; children?: unknown[] }>));
    setUsers((userRes.list || []).map((u) => ({ value: u.id, label: `${u.nickname || u.username}${u.email ? ` (${u.email})` : ""}` })));
  }

  async function loadRules(nextProjectID?: number) {
    const useProjectID = nextProjectID ?? projectId;
    const res = await listAlertMonitorRules({ page: 1, page_size: 500, project_id: useProjectID });
    setRules(res.list ?? []);
    if (res.list?.length) {
      if (!res.list.find((r) => r.id === ruleId)) {
        setRuleId(res.list[0].id);
      }
    } else {
      setRuleId(undefined);
    }
  }

  async function loadBlocks(nextRuleID?: number, nextProjectID?: number) {
    setLoading(true);
    try {
      const useRule = nextRuleID ?? ruleId;
      const useProject = nextProjectID ?? projectId;
      const res = await listDutyBlocks({ monitor_rule_id: useRule, project_id: useProject, page: 1, page_size: 1000 });
      setList(res.list ?? []);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void loadProjects();
    void loadRules();
    void loadPeopleMeta();
  }, []);

  useEffect(() => {
    void loadRules(projectId);
    void loadBlocks(undefined, projectId);
  }, [projectId]);

  useEffect(() => {
    void loadBlocks(ruleId, projectId);
  }, [ruleId]);

  function openCreate() {
    setCurrent(null);
    form.setFieldsValue({
      monitor_rule_ids: ruleId ? [ruleId] : [],
      title: "",
      range: [dayjs().startOf("hour"), dayjs().add(8, "hour").startOf("hour")],
      user_ids: [],
      department_ids: [],
      extra_emails: [],
      remark: "",
    });
    setOpen(true);
  }

  function openEdit(row: AlertDutyBlockItem) {
    setCurrent(row);
    form.setFieldsValue({
      monitor_rule_ids: row.monitor_rule_id,
      title: row.title || "",
      range: [dayjs(row.starts_at), dayjs(row.ends_at)],
      user_ids: parseUintArrayJSON(row.user_ids_json),
      department_ids: parseUintArrayJSON(row.department_ids_json),
      extra_emails: parseStringArrayJSON(row.extra_emails_json),
      remark: row.remark || "",
    });
    setOpen(true);
  }

  async function submit() {
    const v = await form.validateFields();
    const range = v.range as dayjs.Dayjs[];
    if (!range?.[0] || !range?.[1]) {
      message.error("请选择值班时间范围");
      return;
    }
    const ruleRaw = v.monitor_rule_ids;
    const ruleIDs = Array.isArray(ruleRaw)
      ? ruleRaw.map((it: unknown) => Number(it)).filter((id: number) => id > 0)
      : [Number(ruleRaw || 0)].filter((id: number) => id > 0);
    if (!ruleIDs.length) {
      message.error("请至少选择一条监控规则");
      return;
    }
    const payloadBase = {
      starts_at: range[0].toISOString(),
      ends_at: range[1].toISOString(),
      title: String(v.title || "").trim(),
      user_ids_json: JSON.stringify(Array.isArray(v.user_ids) ? v.user_ids.map((it: unknown) => Number(it)).filter((n: number) => n > 0) : []),
      department_ids_json: JSON.stringify(
        Array.isArray(v.department_ids) ? v.department_ids.map((it: unknown) => Number(it)).filter((n: number) => n > 0) : [],
      ),
      extra_emails_json: JSON.stringify(
        Array.isArray(v.extra_emails)
          ? v.extra_emails.map((it: unknown) => String(it || "").trim()).filter(Boolean)
          : [],
      ),
      remark: String(v.remark || "").trim(),
    };
    setSaving(true);
    try {
      if (current) {
        await updateDutyBlock(current.id, { ...payloadBase, monitor_rule_id: ruleIDs[0] });
        message.success("值班班次已更新");
      } else {
        await Promise.all(ruleIDs.map((rid: number) => createDutyBlock({ ...payloadBase, monitor_rule_id: rid })));
        message.success(`已创建 ${ruleIDs.length} 条值班班次`);
      }
      setOpen(false);
      await loadBlocks(ruleId, projectId);
    } finally {
      setSaving(false);
    }
  }

  return (
    <Card
      className="table-card"
      title="值班总览"
      extra={
        <Space>
          <Select allowClear style={{ width: 240 }} options={projectOptions} value={projectId} onChange={setProjectId} placeholder="项目维度（可选）" />
          <Select allowClear style={{ width: 320 }} options={ruleOptions} value={ruleId} onChange={setRuleId} placeholder="规则维度（可选）" />
          <Segmented
            value={viewMode}
            options={[
              { label: "按天", value: "day" },
              { label: "按周", value: "week" },
            ]}
            onChange={(v) => setViewMode(v as "day" | "week")}
          />
          <DatePicker value={anchorDate} onChange={(v) => setAnchorDate(v || dayjs())} />
          <Button icon={<ReloadOutlined />} onClick={() => void loadBlocks()}>
            刷新
          </Button>
          <Button onClick={() => (window.location.href = "/alert-monitor-platform?tab=rules")}>去配置规则</Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
            新增值班
          </Button>
        </Space>
      }
    >
      <div style={{ width: "100%", overflow: "hidden" }}>
      <Card
        size="small"
        title={`班次甘特视图（${viewMode === "day" ? "天" : "周"}）`}
        style={{ marginBottom: 12 }}
        bodyStyle={{ padding: 12 }}
      >
        <Typography.Text type="secondary">
          展示窗口：{rangeStart.format("YYYY-MM-DD HH:mm")} ~ {rangeEnd.format("YYYY-MM-DD HH:mm")}
        </Typography.Text>
        <div style={{ marginTop: 10, position: "relative", height: 22, borderRadius: 6, background: "rgba(120,120,120,0.12)" }}>
          {timeAxisTicks.map((tick, i) => {
            const ratio = (i / Math.max(1, timeAxisTicks.length - 1)) * 100;
            return (
              <div key={tick.valueOf()} style={{ position: "absolute", left: `${ratio}%`, top: 0, bottom: 0 }}>
                <div style={{ position: "absolute", top: 0, bottom: 0, width: 1, background: "rgba(255,255,255,0.15)" }} />
                <span
                  style={{
                    position: "absolute",
                    top: 2,
                    left: i === 0 ? 2 : i === timeAxisTicks.length - 1 ? -42 : 4,
                    fontSize: 11,
                    color: "rgba(255,255,255,0.75)",
                  }}
                >
                  {viewMode === "day" ? tick.format("HH:mm") : tick.format("MM-DD")}
                </span>
              </div>
            );
          })}
          <div style={{ position: "absolute", left: `${nowRatio}%`, top: 0, bottom: 0, width: 2, background: "#ff4d4f", boxShadow: "0 0 6px #ff4d4f" }} />
        </div>
        <div style={{ marginTop: 12, display: "grid", gap: 10 }}>
          {groupedBars.length === 0 ? (
            <Typography.Text type="secondary">当前窗口内无值班班次</Typography.Text>
          ) : (
            groupedBars.map((g) => (
              <div key={g.rid}>
                <div style={{ marginBottom: 6 }}>
                  <Tag color="blue">{ruleNameMap.get(g.rid) || "未命名规则"}</Tag>
                </div>
                <div style={{ position: "relative", height: 28, borderRadius: 6, background: "rgba(120,120,120,0.15)", overflow: "hidden" }}>
                  {g.blocks.map((b) => {
                    const s = Math.max(dayjs(b.starts_at).valueOf(), rangeStart.valueOf());
                    const e = Math.min(dayjs(b.ends_at).valueOf(), rangeEnd.valueOf());
                    const left = ((s - rangeStart.valueOf()) / totalRangeMs) * 100;
                    const width = Math.max(1, ((e - s) / totalRangeMs) * 100);
                    return (
                      <div
                        key={b.id}
                        title={`${b.title || "班次"} (${formatDateTime(b.starts_at)} ~ ${formatDateTime(b.ends_at)})`}
                        style={{
                          position: "absolute",
                          left: `${left}%`,
                          width: `${width}%`,
                          top: 3,
                          bottom: 3,
                          borderRadius: 4,
                          background: dayjs(b.ends_at).isBefore(dayjs())
                            ? "linear-gradient(90deg, rgba(120,120,120,0.68), rgba(90,90,90,0.75))"
                            : "linear-gradient(90deg, rgba(22,119,255,0.9), rgba(64,169,255,0.9))",
                          color: "#fff",
                          fontSize: 12,
                          lineHeight: "22px",
                          padding: "0 6px",
                          whiteSpace: "nowrap",
                          overflow: "hidden",
                          textOverflow: "ellipsis",
                        }}
                      >
                        {b.title || "未命名班次"}
                      </div>
                    );
                  })}
                </div>
              </div>
            ))
          )}
        </div>
      </Card>
      </div>

      <Table<AlertDutyBlockItem>
        rowKey="id"
        loading={loading}
        dataSource={list}
        pagination={{ pageSize: 10, showSizeChanger: true, pageSizeOptions: [10, 20, 50] }}
        scroll={{ x: 1400 }}
        columns={[
          { title: "ID", dataIndex: "id", width: 80 },
          { title: "规则", width: 220, render: (_, row) => <Tag>{ruleNameMap.get(row.monitor_rule_id) || "未命名规则"}</Tag> },
          { title: "班次标题", dataIndex: "title", ellipsis: true },
          {
            title: "值班人",
            width: 260,
            render: (_, row) => {
              const names = parseUintArrayJSON(row.user_ids_json).map((id) => userNameMap.get(id)).filter(Boolean) as string[];
              if (!names.length) return "-";
              return (
                <Space size={[4, 4]} wrap>
                  {names.map((n) => (
                    <Tag key={n} color="blue">
                      {n}
                    </Tag>
                  ))}
                </Space>
              );
            },
          },
          {
            title: "值班部门",
            width: 240,
            render: (_, row) => {
              const names = parseUintArrayJSON(row.department_ids_json).map((id) => deptNameMap.get(id)).filter(Boolean) as string[];
              if (!names.length) return "-";
              return (
                <Space size={[4, 4]} wrap>
                  {names.map((n) => (
                    <Tag key={n}>{n}</Tag>
                  ))}
                </Space>
              );
            },
          },
          {
            title: "额外邮箱",
            width: 220,
            render: (_, row) => {
              const emails = parseStringArrayJSON(row.extra_emails_json);
              return emails.length ? emails.join(", ") : "-";
            },
          },
          { title: "开始时间", dataIndex: "starts_at", render: (v: string) => formatDateTime(v), width: 180 },
          { title: "结束时间", dataIndex: "ends_at", render: (v: string) => formatDateTime(v), width: 180 },
          {
            title: "状态",
            width: 100,
            render: (_, row) => (dayjs(row.ends_at).isBefore(dayjs()) ? <Tag color="default">已结束</Tag> : <Tag color="green">进行中/待生效</Tag>),
          },
          {
            title: "操作",
            width: 150,
            render: (_, row) => (
              <Space>
                <Button type="link" icon={<EditOutlined />} onClick={() => openEdit(row)}>
                  编辑
                </Button>
                <Popconfirm
                  title="确认删除该值班班次吗？"
                  onConfirm={() =>
                    void (async () => {
                      await deleteDutyBlock(row.id);
                      message.success("已删除");
                      await loadBlocks();
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

      <Modal title={current ? "编辑值班班次" : "新增值班"} open={open} onCancel={() => setOpen(false)} onOk={() => void submit()} confirmLoading={saving}>
        <Form form={form} layout="vertical">
          <Form.Item
            name="monitor_rule_ids"
            label={current ? "关联监控规则（编辑时仅保留一条）" : "关联监控规则（可多选）"}
            rules={[{ required: true, message: "请选择规则" }]}
          >
            <Select mode={current ? undefined : "multiple"} options={ruleOptions} optionFilterProp="label" />
          </Form.Item>
          <Form.Item name="title" label="班次标题">
            <Input placeholder="例如：数据库值班（夜班）" />
          </Form.Item>
          <Form.Item name="range" label="值班时间" rules={[{ required: true, message: "请选择时间范围" }]}>
            <DatePicker.RangePicker showTime style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item name="user_ids" label="值班人">
            <Select mode="multiple" options={users} optionFilterProp="label" placeholder="选择值班人" />
          </Form.Item>
          <Form.Item name="department_ids" label="值班部门">
            <TreeSelect
              treeCheckable
              showCheckedStrategy={TreeSelect.SHOW_PARENT}
              treeData={deptTree}
              placeholder="选择值班部门（包含子部门）"
              style={{ width: "100%" }}
            />
          </Form.Item>
          <Form.Item name="extra_emails" label="额外邮箱">
            <Select mode="tags" tokenSeparators={[",", " ", ";"]} placeholder="输入后回车" />
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <Input.TextArea rows={2} />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}

function parseUintArrayJSON(raw?: string): number[] {
  const s = String(raw || "").trim();
  if (!s) return [];
  try {
    const arr = JSON.parse(s) as unknown[];
    if (!Array.isArray(arr)) return [];
    return arr.map((it) => Number(it)).filter((n) => Number.isFinite(n) && n > 0);
  } catch {
    return [];
  }
}

function parseStringArrayJSON(raw?: string): string[] {
  const s = String(raw || "").trim();
  if (!s) return [];
  try {
    const arr = JSON.parse(s) as unknown[];
    if (!Array.isArray(arr)) return [];
    return arr.map((it) => String(it || "").trim()).filter(Boolean);
  } catch {
    return [];
  }
}

