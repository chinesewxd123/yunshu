import { Card, Select, Space, Typography, message } from "antd";
import { useEffect, useState } from "react";
import { ProjectMembersPanel } from "../components/project-members-panel";
import { getProjects, type ProjectItem } from "../services/projects";

export function ProjectMembersPage() {
  const [projects, setProjects] = useState<ProjectItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [projectId, setProjectId] = useState<number | undefined>();

  async function loadProjects() {
    setLoading(true);
    try {
      const data = await getProjects({ page: 1, page_size: 500, keyword: "" });
      setProjects(data.list ?? []);
      if (!projectId && data.list?.length) {
        setProjectId(data.list[0].id);
      }
    } catch {
      message.error("加载项目列表失败");
      setProjects([]);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void loadProjects();
  }, []);

  return (
    <Card
      title="项目成员"
      loading={loading}
      extra={
        <Typography.Text type="secondary" style={{ fontSize: 12 }}>
          先选项目，再管理成员与项目内角色（与全局 RBAC 独立）
        </Typography.Text>
      }
    >
      <Space direction="vertical" style={{ width: "100%" }} size="large">
        <Space wrap>
          <span>当前项目</span>
          <Select
            showSearch
            allowClear={false}
            placeholder="选择项目"
            style={{ minWidth: 280 }}
            loading={loading}
            value={projectId}
            options={projects.map((p) => ({ label: `${p.name} (${p.code})`, value: p.id }))}
            onChange={(v) => setProjectId(v)}
            filterOption={(input, option) =>
              String(option?.label ?? "")
                .toLowerCase()
                .includes(input.toLowerCase())
            }
          />
        </Space>
        {projectId ? <ProjectMembersPanel projectId={projectId} /> : null}
      </Space>
    </Card>
  );
}
