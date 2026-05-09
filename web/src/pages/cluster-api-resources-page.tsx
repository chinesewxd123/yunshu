import { ReloadOutlined } from "@ant-design/icons";
import { Alert, Button, Card, Select, Space, Table, Tag, Typography } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useCallback, useEffect, useState } from "react";
import type { APIResourceDiscoveryItem } from "../services/cluster-api-resources";
import { listClusterAPIResources } from "../services/cluster-api-resources";
import { getClusters } from "../services/clusters";

export function ClusterApiResourcesPage() {
  const [loading, setLoading] = useState(false);
  const [clusters, setClusters] = useState<{ id: number; name: string }[]>([]);
  const [clusterId, setClusterId] = useState<number>();
  const [nsFilter, setNsFilter] = useState<"all" | "true" | "false">("all");
  const [rows, setRows] = useState<APIResourceDiscoveryItem[]>([]);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      const res = await getClusters({ page: 1, page_size: 200 });
      if (cancelled) return;
      const list = (res.list ?? []).map((c) => ({ id: c.id, name: c.name }));
      setClusters(list);
      setClusterId((prev) => prev ?? list[0]?.id);
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const loadResources = useCallback(async () => {
    if (!clusterId) {
      setRows([]);
      return;
    }
    setLoading(true);
    try {
      const ns =
        nsFilter === "all" ? null : nsFilter === "true" ? true : false;
      const data = await listClusterAPIResources(clusterId, ns);
      setRows(data.list ?? []);
    } finally {
      setLoading(false);
    }
  }, [clusterId, nsFilter]);

  useEffect(() => {
    void loadResources();
  }, [loadResources]);

  const columns: ColumnsType<APIResourceDiscoveryItem> = [
    { title: "API 版本", dataIndex: "group_version", width: 200, ellipsis: true },
    { title: "资源名", dataIndex: "name", width: 200 },
    { title: "Kind", dataIndex: "kind", width: 160 },
    {
      title: "作用域",
      dataIndex: "namespaced",
      width: 100,
      render: (v: boolean) => (v ? <Tag color="blue">命名空间</Tag> : <Tag>集群</Tag>),
    },
    {
      title: "Verbs",
      dataIndex: "verbs",
      render: (verbs: string[]) => (
        <Typography.Text style={{ fontSize: 12 }}>{(verbs ?? []).join(", ") || "-"}</Typography.Text>
      ),
    },
  ];

  return (
    <div>
      <Card className="table-card" title="API 资源发现（kubectl api-resources）">
        <Space direction="vertical" size="middle" style={{ width: "100%" }}>
          <Alert
            type="info"
            showIcon
            message="用途：排查 CRD、APIService、多版本资源是否与 kubectl 一致；不含 MCP/AI。"
          />
          <Space wrap>
            <Typography.Text className="inline-muted">集群</Typography.Text>
            <Select
              style={{ minWidth: 280 }}
              placeholder="选择集群"
              value={clusterId}
              options={clusters.map((c) => ({ label: `${c.name} (#${c.id})`, value: c.id }))}
              onChange={(v) => setClusterId(v)}
            />
            <Typography.Text className="inline-muted">过滤</Typography.Text>
            <Select
              style={{ width: 200 }}
              value={nsFilter}
              onChange={(v) => setNsFilter(v)}
              options={[
                { label: "全部", value: "all" },
                { label: "仅命名空间级", value: "true" },
                { label: "仅集群级", value: "false" },
              ]}
            />
            <Button icon={<ReloadOutlined />} onClick={() => void loadResources()} loading={loading}>
              刷新
            </Button>
          </Space>
          <Table<APIResourceDiscoveryItem>
            rowKey={(r) => `${r.group_version}/${r.name}/${r.kind}`}
            loading={loading}
            columns={columns}
            dataSource={rows}
            pagination={{ pageSize: 50, showSizeChanger: true, showTotal: (t) => `共 ${t} 条` }}
            scroll={{ x: "max-content" }}
            size="small"
          />
        </Space>
      </Card>
    </div>
  );
}
