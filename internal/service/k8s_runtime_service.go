package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
	"yunshu/internal/pkg/apperror"
	"yunshu/internal/pkg/constants"
	logx "yunshu/internal/pkg/logger"
	"yunshu/internal/service/svcerr"

	"yunshu/internal/model"
	"yunshu/internal/pkg/eventbus"
	"yunshu/internal/pkg/extension"
	"yunshu/internal/repository"

	"github.com/weibaohui/kom/callbacks"
	kom "github.com/weibaohui/kom/kom"
	"k8s.io/client-go/kubernetes"
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

// NewK8sRuntimeService 创建相关逻辑。
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

	// 使用 kom 原生 RegisterByStringWithID，与 RegisterByPathWithID 等价，避免临时文件在容器/Windows 下的路径问题。
	_, err := kom.Clusters().RegisterByStringWithID(kubeconfig, clusterID)
	if err != nil {
		logx.LogErr("k8s.runtime", "registerClusterIfNeeded", err, "cluster_id", clusterID)
		s.komMu.Lock()
		st.State = "degraded"
		st.LastError = err.Error()
		st.ConsecutiveFailures++
		s.connState[clusterID] = st
		s.komMu.Unlock()
		extension.NotifyKomRegister(clusterID, false, err.Error())
		eventbus.Default().Publish(eventbus.Event{
			Type: eventbus.ClusterKomRegisterFail,
			Payload: map[string]any{"cluster_id": clusterID, "error": err.Error()},
		})
		return apperror.MarkLogged(err)
	}
	s.komMu.Lock()
	s.registeredHash[clusterID] = hash
	st.State = "ready"
	st.LastError = ""
	st.LastSuccessAt = time.Now()
	st.ConsecutiveFailures = 0
	s.connState[clusterID] = st
	s.komMu.Unlock()
	extension.NotifyKomRegister(clusterID, true, "")
	eventbus.Default().Publish(eventbus.Event{
		Type:    eventbus.ClusterKomRegisterOK,
		Payload: map[string]any{"cluster_id": clusterID},
	})
	return nil
}

// DeleteRegisterCache 删除相关的业务逻辑。
func (s *K8sRuntimeService) DeleteRegisterCache(clusterID uint) {
	s.komMu.Lock()
	key := strconv.FormatUint(uint64(clusterID), 10)
	delete(s.registeredHash, key)
	st := s.connState[key]
	st.State = "unknown"
	s.connState[key] = st
	s.komMu.Unlock()
}

// GetClusterKubectl 获取相关的业务逻辑。
func (s *K8sRuntimeService) GetClusterKubectl(ctx context.Context, id uint) (*model.K8sCluster, *kom.Kubectl, error) {
	cluster, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, k8sRepoErr("k8s.runtime", "GetClusterKubectl", err, "cluster_id", id)
	}
	if cluster.Status != 1 {
		return nil, nil, constants.ErrForbiddenWithMsg(constants.ErrMsgb0e556f1ccc5)
	}
	clusterID := strconv.FormatUint(uint64(id), 10)
	kubeconfig, kerr := resolveClusterKubeconfig(cluster)
	if kerr != nil {
		return nil, nil, constants.ErrBadRequestWithMsg(kerr.Error())
	}
	force := kubeconfig != strings.TrimSpace(cluster.Kubeconfig)
	if err := s.registerClusterIfNeeded(clusterID, kubeconfig, force); err != nil {
		return nil, nil, svcerr.Internal("k8s.runtime", "GetClusterKubectl", err, constants.ErrFmtac130d1176b3, "cluster_id", id)
	}
	k := kom.Cluster(clusterID)
	if k == nil {
		return nil, nil, svcerr.InternalMsg("k8s.runtime", "GetClusterKubectl", constants.ErrMsg5248c9e19a3f, "cluster_id", id)
	}
	return cluster, k, nil
}

// serverGitVersionFromKubeconfig 使用 client-go Discovery 拉取 GitVersion（与 kubectl 一致）。
// kom 在进程重启后偶发 Status().ServerVersion() 为空，不能作为心跳唯一依据。
func serverGitVersionFromKubeconfig(kubeconfig string) (string, error) {
	cfg, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		return "", err
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return "", err
	}
	sv, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return "", err
	}
	if sv == nil || strings.TrimSpace(sv.GitVersion) == "" {
		return "", fmt.Errorf("APIServer 返回的版本信息为空")
	}
	return strings.TrimSpace(sv.GitVersion), nil
}

// CheckClusterHeartbeat 执行对应的业务逻辑。
func (s *K8sRuntimeService) CheckClusterHeartbeat(ctx context.Context, id uint) (string, ClusterConnState, error) {
	cluster, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return "", ClusterConnState{}, k8sRepoErr("k8s.runtime", "CheckClusterHeartbeat", err, "cluster_id", id)
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
	kubeconfig, kerr := resolveClusterKubeconfig(cluster)
	if kerr != nil {
		return "", s.GetClusterConnState(id), constants.ErrBadRequestWithMsg(kerr.Error())
	}
	force := kubeconfig != strings.TrimSpace(cluster.Kubeconfig)
	if err := s.registerClusterIfNeeded(clusterID, kubeconfig, force); err != nil {
		return "", s.GetClusterConnState(id), err
	}
	k := kom.Cluster(clusterID)
	if k == nil {
		return "", s.GetClusterConnState(id), constants.ErrInternalWithMsg(constants.ErrMsg5248c9e19a3f)
	}
	gitVer, verr := serverGitVersionFromKubeconfig(kubeconfig)
	if verr != nil || gitVer == "" {
		s.DeleteRegisterCache(id)
		if e := s.registerClusterIfNeeded(clusterID, kubeconfig, true); e != nil {
			return "", s.GetClusterConnState(id), svcerr.Internal("k8s.runtime", "heartbeat_reregister", e, constants.ErrFmt8648d0eaa652)
		}
		if kom.Cluster(clusterID) == nil {
			return "", s.GetClusterConnState(id), constants.ErrInternalWithMsg(constants.ErrMsgb9cf6d1a2c2e)
		}
		gitVer, verr = serverGitVersionFromKubeconfig(kubeconfig)
		if verr != nil || gitVer == "" {
			errMsg := "server version empty"
			if verr != nil {
				errMsg = verr.Error()
			}
			s.komMu.Lock()
			st := s.connState[clusterID]
			st.State = "degraded"
			st.LastAttemptAt = time.Now()
			st.LastError = errMsg
			st.ConsecutiveFailures++
			s.connState[clusterID] = st
			s.komMu.Unlock()
			return "", s.GetClusterConnState(id), svcerr.InternalMsg("k8s.runtime", "CheckClusterHeartbeat", fmt.Sprintf(constants.ErrFmt5d75fe17f8ef, errMsg), "cluster_id", id, "detail", errMsg)
		}
	}
	if probeErr := s.probeClusterListNamespacesKom(ctx, id); probeErr != nil {
		errMsg := probeErr.Error()
		s.komMu.Lock()
		st := s.connState[clusterID]
		st.State = "degraded"
		st.LastAttemptAt = time.Now()
		st.LastError = errMsg
		st.ConsecutiveFailures++
		s.connState[clusterID] = st
		s.komMu.Unlock()
		if _, ok := apperror.IsAppError(probeErr); ok {
			return "", s.GetClusterConnState(id), probeErr
		}
		return "", s.GetClusterConnState(id), svcerr.InternalMsg("k8s.runtime", "CheckClusterHeartbeat", errMsg, "cluster_id", id)
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
	return gitVer, s.GetClusterConnState(id), nil
}

// GetClusterConnState 获取相关的业务逻辑。
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

// GetClusterRestConfig 获取相关的业务逻辑。
func (s *K8sRuntimeService) GetClusterRestConfig(ctx context.Context, id uint) (*model.K8sCluster, *rest.Config, error) {
	cluster, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, nil, k8sRepoErr("k8s.runtime", "GetClusterRestConfig", err, "cluster_id", id)
	}
	if cluster.Status != 1 {
		return nil, nil, constants.ErrForbiddenWithMsg(constants.ErrMsgb0e556f1ccc5)
	}
	kubeconfig, kerr := resolveClusterKubeconfig(cluster)
	if kerr != nil {
		return nil, nil, constants.ErrBadRequestWithMsg(kerr.Error())
	}
	cfg, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		return nil, nil, svcerr.Internal("k8s.runtime", "api", err, constants.ErrFmtd7f0c3fe8497)
	}
	return cluster, cfg, nil
}
