package service

// 通用查询/请求结构体：用于复用完全一致的字段定义与 binding tags。

type ClusterKeywordQuery struct {
	ClusterID uint   `form:"cluster_id" binding:"required"`
	Keyword   string `form:"keyword"`
}

type ClusterNameQuery struct {
	ClusterID uint   `form:"cluster_id" binding:"required"`
	Name      string `form:"name" binding:"required"`
}

type ClusterNamespaceKeywordQuery struct {
	ClusterID uint   `form:"cluster_id" binding:"required"`
	Namespace string `form:"namespace" binding:"required"`
	Keyword   string `form:"keyword"`
}

type ClusterNamespaceOptionalKeywordQuery struct {
	ClusterID uint   `form:"cluster_id" binding:"required"`
	Namespace string `form:"namespace"`
	Keyword   string `form:"keyword"`
}

type ClusterNamespaceNameQuery struct {
	ClusterID uint   `form:"cluster_id" binding:"required"`
	Namespace string `form:"namespace" binding:"required"`
	Name      string `form:"name" binding:"required"`
}

type ClusterManifestApplyRequest struct {
	ClusterID uint   `json:"cluster_id" binding:"required"`
	Manifest  string `json:"manifest" binding:"required"`
}

type ClusterOnlyQuery struct {
	ClusterID uint `form:"cluster_id" binding:"required"`
}

type ClusterNamespaceNameRequest struct {
	ClusterID uint   `json:"cluster_id" binding:"required"`
	Namespace string `json:"namespace" binding:"required"`
	Name      string `json:"name" binding:"required"`
}

type ClusterNamespaceNameScaleRequest struct {
	ClusterID uint   `json:"cluster_id" binding:"required"`
	Namespace string `json:"namespace" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Replicas  int32  `json:"replicas" binding:"required"`
}

type ClusterNamespaceNameSuspendRequest struct {
	ClusterID uint   `json:"cluster_id" binding:"required"`
	Namespace string `json:"namespace" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Suspend   bool   `json:"suspend"`
}

type ClusterNamespaceNameCommandRequest struct {
	ClusterID uint   `json:"cluster_id" binding:"required"`
	Namespace string `json:"namespace" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Command   string `json:"command" binding:"required"`
}

// NodeSchedulabilityRequest 设置节点是否禁止调度（kubectl cordon/uncordon）。
type NodeSchedulabilityRequest struct {
	ClusterID     uint   `json:"cluster_id" binding:"required"`
	Name          string `json:"name" binding:"required"`
	Unschedulable bool   `json:"unschedulable"`
}

// NodeTaintsReplaceRequest 替换节点污点列表（与 kubectl taint 全量替换 spec 行为一致由调用方保证）。
type NodeTaintsReplaceRequest struct {
	ClusterID uint        `json:"cluster_id" binding:"required"`
	Name      string      `json:"name" binding:"required"`
	Taints    []NodeTaint `json:"taints"`
}
