/** 告警路由 / 订阅树等产品术语（界面文案统一） */
export const ALERT_ROUTING_TERMS = {
  tabRouting: "告警路由",
  treeTitle: "告警路由树",
  rootPolicyName: "路由策略",
  nodeName: "路由节点名称",
  nodeCode: "路由节点编码",
  selectNodeHint: "选择路由节点进行编辑",
  continueMatchChildren: "继续匹配子路由",
  receiverGroup: "通知接收组",
  matchSeverity: "匹配级别",
  copyTemplate: "从项目复制路由模板",
  historySourceFilter: "告警来源",
} as const;

/** 树节点展示名：迁移前缀与内部编码友好化 */
export function formatRouteNodeTreeTitle(name: string, enabled: boolean): string {
  let n = String(name ?? "").trim();
  if (n.startsWith("migrated:")) {
    n = n.slice("migrated:".length);
  }
  const suffix = enabled ? "" : "（停用）";
  return `${n || "未命名"}${suffix}`;
}

/** 接收组下拉展示名 */
export function formatReceiverGroupLabel(name: string, id: number): string {
  const n = String(name ?? "").trim();
  if (n.startsWith("migrated:")) {
    return n.slice("migrated:".length) || `接收组 ${id}`;
  }
  return n || `接收组 ${id}`;
}
