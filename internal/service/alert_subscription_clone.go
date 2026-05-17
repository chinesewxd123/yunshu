package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"yunshu/internal/model"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/service/svcerr"

	"gorm.io/gorm"
)

// CloneProjectRoutingRequest 从源项目复制订阅树 + 接收组到目标项目。
type CloneProjectRoutingRequest struct {
	SourceProjectID uint `json:"source_project_id" binding:"required"`
	TargetProjectID uint `json:"target_project_id" binding:"required"`
	// ReplaceCluster 非空时覆盖 match_labels 中的 cluster（常用于新环境/新数据源）。
	ReplaceCluster string `json:"replace_cluster"`
	// ReplaceRoute 非空时覆盖 match_labels 中的 route。
	ReplaceRoute string `json:"replace_route"`
	// IncludeDisabled 是否复制已停用的节点与接收组。
	IncludeDisabled bool `json:"include_disabled"`
	// SkipIfTargetHasNodes 目标项目已有订阅节点时直接返回（不覆盖）。
	SkipIfTargetHasNodes bool `json:"skip_if_target_has_nodes"`
}

// CloneProjectRoutingReport 复制结果统计。
type CloneProjectRoutingReport struct {
	ReceiverGroupsCreated int  `json:"receiver_groups_created"`
	NodesCreated          int  `json:"nodes_created"`
	Skipped               bool `json:"skipped"`
	Message               string `json:"message,omitempty"`
}

func replaceLabelsJSON(raw string, replaceCluster, replaceRoute string, targetProjectID uint) string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" {
		if targetProjectID > 0 {
			return fmt.Sprintf(`{"project_id":"%d"}`, targetProjectID)
		}
		return "{}"
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(raw), &m); err != nil || m == nil {
		m = map[string]string{}
	}
	if rc := strings.TrimSpace(replaceCluster); rc != "" {
		m["cluster"] = rc
	}
	if rr := strings.TrimSpace(replaceRoute); rr != "" {
		m["route"] = rr
	}
	if targetProjectID > 0 {
		m["project_id"] = fmt.Sprintf("%d", targetProjectID)
	}
	bs, err := json.Marshal(m)
	if err != nil {
		return raw
	}
	return string(bs)
}

// CloneProjectRouting 复制订阅路由模板（接收组 + 订阅树节点）。
func (s *AlertSubscriptionService) CloneProjectRouting(ctx context.Context, req CloneProjectRoutingRequest) (*CloneProjectRoutingReport, error) {
	if s == nil || s.db == nil {
		return nil, svcerr.InternalMsg("alert.subscription", "api", "subscription service unavailable")
	}
	if req.SourceProjectID == 0 || req.TargetProjectID == 0 {
		return nil, constants.ErrBadRequestWithMsg("source_project_id 与 target_project_id 不能为空")
	}
	if req.SourceProjectID == req.TargetProjectID {
		return nil, constants.ErrBadRequestWithMsg("源项目与目标项目不能相同")
	}

	rep := &CloneProjectRoutingReport{}

	var targetCount int64
	if err := s.db.WithContext(ctx).Model(&model.AlertSubscriptionNode{}).
		Where("project_id = ?", req.TargetProjectID).
		Count(&targetCount).Error; err != nil {
		return nil, svcerr.Pass("alert.subscription", "CloneProjectRouting", err)
	}
	if targetCount > 0 && req.SkipIfTargetHasNodes {
		rep.Skipped = true
		rep.Message = "目标项目已有订阅节点，已跳过（可取消「目标为空才复制」后重试或手动清理目标项目）"
		return rep, nil
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if targetCount > 0 && !req.SkipIfTargetHasNodes {
			if err := tx.Where("project_id = ?", req.TargetProjectID).Delete(&model.AlertSubscriptionNode{}).Error; err != nil {
				return svcerr.Pass("alert.subscription", "CloneProjectRouting", err)
			}
			if err := tx.Where("project_id = ?", req.TargetProjectID).Delete(&model.AlertReceiverGroup{}).Error; err != nil {
				return svcerr.Pass("alert.subscription", "CloneProjectRouting", err)
			}
		}

		rgTx := tx.Model(&model.AlertReceiverGroup{}).Where("project_id = ?", req.SourceProjectID)
		if !req.IncludeDisabled {
			rgTx = rgTx.Where("enabled = ?", true)
		}
		var srcGroups []model.AlertReceiverGroup
		if err := rgTx.Order("id ASC").Find(&srcGroups).Error; err != nil {
			return svcerr.Pass("alert.subscription", "CloneProjectRouting", err)
		}
		rgMap := make(map[uint]uint, len(srcGroups))
		for i := range srcGroups {
			g := srcGroups[i]
			row := model.AlertReceiverGroup{
				ProjectID:           req.TargetProjectID,
				Name:                strings.TrimSpace(g.Name),
				Description:         strings.TrimSpace(g.Description),
				ChannelIDsJSON:      strings.TrimSpace(g.ChannelIDsJSON),
				EmailRecipientsJSON: strings.TrimSpace(g.EmailRecipientsJSON),
				ActiveTimeStart:     g.ActiveTimeStart,
				ActiveTimeEnd:       g.ActiveTimeEnd,
				WeekdaysJSON:        strings.TrimSpace(g.WeekdaysJSON),
				EscalationLevel:     g.EscalationLevel,
				Enabled:             g.Enabled,
			}
			if row.ChannelIDsJSON == "" {
				row.ChannelIDsJSON = "[]"
			}
			if row.EmailRecipientsJSON == "" {
				row.EmailRecipientsJSON = "[]"
			}
			if err := tx.Create(&row).Error; err != nil {
				return svcerr.Pass("alert.subscription", "CloneProjectRouting", err)
			}
			rgMap[g.ID] = row.ID
			rep.ReceiverGroupsCreated++
		}

		nodeTx := tx.Model(&model.AlertSubscriptionNode{}).Where("project_id = ?", req.SourceProjectID)
		if !req.IncludeDisabled {
			nodeTx = nodeTx.Where("enabled = ?", true)
		}
		var srcNodes []model.AlertSubscriptionNode
		if err := nodeTx.Order("level ASC, id ASC").Find(&srcNodes).Error; err != nil {
			return svcerr.Pass("alert.subscription", "CloneProjectRouting", err)
		}
		if len(srcNodes) == 0 {
			return nil
		}

		nodeMap := make(map[uint]uint, len(srcNodes))
		for _, sn := range srcNodes {
			var newParentID *uint
			if sn.ParentID != nil && *sn.ParentID > 0 {
				if np, ok := nodeMap[*sn.ParentID]; ok {
					newParentID = &np
				} else {
					continue
				}
			}

			rgIDs := parseUintSliceJSON(sn.ReceiverGroupIDsJSON)
			newRGIDs := make([]uint, 0, len(rgIDs))
			for _, oid := range rgIDs {
				if nid, ok := rgMap[oid]; ok {
					newRGIDs = append(newRGIDs, nid)
				}
			}
			rgJSON := "[]"
			if len(newRGIDs) > 0 {
				bs, _ := json.Marshal(newRGIDs)
				rgJSON = string(bs)
			}

			level := 0
			path := fmt.Sprintf("/%d", req.TargetProjectID)
			if newParentID != nil {
				var parent model.AlertSubscriptionNode
				if err := tx.First(&parent, *newParentID).Error; err != nil {
					return svcerr.Pass("alert.subscription", "CloneProjectRouting", err)
				}
				level = parent.Level + 1
				path = fmt.Sprintf("%s/%d", parent.Path, *newParentID)
			}

			row := model.AlertSubscriptionNode{
				ProjectID:            req.TargetProjectID,
				ParentID:             newParentID,
				Level:                level,
				Path:                 path,
				Name:                 strings.TrimSpace(sn.Name),
				Code:                 strings.TrimSpace(sn.Code),
				MatchLabelsJSON:      replaceLabelsJSON(sn.MatchLabelsJSON, req.ReplaceCluster, req.ReplaceRoute, req.TargetProjectID),
				MatchRegexJSON:       safeJSONObj(sn.MatchRegexJSON),
				MatchSeverity:        strings.TrimSpace(sn.MatchSeverity),
				Continue:             sn.Continue,
				Enabled:              sn.Enabled,
				ReceiverGroupIDsJSON: rgJSON,
				SilenceSeconds:       sn.SilenceSeconds,
				NotifyResolved:       sn.NotifyResolved,
			}
			if row.MatchLabelsJSON == "" {
				row.MatchLabelsJSON = "{}"
			}
			if err := tx.Create(&row).Error; err != nil {
				return svcerr.Pass("alert.subscription", "CloneProjectRouting", err)
			}
			nodeMap[sn.ID] = row.ID
			rep.NodesCreated++
		}
		return nil
	})
	if err != nil {
		return nil, svcerr.Pass("alert.subscription", "CloneProjectRouting", err)
	}

	s.InvalidateCache()
	if rep.NodesCreated == 0 && rep.ReceiverGroupsCreated == 0 {
		rep.Message = "源项目无订阅节点可复制"
	}
	return rep, nil
}
