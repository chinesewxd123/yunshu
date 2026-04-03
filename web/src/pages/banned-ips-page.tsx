import { ExclamationCircleOutlined, ReloadOutlined } from "@ant-design/icons";
import { Button, Card, Popconfirm, Space, Table, message } from "antd";
import { useEffect, useState } from "react";
import { getBannedIPs, unbanIP, BannedIPItem } from "../services/admin";

export function BannedIPsPage() {
  const [list, setList] = useState<BannedIPItem[]>([]);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    void loadList();
  }, []);

  async function loadList() {
    setLoading(true);
    try {
      const res = await getBannedIPs();
      setList(res.list || []);
    } finally {
      setLoading(false);
    }
  }

  async function handleUnban(ip: string) {
    try {
      await unbanIP(ip);
      message.success("已解除封禁");
      void loadList();
    } catch (err) {
      // error handled by http interceptor
    }
  }

  return (
    <div>
      <Card className="table-card">
        <div className="toolbar">
          <Space>
            <Button icon={<ReloadOutlined />} onClick={() => void loadList()}>
              刷新
            </Button>
          </Space>
        </div>

        <Table
          rowKey={(r) => r.ip}
          loading={loading}
          dataSource={list}
          pagination={false}
          columns={[
            { title: "IP", dataIndex: "ip", key: "ip" },
            { title: "剩余封禁时间(秒)", dataIndex: "ttl_seconds", key: "ttl_seconds", width: 180 },
            {
              title: "操作",
              key: "action",
              width: 160,
              render: (_: unknown, record: BannedIPItem) => (
                <Space>
                  <Popconfirm
                    title={`确定解除 ${record.ip} 的封禁吗？`}
                    okText="解除"
                    cancelText="取消"
                    icon={<ExclamationCircleOutlined />}
                    onConfirm={() => void handleUnban(record.ip)}
                  >
                    <Button type="link">解除封禁</Button>
                  </Popconfirm>
                </Space>
              ),
            },
          ]}
        />
      </Card>
    </div>
  );
}

export default BannedIPsPage;
