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
	); err != nil {
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
	return dropLegacyUnusedTables(db)
}
