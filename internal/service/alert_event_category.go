package service

import (
	"strings"

	"gorm.io/gorm"
)

// AlertEventCategory 历史告警策略分类（与 web/src/utils/alert-event-reasons.ts 对齐）。
const (
	AlertEventCategoryDelivery    = "delivery"
	AlertEventCategoryRouting     = "routing"
	AlertEventCategorySilence     = "silence"
	AlertEventCategoryInhibition  = "inhibition"
	AlertEventCategoryTiming      = "timing"
	AlertEventCategoryResolved    = "resolved"
	AlertEventCategoryFailure     = "failure"
	AlertEventCategoryOther       = "other"
)

// ValidAlertEventCategory 是否为支持的 category 查询参数。
func ValidAlertEventCategory(category string) bool {
	switch strings.TrimSpace(strings.ToLower(category)) {
	case AlertEventCategoryDelivery,
		AlertEventCategoryRouting,
		AlertEventCategorySilence,
		AlertEventCategoryInhibition,
		AlertEventCategoryTiming,
		AlertEventCategoryResolved,
		AlertEventCategoryFailure,
		AlertEventCategoryOther:
		return true
	default:
		return false
	}
}

func applyAlertEventCategoryFilter(tx *gorm.DB, category string) *gorm.DB {
	cat := strings.TrimSpace(strings.ToLower(category))
	if cat == "" {
		return tx
	}
	switch cat {
	case AlertEventCategoryInhibition:
		return tx.Where("error_message LIKE ?", "inhibition_suppressed:%")
	case AlertEventCategorySilence:
		return tx.Where("error_message IN ?", []string{"silence_suppressed", "subscription_suppressed"})
	case AlertEventCategoryTiming:
		return tx.Where("error_message IN ?", []string{
			"group_wait_suppressed",
			"group_interval_suppressed",
			"repeat_suppressed",
			"group_throttled",
		})
	case AlertEventCategoryResolved:
		return tx.Where("error_message IN ?", []string{
			"resolved_aggregate_suppressed",
			"resolved_no_prior_firing_delivery",
		})
	case AlertEventCategoryRouting:
		return tx.Where("error_message IN ?", []string{
			"no_policy_matched",
			"no_enabled_channels",
			"no_channel_matched",
			"no_channel_matched_subscription",
		})
	case AlertEventCategoryFailure:
		return tx.Where("success = ? OR error_message = ?", false, "all_channel_delivery_failed")
	case AlertEventCategoryDelivery:
		return tx.Where(
			"success = ? AND channel_id > 0 AND (error_message IS NULL OR TRIM(error_message) = '')",
			true,
		)
	case AlertEventCategoryOther:
		return tx.Where(
			`success = ? AND TRIM(COALESCE(error_message, '')) != '' 
AND error_message NOT LIKE ? 
AND error_message NOT IN ?`,
			true,
			"inhibition_suppressed:%",
			[]string{
				"silence_suppressed", "subscription_suppressed",
				"group_wait_suppressed", "group_interval_suppressed", "repeat_suppressed", "group_throttled",
				"resolved_aggregate_suppressed", "resolved_no_prior_firing_delivery",
				"no_policy_matched", "no_enabled_channels", "no_channel_matched", "no_channel_matched_subscription",
				"all_channel_delivery_failed",
			},
		)
	default:
		return tx
	}
}
