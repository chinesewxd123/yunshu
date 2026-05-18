import { CloudUploadOutlined, DeleteOutlined, EditOutlined, LinkOutlined, PlusOutlined, ReloadOutlined, ThunderboltOutlined } from "@ant-design/icons";
import {
  Alert,
  Button,
  Card,
  Drawer,
  Form,
  Input,
  InputNumber,
  Popconfirm,
  Select,
  Space,
  Switch,
  Table,
  Tabs,
  Tag,
  Typography,
  message,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import { useCallback, useEffect, useState } from "react";
import { Link } from "react-router-dom";
import {
  checkMysqlRemoteBackup,
  createMysqlBackupInstance,
  deleteMysqlBackupInstance,
  listMysqlBackupInstances,
  listMysqlBackupJobs,
  pingMysqlBackupInstance,
  presignMysqlBackupJob,
  runMysqlBackup,
  updateMysqlBackupInstance,
  type MysqlBackupInstance,
  type MysqlBackupInstancePayload,
  type MysqlBackupJob,
} from "../services/mysql-backup";
import { getProjectServers, getProjects, type ProjectItem, type ServerItem } from "../services/projects";
import { formatDateTime } from "../utils/format";

export function MysqlBackupPage() {
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [servers, setServers] = useState<ServerItem[]>([]);
  const [instances, setInstances] = useState<MysqlBackupInstance[]>([]);
  const [jobs, setJobs] = useState<MysqlBackupJob[]>([]);
  const [loading, setLoading] = useState(false);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [editing, setEditing] = useState<MysqlBackupInstance | null>(null);
  const [form] = Form.useForm<MysqlBackupInstancePayload & { mysql_password?: string }>();

  useEffect(() => {
    void getProjects({ page: 1, page_size: 200 }).then((r) => setProjects(r.list || []));
  }, []);

  const loadServers = useCallback(async () => {
    if (!projectId) return;
    const res = await getProjectServers(projectId, { page: 1, page_size: 500 });
    setServers(res.list || []);
  }, [projectId]);

  const load = useCallback(async () => {
    if (!projectId) return;
    setLoading(true);
    try {
      const [instRes, jobRes] = await Promise.all([
        listMysqlBackupInstances(projectId, { page: 1, page_size: 100 }),
        listMysqlBackupJobs(projectId, { page: 1, page_size: 50 }),
      ]);
      setInstances(instRes.list || []);
      setJobs(jobRes.list || []);
    } catch (e) {
      message.error(String(e));
    } finally {
      setLoading(false);
    }
  }, [projectId]);

  useEffect(() => {
    void loadServers();
    void load();
  }, [loadServers, load]);

  const openCreate = () => {
    setEditing(null);
    form.setFieldsValue({
      name: "",
      mysql_host: "127.0.0.1",
      mysql_port: 3306,
      mysql_user: "root",
      mysql_password: "",
      backup_mode: "mysqldump",
      backup_scope: "all",
      enabled: true,
      schedule_enabled: false,
      cron_spec: "",
      database_name: "",
      table_name: "",
      database_names: "",
      remote_data_dir: "/export/servers/data/mybackup/my3306/xtrabackup/data",
      remote_log_dir: "/export/servers/data/mybackup/my3306/xtrabackup/log",
    });
    setDrawerOpen(true);
  };

  const openEdit = (row: MysqlBackupInstance) => {
    setEditing(row);
    form.setFieldsValue({
      name: row.name,
      server_id: row.server_id,
      mysql_host: row.mysql_host,
      mysql_port: row.mysql_port,
      mysql_user: row.mysql_user,
      backup_mode: row.backup_mode,
      backup_scope: row.backup_scope || "all",
      enabled: row.enabled,
      schedule_enabled: row.schedule_enabled,
      cron_spec: row.cron_spec,
      database_name: row.database_name,
      table_name: row.table_name,
      database_names: row.database_names,
      remote_data_dir: row.remote_data_dir,
      remote_log_dir: row.remote_log_dir,
    });
    setDrawerOpen(true);
  };

  const onSave = async () => {
    if (!projectId) return;
    const v = await form.validateFields();
    try {
      if (editing) {
        await updateMysqlBackupInstance(projectId, editing.id, v);
        message.success("已更新");
      } else {
        await createMysqlBackupInstance(projectId, v);
        message.success("已创建");
      }
      setDrawerOpen(false);
      void load();
    } catch (e) {
      message.error(String(e));
    }
  };

  const instanceColumns: ColumnsType<MysqlBackupInstance> = [
    { title: "名称", dataIndex: "name", width: 140 },
    {
      title: "服务器",
      render: (_, r) => r.server_name || `#${r.server_id}`,
      width: 160,
    },
    { title: "MySQL", render: (_, r) => `${r.mysql_user}@${r.mysql_host}:${r.mysql_port}`, ellipsis: true },
    {
      title: "范围",
      width: 120,
      render: (_, r) => {
        if (r.backup_mode === "remote_check") return <Tag>全量</Tag>;
        const s = r.backup_scope || "all";
        if (s === "table") return <Tag color="cyan">{r.database_name}.{r.table_name}</Tag>;
        if (s === "database") return <Tag color="geekblue">{r.database_name || r.database_names || "单库"}</Tag>;
        return <Tag>全部库</Tag>;
      },
    },
    {
      title: "模式",
      dataIndex: "backup_mode",
      width: 110,
      render: (m: string) => (m === "remote_check" ? <Tag color="purple">远端检查</Tag> : <Tag color="blue">mysqldump</Tag>),
    },
    {
      title: "定时",
      width: 100,
      render: (_, r) =>
        r.schedule_enabled && r.cron_spec ? (
          <Tag color="processing" title={r.cron_spec}>
            Cron
          </Tag>
        ) : (
          <Tag>手动</Tag>
        ),
    },
    {
      title: "启用",
      dataIndex: "enabled",
      width: 72,
      render: (v: boolean) => (v ? <Tag color="success">是</Tag> : <Tag>否</Tag>),
    },
    {
      title: "操作",
      key: "actions",
      fixed: "right",
      width: 320,
      render: (_, row) => (
        <Space wrap>
          <Button size="small" onClick={() => void pingMysqlBackupInstance(projectId!, row.id).then((r) => message.info(r.message))}>
            Ping
          </Button>
          {row.backup_mode === "remote_check" ? (
            <Button
              size="small"
              onClick={() =>
                void checkMysqlRemoteBackup(projectId!, row.id).then((r) => message.info(r.message || (r.ok ? "检查通过" : "检查失败")))
              }
            >
              检查备份
            </Button>
          ) : null}
          <Button
            size="small"
            type="primary"
            icon={<ThunderboltOutlined />}
            onClick={() =>
              void runMysqlBackup(projectId!, row.id)
                .then(() => {
                  message.success("备份任务已完成");
                  void load();
                })
                .catch((e) => message.error(String(e)))
            }
          >
            执行备份
          </Button>
          <Button size="small" icon={<EditOutlined />} onClick={() => openEdit(row)} />
          <Popconfirm title="确定删除？" onConfirm={() => void deleteMysqlBackupInstance(projectId!, row.id).then(() => load())}>
            <Button size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ];

  const jobColumns: ColumnsType<MysqlBackupJob> = [
    { title: "ID", dataIndex: "id", width: 64 },
    { title: "实例", dataIndex: "instance_id", width: 72 },
    {
      title: "触发",
      dataIndex: "trigger_type",
      width: 72,
      render: (t?: string) => (t === "scheduled" ? <Tag color="blue">定时</Tag> : <Tag>手动</Tag>),
    },
    {
      title: "范围",
      width: 120,
      render: (_, r) => {
        if (r.backup_scope === "table") return `${r.database_name}.${r.table_name}`;
        if (r.backup_scope === "database") return r.database_name || "-";
        return r.backup_scope === "all" ? "全部库" : "-";
      },
    },
    {
      title: "状态",
      dataIndex: "status",
      width: 90,
      render: (s: string) => {
        const color = s === "success" ? "success" : s === "failed" ? "error" : "processing";
        return <Tag color={color}>{s}</Tag>;
      },
    },
    {
      title: "大小",
      dataIndex: "file_size",
      width: 100,
      render: (n?: number) => (n ? `${(n / 1024 / 1024).toFixed(2)} MiB` : "-"),
    },
    { title: "MinIO 对象", dataIndex: "minio_object", ellipsis: true },
    { title: "完成时间", dataIndex: "finished_at", width: 168, render: (v?: string) => formatDateTime(v) },
    {
      title: "",
      width: 100,
      render: (_, row) =>
        row.status === "success" && row.minio_object ? (
          <Button
            size="small"
            icon={<LinkOutlined />}
            onClick={() =>
              void presignMysqlBackupJob(projectId!, row.id).then((r) => {
                window.open(r.url, "_blank");
              })
            }
          >
            下载
          </Button>
        ) : null,
    },
  ];

  return (
    <Card
      className="table-card"
      title="MySQL 备份"
      extra={
        <Space>
          <Select
            style={{ width: 220 }}
            placeholder="选择项目"
            value={projectId}
            onChange={(v) => setProjectId(v)}
            options={projects.map((p) => ({ label: p.name, value: p.id }))}
          />
          <Button icon={<ReloadOutlined />} onClick={() => void load()} disabled={!projectId}>
            刷新
          </Button>
          <Button type="primary" icon={<PlusOutlined />} onClick={openCreate} disabled={!projectId}>
            新建实例
          </Button>
        </Space>
      }
    >
      <Alert
        type="info"
        showIcon
        style={{ marginBottom: 12 }}
        message={
          <span>
            复用项目内服务器 SSH 凭据；归档至 MinIO 请在 <Link to="/runtime-config">运行期配置</Link> 填写{" "}
            <Typography.Text code>minio_*</Typography.Text> 字典项。远端 xtrabackup 检查逻辑对齐 mysql_golang_tools。
          </span>
        }
      />
      <Tabs
        items={[
          {
            key: "instances",
            label: "备份实例",
            children: <Table rowKey="id" loading={loading} columns={instanceColumns} dataSource={instances} scroll={{ x: 1100 }} pagination={false} />,
          },
          {
            key: "jobs",
            label: "备份记录",
            children: <Table rowKey="id" loading={loading} columns={jobColumns} dataSource={jobs} scroll={{ x: 900 }} pagination={false} />,
          },
        ]}
      />
      <Drawer
        title={editing ? "编辑备份实例" : "新建备份实例"}
        width={520}
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        destroyOnClose
        extra={
          <Button type="primary" icon={<CloudUploadOutlined />} onClick={() => void onSave()}>
            保存
          </Button>
        }
      >
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="server_id" label="目标服务器（SSH）" rules={[{ required: true }]}>
            <Select options={servers.map((s) => ({ label: `${s.name} (${s.host})`, value: s.id }))} showSearch optionFilterProp="label" />
          </Form.Item>
          <Form.Item name="backup_mode" label="备份模式" rules={[{ required: true }]}>
            <Select
              options={[
                { label: "mysqldump + 上传 MinIO", value: "mysqldump" },
                { label: "远端 xtrabackup 检查 + 上传", value: "remote_check" },
              ]}
            />
          </Form.Item>
          <Form.Item name="enabled" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item name="mysql_host" label="MySQL Host">
            <Input placeholder="127.0.0.1" />
          </Form.Item>
          <Form.Item name="mysql_port" label="MySQL 端口">
            <InputNumber min={1} max={65535} style={{ width: "100%" }} />
          </Form.Item>
          <Form.Item name="mysql_user" label="MySQL 用户" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="mysql_password" label="MySQL 密码" extra={editing ? "留空表示不修改" : "必填"} rules={editing ? [] : [{ required: true }]}>
            <Input.Password autoComplete="new-password" />
          </Form.Item>
          <Form.Item noStyle shouldUpdate={(p, c) => p.backup_mode !== c.backup_mode}>
            {({ getFieldValue }) =>
              getFieldValue("backup_mode") === "mysqldump" ? (
                <>
                  <Form.Item name="backup_scope" label="备份范围" rules={[{ required: true }]}>
                    <Select
                      options={[
                        { label: "全部库", value: "all" },
                        { label: "单库", value: "database" },
                        { label: "单表", value: "table" },
                      ]}
                    />
                  </Form.Item>
                  <Form.Item noStyle shouldUpdate>
                    {({ getFieldValue: gf }) =>
                      gf("backup_scope") === "database" ? (
                        <Form.Item name="database_name" label="数据库名" rules={[{ required: true }]}>
                          <Input placeholder="mydb" />
                        </Form.Item>
                      ) : gf("backup_scope") === "table" ? (
                        <>
                          <Form.Item name="database_name" label="数据库名" rules={[{ required: true }]}>
                            <Input placeholder="mydb" />
                          </Form.Item>
                          <Form.Item name="table_name" label="表名" rules={[{ required: true }]}>
                            <Input placeholder="users" />
                          </Form.Item>
                        </>
                      ) : (
                        <Form.Item name="database_names" label="多库（可选，逗号分隔）" extra="留空表示 --all-databases">
                          <Input placeholder="db1,db2" />
                        </Form.Item>
                      )
                    }
                  </Form.Item>
                </>
              ) : null
            }
          </Form.Item>
          <Form.Item name="schedule_enabled" label="定时备份" valuePropName="checked" extra="Cron 五段或六段（可选秒），如 0 0 2 * * * 表示每天 2 点">
            <Switch />
          </Form.Item>
          <Form.Item noStyle shouldUpdate>
            {({ getFieldValue }) =>
              getFieldValue("schedule_enabled") ? (
                <Form.Item name="cron_spec" label="Cron 表达式" rules={[{ required: true, message: "请填写 Cron" }]}>
                  <Input placeholder="0 0 2 * * *" />
                </Form.Item>
              ) : null
            }
          </Form.Item>
          <Form.Item noStyle shouldUpdate={(p, c) => p.backup_mode !== c.backup_mode}>
            {({ getFieldValue }) =>
              getFieldValue("backup_mode") === "remote_check" ? (
                <>
                  <Form.Item name="remote_data_dir" label="远端备份数据目录">
                    <Input />
                  </Form.Item>
                  <Form.Item name="remote_log_dir" label="远端备份日志目录">
                    <Input />
                  </Form.Item>
                </>
              ) : null
            }
          </Form.Item>
        </Form>
      </Drawer>
    </Card>
  );
}
