import { Navigate, useSearchParams } from "react-router-dom";

/** 已并入「告警监控平台 → 策略与联调」；保留路由以兼容旧书签与菜单。 */
export function AlertConfigCenterPage() {
  const [searchParams] = useSearchParams();
  const legacy = searchParams.get("tab");
  const cfg = legacy === "history" || legacy === "templates" ? legacy : "policies";
  return <Navigate to={`/alert-monitor-platform?tab=config&cfg=${cfg}`} replace />;
}
