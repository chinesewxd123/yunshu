import { CloseOutlined, PauseOutlined, PlayCircleOutlined } from "@ant-design/icons";
import { Button, Space, Tag } from "antd";
import { Link, useLocation } from "react-router-dom";
import { useLogStreamOptional } from "../contexts/log-stream-context";

export function LogStreamDockBar() {
  const loc = useLocation();
  const stream = useLogStreamOptional();
  if (!stream?.streaming) return null;
  if (loc.pathname.includes("project-logs")) return null;

  const { form, streamModeHint, lineCount, linesPerSec, stop, togglePause, paused } = stream;
  const qs = new URLSearchParams();
  if (form.project_id) qs.set("project_id", String(form.project_id));
  if (form.server_id) qs.set("server_id", String(form.server_id));
  if (form.log_source_id) qs.set("log_source_id", String(form.log_source_id));
  qs.set("autostart", "1");

  return (
    <div className="log-stream-dock-bar">
      <Space wrap size="middle">
        <Tag color="processing">日志流后台运行</Tag>
        <span>{streamModeHint}</span>
        <span>
          项目 #{form.project_id} · 服务器 #{form.server_id} · 源 #{form.log_source_id}
        </span>
        <span>
          {lineCount} 行 · {linesPerSec} 行/秒
        </span>
        <Link to={`/project-logs?${qs.toString()}`}>
          <Button size="small" type="primary">
            返回日志页
          </Button>
        </Link>
        <Button size="small" icon={paused ? <PlayCircleOutlined /> : <PauseOutlined />} onClick={togglePause}>
          {paused ? "继续显示" : "暂停显示"}
        </Button>
        <Button size="small" danger icon={<CloseOutlined />} onClick={stop}>
          停止
        </Button>
      </Space>
    </div>
  );
}
