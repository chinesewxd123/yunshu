package alertdispatch

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// WebhookJSONAPIFailure 解析钉钉/企业微信等「HTTP 200 + JSON errcode」类响应。
// 若存在 errcode 且非 0，返回具体错误文案；无 errcode 字段则视为非此类协议，不覆盖 HTTP 语义。
func WebhookJSONAPIFailure(respBody string) (checked bool, errMsg string) {
	body := strings.TrimSpace(respBody)
	if len(body) == 0 || body[0] != '{' {
		return false, ""
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		return false, ""
	}
	raw, ok := m["errcode"]
	if !ok {
		return false, ""
	}
	code := 0
	switch v := raw.(type) {
	case float64:
		code = int(v)
	case int:
		code = v
	case int64:
		code = int(v)
	case string:
		code, _ = strconv.Atoi(strings.TrimSpace(v))
	default:
		return false, ""
	}
	if code == 0 {
		return true, ""
	}
	msg := strings.TrimSpace(fmt.Sprintf("%v", m["errmsg"]))
	if msg == "" || msg == "<nil>" {
		msg = "errmsg empty"
	}
	return true, fmt.Sprintf("API errcode=%d: %s", code, msg)
}
