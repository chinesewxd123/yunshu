package service

import (
	"fmt"
	"strings"

	"yunshu/internal/pkg/parseutil"
)

func buildWechatPayload(title string, message string, payload map[string]interface{}, settings map[string]interface{}, atMobiles []string, atUsers []string) map[string]interface{} {
	mode := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", settings["wecomMode"])))
	if mode == "" {
		mode = "robot"
	}

	content := message
	atMobiles = parseutil.UniqueStrings(atMobiles)
	atUsers = parseutil.UniqueStrings(atUsers)
	isAtAll := parseutil.ParseBool(settings["isAtAll"])

	if mode == "robot" && (len(atMobiles) > 0 || len(atUsers) > 0 || isAtAll) {
		return map[string]interface{}{
			"msgtype": "text",
			"text": map[string]interface{}{
				"content":               content,
				"mentioned_list":        atUsers,
				"mentioned_mobile_list": atMobiles,
			},
		}
	}

	return map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"content": content,
		},
	}
}

func buildDingTalkPayload(title string, message string, payload map[string]interface{}, settings map[string]interface{}, atMobiles []string, atUsers []string) map[string]interface{} {
	text := message
	isAtAll := parseutil.ParseBool(settings["isAtAll"])
	return map[string]interface{}{
		"msgtype": "markdown",
		"markdown": map[string]string{
			"title": title,
			"text":  text,
		},
		"at": map[string]interface{}{
			"atMobiles": parseutil.UniqueStrings(atMobiles),
			"atUserIds": parseutil.UniqueStrings(atUsers),
			"isAtAll":   isAtAll,
		},
	}
}
