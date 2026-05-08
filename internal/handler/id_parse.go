package handler

import (
	"strconv"
	"yunshu/internal/pkg/constants"

	"github.com/gin-gonic/gin"
)

// parseUintParam parses a uint path parameter and returns a BadRequest app error on failure.
func parseUintParam(c *gin.Context, key string) (uint, error) {
	id, err := strconv.ParseUint(c.Param(key), 10, 64)
	if err != nil {
		return 0, constants.ErrInvalidRequestParam
	}
	return uint(id), nil
}
