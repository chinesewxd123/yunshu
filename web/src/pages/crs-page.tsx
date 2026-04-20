import { DeleteOutlined, EditOutlined, EyeOutlined, FileAddOutlined, ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Drawer, Form, Input, Modal, Popconfirm, Select, Space, Table, Tag, TreeSelect, Typography, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useEffect, useMemo, useState } from "react";
import { getClusters, listNamespaces as listClusterNamespaces, type ClusterItem } from "../services/clusters";
import { applyCr, deleteCr, getCrDetail, listCrResources, listCrs, type CrDetail, type CrItem, type CrResourceItem } from "../services/crs";

export function CrsPage() {
  const [clusters, setClusters] = useState<ClusterItem[]>([]);
  const [clusterId, setClusterId] = useState<number>();
  const [resources, setResources] = useState<CrResourceItem[]>([]);
  const [selectedResourceName, setSelectedResourceName] = useState<string>();
  const [namespaces, setNamespaces] = useState<Array<{ label: string; value: string }>>([]);
  const [namespace, setNamespace] = useState<string>("default");
  const [keyword, setKeyword] = useState("");
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<CrItem[]>([]);

  const [detailOpen, setDetailOpen] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detailName, setDetailName] = useState("");
  const [detail, setDetail] = useState<CrDetail | null>(null);
  const [detailYaml, setDetailYaml] = useState("");
  const [detailSubmitting, setDetailSubmitting] = useState(false);

  const [applyOpen, setApplyOpen] = useState(false);
  const [applyLoading, setApplyLoading] = useState(false);
  const [manifest, setManifest] = useState("");

  const selectedResource = useMemo(
    () => resources.find((r) => r.name === selectedResourceName),
    [resources, selectedResourceName],
  );
  const resourceTree = useMemo(() => {
    const groups = new Map<string, CrResourceItem[]>();
    for (const r of resources) {
      const k = r.group || "core";
      if (!groups.has(k)) groups.set(k, []);
      groups.get(k)?.push(r);
    }
    return Array.from(groups.entries()).map(([group, list]) => ({
      title: group,
      value: `group:${group}`,
      selectable: false,
      children: list.map((r) => ({
        title: `${r.kind} (${r.version}) - ${r.resource}`,
        value: r.name,
      })),
    }));
  }, [resources]);

  const columns: ColumnsType<CrItem> = [
    { title: "名称", dataIndex: "name", width: 220 },
    { title: "命名空间", dataIndex: "namespace", width: 140, render: (v?: string) => v || "-" },
    { title: "APIVersion", dataIndex: "api_version", width: 220 },
    { title: "Kind", dataIndex: "kind", width: 180 },
    { title: "创建时间", dataIndex: "creation_time", width: 180, fixed: "right" },
    {
      title: "操作",
      key: "action",
      width: 240,
      fixed: "right",
      render: (_: unknown, record: CrItem) => (
        <Space>
          <Button type="link" icon={<EyeOutlined />} onClick={() => void openDetail(record.name)}>
            详情
          </Button>
          <Button type="link" icon={<EditOutlined />} onClick={() => void openEdit(record.name)}>
            编辑
          </Button>
          <Popconfirm title={`确认删除 ${record.name} 吗？`} onConfirm={() => void doDelete(record.name)}>
            <Button danger type="link" icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  async function loadClusters() {
    const res = await getClusters({ page: 1, page_size: 200 });
    const list = res.list ?? [];
    setClusters(list);
    if (!clusterId) {
      const first = list.find((c) => c.status === 1);
      if (first) setClusterId(first.id);
    }
  }

  async function loadResources(cid: number) {
    const list = await listCrResources(cid);
    setResources(list);
    if (!list.some((x) => x.name === selectedResourceName)) {
      setSelectedResourceName(list[0]?.name);
    }
  }

  async function loadNamespaces(cid: number) {
    const res = await listClusterNamespaces(cid);
    const opts = (res.list ?? []).map((n) => ({ label: n.name, value: n.name }));
    setNamespaces(opts);
    if (!opts.some((o) => o.value === namespace)) {
      setNamespace(opts[0]?.value ?? "default");
    }
  }

  async function reload() {
    if (!clusterId || !selectedResource) return;
    if (selectedResource.namespaced && !namespace) return;
    setLoading(true);
    try {
      const list = await listCrs({
        clusterId,
        group: selectedResource.group,
        version: selectedResource.version,
        resource: selectedResource.resource,
        namespace: selectedResource.namespaced ? namespace : undefined,
        keyword: keyword.trim() || undefined,
      });
      setData(list ?? []);
    } finally {
      setLoading(false);
    }
  }

  async function openDetail(name: string) {
    if (!clusterId || !selectedResource) return;
    setDetailOpen(true);
    setDetailLoading(true);
    setDetailName(name);
    setDetail(null);
    try {
      const d = await getCrDetail({
        clusterId,
        group: selectedResource.group,
        version: selectedResource.version,
        resource: selectedResource.resource,
        namespace: selectedResource.namespaced ? namespace : undefined,
        name,
      });
      setDetail(d);
      setDetailYaml(d.yaml ?? "");
    } finally {
      setDetailLoading(false);
    }
  }

  async function openEdit(name: string) {
    if (!clusterId || !selectedResource) return;
    setApplyOpen(true);
    setApplyLoading(true);
    try {
      const d = await getCrDetail({
        clusterId,
        group: selectedResource.group,
        version: selectedResource.version,
        resource: selectedResource.resource,
        namespace: selectedResource.namespaced ? namespace : undefined,
        name,
      });
      setManifest(d.yaml ?? "");
    } finally {
      setApplyLoading(false);
    }
  }

  async function doDelete(name: string) {
    if (!clusterId || !selectedResource) return;
    await deleteCr({
      clusterId,
      group: selectedResource.group,
      version: selectedResource.version,
      resource: selectedResource.resource,
      namespace: selectedResource.namespaced ? namespace : undefined,
      name,
    });
    message.success("删除成功");
    await reload();
  }

  useEffect(() => {
    void loadClusters();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (!clusterId) return;
    void (async () => {
      await Promise.all([loadResources(clusterId), loadNamespaces(clusterId)]);
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [clusterId]);

  useEffect(() => {
    void reload();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [clusterId, selectedResourceName, namespace]);

  return (
    <Card className="table-card" title="CR 实例管理">
      <div style={{ display: "flex", justifyContent: "space-between", marginBottom: 12, gap: 12, flexWrap: "wrap" }}>
        <Space wrap>
          <Select
            placeholder="选择集群"
            style={{ minWidth: 220 }}
            value={clusterId}
            onChange={setClusterId}
            options={clusters.map((c) => ({
              label: c.status === 1 ? c.name : `${c.name}（已停用）`,
              value: c.id,
              disabled: c.status !== 1,
            }))}
          />
          <TreeSelect
            placeholder="选择 CR 类型（按 Group/Kind）"
            style={{ minWidth: 420 }}
            value={selectedResourceName}
            onChange={(v) => setSelectedResourceName(String(v || ""))}
            treeData={resourceTree}
            showSearch
            treeDefaultExpandAll
            allowClear
          />
          {selectedResource?.namespaced ? (
            <Select
              placeholder="命名空间"
              style={{ minWidth: 180 }}
              value={namespace}
              onChange={setNamespace}
              options={namespaces}
              showSearch
              optionFilterProp="label"
            />
          ) : null}
          <Input.Search
            allowClear
            placeholder="搜索名称"
            style={{ width: 240 }}
            onSearch={(v) => {
              setKeyword(v);
              void reload();
            }}
          />
        </Space>
        <Space>
          <Button
            icon={<FileAddOutlined />}
            onClick={() => {
              const defaultNs = selectedResource?.namespaced ? `  namespace: ${namespace || "default"}\n` : "";
              const apiVersion = selectedResource ? `${selectedResource.group}/${selectedResource.version}` : "example.com/v1";
              const kind = selectedResource?.kind || "Example";
              setManifest(`apiVersion: ${apiVersion}
kind: ${kind}
metadata:
  name: demo-${kind.toLowerCase()}
${defaultNs}spec: {}
`);
              setApplyOpen(true);
            }}
          >
            快捷创建
          </Button>
          <Button
            type="primary"
            icon={<FileAddOutlined />}
            onClick={() => {
              setManifest("");
              setApplyOpen(true);
            }}
          >
            应用 YAML
          </Button>
          <Button icon={<ReloadOutlined />} onClick={() => void reload()}>
            刷新
          </Button>
        </Space>
      </div>

      <div style={{ marginBottom: 8 }}>
        {selectedResource ? (
          <Space>
            <Tag color="blue">{selectedResource.kind}</Tag>
            <Tag>{selectedResource.group}</Tag>
            <Tag>{selectedResource.version}</Tag>
            <Tag>{selectedResource.resource}</Tag>
            <Tag color={selectedResource.namespaced ? "green" : "purple"}>
              {selectedResource.namespaced ? "Namespaced" : "Cluster"}
            </Tag>
          </Space>
        ) : null}
      </div>

      <Table<CrItem>
        rowKey={(r) => `${r.namespace || "_cluster_"}-${r.name}`}
        columns={columns}
        dataSource={data}
        loading={loading}
        pagination={{ pageSize: 10, showSizeChanger: true, pageSizeOptions: [10, 20, 50, 100], showQuickJumper: true }}
        scroll={{ x: "max-content" }}
      />

      <Drawer
        title={`详情 - ${detailName}`}
        open={detailOpen}
        width={980}
        onClose={() => setDetailOpen(false)}
        className="detail-edit-drawer"
        extra={
          <Space>
            <Button
              icon={<EditOutlined />}
              onClick={() => {
                setManifest(detail?.yaml ?? "");
                setApplyOpen(true);
              }}
            >
              基于详情编辑
            </Button>
            <Button
              type="primary"
              loading={detailSubmitting}
              onClick={() => {
                if (!clusterId) {
                  message.warning("请先选择集群");
                  return;
                }
                setDetailSubmitting(true);
                void (async () => {
                  try {
                    await applyCr(clusterId, detailYaml);
                    message.success("详情修改已保存");
                    const latest = await getCrDetail({
                      clusterId,
                      group: selectedResource?.group || "",
                      version: selectedResource?.version || "",
                      resource: selectedResource?.resource || "",
                      namespace: selectedResource?.namespaced ? namespace : undefined,
                      name: detailName,
                    });
                    setDetail(latest);
                    setDetailYaml(latest.yaml ?? "");
                    await reload();
                  } finally {
                    setDetailSubmitting(false);
                  }
                })();
              }}
            >
              保存修改
            </Button>
          </Space>
        }
      >
        {detailLoading ? (
          <Typography.Text type="secondary">加载中...</Typography.Text>
        ) : (
          <Form layout="vertical">
            <Form.Item label="资源名称">
              <Input value={detailName} readOnly />
            </Form.Item>
            <Form.Item label="资源类型">
              <Input value={selectedResource ? `${selectedResource.kind} (${selectedResource.group}/${selectedResource.version})` : "-"} readOnly />
            </Form.Item>
            <Form.Item label="命名空间">
              <Input value={selectedResource?.namespaced ? namespace : "Cluster Scope"} readOnly />
            </Form.Item>
            <Form.Item label="YAML">
              <Input.TextArea value={detailYaml} onChange={(e) => setDetailYaml(e.target.value)} autoSize={{ minRows: 20, maxRows: 28 }} />
            </Form.Item>
          </Form>
        )}
      </Drawer>

      <Modal
        title="应用 YAML"
        open={applyOpen}
        width={980}
        confirmLoading={applyLoading}
        onCancel={() => setApplyOpen(false)}
        onOk={() => {
          if (!clusterId) {
            message.warning("请先选择集群");
            return;
          }
          setApplyLoading(true);
          void (async () => {
            try {
              await applyCr(clusterId, manifest);
              message.success("应用成功");
              setApplyOpen(false);
              await reload();
            } finally {
              setApplyLoading(false);
            }
          })();
        }}
      >
        <Input.TextArea value={manifest} onChange={(e) => setManifest(e.target.value)} autoSize={{ minRows: 20, maxRows: 28 }} />
      </Modal>
    </Card>
  );
}
