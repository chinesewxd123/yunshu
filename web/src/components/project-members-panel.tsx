import { Button, Popconfirm, Select, Space, Table, message } from "antd";
import { useEffect, useState } from "react";
import {
  addProjectMember,
  listProjectMembers,
  removeProjectMember,
  type ProjectMemberItem,
} from "../services/projects";
import { getUsers } from "../services/users";
import type { UserItem } from "../types/api";

type Props = {
  projectId: number;
};

export function ProjectMembersPanel({ projectId }: Props) {
  const [memberList, setMemberList] = useState<ProjectMemberItem[]>([]);
  const [memberLoading, setMemberLoading] = useState(false);
  const [memberAddUserId, setMemberAddUserId] = useState<number | undefined>();
  const [userPickOptions, setUserPickOptions] = useState<{ label: string; value: number }[]>([]);

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
      await addProjectMember(projectId, { user_id: memberAddUserId });
      message.success("已添加成员");
      setMemberAddUserId(undefined);
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

  return (
    <Space direction="vertical" style={{ width: "100%" }} size="middle">
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
        <Button type="primary" onClick={() => void onAddMember()}>
          添加成员
        </Button>
      </Space>
      <Table<ProjectMemberItem>
        rowKey="id"
        size="small"
        loading={memberLoading}
        pagination={false}
        dataSource={memberList}
        columns={[
          { title: "用户", width: 200, render: (_, r) => r.nickname || r.username },
          { title: "账号", dataIndex: "username", width: 140 },
          { title: "邮箱", dataIndex: "email", ellipsis: true, render: (e: string | null | undefined) => e || "—" },
          {
            title: "操作",
            width: 90,
            render: (_, r) => (
              <Popconfirm title="从项目移除该用户？" onConfirm={() => void onRemoveMember(r.id)}>
                <Button type="link" size="small" danger>
                  移除
                </Button>
              </Popconfirm>
            ),
          },
        ]}
      />
      <div style={{ color: "rgba(0,0,0,0.45)", fontSize: 12 }}>
        项目成员不再区分 owner/admin/member/readonly，权限统一由现有全局 RBAC/Casbin 与 K8s 三元策略控制。
      </div>
      <div style={{ color: "rgba(0,0,0,0.45)", fontSize: 12 }}>
        绑定项目的监控规则触发告警时，除「规则处理人」外，会将<strong>本项目启用成员</strong>的邮箱一并纳入邮件通知收件人（去重）。
      </div>
    </Space>
  );
}
