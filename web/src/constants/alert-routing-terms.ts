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
  receiverGroupManage: "通知接收组管理",
  receiverGroupManageHint:
    "接收组绑定告警通道（钉钉/邮件/企微等）。critical 且仅绑钉钉/企微时：自动向「监控规则 → 处理人」中显式配置的用户邮箱发信（不展开部门子树、不含项目全员）；钉钉仍会 @ 处理人手机号。接收组已绑邮件通道时按原逻辑投递。",
  receiverGroupStaticCC: "静态抄送（可选）",
} as const;

/** 历史节点名 → 产品展示名 */
const LEGACY_ROUTE_NODE_DISPLAY_NAMES: Record<string, string> = {
  通知策略: "路由策略",
};

/** 树节点展示名：迁移前缀、历史名称与内部编码友好化 */
export function formatRouteNodeTreeTitle(name: string, enabled: boolean): string {
  let n = String(name ?? "").trim();
  if (LEGACY_ROUTE_NODE_DISPLAY_NAMES[n]) {
    n = LEGACY_ROUTE_NODE_DISPLAY_NAMES[n];
  }
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
