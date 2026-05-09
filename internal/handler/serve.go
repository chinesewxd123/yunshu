package handler

import (
	"context"

	"yunshu/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

// 本文件提供「路由层一句话」的薄封装，减少重复的 List/Create/Update/Delete 样板代码。
// 批量将 handleQuery/handleJSON/… 换为 Serve*：python scripts/batch_serve_handlers.py
// 复杂场景（多路径参数、非标准响应体）仍可直接写 gin.HandlerFunc；底层绑定仍见 handler_exec.go。

const defaultIDParam = "id"

// ServeQuery：GET + Query 绑定 → 业务函数 → 200 + data
func ServeQuery[T any, R any](c *gin.Context, fn func(context.Context, T) (R, error)) {
	handleQuery(c, fn)
}

// ServeJSON：JSON 绑定 → 业务函数 → 200 + data
func ServeJSON[T any, R any](c *gin.Context, fn func(context.Context, T) (R, error)) {
	handleJSON(c, fn)
}

// ServeJSON201：JSON 绑定 → 业务函数 → 201 + data
func ServeJSON201[T any, R any](c *gin.Context, fn func(context.Context, T) (R, error)) {
	handleJSONCreated(c, fn)
}

// ServeQueryOK：Query 绑定 → 仅副作用 → 固定 okData
func ServeQueryOK[T any](c *gin.Context, okData any, fn func(context.Context, T) error) {
	handleQueryOK(c, okData, fn)
}

// ServeJSONOK：JSON 绑定 → 仅副作用 → 固定 okData
func ServeJSONOK[T any](c *gin.Context, okData any, fn func(context.Context, T) error) {
	handleJSONOK(c, okData, fn)
}

// ServeQueryWithKind：Query + kind 字符串 → 业务函数
func ServeQueryWithKind[T any, R any](c *gin.Context, fn func(context.Context, string, T) (R, error)) {
	handleQueryWithKind(c, fn)
}

// ServeQueryWithKindOK：Query + kind → 仅副作用 → 固定 okData
func ServeQueryWithKindOK[T any](c *gin.Context, okData any, fn func(context.Context, string, T) error) {
	handleQueryWithKindOK(c, okData, fn)
}

// ServeDelete：路径参数 id（可改 idParam）→ Delete(ctx, id) → 固定成功体
func ServeDelete(c *gin.Context, fn func(context.Context, uint) error, idParam string) {
	if idParam == "" {
		idParam = defaultIDParam
	}
	id, err := parseUintParam(c, idParam)
	if err != nil {
		response.Error(c, err)
		return
	}
	if err := fn(c.Request.Context(), id); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, gin.H{"message": "deleted"})
}

// ServePatch：路径参数 id + JSON 体 → fn(ctx, id, body) → 200 + data
func ServePatch[Req any, Resp any](c *gin.Context, fn func(context.Context, uint, Req) (Resp, error), idParam string) {
	if idParam == "" {
		idParam = defaultIDParam
	}
	id, err := parseUintParam(c, idParam)
	if err != nil {
		response.Error(c, err)
		return
	}
	handleJSON(c, func(ctx context.Context, body Req) (Resp, error) {
		return fn(ctx, id, body)
	})
}
