package service

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"yunshu/internal/model"
	"yunshu/internal/pkg/apperror"
	"yunshu/internal/pkg/pagination"
	"yunshu/internal/repository"

	"gorm.io/gorm"
)

// 与 MySQL MEDIUMTEXT 上限一致的量级；在 binding 层不设 max，避免大 kubeconfig 被 ShouldBindJSON 拒绝。
const dictEntryValueMaxBytes = 16 << 20 // 16 MiB

func validateDictEntryValueBytes(v string) error {
	if len(v) > dictEntryValueMaxBytes {
		return apperror.BadRequest(fmt.Sprintf("字典值过大（当前约 %d 字节，上限 %d 字节）", len(v), dictEntryValueMaxBytes))
	}
	return nil
}

func intRef(v int) *int {
	p := v
	return &p
}

// dictEntrySort 将 JSON 中的 null/省略映射为 0（前端 InputNumber 清空会提交 null，不能直接绑定到 int）。
func dictEntrySort(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

type DictEntryListQuery struct {
	DictType string `form:"dict_type"`
	Keyword  string `form:"keyword"`
	Status   *int   `form:"status"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

type DictEntryCreateRequest struct {
	DictType string `json:"dict_type" binding:"required,max=64"`
	Label    string `json:"label" binding:"required,max=128"`
	Value    string `json:"value" binding:"required"`
	Sort     *int   `json:"sort"`
	Status   int    `json:"status" binding:"oneof=0 1"`
	Remark   string `json:"remark" binding:"omitempty,max=512"`
}

type DictEntryUpdateRequest struct {
	DictType string `json:"dict_type" binding:"required,max=64"`
	Label    string `json:"label" binding:"required,max=128"`
	Value    string `json:"value" binding:"required"`
	Sort     *int   `json:"sort"`
	Status   int    `json:"status" binding:"oneof=0 1"`
	Remark   string `json:"remark" binding:"omitempty,max=512"`
}

type DictEntryOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type DictEntryService struct {
	repo     *repository.DictEntryRepository
	initOnce sync.Once
}

const (
	dictTypeAlertPromQLLabelKey      = "alert_promql_label_key"
	dictTypeAlertSilenceMatcherName  = "alert_silence_matcher_name"
)

func canonicalDictType(dictType string) string {
	t := strings.TrimSpace(dictType)
	if t == dictTypeAlertSilenceMatcherName {
		return dictTypeAlertPromQLLabelKey
	}
	return t
}

func NewDictEntryService(repo *repository.DictEntryRepository) *DictEntryService {
	return &DictEntryService{repo: repo}
}

func (s *DictEntryService) ensureBuiltins(ctx context.Context) {
	// 每次进入字典服务都先做一次历史去重，避免依赖 initOnce 触发时机。
	// 这样即使服务已运行较久、或历史版本已产生重复，也能自动收敛。
	_ = s.repo.CleanupDuplicateTypeLabel(ctx)
	_ = s.repo.CleanupDuplicateTypeValue(ctx)

	s.initOnce.Do(func() {
		// 历史收敛：静默 matcher key 已统一到 alert_promql_label_key，先迁移再删除旧类型。
		s.migrateAlertSilenceMatcherKeys(ctx)

		// 不再使用数据字典维护 HTTP 方法；清理历史遗留行，避免与「仅保留敏感配置类字典」目标冲突。
		_ = s.repo.DeleteByTypes(ctx, []string{"http_action"})

		seed := []DictEntryCreateRequest{
			{DictType: "alert_webhook_url", Label: "企业微信机器人 URL 示例", Value: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=replace-me", Sort: intRef(1), Status: 0, Remark: "Webhook 通道 URL 候选"},
			{DictType: "wecom_webhook_url", Label: "企业微信机器人 URL 示例", Value: "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=replace-me", Sort: intRef(1), Status: 0, Remark: "企业微信 Webhook URL"},
			{DictType: "dingtalk_webhook_url", Label: "钉钉机器人 URL 示例", Value: "https://oapi.dingtalk.com/robot/send?access_token=replace-me", Sort: intRef(1), Status: 0, Remark: "钉钉 Webhook URL"},
			{DictType: "agent_platform_url", Label: "本机平台地址", Value: "http://127.0.0.1:8080", Sort: intRef(1), Status: 1, Remark: "Agent 部署时平台地址"},
			// Alert 运行配置（字典优先，YAML 兜底）
			{DictType: "alert_webhook_token", Label: "Webhook Token 示例", Value: "change-me-alert-token", Sort: intRef(1), Status: 0, Remark: "alert.webhook_token：用于 Alertmanager Webhook 鉴权"},
			{DictType: "alert_enrich_prometheus_url", Label: "Prometheus 地址示例", Value: "http://127.0.0.1:9090", Sort: intRef(1), Status: 0, Remark: "alert.prometheus_url：用于告警增强查询"},
			{DictType: "alert_enrich_prometheus_token", Label: "Prometheus Token（可选）", Value: "", Sort: intRef(1), Status: 0, Remark: "alert.prometheus_token：敏感信息建议仅在生产库维护"},
			{DictType: "wecom_notify_mode", Label: "群机器人(robot)", Value: "robot", Sort: intRef(1), Status: 1, Remark: "企业微信通知模式"},
			{DictType: "wecom_notify_mode", Label: "企业应用(app)", Value: "app", Sort: intRef(2), Status: 1, Remark: "企业微信通知模式"},
			{DictType: "dingtalk_notify_mode", Label: "群机器人(robot)", Value: "robot", Sort: intRef(1), Status: 1, Remark: "钉钉通知模式"},
			{DictType: "dingtalk_notify_mode", Label: "应用会话(app_chat)", Value: "app_chat", Sort: intRef(2), Status: 1, Remark: "钉钉通知模式"},
			{DictType: "alert_datasource_base_url", Label: "Prometheus 根地址示例", Value: "http://prometheus:9090", Sort: intRef(1), Status: 1, Remark: "告警数据源 Base URL，可在数据字典增删"},
			{DictType: "alert_datasource_basic_user", Label: "Basic 用户示例", Value: "prometheus", Sort: intRef(1), Status: 1, Remark: "告警数据源 Basic 用户，可在数据字典增删"},
			// PromQL 标签键候选（规则编辑页下拉）
			{DictType: "alert_promql_label_key", Label: "instance", Value: "instance", Sort: intRef(1), Status: 1, Remark: "PromQL 标签键"},
			{DictType: "alert_promql_label_key", Label: "job", Value: "job", Sort: intRef(2), Status: 1, Remark: "PromQL 标签键"},
			{DictType: "alert_promql_label_key", Label: "cluster", Value: "cluster", Sort: intRef(3), Status: 1, Remark: "PromQL 标签键"},
			{DictType: "alert_promql_label_key", Label: "namespace", Value: "namespace", Sort: intRef(4), Status: 1, Remark: "PromQL 标签键"},
			{DictType: "alert_promql_label_key", Label: "pod", Value: "pod", Sort: intRef(5), Status: 1, Remark: "PromQL 标签键"},
			{DictType: "alert_promql_label_key", Label: "service", Value: "service", Sort: intRef(6), Status: 1, Remark: "PromQL 标签键"},
			{DictType: "alert_promql_label_key", Label: "node", Value: "node", Sort: intRef(7), Status: 1, Remark: "PromQL 标签键"},
			{DictType: "alert_promql_label_key", Label: "severity", Value: "severity", Sort: intRef(8), Status: 1, Remark: "PromQL 标签键"},
			{DictType: "alert_promql_label_key", Label: "alertname", Value: "alertname", Sort: intRef(9), Status: 1, Remark: "PromQL 标签键"},
			{DictType: "alert_promql_label_key", Label: "path", Value: "path", Sort: intRef(10), Status: 1, Remark: "PromQL 标签键"},
			{DictType: "alert_promql_label_key", Label: "device", Value: "device", Sort: intRef(11), Status: 1, Remark: "PromQL 标签键"},
			{DictType: "alert_promql_label_key", Label: "fstype", Value: "fstype", Sort: intRef(12), Status: 1, Remark: "PromQL 标签键"},
			{DictType: "alert_promql_label_key", Label: "mountpoint", Value: "mountpoint", Sort: intRef(13), Status: 1, Remark: "PromQL 标签键"},
			// 企业微信 / 钉钉专用字典（用于通道配置自动填充）
			{DictType: "wecom_corp_id", Label: "企业微信 CorpID 示例", Value: "wwxxxxxxxxxxxxxxxx", Sort: intRef(1), Status: 0, Remark: "企业微信应用模式 corpID"},
			{DictType: "wecom_corp_secret", Label: "企业微信 CorpSecret 示例", Value: "", Sort: intRef(1), Status: 0, Remark: "企业微信应用模式 corpSecret，敏感信息建议仅在生产维护"},
			{DictType: "wecom_agent_id", Label: "企业微信 AgentID 示例", Value: "1000002", Sort: intRef(1), Status: 0, Remark: "企业微信应用模式 agentId"},
			{DictType: "dingtalk_app_key", Label: "钉钉 AppKey 示例", Value: "dingxxxxxxxx", Sort: intRef(1), Status: 0, Remark: "钉钉 app_chat 模式 appKey"},
			{DictType: "dingtalk_app_secret", Label: "钉钉 AppSecret 示例", Value: "", Sort: intRef(1), Status: 0, Remark: "钉钉 app_chat 模式 appSecret，敏感信息建议仅在生产维护"},
			{DictType: "dingtalk_chat_id", Label: "钉钉 ChatID 示例", Value: "chatxxxxxxxx", Sort: intRef(1), Status: 0, Remark: "钉钉 app_chat 模式 chatId"},
			// 须为启用：告警渠道「从字典填充 signSecret」走 /dict/options，仅返回 status=1 的条目
			{DictType: "dingtalk_sign_secret", Label: "钉钉 SignSecret 示例", Value: "", Sort: intRef(1), Status: 1, Remark: "钉钉 robot 模式加签 SEC（在字典中填写真实值；与 app_chat 的 dingtalk_app_secret 不同）"},
			// 阈值单位候选
			{DictType: "alert_threshold_unit", Label: "原始值", Value: "raw", Sort: intRef(1), Status: 1, Remark: "不指定单位"},
			{DictType: "alert_threshold_unit", Label: "百分比(%)", Value: "percent", Sort: intRef(2), Status: 1, Remark: "用于使用率类指标"},
			{DictType: "alert_threshold_unit", Label: "字节(bytes)", Value: "bytes", Sort: intRef(3), Status: 1, Remark: "用于容量类指标"},
			{DictType: "alert_threshold_unit", Label: "毫秒(ms)", Value: "ms", Sort: intRef(4), Status: 1, Remark: "用于耗时类指标"},
			{DictType: "alert_threshold_unit", Label: "计数(count)", Value: "count", Sort: intRef(5), Status: 1, Remark: "用于请求数/错误数"},
			// 集群 kubeconfig 模板（请替换 server/token）；「集群管理」表单可一键插入
			{DictType: "k8s_kubeconfig_template", Label: "单集群 kubeconfig 模板", Value: "kubeconfig文件", Sort: intRef(1), Status: 1, Remark: "占位说明：可在字典中维护完整 kubeconfig 供集群管理选择；勿将生产密钥提交到 Git"},
			// 集群直连配置模板：label 作为配置键，value 存直连 JSON（可在集群管理 direct 模式通过 dict_config_key 引用）
			{DictType: "k8s_direct_config", Label: "prod-sa-token", Value: `{"server":"https://10.0.0.10:6443","token":"replace-with-service-account-token","ca_data":"replace-with-base64-ca","insecure_skip_tls_verify":false}`, Sort: intRef(1), Status: 1, Remark: "生产集群直连示例（token 认证）"},
			{DictType: "k8s_direct_config", Label: "staging-basic-auth", Value: `{"server":"https://10.0.0.20:6443","username":"admin","password":"replace-with-password","insecure_skip_tls_verify":true}`, Sort: intRef(2), Status: 1, Remark: "测试集群直连示例（用户名密码认证）"},

			// Mail（作为字典权威来源，覆盖 config.yaml）
			{DictType: "mail_host", Label: "163 SMTP", Value: "smtp.163.com", Sort: intRef(1), Status: 1, Remark: "mail.host：字典存在则覆盖 config.yaml"},
			{DictType: "mail_port", Label: "SMTP 端口(SSL)", Value: "465", Sort: intRef(1), Status: 1, Remark: "mail.port：字典存在则覆盖 config.yaml"},
			{DictType: "mail_use_tls", Label: "启用 TLS", Value: "true", Sort: intRef(1), Status: 1, Remark: "mail.use_tls：true/false"},
			// 服务器管理相关字典（分组/OS/认证/凭据模板/端口模板）
			{DictType: "server_group_category", Label: "自建服务器", Value: "self_hosted", Sort: intRef(1), Status: 1, Remark: "服务器分组类别"},
			{DictType: "server_group_category", Label: "云服务器", Value: "cloud", Sort: intRef(2), Status: 1, Remark: "服务器分组类别"},
			{DictType: "server_os_type", Label: "Linux", Value: "linux", Sort: intRef(1), Status: 1, Remark: "服务器操作系统类型"},
			{DictType: "server_os_type", Label: "Windows", Value: "windows", Sort: intRef(2), Status: 1, Remark: "服务器操作系统类型"},
			{DictType: "server_auth_type", Label: "密码", Value: "password", Sort: intRef(1), Status: 1, Remark: "服务器 SSH 认证方式"},
			{DictType: "server_auth_type", Label: "私钥", Value: "key", Sort: intRef(2), Status: 1, Remark: "服务器 SSH 认证方式"},
			{DictType: "server_ssh_username", Label: "root", Value: "root", Sort: intRef(1), Status: 1, Remark: "服务器 SSH 用户名模板"},
			{DictType: "server_ssh_username", Label: "admin", Value: "admin", Sort: intRef(2), Status: 1, Remark: "服务器 SSH 用户名模板"},
			{DictType: "server_ssh_password", Label: "默认密码模板（示例）", Value: "change-me-password", Sort: intRef(1), Status: 1, Remark: "服务器 SSH 密码模板，生产建议改为真实值"},
			{DictType: "server_port", Label: "SSH 默认端口 22", Value: "22", Sort: intRef(1), Status: 1, Remark: "服务器连接端口模板"},
			{DictType: "server_port", Label: "RDP 默认端口 3389", Value: "3389", Sort: intRef(2), Status: 1, Remark: "服务器连接端口模板"},
		}
		singletonTypes := map[string]struct{}{
			"mail_host":                     {},
			"mail_port":                     {},
			"mail_use_tls":                  {},
			"mail_username":                 {},
			"mail_password":                 {},
			"mail_from_email":               {},
			"mail_from_name":                {},
			"alert_webhook_token":           {},
			"alert_enrich_prometheus_url":   {},
			"alert_enrich_prometheus_token": {},
		}
		for _, item := range seed {
			var (
				exists bool
				err    error
			)
			if _, ok := singletonTypes[item.DictType]; ok {
				exists, err = s.repo.ExistsByType(ctx, item.DictType, 0)
			} else {
				// 对内置种子按「类型 + 标签」幂等，避免值被人工改动后再次 seed 产生同标签重复项。
				exists, err = s.repo.ExistsByTypeLabel(ctx, item.DictType, item.Label, 0)
			}
			if err != nil || exists {
				continue
			}
			_ = s.repo.Create(ctx, &model.DictEntry{
				DictType: strings.TrimSpace(item.DictType),
				Label:    strings.TrimSpace(item.Label),
				Value:    strings.TrimSpace(item.Value),
				Sort:     dictEntrySort(item.Sort),
				Status:   item.Status,
				Remark:   strings.TrimSpace(item.Remark),
			})
		}
		// 清理早期内置的长 YAML 示例，避免与占位条目重复。
		_ = s.repo.DeleteByTypeAndLabel(ctx, "k8s_kubeconfig_template", "单集群 Bearer 模板")
		// 清理历史 SMTP 示例条目，避免每次 seed 后出现“示例邮箱/名称”重复项。
		_ = s.repo.DeleteByTypeAndValue(ctx, "mail_username", "root@example.com")
		_ = s.repo.DeleteByTypeAndValue(ctx, "mail_from_email", "root@example.com")
		_ = s.repo.DeleteByTypeAndValue(ctx, "mail_from_name", "YunShu")
		// 旧类型清理：收敛为单一 alert_promql_label_key 后，不再保留旧 dict_type。
		_ = s.repo.DeleteByTypes(ctx, []string{dictTypeAlertSilenceMatcherName})
	})

	// 内置种子处理后再做一次去重，兜底并发/历史脏数据场景。
	_ = s.repo.CleanupDuplicateTypeLabel(ctx)
	_ = s.repo.CleanupDuplicateTypeValue(ctx)
}

func (s *DictEntryService) migrateAlertSilenceMatcherKeys(ctx context.Context) {
	oldList, err := s.repo.ListByType(ctx, dictTypeAlertSilenceMatcherName)
	if err != nil || len(oldList) == 0 {
		return
	}
	for _, it := range oldList {
		targetLabel := strings.TrimSpace(it.Label)
		targetValue := strings.TrimSpace(it.Value)
		if targetLabel == "" || targetValue == "" {
			continue
		}
		existsByValue, err := s.repo.ExistsByTypeValue(ctx, dictTypeAlertPromQLLabelKey, targetValue, 0)
		if err == nil && existsByValue {
			continue
		}
		existsByLabel, err := s.repo.ExistsByTypeLabel(ctx, dictTypeAlertPromQLLabelKey, targetLabel, 0)
		if err == nil && existsByLabel {
			continue
		}
		_ = s.repo.Create(ctx, &model.DictEntry{
			DictType: dictTypeAlertPromQLLabelKey,
			Label:    targetLabel,
			Value:    targetValue,
			Sort:     it.Sort,
			Status:   it.Status,
			Remark:   strings.TrimSpace(it.Remark),
		})
	}
}

func (s *DictEntryService) List(ctx context.Context, query DictEntryListQuery) (*pagination.Result[model.DictEntry], error) {
	s.ensureBuiltins(ctx)
	query.DictType = canonicalDictType(query.DictType)
	page, pageSize := pagination.Normalize(query.Page, query.PageSize)
	list, total, err := s.repo.List(ctx, query.DictType, query.Keyword, query.Status, page, pageSize)
	if err != nil {
		return nil, err
	}
	return &pagination.Result[model.DictEntry]{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *DictEntryService) Create(ctx context.Context, req DictEntryCreateRequest) (*model.DictEntry, error) {
	s.ensureBuiltins(ctx)
	rawVal := req.Value
	if err := validateDictEntryValueBytes(rawVal); err != nil {
		return nil, err
	}
	item := model.DictEntry{
		DictType: canonicalDictType(req.DictType),
		Label:    strings.TrimSpace(req.Label),
		Value:    strings.TrimSpace(rawVal),
		Sort:     dictEntrySort(req.Sort),
		Status:   req.Status,
		Remark:   strings.TrimSpace(req.Remark),
	}
	if item.DictType == "" || item.Label == "" || item.Value == "" {
		return nil, apperror.BadRequest("字典类型、标签和值不能为空")
	}
	if exists, err := s.repo.ExistsByTypeLabel(ctx, item.DictType, item.Label, 0); err == nil && exists {
		return nil, apperror.BadRequest("同字典类型下该标签已存在，请勿重复创建")
	}
	if exists, err := s.repo.ExistsByTypeValue(ctx, item.DictType, item.Value, 0); err == nil && exists {
		return nil, apperror.BadRequest("同字典类型下该值已存在，请勿重复创建")
	}
	if err := s.repo.Create(ctx, &item); err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *DictEntryService) Update(ctx context.Context, id uint, req DictEntryUpdateRequest) (*model.DictEntry, error) {
	item, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("字典条目不存在")
		}
		return nil, err
	}
	rawVal := req.Value
	if err := validateDictEntryValueBytes(rawVal); err != nil {
		return nil, err
	}
	item.DictType = strings.TrimSpace(req.DictType)
	item.DictType = canonicalDictType(item.DictType)
	item.Label = strings.TrimSpace(req.Label)
	item.Value = strings.TrimSpace(rawVal)
	item.Sort = dictEntrySort(req.Sort)
	item.Status = req.Status
	item.Remark = strings.TrimSpace(req.Remark)
	if item.DictType == "" || item.Label == "" || item.Value == "" {
		return nil, apperror.BadRequest("字典类型、标签和值不能为空")
	}
	if exists, err2 := s.repo.ExistsByTypeLabel(ctx, item.DictType, item.Label, item.ID); err2 == nil && exists {
		return nil, apperror.BadRequest("同字典类型下该标签已存在，请勿重复保存")
	}
	if exists, err2 := s.repo.ExistsByTypeValue(ctx, item.DictType, item.Value, item.ID); err2 == nil && exists {
		return nil, apperror.BadRequest("同字典类型下该值已存在，请勿重复保存")
	}
	if err = s.repo.Update(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *DictEntryService) Delete(ctx context.Context, id uint) error {
	_, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperror.NotFound("字典条目不存在")
		}
		return err
	}
	return s.repo.Delete(ctx, id)
}

func (s *DictEntryService) Options(ctx context.Context, dictType string) ([]DictEntryOption, error) {
	s.ensureBuiltins(ctx)
	list, err := s.repo.ListByTypeEnabled(ctx, canonicalDictType(dictType))
	if err != nil {
		return nil, err
	}
	options := make([]DictEntryOption, 0, len(list))
	for _, item := range list {
		options = append(options, DictEntryOption{
			Label: item.Label,
			Value: item.Value,
		})
	}
	return options, nil
}
