package service

import (
	"encoding/json"
	"strings"
	"sync"
	"time"

	"yunshu/internal/model"

	"gorm.io/gorm"
)

// ReceiverGroupCache 接收组缓存
type ReceiverGroupCache struct {
	db       *gorm.DB
	mu       sync.RWMutex
	groups   map[uint]*CachedReceiverGroup
	lastLoad time.Time
}

// CachedReceiverGroup 缓存的接收组
type CachedReceiverGroup struct {
	ID              uint
	ProjectID       uint
	Name            string
	ChannelIDs      []uint
	EmailRecipients []string
	ActiveTimeStart *string
	ActiveTimeEnd   *string
	Weekdays        []int
}

// IsActiveNow 检查接收组当前是否生效
func (g *CachedReceiverGroup) IsActiveNow() bool {
	now := time.Now()

	// 检查星期
	if len(g.Weekdays) > 0 {
		weekday := int(now.Weekday())
		found := false
		for _, w := range g.Weekdays {
			if w == weekday {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// 检查时间范围
	if g.ActiveTimeStart != nil && g.ActiveTimeEnd != nil {
		currentTime := now.Format("15:04")
		if currentTime < *g.ActiveTimeStart || currentTime > *g.ActiveTimeEnd {
			return false
		}
	}

	return true
}

// NewReceiverGroupCache 创建接收组缓存
func NewReceiverGroupCache(db *gorm.DB) *ReceiverGroupCache {
	cache := &ReceiverGroupCache{
		db:     db,
		groups: make(map[uint]*CachedReceiverGroup),
	}
	// 启动时预热
	_ = cache.Refresh()
	return cache
}

// Get 获取接收组
func (c *ReceiverGroupCache) Get(id uint) (*CachedReceiverGroup, error) {
	c.mu.RLock()
	if group, ok := c.groups[id]; ok {
		c.mu.RUnlock()
		return group, nil
	}
	c.mu.RUnlock()

	// 未命中，刷新缓存
	if err := c.Refresh(); err != nil {
		return nil, err
	}

	c.mu.RLock()
	defer c.mu.RUnlock()
	if group, ok := c.groups[id]; ok {
		return group, nil
	}
	return nil, nil
}

// Refresh 刷新缓存
func (c *ReceiverGroupCache) Refresh() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// 每30秒刷新一次
	if time.Since(c.lastLoad) < 30*time.Second && len(c.groups) > 0 {
		return nil
	}

	var groups []model.AlertReceiverGroup
	if err := c.db.Where("enabled = ?", true).Find(&groups).Error; err != nil {
		return err
	}

	newGroups := make(map[uint]*CachedReceiverGroup, len(groups))
	for _, g := range groups {
		cached := &CachedReceiverGroup{
			ID:              g.ID,
			ProjectID:       g.ProjectID,
			Name:            g.Name,
			ChannelIDs:      parseUintSliceJSON(g.ChannelIDsJSON),
			EmailRecipients: parseStringSliceJSON(g.EmailRecipientsJSON),
			ActiveTimeStart: g.ActiveTimeStart,
			ActiveTimeEnd:   g.ActiveTimeEnd,
			Weekdays:        parseIntSliceJSON(g.WeekdaysJSON),
		}
		newGroups[g.ID] = cached
	}

	c.groups = newGroups
	c.lastLoad = time.Now()
	return nil
}

// Invalidate 使缓存失效
func (c *ReceiverGroupCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.groups = make(map[uint]*CachedReceiverGroup)
	c.lastLoad = time.Time{}
}

// 辅助函数
func parseStringSliceJSON(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil
	}
	var out []string
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}

func parseIntSliceJSON(raw string) []int {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil
	}
	var out []int
	_ = json.Unmarshal([]byte(raw), &out)
	return out
}
