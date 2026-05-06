package service

import (
	"encoding/json"
	"regexp"
	"strings"
)

func parseMapJSON(raw string) map[string]string {
	out := map[string]string{}
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" {
		return out
	}
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func parseMapJSONSafe(raw string) map[string]string {
	return parseMapJSON(raw)
}

func parseUintSliceJSON(raw string) []uint {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil
	}
	var out []uint
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func compileRegexMapSafe(raw string) map[string]*regexp.Regexp {
	out := map[string]*regexp.Regexp{}
	for k, v := range parseMapJSONSafe(raw) {
		if re, err := regexp.Compile(v); err == nil {
			out[k] = re
		}
	}
	return out
}

