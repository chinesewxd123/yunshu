package service

type NamespacedListQuery struct {
	ClusterNamespaceKeywordQuery
	LabelQuery string `form:"label_selector"`
}

type NamespacedDetailQuery = ClusterNamespaceNameQuery
type NamespacedApplyRequest = ClusterManifestApplyRequest
type NamespacedDeleteRequest = ClusterNamespaceNameQuery

type WorkloadScaleRequest = ClusterNamespaceNameScaleRequest
type CronJobSuspendRequest = ClusterNamespaceNameSuspendRequest

type CronJobTriggerRequest = ClusterNamespaceNameRequest
type JobRerunRequest = ClusterNamespaceNameRequest

type WorkloadItem struct {
	Name           string            `json:"name"`
	Namespace      string            `json:"namespace"`
	Ready          string            `json:"ready,omitempty"`
	Replicas       string            `json:"replicas,omitempty"`
	Available      string            `json:"available,omitempty"`
	Updated        string            `json:"updated,omitempty"`
	ReadyPercent   int               `json:"ready_percent,omitempty"`
	ResourceText   string            `json:"resource_text,omitempty"`
	ContainersText string            `json:"containers_text,omitempty"`
	ConditionsText string            `json:"conditions_text,omitempty"`
	CPUUsage       string            `json:"cpu_usage,omitempty"`
	MemUsage       string            `json:"mem_usage,omitempty"`
	CPUPctRequest  float64           `json:"cpu_pct_request,omitempty"`
	CPUPctLimit    float64           `json:"cpu_pct_limit,omitempty"`
	MemPctRequest  float64           `json:"mem_pct_request,omitempty"`
	MemPctLimit    float64           `json:"mem_pct_limit,omitempty"`
	Labels         map[string]string `json:"labels,omitempty"`
	Annotations    map[string]string `json:"annotations,omitempty"`
	Active         string            `json:"active,omitempty"`
	Failed         string            `json:"failed,omitempty"`
	StartTime      string            `json:"start_time,omitempty"`
	CompletionTime string            `json:"completion_time,omitempty"`
	Age            string            `json:"age,omitempty"`
	CreationTime   string            `json:"creation_time"`
}

type CronJobItem struct {
	Name               string            `json:"name"`
	Namespace          string            `json:"namespace"`
	Schedule           string            `json:"schedule"`
	Suspend            bool              `json:"suspend"`
	ReadyPercent       int               `json:"ready_percent,omitempty"`
	ResourceText       string            `json:"resource_text,omitempty"`
	ContainersText     string            `json:"containers_text,omitempty"`
	ConditionsText     string            `json:"conditions_text,omitempty"`
	Labels             map[string]string `json:"labels,omitempty"`
	Annotations        map[string]string `json:"annotations,omitempty"`
	LastScheduleTime   string            `json:"last_schedule_time,omitempty"`
	LastSuccessfulTime string            `json:"last_successful_time,omitempty"`
	ActiveCount        string            `json:"active_count,omitempty"`
	CPUUsage           string            `json:"cpu_usage,omitempty"`
	MemUsage           string            `json:"mem_usage,omitempty"`
	CPUPctRequest      float64           `json:"cpu_pct_request,omitempty"`
	CPUPctLimit        float64           `json:"cpu_pct_limit,omitempty"`
	MemPctRequest      float64           `json:"mem_pct_request,omitempty"`
	MemPctLimit        float64           `json:"mem_pct_limit,omitempty"`
	Age                string            `json:"age,omitempty"`
	CreationTime       string            `json:"creation_time"`
}

type WorkloadDetail struct {
	YAML   string `json:"yaml"`
	Object any    `json:"object,omitempty"`
}

type K8sWorkloadService struct {
	runtime *K8sRuntimeService
	dyn     *DynamicResourceService
}

// NewK8sWorkloadService 创建相关逻辑。
type RelatedPodsQuery = ClusterNamespaceNameQuery

type RelatedPodItem struct {
	Name         string `json:"name"`
	Namespace    string `json:"namespace"`
	Phase        string `json:"phase"`
	NodeName     string `json:"node_name"`
	PodIP        string `json:"pod_ip"`
	RestartCount int32  `json:"restart_count"`
	StartTime    string `json:"start_time,omitempty"`
}

// DeploymentPods 执行对应的业务逻辑。

func NewK8sWorkloadService(runtime *K8sRuntimeService) *K8sWorkloadService {
	return &K8sWorkloadService{runtime: runtime, dyn: NewDynamicResourceService(runtime)}
}