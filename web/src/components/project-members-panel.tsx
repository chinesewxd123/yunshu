import { Button, Popconfirm, Select, Space, Table, message } from "antd";
import { useEffect, useMemo, useState } from "react";
import { useAuth } from "../contexts/auth-context";
import {
  addProjectMember,
  listProjectMembers,
  removeProjectMember,
  updateProjectMember,
  type ProjectMemberItem,
} from "../services/projects";
import { getUsers } from "../services/users";
import type { UserItem } from "../types/api";

const PROJECT_ROLE_OPTIONS = [
  { value: "owner", label: "负责人 owner" },
  { value: "admin", label: "管理员 admin" },
  { value: "member", label: "成员 member" },
  { value: "readonly", label: "只读 readonly" },
];

function isSuperAdminUser(u: UserItem | null | undefined): boolean {
  return Boolean(u?.roles?.some((r) => r.code === "super-admin"));
}

function canManageProjectMembers(isSuper: boolean, myRole: string | undefined): boolean {
  if (isSuper) return true;
  if (!myRole) return false;
  return myRole === "owner" || myRole === "admin";
}

type Props = {
  projectId: number;
};

export function ProjectMembersPanel({ projectId }: Props) {
  const { user } = useAuth();
  const isSuper = useMemo(() => isSuperAdminUser(user), [user]);
  const [memberList, setMemberList] = useState<ProjectMemberItem[]>([]);
  const [memberLoading, setMemberLoading] = useState(false);
  const [memberAddUserId, setMemberAddUserId] = useState<number | undefined>();
  const [memberAddRole, setMemberAddRole] = useState<string>("member");
  const [userPickOptions, setUserPickOptions] = useState<{ label: string; value: number }[]>([]);

  const myRole = useMemo(() => {
    if (!user) return undefined;
    return memberList.find((m) => m.user_id === user.id)?.role;
  }, [memberList, user]);

  const canManage = canManageProjectMembers(isSuper, myRole);

  useEffect(() => {
    void (async () => {
      setMemberLoading(true);
      try {
        const data = await listProjectMembers(projectId);
        setMemberList(data.list ?? []);
      } catch {
        setMemberList([]);
      } finally {
        setMemberLoading(false);
      }
    })();
  }, [projectId]);

  useEffect(() => {
    void (async () => {
      try {
        const page = await getUsers({ page: 1, page_size: 200, keyword: "" });
        const opts = (page.list ?? []).map((u: UserItem) => ({
          value: u.id,
          label: `${u.nickname || u.username} (${u.username})`,
        }));
        setUserPickOptions(opts);
      } catch {
        setUserPickOptions([]);
      }
    })();
  }, []);

  async function onAddMember() {
    if (!memberAddUserId) {
      message.warning("请选择要加入的用户");
      return;
    }
    try {
      await addProjectMember(projectId, { user_id: memberAddUserId, role: memberAddRole });
      message.success("已添加成员");
      setMemberAddUserId(undefined);
      setMemberAddRole("member");
      const data = await listProjectMembers(projectId);
      setMemberList(data.list ?? []);
    } catch (e) {
      message.error(e instanceof Error ? e.message : "添加失败");
    }
  }

  async function onRemoveMember(memberId: number) {
    try {
      await removeProjectMember(projectId, memberId);
      message.success("已移除");
      const data = await listProjectMembers(projectId);
      setMemberList(data.list ?? []);
    } catch (e) {
      message.error(e instanceof Error ? e.message : "移除失败");
    }
  }

  async function onChangeMemberRole(memberId: number, role: string) {
    try {
      await updateProjectMember(projectId, memberId, { role });
      message.success("角色已更新");
      const data = await listProjectMembers(projectId);
      setMemberList(data.list ?? []);
    } catch (e) {
      message.error(e instanceof Error ? e.message : "更新角色失败");
    }
  }

  return (
    <Space direction="vertical" style={{ width: "100%" }} size="middle">
      {canManage ? (
        <Space wrap style={{ width: "100%" }}>
          <Select
            showSearch
            allowClear
            placeholder="选择用户加入项目"
            style={{ minWidth: 260 }}
            options={userPickOptions}
            value={memberAddUserId}
            onChange={(v) => setMemberAddUserId(v)}
            filterOption={(input, option) =>
              String(option?.label ?? "")
                .toLowerCase()
                .includes(input.toLowerCase())
            }
          />
          <Select
            style={{ width: 200 }}
            value={memberAddRole}
            onChange={(v) => setMemberAddRole(String(v))}
            options={PROJECT_ROLE_OPTIONS}
          />
          <Button type="primary" onClick={() => void onAddMember()}>
            添加成员
          </Button>
        </Space>
      ) : (
        <div style={{ color: "rgba(0,0,0,0.45)", fontSize: 12 }}>
          你当前为项目只读或普通成员，无法在此添加/移除成员或调整角色；项目内写操作与成员管理需 owner/admin（超级管理员除外）。
        </div>
      )}
      <Table<ProjectMemberItem>
        rowKey="id"
        size="small"
        loading={memberLoading}
        pagination={false}
        dataSource={memberList}
        columns={[
          { title: "用户", width: 200, render: (_, r) => r.nickname || r.username },
          { title: "账号", dataIndex: "username", width: 140 },
          {
            title: "项目角色",
            width: 200,
            render: (_, r) =>
              canManage ? (
                <Select
                  size="small"
                  style={{ width: "100%" }}
                  value={r.role || "member"}
                  options={PROJECT_ROLE_OPTIONS}
                  onChange={(v) => void onChangeMemberRole(r.id, String(v))}
                />
              ) : (
                <span>{PROJECT_ROLE_OPTIONS.find((o) => o.value === (r.role || "member"))?.label ?? (r.role || "member")}</span>
              ),
          },
          { title: "邮箱", dataIndex: "email", ellipsis: true, render: (e: string | null | undefined) => e || "—" },
          {
            title: "操作",
            width: 90,
            render: (_, r) =>
              canManage ? (
                <Popconfirm title="从项目移除该用户？" onConfirm={() => void onRemoveMember(r.id)}>
                  <Button type="link" size="small" danger>
                    移除
                  </Button>
                </Popconfirm>
              ) : (
                <span className="inline-muted">—</span>
              ),
          },
        ]}
      />
      <div style={{ color: "rgba(0,0,0,0.45)", fontSize: 12 }}>
        项目内角色（owner/admin/member/readonly）与网关中间件一致：只读成员仅允许 GET/HEAD；修改项目、成员 CRUD 需 owner/admin。全局 RBAC 与 K8s 集群档位仍各自生效。
      </div>
      <div style={{ color: "rgba(0,0,0,0.45)", fontSize: 12 }}>
        项目成员用于权限与资源隔离；告警邮件收件人由「告警监控平台 → 监控规则 → 处理人」配置，<strong>不会</strong>自动发给本项目全部成员。
      </div>
    </Space>
  );
}
