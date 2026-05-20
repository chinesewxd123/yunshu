package mysqlbackup

import (
	"strings"

	"yunshu/internal/model"
)

// DumpTarget mysqldump 命令尾部参数（库/表名或 --all-databases）。
type DumpTarget struct {
	UseAllDatabases bool
	Parts           []string // 未转义的库名、表名
	Scope           string
	Database        string
	Table           string
	ObjectLabel     string
}

// BuildDumpTarget 根据实例配置生成 mysqldump 目标。
func BuildDumpTarget(inst *model.MysqlBackupInstance) DumpTarget {
	scope := strings.TrimSpace(inst.BackupScope)
	if scope == "" {
		scope = model.MysqlBackupScopeAll
	}
	db := strings.TrimSpace(inst.DatabaseName)
	tbl := strings.TrimSpace(inst.BackupTable)

	switch scope {
	case model.MysqlBackupScopeTable:
		if db != "" && tbl != "" {
			return DumpTarget{
				Parts:       []string{db, tbl},
				Scope:       model.MysqlBackupScopeTable,
				Database:    db,
				Table:       tbl,
				ObjectLabel: db + "." + tbl,
			}
		}
	case model.MysqlBackupScopeDatabase:
		if db != "" {
			return DumpTarget{
				Parts:       []string{db},
				Scope:       model.MysqlBackupScopeDatabase,
				Database:    db,
				ObjectLabel: db,
			}
		}
	}

	if names := strings.TrimSpace(inst.DatabaseNames); names != "" {
		parts := strings.Split(names, ",")
		var dbs []string
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				dbs = append(dbs, p)
			}
		}
		if len(dbs) == 1 {
			return DumpTarget{
				Parts:       []string{dbs[0]},
				Scope:       model.MysqlBackupScopeDatabase,
				Database:    dbs[0],
				ObjectLabel: dbs[0],
			}
		}
		if len(dbs) > 1 {
			return DumpTarget{
				Parts:       dbs,
				Scope:       model.MysqlBackupScopeDatabase,
				ObjectLabel: "multi-db",
			}
		}
	}

	return DumpTarget{
		UseAllDatabases: true,
		Scope:           model.MysqlBackupScopeAll,
		ObjectLabel:     "all-databases",
	}
}

// FormatDumpArgsShell 将目标格式化为可嵌入 shell 的 mysqldump 尾部参数。
func FormatDumpArgsShell(target DumpTarget, quote func(string) string) string {
	if target.UseAllDatabases {
		return "--all-databases"
	}
	var quoted []string
	for _, p := range target.Parts {
		quoted = append(quoted, quote(p))
	}
	return strings.Join(quoted, " ")
}
