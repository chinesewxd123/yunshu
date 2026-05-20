import {
  CloudUploadOutlined,
  DeleteOutlined,
  EditOutlined,
  FileTextOutlined,
  LinkOutlined,
  PlusOutlined,
  ReloadOutlined,
  ThunderboltOutlined,
} from "@ant-design/icons";
import {
  Alert,
  Button,
  Card,
  Checkbox,
  Divider,
  Drawer,
  Form,
  Input,
  InputNumber,
  Modal,
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
import { useCallback, useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";
import {
  checkMysqlRemoteBackup,
  createMysqlBackupInstance,
  deleteMysqlBackupInstance,
  listMysqlBackupInstances,
  listMysqlBackupJobs,
  listMysqldumpOptions,
  pingMysqlBackupInstance,
  type MysqldumpOptionItem,
  presignMysqlBackupJob,
  runMysqlBackup,
  updateMysqlBackupInstance,
  type MysqlBackupInstance,
  type MysqlBackupInstancePayload,
  type MysqlBackupJob,
} from "../services/mysql-backup";
import { getProjectServers, getProjects, type ProjectItem, type ServerItem } from "../services/projects";
import { formatDateTime } from "../utils/format";

function isXtrabackupMode(mode?: string) {
  return mode === "xtrabackup" || mode === "remote_check";
}

function MysqldumpOptionsField({ catalog }: { catalog: MysqldumpOptionItem[] }) {
  const grouped = useMemo(() => {
    const map = new Map<string, MysqldumpOptionItem[]>();
    for (const o of catalog) {
      const g = o.group || "其他";
      const list = map.get(g) ?? [];
      list.push(o);
      map.set(g, list);
    }
    return [...map.entries()];
  }, [catalog]);

  return (
    <Form.Item
      name="mysqldump_options"
      label="mysqldump 选项"
      extra="互斥项请勿同时勾选（如 --single-transaction 与 --lock-all-tables）；条件导出请用下方「额外参数」填写 --where=..."
    >
      <Checkbox.Group style={{ width: "100%" }}>
        {grouped.map(([group, items]) => (
          <div key={group} style={{ marginBottom: 12 }}>
            <Typography.Text type="secondary" style={{ display: "block", marginBottom: 6 }}>
              {group}
            </Typography.Text>
            <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: "4px 12px" }}>
              {items.map((o) => (
                <Checkbox key={o.id} value={o.id}>
                  {o.label}
                </Checkbox>
              ))}
            </div>
          </div>
        ))}
      </Checkbox.Group>
    </Form.Item>
  );
}

export function MysqlBackupPage() {
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [servers, setServers] = useState<ServerItem[]>([]);
  const [instances, setInstances] = useState<MysqlBackupInstance[]>([]);
  const [jobs, setJobs] = useState<MysqlBackupJob[]>([]);
  const [jobsTotal, setJobsTotal] = useState(0);
  const [jobQuery, setJobQuery] = useState({ page: 1, page_size: 10 });
  const [mysqldumpOptions, setMysqldumpOptions] = useState<MysqldumpOptionItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [jobsLoading, setJobsLoading] = useState(false);
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [logJob, setLogJob] = useState<MysqlBackupJob | null>(null);
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

  const loadInstances = useCallback(async () => {
    if (!projectId) return;
    setLoading(true);
    try {
      const instRes = await listMysqlBackupInstances(projectId, { page: 1, page_size: 100 });
      setInstances(instRes.list || []);
    } catch (e) {
      message.error(String(e));
    } finally {
      setLoading(false);
    }
  }, [projectId]);

  const loadJobs = useCallback(async () => {
    if (!projectId) return;
    setJobsLoading(true);
    try {
      const jobRes = await listMysqlBackupJobs(projectId, jobQuery);
      setJobs(jobRes.list || []);
      setJobsTotal(jobRes.total ?? 0);
    } catch (e) {
      message.error(String(e));
    } finally {
      setJobsLoading(false);
    }
  }, [projectId, jobQuery]);

  const load = useCallback(async () => {
    await Promise.all([loadInstances(), loadJobs()]);
  }, [loadInstances, loadJobs]);

  useEffect(() => {
    void loadServers();
  }, [loadServers]);

  useEffect(() => {
    if (!projectId) return;
    void loadInstances();
    void listMysqldumpOptions(projectId).then(setMysqldumpOptions).catch(() => setMysqldumpOptions([]));
    setJobQuery((q) => ({ ...q, page: 1 }));
  }, [projectId, loadInstances]);

  useEffect(() => {
    if (!projectId) return;
    void loadJobs();
  }, [projectId, loadJobs]);

  const hasRunningJob = jobs.some((j) => j.status === "running");

  useEffect(() => {
    if (!projectId || !hasRunningJob) return;
    const timer = window.setInterval(() => {
      void loadJobs();
    }, 4000);
    return () => window.clearInterval(timer);
  }, [projectId, hasRunningJob, loadJobs]);

  useEffect(() => {
    if (!logJob) return;
    const fresh = jobs.find((j) => j.id === logJob.id);
    if (fresh) {
      setLogJob(fresh);
    }
  }, [jobs, logJob?.id]);

  const openCreate = () => {
    setEditing(null);
    form.setFieldsValue({
      name: "",
      mysql_host: "127.0.0.1",
      mysql_port: 3306,
      mysql_user: "root",
      mysql_password: "",
      backup_mode: "xtrabackup",
      backup_scope: "all",
      enabled: true,
      schedule_enabled: false,
      cron_spec: "",
      database_name: "",
      table_name: "",
      database_names: "",
      upload_to_minio: true,
      mysql_datadir: "/export/mysql_data",
      remote_data_dir: "/export/servers/data/mybackup/my3306/xtrabackup/data",
      remote_log_dir: "/export/servers/data/mybackup/my3306/xtrabackup/log",
      mysqldump_work_dir: "/export/servers/data/mybackup/my3306/mysqldump",
      mysqldump_options: ["single_transaction", "quick", "routines", "triggers"],
      mysqldump_extra_args: "",
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
      upload_to_minio: row.upload_to_minio !== false,
      mysql_datadir: row.mysql_datadir || "/export/mysql_data",
      remote_data_dir: row.remote_data_dir,
      remote_log_dir: row.remote_log_dir,
      mysqldump_work_dir: row.mysqldump_work_dir || "/export/backup/yunshu",
      mysqldump_options: row.mysqldump_options?.length
        ? row.mysqldump_options
        : ["single_transaction", "quick", "routines", "triggers"],
      mysqldump_extra_args: row.mysqldump_extra_args || "",
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
        if (isXtrabackupMode(r.backup_mode)) return <Tag>全量</Tag>;
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
      render: (m: string) => (isXtrabackupMode(m) ? <Tag color="purple">xtrabackup</Tag> : <Tag color="blue">mysqldump</Tag>),
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
          {isXtrabackupMode(row.backup_mode) ? (
            <Button
              size="small"
              onClick={() =>
                void checkMysqlRemoteBackup(projectId!, row.id).then((r) => {
                  if (r.ok) {
                    message.success(r.message || "检查通过");
                  } else {
                    message.error(r.message || "检查失败");
                  }
                })
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
                .then((job) => {
                  message.success(`备份任务 #${job.id} 已提交，请在「备份记录」查看进度`);
                  void loadJobs();
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
      title: "实际方式",
      dataIndex: "backup_mode",
      width: 100,
      render: (m?: string) => {
        if (m === "xtrabackup") return <Tag color="purple">xtrabackup</Tag>;
        if (m === "mysqldump") return <Tag color="blue">mysqldump</Tag>;
        if (isXtrabackupMode(m)) return <Tag color="purple">xtrabackup</Tag>;
        return <Tag>{m || "-"}</Tag>;
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
    { title: "远端路径", dataIndex: "remote_path", width: 140, ellipsis: true },
    { title: "完成时间", dataIndex: "finished_at", width: 168, render: (v?: string) => formatDateTime(v) },
    {
      title: "操作",
      width: 160,
      fixed: "right",
      render: (_, row) => (
        <Space>
          <Button size="small" icon={<FileTextOutlined />} onClick={() => setLogJob(row)}>
            日志
          </Button>
          {row.status === "success" && row.minio_object ? (
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
          ) : null}
        </Space>
      ),
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
            复用项目内服务器 SSH 凭据；归档至 MinIO 请在{" "}
            <Link to="/dict-entries?keyword=minio_">数据字典</Link> 维护 <Typography.Text code>minio_*</Typography.Text>、
            <Typography.Text code>mysql_backup_scheduler_*</Typography.Text> 等项。
            <strong>mysqldump</strong>：逻辑备份；<strong>xtrabackup</strong>：经 SSH 执行物理热备（须目标机已安装 xtrabackup）。
            文件命名统一为 <Typography.Text code>项目名_IP_端口_年月日_时分秒</Typography.Text>（.sql.gz / .tar.gz）。
            「检查备份」按该前缀匹配最新有效包。上传 MinIO 可关；上传后<strong>不删除</strong>远端文件。
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
            children: (
              <Table
                rowKey="id"
                loading={jobsLoading}
                columns={jobColumns}
                dataSource={jobs}
                scroll={{ x: 900 }}
                pagination={{
                  current: jobQuery.page,
                  pageSize: jobQuery.page_size,
                  total: jobsTotal,
                  showSizeChanger: true,
                  pageSizeOptions: [10, 20, 50, 100],
                  showQuickJumper: true,
                  showTotal: (t, range) => `${range[0]}-${range[1]} / 共 ${t} 条`,
                  onChange: (page, pageSize) => setJobQuery({ page, page_size: pageSize }),
                }}
              />
            ),
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
                { label: "mysqldump（逻辑备份）", value: "mysqldump" },
                { label: "xtrabackup（物理备份）", value: "xtrabackup" },
              ]}
            />
          </Form.Item>
          <Form.Item name="enabled" label="启用" valuePropName="checked">
            <Switch />
          </Form.Item>
          <Form.Item
            name="upload_to_minio"
            label="上传 MinIO"
            valuePropName="checked"
            extra="关闭时仅在 SSH 服务器保留备份文件，任务日志中可查看远端路径"
          >
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
            {({ getFieldValue }) => {
              const mode = getFieldValue("backup_mode");
              if (mode === "mysqldump") {
                return (
                  <>
                    <Divider orientation="left" plain>
                      mysqldump
                    </Divider>
                    <Form.Item
                      name="mysqldump_work_dir"
                      label="远端落盘目录"
                      extra="文件命名：项目名_IP_端口_年月日_时分秒.sql.gz，勿与 xtrabackup 目录相同"
                      rules={[{ required: true }, { pattern: /^\//, message: "须以 / 开头的绝对路径" }]}
                    >
                      <Input placeholder="/export/backup/yunshu" />
                    </Form.Item>
                    <MysqldumpOptionsField catalog={mysqldumpOptions} />
                    <Form.Item
                      name="mysqldump_extra_args"
                      label="额外参数"
                      extra="空格分隔，如 --where=id>1000 --max-allowed-packet=1G（须以 - 开头）"
                    >
                      <Input.TextArea rows={2} placeholder="--where=status=1" />
                    </Form.Item>
                  </>
                );
              }
              if (isXtrabackupMode(mode)) {
                return (
                  <>
                    <Divider orientation="left" plain>
                      xtrabackup
                    </Divider>
                    <Form.Item
                      name="mysql_datadir"
                      label="MySQL 数据目录（datadir）"
                      rules={[{ required: true }, { pattern: /^\//, message: "须为绝对路径" }]}
                      extra="宿主机真实路径；Docker 映射目录如 /export/mysql_data（不是容器内 /var/lib/mysql）"
                    >
                      <Input placeholder="/export/mysql_data" />
                    </Form.Item>
                    <Form.Item
                      name="remote_data_dir"
                      label="备份产物目录"
                      rules={[{ required: true }, { pattern: /^\//, message: "须为绝对路径" }]}
                      extra="xtrabackup 打包后的 项目名_IP_端口_时间.tar.gz 写在此目录"
                    >
                      <Input />
                    </Form.Item>
                    <Form.Item
                      name="remote_log_dir"
                      label="远端日志目录"
                      rules={[{ required: true }, { pattern: /^\//, message: "须为绝对路径" }]}
                      extra="同名 .log，末行须含 yunshu backup completed OK!"
                    >
                      <Input />
                    </Form.Item>
                  </>
                );
              }
              return null;
            }}
          </Form.Item>
        </Form>
      </Drawer>
      <Modal
        title={logJob ? `备份日志 #${logJob.id}` : "备份日志"}
        open={!!logJob}
        onCancel={() => setLogJob(null)}
        footer={null}
        width={720}
        destroyOnClose
      >
        {logJob ? (
          <Space direction="vertical" style={{ width: "100%" }} size="middle">
            {logJob.error_message ? (
              <Alert type="error" showIcon message="错误信息" description={<pre style={{ margin: 0, whiteSpace: "pre-wrap" }}>{logJob.error_message}</pre>} />
            ) : null}
            <div>
              <Typography.Text type="secondary">执行日志（mysqldump 输出或远端 xtrabackup 日志尾部）</Typography.Text>
              <pre
                style={{
                  marginTop: 8,
                  maxHeight: 420,
                  overflow: "auto",
                  padding: 12,
                  background: "var(--ant-color-fill-quaternary, #f5f5f5)",
                  borderRadius: 6,
                  whiteSpace: "pre-wrap",
                  wordBreak: "break-all",
                }}
              >
                {logJob.status === "running"
                  ? "备份进行中，请稍候刷新…（mysqldump 全库可能需数分钟）"
                  : logJob.log_excerpt?.trim() || "（无日志内容，请确认备份已执行完成）"}
              </pre>
            </div>
          </Space>
        ) : null}
      </Modal>
    </Card>
  );
}
