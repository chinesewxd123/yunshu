import { CheckCircleOutlined, CloseCircleOutlined, ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Form, Input, Modal, Popconfirm, Radio, Select, Space, Table, Tag, Typography, message } from "antd";
import { useEffect, useState } from "react";
import { getRegistrations, reviewRegistration } from "../services/registrations";
import type { RegistrationRequestItem } from "../services/registrations";

const defaultQuery = { keyword: "", page: 1, page_size: 10 };

export function RegistrationsPage() {
  const [list, setList] = useState<RegistrationRequestItem[]>([]);
  const [total, setTotal] = useState(0);
  const [query, setQuery] = useState(defaultQuery);
  const [loading, setLoading] = useState(false);
  const [reviewOpen, setReviewOpen] = useState(false);
  const [reviewTarget, setReviewTarget] = useState<RegistrationRequestItem | null>(null);
  const [reviewForm] = Form.useForm();
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    void loadList(query);
  }, [query]);

  async function loadList(nextQuery = query) {
    setLoading(true);
    try {
      const result = await getRegistrations(nextQuery);
      setList(result.list);
      setTotal(result.total);
    } finally {
      setLoading(false);
    }
  }

  function openReview(record: RegistrationRequestItem) {
    setReviewTarget(record);
    reviewForm.resetFields();
    setReviewOpen(true);
  }

  async function handleReview() {
    if (!reviewTarget) return;
    const values = await reviewForm.validateFields();
    setSubmitting(true);
    try {
      await reviewRegistration(reviewTarget.id, values);
      message.success("审核完成");
      setReviewOpen(false);
      void loadList();
    } finally {
      setSubmitting(false);
    }
  }

  function statusTag(status: number) {
    switch (status) {
      case 0:
        return <Tag color="warning">待审核</Tag>;
      case 1:
        return <Tag color="success">已通过</Tag>;
      case 2:
        return <Tag color="error">已拒绝</Tag>;
      default:
        return <Tag>未知</Tag>;
    }
  }

  return (
    <div>
      <Card className="table-card">
        <div className="toolbar">
          <Space>
            <Select
              allowClear
              placeholder="状态筛选"
              style={{ width: 120 }}
              onChange={(value) =>
                setQuery((prev) => ({
                  ...prev,
                  status: value ?? undefined,
                  page: 1,
                }))
              }
              options={[
                { label: "待审核", value: 0 },
                { label: "已通过", value: 1 },
                { label: "已拒绝", value: 2 },
              ]}
            />
          </Space>
          <Input.Search
            allowClear
            placeholder="搜索用户名/邮箱"
            style={{ width: 240 }}
            onSearch={(keyword) =>
              setQuery((prev) => ({ ...prev, keyword, page: 1 }))
            }
          />
          <div className="toolbar__actions">
            <Button icon={<ReloadOutlined />} onClick={() => void loadList()}>
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
            onChange: (page, pageSize) =>
              setQuery((prev) => ({ ...prev, page, page_size: pageSize })),
          }}
          columns={[
            { title: "ID", dataIndex: "id", width: 70 },
            { title: "用户名", dataIndex: "username" },
            { title: "邮箱", dataIndex: "email" },
            { title: "昵称", dataIndex: "nickname" },
            { title: "状态", dataIndex: "status", render: statusTag },
            { title: "申请时间", dataIndex: "created_at" },
            { title: "审核人ID", dataIndex: "reviewer_id", render: (v?: number) => v || "-" },
            { title: "审核备注", dataIndex: "review_comment", render: (v?: string) => v || "-" },
            {
              title: "操作",
              key: "action",
              width: 120,
              render: (_: unknown, record: RegistrationRequestItem) =>
                record.status === 0 ? (
                  <Button type="link" onClick={() => openReview(record)}>
                    审核
                  </Button>
                ) : (
                  <span style={{ color: "#999" }}>已完成</span>
                ),
            },
          ]}
        />
      </Card>

      <Modal
        title={`审核注册申请 #${reviewTarget?.id}`}
        open={reviewOpen}
        onCancel={() => setReviewOpen(false)}
        onOk={() => void handleReview()}
        confirmLoading={submitting}
        destroyOnClose
        width={500}
      >
        <Form form={reviewForm} layout="vertical" initialValues={{ status: 1 }}>
          <Typography.Text strong>申请人信息</Typography.Text>
          <div style={{ marginBottom: 16, color: "#666" }}>
            用户名：{reviewTarget?.username} / 邮箱：{reviewTarget?.email} / 昵称：{reviewTarget?.nickname}
          </div>
          <Form.Item name="status" label="审核结果" rules={[{ required: true }]}>
            <Radio.Group>
              <Radio.Button value={1}>
                <CheckCircleOutlined /> 通过
              </Radio.Button>
              <Radio.Button value={2}>
                <CloseCircleOutlined /> 拒绝
              </Radio.Button>
            </Radio.Group>
          </Form.Item>
          <Form.Item name="comment" label="审核备注">
            <Input.TextArea rows={3} placeholder="请输入审核备注（选填）" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}
