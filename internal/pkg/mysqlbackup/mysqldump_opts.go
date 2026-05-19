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
}

// MysqldumpOptionCatalog 供 API/前端展示。
var MysqldumpOptionCatalog = []MysqldumpOption{
	{ID: "single_transaction", Label: "单事务 (--single-transaction)", Flag: "--single-transaction"},
	{ID: "quick", Label: "快速导出 (--quick)", Flag: "--quick"},
	{ID: "routines", Label: "存储过程/函数 (--routines)", Flag: "--routines"},
	{ID: "triggers", Label: "触发器 (--triggers)", Flag: "--triggers"},
	{ID: "events", Label: "事件 (--events)", Flag: "--events"},
	{ID: "hex_blob", Label: "BLOB 十六进制 (--hex-blob)", Flag: "--hex-blob"},
	{ID: "set_gtid_purged_off", Label: "GTID (--set-gtid-purged=OFF)", Flag: "--set-gtid-purged=OFF"},
	{ID: "no_tablespaces", Label: "不导出表空间 (--no-tablespaces)", Flag: "--no-tablespaces"},
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
