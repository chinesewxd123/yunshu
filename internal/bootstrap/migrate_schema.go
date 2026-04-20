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

// dropAlertMonitorRulesProjectID deletes the legacy column alert_monitor_rules.project_id.
// Phase-2 cleanup: project_id is now derived from alert_datasources.project_id.
func dropAlertMonitorRulesProjectID(db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable(&model.AlertMonitorRule{}) {
		return nil
	}
	if db.Dialector.Name() != "mysql" {
		return nil
	}
	// If column doesn't exist, this will error; we ignore common MySQL "missing" errors.
	err := db.Exec("ALTER TABLE `alert_monitor_rules` DROP COLUMN `project_id`").Error
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

// migrateAlertDatasourcesProjectID ensures alert_datasources.project_id exists and is backfilled.
// Strategy:
// - Add the column (default 0) if missing.
// - If a datasource is referenced by rules that belong to exactly one project -> backfill to that project.
// - If referenced by multiple projects -> duplicate datasource per project and rewrite rules to the per-project datasource.
// - Any remaining project_id=0 -> set to smallest active project id (fallback), else 1.
func migrateAlertDatasourcesProjectID(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	if !db.Migrator().HasTable(&model.AlertDatasource{}) {
		return nil
	}

	// 1) Add column if missing (keep it nullable/default during backfill).
	if !db.Migrator().HasColumn(&model.AlertDatasource{}, "project_id") {
		// Add as NOT NULL with default 0 for MySQL compatibility.
		if err := db.Exec("ALTER TABLE `alert_datasources` ADD COLUMN `project_id` BIGINT UNSIGNED NOT NULL DEFAULT 0").Error; err != nil {
			return err
		}
		_ = db.Exec("CREATE INDEX `idx_alert_datasources_project_id` ON `alert_datasources`(`project_id`)").Error
	}

	// Quick exit if already backfilled.
	var zeros int64
	if err := db.Model(&model.AlertDatasource{}).Where("project_id = 0").Count(&zeros).Error; err == nil && zeros == 0 {
		return nil
	}

	// 2) Backfill from rules where unambiguous.
	// In phase-2, alert_monitor_rules.project_id may already be dropped.
	// Only use this legacy backfill path when the old column still exists.
	legacyRuleProjectColumnExists := false
	if db.Dialector.Name() == "mysql" {
		type colCountRow struct {
			Count int64 `gorm:"column:cnt"`
		}
		var row colCountRow
		_ = db.Raw(
			"SELECT COUNT(*) AS cnt FROM INFORMATION_SCHEMA.columns WHERE table_schema = DATABASE() AND table_name = 'alert_monitor_rules' AND column_name = 'project_id'",
		).Scan(&row).Error
		legacyRuleProjectColumnExists = row.Count > 0
	}

	type dsProjRow struct {
		DatasourceID uint
		ProjectID    uint
		Cnt          int64
	}
	var rows []dsProjRow
	if legacyRuleProjectColumnExists {
		if err := db.Raw(`
SELECT datasource_id AS datasource_id, project_id AS project_id, COUNT(*) AS cnt
FROM alert_monitor_rules
WHERE deleted_at IS NULL
GROUP BY datasource_id, project_id
`).Scan(&rows).Error; err != nil {
			return err
		}
	}

	// Build datasource -> distinct projects map.
	dsToProjects := map[uint]map[uint]struct{}{}
	for _, r := range rows {
		if r.DatasourceID == 0 || r.ProjectID == 0 {
			continue
		}
		m, ok := dsToProjects[r.DatasourceID]
		if !ok {
			m = map[uint]struct{}{}
			dsToProjects[r.DatasourceID] = m
		}
		m[r.ProjectID] = struct{}{}
	}

	// Load all datasources with project_id=0 (or all, small table).
	var dss []model.AlertDatasource
	if err := db.Model(&model.AlertDatasource{}).Find(&dss).Error; err != nil {
		return err
	}

	// For each datasource referenced by >1 projects, duplicate.
	for _, ds := range dss {
		projSet := dsToProjects[ds.ID]
		if len(projSet) == 0 {
			continue
		}
		if len(projSet) == 1 {
			var pid uint
			for p := range projSet {
				pid = p
			}
			_ = db.Model(&model.AlertDatasource{}).Where("id = ? AND project_id = 0", ds.ID).Update("project_id", pid).Error
			continue
		}

		// multi-project: create per project datasource, then rewrite rules.
		for pid := range projSet {
			newDS := model.AlertDatasource{
				ProjectID:     pid,
				Name:          ds.Name,
				Type:          ds.Type,
				BaseURL:       ds.BaseURL,
				BearerToken:   ds.BearerToken,
				BasicUser:     ds.BasicUser,
				BasicPassword: ds.BasicPassword,
				SkipTLSVerify: ds.SkipTLSVerify,
				Enabled:       ds.Enabled,
				Remark:        ds.Remark,
			}
			if err := db.Create(&newDS).Error; err != nil {
				return err
			}
			if err := db.Model(&model.AlertMonitorRule{}).
				Where("datasource_id = ? AND project_id = ?", ds.ID, pid).
				Update("datasource_id", newDS.ID).Error; err != nil {
				return err
			}
		}
		// After duplication, keep the original datasource but assign it later by fallback (or set to one project if you prefer).
	}

	// 3) Fallback for remaining project_id=0.
	var minProjectID uint
	_ = db.Raw("SELECT id FROM projects WHERE deleted_at IS NULL AND status = 1 ORDER BY id ASC LIMIT 1").Scan(&minProjectID).Error
	if minProjectID == 0 {
		minProjectID = 1
	}
	return db.Model(&model.AlertDatasource{}).Where("project_id = 0").Update("project_id", minProjectID).Error
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
	if err := dropAlertMonitorRulesProjectID(db); err != nil {
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
		&model.AlertPolicy{},
		&model.AlertDatasource{},
		&model.AlertSilence{},
		&model.AlertMonitorRule{},
		&model.AlertRuleAssignee{},
		&model.AlertDutyBlock{},
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
	if err := migrateAlertDatasourcesProjectID(db); err != nil {
		return err
	}
	return dropLegacyUnusedTables(db)
}
