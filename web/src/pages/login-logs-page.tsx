import { DeleteOutlined, ReloadOutlined, SearchOutlined } from "@ant-design/icons";
import { Button, Card, Input, Popconfirm, Select, Space, Table, Tag, Typography, message } from "antd";
import { useEffect, useState } from "react";
import { PageHero } from "../components/page-hero";
import { batchDeleteLoginLogs, deleteLoginLog, getLoginLogs, exportLoginLogs } from "../services/login-logs";
import type { LoginLogItem, LoginLogQuery } from "../services/login-logs";
import { formatDateTime } from "../utils/format";

const defaultQuery: LoginLogQuery = { page: 1, page_size: 10 };

function sourceLabel(source: string) {
  if (source === "email") return "邮箱验证码";
  if (source === "password") return "用户名密码";
  return source || "-";
}

function sourceColor(source: string) {
  if (source === "email") return "cyan";
  if (source === "password") return "blue";
  return "default";
}

export function LoginLogsPage() {
  const [list, setList] = useState<LoginLogItem[]>([]);
  const [total, setTotal] = useState(0);
  const [query, setQuery] = useState<LoginLogQuery>(defaultQuery);
  const [loading, setLoading] = useState(false);
  const [selectedRowKeys, setSelectedRowKeys] = useState<number[]>([]);
  const [filters, setFilters] = useState({ username: "", status: undefined as number | undefined, source: "" });
  const fileRef = (null as unknown) as HTMLInputElement | null;

  useEffect(() => {
    void loadList();
  }, [query]);

  async function loadList() {
    setLoading(true);
    try {
      const result = await getLoginLogs(query);
      setList(result.list || []);
      setTotal(result.total || 0);
    } finally {
      setLoading(false);
    }
  }

  function handleSearch() {
    setQuery({ ...filters, page: 1, page_size: query.page_size });
  }

  function handleReset() {
    setFilters({ username: "", status: undefined, source: "" });
    setQuery(defaultQuery);
  }

  async function handleBatchDelete() {
    if (selectedRowKeys.length === 0) {
      message.warning("请先选择要删除的记录");
      return;
    }
    await batchDeleteLoginLogs(selectedRowKeys);
    message.success("已删除");
    setSelectedRowKeys([]);
    void loadList();
  }

  async function handleExport() {
    try {
      const res = await exportLoginLogs({ username: filters.username, status: filters.status, source: filters.source });
      const blob = new Blob([res], { type: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" });
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = "login_logs.xlsx";
      a.click();
      window.URL.revokeObjectURL(url);
    } catch (err) {
      message.error("导出失败");
    }
  }

  async function handleDelete(id: number) {
    await deleteLoginLog(id);
    message.success("已删除");
    void loadList();
  }

  return (
    <div>
      <PageHero
        title="登录日志"
        subtitle="记录用户登录行为，区分用户名密码登录与邮箱验证码登录。"
        breadcrumbItems={[{ title: "控制台" }, { title: "系统管理" }, { title: "登录日志" }]}
      />

      <Card className="table-card">
        <div className="toolbar">
          <Space wrap>
            <Input
              allowClear
              placeholder="搜索用户名"
              style={{ width: 180 }}
              value={filters.username}
              onChange={(e) => setFilters((p) => ({ ...p, username: e.target.value }))}
              onPressEnter={handleSearch}
            />
            <Select
              allowClear
              placeholder="状态"
              style={{ width: 120 }}
              value={filters.status}
              onChange={(v) => setFilters((p) => ({ ...p, status: v }))}
              options={[
                { label: "成功", value: 1 },
                { label: "失败", value: 0 },
              ]}
            />
            <Select
              allowClear
              placeholder="登录来源"
              style={{ width: 160 }}
              value={filters.source || undefined}
              onChange={(v) => setFilters((p) => ({ ...p, source: v ?? "" }))}
              options={[
                { label: "用户名密码", value: "password" },
                { label: "邮箱验证码", value: "email" },
              ]}
            />
            <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
              查询
            </Button>
            <Button icon={<ReloadOutlined />} onClick={handleReset}>
              重置
            </Button>
          </Space>
          <div className="toolbar__actions">
            <Button onClick={() => void handleExport()}>导出</Button>
            <Popconfirm title="确定删除选中的日志？" onConfirm={() => void handleBatchDelete()}>
              <Button danger icon={<DeleteOutlined />} disabled={selectedRowKeys.length === 0}>
                批量删除
              </Button>
            </Popconfirm>
          </div>
        </div>

        <Table
          rowKey="id"
          loading={loading}
          rowSelection={{
            selectedRowKeys,
            onChange: (keys) => setSelectedRowKeys(keys as number[]),
          }}
          dataSource={list}
          pagination={{
            current: query.page,
            pageSize: query.page_size,
            total,
            showSizeChanger: true,
            showTotal: (t) => `共 ${t} 条`,
            onChange: (page, pageSize) => setQuery((p) => ({ ...p, page, page_size: pageSize })),
          }}
          scroll={{ x: 1200 }}
          columns={[
            { title: "ID", dataIndex: "id", width: 70 },
            { title: "用户名", dataIndex: "username", width: 120, render: (v: string) => v || "-" },
            {
              title: "登录来源",
              dataIndex: "source",
              width: 120,
              render: (s: string) => <Tag color={sourceColor(s)}>{sourceLabel(s)}</Tag>,
            },
            { title: "登录IP", dataIndex: "ip", width: 140, render: (v: string) => v || "-" },
            {
              title: "状态",
              dataIndex: "status",
              width: 90,
              render: (v: number) =>
                v === 1 ? <Tag color="success">成功</Tag> : <Tag color="error">失败</Tag>,
            },
            { title: "详情", dataIndex: "detail", ellipsis: true, render: (v: string) => v || "-" },
            {
              title: "浏览器/设备",
              dataIndex: "user_agent",
              width: 200,
              ellipsis: true,
              render: (ua: string) => (
                <Typography.Text ellipsis={{ tooltip: ua }} style={{ maxWidth: 180 }}>
                  {ua || "-"}
                </Typography.Text>
              ),
            },
            {
              title: "登录时间",
              dataIndex: "created_at",
              width: 180,
              render: formatDateTime,
            },
            {
              title: "操作",
              key: "action",
              width: 90,
              fixed: "right",
              render: (_: unknown, record: LoginLogItem) => (
                <Popconfirm title="确定删除？" onConfirm={() => void handleDelete(record.id)}>
                  <Button type="link" size="small" danger icon={<DeleteOutlined />}>
                    删除
                  </Button>
                </Popconfirm>
              ),
            },
          ]}
        />
      </Card>
    </div>
  );
}