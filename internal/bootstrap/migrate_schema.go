package bootstrap

import (
	"strings"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

// dropLegacyUnusedTables 删除代码中未引用的历史表，避免与当前「监控规则 + alert_duty_blocks」模型混淆。
func dropLegacyUnusedTables(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	for _, t := range []string{"alert_rule_templates"} {
		if !db.Migrator().HasTable(t) {
			continue
		}
		if err := db.Migrator().DropTable(t); err != nil {
			return err
		}
	}
	return nil
}

// dropAlertMonitorRulesLegacyDutyScheduleID 删除旧版 alert_monitor_rules.duty_schedule_id（值班现用 alert_duty_blocks.monitor_rule_id）。
// 当前 Go 模型无该字段，HasColumn 无法检测，故对 MySQL 使用 ALTER 并忽略「列不存在」错误。
func dropAlertMonitorRulesLegacyDutyScheduleID(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable(&model.AlertMonitorRule{}) {
		return nil
	}
	if db.Dialector.Name() != "mysql" {
		return nil
	}
	err := db.Exec("ALTER TABLE `alert_monitor_rules` DROP COLUMN `duty_schedule_id`").Error
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "1091") || strings.Contains(msg, "check that column/key exists") ||
		strings.Contains(msg, "unknown column") || strings.Contains(msg, "1054") {
		return nil
	}
	return err
}

// dropAlertDutyBlocksLegacyScheduleID 删除旧版 alert_duty_blocks.schedule_id 字段。
// 历史版本中该字段为 NOT NULL，当前模型已改为 monitor_rule_id，若不清理会导致插入时报错：
// Error 1364 (HY000): Field 'schedule_id' doesn't have a default value
func dropAlertDutyBlocksLegacyScheduleID(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if !db.Migrator().HasTable(&model.AlertDutyBlock{}) {
		return nil
	}
	if !db.Migrator().HasColumn(&model.AlertDutyBlock{}, "schedule_id") {
		return nil
	}
	return db.Migrator().DropColumn(&model.AlertDutyBlock{}, "schedule_id")
}

// dropDictEntriesLegacyCompositeIndex 删除旧版 dict_entries 上 (dict_type,value,deleted_at) 复合索引。
// 否则将 value 扩为 TEXT/VARCHAR 时 MySQL 报错：BLOB/TEXT column 'value' used in key specification without a key length (1170)。
func dropDictEntriesLegacyCompositeIndex(db *gorm.DB) error {
	if db == nil || db.Dialector.Name() != "mysql" {
		return nil
	}
	if !db.Migrator().HasTable(&model.DictEntry{}) {
		return nil
	}
	err := db.Exec("ALTER TABLE `dict_entries` DROP INDEX `idx_dict_type_value_deleted`").Error
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	// MySQL 1091: Can't DROP '...'; check that column/key exists
	if strings.Contains(msg, "1091") || strings.Contains(msg, "check that it exists") {
		return nil
	}
	return err
}

// cleanupDictEntriesDuplicatesOnBoot 在服务启动迁移阶段清理历史重复数据。
// 规则：
// 1) dict_type + TRIM(label) 重复时保留最小 id
// 2) dict_type + TRIM(value) 重复时保留最小 id
// 这样即使没有进入字典业务接口，也能在重启后自动收敛脏数据。
func cleanupDictEntriesDuplicatesOnBoot(db *gorm.DB) error {
	if db == nil || db.Dialector.Name() != "mysql" {
		return nil
	}
	if !db.Migrator().HasTable(&model.DictEntry{}) {
		return nil
	}
	sqlByLabel := `
DELETE d1
FROM dict_entries d1
JOIN dict_entries d2
  ON d1.dict_type = d2.dict_type
 AND TRIM(d1.label) = TRIM(d2.label)
 AND d1.id > d2.id
WHERE d1.deleted_at IS NULL
  AND d2.deleted_at IS NULL
`
	if err := db.Exec(sqlByLabel).Error; err != nil {
		return err
	}
	sqlByValue := `
DELETE d1
FROM dict_entries d1
JOIN dict_entries d2
  ON d1.dict_type = d2.dict_type
 AND TRIM(d1.value) = TRIM(d2.value)
 AND d1.id > d2.id
WHERE d1.deleted_at IS NULL
  AND d2.deleted_at IS NULL
`
	return db.Exec(sqlByValue).Error
}

// migrateEnableDingTalkSignSecretDictSeed 将历史内置的「钉钉 SignSecret 示例」从停用改为启用。
// 告警渠道「从字典填充 signSecret」仅拉取 status=1 的条目；旧种子为停用会导致下拉暂无数据。
func migrateEnableDingTalkSignSecretDictSeed(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if !db.Migrator().HasTable(&model.DictEntry{}) {
		return nil
	}
	return db.Model(&model.DictEntry{}).
		Where("dict_type = ? AND label = ? AND status = ?", "dingtalk_sign_secret", "钉钉 SignSecret 示例", 0).
		Update("status", 1).Error
}

// migrateFixWecomNotifyModeDictTypo 修正历史误录入的 dict_type（多打了一个 w），否则告警通道无法命中 wecom_notify_mode。
func migrateFixWecomNotifyModeDictTypo(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if !db.Migrator().HasTable(&model.DictEntry{}) {
		return nil
	}
	return db.Model(&model.DictEntry{}).
		Where("dict_type = ?", "wwcom_notify_mode").
		Update("dict_type", "wecom_notify_mode").Error
}

// migrateLogAgentsClearPlaceholderListenPort 历史占位 12580 并非真实监听端口；当前 Agent 为出站 gRPC，本机无监听。
func migrateLogAgentsClearPlaceholderListenPort(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if !db.Migrator().HasTable(&model.LogAgent{}) {
		return nil
	}
	return db.Model(&model.LogAgent{}).
		Where("listen_port = ?", 12580).
		Update("listen_port", 0).Error
}

// migrateNormalizeAlertEventStatus 将历史告警行的 status 规范为小写，便于走 status 单列索引（避免 LOWER(TRIM(status)) 无法命中索引）。
func migrateNormalizeAlertEventStatus(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable(&model.AlertEvent{}) {
		return nil
	}
	switch db.Dialector.Name() {
	case "mysql":
		return db.Exec(`
UPDATE alert_events
SET status = LOWER(TRIM(status))
WHERE deleted_at IS NULL
  AND status IS NOT NULL
  AND status <> LOWER(TRIM(status))
`).Error
	case "postgres":
		return db.Exec(`
UPDATE alert_events
SET status = LOWER(TRIM(status))
WHERE deleted_at IS NULL
  AND status IS NOT NULL
  AND status <> LOWER(TRIM(status))
`).Error
	case "sqlite":
		return db.Exec(`
UPDATE alert_events
SET status = LOWER(TRIM(status))
WHERE deleted_at IS NULL
  AND status IS NOT NULL
  AND LOWER(TRIM(status)) <> status
`).Error
	default:
		return nil
	}
}

// AutoMigrateModels 与 `go run . migrate` 使用同一套表结构；server 启动时执行可避免漏跑迁移导致 500。
func AutoMigrateModels(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if err := dropDictEntriesLegacyCompositeIndex(db); err != nil {
		return err
	}
	if err := cleanupDictEntriesDuplicatesOnBoot(db); err != nil {
		return err
	}
	if err := dropAlertDutyBlocksLegacyScheduleID(db); err != nil {
		return err
	}
	if err := dropAlertMonitorRulesLegacyDutyScheduleID(db); err != nil {
		return err
	}
	if err := db.AutoMigrate(
		&model.Department{},
		&model.User{},
		&model.Role{},
		&model.Permission{},
		&model.UserRole{},
		&model.RegistrationRequest{},
		&model.Menu{},
		&model.LoginLog{},
		&model.OperationLog{},
		&model.DictEntry{},
		&model.K8sCluster{},
		&model.AlertChannel{},
		&model.AlertEvent{},
		&model.AlertDatasource{},
		&model.AlertSilence{},
		&model.AlertMonitorRule{},
		&model.AlertRuleAssignee{},
		&model.AlertDutyBlock{},
		&model.AlertInhibitionRule{},
		&model.AlertInhibitionEvent{},
		&model.AlertSubscriptionNode{},
		&model.AlertReceiverGroup{},
		&model.AlertSubscriptionMatch{},
		&model.AlertFiringDelivery{},
		&model.CloudExpiryRule{},
		&model.Project{},
		&model.ProjectMember{},
		&model.ServerGroup{},
		&model.Server{},
		&model.ServerCredential{},
		&model.CloudAccount{},
		&model.Service{},
		&model.ServiceLogSource{},
		&model.LogAgent{},
		&model.AgentDiscovery{},
		&model.UserGroup{},
		&model.UserGroupUser{},
		&model.K8sNamespaceDenyRule{},
		&model.K8sNamespaceAllowRule{},
		&model.K8sClusterAccessGrant{},
		&model.K8sForwardedEvent{},
		&model.K8sEventForwardRule{},
		&model.K8sEventForwardSetting{},
	); err != nil {
		return err
	}
	if err := migrateK8sLegacyRoleCodeToPrincipal(db); err != nil {
		return err
	}
	if err := migrateDropLegacyK8sCasbinPolicies(db); err != nil {
		return err
	}
	if err := migrateNormalizeAlertEventStatus(db); err != nil {
		return err
	}
	if err := migrateEnableDingTalkSignSecretDictSeed(db); err != nil {
		return err
	}
	if err := migrateFixWecomNotifyModeDictTypo(db); err != nil {
		return err
	}
	if err := migrateLogAgentsClearPlaceholderListenPort(db); err != nil {
		return err
	}
	if err := migrateAgentDiscoveryUniqueIndex(db); err != nil {
		return err
	}
	if err := dropLegacyUnusedTables(db); err != nil {
		return err
	}
	return nil
}

// migrateAgentDiscoveryUniqueIndex 为发现项 upsert 提供唯一键（MySQL value 前缀索引）。
func migrateAgentDiscoveryUniqueIndex(db *gorm.DB) error {
	if db == nil || db.Dialector.Name() != "mysql" {
		return nil
	}
	if !db.Migrator().HasTable("agent_discoveries") {
		return nil
	}
	if db.Migrator().HasIndex("agent_discoveries", "idx_agent_discovery_unique") {
		return nil
	}
	err := db.Exec(
		"CREATE UNIQUE INDEX idx_agent_discovery_unique ON agent_discoveries (project_id, server_id, kind, value(512))",
	).Error
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate") {
		return err
	}
	return nil
}

// migrateDropLegacyK8sCasbinPolicies 移除历史写入的 k8s:cluster:* Casbin 策略，集群权限改由 k8s_cluster_access_grants 表维护。
func migrateDropLegacyK8sCasbinPolicies(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable("casbin_rule") {
		return nil
	}
	return db.Exec("DELETE FROM casbin_rule WHERE ptype = 'p' AND v1 LIKE 'k8s:cluster%'").Error
}

// migrateK8sGrantLegacy 仅用于检测/删除历史 role_code 列（无业务引用）。
type migrateK8sGrantLegacy struct {
	RoleCode string `gorm:"column:role_code"`
}

func (migrateK8sGrantLegacy) TableName() string { return "k8s_cluster_access_grants" }

type migrateK8sDenyLegacy struct {
	RoleCode string `gorm:"column:role_code"`
}

func (migrateK8sDenyLegacy) TableName() string { return "k8s_namespace_deny_rules" }

// migrateK8sLegacyRoleCodeToPrincipal 将历史 role_code 列回填为 principal_kind=role + principal_ref 后删除旧列（对齐 k8m 主体模型）。
func migrateK8sLegacyRoleCodeToPrincipal(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	var gProbe migrateK8sGrantLegacy
	if db.Migrator().HasTable(&model.K8sClusterAccessGrant{}) && db.Migrator().HasColumn(&gProbe, "RoleCode") {
		if err := db.Exec(`UPDATE k8s_cluster_access_grants SET principal_kind = ?, principal_ref = TRIM(role_code) WHERE TRIM(COALESCE(role_code,'')) <> ''`, model.K8sPrincipalRole).Error; err != nil {
			return err
		}
		_ = db.Migrator().DropColumn(&gProbe, "RoleCode")
	}

	var dProbe migrateK8sDenyLegacy
	if db.Migrator().HasTable(&model.K8sNamespaceDenyRule{}) && db.Migrator().HasColumn(&dProbe, "RoleCode") {
		if err := db.Exec(`UPDATE k8s_namespace_deny_rules SET principal_kind = ?, principal_ref = TRIM(role_code) WHERE TRIM(COALESCE(role_code,'')) <> ''`, model.K8sPrincipalRole).Error; err != nil {
			return err
		}
		_ = db.Migrator().DropColumn(&dProbe, "RoleCode")
	}
	return nil
}
