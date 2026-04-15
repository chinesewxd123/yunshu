package alertnotify

import "strconv"

// EscapePromQLLabelValue 将字符串编码为 PromQL 双引号 label 值字面量。
func EscapePromQLLabelValue(s string) string {
	return strconv.Quote(s)
}
