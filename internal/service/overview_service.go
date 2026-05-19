package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"yunshu/internal/pkg/auth"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/service/svcerr"
	"yunshu/internal/pkg/k8sauth"

	"yunshu/internal/model"
	"yunshu/internal/repository"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
)

// logAgentOnlineWindow 须与 LogAgentService 心跳超时一致，用于总览「在线 Agent」计数。
const logAgentOnlineWindow = 90 * time.Second

type OverviewResponse struct {
	UsersCount    int64 `json:"users_count"`
	ClustersCount int64 `json:"clusters_count"`

	PendingRegistrationsCount int64 `json:"pending_registrations_count"`
	ServersCount              int64 `json:"servers_count"`

	PodNormalCount   int64 `json:"pod_normal_count"`
	PodAbnormalCount int64 `json:"pod_abnormal_count"`

	// Number of clusters that failed during pod aggregation.
	PodClusterErrors int64 `json:"pod_cluster_errors"`

	// Event stats (sampled per cluster to control latency).
	EventTotalCount    int64 `json:"event_total_count"`
	EventWarningCount  int64 `json:"event_warning_count"`
	EventClusterErrors int64 `json:"event_cluster_errors"`

	// Alert events（告警监控）
	AlertFiringCount      int64 `json:"alert_firing_count"`
	AlertEventsTodayCount int64 `json:"alert_events_today_count"`

	// Log agents（与 Agent 列表「在线」判定一致：最近心跳在 90s 内）
	LogAgentsOnlineCount  int64 `json:"log_agents_online_count"`
	LogAgentsOfflineCount int64 `json:"log_agents_offline_count"`
}

type OverviewTrendsResponse struct {
	Days           []string `json:"days"`
	LoginSuccess   []int64  `json:"login_success"`
	LoginFail      []int64  `json:"login_fail"`
	OperationTotal []int64  `json:"operation_total"`
}

type OverviewService struct {
	db         *gorm.DB
	runtime    *K8sRuntimeService
	redis      *redis.Client
	memberRepo *repository.ProjectMemberRepository
	accessRepo *repository.K8sClusterAccessRepository
}

// NewOverviewService 创建相关逻辑。
func NewOverviewService(
	db *gorm.DB,
	runtime *K8sRuntimeService,
	redisClient *redis.Client,
	memberRepo *repository.ProjectMemberRepository,
	accessRepo *repository.K8sClusterAccessRepository,
) *OverviewService {
	return &OverviewService{db: db, runtime: runtime, redis: redisClient, memberRepo: memberRepo, accessRepo: accessRepo}
}

// Trends 执行对应的业务逻辑。
func (s *OverviewService) Trends(ctx context.Context, days int) (*OverviewTrendsResponse, error) {
	if s.db == nil {
		return nil, constants.ErrInternal
	}
	if days <= 0 || days > 31 {
		days = 7
	}

	cacheKey := fmt.Sprintf("overview:trends:v1:days:%d", days)
	if s.redis != nil {
		if raw, err := s.redis.Get(ctx, cacheKey).Result(); err == nil && raw != "" {
			var cached OverviewTrendsResponse
			if json.Unmarshal([]byte(raw), &cached) == nil {
				return &cached, nil
			}
		}
	}

	now := time.Now()
	loc := now.Location()
	end := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc).AddDate(0, 0, 1)
	start := end.AddDate(0, 0, -days)

	out := &OverviewTrendsResponse{
		Days:           make([]string, 0, days),
		LoginSuccess:   make([]int64, days),
		LoginFail:      make([]int64, days),
		OperationTotal: make([]int64, days),
	}

	index := make(map[string]int, days)
	for i := 0; i < days; i++ {
		d := start.AddDate(0, 0, i)
		key := d.Format("2006-01-02")
		out.Days = append(out.Days, d.Format("01-02"))
		index[key] = i
	}

	type row struct {
		Day string
		Cnt int64
	}

	dialect := s.db.Dialector.Name()
	dayExpr := "DATE(created_at)"
	switch dialect {
	case "postgres":
		dayExpr = "to_char(created_at, 'YYYY-MM-DD')"
	case "mysql":
		dayExpr = "DATE_FORMAT(created_at, '%Y-%m-%d')"
	case "sqlite":
		dayExpr = "strftime('%Y-%m-%d', created_at)"
	}

	loadCounts := func(table string, where string, args ...any) (map[string]int64, error) {
		var rows []row
		query := fmt.Sprintf("SELECT %s AS day, COUNT(*) AS cnt FROM %s WHERE created_at >= ? AND created_at < ? %s GROUP BY %s",
			dayExpr, table, where, dayExpr)
		allArgs := append([]any{start, end}, args...)
		if err := s.db.WithContext(ctx).Raw(query, allArgs...).Scan(&rows).Error; err != nil {
			return nil, svcerr.Pass(ctx, "overview", "Trends", err)
		}
		m := make(map[string]int64, len(rows))
		for _, r := range rows {
			m[r.Day] = r.Cnt
		}
		return m, nil
	}

	successCounts, err := loadCounts("login_logs", "AND status = ?", model.LoginLogStatusSuccess)
	if err != nil {
		return nil, svcerr.Pass(ctx, "overview", "Trends", err)
	}
	failCounts, err := loadCounts("login_logs", "AND status = ?", model.LoginLogStatusFail)
	if err != nil {
		return nil, svcerr.Pass(ctx, "overview", "Trends", err)
	}
	opCounts, err := loadCounts("operation_logs", "")
	if err != nil {
		return nil, svcerr.Pass(ctx, "overview", "Trends", err)
	}

	for day, i := range index {
		if v, ok := successCounts[day]; ok {
			out.LoginSuccess[i] = v
		}
		if v, ok := failCounts[day]; ok {
			out.LoginFail[i] = v
		}
		if v, ok := opCounts[day]; ok {
			out.OperationTotal[i] = v
		}
	}

	if s.redis != nil {
		if b, err := json.Marshal(out); err == nil {
			_ = s.redis.Set(ctx, cacheKey, string(b), 60*time.Second).Err()
		}
	}
	return out, nil
}

// Get 获取相关的业务逻辑。
func (s *OverviewService) Get(ctx context.Context) (*OverviewResponse, error) {
	if s.db == nil {
		return nil, constants.ErrInternal
	}

	cacheKey := overviewMetricsCacheKey(ctx)
	if s.redis != nil {
		if raw, err := s.redis.Get(ctx, cacheKey).Result(); err == nil && raw != "" {
			var cached OverviewResponse
			if json.Unmarshal([]byte(raw), &cached) == nil {
				return &cached, nil
			}
		}
	}

	out := &OverviewResponse{}
	if err := s.db.WithContext(ctx).Raw(
		`SELECT
			(SELECT COUNT(*) FROM users WHERE deleted_at IS NULL) AS users_count,
			(SELECT COUNT(*) FROM k8s_clusters WHERE deleted_at IS NULL) AS clusters_count,
			(SELECT COUNT(*) FROM registration_requests WHERE status = ?) AS pending_registrations_count,
			(SELECT COUNT(*) FROM servers WHERE deleted_at IS NULL) AS servers_count`,
		model.RegistrationPending,
	).Scan(out).Error; err != nil {
		return nil, svcerr.Pass(ctx, "overview", "Get", err)
	}

	s.fillOverviewAlertAndAgents(ctx, out)

	// Pod stats: aggregate across enabled clusters visible to current user.
	var clusters []model.K8sCluster
	clusterQ := s.db.WithContext(ctx).Model(&model.K8sCluster{}).Where("status = ?", 1)
	if u, ok := auth.RequestUserFromContext(ctx); ok && u != nil && !auth.IsSuperAdminRole(u.RoleCodes) && s.memberRepo != nil {
		ids, _ := s.memberRepo.ListProjectIDsByUser(ctx, u.ID)
		if len(ids) == 0 {
			clusterQ = clusterQ.Where("owning_project_id IS NULL")
		} else {
			clusterQ = clusterQ.Where("(owning_project_id IS NULL OR owning_project_id IN ?)", ids)
		}
	}
	if err := clusterQ.Find(&clusters).Error; err != nil {
		return nil, svcerr.Pass(ctx, "overview", "Get", err)
	}
	clusters = s.filterOverviewClusters(ctx, clusters)
	if len(clusters) == 0 {
		if s.redis != nil {
			if b, err := json.Marshal(out); err == nil {
				_ = s.redis.Set(ctx, cacheKey, string(b), 15*time.Second).Err()
			}
		}
		return out, nil
	}

	// 产品侧体验优先：总时限内返回“可得数据 + 失败计数”，而不是让首页等待到超时。
	overallCtx, overallCancel := context.WithTimeout(ctx, 8*time.Second)
	defer overallCancel()

	var (
		mu sync.Mutex
		wg sync.WaitGroup
	)
	for _, c := range clusters {
		cid := c.ID
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Guard against unexpected panics from 3rd-party k8s wrapper (kom),
			// so one cluster failure won't crash the whole backend process.
			defer func() {
				if r := recover(); r != nil {
					mu.Lock()
					out.PodClusterErrors++
					out.EventClusterErrors++
					mu.Unlock()
				}
			}()

			// 连接探测保持短超时，避免不可达集群拖慢全局。
			cctx, cancel := context.WithTimeout(overallCtx, 2*time.Second)
			_, k, err := s.runtime.GetClusterKubectl(cctx, cid)
			cancel()
			if err != nil {
				mu.Lock()
				out.PodClusterErrors++
				mu.Unlock()
				return
			}

			// Pod 聚合也限制时长，超时按失败集群处理。
			pctx, pcancel := context.WithTimeout(overallCtx, 4*time.Second)
			var pods []corev1.Pod
			podQuery := k.WithContext(pctx)
			if podQuery == nil {
				pcancel()
				mu.Lock()
				out.PodClusterErrors++
				mu.Unlock()
				return
			}
			err = podQuery.Resource(&corev1.Pod{}).AllNamespace().List(&pods).Error
			pcancel()
			if err != nil {
				mu.Lock()
				out.PodClusterErrors++
				mu.Unlock()
				return
			}

			normal := int64(0)
			abnormal := int64(0)
			for _, p := range pods {
				if isPodNormal(p) {
					normal++
				} else {
					abnormal++
				}
			}
			mu.Lock()
			out.PodNormalCount += normal
			out.PodAbnormalCount += abnormal
			mu.Unlock()

			// Event 概览仅采样最近 500 条，避免在大集群拖慢首页。
			ectx, ecancel := context.WithTimeout(overallCtx, 4*time.Second)
			var events []corev1.Event
			eventQuery := k.WithContext(ectx)
			if eventQuery == nil {
				ecancel()
				mu.Lock()
				out.EventClusterErrors++
				mu.Unlock()
				return
			}
			err = eventQuery.Resource(&corev1.Event{}).AllNamespace().Limit(500).List(&events).Error
			ecancel()
			if err != nil {
				mu.Lock()
				out.EventClusterErrors++
				mu.Unlock()
				return
			}
			total := int64(len(events))
			warnings := int64(0)
			for _, ev := range events {
				if ev.Type == "Warning" {
					warnings++
				}
			}
			mu.Lock()
			out.EventTotalCount += total
			out.EventWarningCount += warnings
			mu.Unlock()
		}()
	}
	wg.Wait()

	if s.redis != nil {
		if b, err := json.Marshal(out); err == nil {
			_ = s.redis.Set(ctx, cacheKey, string(b), 15*time.Second).Err()
		}
	}
	return out, nil
}

// fillOverviewAlertAndAgents 聚合告警与日志 Agent 指标；表不存在或查询失败时保持 0，不阻断总览。
func (s *OverviewService) fillOverviewAlertAndAgents(ctx context.Context, out *OverviewResponse) {
	if s.db == nil {
		return
	}
	cutoff := time.Now().Add(-logAgentOnlineWindow)

	_ = s.db.WithContext(ctx).Model(&model.AlertEvent{}).Where("status = ?", "firing").Count(&out.AlertFiringCount).Error

	now := time.Now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dayEnd := dayStart.Add(24 * time.Hour)
	_ = s.db.WithContext(ctx).Model(&model.AlertEvent{}).
		Where("created_at >= ? AND created_at < ?", dayStart, dayEnd).
		Count(&out.AlertEventsTodayCount).Error

	var online int64
	_ = s.db.WithContext(ctx).Model(&model.LogAgent{}).
		Where("deleted_at IS NULL AND status = ?", 1).
		Where("last_seen_at IS NOT NULL AND last_seen_at >= ?", cutoff).
		Count(&online).Error
	out.LogAgentsOnlineCount = online

	var totalAgents int64
	_ = s.db.WithContext(ctx).Model(&model.LogAgent{}).
		Where("deleted_at IS NULL AND status = ?", 1).
		Count(&totalAgents).Error
	if totalAgents >= online {
		out.LogAgentsOfflineCount = totalAgents - online
	}
}

func isPodNormal(p corev1.Pod) bool {
	// A pragmatic definition:
	// - phase is Running
	// - all containers are ready (or no container status found -> abnormal)
	if string(p.Status.Phase) != "Running" {
		return false
	}
	if len(p.Status.ContainerStatuses) == 0 {
		return false
	}
	for _, st := range p.Status.ContainerStatuses {
		if !st.Ready {
			return false
		}
	}
	return true
}

func overviewMetricsCacheKey(ctx context.Context) string {
	base := "overview:metrics:v4"
	u, ok := auth.RequestUserFromContext(ctx)
	if !ok || u == nil {
		return base + ":anon"
	}
	if auth.IsSuperAdminRole(u.RoleCodes) {
		return base + ":super"
	}
	return fmt.Sprintf("%s:u:%d", base, u.ID)
}

func (s *OverviewService) filterOverviewClusters(ctx context.Context, clusters []model.K8sCluster) []model.K8sCluster {
	if len(clusters) == 0 {
		return clusters
	}
	u, ok := auth.RequestUserFromContext(ctx)
	if !ok || u == nil || auth.IsSuperAdminRole(u.RoleCodes) {
		return clusters
	}
	if s.accessRepo == nil {
		return nil
	}
	pack := k8sauth.PackFromCurrentUser(u)
	idx, err := s.accessRepo.BuildEffectiveTierIndex(ctx, pack)
	if err != nil {
		return nil
	}
	if idx.GlobalRank < K8sAccessRankReadonly && len(idx.PerCluster) == 0 {
		return nil
	}
	out := make([]model.K8sCluster, 0, len(clusters))
	for _, c := range clusters {
		if idx.ClusterAccessible(c.ID, K8sAccessRankReadonly) {
			out = append(out, c)
		}
	}
	return out
}

// String 的功能实现。
func (o OverviewResponse) String() string {
	return fmt.Sprintf("users=%d clusters=%d alerts_firing=%d alerts_today=%d agents_on=%d agents_off=%d pod_normal=%d pod_abnormal=%d pod_errors=%d event_total=%d event_warning=%d event_errors=%d",
		o.UsersCount, o.ClustersCount, o.AlertFiringCount, o.AlertEventsTodayCount, o.LogAgentsOnlineCount, o.LogAgentsOfflineCount,
		o.PodNormalCount, o.PodAbnormalCount, o.PodClusterErrors, o.EventTotalCount, o.EventWarningCount, o.EventClusterErrors)
}
