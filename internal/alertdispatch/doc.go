// Package alertdispatch 将「投递信封、通道类型、模板变量、Webhook 响应解析、历史入库瘦身」与 AlertService 解耦，
// 演进方向对齐 WatchAlert 的「告警事件 → 通知模板 → 多渠道 → 审计」分层。
//
// 云枢（本项目）在告警侧保留并强化的能力（与 WatchAlert 互补，不在此包内实现）包括：
//   - 订阅树路由（项目维度 + labels/severity 匹配 + 接收组 + 静默窗口）
//   - 告警静默、抑制规则、值班块与监控规则处理人
//   - 平台 PromQL 监控规则、Prometheus 查询增强、云到期规则
//   - 与项目/云账号加密集成的凭据解密等运维域能力
//
// 本包仅承载与「外发契约」强相关的可复用纯逻辑，避免 service 包继续膨胀。
package alertdispatch
