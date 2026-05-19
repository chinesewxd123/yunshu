package service

import (
	"context"
	"strconv"

	corev1 "k8s.io/api/core/v1"
)

// ListNamespacesViaKom 通过 kom SDK 列举命名空间；Unauthorized 时强制重注册 kom 后重试一次。
func (s *K8sRuntimeService) ListNamespacesViaKom(ctx context.Context, clusterID uint) ([]corev1.Namespace, error) {
	list, err := s.listNamespacesKomOnce(ctx, clusterID)
	if err == nil {
		return list, nil
	}
	if !isK8sUnauthorizedErr(err) {
		return nil, err
	}

	s.DeleteRegisterCache(clusterID)
	cluster, dbErr := s.repo.GetByID(ctx, clusterID)
	if dbErr != nil {
		return nil, err
	}
	kc, kcErr := resolveClusterKubeconfig(cluster)
	if kcErr != nil {
		return nil, err
	}
	cid := strconv.FormatUint(uint64(clusterID), 10)
	if regErr := s.registerClusterIfNeeded(cid, kc, true); regErr != nil {
		return nil, regErr
	}
	return s.listNamespacesKomOnce(ctx, clusterID)
}

func (s *K8sRuntimeService) listNamespacesKomOnce(ctx context.Context, clusterID uint) ([]corev1.Namespace, error) {
	_, k, err := s.GetClusterKubectl(ctx, clusterID)
	if err != nil {
		return nil, err
	}
	var list []corev1.Namespace
	if listErr := k.WithContext(ctx).Resource(&corev1.Namespace{}).List(&list).Error; listErr != nil {
		return nil, k8sFail(ctx, "k8s.runtime", "ListNamespacesViaKom", listErr, "cluster_id", clusterID)
	}
	return list, nil
}

// probeClusterListNamespacesKom 心跳：用 kom 校验能否列举命名空间（与业务列表同路径）。
func (s *K8sRuntimeService) probeClusterListNamespacesKom(ctx context.Context, clusterID uint) error {
	_, err := s.ListNamespacesViaKom(ctx, clusterID)
	return err
}
