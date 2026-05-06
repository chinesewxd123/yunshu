package alertdispatch

// ChannelTemplateVariableDoc 描述通道自定义模板（Go text/template）可用变量，
// 与 WatchAlert 文档中的「通知模板」能力说明对齐，供 API 与前端展示。
type ChannelTemplateVariableDoc struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ChannelTemplateFieldList 与 BuildChannelTemplateData 注入模板的字段名一致（PascalCase，与模板中 {{.Title}} 对应）。
func ChannelTemplateFieldList() []string {
	return []string{
		"Title", "Severity", "Status", "StatusText", "IsResolved",
		"Summary", "Description", "ProjectName", "Cluster", "OccurredAt",
		"StartsAt", "EndsAt", "Current", "Count", "Fingerprint", "GeneratorURL",
		"Labels", "LabelsText",
	}
}

// ChannelTemplateVariableDocs 返回各字段含义；Labels 为 map，模板中常用 {{index .Labels "alertname"}}。
func ChannelTemplateVariableDocs() []ChannelTemplateVariableDoc {
	return []ChannelTemplateVariableDoc{
		{Name: "Title", Description: "统一标题（含级别、项目、告警名等，由服务端生成）。"},
		{Name: "Severity", Description: "告警级别，如 critical / warning。"},
		{Name: "Status", Description: "原始状态字符串：firing / resolved。"},
		{Name: "StatusText", Description: "中文状态文案：告警触发 / 告警恢复。"},
		{Name: "IsResolved", Description: "布尔：是否为恢复态。"},
		{Name: "Summary", Description: "摘要：来自 payload.summary 或 annotations.summary。"},
		{Name: "Description", Description: "描述：来自 annotations.description。"},
		{Name: "ProjectName", Description: "项目名称：由 project_id 等解析（云枢 enrich 语义）。"},
		{Name: "Cluster", Description: "集群/环境展示字段。"},
		{Name: "OccurredAt", Description: "发生时间（RFC3339 或本地格式字符串）。"},
		{Name: "StartsAt", Description: "告警开始时间（格式化）。"},
		{Name: "EndsAt", Description: "告警结束时间（格式化）。"},
		{Name: "Current", Description: "当前值/查询结果（如 Prom 增强写入）。"},
		{Name: "Count", Description: "计数类扩展字段。"},
		{Name: "Fingerprint", Description: "Alertmanager 指纹。"},
		{Name: "GeneratorURL", Description: "Prometheus/告警规则跳转链接。"},
		{Name: "Labels", Description: "标签 map；模板中建议 {{index .Labels \"alertname\"}}。"},
		{Name: "LabelsText", Description: "紧凑标签展示串，适合直接插入正文。"},
	}
}
