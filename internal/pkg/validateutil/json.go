package validateutil

import (
	"encoding/json"
	"strings"

	"go-permission-system/internal/pkg/apperror"
)

func ValidateJSONObjectString(v string, fieldName string) error {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(v), &m); err != nil {
		if strings.TrimSpace(fieldName) == "" {
			fieldName = "字段"
		}
		return apperror.BadRequest(fieldName + " 必须是 JSON 对象字符串")
	}
	return nil
}

