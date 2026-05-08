package validateutil

import (
	"encoding/json"
	"fmt"
	"strings"

	"yunshu/internal/pkg/constants"
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
		return constants.ErrBadRequestWithMsg(fmt.Sprintf(constants.ErrFmtJSONFieldMustBeObject, fieldName))
	}
	return nil
}
