package service

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"yunshu/internal/pkg/apperror"
	"yunshu/internal/pkg/k8sutil"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

type NodeListQuery = ClusterKeywordQuery
type NodeDetailQuery = ClusterNameQuery

type NodeAddressItem struct {
	Type    string `json:"type"`
	Address string `json:"address"`
}

type NodeListItem struct {
	Name          string            `json:"name"`
	Status        string            `json:"status"`
	Unschedulable bool              `json:"unschedulable"`
	Roles         []string          `json:"roles,omitempty"`
	Kernel        string            `json:"kernel"`
	Kubelet       string            `json:"kubelet"`
	OsImage       string            `json:"os_image"`
	Container     string            `json:"container_runtime"`
	Architecture  string            `json:"architecture"`
	Labels        map[string]string `json:"labels,omitempty"`
	Annotations   map[string]string `json:"annotations,omitempty"`
	Taints        []NodeTaint       `json:"taints,omitempty"`
	InternalIP    string            `json:"internal_ip,omitempty"`
	CreationTime  string            `json:"creation_time"`
	Age           string            `json:"age,omitempty"`

	PodCount        int     `json:"pod_count"`
	PodCapacity     int     `json:"pod_capacity,omitempty"`
	PodUsage        string  `json:"pod_usage,omitempty"`
	PodUsagePercent float64 `json:"pod_usage_percent,omitempty"`

	CPUUsage        string  `json:"cpu_usage,omitempty"`
	CPURequests     string  `json:"cpu_requests,omitempty"`
	CPULimits       string  `json:"cpu_limits,omitempty"`
	MemUsage        string  `json:"mem_usage,omitempty"`
	MemRequests     string  `json:"mem_requests,omitempty"`
	MemLimits       string  `json:"mem_limits,omitempty"`
	CPUUsagePercent float64 `json:"cpu_usage_percent,omitempty"`
	MemUsagePercent float64 `json:"mem_usage_percent,omitempty"`
}

type NodeDetail struct {
	Item        NodeListItem      `json:"item"`
	Addresses   []NodeAddressItem `json:"addresses"`
	Conditions  []NodeCondition   `json:"conditions"`
	Taints      []NodeTaint       `json:"taints"`
	Capacity    map[string]string `json:"capacity"`
	Allocatable map[string]string `json:"allocatable"`
	YAML        string            `json:"yaml"`
}

type NodeCondition struct {
	Type               string `json:"type"`
	Status             string `json:"status"`
	Reason             string `json:"reason,omitempty"`
	Message            string `json:"message,omitempty"`
	LastHeartbeatTime  string `json:"last_heartbeat_time,omitempty"`
	LastTransitionTime string `json:"last_transition_time,omitempty"`
}

type NodeTaint struct {
	Key       string `json:"key"`
	Value     string `json:"value,omitempty"`
	Effect    string `json:"effect,omitempty"`
	TimeAdded string `json:"time_added,omitempty"`
}

type K8sNodeService struct {
	runtime *K8sRuntimeService
	dyn     *DynamicResourceService
}

// NewK8sNodeService 创建相关逻辑。
func NewK8sNodeService(runtime *K8sRuntimeService) *K8sNodeService {
	return &K8sNodeService{runtime: runtime, dyn: NewDynamicResourceService(runtime)}
}

func nodeReadyStatus(n corev1.Node) string {
	for _, c := range n.Status.Conditions {
		if c.Type == corev1.NodeReady {
			if c.Status == corev1.ConditionTrue {
				return "Ready"
			}
			return "NotReady"
		}
	}
	return "Unknown"
}

func mapNodeItem(n corev1.Node) NodeListItem {
	internalIP := ""
	for _, a := range n.Status.Addresses {
		if a.Type == corev1.NodeInternalIP {
			internalIP = a.Address
			break
		}
	}
	roles := extractNodeRoles(n.Labels)
	taints := make([]NodeTaint, 0, len(n.Spec.Taints))
	for _, t := range n.Spec.Taints {
		item := NodeTaint{
			Key:    t.Key,
			Value:  t.Value,
			Effect: string(t.Effect),
		}
		if t.TimeAdded != nil && !t.TimeAdded.IsZero() {
			item.TimeAdded = t.TimeAdded.Time.Format("2006-01-02 15:04:05")
		}
		taints = append(taints, item)
	}
	return NodeListItem{
		Name:          n.Name,
		Status:        nodeReadyStatus(n),
		Unschedulable: n.Spec.Unschedulable,
		Roles:         roles,
		Kernel:        n.Status.NodeInfo.KernelVersion,
		Kubelet:       n.Status.NodeInfo.KubeletVersion,
		OsImage:       n.Status.NodeInfo.OSImage,
		Container:     n.Status.NodeInfo.ContainerRuntimeVersion,
		Architecture:  n.Status.NodeInfo.Architecture,
		Labels:        n.Labels,
		Annotations:   n.Annotations,
		Taints:        taints,
		InternalIP:    internalIP,
		CreationTime:  n.CreationTimestamp.Time.Format("2006-01-02 15:04:05"),
		Age:           k8sutil.HumanAge(n.CreationTimestamp.Time),
	}
}

func quantityMapToStringMap(src map[corev1.ResourceName]resource.Quantity) map[string]string {
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[string(k)] = v.String()
	}
	return out
}

// List 查询列表相关的业务逻辑。
func (s *K8sNodeService) List(ctx context.Context, query NodeListQuery) ([]NodeListItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, query.ClusterID)
	if err != nil {
		return nil, err
	}
	var list []corev1.Node
	if err := k.WithContext(ctx).Resource(&corev1.Node{}).List(&list).Error; err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取 Node 列表失败: %v", err))
	}

	// 统计每个 node 上的 pod 数
	podCountByNode := map[string]int{}
	podReqCPUByNode := map[string]resource.Quantity{}
	podReqMemByNode := map[string]resource.Quantity{}
	podLimCPUByNode := map[string]resource.Quantity{}
	podLimMemByNode := map[string]resource.Quantity{}
	{
		var pods []corev1.Pod
		if e := k.WithContext(ctx).Resource(&corev1.Pod{}).List(&pods).Error; e == nil {
			for _, p := range pods {
				nn := strings.TrimSpace(p.Spec.NodeName)
				if nn == "" {
					continue
				}
				podCountByNode[nn]++
				reqCPU := podReqCPUByNode[nn]
				reqMem := podReqMemByNode[nn]
				limCPU := podLimCPUByNode[nn]
				limMem := podLimMemByNode[nn]
				for _, c := range p.Spec.Containers {
					if rq := c.Resources.Requests; rq != nil {
						reqCPU.Add(rq[corev1.ResourceCPU])
						reqMem.Add(rq[corev1.ResourceMemory])
					}
					if lm := c.Resources.Limits; lm != nil {
						limCPU.Add(lm[corev1.ResourceCPU])
						limMem.Add(lm[corev1.ResourceMemory])
					}
				}
				podReqCPUByNode[nn] = reqCPU
				podReqMemByNode[nn] = reqMem
				podLimCPUByNode[nn] = limCPU
				podLimMemByNode[nn] = limMem
			}
		}
	}

	// 尝试读取 metrics.k8s.io 的 NodeMetrics（若未安装 metrics-server，会失败，忽略即可）
	metricsByNode := map[string]nodeMetricSummary{}
	{
		metricsGVK := schema.GroupVersionKind{Group: "metrics.k8s.io", Version: "v1beta1", Kind: "NodeMetrics"}
		items, e := s.dyn.ListByGVK(ctx, k, metricsGVK, "")
		if e == nil {
			for _, u := range items {
				var nm nodeMetrics
				if convErr := runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &nm); convErr != nil {
					continue
				}
				name := strings.TrimSpace(nm.Metadata.Name)
				if name == "" {
					continue
				}
				cpuQ, cpuErr := resource.ParseQuantity(strings.TrimSpace(nm.Usage.CPU))
				memQ, memErr := resource.ParseQuantity(strings.TrimSpace(nm.Usage.Memory))
				if cpuErr != nil && memErr != nil {
					continue
				}
				metricsByNode[name] = nodeMetricSummary{CPU: cpuQ, Mem: memQ}
			}
		}
	}

	kw := strings.ToLower(strings.TrimSpace(query.Keyword))
	out := make([]NodeListItem, 0, len(list))
	for _, n := range list {
		item := mapNodeItem(n)
		item.PodCount = podCountByNode[item.Name]
		item.CPURequests = quantityOrDashNode(podReqCPUByNode[item.Name])
		item.MemRequests = quantityOrDashNode(podReqMemByNode[item.Name])
		item.CPULimits = quantityOrDashNode(podLimCPUByNode[item.Name])
		item.MemLimits = quantityOrDashNode(podLimMemByNode[item.Name])
		if podCap, ok := n.Status.Allocatable[corev1.ResourcePods]; ok {
			item.PodCapacity = int(podCap.Value())
		}
		if item.PodCapacity > 0 {
			item.PodUsage = fmt.Sprintf("%d/%d", item.PodCount, item.PodCapacity)
			item.PodUsagePercent = (float64(item.PodCount) / float64(item.PodCapacity)) * 100
		} else {
			item.PodUsage = fmt.Sprintf("%d/-", item.PodCount)
		}

		if m, ok := metricsByNode[item.Name]; ok {
			item.CPUUsage = quantityOrDashNode(m.CPU)
			item.MemUsage = quantityOrDashNode(m.Mem)
			allocCPU := n.Status.Allocatable[corev1.ResourceCPU]
			allocMem := n.Status.Allocatable[corev1.ResourceMemory]
			item.CPUUsagePercent = quantityPercent(m.CPU, allocCPU)
			item.MemUsagePercent = quantityPercent(m.Mem, allocMem)
		} else {
			item.CPUUsage = "-"
			item.MemUsage = "-"
		}
		if kw != "" {
			hay := strings.ToLower(item.Name + " " + item.InternalIP + " " + item.OsImage + " " + item.Kubelet + " " + strings.Join(item.Roles, ",") + " " + strings.Join(mapKeys(item.Labels), ","))
			if !strings.Contains(hay, kw) {
				continue
			}
		}
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func mapKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Detail 查询详情相关的业务逻辑。
func (s *K8sNodeService) Detail(ctx context.Context, query NodeDetailQuery) (*NodeDetail, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, query.ClusterID)
	if err != nil {
		return nil, err
	}
	var n corev1.Node
	if err := k.WithContext(ctx).Resource(&corev1.Node{}).Name(query.Name).Get(&n).Error; err != nil {
		if apierrors.IsNotFound(err) {
			return nil, apperror.BadRequest("节点不存在")
		}
		return nil, apperror.Internal(fmt.Sprintf("获取 Node 详情失败: %v", err))
	}
	copyObj := n.DeepCopy()
	copyObj.APIVersion = "v1"
	copyObj.Kind = "Node"
	copyObj.ManagedFields = nil
	y, _ := yaml.Marshal(copyObj)
	addrs := make([]NodeAddressItem, 0, len(n.Status.Addresses))
	for _, a := range n.Status.Addresses {
		addrs = append(addrs, NodeAddressItem{Type: string(a.Type), Address: a.Address})
	}
	conditions := make([]NodeCondition, 0, len(n.Status.Conditions))
	for _, c := range n.Status.Conditions {
		item := NodeCondition{
			Type:    string(c.Type),
			Status:  string(c.Status),
			Reason:  c.Reason,
			Message: c.Message,
		}
		if !c.LastHeartbeatTime.IsZero() {
			item.LastHeartbeatTime = c.LastHeartbeatTime.Time.Format("2006-01-02 15:04:05")
		}
		if !c.LastTransitionTime.IsZero() {
			item.LastTransitionTime = c.LastTransitionTime.Time.Format("2006-01-02 15:04:05")
		}
		conditions = append(conditions, item)
	}
	taints := make([]NodeTaint, 0, len(n.Spec.Taints))
	for _, t := range n.Spec.Taints {
		item := NodeTaint{
			Key:    t.Key,
			Value:  t.Value,
			Effect: string(t.Effect),
		}
		if t.TimeAdded != nil && !t.TimeAdded.IsZero() {
			item.TimeAdded = t.TimeAdded.Time.Format("2006-01-02 15:04:05")
		}
		taints = append(taints, item)
	}
	return &NodeDetail{
		Item:        mapNodeItem(n),
		Addresses:   addrs,
		Conditions:  conditions,
		Taints:      taints,
		Capacity:    quantityMapToStringMap(n.Status.Capacity),
		Allocatable: quantityMapToStringMap(n.Status.Allocatable),
		YAML:        string(y),
	}, nil
}

type nodeMetricSummary struct {
	CPU resource.Quantity
	Mem resource.Quantity
}

type nodeMetrics struct {
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Usage struct {
		CPU    string `json:"cpu"`
		Memory string `json:"memory"`
	} `json:"usage"`
}

func extractNodeRoles(labels map[string]string) []string {
	out := make([]string, 0, 2)
	for k := range labels {
		if strings.HasPrefix(k, "node-role.kubernetes.io/") {
			role := strings.TrimPrefix(k, "node-role.kubernetes.io/")
			role = strings.TrimSpace(role)
			if role == "" {
				role = "master"
			}
			out = append(out, role)
		}
	}
	sort.Strings(out)
	if len(out) == 0 {
		return []string{"worker"}
	}
	return out
}

func quantityOrDashNode(q resource.Quantity) string {
	if q.IsZero() {
		return "-"
	}
	return q.String()
}

func quantityPercent(usage, alloc resource.Quantity) float64 {
	if usage.IsZero() || alloc.IsZero() {
		return 0
	}
	u := usage.AsApproximateFloat64()
	a := alloc.AsApproximateFloat64()
	if a <= 0 {
		return 0
	}
	p := (u / a) * 100
	if p < 0 {
		return 0
	}
	if p > 1000 {
		// 极端异常时避免把 UI 撑爆
		return 1000
	}
	return p
}

func normalizeTaintEffect(s string) (corev1.TaintEffect, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", apperror.BadRequest("污点 effect 不能为空（NoSchedule / PreferNoSchedule / NoExecute）")
	}
	lower := strings.ToLower(s)
	switch lower {
	case "noschedule":
		return corev1.TaintEffectNoSchedule, nil
	case "prefernoschedule":
		return corev1.TaintEffectPreferNoSchedule, nil
	case "noexecute":
		return corev1.TaintEffectNoExecute, nil
	}
	switch corev1.TaintEffect(s) {
	case corev1.TaintEffectNoSchedule, corev1.TaintEffectPreferNoSchedule, corev1.TaintEffectNoExecute:
		return corev1.TaintEffect(s), nil
	default:
		return "", apperror.BadRequest("无效的 effect，需为 NoSchedule、PreferNoSchedule 或 NoExecute")
	}
}

func nodeTaintsToCore(in []NodeTaint) ([]corev1.Taint, error) {
	out := make([]corev1.Taint, 0, len(in))
	for _, t := range in {
		key := strings.TrimSpace(t.Key)
		if key == "" {
			return nil, apperror.BadRequest("污点 key 不能为空")
		}
		eff, err := normalizeTaintEffect(t.Effect)
		if err != nil {
			return nil, err
		}
		out = append(out, corev1.Taint{
			Key:    key,
			Value:  strings.TrimSpace(t.Value),
			Effect: eff,
		})
	}
	return out, nil
}

// SetSchedulability 对应 kubectl cordon（禁止调度）/ uncordon（恢复调度）。
func (s *K8sNodeService) SetSchedulability(ctx context.Context, req NodeSchedulabilityRequest) error {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return apperror.BadRequest("节点名称不能为空")
	}
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	var n corev1.Node
	if err := k.WithContext(ctx).Resource(&corev1.Node{}).Name(name).Get(&n).Error; err != nil {
		if apierrors.IsNotFound(err) {
			return apperror.BadRequest("节点不存在")
		}
		return apperror.Internal(fmt.Sprintf("获取 Node 失败: %v", err))
	}
	updated := n.DeepCopy()
	updated.Spec.Unschedulable = req.Unschedulable
	if err := k.WithContext(ctx).Resource(&corev1.Node{}).Update(updated).Error; err != nil {
		return apperror.Internal(fmt.Sprintf("更新 Node 调度状态失败: %v", err))
	}
	return nil
}

// ReplaceTaints 用请求体中的列表替换节点全部污点。
func (s *K8sNodeService) ReplaceTaints(ctx context.Context, req NodeTaintsReplaceRequest) error {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return apperror.BadRequest("节点名称不能为空")
	}
	taints, err := nodeTaintsToCore(req.Taints)
	if err != nil {
		return err
	}
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	var n corev1.Node
	if err := k.WithContext(ctx).Resource(&corev1.Node{}).Name(name).Get(&n).Error; err != nil {
		if apierrors.IsNotFound(err) {
			return apperror.BadRequest("节点不存在")
		}
		return apperror.Internal(fmt.Sprintf("获取 Node 失败: %v", err))
	}
	updated := n.DeepCopy()
	updated.Spec.Taints = taints
	if err := k.WithContext(ctx).Resource(&corev1.Node{}).Update(updated).Error; err != nil {
		return apperror.Internal(fmt.Sprintf("更新 Node 污点失败: %v", err))
	}
	return nil
}
