package handler

import (
	"net/http"
	"strconv"

	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

// AlertSubscriptionHandler 订阅树Handler
type AlertSubscriptionHandler struct {
	svc *service.AlertSubscriptionService
}

// NewAlertSubscriptionHandler 创建Handler
func NewAlertSubscriptionHandler(svc *service.AlertSubscriptionService) *AlertSubscriptionHandler {
	return &AlertSubscriptionHandler{svc: svc}
}

// ListNodes godoc
// @Summary 查询订阅树节点列表
// @Description 分页查询订阅树节点，支持按项目、父节点、关键词过滤
// @Tags alert-subscription
// @Accept json
// @Produce json
// @Param project_id query int true "项目ID"
// @Param parent_id query int false "父节点ID，0表示根节点"
// @Param keyword query string false "关键词"
// @Param enabled query bool false "是否启用"
// @Param page query int false "页码，默认1"
// @Param page_size query int false "每页数量，默认20"
// @Success 200 {object} service.AlertSubscriptionListResponse
// @Router /api/v1/alerts/subscriptions [get]
func (h *AlertSubscriptionHandler) ListNodes(c *gin.Context) {
	var query service.AlertSubscriptionNodeListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 转换parent_id
	if parentIDStr := c.Query("parent_id"); parentIDStr != "" {
		if parentIDStr == "0" {
			var zero uint = 0
			query.ParentID = &zero
		} else {
			pid, err := strconv.ParseUint(parentIDStr, 10, 64)
			if err == nil && pid > 0 {
				u := uint(pid)
				query.ParentID = &u
			}
		}
	}

	list, total, page, pageSize, err := h.svc.ListNodes(c.Request.Context(), query)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{
		"list":      list,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetNodeTree godoc
// @Summary 获取完整订阅树
// @Description 获取指定项目的完整订阅树结构（包含所有层级）
// @Tags alert-subscription
// @Accept json
// @Produce json
// @Param project_id query int true "项目ID"
// @Success 200 {object} []model.AlertSubscriptionNode
// @Router /api/v1/alerts/subscriptions/tree [get]
func (h *AlertSubscriptionHandler) GetNodeTree(c *gin.Context) {
	projectIDStr := c.Query("project_id")
	if projectIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "project_id is required"})
		return
	}

	projectID, err := strconv.ParseUint(projectIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid project_id"})
		return
	}

	tree, err := h.svc.GetNodeTree(c.Request.Context(), uint(projectID))
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, tree)
}

// CreateNode godoc
// @Summary 创建订阅树节点
// @Description 创建新的订阅树节点
// @Tags alert-subscription
// @Accept json
// @Produce json
// @Param body body service.AlertSubscriptionNodeUpsertRequest true "节点信息"
// @Success 200 {object} model.AlertSubscriptionNode
// @Router /api/v1/alerts/subscriptions [post]
func (h *AlertSubscriptionHandler) CreateNode(c *gin.Context) {
	var req service.AlertSubscriptionNodeUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	node, err := h.svc.CreateNode(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, node)
}

// UpdateNode godoc
// @Summary 更新订阅树节点
// @Description 更新指定订阅树节点
// @Tags alert-subscription
// @Accept json
// @Produce json
// @Param id path int true "节点ID"
// @Param body body service.AlertSubscriptionNodeUpsertRequest true "节点信息"
// @Success 200 {object} model.AlertSubscriptionNode
// @Router /api/v1/alerts/subscriptions/{id} [put]
func (h *AlertSubscriptionHandler) UpdateNode(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req service.AlertSubscriptionNodeUpsertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	node, err := h.svc.UpdateNode(c.Request.Context(), uint(id), req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, node)
}

// DeleteNode godoc
// @Summary 删除订阅树节点
// @Description 删除指定订阅树节点（有子节点时无法删除）
// @Tags alert-subscription
// @Accept json
// @Produce json
// @Param id path int true "节点ID"
// @Success 200 {object} gin.H
// @Router /api/v1/alerts/subscriptions/{id} [delete]
func (h *AlertSubscriptionHandler) DeleteNode(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := h.svc.DeleteNode(c.Request.Context(), uint(id)); err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, gin.H{"message": "deleted"})
}

// MoveNode godoc
// @Summary 移动订阅树节点
// @Description 移动节点到新的父节点下
// @Tags alert-subscription
// @Accept json
// @Produce json
// @Param id path int true "节点ID"
// @Param body body MoveNodeRequest true "移动信息"
// @Success 200 {object} model.AlertSubscriptionNode
// @Router /api/v1/alerts/subscriptions/{id}/move [post]
type MoveNodeRequest struct {
	NewParentID *uint `json:"new_parent_id"`
}

func (h *AlertSubscriptionHandler) MoveNode(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	var req MoveNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 通过更新父节点实现移动
	node, err := h.svc.GetNodeByID(c.Request.Context(), uint(id))
	if err != nil {
		response.Error(c, err)
		return
	}

	updateReq := service.AlertSubscriptionNodeUpsertRequest{
		ProjectID:             node.ProjectID,
		ParentID:              req.NewParentID,
		Name:                  node.Name,
		Code:                  node.Code,
		MatchLabelsJSON:       node.MatchLabelsJSON,
		MatchRegexJSON:        node.MatchRegexJSON,
		MatchSeverity:         node.MatchSeverity,
		Continue:              node.Continue,
		ReceiverGroupIDsJSON: node.ReceiverGroupIDsJSON,
		SilenceSeconds:        node.SilenceSeconds,
		NotifyResolved:        &node.NotifyResolved,
	}

	updated, err := h.svc.UpdateNode(c.Request.Context(), uint(id), updateReq)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, updated)
}

// migrateFromPoliciesBody 迁移旧策略时的可选参数。
type migrateFromPoliciesBody struct {
	DisableOld       *bool `json:"disable_old"`
	DefaultProjectID *uint `json:"default_project_id"`
}

// MigrateFromPolicies 将旧告警策略迁移为订阅树节点 + 接收组。
func (h *AlertSubscriptionHandler) MigrateFromPolicies(c *gin.Context) {
	var body migrateFromPoliciesBody
	_ = c.ShouldBindJSON(&body)
	disableOld := true
	if body.DisableOld != nil {
		disableOld = *body.DisableOld
	}
	var defaultPID uint
	if body.DefaultProjectID != nil {
		defaultPID = *body.DefaultProjectID
	}
	rep, err := h.svc.MigrateFromPolicies(c.Request.Context(), service.MigrateFromPoliciesOptions{
		DisableOld:       disableOld,
		DefaultProjectID: defaultPID,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, rep)
}
