package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/service/svcerr"
	"yunshu/internal/pkg/k8sauth"
	"yunshu/internal/pkg/k8sutil"
	"yunshu/internal/repository"

	kom "github.com/weibaohui/kom/kom"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

type NamespaceListQuery = ClusterKeywordQuery
type NamespaceDetailQuery = ClusterNameQuery

// NamespaceApplyRequest 命名空间 YAML 下发；FailIfExists 为 true 时若资源已存在则拒绝（表单「创建」场景），YAML 页更新标签等不传该字段。
type NamespaceApplyRequest struct {
	ClusterID    uint   `json:"cluster_id" binding:"required"`
	Manifest     string `json:"manifest" binding:"required"`
	FailIfExists bool   `json:"fail_if_exists"`
}

type NamespaceDeleteRequest = ClusterNameQuery

type NamespaceListItem struct {
	Name         string            `json:"name"`
	Status       string            `json:"status"`
	CreationTime string            `json:"creation_time"`
	Labels       map[string]string `json:"labels,omitempty"`
	Annotations  map[string]string `json:"annotations,omitempty"`

	PodCount    int    `json:"pod_count"`
	CPURequests string `json:"cpu_requests,omitempty"`
	CPULimits   string `json:"cpu_limits,omitempty"`
	MemRequests string `json:"mem_requests,omitempty"`
	MemLimits   string `json:"mem_limits,omitempty"`
	CPUUsage    string `json:"cpu_usage,omitempty"`
	MemUsage    string `json:"mem_usage,omitempty"`
	// 列表展示用数值（核 / Gi），与 k8m 风格「Request / Limit / 实时」对齐
	CPUCoresRequest float64 `json:"cpu_cores_request"`
	CPUCoresLimit   float64 `json:"cpu_cores_limit"`
	CPUCoresUsage   float64 `json:"cpu_cores_usage"`
	MemGiRequest    float64 `json:"mem_gi_request"`
	MemGiLimit      float64 `json:"mem_gi_limit"`
	MemGiUsage      float64 `json:"mem_gi_usage"`
	// ResourceQuota 摘要（与「工作负载 request 汇总」不同；未创建 ResourceQuota 时为空）
	ResourceQuotaSummary string `json:"resource_quota_summary,omitempty"`
}

type NamespaceDetail struct {
	Item           NamespaceListItem         `json:"item"`
	Finalizers     []string                  `json:"finalizers,omitempty"`
	ResourceQuotas []NamespaceQuotaItem      `json:"resource_quotas,omitempty"`
	LimitRanges    []NamespaceLimitRangeItem `json:"limit_ranges,omitempty"`
	RecentEvents   []NamespaceEventItem      `json:"recent_events,omitempty"`
	YAML           string                    `json:"yaml"`
}

type NamespaceQuotaItem struct {
	Name  string            `json:"name"`
	Hard  map[string]string `json:"hard,omitempty"`
	Used  map[string]string `json:"used,omitempty"`
	Scope []string          `json:"scope,omitempty"`
}

type NamespaceLimitRangeItem struct {
	Name   string                  `json:"name"`
	Limits []corev1.LimitRangeItem `json:"limits,omitempty"`
}

type NamespaceEventItem struct {
	Type     string `json:"type"`
	Reason   string `json:"reason"`
	Message  string `json:"message"`
	LastTime string `json:"last_time,omitempty"`
	Count    int32  `json:"count"`
}

type K8sNamespaceService struct {
	runtime     *K8sRuntimeService
	dyn         *DynamicResourceService
	nsDenyRepo  *repository.K8sNamespaceDenyRepository
	nsAllowRepo *repository.K8sNamespaceAllowRepository
}

// NewK8sNamespaceService 创建相关逻辑。
func NewK8sNamespaceService(
	runtime *K8sRuntimeService,
	nsDeny *repository.K8sNamespaceDenyRepository,
	nsAllow *repository.K8sNamespaceAllowRepository,
) *K8sNamespaceService {
	return &K8sNamespaceService{
		runtime:     runtime,
		dyn:         NewDynamicResourceService(runtime),
		nsDenyRepo:  nsDeny,
		nsAllowRepo: nsAllow,
	}
}

var namespaceGVK = schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Namespace"}

// List 查询列表相关的业务逻辑。
func (s *K8sNamespaceService) List(ctx context.Context, query NamespaceListQuery, pack *k8sauth.PrincipalPack) ([]NamespaceListItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, query.ClusterID)
	if err != nil {
		return nil, err
	}

	podSummary := map[string]namespacePodSummary{}
	{
		var pods []corev1.Pod
		// 全量拉取一次，按 namespace 聚合（比对每个 namespace 分别 List 更省请求数）
		if e := k.WithContext(ctx).Resource(&corev1.Pod{}).List(&pods).Error; e == nil {
			for _, p := range pods {
				ns := strings.TrimSpace(p.Namespace)
				if ns == "" {
					continue
				}
				sum := podSummary[ns]
				sum.PodCount++
				rCPU, rMem, lCPU, lMem := podSpecResourceTotals(p.Spec)
				sum.CPURequests.Add(rCPU)
				sum.MemRequests.Add(rMem)
				sum.CPULimits.Add(lCPU)
				sum.MemLimits.Add(lMem)
				podSummary[ns] = sum
			}
		}
	}

	nsUsage := aggregatePodMetricsUsageByNamespace(ctx, k)

	rqByNS := map[string][]corev1.ResourceQuota{}
	{
		var rqs []corev1.ResourceQuota
		if e := k.WithContext(ctx).Resource(&corev1.ResourceQuota{}).AllNamespace().List(&rqs).Error; e == nil {
			for i := range rqs {
				q := strings.TrimSpace(rqs[i].Namespace)
				if q == "" {
					continue
				}
				rqByNS[q] = append(rqByNS[q], rqs[i])
			}
		}
	}

	list, err := s.runtime.ListNamespacesViaKom(ctx, query.ClusterID)
	if err != nil {
		return nil, err
	}
	kw := strings.ToLower(strings.TrimSpace(query.Keyword))
	out := make([]NamespaceListItem, 0, len(list))
	for _, ns := range list {
		if kw != "" && !strings.Contains(strings.ToLower(ns.Name), kw) {
			continue
		}
		sum := podSummary[ns.Name]
		u := nsUsage[ns.Name]
		cpuUse := "-"
		memUse := "-"
		if !u.CPU.IsZero() || !u.Mem.IsZero() {
			cpuUse = formatQuantityCPUReadable(u.CPU)
			memUse = formatQuantityMemReadable(u.Mem)
		}
		quotaSum := summarizeResourceQuotasForList(rqByNS[ns.Name])
		out = append(out, NamespaceListItem{
			Name:         ns.Name,
			Status:       string(ns.Status.Phase),
			CreationTime: ns.CreationTimestamp.Time.Format("2006-01-02 15:04:05"),
			Labels:       ns.Labels,
			Annotations:  ns.Annotations,
			PodCount:     sum.PodCount,
			CPURequests:  quantityOrDash(sum.CPURequests),
			CPULimits:    quantityOrDash(sum.CPULimits),
			MemRequests:  quantityOrDash(sum.MemRequests),
			MemLimits:    quantityOrDash(sum.MemLimits),
			CPUUsage:     cpuUse,
			MemUsage:     memUse,
			CPUCoresRequest: quantityToCoresApprox(sum.CPURequests),
			CPUCoresLimit:   quantityToCoresApprox(sum.CPULimits),
			CPUCoresUsage:   quantityToCoresApprox(u.CPU),
			MemGiRequest:    quantityToGiApprox(sum.MemRequests),
			MemGiLimit:      quantityToGiApprox(sum.MemLimits),
			MemGiUsage:      quantityToGiApprox(u.Mem),
			ResourceQuotaSummary: quotaSum,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })

	if pack != nil && len(pack.PrincipalRows()) > 0 && query.ClusterID > 0 {
		nsNames := make([]string, len(out))
		for i := range out {
			nsNames[i] = out[i].Name
		}
		var ferr error
		nsNames, ferr = FilterNamespaceNamesByPolicy(ctx, s.nsDenyRepo, s.nsAllowRepo, *pack, query.ClusterID, nsNames)
		if ferr != nil {
			return nil, ferr
		}
		keep := make(map[string]struct{}, len(nsNames))
		for _, n := range nsNames {
			keep[n] = struct{}{}
		}
		filtered := out[:0]
		for _, it := range out {
			if _, ok := keep[it.Name]; ok {
				filtered = append(filtered, it)
			}
		}
		out = filtered
	}

	return out, nil
}

type namespacePodSummary struct {
	PodCount    int
	CPURequests resource.Quantity
	CPULimits   resource.Quantity
	MemRequests resource.Quantity
	MemLimits   resource.Quantity
}

func quantityToCoresApprox(q resource.Quantity) float64 {
	if q.IsZero() {
		return 0
	}
	return q.AsApproximateFloat64()
}

func quantityToGiApprox(q resource.Quantity) float64 {
	if q.IsZero() {
		return 0
	}
	return q.AsApproximateFloat64() / (1024 * 1024 * 1024)
}

func quantityOrDash(q resource.Quantity) string {
	if q.IsZero() {
		return "-"
	}
	return q.String()
}

// formatQuantityCPUReadable 列表/摘要展示用（避免巨量毫核整数）。
func formatQuantityCPUReadable(q resource.Quantity) string {
	if q.IsZero() {
		return "0"
	}
	m := q.MilliValue()
	if m < 1000 || m%1000 != 0 {
		return fmt.Sprintf("%dm", m)
	}
	c := float64(m) / 1000.0
	if c == math.Trunc(c) {
		return fmt.Sprintf("%.0f", c)
	}
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.3f", c), "0"), ".")
}

// formatQuantityMemReadable 将内存用量格式化为 Ki/Mi/Gi。
func formatQuantityMemReadable(q resource.Quantity) string {
	if q.IsZero() {
		return "0"
	}
	b := q.Value()
	if b <= 0 {
		return q.String()
	}
	const (
		Ki int64 = 1024
		Mi int64 = Ki * 1024
		Gi int64 = Mi * 1024
	)
	switch {
	case b >= Gi:
		return fmt.Sprintf("%.2fGi", float64(b)/float64(Gi))
	case b >= Mi:
		return fmt.Sprintf("%.2fMi", float64(b)/float64(Mi))
	case b >= Ki:
		return fmt.Sprintf("%.2fKi", float64(b)/float64(Ki))
	default:
		return fmt.Sprintf("%dB", b)
	}
}

func quotaPickCPUMem(hard, used corev1.ResourceList, wantCPU bool) (hardQ, usedQ resource.Quantity) {
	cpuNames := []corev1.ResourceName{corev1.ResourceRequestsCPU, corev1.ResourceLimitsCPU, corev1.ResourceCPU}
	memNames := []corev1.ResourceName{corev1.ResourceRequestsMemory, corev1.ResourceLimitsMemory, corev1.ResourceMemory}
	names := memNames
	if wantCPU {
		names = cpuNames
	}
	for _, n := range names {
		if v, ok := hard[n]; ok {
			hardQ = v
			break
		}
	}
	for _, n := range names {
		if v, ok := used[n]; ok {
			usedQ = v
			break
		}
	}
	return hardQ, usedQ
}

func formatQuotaUsedHardLine(used, hard resource.Quantity, isCPU bool) string {
	uStr, hStr := "-", "-"
	if !used.IsZero() {
		if isCPU {
			uStr = formatQuantityCPUReadable(used)
		} else {
			uStr = formatQuantityMemReadable(used)
		}
	}
	if !hard.IsZero() {
		if isCPU {
			hStr = formatQuantityCPUReadable(hard)
		} else {
			hStr = formatQuantityMemReadable(hard)
		}
	}
	if uStr == "-" && hStr == "-" {
		return "- / -"
	}
	return fmt.Sprintf("%s / %s", uStr, hStr)
}

// summarizeResourceQuotasForList 列表「配额」列：取字典序第一个 ResourceQuota 的 used/hard（多配额时提示见详情）。
func summarizeResourceQuotasForList(rqs []corev1.ResourceQuota) string {
	if len(rqs) == 0 {
		return ""
	}
	rr := append([]corev1.ResourceQuota(nil), rqs...)
	sort.Slice(rr, func(i, j int) bool { return rr[i].Name < rr[j].Name })
	q := rr[0]
	hard := q.Status.Hard
	if len(hard) == 0 {
		hard = q.Spec.Hard
	}
	used := q.Status.Used
	cpuH, cpuU := quotaPickCPUMem(hard, used, true)
	memH, memU := quotaPickCPUMem(hard, used, false)
	var b strings.Builder
	fmt.Fprintf(&b, "CPU: %s\n", formatQuotaUsedHardLine(cpuU, cpuH, true))
	fmt.Fprintf(&b, "MEM: %s", formatQuotaUsedHardLine(memU, memH, false))
	if len(rr) > 1 {
		fmt.Fprintf(&b, "\n（另有 %d 个 ResourceQuota，见详情）", len(rr)-1)
	}
	return b.String()
}

// Detail 查询详情相关的业务逻辑。
func (s *K8sNamespaceService) Detail(ctx context.Context, query NamespaceDetailQuery) (*NamespaceDetail, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, query.ClusterID)
	if err != nil {
		return nil, err
	}
	u, err := s.dyn.GetByGVK(ctx, k, namespaceGVK, "", query.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, constants.ErrBadRequestWithMsg(constants.ErrMsg52d9e6e7f573)
		}
		return nil, svcerr.Internal(ctx, "k8s.namespace", "api", err, constants.ErrFmt059d07c698fe)
	}
	var ns corev1.Namespace
	_ = runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &ns)
	copyObj := ns.DeepCopy()
	copyObj.APIVersion = "v1"
	copyObj.Kind = "Namespace"
	copyObj.ManagedFields = nil
	y, _ := yaml.Marshal(copyObj)
	quotaItems, _ := s.listNamespaceQuotas(ctx, k, query.Name)
	limitItems, _ := s.listNamespaceLimitRanges(ctx, k, query.Name)
	eventItems, _ := s.listNamespaceEvents(ctx, k, query.Name)
	finalizers := make([]string, 0, len(copyObj.Spec.Finalizers))
	for _, f := range copyObj.Spec.Finalizers {
		finalizers = append(finalizers, string(f))
	}
	return &NamespaceDetail{
		Item: NamespaceListItem{
			Name:         copyObj.Name,
			Status:       string(copyObj.Status.Phase),
			CreationTime: copyObj.CreationTimestamp.Time.Format("2006-01-02 15:04:05"),
			Labels:       copyObj.Labels,
			Annotations:  copyObj.Annotations,
		},
		Finalizers:     finalizers,
		ResourceQuotas: quotaItems,
		LimitRanges:    limitItems,
		RecentEvents:   eventItems,
		YAML:           string(y),
	}, nil
}

func mapQuantityToString(v corev1.ResourceList) map[string]string {
	out := make(map[string]string, len(v))
	for k, q := range v {
		out[string(k)] = q.String()
	}
	return out
}

func (s *K8sNamespaceService) listNamespaceQuotas(ctx context.Context, k *kom.Kubectl, namespace string) ([]NamespaceQuotaItem, error) {
	var list []corev1.ResourceQuota
	if err := k.WithContext(ctx).Resource(&corev1.ResourceQuota{}).Namespace(namespace).List(&list).Error; err != nil {
		return nil, err
	}
	out := make([]NamespaceQuotaItem, 0, len(list))
	for _, q := range list {
		scope := make([]string, 0, len(q.Spec.Scopes))
		for _, s := range q.Spec.Scopes {
			scope = append(scope, string(s))
		}
		hardList := q.Status.Hard
		if len(hardList) == 0 {
			hardList = q.Spec.Hard
		}
		out = append(out, NamespaceQuotaItem{
			Name:  q.Name,
			Hard:  mapQuantityToString(hardList),
			Used:  mapQuantityToString(q.Status.Used),
			Scope: scope,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *K8sNamespaceService) listNamespaceLimitRanges(ctx context.Context, k *kom.Kubectl, namespace string) ([]NamespaceLimitRangeItem, error) {
	var list []corev1.LimitRange
	if err := k.WithContext(ctx).Resource(&corev1.LimitRange{}).Namespace(namespace).List(&list).Error; err != nil {
		return nil, err
	}
	out := make([]NamespaceLimitRangeItem, 0, len(list))
	for _, r := range list {
		out = append(out, NamespaceLimitRangeItem{
			Name:   r.Name,
			Limits: r.Spec.Limits,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func (s *K8sNamespaceService) listNamespaceEvents(ctx context.Context, k *kom.Kubectl, namespace string) ([]NamespaceEventItem, error) {
	var list []corev1.Event
	if err := k.WithContext(ctx).Resource(&corev1.Event{}).Namespace(namespace).Limit(20).List(&list).Error; err != nil {
		return nil, err
	}
	out := make([]NamespaceEventItem, 0, len(list))
	for _, e := range list {
		lastTime := ""
		if !e.LastTimestamp.IsZero() {
			lastTime = e.LastTimestamp.Time.Format("2006-01-02 15:04:05")
		} else if !e.CreationTimestamp.IsZero() {
			lastTime = e.CreationTimestamp.Time.Format("2006-01-02 15:04:05")
		}
		out = append(out, NamespaceEventItem{
			Type:     e.Type,
			Reason:   e.Reason,
			Message:  e.Message,
			LastTime: lastTime,
			Count:    e.Count,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LastTime > out[j].LastTime })
	if len(out) > 10 {
		out = out[:10]
	}
	return out, nil
}

// Apply 提交申请相关的业务逻辑。
func (s *K8sNamespaceService) Apply(ctx context.Context, req NamespaceApplyRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(req.Manifest) == "" {
		return constants.ErrBadRequestWithMsg(constants.ErrMsg01433598170d)
	}
	refs := extractNamespaceRefs(req.Manifest)
	if req.FailIfExists && len(refs) > 0 {
		for _, name := range refs {
			_, ge := s.dyn.GetByGVK(ctx, k, namespaceGVK, "", name)
			if ge == nil {
				return constants.ErrK8sNamespaceAlreadyExistsMsg(name)
			}
			if !apierrors.IsNotFound(ge) {
				return constants.ErrInternalWithMsg(fmt.Sprintf(constants.ErrFmt6d3ec85d0a18, ge))
			}
		}
	}
	err = s.dyn.ApplyManifest(ctx, k, req.Manifest, func(c context.Context) bool {
		if len(refs) == 0 {
			return false
		}
		for _, name := range refs {
			if _, e := s.dyn.GetByGVK(c, k, namespaceGVK, "", name); e != nil {
				return false
			}
		}
		return true
	})
	if err != nil {
		return k8sFail(ctx, "k8s.namespace", "api", err)
	}
	return nil
}

func extractNamespaceRefs(manifest string) []string {
	docs := k8sutil.SplitYAMLDocs(manifest)
	out := make([]string, 0)
	for _, doc := range docs {
		docTrim := strings.TrimSpace(doc)
		if docTrim == "" {
			continue
		}
		var m map[string]any
		if err := yaml.Unmarshal([]byte(docTrim), &m); err != nil {
			continue
		}
		kind, _ := m["kind"].(string)
		if strings.TrimSpace(kind) != "Namespace" {
			continue
		}
		meta, _ := m["metadata"].(map[string]any)
		if meta == nil {
			continue
		}
		name, _ := meta["name"].(string)
		name = strings.TrimSpace(name)
		if name != "" {
			out = append(out, name)
		}
	}
	return out
}

// Delete 删除相关的业务逻辑。
func (s *K8sNamespaceService) Delete(ctx context.Context, req NamespaceDeleteRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if err := s.dyn.DeleteByGVK(ctx, k, namespaceGVK, "", req.Name); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return k8sFail(ctx, "k8s.namespace", "api", err)
	}
	return nil
}
