import { Navigate } from "react-router-dom";

/** 已合并到「告警配置中心 → 历史告警记录」，保留路由以兼容旧书签与权限路径。 */
export function AlertEventsPage() {
  return <Navigate to="/alert-monitor-platform?tab=config&cfg=history" replace />;
}
