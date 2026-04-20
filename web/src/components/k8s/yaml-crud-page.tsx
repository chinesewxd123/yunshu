import { DeleteOutlined, EyeOutlined, PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Collapse, Drawer, Input, Modal, Popconfirm, Select, Space, Table, Tabs, Typography, message } from "antd";
import type { ColumnsType } from "antd/es/table";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import YAML from "yaml";
import { getClusters } from "../../services/clusters";
import type { ClusterItem } from "../../services/clusters";

export type ClusterOption = { label: string; value: number; disabled?: boolean };
export type NamespaceOption = { label: string; value: string };

export type YamlCrudListArgs = {
  clusterId: number;
  namespace?: string;
  keyword?: string;
};

export type YamlCrudDetailArgs = {
  clusterId: number;
  namespace?: string;
  name: string;
};

export type YamlCrudApplyArgs = {
  clusterId: number;
  manifest: string;
};

export type YamlCrudDeleteArgs = {
  clusterId: number;
  namespace?: string;
  name: string;
};

export interface YamlCrudApi<TItem, TDetail> {
  list: (args: YamlCrudListArgs) => Promise<TItem[]>;
  detail: (args: YamlCrudDetailArgs) => Promise<TDetail>;
  apply?: (args: YamlCrudApplyArgs) => Promise<unknown>;
  remove?: (args: YamlCrudDeleteArgs) => Promise<unknown>;
}

/** 工具栏及创建流程回调使用的上下文 */
export type YamlCrudToolbarCtx = {
  clusterId?: number;
  namespace?: string;
  reload: () => void;
};

/** 创建抽屉打开时传给子组件（含关闭外层抽屉） */
export type YamlCrudCreateCtx = YamlCrudToolbarCtx & {
  closeCreateDrawer: () => void;
};

export interface YamlCrudPageProps<TItem extends { name: string }, TDetail extends { yaml: string }> {
  title: string;
  needNamespace?: boolean;
  namespaceOptions?: NamespaceOption[];
  onLoadNamespaces?: (clusterId: number) => Promise<NamespaceOption[]>;
  columns: ColumnsType<TItem>;
  api: YamlCrudApi<TItem, TDetail>;
  extraRowActions?: (record: TItem, ctx: { clusterId: number; namespace?: string; reload: () => void }) => React.ReactNode;
  detailExtra?: (detail: TDetail) => React.ReactNode;
  createTemplate?: (ctx: { namespace?: string }) => string;
  /** 点击「创建」打开右侧抽屉后调用（准备表单初始值等，如 prepareCreate） */
  onCreateDrawerOpen?: (ctx: YamlCrudCreateCtx) => void;
  /** 「表单创建」Tab 内容（与 Pod 页一致：与 YAML 同在一个创建抽屉内） */
  renderCreateFormTab?: (ctx: YamlCrudCreateCtx) => React.ReactNode;
  /** 集群/命名空间/搜索变化时回调，便于父组件同步 reload 引用 */
  onToolbarReady?: (ctx: YamlCrudToolbarCtx) => void;
  renderToolbarExtraRight?: (ctx: YamlCrudToolbarCtx) => React.ReactNode;
  renderDetail?: (detail: TDetail, ctx: { detailYaml: string; setDetailYaml: (next: string) => void }) => React.ReactNode;
  showEditButton?: boolean;
  confirmOverwrite?: boolean;
  disableMutations?: boolean;
  /** 操作列宽度，节点等页面操作较多时可加大 */
  actionColumnWidth?: number;
}

export function YamlCrudPage<TItem extends { name: string }, TDetail extends { yaml: string }>(props: YamlCrudPageProps<TItem, TDetail>) {
  const {
    title,
    needNamespace,
    columns,
    api,
    extraRowActions,
    onLoadNamespaces,
    detailExtra,
    createTemplate,
    onCreateDrawerOpen,
    renderCreateFormTab,
    onToolbarReady,
    renderToolbarExtraRight,
    renderDetail,
    showEditButton = true,
    confirmOverwrite = true,
    disableMutations = false,
    actionColumnWidth = 260,
  } = props;
  const [clusters, setClusters] = useState<ClusterItem[]>([]);
  const [clusterId, setClusterId] = useState<number | undefined>(undefined);
  const [namespace, setNamespace] = useState<string | undefined>(needNamespace ? "default" : undefined);
  const [namespaceOptions, setNamespaceOptions] = useState<NamespaceOption[]>(props.namespaceOptions ?? []);
  const [keyword, setKeyword] = useState<string>("");
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState<TItem[]>([]);

  const [detailOpen, setDetailOpen] = useState(false);
  const [detailLoading, setDetailLoading] = useState(false);
  const [detail, setDetail] = useState<TDetail | null>(null);
  const [detailName, setDetailName] = useState<string>("");
  const [yamlPanelActive, setYamlPanelActive] = useState(false);
  const [detailYaml, setDetailYaml] = useState<string>("");
  const [detailSaving, setDetailSaving] = useState(false);

  const [createDrawerOpen, setCreateDrawerOpen] = useState(false);
  const [applyLoading, setApplyLoading] = useState(false);
  const [manifest, setManifest] = useState<string>("");

  const closeCreateDrawer = useCallback(() => setCreateDrawerOpen(false), []);
  const clusterOptions: ClusterOption[] = useMemo(
    () =>
      clusters.map((c) => ({
        label: c.status === 1 ? c.name : `${c.name}（已停用）`,
        value: c.id,
        disabled: c.status !== 1,
      })),
    [clusters],
  );

  async function loadClusters() {
    const res = await getClusters({ page: 1, page_size: 200 });
    setClusters(res.list ?? []);
    if (!clusterId) {
      const first = (res.list ?? []).find((c) => c.status === 1);
      if (first) setClusterId(first.id);
    }
  }

  async function loadNamespaces(cid: number) {
    if (!needNamespace) return;
    const loader = onLoadNamespaces;
    if (!loader) return;
    const opts = await loader(cid);
    setNamespaceOptions(opts);
    if (!namespace || !opts.some((o) => o.value === namespace)) {
      const first = opts[0]?.value ?? "default";
      setNamespace(first);
    }
  }

  async function reload(overrideKeyword?: string) {
    if (!clusterId) return;
    if (needNamespace && !namespace) return;
    setLoading(true);
    try {
      const effectiveKeyword = (overrideKeyword ?? keyword).trim();
      const list = await api.list({ clusterId, namespace, keyword: effectiveKeyword || undefined });
      setData(list ?? []);
    } finally {
      setLoading(false);
    }
  }

  const onToolbarReadyRef = useRef(onToolbarReady);
  onToolbarReadyRef.current = onToolbarReady;

  useEffect(() => {
    const fn = onToolbarReadyRef.current;
    if (!fn) return;
    fn({
      clusterId,
      namespace,
      reload: () => void reload(),
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [clusterId, namespace, keyword]);

  useEffect(() => {
    void loadClusters();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (!clusterId) return;
    void (async () => {
      try {
        await loadNamespaces(clusterId);
      } catch (e) {
        message.error(e instanceof Error ? e.message : "加载命名空间失败");
      }
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [clusterId]);

  useEffect(() => {
    void reload();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [clusterId, namespace]);

  useEffect(() => {
    if (!clusterId) return;
    if (needNamespace && !namespace) return;
    const timer = window.setInterval(() => {
      void reload();
    }, 10000);
    return () => window.clearInterval(timer);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [clusterId, namespace, needNamespace]);

  const actionCol: ColumnsType<TItem>[number] = {
    title: "操作",
    key: "action",
    width: actionColumnWidth,
    fixed: "right",
    render: (_: unknown, record: TItem) => (
      <Space>
        <Button
          type="link"
          icon={<EyeOutlined />}
          onClick={() => {
            if (!clusterId) return;
            // 清理可能残留的 confirm/info 遮罩，避免遮住详情弹窗
            Modal.destroyAll();
            message.loading({ content: "正在加载详情...", key: "yaml-crud-detail", duration: 0 });
            setDetailOpen(true);
            // 详情默认进入可编辑态，避免“看起来只能查看”的误解。
            setYamlPanelActive(true);
            setDetailName(record.name);
            setDetail(null);
            setDetailLoading(true);
            void (async () => {
              try {
                const d = await api.detail({ clusterId, namespace, name: record.name });
                setDetail(d);
                setDetailYaml(d.yaml ?? "");
              } catch (e) {
                const status = (e as any)?.response?.status;
                if (status === 403) {
                  message.error({ content: "无访问权限", key: "forbidden" });
                } else {
                message.error({
                  content: e instanceof Error ? e.message : "加载详情失败",
                  key: "yaml-crud-detail",
                });
                }
              } finally {
                setDetailLoading(false);
                message.destroy("yaml-crud-detail");
              }
            })();
          }}
        >
          详情
        </Button>
        {extraRowActions?.(record, { clusterId: clusterId ?? 0, namespace, reload })}
        {!disableMutations && api.remove ? (
          <Popconfirm
            title={`确认删除 ${record.name} 吗？`}
            onConfirm={() => {
              if (!clusterId) return;
              void (async () => {
                await api.remove?.({ clusterId, namespace, name: record.name });
                message.success("删除成功");
                await reload();
              })();
            }}
          >
            <Button danger type="link" icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        ) : null}
      </Space>
    ),
  };

  const toolbarCtx: YamlCrudToolbarCtx = {
    clusterId,
    namespace,
    reload: () => void reload(),
  };

  const createCtx: YamlCrudCreateCtx = {
    ...toolbarCtx,
    closeCreateDrawer,
  };

  const hasFormTab = Boolean(renderCreateFormTab);
  const hasYamlTab = Boolean(api.apply);
  const canOpenCreate = !disableMutations && (hasFormTab || hasYamlTab);

  function handleOpenCreateDrawer() {
    if (!clusterId) return;
    setManifest(createTemplate ? createTemplate({ namespace }) : "");
    setCreateDrawerOpen(true);
    onCreateDrawerOpen?.(createCtx);
  }

  async function submitCreateYaml() {
    const applyFn = api.apply;
    if (!clusterId || !applyFn) return;

    const doApply = async () => {
      setApplyLoading(true);
      try {
        await applyFn({ clusterId, manifest });
        message.success("应用成功");
        closeCreateDrawer();
        await reload();
      } finally {
        setApplyLoading(false);
      }
    };

    const confirmAndApply = async () => {
      if (!confirmOverwrite) {
        await doApply();
        return;
      }
      if (!manifest.trim()) {
        await doApply();
        return;
      }

      try {
        const docs = YAML.parseAllDocuments(manifest);
        let targetName: string | undefined;
        let targetNamespace: string | undefined;

        for (const doc of docs) {
          const v: any = doc.toJSON();
          const md = v?.metadata;
          const n = md?.name;
          if (n) {
            targetName = String(n);
            targetNamespace = md?.namespace ? String(md.namespace) : namespace ?? "default";
            break;
          }
        }

        if (!targetName) {
          await doApply();
          return;
        }

        let exists = false;
        try {
          await api.detail({ clusterId, namespace: targetNamespace, name: targetName });
          exists = true;
        } catch {
          exists = false;
        }

        if (!exists) {
          await doApply();
          return;
        }

        Modal.confirm({
          title: "检测到同名对象",
          content: `${targetNamespace}/${targetName} 已存在，确认覆盖吗？（apply 会直接更新）`,
          okText: "覆盖并应用",
          cancelText: "取消",
          onOk: async () => {
            await doApply();
          },
        });
        return;
      } catch {
        await doApply();
        return;
      }
    };

    void confirmAndApply();
  }

  const yamlCreatePanel = (
    <Space direction="vertical" style={{ width: "100%" }} size="middle">
      <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
        支持直接粘贴 Kubernetes YAML（底层使用 Kom SDK 的 apply）。打开时已预填模板（若有），可清空后自行编写。
      </Typography.Paragraph>
      <Space wrap>
        {createTemplate ? (
          <Button size="small" onClick={() => setManifest(createTemplate({ namespace }))}>
            填入模板
          </Button>
        ) : null}
        <Button size="small" onClick={() => setManifest("")}>
          清空内容
        </Button>
      </Space>
      <Input.TextArea
        value={manifest}
        onChange={(e) => setManifest(e.target.value)}
        autoSize={{ minRows: 14, maxRows: 26 }}
        placeholder={`apiVersion: v1
kind: ...
metadata:
  name: ...`}
      />
      <Button type="primary" loading={applyLoading} onClick={() => void submitCreateYaml()}>
        创建
      </Button>
      <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }}>
        提示：如果要修改现有对象，建议保留 name/namespace 并直接 apply。
      </Typography.Paragraph>
    </Space>
  );

  return (
    <Card className="table-card" title={title}>
      <div style={{ display: "flex", gap: 12, alignItems: "center", justifyContent: "space-between", marginBottom: 12, flexWrap: "wrap" }}>
        <Space wrap>
          <Select
            placeholder="选择集群"
            style={{ minWidth: 240 }}
            value={clusterId}
            onChange={setClusterId}
            options={clusterOptions}
          />
          {needNamespace ? (
            <Select
              placeholder="命名空间"
              style={{ minWidth: 200 }}
              value={namespace}
              onChange={setNamespace}
              options={namespaceOptions}
              showSearch
              optionFilterProp="label"
            />
          ) : null}
          <Input.Search
            allowClear
            placeholder="搜索名称"
            style={{ width: 260 }}
            onSearch={(v) => {
              setKeyword(v);
              void reload(v);
            }}
          />
        </Space>
        <Space wrap>
          {renderToolbarExtraRight ? renderToolbarExtraRight(toolbarCtx) : null}
          {canOpenCreate ? (
            <Button type="primary" icon={<PlusOutlined />} disabled={!clusterId} onClick={handleOpenCreateDrawer}>
              创建
            </Button>
          ) : null}
          <Button icon={<ReloadOutlined />} onClick={() => void reload()}>
            刷新
          </Button>
        </Space>
      </div>

      <Table
        rowKey={(r) => (r as any).name}
        loading={loading}
        dataSource={data}
        pagination={{ pageSize: 10, showSizeChanger: true, pageSizeOptions: [10, 20, 50, 100], showQuickJumper: true }}
        columns={[...columns, actionCol]}
        scroll={{ x: "max-content" }}
      />

      {!disableMutations && (hasFormTab || hasYamlTab) ? (
        <Drawer
          title={
            <Space direction="vertical" size={0}>
              <span>创建</span>
              {needNamespace ? (
                <Typography.Text type="secondary" style={{ fontSize: 13, fontWeight: "normal" }}>
                  目标命名空间：{namespace ?? "—"}
                </Typography.Text>
              ) : null}
            </Space>
          }
          placement="right"
          width={960}
          open={createDrawerOpen}
          onClose={closeCreateDrawer}
          destroyOnClose
          maskClosable={false}
          styles={{ body: { paddingBottom: 24 } }}
          zIndex={1200}
          extra={<Button onClick={closeCreateDrawer}>取消</Button>}
        >
          {hasFormTab && hasYamlTab ? (
            <Tabs
              defaultActiveKey="form"
              items={[
                { key: "form", label: "表单创建", children: renderCreateFormTab?.(createCtx) },
                { key: "yaml", label: "YAML 创建", children: yamlCreatePanel },
              ]}
            />
          ) : hasYamlTab ? (
            yamlCreatePanel
          ) : (
            renderCreateFormTab?.(createCtx)
          )}
        </Drawer>
      ) : null}

      <Drawer
        title={`${title} - 详情${detailName ? `：${detailName}` : ""}`}
        open={detailOpen}
        onClose={() => {
          setDetailOpen(false);
          setYamlPanelActive(false);
          setDetailYaml("");
        }}
        destroyOnClose
        width={920}
        zIndex={1300}
        className="detail-edit-drawer"
        extra={
          !disableMutations && api.apply && detail ? (
            <Button
              type="primary"
              loading={detailSaving}
              onClick={() => {
                if (!clusterId || !api.apply || !detailName) return;
                setDetailSaving(true);
                void (async () => {
                  try {
                    await api.apply?.({ clusterId, manifest: detailYaml });
                    message.success("详情修改已保存");
                    await reload();
                    const latest = await api.detail({ clusterId, namespace, name: detailName });
                    setDetail(latest);
                    setDetailYaml(latest.yaml ?? "");
                  } finally {
                    setDetailSaving(false);
                  }
                })();
              }}
            >
              保存修改
            </Button>
          ) : null
        }
      >
        {detailLoading ? (
          <Typography.Text type="secondary">加载中...</Typography.Text>
        ) : detail ? (
          <Space direction="vertical" style={{ width: "100%" }} size="middle">
            {detailExtra?.(detail)}
            {renderDetail ? renderDetail(detail, { detailYaml, setDetailYaml }) : null}
            {!renderDetail && !detailExtra ? (
              <Typography.Paragraph copyable style={{ marginBottom: 0, whiteSpace: "pre-wrap" }}>
                {detail.yaml}
              </Typography.Paragraph>
            ) : null}
            <Collapse
              defaultActiveKey={["yaml"]}
              onChange={(keys) => {
                const arr = Array.isArray(keys) ? keys : [keys];
                setYamlPanelActive(arr.includes("yaml"));
              }}
              items={[
                {
                  key: "yaml",
                  label: "编辑 YAML",
                  children: yamlPanelActive ? (
                    <Input.TextArea
                      value={detailYaml}
                      onChange={(e) => setDetailYaml(e.target.value)}
                      autoSize={{ minRows: 18, maxRows: 28 }}
                      className="detail-edit-form"
                    />
                  ) : (
                    <Typography.Text type="secondary">展开后加载 YAML 编辑器</Typography.Text>
                  ),
                },
              ]}
            />
          </Space>
        ) : (
          <Typography.Text type="secondary">暂无数据</Typography.Text>
        )}
      </Drawer>

    </Card>
  );
}

