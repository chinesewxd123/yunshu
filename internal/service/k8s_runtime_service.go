package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"sync"

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
}

func NewK8sRuntimeService(repo *repository.K8sClusterRepository) *K8sRuntimeService {
	return &K8sRuntimeService{
		repo:           repo,
		registeredHash: map[string]string{},
	}
}

func (s *K8sRuntimeService) ensureKomInit() {
	s.komInitOnce.Do(func() {
		callbacks.RegisterInit()
	})
}

func (s *K8sRuntimeService) registerClusterIfNeeded(clusterID string, kubeconfig string) error {
	s.ensureKomInit()
	sum := sha256.Sum256([]byte(kubeconfig))
	hash := hex.EncodeToString(sum[:])

	s.komMu.Lock()
	prev := s.registeredHash[clusterID]
	if prev != "" && prev == hash {
		s.komMu.Unlock()
		return nil
	}

	f, err := os.CreateTemp("", "kom-kubeconfig-*")
	if err != nil {
		s.komMu.Unlock()
		return err
	}
	tmpPath := f.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := f.WriteString(kubeconfig); err != nil {
		_ = f.Close()
		s.komMu.Unlock()
		return err
	}
	_ = f.Close()

	_, err = kom.Clusters().RegisterByPathWithID(tmpPath, clusterID)
	if err != nil {
		s.komMu.Unlock()
		return err
	}
	s.registeredHash[clusterID] = hash
	s.komMu.Unlock()
	return nil
}

func (s *K8sRuntimeService) DeleteRegisterCache(clusterID uint) {
	s.komMu.Lock()
	delete(s.registeredHash, strconv.FormatUint(uint64(clusterID), 10))
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
	if err := s.registerClusterIfNeeded(clusterID, cluster.Kubeconfig); err != nil {
		return nil, nil, apperror.Internal(fmt.Sprintf("k8s 连接失败: %v", err))
	}
	k := kom.Cluster(clusterID)
	if k == nil {
		return nil, nil, apperror.Internal("k8s 集群实例不存在")
	}
	return cluster, k, nil
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
