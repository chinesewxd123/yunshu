package service

import "strings"

// tencentAPIRegionFromUserInput 将控制台常见写法（中文地域名、大小写）解析为 CVM API 使用的地域标识（小写 ap-xxx）。
// 无法识别时返回空字符串（调用方可回退为原样 region 或跳过）。
func tencentAPIRegionFromUserInput(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	lower := strings.ToLower(s)
	if strings.HasPrefix(lower, "ap-") || strings.HasPrefix(lower, "na-") || strings.HasPrefix(lower, "eu-") {
		return lower
	}
	// 常见中文 / 简称 → 官方 Region（与腾讯云文档一致，按需扩展）
	aliases := map[string]string{
		"广州": "ap-guangzhou", "华南": "ap-guangzhou", "guangzhou": "ap-guangzhou",
		"北京": "ap-beijing", "华北": "ap-beijing", "beijing": "ap-beijing",
		"上海": "ap-shanghai", "华东": "ap-shanghai", "shanghai": "ap-shanghai",
		"南京": "ap-nanjing", "nanjing": "ap-nanjing",
		"成都": "ap-chengdu", "西南": "ap-chengdu", "chengdu": "ap-chengdu",
		"重庆": "ap-chongqing", "chongqing": "ap-chongqing",
		"深圳": "ap-shenzhen", "shenzhen": "ap-shenzhen",
		"香港": "ap-hongkong", "hongkong": "ap-hongkong", "hk": "ap-hongkong",
		"新加坡": "ap-singapore", "singapore": "ap-singapore",
		"东京": "ap-tokyo", "tokyo": "ap-tokyo",
		"首尔": "ap-seoul", "seoul": "ap-seoul",
		"法兰克福": "eu-frankfurt", "frankfurt": "eu-frankfurt",
		"硅谷": "na-siliconvalley", "圣何塞": "na-siliconvalley",
		"弗吉尼亚": "na-ashburn", "ashburn": "na-ashburn",
		"孟买": "ap-mumbai", "mumbai": "ap-mumbai",
		"曼谷": "ap-bangkok", "bangkok": "ap-bangkok",
		"雅加达": "ap-jakarta", "jakarta": "ap-jakarta",
	}
	if api, ok := aliases[s]; ok {
		return api
	}
	if api, ok := aliases[lower]; ok {
		return api
	}
	return ""
}

func instanceMatchesTencentRegionFilter(instanceRegion string, filter map[string]struct{}) bool {
	if len(filter) == 0 {
		return true
	}
	if _, ok := filter[instanceRegion]; ok {
		return true
	}
	ir := strings.ToLower(strings.TrimSpace(instanceRegion))
	for tok := range filter {
		api := tencentAPIRegionFromUserInput(tok)
		if api != "" && api == ir {
			return true
		}
	}
	return false
}
