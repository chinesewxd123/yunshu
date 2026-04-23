import {
  ApiOutlined,
  DeleteOutlined,
  DownloadOutlined,
  EditOutlined,
  LinkOutlined,
  PlusOutlined,
  ReloadOutlined,
  SyncOutlined,
  UploadOutlined,
} from "@ant-design/icons";
import { Button, Card, Col, Form, Input, InputNumber, Modal, Popconfirm, Row, Select, Space, Table, Tag, Tree, Typography, message } from "antd";
import type { DataNode } from "antd/es/tree";
import { useEffect, useMemo, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  batchTestProjectServers,
  deleteProjectServer,
  deleteProjectServerGroup,
  downloadProjectServersImportTemplate,
  exportProjectServers,
  getProjectCloudAccounts,
  getProjectServerDetail,
  getProjectServerGroupTree,
  getProjectServers,
  getProjects,
  importProjectServers,
  runProjectCloudServerAction,
  syncProjectCloudAccount,
  deleteProjectCloudAccount,
  testProjectServer,
  upsertProjectCloudAccount,
  upsertProjectServer,
  upsertProjectServerGroup,
  type CloudAccountItem,
  type ProjectItem,
  type ServerGroupItem,
  type ServerItem,
  type ServerUpsertPayload,
} from "../services/projects";
import { useDictOptions } from "../hooks/use-dict-options";
import { DictLabelFillSelect } from "../components/dict-fill-select";
import { formatDateTime } from "../utils/format";

const defaultQuery = { keyword: "", page: 1, page_size: 10 };
const CLOUD_PROVIDER_LABEL: Record<string, string> = {
  alibaba: "阿里云",
  tencent: "腾讯云",
  jd: "京东云",
  custom: "自定义",
};
const CLOUD_REGION_OPTIONS: Record<string, Array<{ label: string; value: string }>> = {
  alibaba: [
    { label: "华东 1（杭州）cn-hangzhou", value: "cn-hangzhou" },
    { label: "华东 2（上海）cn-shanghai", value: "cn-shanghai" },
    { label: "华北 2（北京）cn-beijing", value: "cn-beijing" },
    { label: "华南 1（深圳）cn-shenzhen", value: "cn-shenzhen" },
    { label: "中国香港 cn-hongkong", value: "cn-hongkong" },
  ],
  tencent: [
    { label: "广州 ap-guangzhou", value: "ap-guangzhou" },
    { label: "上海 ap-shanghai", value: "ap-shanghai" },
    { label: "北京 ap-beijing", value: "ap-beijing" },
    { label: "成都 ap-chengdu", value: "ap-chengdu" },
    { label: "中国香港 ap-hongkong", value: "ap-hongkong" },
  ],
  jd: [
    { label: "华北-北京 cn-north-1", value: "cn-north-1" },
    { label: "华东-上海 cn-east-2", value: "cn-east-2" },
    { label: "华南-广州 cn-south-1", value: "cn-south-1" },
  ],
};
const CLOUD_DICT_BY_PROVIDER: Record<
  string,
  { ak: string; sk: string; username: string; password: string; privateKey: string; port: string }
> = {
  alibaba: {
    ak: "cloud_alibaba_ak",
    sk: "cloud_alibaba_sk",
    username: "server_cloud_alibaba_username",
    password: "server_cloud_alibaba_password",
    privateKey: "server_cloud_alibaba_private_key",
    port: "server_cloud_alibaba_port",
  },
  tencent: {
    ak: "cloud_tencent_ak",
    sk: "cloud_tencent_sk",
    username: "server_cloud_tencent_username",
    password: "server_cloud_tencent_password",
    privateKey: "server_cloud_tencent_private_key",
    port: "server_cloud_tencent_port",
  },
  jd: {
    ak: "cloud_jd_ak",
    sk: "cloud_jd_sk",
    username: "server_cloud_jd_username",
    password: "server_cloud_jd_password",
    privateKey: "server_cloud_jd_private_key",
    port: "server_cloud_jd_port",
  },
};

function mapChargeTypeZh(v: string): string {
  const x = String(v || "").trim().toUpperCase();
  if (!x) return "-";
  const m: Record<string, string> = {
    PREPAID: "包年包月",
    PREPAID_BY_DURATION: "包年包月",
    POSTPAID: "按量付费",
    POSTPAID_BY_USAGE: "按量付费",
    POSTPAID_BY_HOUR: "按小时后付费",
    POSTPAID_BY_DURATION: "按配置后付费",
    CDHPAID: "专有宿主机付费",
  };
  return m[x] || v;
}

function mapNetworkChargeTypeZh(v: string): string {
  const x = String(v || "").trim().toUpperCase();
  if (!x) return "-";
  const m: Record<string, string> = {
    PAYBYTRAFFIC: "按流量计费",
    PAYBYBANDWIDTH: "按带宽计费",
    TRAFFIC_POSTPAID_BY_HOUR: "按流量后付费",
    BANDWIDTH_POSTPAID_BY_HOUR: "按带宽后付费",
    BANDWIDTH_PREPAID: "带宽预付费",
    BANDWIDTH_PACKAGE: "带宽包计费",
    NORMAL: "正常计费",
    OVERDUE: "已到期",
    ARREAR: "欠费",
  };
  return m[x] || v;
}

function renderCloudTags(tagsJSON?: string) {
  const raw = String(tagsJSON || "").trim();
  if (!raw) return "-";
  try {
    const obj = JSON.parse(raw) as Record<string, unknown>;
    const entries = Object.entries(obj).filter(([k]) => String(k).trim() !== "");
    if (!entries.length) return "-";
    return (
      <Space size={[4, 4]} wrap>
        {entries.map(([k, v]) => (
          <Tag key={k} color="blue">{`${k}:${String(v ?? "")}`}</Tag>
        ))}
      </Space>
    );
  } catch {
    return raw;
  }
}

type CloudTagKV = { key: string; value: string };

function parseCloudTagRows(raw?: string): CloudTagKV[] {
  const text = String(raw || "").trim();
  if (!text) return [];
  try {
    const obj = JSON.parse(text) as Record<string, unknown>;
    return Object.entries(obj)
      .map(([key, value]) => ({ key: String(key || "").trim(), value: String(value ?? "").trim() }))
      .filter((it) => it.key);
  } catch {
    return [];
  }
}

function buildCloudTagsJSON(rows: CloudTagKV[]): string {
  const obj: Record<string, string> = {};
  rows.forEach((it) => {
    const key = String(it.key || "").trim();
    if (!key) return;
    obj[key] = String(it.value ?? "").trim();
  });
  const keys = Object.keys(obj);
  if (keys.length === 0) return "";
  return JSON.stringify(obj);
}

export function ProjectServersPage() {
  const navigate = useNavigate();
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [projectId, setProjectId] = useState<number>();
  const [query, setQuery] = useState(defaultQuery);
  const [loading, setLoading] = useState(false);
  const [list, setList] = useState<ServerItem[]>([]);
  const [total, setTotal] = useState(0);
  const [selectedRowKeys, setSelectedRowKeys] = useState<number[]>([]);
  const [batchTesting, setBatchTesting] = useState(false);
  const [batchModal, setBatchModal] = useState<{ total: number; success: number; failed: number; results: { server_id: number; ok: boolean; message: string }[] } | null>(null);

  const [groups, setGroups] = useState<ServerGroupItem[]>([]);
  const [selectedGroupId, setSelectedGroupId] = useState<number>();
  const [selectedGroup, setSelectedGroup] = useState<ServerGroupItem | null>(null);
  const [groupEditorOpen, setGroupEditorOpen] = useState(false);
  const [editingGroup, setEditingGroup] = useState<ServerGroupItem | null>(null);
  const [groupForm] = Form.useForm<{ name: string; category: string; provider: string }>();

  const [cloudAccounts, setCloudAccounts] = useState<CloudAccountItem[]>([]);
  const [syncResultByAccount, setSyncResultByAccount] = useState<Record<number, { added: number; updated: number; disabled: number; unchanged: number }>>({});
  const [cloudSubmitting, setCloudSubmitting] = useState(false);
  const [cloudForm] = Form.useForm<{ id?: number; account_name: string; region_scope: string[]; ak: string; sk: string; ak_dict_label?: string; sk_dict_label?: string }>();
  const [editingCloudAccount, setEditingCloudAccount] = useState<CloudAccountItem | null>(null);
  const [cloudAccountEditorOpen, setCloudAccountEditorOpen] = useState(false);
  const [cloudAccountFilter, setCloudAccountFilter] = useState("");
  const [selectedCloudAccountId, setSelectedCloudAccountId] = useState<number | undefined>(undefined);

  const filteredCloudAccounts = useMemo(() => {
    const q = (cloudAccountFilter || "").trim().toLowerCase();
    if (!q) return cloudAccounts;
    return cloudAccounts.filter((a) => {
      return (a.account_name || "").toLowerCase().includes(q) || (a.region_scope || "").toLowerCase().includes(q);
    });
  }, [cloudAccounts, cloudAccountFilter]);

  const filteredServers = useMemo(() => {
    if (!selectedCloudAccountId) return list;
    const acc = cloudAccounts.find((a) => a.id === selectedCloudAccountId);
    if (!acc) return list;
    const regions = new Set(String(acc.region_scope || "").split(",").map((x) => x.trim()).filter(Boolean));
    if (regions.size === 0) {
      return list.filter((s) => (s.provider || "") === acc.provider);
    }
    return list.filter((s) => regions.has((s.cloud_region || "").trim()));
  }, [list, selectedCloudAccountId, cloudAccounts]);

  function findAccountForServer(s: ServerItem) {
    for (const acc of cloudAccounts) {
      const regions = String(acc.region_scope || "").split(",").map((x) => x.trim()).filter(Boolean);
      if (regions.length && regions.includes((s.cloud_region || "").trim())) return acc;
      if ((s.name || "").toLowerCase().includes((acc.account_name || "").toLowerCase())) return acc;
    }
    return undefined;
  }

  const [editorOpen, setEditorOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [current, setCurrent] = useState<ServerItem | null>(null);
  const [cloudActionSubmitting, setCloudActionSubmitting] = useState(false);
  const [resetPasswordTarget, setResetPasswordTarget] = useState<ServerItem | null>(null);
  const [resetPasswordForm] = Form.useForm<{ new_password: string }>();
  const [form] = Form.useForm<ServerUpsertPayload & { port_dict_label?: string; private_key_dict_label?: string }>();
  const [cloudTagRows, setCloudTagRows] = useState<CloudTagKV[]>([]);
  const fileInputRef = useRef<HTMLInputElement | null>(null);
  const [tableScrollY, setTableScrollY] = useState(360);

  const projectOptions = useMemo(() => projects.map((p) => ({ value: p.id, label: `${p.name} (${p.code})` })), [projects]);
  const serverGroupCategoryOptions = useDictOptions("server_group_category");
  const serverOsOptions = useDictOptions("server_os_type");
  const serverAuthOptions = useDictOptions("server_auth_type");
  const serverPortDict = useDictOptions("server_port");
  const sshUserDict = useDictOptions("server_ssh_username");
  const sshPwdDict = useDictOptions("server_ssh_password");
  const cloudAlibabaAKDict = useDictOptions("cloud_alibaba_ak");
  const cloudAlibabaSKDict = useDictOptions("cloud_alibaba_sk");
  const cloudTencentAKDict = useDictOptions("cloud_tencent_ak");
  const cloudTencentSKDict = useDictOptions("cloud_tencent_sk");
  const cloudJdAKDict = useDictOptions("cloud_jd_ak");
  const cloudJdSKDict = useDictOptions("cloud_jd_sk");
  const cloudAlibabaUserDict = useDictOptions("server_cloud_alibaba_username");
  const cloudAlibabaPwdDict = useDictOptions("server_cloud_alibaba_password");
  const cloudAlibabaKeyDict = useDictOptions("server_cloud_alibaba_private_key");
  const cloudAlibabaPortDict = useDictOptions("server_cloud_alibaba_port");
  const cloudTencentUserDict = useDictOptions("server_cloud_tencent_username");
  const cloudTencentPwdDict = useDictOptions("server_cloud_tencent_password");
  const cloudTencentKeyDict = useDictOptions("server_cloud_tencent_private_key");
  const cloudTencentPortDict = useDictOptions("server_cloud_tencent_port");
  const cloudJdUserDict = useDictOptions("server_cloud_jd_username");
  const cloudJdPwdDict = useDictOptions("server_cloud_jd_password");
  const cloudJdKeyDict = useDictOptions("server_cloud_jd_private_key");
  const cloudJdPortDict = useDictOptions("server_cloud_jd_port");
  const isCloudCategory = selectedGroup?.category === "cloud";
  const isCloudProviderNode = isCloudCategory && selectedGroup?.provider !== "custom";
  const cloudProviderLabel = CLOUD_PROVIDER_LABEL[(selectedGroup?.provider || "").trim()] || (selectedGroup?.provider || "-");
  const regionOptions = CLOUD_REGION_OPTIONS[(selectedGroup?.provider || "").trim()] || [];
  const formProvider = (Form.useWatch("provider", form) as string | undefined) || current?.provider || selectedGroup?.provider || "";
  const currentCloudDictKey = CLOUD_DICT_BY_PROVIDER[(formProvider || "").trim()];
  const cloudAKDictOptions = useMemo(() => {
    const p = (selectedGroup?.provider || "").trim();
    if (p === "alibaba") return cloudAlibabaAKDict;
    if (p === "tencent") return cloudTencentAKDict;
    if (p === "jd") return cloudJdAKDict;
    return [];
  }, [selectedGroup?.provider, cloudAlibabaAKDict, cloudTencentAKDict, cloudJdAKDict]);
  const cloudSKDictOptions = useMemo(() => {
    const p = (selectedGroup?.provider || "").trim();
    if (p === "alibaba") return cloudAlibabaSKDict;
    if (p === "tencent") return cloudTencentSKDict;
    if (p === "jd") return cloudJdSKDict;
    return [];
  }, [selectedGroup?.provider, cloudAlibabaSKDict, cloudTencentSKDict, cloudJdSKDict]);
  const activeUserDict = useMemo(() => {
    const p = (currentCloudDictKey?.username || "").trim();
    if (p === "server_cloud_alibaba_username") return cloudAlibabaUserDict;
    if (p === "server_cloud_tencent_username") return cloudTencentUserDict;
    if (p === "server_cloud_jd_username") return cloudJdUserDict;
    return sshUserDict;
  }, [currentCloudDictKey?.username, cloudAlibabaUserDict, cloudTencentUserDict, cloudJdUserDict, sshUserDict]);
  const activePwdDict = useMemo(() => {
    const p = (currentCloudDictKey?.password || "").trim();
    if (p === "server_cloud_alibaba_password") return cloudAlibabaPwdDict;
    if (p === "server_cloud_tencent_password") return cloudTencentPwdDict;
    if (p === "server_cloud_jd_password") return cloudJdPwdDict;
    return sshPwdDict;
  }, [currentCloudDictKey?.password, cloudAlibabaPwdDict, cloudTencentPwdDict, cloudJdPwdDict, sshPwdDict]);
  const activeKeyDict = useMemo(() => {
    const p = (currentCloudDictKey?.privateKey || "").trim();
    if (p === "server_cloud_alibaba_private_key") return cloudAlibabaKeyDict;
    if (p === "server_cloud_tencent_private_key") return cloudTencentKeyDict;
    if (p === "server_cloud_jd_private_key") return cloudJdKeyDict;
    return [];
  }, [currentCloudDictKey?.privateKey, cloudAlibabaKeyDict, cloudTencentKeyDict, cloudJdKeyDict]);
  const activePortDict = useMemo(() => {
    const p = (currentCloudDictKey?.port || "").trim();
    if (p === "server_cloud_alibaba_port") return cloudAlibabaPortDict;
    if (p === "server_cloud_tencent_port") return cloudTencentPortDict;
    if (p === "server_cloud_jd_port") return cloudJdPortDict;
    return serverPortDict;
  }, [currentCloudDictKey?.port, cloudAlibabaPortDict, cloudTencentPortDict, cloudJdPortDict, serverPortDict]);

  useEffect(() => {
    void (async () => {
      const data = await getProjects({ page: 1, page_size: 1000 });
      setProjects(data.list);
      if (data.list[0]) setProjectId(data.list[0].id);
    })();
  }, []);

  useEffect(() => {
    if (!projectId) return;
    void loadGroups();
  }, [projectId]);

  useEffect(() => {
    if (!projectId || !selectedGroupId) return;
    void loadServers();
    if (isCloudProviderNode) void loadCloudAccounts();
  }, [projectId, selectedGroupId, query.keyword, query.page, query.page_size, isCloudProviderNode]);

  useEffect(() => {
    const recalc = () => {
      // Keep page filled while preserving toolbar/filter/pagination space.
      const next = Math.max(260, window.innerHeight - 430);
      setTableScrollY(next);
    };
    recalc();
    window.addEventListener("resize", recalc);
    return () => window.removeEventListener("resize", recalc);
  }, []);

  async function loadGroups() {
    if (!projectId) return;
    const tree = await getProjectServerGroupTree(projectId);
    setGroups(tree);
    const flat = flatten(tree);
    const picked = flat.find((it) => it.category === "self_hosted") || flat[0];
    if (picked) {
      setSelectedGroupId(picked.id);
      setSelectedGroup(picked);
    }
  }

  function flatten(items: ServerGroupItem[]): ServerGroupItem[] {
    const out: ServerGroupItem[] = [];
    const walk = (arr: ServerGroupItem[]) => {
      arr.forEach((it) => {
        out.push(it);
        if (it.children?.length) walk(it.children);
      });
    };
    walk(items);
    return out;
  }

  async function loadServers() {
    if (!projectId || !selectedGroupId) return;
    setLoading(true);
    try {
      const data = await getProjectServers(projectId, {
        ...query,
        group_id: isCloudCategory && selectedGroup?.provider === "custom" ? undefined : selectedGroupId,
        source_type: isCloudCategory ? "cloud" : "self_hosted",
        provider: isCloudCategory && selectedGroup?.provider !== "custom" ? (selectedGroup?.provider || undefined) : undefined,
        cloud_account_id: selectedCloudAccountId,
      });
      setList(data.list || []);
      setTotal(data.total || 0);
      setSelectedRowKeys([]);
    } finally {
      setLoading(false);
    }
  }

  async function loadCloudAccounts() {
    if (!projectId || !selectedGroupId) return;
    const list = await getProjectCloudAccounts(projectId, selectedGroupId);
    setCloudAccounts(list);
  }

  function toTreeData(items: ServerGroupItem[]): DataNode[] {
    return items.map((it) => ({
      key: String(it.id),
      title: (
        <Space size={6}>
          <span>{it.name}</span>
          <Tag>{it.category === "cloud" ? "云" : "自建"}</Tag>
        </Space>
      ),
      children: it.children ? toTreeData(it.children) : undefined,
    }));
  }

  function removeNode(items: ServerGroupItem[], id: number): { tree: ServerGroupItem[]; removed: ServerGroupItem | null } {
    let removed: ServerGroupItem | null = null;
    const walk = (arr: ServerGroupItem[]): ServerGroupItem[] => {
      const out: ServerGroupItem[] = [];
      for (const it of arr) {
        if (it.id === id) {
          removed = { ...it, children: it.children ? [...it.children] : [] };
          continue;
        }
        out.push({ ...it, children: it.children ? walk(it.children) : [] });
      }
      return out;
    };
    return { tree: walk(items), removed };
  }

  function insertNode(items: ServerGroupItem[], targetID: number, node: ServerGroupItem, dropToGap: boolean, dropAfter: boolean): ServerGroupItem[] {
    const walk = (arr: ServerGroupItem[]): ServerGroupItem[] => {
      const out: ServerGroupItem[] = [];
      for (let i = 0; i < arr.length; i++) {
        const it = { ...arr[i], children: arr[i].children ? walk(arr[i].children!) : [] };
        if (it.id === targetID) {
          if (!dropToGap) {
            const children = [...(it.children || []), node];
            out.push({ ...it, children });
          } else if (dropAfter) {
            out.push(it, node);
          } else {
            out.push(node, it);
          }
        } else {
          out.push(it);
        }
      }
      return out;
    };
    return walk(items);
  }

  function normalizeSort(items: ServerGroupItem[], parentID?: number): ServerGroupItem[] {
    return items.map((it, idx) => ({
      ...it,
      parent_id: parentID ?? null,
      sort: (idx + 1) * 10,
      children: it.children ? normalizeSort(it.children, it.id) : [],
    }));
  }

  function flattenForSave(items: ServerGroupItem[]): ServerGroupItem[] {
    const out: ServerGroupItem[] = [];
    const walk = (arr: ServerGroupItem[]) => {
      arr.forEach((it) => {
        out.push(it);
        if (it.children?.length) walk(it.children);
      });
    };
    walk(items);
    return out;
  }

  function openCreate() {
    if (!projectId || !selectedGroupId) return;
    setCurrent(null);
    form.resetFields();
    form.setFieldsValue({
      project_id: projectId,
      group_id: selectedGroupId,
      port: 22,
      port_dict_label: undefined,
      os_type: "linux",
      status: 1,
      auth_type: "password",
      source_type: selectedGroup?.category === "cloud" ? "cloud" : "self_hosted",
      provider: selectedGroup?.provider || "",
      username_dict_label: undefined,
      password_dict_label: undefined,
      private_key_dict_label: undefined,
      cloud_tags_json: "",
    });
    setCloudTagRows([]);
    setEditorOpen(true);
  }

  async function openEdit(record: ServerItem) {
    const pid = projectId || record.project_id;
    setCurrent(record);
    form.resetFields();
    try {
      const detail = await getProjectServerDetail(pid as number, record.id);
      form.setFieldsValue({
        ...detail,
        id: detail.id,
        project_id: detail.project_id,
        group_id: detail.group_id ?? selectedGroupId,
        auth_type: (detail.auth_type === "key" ? "key" : "password") as "password" | "key",
        username: detail.username ?? "",
        password: "",
        private_key: "",
        passphrase: "",
        port_dict_label: undefined,
        username_dict_label: detail.username_dict_label ?? undefined,
        password_dict_label: detail.password_dict_label ?? undefined,
        private_key_dict_label: detail.private_key_dict_label ?? undefined,
      });
      setCloudTagRows(parseCloudTagRows(detail.cloud_tags_json));
    } catch {
      form.setFieldsValue({
        ...record,
        id: record.id,
        project_id: record.project_id,
        group_id: record.group_id ?? selectedGroupId,
        auth_type: "password",
        port_dict_label: undefined,
        username_dict_label: undefined,
        password_dict_label: undefined,
        private_key_dict_label: undefined,
      });
      setCloudTagRows(parseCloudTagRows(record.cloud_tags_json));
    }
    setEditorOpen(true);
  }

  async function onSubmit() {
    if (!projectId) return;
    const values = await form.validateFields();
    const portText = String(values.port ?? "").trim();
    const normalizedPort =
      typeof values.port === "number"
        ? values.port
        : portText !== "" && Number.isFinite(Number(portText))
          ? Number(portText)
          : undefined;
    setSubmitting(true);
    try {
      const status = typeof values.status === "number" ? values.status : (current?.status ?? 1);
      const payload: ServerUpsertPayload = {
        id: values.id,
        project_id: values.project_id,
        group_id: values.group_id,
        name: values.name,
        host: values.host,
        port: normalizedPort,
        os_type: values.os_type,
        tags: values.tags,
        status,
        source_type: values.source_type,
        provider: values.provider,
        cloud_instance_id: values.cloud_instance_id,
        cloud_region: values.cloud_region,
        cloud_tags_json: buildCloudTagsJSON(cloudTagRows) || undefined,
        auth_type: values.auth_type,
        username: (values.username || "").trim(),
        username_dict_label: (values.username_dict_label || "").trim() || undefined,
        password_dict_label: (values.password_dict_label || "").trim() || undefined,
        private_key_dict_label: (values.private_key_dict_label || "").trim() || undefined,
      };
      const pw = (values.password || "").trim();
      if (pw) payload.password = pw;
      const pk = (values.private_key || "").trim();
      if (pk) payload.private_key = pk;
      const pp = (values.passphrase || "").trim();
      if (pp) payload.passphrase = pp;

      await upsertProjectServer(projectId, payload);
      message.success(current ? "已更新服务器" : "已新增服务器");
      setEditorOpen(false);
      void loadServers();
    } finally {
      setSubmitting(false);
    }
  }

  async function onDelete(record: ServerItem) {
    if (!projectId) return;
    await deleteProjectServer(projectId, record.id);
    message.success("已删除");
    void loadServers();
  }

  async function onBatchTest() {
    if (!projectId) return;
    if (selectedRowKeys.length === 0) return message.warning("请先勾选要测试的服务器");
    setBatchTesting(true);
    try {
      const res = await batchTestProjectServers(projectId, selectedRowKeys, 5);
      setBatchModal(res);
      message.success(`批量测试完成：成功 ${res.success}，失败 ${res.failed}`);
      void loadServers();
    } finally {
      setBatchTesting(false);
    }
  }

  async function onImport(file: File) {
    if (!projectId) return;
    const res = await importProjectServers(projectId, file);
    message.success(`已导入 ${res.imported} 条`);
    void loadServers();
  }

  async function onSaveGroup() {
    if (!projectId) return;
    const values = await groupForm.validateFields();
    await upsertProjectServerGroup(projectId, {
      id: editingGroup?.id,
      name: values.name,
      category: values.category,
      provider: values.provider,
      parent_id: editingGroup?.parent_id ?? (values.category === "cloud" && selectedGroup ? selectedGroup.id : undefined),
      sort: editingGroup?.sort,
      status: editingGroup?.status ?? 1,
    });
    message.success(editingGroup ? "分组已更新" : "分组已保存");
    setEditingGroup(null);
    setGroupEditorOpen(false);
    await loadGroups();
  }

  async function onDeleteGroup() {
    if (!projectId || !selectedGroupId) return;
    await deleteProjectServerGroup(projectId, selectedGroupId);
    message.success("分组已删除");
    await loadGroups();
  }

  async function onSaveCloudAccount() {
    if (!projectId || !selectedGroupId || !selectedGroup) return;
    const values = await cloudForm.validateFields();
    setCloudSubmitting(true);
    try {
      await upsertProjectCloudAccount(projectId, {
        id: values.id,
        group_id: selectedGroupId,
        provider: selectedGroup.provider || "alibaba",
        account_name: values.account_name,
        region_scope: (values.region_scope || []).join(","),
        ak: values.ak,
        sk: values.sk,
        ak_dict_label: (values.ak_dict_label || "").trim() || undefined,
        sk_dict_label: (values.sk_dict_label || "").trim() || undefined,
        status: 1,
      });
      message.success(editingCloudAccount ? "云账号已更新" : "云账号已保存");
      setEditingCloudAccount(null);
      cloudForm.resetFields();
      setCloudAccountEditorOpen(false);
      await loadCloudAccounts();
    } finally {
      setCloudSubmitting(false);
    }
  }

  async function onSync(accountId: number) {
    if (!projectId) return;
    const res = await syncProjectCloudAccount(projectId, accountId);
    setSyncResultByAccount((prev) => ({
      ...prev,
      [accountId]: { added: res.added, updated: res.updated, disabled: res.disabled, unchanged: res.unchanged },
    }));
    message.success(`同步完成：发现 ${res.total}，新增 ${res.added}，更新 ${res.updated}，停用 ${res.disabled}，不变 ${res.unchanged}`);
    await Promise.all([loadCloudAccounts(), loadServers()]);
  }

  async function onDeleteCloudAccount(accountId: number) {
    if (!projectId) return;
    try {
      await deleteProjectCloudAccount(projectId, accountId);
      message.success("云账号已删除");
      await Promise.all([loadCloudAccounts(), loadServers()]);
    } catch (err: any) {
      message.error(err?.message || "删除失败");
    }
  }

  function editCloudAccount(item: CloudAccountItem) {
    setEditingCloudAccount(item);
    cloudForm.setFieldsValue({
      id: item.id,
      account_name: item.account_name,
      region_scope: String(item.region_scope || "")
        .split(",")
        .map((s) => s.trim())
        .filter(Boolean),
      ak: "",
      sk: "",
      ak_dict_label: item.ak_dict_label ?? undefined,
      sk_dict_label: item.sk_dict_label ?? undefined,
    });
    setCloudAccountEditorOpen(true);
  }

  function openNewCloudAccount() {
    setEditingCloudAccount(null);
    cloudForm.resetFields();
    setCloudAccountEditorOpen(true);
  }

  async function rebootCloudServer(row: ServerItem) {
    if (!projectId) return;
    setCloudActionSubmitting(true);
    try {
      const res = await runProjectCloudServerAction(projectId, row.id, { action: "reboot" });
      message.success(res.message || "重启请求已提交");
      await loadServers();
    } finally {
      setCloudActionSubmitting(false);
    }
  }

  async function shutdownCloudServer(row: ServerItem) {
    if (!projectId) return;
    setCloudActionSubmitting(true);
    try {
      const res = await runProjectCloudServerAction(projectId, row.id, { action: "shutdown" });
      message.success(res.message || "关机请求已提交");
      await loadServers();
    } finally {
      setCloudActionSubmitting(false);
    }
  }

  function confirmRebootCloudServer(row: ServerItem) {
    Modal.confirm({
      title: "确认重启云服务器？",
      content: `${row.name || row.cloud_instance_id || row.host || row.id} 将执行重启操作。`,
      okText: "确认重启",
      cancelText: "取消",
      onOk: () => rebootCloudServer(row),
    });
  }

  function confirmShutdownCloudServer(row: ServerItem) {
    Modal.confirm({
      title: "确认关机云服务器？",
      content: `${row.name || row.cloud_instance_id || row.host || row.id} 将执行关机操作。`,
      okText: "确认关机",
      cancelText: "取消",
      okButtonProps: { danger: true },
      onOk: () => shutdownCloudServer(row),
    });
  }

  async function submitResetCloudPassword() {
    if (!projectId || !resetPasswordTarget) return;
    const values = await resetPasswordForm.validateFields();
    setCloudActionSubmitting(true);
    try {
      const res = await runProjectCloudServerAction(projectId, resetPasswordTarget.id, { action: "reset_password", new_password: values.new_password });
      message.success(res.message || "密码重置请求已提交");
      setResetPasswordTarget(null);
      resetPasswordForm.resetFields();
      await loadServers();
    } finally {
      setCloudActionSubmitting(false);
    }
  }

  return (
    <Card title="服务器管理" style={{ height: "calc(100vh - 130px)" }} bodyStyle={{ height: "100%", paddingBottom: 12 }}>
      <Row gutter={16} align="stretch" style={{ height: "100%" }}>
        <Col span={6} style={{ height: "100%", display: "flex" }}>
          <Card
            size="small"
            style={{ height: "100%", width: "100%" }}
            title="分组树"
            extra={<Space><Button size="small" icon={<PlusOutlined />} onClick={() => { setEditingGroup(null); groupForm.setFieldsValue({ name: "", category: "self_hosted", provider: "custom" }); setGroupEditorOpen(true); }}>新增</Button><Button size="small" icon={<ReloadOutlined />} onClick={() => void loadGroups()} /></Space>}
          >
            <div style={{ maxHeight: "calc(100vh - 310px)", overflow: "auto", paddingRight: 4 }}>
              <Tree
                draggable
                selectedKeys={selectedGroupId ? [String(selectedGroupId)] : []}
                treeData={toTreeData(groups)}
                onSelect={(keys) => {
                  const id = Number(keys[0]);
                  const item = flatten(groups).find((it) => it.id === id) || null;
                  setSelectedGroupId(id);
                  setSelectedGroup(item);
                  setQuery((q) => ({ ...q, page: 1 }));
                }}
                onDrop={async (info) => {
                  if (!projectId) return;
                  const dragID = Number((info.dragNode as DataNode).key);
                  const dropID = Number((info.node as DataNode).key);
                  const { tree, removed } = removeNode(groups, dragID);
                  if (!removed) return;
                  const nextTree = insertNode(tree, dropID, removed, info.dropToGap ?? false, info.dropPosition > 0);
                  const normalized = normalizeSort(nextTree);
                  setGroups(normalized);
                  const saveItems = flattenForSave(normalized);
                  for (const it of saveItems) {
                    await upsertProjectServerGroup(projectId, {
                      id: it.id,
                      name: it.name,
                      category: it.category,
                      provider: it.provider,
                      parent_id: it.parent_id ?? undefined,
                      sort: it.sort,
                      status: it.status,
                    });
                  }
                  message.success("分组顺序已更新");
                  await loadGroups();
                }}
              />
            </div>
            {selectedGroup ? (
              <Space style={{ marginTop: 12 }}>
                <Button
                  size="small"
                  icon={<EditOutlined />}
                  onClick={() => {
                    if (!selectedGroup) return;
                    setEditingGroup(selectedGroup);
                    groupForm.setFieldsValue({
                      name: selectedGroup.name,
                      category: selectedGroup.category,
                      provider: selectedGroup.provider || "custom",
                    });
                    setGroupEditorOpen(true);
                  }}
                >
                  编辑/重命名
                </Button>
                <Popconfirm title="确认删除该分组？" onConfirm={() => void onDeleteGroup()}>
                  <Button size="small" danger icon={<DeleteOutlined />}>删除分组</Button>
                </Popconfirm>
              </Space>
            ) : null}
          </Card>
        </Col>
        <Col span={18} style={{ height: "100%", display: "flex" }}>
          <div style={{ width: "100%", height: "100%", display: "flex", flexDirection: "column", gap: 12 }}>
            {isCloudProviderNode ? (
              <Card size="small" title={`云账号与同步（${cloudProviderLabel}）`} extra={<Button type="primary" size="small" onClick={() => void openNewCloudAccount()}>新增云账号</Button>}>
                <div style={{ display: "flex", gap: 8, alignItems: "center" }}>
                  <Input.Search allowClear placeholder="搜索云账号/区域" value={cloudAccountFilter} onChange={(e) => setCloudAccountFilter(e.target.value)} style={{ width: 260 }} />
                </div>
                <div style={{ overflowX: "auto" }}>
                <Table
                  style={{ marginTop: 12 }}
                  rowKey="id"
                  pagination={false}
                  dataSource={filteredCloudAccounts}
                  scroll={{ x: 900 }}
                  columns={[
                    { title: "账号", dataIndex: "account_name" },
                    { title: "厂商", dataIndex: "provider", width: 100 },
                    { title: "区域", dataIndex: "region_scope", width: 220, render: (v: string) => v || "默认" },
                    { title: "最近同步", dataIndex: "last_sync_at", width: 180, render: (v?: string | null) => (v ? formatDateTime(v) : "-") },
                    {
                      title: "最近结果",
                      width: 250,
                      render: (_: unknown, r: CloudAccountItem) => {
                        const stat = syncResultByAccount[r.id];
                        if (!stat) return "-";
                        return (
                          <Space size={4} wrap>
                            <Tag color="success">新增 {stat.added}</Tag>
                            <Tag color="processing">更新 {stat.updated}</Tag>
                            <Tag color="warning">停用 {stat.disabled}</Tag>
                            <Tag>不变 {stat.unchanged}</Tag>
                          </Space>
                        );
                      },
                    },
                    { title: "同步错误", dataIndex: "last_sync_error", ellipsis: true, render: (v?: string | null) => v || "-" },
                    {
                      title: "操作",
                      width: 160,
                      render: (_: unknown, r: CloudAccountItem) => (
                        <Space size={6} wrap>
                          <Button size="small" icon={<EditOutlined />} onClick={() => editCloudAccount(r)}>
                            编辑
                          </Button>
                          <Button size="small" icon={<SyncOutlined />} onClick={() => void onSync(r.id)}>
                            同步
                          </Button>
                          <Popconfirm title="确认删除该云账号？" onConfirm={() => void onDeleteCloudAccount(r.id)} okText="删除" cancelText="取消">
                            <Button size="small" danger icon={<DeleteOutlined />}>
                              删除
                            </Button>
                          </Popconfirm>
                        </Space>
                      ),
                    },
                  ]}
                />
                </div>
              </Card>
            ) : null}
            <Card
              size="small"
              style={{ flex: 1, minHeight: 0 }}
              bodyStyle={{ height: "calc(100% - 57px)", display: "flex", flexDirection: "column", minHeight: 0 }}
              title={selectedGroup ? `当前分组：${selectedGroup.name}` : "服务器列表"}
              extra={
                <Space wrap>
                  <Select
                    allowClear
                    showSearch
                    placeholder="按云账号过滤"
                    options={cloudAccounts.map((a) => ({ label: a.account_name || String(a.id), value: a.id }))}
                    style={{ width: 220 }}
                    value={selectedCloudAccountId}
                    onChange={(v) => setSelectedCloudAccountId(v)}
                    filterOption={(input, option) => (option?.label || "").toLowerCase().includes((input || "").toLowerCase())}
                  />
                  <Input.Search allowClear placeholder="搜索 name/host/tags" onSearch={(keyword) => setQuery((q) => ({ ...q, keyword, page: 1 }))} style={{ width: 220 }} />
                  <Button icon={<ReloadOutlined />} onClick={() => void loadServers()} loading={loading}>刷新</Button>
                  <Button icon={<ApiOutlined />} onClick={() => void onBatchTest()} disabled={selectedRowKeys.length === 0} loading={batchTesting}>批量测试</Button>
                  {!isCloudCategory ? (
                    <>
                      <Button icon={<UploadOutlined />} onClick={() => fileInputRef.current?.click()}>导入</Button>
                      <Button icon={<DownloadOutlined />} onClick={() => projectId && downloadProjectServersImportTemplate(projectId).then((blob) => { const u = URL.createObjectURL(blob); const a = document.createElement("a"); a.href = u; a.download = "servers-import-template.xlsx"; a.click(); URL.revokeObjectURL(u); })}>模板</Button>
                      <Button icon={<DownloadOutlined />} onClick={() => projectId && exportProjectServers(projectId, { keyword: query.keyword }).then((blob) => { const u = URL.createObjectURL(blob); const a = document.createElement("a"); a.href = u; a.download = `project-${projectId}-servers.xlsx`; a.click(); URL.revokeObjectURL(u); })}>导出</Button>
                    </>
                  ) : null}
                  <Button type="primary" icon={<PlusOutlined />} onClick={openCreate}>新增</Button>
                </Space>
              }
            >
              <Select style={{ width: 280, marginBottom: 12, flexShrink: 0 }} placeholder="选择项目" value={projectId} onChange={setProjectId} options={projectOptions} />
              <input ref={fileInputRef} type="file" accept=".xlsx,.xls" style={{ display: "none" }} onChange={(e) => { const f = e.target.files?.[0]; if (f) void onImport(f); e.target.value = ""; }} />
              <div style={{ overflowX: "auto", flex: 1 }}>
                <Table
                style={{ flex: 1, minHeight: 0 }}
                rowKey="id"
                  dataSource={filteredServers}
                loading={loading}
                scroll={{ x: 2200, y: tableScrollY }}
                rowSelection={{ selectedRowKeys, onChange: (keys) => setSelectedRowKeys(keys as number[]) }}
                pagination={{
                  current: query.page,
                  pageSize: query.page_size,
                  total: selectedCloudAccountId ? filteredServers.length : total,
                  showSizeChanger: true,
                  pageSizeOptions: [10, 20, 50, 100],
                  showQuickJumper: true,
                  onChange: (page, pageSize) => setQuery((q) => ({ ...q, page, page_size: pageSize })),
                }}
                columns={[
                  { title: "名称", dataIndex: "name", width: 150 },
                  { title: "云厂商", dataIndex: "provider", width: 100, render: (v: string, r: ServerItem) => (r.source_type === "cloud" ? (CLOUD_PROVIDER_LABEL[v] || v || "-") : "-") },
                  { title: "Host", dataIndex: "host", width: 180 },
                  { title: "Port", dataIndex: "port", width: 80 },
                  {
                    title: "来源",
                    width: 120,
                    render: (_: unknown, r: ServerItem) =>
                      r.source_type === "cloud" ? (
                        <Tag color="processing">{`云/${CLOUD_PROVIDER_LABEL[r.provider] || r.provider || "-"}`}</Tag>
                      ) : (
                        <Tag color="default">自建</Tag>
                      ),
                  },
                  { title: "区域", dataIndex: "cloud_region", width: 100, render: (v: string) => v || "-" },
                  { title: "可用区", dataIndex: "cloud_zone", width: 120, render: (v: string) => v || "-" },
                  { title: "规格", dataIndex: "cloud_spec", width: 140, render: (v: string) => v || "-" },
                  { title: "实例配置", dataIndex: "cloud_config_info", width: 180, render: (v: string) => v || "-" },
                  { title: "云OS", dataIndex: "cloud_os_name", width: 180, render: (v: string) => v || "-" },
                  { title: "网络信息", dataIndex: "cloud_network_info", width: 180, render: (v: string) => v || "-" },
                  { title: "实例计费", dataIndex: "cloud_charge_type", width: 140, render: (v: string) => mapChargeTypeZh(v) },
                  { title: "网络计费", dataIndex: "cloud_network_charge_type", width: 140, render: (v: string) => mapNetworkChargeTypeZh(v) },
                  { title: "标签", dataIndex: "cloud_tags_json", width: 220, render: (v: string) => renderCloudTags(v) },
                  { title: "公网IP", dataIndex: "cloud_public_ip", width: 140, render: (v: string) => v || "-" },
                  { title: "内网IP", dataIndex: "cloud_private_ip", width: 140, render: (v: string) => v || "-" },
                  {
                    title: "云状态",
                    dataIndex: "cloud_status_text",
                    width: 120,
                    render: (v: string) => {
                      const s = String(v || "").toUpperCase();
                      if (!s) return "-";
                      if (s.includes("RUNNING")) return <Tag color="success">{v}</Tag>;
                      if (s.includes("STOP") || s.includes("STOPPED")) return <Tag color="warning">{v}</Tag>;
                      return <Tag color="default">{v}</Tag>;
                    },
                  },
                  { title: "OS / 架构", width: 150, render: (_: unknown, r: ServerItem) => `${r.os_type || "-"} / ${r.os_arch || "-"}` },
                  { title: "启用状态", dataIndex: "status", width: 100, render: (v: number) => (v === 1 ? <Tag color="green">启用</Tag> : <Tag color="red">停用</Tag>) },
                  {
                    title: "连通状态",
                    width: 120,
                    render: (_: unknown, r: ServerItem) => {
                      if (!r.last_test_at) return <Tag>未测试</Tag>;
                      return r.last_test_error ? <Tag color="error">异常</Tag> : <Tag color="success">正常</Tag>;
                    },
                  },
                  { title: "上次测试", dataIndex: "last_test_at", width: 180, render: (v?: string | null) => (v ? formatDateTime(v) : "-") },
                  { title: "测试错误", dataIndex: "last_test_error", ellipsis: true, render: (v?: string | null) => (v ? <span title={v}>{v}</span> : "-") },
                  {
                    title: "操作",
                    width: 260,
                    fixed: "right",
                    render: (_: unknown, r: ServerItem) => (
                      <Space size={6} wrap>
                        <Button size="small"
                          icon={<LinkOutlined />}
                          onClick={() => {
                            if (!projectId) return;
                            navigate(`/server-console?project_id=${projectId}&server_id=${r.id}`);
                          }}
                        >
                          连接
                        </Button>
                        <Button size="small" icon={<ApiOutlined />} onClick={() => projectId && testProjectServer(projectId, r.id).then((x) => { x.ok ? message.success(x.message || "连通性 OK") : message.error(x.message || "连通性失败"); void loadServers(); })}>测试</Button>
                        {r.source_type === "cloud" ? (
                          <Button size="small" loading={cloudActionSubmitting} onClick={() => confirmRebootCloudServer(r)}>重启</Button>
                        ) : null}
                        {r.source_type === "cloud" ? (
                          <Button size="small" loading={cloudActionSubmitting} onClick={() => confirmShutdownCloudServer(r)}>关机</Button>
                        ) : null}
                        {r.source_type === "cloud" ? (
                          <Button
                            size="small"
                            loading={cloudActionSubmitting}
                            onClick={() => {
                              setResetPasswordTarget(r);
                              resetPasswordForm.resetFields();
                            }}
                          >
                            改密
                          </Button>
                        ) : null}
                        <Button size="small" icon={<EditOutlined />} onClick={() => openEdit(r)}>编辑</Button>
                        <Popconfirm title="确定删除该服务器？" onConfirm={() => void onDelete(r)}><Button size="small" danger icon={<DeleteOutlined />}>删除</Button></Popconfirm>
                      </Space>
                    ),
                  },
                ]}
              />
              </div>
            </Card>
          </div>
        </Col>
      </Row>

      <Modal title={editingGroup ? "编辑分组" : "新增分组"} open={groupEditorOpen} onCancel={() => { setGroupEditorOpen(false); setEditingGroup(null); }} onOk={() => void onSaveGroup()} destroyOnClose>
        <Form form={groupForm} layout="vertical" initialValues={{ category: "self_hosted", provider: "custom" }}>
          <Form.Item label="名称" name="name" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item label="类型" name="category" rules={[{ required: true }]}>
            <Select options={serverGroupCategoryOptions} />
          </Form.Item>
          <Form.Item label="厂商标识" name="provider">
            <Input placeholder="custom / alibaba / tencent / jd" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal title={current ? "编辑服务器" : "新增服务器"} open={editorOpen} onCancel={() => setEditorOpen(false)} onOk={() => void onSubmit()} confirmLoading={submitting} destroyOnClose width={720}>
        <Form layout="vertical" form={form} autoComplete="off">
          <Form.Item name="id" hidden><Input /></Form.Item>
          <Form.Item name="project_id" hidden><Input /></Form.Item>
          <Form.Item name="group_id" hidden><Input /></Form.Item>
          <Form.Item name="status" hidden><InputNumber /></Form.Item>
          <Form.Item name="source_type" hidden><Input /></Form.Item>
          <Form.Item name="provider" hidden><Input /></Form.Item>
          <Form.Item label="名称" name="name" rules={[{ required: true }]}><Input /></Form.Item>
          <Space style={{ width: "100%" }} size={16} align="start">
            <Form.Item label="Host" name="host" rules={[{ required: true }]} style={{ flex: 1 }}><Input /></Form.Item>
            <Form.Item label="Port" name="port" style={{ width: 160 }}>
              <InputNumber min={1} max={65535} style={{ width: "100%" }} />
            </Form.Item>
            <Form.Item label="OS" name="os_type" style={{ width: 160 }}><Select options={serverOsOptions} /></Form.Item>
          </Space>
          <Form.Item
            label="从数据字典填端口"
            name="port_dict_label"
            extra="按标签选择后自动写入上方 Port；手动修改 Port 不受影响。"
          >
            <DictLabelFillSelect
              form={form}
              labelFieldName="port_dict_label"
              targetFieldName="port"
              options={activePortDict}
              placeholder="选择字典中的端口模板"
            />
          </Form.Item>
          <Form.Item label="区域" name="cloud_region"><Input placeholder="例如：cn-hangzhou / 华东-1 / IDC-A" /></Form.Item>
          <Form.Item label="Tags（逗号分隔）" name="tags"><Input /></Form.Item>
          <Form.Item noStyle shouldUpdate={(prev, next) => prev.source_type !== next.source_type || prev.provider !== next.provider}>
            {({ getFieldValue }) => {
              const sourceType = String(getFieldValue("source_type") || "");
              const provider = String(getFieldValue("provider") || "");
              if (sourceType !== "cloud" || !["tencent", "alibaba", "jd"].includes(provider)) return null;
              return (
                <Card size="small" title={`云厂商标签（${CLOUD_PROVIDER_LABEL[provider] || provider || "-"}）`}>
                  <Space direction="vertical" style={{ width: "100%" }} size={8}>
                    {cloudTagRows.map((row, idx) => (
                      <Space key={`${idx}-${row.key}`} style={{ width: "100%" }} size={8}>
                        <Input
                          placeholder="标签键（Tag Key）"
                          value={row.key}
                          onChange={(e) =>
                            setCloudTagRows((prev) => prev.map((it, i) => (i === idx ? { ...it, key: e.target.value } : it)))
                          }
                        />
                        <Input
                          placeholder="标签值（Tag Value）"
                          value={row.value}
                          onChange={(e) =>
                            setCloudTagRows((prev) => prev.map((it, i) => (i === idx ? { ...it, value: e.target.value } : it)))
                          }
                        />
                        <Button danger onClick={() => setCloudTagRows((prev) => prev.filter((_, i) => i !== idx))}>
                          删除
                        </Button>
                      </Space>
                    ))}
                    <Space>
                      <Button onClick={() => setCloudTagRows((prev) => [...prev, { key: "", value: "" }])}>新增标签</Button>
                      <Typography.Text type="secondary">保存后会回写腾讯云标签并同步到本地。</Typography.Text>
                    </Space>
                  </Space>
                </Card>
              );
            }}
          </Form.Item>
          <Card size="small" title="SSH 凭据（可选）">
            <Space style={{ width: "100%" }} size={16} align="start">
              <Form.Item label="认证方式" name="auth_type" style={{ width: 180 }}><Select options={serverAuthOptions} /></Form.Item>
            </Space>
            <Form.Item
              label="从数据字典填用户名"
              name="username_dict_label"
              extra="按标签选择，避免下拉展示过长内容；选后写入「用户名」。手改用户名会清空此处。"
            >
              <DictLabelFillSelect
                form={form}
                labelFieldName="username_dict_label"
                targetFieldName="username"
                options={activeUserDict}
                placeholder="选择字典中的用户名模板"
              />
            </Form.Item>
            <Form.Item label="用户名" name="username">
              <Input autoComplete="off" />
            </Form.Item>
            <Form.Item noStyle shouldUpdate={(a, b) => a.auth_type !== b.auth_type}>
              {({ getFieldValue }) =>
                getFieldValue("auth_type") === "key" ? (
                  <>
                    <Form.Item
                      label="从数据字典填私钥"
                      name="private_key_dict_label"
                      extra="按标签选择；选后写入下方私钥框。手改私钥会清空此处。"
                    >
                      <DictLabelFillSelect
                        form={form}
                        labelFieldName="private_key_dict_label"
                        targetFieldName="private_key"
                        options={activeKeyDict}
                        placeholder="选择字典中的私钥模板"
                      />
                    </Form.Item>
                    <Form.Item label="私钥（PEM）" name="private_key"><Input.TextArea rows={6} /></Form.Item>
                    <Form.Item label="私钥口令（可选）" name="passphrase"><Input.Password /></Form.Item>
                  </>
                ) : (
                  <>
                    <Form.Item
                      label="从数据字典填密码"
                      name="password_dict_label"
                      extra="按标签选择；选后写入下方密码框。编辑时密码可留空以保留原密码；手改密码会清空此处。"
                    >
                      <DictLabelFillSelect
                        form={form}
                        labelFieldName="password_dict_label"
                        targetFieldName="password"
                        options={activePwdDict}
                        placeholder="选择字典中的密码模板"
                      />
                    </Form.Item>
                    <Form.Item label="密码" name="password">
                      <Input.Password placeholder={current ? "留空表示保留原密码" : undefined} autoComplete="new-password" />
                    </Form.Item>
                  </>
                )
              }
            </Form.Item>
          </Card>
        </Form>
      </Modal>

      <Modal
        title={"重置云服务器密码"}
        open={!!resetPasswordTarget}
        onCancel={() => {
          setResetPasswordTarget(null);
          resetPasswordForm.resetFields();
        }}
        onOk={() => void submitResetCloudPassword()}
        confirmLoading={cloudActionSubmitting}
        destroyOnClose
      >
        <Form form={resetPasswordForm} layout="vertical" autoComplete="off">
          <Form.Item
            label="新密码"
            name="new_password"
            rules={[
              { required: true, message: "请输入新密码" },
              { min: 8, message: "密码至少 8 位" },
            ]}
          >
            <Input.Password autoComplete="new-password" placeholder="请输入云厂商接口要求的新密码" />
          </Form.Item>
        </Form>
      </Modal>

      <Modal title={batchModal ? `批量测试结果（成功 ${batchModal.success} / 失败 ${batchModal.failed}）` : "批量测试结果"} open={!!batchModal} onCancel={() => setBatchModal(null)} footer={null} width={720}>
        {batchModal ? (
          <Table
            size="small"
            rowKey="server_id"
            pagination={false}
            dataSource={batchModal.results}
            columns={[
              { title: "ServerID", dataIndex: "server_id", width: 90 },
              { title: "结果", dataIndex: "ok", width: 90, render: (v: boolean) => (v ? <Tag color="success">OK</Tag> : <Tag color="error">FAIL</Tag>) },
              { title: "说明", dataIndex: "message", render: (v: string) => <Typography.Text ellipsis={{ tooltip: v }}>{v || "-"}</Typography.Text> },
            ]}
          />
        ) : null}
      </Modal>

      <Modal title={editingCloudAccount ? "编辑云账号" : "新增云账号"} open={cloudAccountEditorOpen} onCancel={() => { setCloudAccountEditorOpen(false); setEditingCloudAccount(null); cloudForm.resetFields(); }} onOk={() => void onSaveCloudAccount()} confirmLoading={cloudSubmitting} destroyOnClose width={720}>
        <Form form={cloudForm} layout="vertical" autoComplete="off">
          <Form.Item name="id" hidden>
            <Input />
          </Form.Item>
          <Form.Item label="账号名称" name="account_name" rules={[{ required: true, message: "请输入账号名称" }]}>
            <Input placeholder="账号名称" autoComplete="off" />
          </Form.Item>
          <Form.Item label="同步区域" name="region_scope">
            <Select
              mode="multiple"
              allowClear
              showSearch
              placeholder={regionOptions.length > 0 ? "选择同步区域（可多选）" : "输入或选择区域"}
              options={regionOptions}
            />
          </Form.Item>
          <Form.Item label="从字典填 AK（可选）" name="ak_dict_label">
            <DictLabelFillSelect
              form={cloudForm}
              labelFieldName="ak_dict_label"
              targetFieldName="ak"
              options={cloudAKDictOptions}
              placeholder="从字典填 AK（可选）"
            />
          </Form.Item>
          <Form.Item label="AccessKey ID" name="ak" rules={[{ required: true, message: "请输入 AK" }]}>
            <Input placeholder="AccessKey ID" autoComplete="off" />
          </Form.Item>
          <Form.Item label="从字典填 SK（可选）" name="sk_dict_label">
            <DictLabelFillSelect
              form={cloudForm}
              labelFieldName="sk_dict_label"
              targetFieldName="sk"
              options={cloudSKDictOptions}
              placeholder="从字典填 SK（可选）"
            />
          </Form.Item>
          <Form.Item label="AccessKey Secret" name="sk" rules={[{ required: true, message: "请输入 SK" }]}>
            <Input.Password placeholder="AccessKey Secret" autoComplete="new-password" />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  );
}
