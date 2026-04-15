import { Card, Input, Select, Space, Table, Tag, Typography } from "antd";
import { useEffect, useState } from "react";
import { listAlertEvents, type AlertEventItem } from "../services/alerts";
import { formatDateTime } from "../utils/format";

export function AlertEventsPage() {
  const [loading, setLoading] = useState(false);
  const [list, setList] = useState<AlertEventItem[]>([]);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [total, setTotal] = useState(0);
  const [keyword, setKeyword] = useState("");
  const [cluster, setCluster] = useState("");
  const [groupKey, setGroupKey] = useState("");
  const clusterOptions = Array.from(new Set((list ?? []).map((it) => (it.cluster || "").trim()).filter(Boolean))).map((v) => ({
    label: v,
    value: v,
  }));

  async function load(nextPage = page, nextPageSize = pageSize) {
    setLoading(true);
    try {
      const res = await listAlertEvents({
        page: nextPage,
        page_size: nextPageSize,
        keyword: keyword.trim() || undefined,
        cluster: cluster.trim() || undefined,
        group_key: groupKey.trim() || undefined,
      });
      setList(res.list ?? []);
      setTotal(res.total ?? 0);
      setPage(res.page ?? nextPage);
      setPageSize(res.page_size ?? nextPageSize);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load(1, 10);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    const timer = window.setTimeout(() => {
      void load(1, pageSize);
    }, 300);
    return () => window.clearTimeout(timer);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [keyword, cluster, groupKey]);

  return (
    <Card className="table-card" title="Webhook 告警事件">
      <Space style={{ width: "100%", marginBottom: 12 }} wrap>
        <Input
          style={{ width: 260 }}
          placeholder="关键词（标题/错误/通道）"
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
          allowClear
        />
        <Select
          style={{ width: 260 }}
          placeholder="按 cluster 筛选（如 prod-1）"
          value={cluster || undefined}
          options={clusterOptions}
          showSearch
          allowClear
          onSearch={(v) => setCluster(v)}
          onChange={(v) => setCluster((v as string) || "")}
          filterOption={(input, option) => String(option?.value ?? "").toLowerCase().includes(input.toLowerCase())}
        />
        <Input
          style={{ width: 220 }}
          placeholder="group_key"
          value={groupKey}
          onChange={(e) => setGroupKey(e.target.value)}
          allowClear
        />
      </Space>
      <Table
        rowKey="id"
        loading={loading}
        dataSource={list}
        pagination={{
          current: page,
          pageSize,
          total,
          showSizeChanger: true,
          onChange: (p, ps) => {
            void load(p, ps);
          },
        }}
        columns={[
          { title: "ID", dataIndex: "id", width: 90 },
          { title: "标题", dataIndex: "title", width: 260, ellipsis: true },
          { title: "集群", dataIndex: "cluster", width: 160, ellipsis: true, render: (v: string) => v || "-" },
          { title: "GroupKey", dataIndex: "group_key", width: 160, ellipsis: true, render: (v: string) => v || "-" },
          { title: "来源", dataIndex: "source", width: 140 },
          {
            title: "级别",
            dataIndex: "severity",
            width: 110,
            render: (v: string) => {
              const color = v === "critical" ? "red" : v === "warning" ? "orange" : "blue";
              return <Tag color={color}>{v || "-"}</Tag>;
            },
          },
          {
            title: "状态",
            dataIndex: "status",
            width: 100,
            render: (v: string) => <Tag>{v || "-"}</Tag>,
          },
          { title: "通道", dataIndex: "channel_name", width: 160 },
          {
            title: "发送结果",
            dataIndex: "success",
            width: 110,
            render: (v: boolean) => (v ? <Tag color="success">成功</Tag> : <Tag color="error">失败</Tag>),
          },
          { title: "HTTP", dataIndex: "http_status_code", width: 90 },
          {
            title: "错误信息",
            dataIndex: "error_message",
            ellipsis: true,
            render: (v: string) =>
              v ? (
                <Typography.Text type="danger" ellipsis={{ tooltip: v }}>
                  {v}
                </Typography.Text>
              ) : (
                "-"
              ),
          },
          { title: "时间", dataIndex: "created_at", width: 170, render: (v: string) => formatDateTime(v) },
        ]}
      />
    </Card>
  );
}

