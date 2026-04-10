package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"go-permission-system/internal/model"
	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/repository"

	"github.com/weibaohui/kom/callbacks"
	kom "github.com/weibaohui/kom/kom"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type K8sRuntimeService struct {
	repo *repository.K8sClusterRepository

	komInitOnce    sync.Once
	komMu          sync.Mutex
	registeredHash map[string]string
	connState      map[string]ClusterConnState
	regLocks       map[string]*sync.Mutex
}

type ClusterConnState struct {
	State               string    `json:"state"`
	LastError           string    `json:"last_error,omitempty"`
	LastAttemptAt       time.Time `json:"last_attempt_at,omitempty"`
	LastSuccessAt       time.Time `json:"last_success_at,omitempty"`
	ConsecutiveFailures int       `json:"consecutive_failures"`
}

func NewK8sRuntimeService(repo *repository.K8sClusterRepository) *K8sRuntimeService {
	return &K8sRuntimeService{
		repo:           repo,
		registeredHash: map[string]string{},
		connState:      map[string]ClusterConnState{},
		regLocks:       map[string]*sync.Mutex{},
	}
}

func (s *K8sRuntimeService) getRegLock(clusterID string) *sync.Mutex {
	s.komMu.Lock()
	defer s.komMu.Unlock()
	if lk, ok := s.regLocks[clusterID]; ok && lk != nil {
		return lk
	}
	lk := &sync.Mutex{}
	s.regLocks[clusterID] = lk
	return lk
}

func (s *K8sRuntimeService) ensureKomInit() {
	s.komInitOnce.Do(func() {
		callbacks.RegisterInit()
	})
}

func (s *K8sRuntimeService) registerClusterIfNeeded(clusterID string, kubeconfig string, force bool) error {
	s.ensureKomInit()
	sum := sha256.Sum256([]byte(kubeconfig))
	hash := hex.EncodeToString(sum[:])

	lk := s.getRegLock(clusterID)
	lk.Lock()
	defer lk.Unlock()

	s.komMu.Lock()
	st := s.connState[clusterID]
	st.LastAttemptAt = time.Now()
	st.State = "connecting"
	s.connState[clusterID] = st
	prev := s.registeredHash[clusterID]
	if !force && prev != "" && prev == hash {
		st.State = "ready"
		st.LastError = ""
		st.LastSuccessAt = time.Now()
		st.ConsecutiveFailures = 0
		s.connState[clusterID] = st
		s.komMu.Unlock()
		return nil
	}
	s.komMu.Unlock()

	f, err := os.CreateTemp("", "kom-kubeconfig-*")
	if err != nil {
		return err
	}
	tmpPath := f.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := f.WriteString(kubeconfig); err != nil {
		_ = f.Close()
		return err
	}
	_ = f.Close()

	_, err = kom.Clusters().RegisterByPathWithID(tmpPath, clusterID)
	if err != nil {
		s.komMu.Lock()
		st.State = "degraded"
		st.LastError = err.Error()
		st.ConsecutiveFailures++
		s.connState[clusterID] = st
		s.komMu.Unlock()
		return err
	}
	s.komMu.Lock()
	s.registeredHash[clusterID] = hash
	st.State = "ready"
	st.LastError = ""
	st.LastSuccessAt = time.Now()
	st.ConsecutiveFailures = 0
	s.connState[clusterID] = st
	s.komMu.Unlock()
	return nil
}

func (s *K8sRuntimeService) DeleteRegisterCache(clusterID uint) {
	s.komMu.Lock()
	key := strconv.FormatUint(uint64(clusterID), 10)
	delete(s.registeredHash, key)
	st := s.connState[key]
	st.State = "unknown"
	s.connState[key] = st
	s.komMu.Unlock()
}

func (s *K8sRuntimeService) GetClusterKubectl(ctx context.Context, id uint) (*model.K8sCluster, *kom.Kubectl, error) {
	cluster, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if cluster.Status != 1 {
		return nil, nil, apperror.Forbidden("集群已停用")
	}
	clusterID := strconv.FormatUint(uint64(id), 10)
	if err := s.registerClusterIfNeeded(clusterID, cluster.Kubeconfig, false); err != nil {
		return nil, nil, apperror.Internal(fmt.Sprintf("k8s 连接失败: %v", err))
	}
	k := kom.Cluster(clusterID)
	if k == nil {
		return nil, nil, apperror.Internal("k8s 集群实例不存在")
	}
	return cluster, k, nil
}

func (s *K8sRuntimeService) CheckClusterHeartbeat(ctx context.Context, id uint) (string, ClusterConnState, error) {
	cluster, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return "", ClusterConnState{}, err
	}
	if cluster.Status != 1 {
		// 停用集群：不做心跳/重连，直接标记 disabled
		s.komMu.Lock()
		key := strconv.FormatUint(uint64(id), 10)
		st := s.connState[key]
		st.State = "disabled"
		s.connState[key] = st
		s.komMu.Unlock()
		return "", s.GetClusterConnState(id), nil
	}
	clusterID := strconv.FormatUint(uint64(id), 10)
	if err := s.registerClusterIfNeeded(clusterID, cluster.Kubeconfig, false); err != nil {
		return "", s.GetClusterConnState(id), err
	}
	k := kom.Cluster(clusterID)
	if k == nil {
		return "", s.GetClusterConnState(id), apperror.Internal("k8s 集群实例不存在")
	}
	info := k.Status().ServerVersion()
	if info == nil || strings.TrimSpace(info.GitVersion) == "" {
		// 失败后清理并强制重连一次
		s.DeleteRegisterCache(id)
		if e := s.registerClusterIfNeeded(clusterID, cluster.Kubeconfig, true); e != nil {
			return "", s.GetClusterConnState(id), apperror.Internal(fmt.Sprintf("k8s 心跳失败: %v", e))
		}
		k = kom.Cluster(clusterID)
		if k == nil {
			return "", s.GetClusterConnState(id), apperror.Internal("k8s 集群重连失败")
		}
		info = k.Status().ServerVersion()
		if info == nil || strings.TrimSpace(info.GitVersion) == "" {
			s.komMu.Lock()
			st := s.connState[clusterID]
			st.State = "degraded"
			st.LastAttemptAt = time.Now()
			st.LastError = "server version empty"
			st.ConsecutiveFailures++
			s.connState[clusterID] = st
			s.komMu.Unlock()
			return "", s.GetClusterConnState(id), apperror.Internal("k8s 心跳失败: 版本信息为空")
		}
	}
	s.komMu.Lock()
	st := s.connState[clusterID]
	st.State = "ready"
	st.LastAttemptAt = time.Now()
	st.LastSuccessAt = time.Now()
	st.LastError = ""
	st.ConsecutiveFailures = 0
	s.connState[clusterID] = st
	s.komMu.Unlock()
	return info.GitVersion, s.GetClusterConnState(id), nil
}

func (s *K8sRuntimeService) GetClusterConnState(id uint) ClusterConnState {
	s.komMu.Lock()
	defer s.komMu.Unlock()
	key := strconv.FormatUint(uint64(id), 10)
	st := s.connState[key]
	if strings.TrimSpace(st.State) == "" {
		st.State = "unknown"
	}
	return st
}

func (s *K8sRuntimeService) GetClusterRestConfig(ctx context.Context, id uint) (*model.K8sCluster, *rest.Config, error) {
	cluster, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	if cluster.Status != 1 {
		return nil, nil, apperror.Forbidden("集群已停用")
	}
	cfg, err := clientcmd.RESTConfigFromKubeConfig([]byte(cluster.Kubeconfig))
	if err != nil {
		return nil, nil, apperror.Internal(fmt.Sprintf("解析 kubeconfig 失败: %v", err))
	}
	return cluster, cfg, nil
}
