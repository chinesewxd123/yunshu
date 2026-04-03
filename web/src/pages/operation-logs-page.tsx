import { DeleteOutlined, EyeOutlined, ReloadOutlined, SearchOutlined } from "@ant-design/icons";
import { Button, Card, Input, InputNumber, Modal, Popconfirm, Select, Space, Table, Tag, Typography, message } from "antd";
import { useEffect, useState } from "react";
import { batchDeleteOperationLogs, deleteOperationLog, getOperationLogs, exportOperationLogs } from "../services/operation-logs";
import type { OperationLogItem, OperationLogQuery } from "../services/operation-logs";
import { formatDateTime } from "../utils/format";

const defaultQuery: OperationLogQuery = { page: 1, page_size: 10 };

function methodColor(method: string) {
  switch (method) {
    case "GET":
      return "green";
    case "POST":
      return "blue";
    case "PUT":
      return "orange";
    case "DELETE":
      return "red";
    default:
      return "default";
  }
}

function statusColor(code: number) {
  if (code >= 200 && code < 300) return "success";
  if (code >= 400 && code < 500) return "warning";
  if (code >= 500) return "error";
  return "default";
}

export function OperationLogsPage() {
  const [list, setList] = useState<OperationLogItem[]>([]);
  const [total, setTotal] = useState(0);
  const [query, setQuery] = useState<OperationLogQuery>(defaultQuery);
  const [loading, setLoading] = useState(false);
  const [selectedRowKeys, setSelectedRowKeys] = useState<number[]>([]);
  const [filters, setFilters] = useState({ method: "", path: "", status_code: undefined as number | undefined });
  const [bodyModal, setBodyModal] = useState<{ title: string; content: string } | null>(null);

  useEffect(() => {
    void loadList();
  }, [query]);

  async function loadList() {
    setLoading(true);
    try {
      const result = await getOperationLogs(query);
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
    setFilters({ method: "", path: "", status_code: undefined });
    setQuery(defaultQuery);
  }

  async function handleBatchDelete() {
    if (selectedRowKeys.length === 0) {
      message.warning("请先选择要删除的记录");
      return;
    }
    await batchDeleteOperationLogs(selectedRowKeys);
    message.success("已删除");
    setSelectedRowKeys([]);
    void loadList();
  }

  async function handleExport() {
    try {
      const res = await exportOperationLogs({ method: filters.method, path: filters.path, status_code: filters.status_code });
      const blob = res;
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = "operation_logs.xlsx";
      a.click();
      window.URL.revokeObjectURL(url);
    } catch (err) {
      message.error("导出失败");
    }
  }

  async function handleDelete(id: number) {
    await deleteOperationLog(id);
    message.success("已删除");
    void loadList();
  }

  function operatorText(row: OperationLogItem) {
    const nick = row.nickname ? `(${row.nickname})` : "";
    return `${row.username}${nick}`;
  }

  return (
    <div>
      <Card className="table-card">
        <div className="toolbar">
          <Space wrap>
            <Select
              allowClear
              placeholder="请求方法"
              style={{ width: 120 }}
              value={filters.method || undefined}
              onChange={(v) => setFilters((p) => ({ ...p, method: v ?? "" }))}
              options={[
                { label: "GET", value: "GET" },
                { label: "POST", value: "POST" },
                { label: "PUT", value: "PUT" },
                { label: "DELETE", value: "DELETE" },
              ]}
            />
            <Input
              allowClear
              placeholder="请求路径"
              style={{ width: 220 }}
              value={filters.path}
              onChange={(e) => setFilters((p) => ({ ...p, path: e.target.value }))}
              onPressEnter={handleSearch}
            />
            <InputNumber
              placeholder="状态码"
              style={{ width: 120 }}
              value={filters.status_code}
              onChange={(v) => setFilters((p) => ({ ...p, status_code: v ?? undefined }))}
              min={100}
              max={599}
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
          scroll={{ x: 1400 }}
          columns={[
            { title: "ID", dataIndex: "id", width: 70 },
            { title: "操作人", width: 140, render: (_: unknown, r: OperationLogItem) => operatorText(r) },
            { title: "IP", dataIndex: "ip", width: 130, render: (v: string) => v || "-" },
            {
              title: "方法",
              dataIndex: "method",
              width: 90,
              render: (m: string) => <Tag color={methodColor(m)}>{m}</Tag>,
            },
            {
              title: "路径",
              dataIndex: "path",
              width: 280,
              ellipsis: true,
              render: (v: string) => (
                <Typography.Text ellipsis={{ tooltip: v }} style={{ maxWidth: 260 }}>
                  {v}
                </Typography.Text>
              ),
            },
            {
              title: "状态码",
              dataIndex: "status_code",
              width: 90,
              render: (code: number) => <Tag color={statusColor(code)}>{code}</Tag>,
            },
            { title: "耗时(ms)", dataIndex: "latency_ms", width: 100, render: (v: number) => v || 0 },
            {
              title: "请求体",
              width: 100,
              render: (_: unknown, r: OperationLogItem) =>
                r.request_body ? (
                  <Button
                    type="link"
                    size="small"
                    icon={<EyeOutlined />}
                    onClick={() => setBodyModal({ title: "请求体", content: r.request_body })}
                  >
                    查看
                  </Button>
                ) : (
                  "-"
                ),
            },
            {
              title: "响应体",
              width: 100,
              render: (_: unknown, r: OperationLogItem) =>
                r.response_body ? (
                  <Button
                    type="link"
                    size="small"
                    icon={<EyeOutlined />}
                    onClick={() => setBodyModal({ title: "响应体", content: r.response_body })}
                  >
                    查看
                  </Button>
                ) : (
                  "-"
                ),
            },
            {
              title: "操作时间",
              dataIndex: "created_at",
              width: 180,
              render: formatDateTime,
            },
            {
              title: "操作",
              key: "action",
              width: 90,
              fixed: "right",
              render: (_: unknown, record: OperationLogItem) => (
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

      <Modal
        title={bodyModal?.title}
        open={!!bodyModal}
        onCancel={() => setBodyModal(null)}
        footer={null}
        width={700}
      >
        {bodyModal && (
          <pre style={{ maxHeight: 400, overflow: "auto", background: "#f5f5f5", padding: 12, borderRadius: 4 }}>
            {bodyModal.content}
          </pre>
        )}
      </Modal>
    </div>
  );
}