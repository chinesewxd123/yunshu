import { EyeOutlined, ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Empty, Input, Select, Space, Table, Tag, Tooltip, Typography, message } from "antd";
import { useEffect, useMemo, useState } from "react";
import { getClusters, listComponentStatuses, type ComponentStatusItem, type ClusterItem } from "../services/clusters";

export function ComponentStatusPage() {
  const [loading, setLoading] = useState(false);
  const [clusterLoading, setClusterLoading] = useState(false);
  const [clusters, setClusters] = useState<ClusterItem[]>([]);
  const [selectedClusterId, setSelectedClusterId] = useState<number>();
  const [keyword, setKeyword] = useState("");
  const [list, setList] = useState<ComponentStatusItem[]>([]);

  useEffect(() => {
    void loadClusters();
  }, []);

  async function loadClusters() {
    setClusterLoading(true);
    try {
      const result = await getClusters({ page: 1, page_size: 200 });
      const enabled = result.list.filter((it) => it.status === 1);
      setClusters(enabled);
      const nextCluster = enabled[0]?.id;
      setSelectedClusterId((prev) => prev ?? nextCluster);
      if (nextCluster) {
        await loadComponentStatuses(nextCluster);
      } else {
        setList([]);
      }
    } finally {
      setClusterLoading(false);
    }
  }

  async function loadComponentStatuses(clusterId: number) {
    setLoading(true);
    try {
      const result = await listComponentStatuses(clusterId);
      setList(result.list ?? []);
    } catch (error) {
      const msg = error instanceof Error ? error.message : "加载组件状态失败";
      message.error(msg || "加载组件状态失败");
      setList([]);
    } finally {
      setLoading(false);
    }
  }

  const filtered = useMemo(() => {
    const kw = keyword.trim().toLowerCase();
    if (!kw) return list;
    return list.filter((it) => it.name.toLowerCase().includes(kw));
  }, [keyword, list]);

  return (
    <Card className="table-card" title="组件状态">
      <Space direction="vertical" size={12} style={{ width: "100%" }}>
        <Space wrap style={{ width: "100%", justifyContent: "space-between" }}>
          <Space wrap>
            <Typography.Text className="inline-muted">集群</Typography.Text>
            <Select
              style={{ minWidth: 260 }}
              loading={clusterLoading}
              value={selectedClusterId}
              placeholder="请选择集群"
              options={clusters.map((c) => ({ label: `${c.name} (#${c.id})`, value: c.id }))}
              onChange={(v) => {
                setSelectedClusterId(v);
                void loadComponentStatuses(v);
              }}
            />
            <Input
              allowClear
              style={{ width: 240 }}
              value={keyword}
              onChange={(e) => setKeyword(e.target.value)}
              placeholder="按组件名称过滤"
            />
          </Space>
          <Button
            icon={<ReloadOutlined />}
            onClick={() => {
              if (!selectedClusterId) return;
              void loadComponentStatuses(selectedClusterId);
            }}
          >
            刷新
          </Button>
        </Space>

        <Table
          rowKey="name"
          loading={loading}
          dataSource={filtered}
          size="small"
          locale={{ emptyText: <Empty description={selectedClusterId ? "暂无组件状态数据" : "请先选择集群"} image={Empty.PRESENTED_IMAGE_SIMPLE} /> }}
          pagination={{ pageSize: 10, showSizeChanger: true }}
          scroll={{ x: "max-content" }}
          columns={[
            { title: "名称", dataIndex: "name", width: 220 },
            {
              title: "状态",
              dataIndex: "status",
              width: 120,
              render: (v: string, record: ComponentStatusItem) => (
                <Tag color={record.healthy ? "success" : v === "Unhealthy" ? "error" : "default"}>{v || "Unknown"}</Tag>
              ),
            },
            {
              title: "探测信息",
              dataIndex: "message",
              render: (v: string) => v || "-",
            },
            {
              title: "错误",
              dataIndex: "error",
              render: (v: string) =>
                v ? (
                  <Tooltip title={v}>
                    <Tag color="error" icon={<EyeOutlined />}>
                      查看
                    </Tag>
                  </Tooltip>
                ) : (
                  "-"
                ),
            },
            { title: "最近探测时间", dataIndex: "last_probe_at", width: 180, render: (v: string) => v || "-" },
          ]}
        />
      </Space>
    </Card>
  );
}

