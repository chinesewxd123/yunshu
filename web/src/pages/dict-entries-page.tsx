import { DeleteOutlined, EditOutlined, PlusOutlined, ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Form, Input, InputNumber, Modal, Popconfirm, Select, Space, Table, Tag, Tooltip, Typography, message } from "antd";
import { useEffect, useMemo, useState } from "react";
import { PROJECT_DICT_TYPE_OPTIONS } from "../constants/dict-types";
import { createDictEntry, deleteDictEntry, getDictEntries, updateDictEntry, type DictEntryItem, type DictPayload } from "../services/dict";
import { formatDateTime } from "../utils/format";

const defaultQuery = { keyword: "", dict_type: "", page: 1, page_size: 10 };

export function DictEntriesPage() {
  const [list, setList] = useState<DictEntryItem[]>([]);
  const [total, setTotal] = useState(0);
  const [query, setQuery] = useState(defaultQuery);
  const [loading, setLoading] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [current, setCurrent] = useState<DictEntryItem | null>(null);
  const [open, setOpen] = useState(false);
  const [form] = Form.useForm<DictPayload>();
  const formDictType = Form.useWatch("dict_type", form) as string | undefined;
  const autoSortLocked = !current && String(formDictType || "") === "alert_promql_label_key";
  const isAlertLabelGovernedDictType = String(formDictType || "").trim() === "alert_promql_label_key";
  const dictTypeOptions = useMemo(() => {
    const fromList = Array.from(new Set(list.map((item) => String(item.dict_type || "").trim()).filter(Boolean))).map((v) => ({
      label: `${v}（现有）`,
      value: v,
    }));
    const merged = [...PROJECT_DICT_TYPE_OPTIONS];
    for (const it of fromList) {
      if (!merged.some((m) => m.value === it.value)) merged.push(it);
    }
    if (current?.dict_type && !merged.some((m) => m.value === current.dict_type)) {
      merged.push({ label: `${current.dict_type}（当前）`, value: current.dict_type });
    }
    return merged.sort((a, b) => String(a.value).localeCompare(String(b.value)));
  }, [list, current?.dict_type]);

  useEffect(() => {
    void loadData(query);
  }, [query]);

  useEffect(() => {
    if (!open || current || !formDictType) return;
    let cancelled = false;
    void (async () => {
      try {
        const result = await getDictEntries({ dict_type: formDictType, page: 1, page_size: 500 });
        if (cancelled) return;
        const maxSort = (result.list ?? []).reduce((max, it) => Math.max(max, Number(it.sort || 0)), 0);
        const currentSort = form.getFieldValue("sort");
        if (currentSort == null || Number.isNaN(Number(currentSort)) || autoSortLocked) {
          form.setFieldValue("sort", maxSort + 1);
        }
      } catch {
        if (!cancelled && autoSortLocked) {
          form.setFieldValue("sort", 1);
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [open, current, formDictType, form, autoSortLocked]);

  async function loadData(next = query) {
    setLoading(true);
    try {
      const result = await getDictEntries(next);
      setList(result.list);
      setTotal(result.total);
    } finally {
      setLoading(false);
    }
  }

  function openCreate() {
    setCurrent(null);
    form.resetFields();
    form.setFieldsValue({ status: 1, sort: 0 });
    setOpen(true);
  }

  function openEdit(record: DictEntryItem) {
    setCurrent(record);
    form.setFieldsValue({
      dict_type: record.dict_type,
      label: record.label,
      value: record.value,
      sort: record.sort,
      status: record.status,
      remark: record.remark,
    });
    setOpen(true);
  }

  async function submit() {
    const values = await form.validateFields();
    setSubmitting(true);
    try {
      if (current) {
        await updateDictEntry(current.id, values);
        message.success("字典条目已更新");
      } else {
        await createDictEntry(values);
        message.success("字典条目创建成功");
      }
      setOpen(false);
      await loadData();
    } finally {
      setSubmitting(false);
    }
  }

  async function remove(record: DictEntryItem) {
    await deleteDictEntry(record.id);
    message.success(`已删除条目 ${record.label}`);
    await loadData();
  }

  return (
    <Card className="table-card">
      <div className="toolbar">
        <Space wrap>
          <Input.Search
            allowClear
            placeholder="搜索标签/值/备注"
            style={{ width: 260 }}
            onSearch={(keyword) => setQuery((prev) => ({ ...prev, keyword, page: 1 }))}
          />
          <Select
            allowClear
            showSearch
            placeholder="按字典类型筛选"
            style={{ width: 220 }}
            value={query.dict_type}
            options={dictTypeOptions}
            filterOption={(input, option) => String(option?.label ?? "").toLowerCase().includes(input.toLowerCase())}
            onChange={(v) => setQuery((prev) => ({ ...prev, dict_type: String(v || ""), page: 1 }))}
          />
        </Space>
        <div className="toolbar__actions">
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>
            新建条目
          </Button>
          <Button icon={<ReloadOutlined />} onClick={() => void loadData()}>
            刷新
          </Button>
        </div>
      </div>

      <Table
        rowKey="id"
        loading={loading}
        dataSource={list}
        pagination={{
          current: query.page,
          pageSize: query.page_size,
          total,
          showSizeChanger: true,
          pageSizeOptions: [10, 20, 50, 100],
          showQuickJumper: true,
          onChange: (page, pageSize) => setQuery((prev) => ({ ...prev, page, page_size: pageSize })),
        }}
        scroll={{ x: 1100 }}
        columns={[
          { title: "ID", dataIndex: "id", width: 80 },
          { title: "字典类型", dataIndex: "dict_type", width: 200, render: (v: string) => <Tag color="geekblue">{v}</Tag> },
          { title: "标签", dataIndex: "label", width: 200, ellipsis: true },
          {
            title: "值",
            dataIndex: "value",
            width: 280,
            render: (v: string) => (
              <Tooltip placement="topLeft" title={v}>
                <Typography.Text ellipsis style={{ maxWidth: 260, display: "block" }}>
                  {v || "—"}
                </Typography.Text>
              </Tooltip>
            ),
          },
          { title: "排序", dataIndex: "sort", width: 80 },
          { title: "状态", dataIndex: "status", width: 90, render: (v: number) => (v === 1 ? <Tag color="success">启用</Tag> : <Tag>停用</Tag>) },
          { title: "备注", dataIndex: "remark", render: (v?: string) => v || "-" },
          { title: "更新时间", dataIndex: "updated_at", width: 180, render: formatDateTime },
          {
            title: "操作",
            key: "action",
            width: 180,
            render: (_: unknown, record: DictEntryItem) => (
              <Space>
                <Button type="link" icon={<EditOutlined />} onClick={() => openEdit(record)}>
                  编辑
                </Button>
                <Popconfirm title="确认删除该条目吗？" onConfirm={() => void remove(record)}>
                  <Button type="link" danger icon={<DeleteOutlined />}>
                    删除
                  </Button>
                </Popconfirm>
              </Space>
            ),
          },
        ]}
      />

      <Modal
        title={current ? `编辑字典条目 #${current.id}` : "新建字典条目"}
        open={open}
        onCancel={() => setOpen(false)}
        onOk={() => void submit()}
        confirmLoading={submitting}
        destroyOnClose
      >
        <Form form={form} layout="vertical" initialValues={{ status: 1, sort: 0 }}>
          <Form.Item
            label="字典类型"
            name="dict_type"
            rules={[{ required: true, message: "请选择字典类型" }]}
            extra={
              <>
                <Typography.Paragraph type="secondary" style={{ marginBottom: 0, fontSize: 12 }}>
                  字典类型按项目规范统一管理；新增配置时请优先从下拉选择，避免同义多名导致代码读取不到。
                </Typography.Paragraph>
                {isAlertLabelGovernedDictType ? (
                  <Typography.Paragraph type="warning" style={{ marginBottom: 0, fontSize: 12 }}>
                    告警标签键已统一以 `alert_promql_label_key` 为唯一来源，策略匹配与静默匹配都读取该类型。
                  </Typography.Paragraph>
                ) : null}
              </>
            }
          >
            <Select
              showSearch
              placeholder="请选择字典类型"
              options={dictTypeOptions}
              filterOption={(input, option) => String(option?.label ?? "").toLowerCase().includes(input.toLowerCase())}
            />
          </Form.Item>
          <Form.Item label="标签" name="label" rules={[{ required: true, message: "请输入标签" }]}>
            <Input placeholder="例如 启用" />
          </Form.Item>
          <Form.Item
            label="值"
            name="value"
            rules={[{ required: true, message: "请输入值" }]}
            extra="支持完整 kubeconfig（含证书）；库字段为 MEDIUMTEXT，服务端上限约 16MB。极长内容建议仍用集群管理直接粘贴。"
          >
            <Input.TextArea rows={4} placeholder="例如 1 / GET / 多行 yaml" style={{ fontFamily: "ui-monospace, Menlo, Consolas, monospace" }} />
          </Form.Item>
          <Space style={{ width: "100%" }} size="middle">
            <Form.Item
              label="排序"
              name="sort"
              style={{ width: 140, marginBottom: 0 }}
              extra={autoSortLocked ? "该类型自动按当前最大序号+1分配" : undefined}
            >
              <InputNumber min={0} style={{ width: "100%" }} disabled={autoSortLocked} />
            </Form.Item>
            <Form.Item label="状态" name="status" style={{ width: 160, marginBottom: 0 }}>
              <Select options={[{ label: "启用", value: 1 }, { label: "停用", value: 0 }]} />
            </Form.Item>
          </Space>
          <Form.Item label="备注" name="remark" style={{ marginTop: 12 }}>
            <Input.TextArea rows={3} />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
