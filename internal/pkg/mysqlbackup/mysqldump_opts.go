package mysqlbackup

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// MysqldumpOption 可选 mysqldump 参数（前端勾选，后端白名单映射）。
type MysqldumpOption struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Flag  string `json:"flag"`
	Group string `json:"group,omitempty"`
}

// MysqldumpOptionCatalog 供 API/前端展示（白名单，勿与冲突项同时勾选，如 lock-all-tables 与 single-transaction）。
var MysqldumpOptionCatalog = []MysqldumpOption{
	// 一致性与锁
	{ID: "single_transaction", Group: "一致性与锁", Label: "InnoDB 单事务 (--single-transaction)", Flag: "--single-transaction"},
	{ID: "quick", Group: "一致性与锁", Label: "逐行读取 (--quick)", Flag: "--quick"},
	{ID: "skip_lock_tables", Group: "一致性与锁", Label: "不锁表 (--skip-lock-tables)", Flag: "--skip-lock-tables"},
	{ID: "lock_tables", Group: "一致性与锁", Label: "导出前锁表 (--lock-tables)", Flag: "--lock-tables"},
	{ID: "lock_all_tables", Group: "一致性与锁", Label: "锁全库 (--lock-all-tables)", Flag: "--lock-all-tables"},
	{ID: "flush_logs", Group: "一致性与锁", Label: "刷新 binlog (--flush-logs)", Flag: "--flush-logs"},
	{ID: "flush_privileges", Group: "一致性与锁", Label: "刷新权限 (--flush-privileges)", Flag: "--flush-privileges"},
	{ID: "source_data_2", Group: "一致性与锁", Label: "记录复制位点注释 (--source-data=2)", Flag: "--source-data=2"},
	{ID: "master_data_2", Group: "一致性与锁", Label: "记录 binlog 位点注释 (--master-data=2)", Flag: "--master-data=2"},
	{ID: "set_gtid_purged_off", Group: "一致性与锁", Label: "GTID 导出 OFF (--set-gtid-purged=OFF)", Flag: "--set-gtid-purged=OFF"},
	{ID: "set_gtid_purged_on", Group: "一致性与锁", Label: "GTID 导出 ON (--set-gtid-purged=ON)", Flag: "--set-gtid-purged=ON"},
	// 对象结构
	{ID: "routines", Group: "对象结构", Label: "存储过程/函数 (--routines)", Flag: "--routines"},
	{ID: "triggers", Group: "对象结构", Label: "触发器 (--triggers)", Flag: "--triggers"},
	{ID: "events", Group: "对象结构", Label: "事件 (--events)", Flag: "--events"},
	{ID: "add_drop_table", Group: "对象结构", Label: "含 DROP TABLE (--add-drop-table)", Flag: "--add-drop-table"},
	{ID: "skip_add_drop_table", Group: "对象结构", Label: "不含 DROP TABLE (--skip-add-drop-table)", Flag: "--skip-add-drop-table"},
	{ID: "add_drop_database", Group: "对象结构", Label: "含 DROP DATABASE (--add-drop-database)", Flag: "--add-drop-database"},
	{ID: "create_options", Group: "对象结构", Label: "含建表选项 (--create-options)", Flag: "--create-options"},
	{ID: "no_create_db", Group: "对象结构", Label: "不输出 CREATE DATABASE (--no-create-db)", Flag: "--no-create-db"},
	// 数据格式
	{ID: "hex_blob", Group: "数据格式", Label: "BLOB 十六进制 (--hex-blob)", Flag: "--hex-blob"},
	{ID: "complete_insert", Group: "数据格式", Label: "INSERT 带列名 (--complete-insert)", Flag: "--complete-insert"},
	{ID: "extended_insert", Group: "数据格式", Label: "合并多行 INSERT (--extended-insert)", Flag: "--extended-insert"},
	{ID: "skip_extended_insert", Group: "数据格式", Label: "单行 INSERT (--skip-extended-insert)", Flag: "--skip-extended-insert"},
	{ID: "replace", Group: "数据格式", Label: "REPLACE 语句 (--replace)", Flag: "--replace"},
	{ID: "insert_ignore", Group: "数据格式", Label: "INSERT IGNORE (--insert-ignore)", Flag: "--insert-ignore"},
	// 导入优化
	{ID: "add_locks", Group: "导入优化", Label: "导入前 LOCK (--add-locks)", Flag: "--add-locks"},
	{ID: "skip_add_locks", Group: "导入优化", Label: "不加 LOCK (--skip-add-locks)", Flag: "--skip-add-locks"},
	{ID: "disable_keys", Group: "导入优化", Label: "导入前禁用键 (--disable-keys)", Flag: "--disable-keys"},
	{ID: "skip_disable_keys", Group: "导入优化", Label: "不禁用键 (--skip-disable-keys)", Flag: "--skip-disable-keys"},
	{ID: "order_by_primary", Group: "导入优化", Label: "按主键排序 (--order-by-primary)", Flag: "--order-by-primary"},
	{ID: "allow_keywords", Group: "导入优化", Label: "允许关键字表名 (--allow-keywords)", Flag: "--allow-keywords"},
	// 字符集与兼容
	{ID: "default_charset_utf8mb4", Group: "字符集与兼容", Label: "字符集 utf8mb4 (--default-character-set=utf8mb4)", Flag: "--default-character-set=utf8mb4"},
	{ID: "set_charset", Group: "字符集与兼容", Label: "输出 SET NAMES (--set-charset)", Flag: "--set-charset"},
	{ID: "skip_set_charset", Group: "字符集与兼容", Label: "不输出 SET NAMES (--skip-set-charset)", Flag: "--skip-set-charset"},
	{ID: "column_statistics_off", Group: "字符集与兼容", Label: "列统计 OFF (--column-statistics=0)", Flag: "--column-statistics=0"},
	{ID: "no_tablespaces", Group: "字符集与兼容", Label: "不导出表空间 (--no-tablespaces)", Flag: "--no-tablespaces"},
	{ID: "compatible", Group: "字符集与兼容", Label: "兼容模式 (--compatible=ansi)", Flag: "--compatible=ansi"},
	// 网络与输出
	{ID: "compress", Group: "网络与输出", Label: "客户端压缩 (--compress)", Flag: "--compress"},
	{ID: "skip_comments", Group: "网络与输出", Label: "省略注释 (--skip-comments)", Flag: "--skip-comments"},
	{ID: "compact", Group: "网络与输出", Label: "紧凑 SQL (--compact)", Flag: "--compact"},
	{ID: "skip_dump_date", Group: "网络与输出", Label: "省略 dump 时间 (--skip-dump-date)", Flag: "--skip-dump-date"},
	{ID: "dump_date", Group: "网络与输出", Label: "输出 dump 时间 (--dump-date)", Flag: "--dump-date"},
	{ID: "tz_utc", Group: "网络与输出", Label: "时区 UTC (--tz-utc)", Flag: "--tz-utc"},
	{ID: "verbose", Group: "网络与输出", Label: "详细日志 (--verbose)", Flag: "--verbose"},
	// 性能与包大小
	{ID: "max_allowed_packet_256m", Group: "性能与包大小", Label: "max_allowed_packet=256M", Flag: "--max-allowed-packet=256M"},
	{ID: "max_allowed_packet_512m", Group: "性能与包大小", Label: "max_allowed_packet=512M", Flag: "--max-allowed-packet=512M"},
	{ID: "max_allowed_packet_1g", Group: "性能与包大小", Label: "max_allowed_packet=1G", Flag: "--max-allowed-packet=1G"},
	{ID: "net_buffer_length_16k", Group: "性能与包大小", Label: "net_buffer_length=16K", Flag: "--net-buffer-length=16384"},
	{ID: "net_buffer_length_1m", Group: "性能与包大小", Label: "net_buffer_length=1M", Flag: "--net-buffer-length=1048576"},
}

var mysqldumpOptionByID map[string]string

func init() {
	mysqldumpOptionByID = make(map[string]string, len(MysqldumpOptionCatalog))
	for _, o := range MysqldumpOptionCatalog {
		mysqldumpOptionByID[o.ID] = o.Flag
	}
}

// DefaultMysqldumpOptionIDs 与历史硬编码行为一致。
func DefaultMysqldumpOptionIDs() []string {
	return []string{"single_transaction", "quick", "routines", "triggers"}
}

const DefaultMysqldumpWorkDir = "/export/backup/yunshu"

var mysqldumpExtraArgsRe = regexp.MustCompile(`^[a-zA-Z0-9_\-./=, ]+$`)

// ParseMysqldumpOptionIDs 解析实例保存的 JSON 选项列表。
func ParseMysqldumpOptionIDs(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return DefaultMysqldumpOptionIDs(), nil
	}
	var ids []string
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return nil, fmt.Errorf("mysqldump_options 须为 JSON 字符串数组")
	}
	if len(ids) == 0 {
		return DefaultMysqldumpOptionIDs(), nil
	}
	out := make([]string, 0, len(ids))
	seen := map[string]struct{}{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := mysqldumpOptionByID[id]; !ok {
			return nil, fmt.Errorf("不支持的 mysqldump 选项: %s", id)
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) == 0 {
		return DefaultMysqldumpOptionIDs(), nil
	}
	return out, nil
}

// FormatMysqldumpFlags 将选项 ID 与额外参数拼为 mysqldump 命令行 flags。
func FormatMysqldumpFlags(optionIDs []string, extraArgs string) (string, error) {
	var flags []string
	for _, id := range optionIDs {
		if f, ok := mysqldumpOptionByID[id]; ok {
			flags = append(flags, f)
		}
	}
	extra := strings.TrimSpace(extraArgs)
	if extra != "" {
		if !mysqldumpExtraArgsRe.MatchString(extra) {
			return "", fmt.Errorf("mysqldump_extra_args 含非法字符")
		}
		for _, p := range strings.Fields(extra) {
			if !strings.HasPrefix(p, "-") {
				return "", fmt.Errorf("额外参数须以 - 开头: %s", p)
			}
		}
		flags = append(flags, extra)
	}
	return strings.Join(flags, " "), nil
}

// NormalizeMysqldumpWorkDir 远端落盘目录，须为绝对路径。
func NormalizeMysqldumpWorkDir(dir string) (string, error) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		dir = DefaultMysqldumpWorkDir
	}
	if !strings.HasPrefix(dir, "/") {
		return "", fmt.Errorf("mysqldump_work_dir 须为绝对路径（以 / 开头）")
	}
	dir = strings.TrimSuffix(dir, "/")
	if dir == "" {
		return "", fmt.Errorf("mysqldump_work_dir 无效")
	}
	return dir, nil
}
