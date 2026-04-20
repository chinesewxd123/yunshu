// genopenapi 从 internal/router/router.go 解析 Gin 路由，生成 OpenAPI 3.0.3 文档。
// 用法：go run ./tools/genopenapi -router internal/router/router.go -out docs/apipost/permission-system.openapi.yaml
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

type route struct {
	method string
	path   string
}

func joinURLPath(base, p string) string {
	if p == "" {
		return base
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if base == "" {
		return p
	}
	return strings.TrimSuffix(base, "/") + p
}

func ginToOpenAPIPath(p string) string {
	re := regexp.MustCompile(`:([a-zA-Z0-9_]+)`)
	return re.ReplaceAllString(p, "{$1}")
}

func isPublic(method, path string) bool {
	if method == "GET" && path == "/api/v1/health" {
		return true
	}
	public := map[string]bool{
		"POST /api/v1/auth/verification-code":     true,
		"POST /api/v1/auth/login-code":             true,
		"POST /api/v1/auth/password-login-code":    true,
		"POST /api/v1/auth/login":                  true,
		"POST /api/v1/auth/email-login":            true,
		"POST /api/v1/auth/register":               true,
		"POST /api/v1/alerts/webhook/alertmanager": true,
		"POST /api/v1/agents/public-register":     true,
		"POST /api/v1/agents/health/report":        true,
		"POST /api/v1/agents/discovery/report":     true,
		"GET /api/v1/agents/runtime-config":       true,
	}
	return public[method+" "+path]
}

func tagForPath(path string) string {
	p := strings.TrimPrefix(path, "/api/v1/")
	if p == "" {
		return "System"
	}
	parts := strings.Split(p, "/")
	seg := parts[0]
	switch seg {
	case "auth":
		return "Auth"
	case "health":
		return "System"
	case "alerts":
		if len(parts) > 1 && parts[1] == "webhook" {
			return "AlertsWebhook"
		}
		return "Alerts"
	case "agents":
		return "Agents"
	case "dict":
		return "Dict"
	case "k8s-policies":
		return "K8sScopedPolicy"
	default:
		return upperFirst(seg)
	}
}

func upperFirst(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}

func main() {
	routerPath := flag.String("router", "internal/router/router.go", "path to router.go")
	outPath := flag.String("out", "docs/apipost/permission-system.openapi.yaml", "output openapi yaml")
	flag.Parse()

	f, err := os.Open(*routerPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "open router: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	reEngine := regexp.MustCompile(`^\s*(\w+)\s*:=\s*app\.Engine\.Group\("([^"]*)"\)`)
	reGroup := regexp.MustCompile(`^\s*(\w+)\s*:=\s*(\w+)\.Group\("([^"]*)"\)`)
	reRoute := regexp.MustCompile(`^\s*(\w+)\.(GET|POST|PUT|DELETE|PATCH)\("([^"]*)"`)

	groups := map[string]string{}
	var routes []route
	sc2 := bufio.NewScanner(f)
	for sc2.Scan() {
		line := strings.Split(sc2.Text(), "//")[0]
		if m := reEngine.FindStringSubmatch(line); m != nil {
			groups[m[1]] = m[2]
			continue
		}
		if m := reGroup.FindStringSubmatch(line); m != nil {
			parent := groups[m[2]]
			if parent != "" {
				groups[m[1]] = joinURLPath(parent, m[3])
			}
			continue
		}
		if m := reRoute.FindStringSubmatch(line); m != nil {
			gv, method, suffix := m[1], m[2], m[3]
			base := groups[gv]
			if base == "" {
				continue
			}
			full := joinURLPath(base, suffix)
			routes = append(routes, route{method: method, path: full})
		}
	}
	if err := sc2.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "scan: %v\n", err)
		os.Exit(1)
	}

	// Dedupe by method+path (last wins - should be unique)
	keySeen := map[string]bool{}
	var uniq []route
	for _, r := range routes {
		k := r.method + " " + r.path
		if keySeen[k] {
			continue
		}
		keySeen[k] = true
		uniq = append(uniq, r)
	}
	sort.Slice(uniq, func(i, j int) bool {
		if uniq[i].path != uniq[j].path {
			return uniq[i].path < uniq[j].path
		}
		return uniq[i].method < uniq[j].method
	})

	var sb strings.Builder
	sb.WriteString(`openapi: 3.0.3
info:
  title: YunShu CMDB / Permission System API
  version: "1.0.0"
  description: |
    由 tools/genopenapi 从 internal/router/router.go 自动生成，请勿手工编辑本文件。
    重新生成：go run ./tools/genopenapi -out docs/apipost/permission-system.openapi.yaml
servers:
  - url: http://127.0.0.1:8080
    description: Local
  - url: /
    description: Relative (behind reverse proxy)
tags: []
paths:
`)

	pathOps := map[string]map[string]route{}
	for _, r := range uniq {
		o := ginToOpenAPIPath(r.path)
		if pathOps[o] == nil {
			pathOps[o] = map[string]route{}
		}
		pathOps[o][strings.ToLower(r.method)] = r
	}
	var paths []string
	for p := range pathOps {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	for _, p := range paths {
		ops := pathOps[p]
		var methods []string
		for m := range ops {
			methods = append(methods, m)
		}
		sort.Strings(methods)

		fmt.Fprintf(&sb, "  %s:\n", p)
		for _, lm := range methods {
			r := ops[lm]
			tag := tagForPath(r.path)
			opID := strings.ReplaceAll(strings.TrimPrefix(p, "/"), "/", "_")
			opID = strings.ReplaceAll(opID, "{", "")
			opID = strings.ReplaceAll(opID, "}", "_")
			opID = fmt.Sprintf("%s_%s", strings.ToLower(r.method), opID)
			sec := ""
			if !isPublic(r.method, r.path) {
				sec = `      security:
        - bearerAuth: []
`
			}
			desc := ""
			if strings.Contains(p, "terminal") || strings.Contains(p, "ws") {
				desc = "      description: WebSocket 或需通过查询参数传递 Token，详见 WS 认证中间件。\n"
			}
			fmt.Fprintf(&sb, "    %s:\n", strings.ToLower(r.method))
			fmt.Fprintf(&sb, "      tags: [%q]\n", tag)
			fmt.Fprintf(&sb, "      summary: Auto-generated from router\n")
			fmt.Fprintf(&sb, "      operationId: %s\n", opID)
			if desc != "" {
				fmt.Fprintf(&sb, "%s", desc)
			}
			if sec != "" {
				fmt.Fprintf(&sb, "%s", sec)
			}
			fmt.Fprintf(&sb, `      responses:
        "200":
          description: OK（统一 JSON，见 StandardResponse.data）
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/StandardResponse"
        "400":
          description: 参数错误
        "401":
          description: 未授权
        "403":
          description: 禁止访问
        "500":
          description: 服务器错误
`)
		}
	}

	sb.WriteString(`components:
  securitySchemes:
    bearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT
      description: Authorization Bearer <access_token>
  schemas:
    StandardResponse:
      type: object
      properties:
        code:
          type: integer
          example: 200
        message:
          type: string
          example: success
        error_code:
          type: string
          description: 业务错误码（失败时）
        data:
          description: 成功时的载荷，结构因接口而异
`)

	out := sb.String()
	if err := os.WriteFile(*outPath, []byte(out), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", *outPath, err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "wrote %d paths (%d operations) to %s\n", len(paths), len(uniq), *outPath)
}
