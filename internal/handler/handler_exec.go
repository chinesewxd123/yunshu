package handler

import (
	"context"

	"go-permission-system/internal/pkg/response"

	"github.com/gin-gonic/gin"
)

func handleQuery[T any, R any](c *gin.Context, call func(context.Context, T) (R, error)) {
	var req T
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, err)
		return
	}
	data, err := call(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func handleJSON[T any, R any](c *gin.Context, call func(context.Context, T) (R, error)) {
	var req T
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	data, err := call(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func handleQueryOK[T any](c *gin.Context, okData any, call func(context.Context, T) error) {
	var req T
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, err)
		return
	}
	if err := call(c.Request.Context(), req); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, okData)
}

func handleJSONOK[T any](c *gin.Context, okData any, call func(context.Context, T) error) {
	var req T
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, err)
		return
	}
	if err := call(c.Request.Context(), req); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, okData)
}

func handleQueryWithKind[T any, R any](c *gin.Context, call func(context.Context, string, T) (R, error)) {
	kind := c.Query("kind")
	var req T
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, err)
		return
	}
	data, err := call(c.Request.Context(), kind, req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, data)
}

func handleQueryWithKindOK[T any](c *gin.Context, okData any, call func(context.Context, string, T) error) {
	kind := c.Query("kind")
	var req T
	if err := c.ShouldBindQuery(&req); err != nil {
		response.Error(c, err)
		return
	}
	if err := call(c.Request.Context(), kind, req); err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, okData)
}
