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
    "接收组绑定告警通道（钉钉/邮件/企微/wechat 等）。规则配置「处理人」且命中接收组后：wechat 等会补发邮件；钉钉/企微在群内 @ 不到处理人（手机号未入企或无法解析）时补发邮件；已在群内可被 @ 时仅走 IM。不含项目全员。",
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
