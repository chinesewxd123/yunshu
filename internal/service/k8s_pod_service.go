package service

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strings"
	"time"

	"go-permission-system/internal/pkg/apperror"

	kom "github.com/weibaohui/kom/kom"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

var (
	// RFC 1123 label: lower case alphanumeric or '-', start/end alphanumeric.
	rfc1123LabelRe = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	// RFC 1123 subdomain (simplified): labels separated by '.', each a label.
	rfc1123SubdomainRe = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`)
)

func validateRFC1123Label(name string) error {
	n := strings.TrimSpace(name)
	if n == "" || !rfc1123LabelRe.MatchString(n) {
		return apperror.BadRequest("容器名称不合法：必须为 RFC1123 label（小写字母/数字/短横线，且首尾为字母或数字）")
	}
	return nil
}

func validateRFC1123Subdomain(name string) error {
	n := strings.TrimSpace(name)
	if n == "" || !rfc1123SubdomainRe.MatchString(n) {
		return apperror.BadRequest("Pod 名称不合法：必须为 RFC1123 subdomain（小写字母/数字/短横线/点，且首尾为字母或数字）")
	}
	return nil
}

type PodListQuery struct {
	ClusterID uint   `form:"cluster_id" binding:"required"`
	Namespace string `form:"namespace"`
	Keyword   string `form:"keyword"`
}

type PodLogsQuery struct {
	ClusterID uint   `form:"cluster_id" binding:"required"`
	Namespace string `form:"namespace" binding:"required"`
	Name      string `form:"name" binding:"required"`
	Container string `form:"container"`
	TailLines int64  `form:"tail_lines"`
	Follow    bool   `form:"follow"`
	Keyword   string `form:"keyword"`
	StartTime string `form:"start_time"`
	EndTime   string `form:"end_time"`
}

type PodExecRequest struct {
	ClusterID uint   `json:"cluster_id" binding:"required"`
	Namespace string `json:"namespace" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Container string `json:"container"`
	Command   string `json:"command" binding:"required"`
}

type PodDeleteRequest struct {
	ClusterID uint   `json:"cluster_id" binding:"required"`
	Namespace string `json:"namespace" binding:"required"`
	Name      string `json:"name" binding:"required"`
}

type PodDetailQuery struct {
	ClusterID uint   `form:"cluster_id" binding:"required"`
	Namespace string `form:"namespace" binding:"required"`
	Name      string `form:"name" binding:"required"`
}

type PodEventQuery struct {
	ClusterID uint   `form:"cluster_id" binding:"required"`
	Namespace string `form:"namespace" binding:"required"`
	Name      string `form:"name" binding:"required"`
}

type PodRestartRequest struct {
	ClusterID uint   `json:"cluster_id" binding:"required"`
	Namespace string `json:"namespace" binding:"required"`
	Name      string `json:"name" binding:"required"`
}

type PodCreateYAMLRequest struct {
	ClusterID uint   `json:"cluster_id" binding:"required"`
	Namespace string `json:"namespace" binding:"required"`
	Manifest  string `json:"manifest" binding:"required"`
}

type PodCreateSimpleRequest struct {
	ClusterID uint   `json:"cluster_id" binding:"required"`
	Namespace string `json:"namespace" binding:"required"`
	Name      string `json:"name" binding:"required"`
	Image     string `json:"image" binding:"required"`
	Command   string `json:"command"`

	ContainerName   string            `json:"container_name"`
	ImagePullPolicy string            `json:"image_pull_policy"`
	RestartPolicy   string            `json:"restart_policy"`
	Port            int32             `json:"port"`
	Env             map[string]string `json:"env"`
	Labels          map[string]string `json:"labels"`

	RequestsCPU    string `json:"requests_cpu"`
	RequestsMemory string `json:"requests_memory"`
	LimitsCPU      string `json:"limits_cpu"`
	LimitsMemory   string `json:"limits_memory"`

	Tolerations       []PodCreateSimpleToleration `json:"tolerations"`
	NodeSelector      map[string]string           `json:"node_selector"`
	PriorityClassName string                      `json:"priority_class_name"`
	Affinity          *corev1.Affinity            `json:"affinity"`
}

type PodCreateSimpleToleration struct {
	Key               string `json:"key"`
	Operator          string `json:"operator"`
	Value             string `json:"value"`
	Effect            string `json:"effect"`
	TolerationSeconds *int64 `json:"toleration_seconds"`
}

type PodItem struct {
	Name         string    `json:"name"`
	Namespace    string    `json:"namespace"`
	Phase        string    `json:"phase"`
	NodeName     string    `json:"node_name"`
	Ready        bool      `json:"ready"`
	PodIP        string    `json:"pod_ip"`
	HostIP       string    `json:"host_ip"`
	QOSClass     string    `json:"qos_class"`
	RestartCount int32     `json:"restart_count"`
	Images       []string  `json:"images"`
	StartTime    time.Time `json:"start_time"`
}

type PodContainerInfo struct {
	Name         string `json:"name"`
	Image        string `json:"image"`
	Ready        bool   `json:"ready"`
	RestartCount int32  `json:"restart_count"`
	State        string `json:"state"`
}

type PodDetail struct {
	Name              string                `json:"name"`
	Namespace         string                `json:"namespace"`
	UID               string                `json:"uid"`
	Phase             string                `json:"phase"`
	NodeName          string                `json:"node_name"`
	ServiceAccount    string                `json:"service_account"`
	PodIP             string                `json:"pod_ip"`
	HostIP            string                `json:"host_ip"`
	QOSClass          string                `json:"qos_class"`
	Labels            map[string]string     `json:"labels"`
	Annotations       map[string]string     `json:"annotations"`
	Containers        []PodContainerInfo    `json:"containers"`
	InitContainers    []PodContainerInfo    `json:"init_containers"`
	Conditions        []corev1.PodCondition `json:"conditions"`
	Volumes           []corev1.Volume       `json:"volumes"`
	Tolerations       []corev1.Toleration   `json:"tolerations"`
	NodeSelector      map[string]string     `json:"node_selector"`
	PriorityClassName string                `json:"priority_class_name"`
	Affinity          *corev1.Affinity      `json:"affinity"`
	StartTime         time.Time             `json:"start_time"`
	CreationTime      time.Time             `json:"creation_time"`
}

type PodEventItem struct {
	Type           string    `json:"type"`
	Reason         string    `json:"reason"`
	Message        string    `json:"message"`
	Count          int32     `json:"count"`
	FirstTimestamp time.Time `json:"first_timestamp"`
	LastTimestamp  time.Time `json:"last_timestamp"`
}

type K8sPodService struct {
	runtime *K8sRuntimeService
}

func NewK8sPodService(runtime *K8sRuntimeService) *K8sPodService {
	return &K8sPodService{runtime: runtime}
}

func (s *K8sPodService) List(ctx context.Context, query PodListQuery) ([]PodItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, query.ClusterID)
	if err != nil {
		return nil, err
	}
	ns := strings.TrimSpace(query.Namespace)
	if ns == "" {
		ns = "default"
	}
	var pods []corev1.Pod
	if err := k.Resource(&corev1.Pod{}).Namespace(ns).List(&pods).Error; err != nil {
		return nil, err
	}
	kw := strings.ToLower(strings.TrimSpace(query.Keyword))
	out := make([]PodItem, 0, len(pods))
	for _, p := range pods {
		item := mapPodItem(p)
		if kw != "" {
			if !strings.Contains(strings.ToLower(item.Name), kw) && !strings.Contains(strings.ToLower(item.NodeName), kw) {
				continue
			}
		}
		out = append(out, item)
	}
	return out, nil
}

func mapPodItem(p corev1.Pod) PodItem {
	var restart int32
	images := make([]string, 0, len(p.Spec.Containers))
	for _, c := range p.Spec.Containers {
		images = append(images, c.Image)
	}
	readyCnt := 0
	for _, st := range p.Status.ContainerStatuses {
		restart += st.RestartCount
		if st.Ready {
			readyCnt++
		}
	}
	ready := len(p.Status.ContainerStatuses) > 0 && readyCnt == len(p.Status.ContainerStatuses)
	startTime := time.Time{}
	if p.Status.StartTime != nil {
		startTime = p.Status.StartTime.Time
	}
	return PodItem{
		Name:         p.Name,
		Namespace:    p.Namespace,
		Phase:        string(p.Status.Phase),
		NodeName:     p.Spec.NodeName,
		Ready:        ready,
		PodIP:        p.Status.PodIP,
		HostIP:       p.Status.HostIP,
		QOSClass:     string(p.Status.QOSClass),
		RestartCount: restart,
		Images:       images,
		StartTime:    startTime,
	}
}

func (s *K8sPodService) Detail(ctx context.Context, query PodDetailQuery) (*PodDetail, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, query.ClusterID)
	if err != nil {
		return nil, err
	}
	pod, err := k.Client().CoreV1().Pods(query.Namespace).Get(ctx, query.Name, metav1.GetOptions{})
	if err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取 Pod 详情失败: %v", err))
	}
	containers := make([]PodContainerInfo, 0, len(pod.Status.ContainerStatuses))
	for _, c := range pod.Status.ContainerStatuses {
		containers = append(containers, PodContainerInfo{
			Name:         c.Name,
			Image:        c.Image,
			Ready:        c.Ready,
			RestartCount: c.RestartCount,
			State:        containerState(c.State),
		})
	}
	initContainers := make([]PodContainerInfo, 0, len(pod.Status.InitContainerStatuses))
	for _, c := range pod.Status.InitContainerStatuses {
		initContainers = append(initContainers, PodContainerInfo{
			Name:         c.Name,
			Image:        c.Image,
			Ready:        c.Ready,
			RestartCount: c.RestartCount,
			State:        containerState(c.State),
		})
	}
	startTime := time.Time{}
	if pod.Status.StartTime != nil {
		startTime = pod.Status.StartTime.Time
	}
	return &PodDetail{
		Name:              pod.Name,
		Namespace:         pod.Namespace,
		UID:               string(pod.UID),
		Phase:             string(pod.Status.Phase),
		NodeName:          pod.Spec.NodeName,
		ServiceAccount:    pod.Spec.ServiceAccountName,
		PodIP:             pod.Status.PodIP,
		HostIP:            pod.Status.HostIP,
		QOSClass:          string(pod.Status.QOSClass),
		Labels:            pod.Labels,
		Annotations:       pod.Annotations,
		Containers:        containers,
		InitContainers:    initContainers,
		Conditions:        pod.Status.Conditions,
		Volumes:           pod.Spec.Volumes,
		Tolerations:       pod.Spec.Tolerations,
		NodeSelector:      pod.Spec.NodeSelector,
		PriorityClassName: pod.Spec.PriorityClassName,
		Affinity:          pod.Spec.Affinity,
		StartTime:         startTime,
		CreationTime:      pod.CreationTimestamp.Time,
	}, nil
}

func containerState(st corev1.ContainerState) string {
	switch {
	case st.Running != nil:
		return "Running"
	case st.Waiting != nil:
		return "Waiting"
	case st.Terminated != nil:
		return "Terminated"
	default:
		return "Unknown"
	}
}

func (s *K8sPodService) Events(ctx context.Context, query PodEventQuery) ([]PodEventItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, query.ClusterID)
	if err != nil {
		return nil, err
	}
	list, err := k.Client().CoreV1().Events(query.Namespace).List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=Pod", query.Name),
	})
	if err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取 Pod 事件失败: %v", err))
	}
	out := make([]PodEventItem, 0, len(list.Items))
	for _, e := range list.Items {
		out = append(out, PodEventItem{
			Type:           e.Type,
			Reason:         e.Reason,
			Message:        e.Message,
			Count:          e.Count,
			FirstTimestamp: e.FirstTimestamp.Time,
			LastTimestamp:  e.LastTimestamp.Time,
		})
	}
	return out, nil
}

func (s *K8sPodService) Delete(ctx context.Context, req PodDeleteRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if err := k.Client().CoreV1().Pods(req.Namespace).Delete(ctx, req.Name, metav1.DeleteOptions{}); err != nil {
		return apperror.Internal(fmt.Sprintf("删除 Pod 失败: %v", err))
	}
	return nil
}

func (s *K8sPodService) Exec(ctx context.Context, req PodExecRequest) (string, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return "", err
	}
	container := strings.TrimSpace(req.Container)
	if container == "" {
		pod, e := k.Client().CoreV1().Pods(req.Namespace).Get(ctx, req.Name, metav1.GetOptions{})
		if e == nil && len(pod.Spec.Containers) > 0 {
			container = pod.Spec.Containers[0].Name
		}
	}
	cmd := strings.Fields(strings.TrimSpace(req.Command))
	if len(cmd) == 0 {
		return "", apperror.BadRequest("command 不能为空")
	}

	var out []byte
	err = k.Namespace(req.Namespace).Name(req.Name).Ctl().Pod().ContainerName(container).Command(cmd[0], cmd[1:]...).Execute(&out).Error
	if err != nil {
		return "", apperror.Internal(fmt.Sprintf("Pod Exec 失败: %v", err))
	}
	return string(out), nil
}

type ExecTerminalSize struct {
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

// ExecTTYStream opens an interactive TTY exec stream to the container.
func (s *K8sPodService) ExecTTYStream(
	ctx context.Context,
	clusterID uint,
	namespace string,
	podName string,
	container string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer,
	sizeQueue remotecommand.TerminalSizeQueue,
) error {
	_, restCfg, err := s.runtime.GetClusterRestConfig(ctx, clusterID)
	if err != nil {
		return err
	}
	_, k, err := s.runtime.GetClusterKubectl(ctx, clusterID)
	if err != nil {
		return err
	}

	if strings.TrimSpace(namespace) == "" || strings.TrimSpace(podName) == "" {
		return apperror.BadRequest("namespace/name 不能为空")
	}

	req := k.Client().CoreV1().RESTClient().
		Post().
		Resource("pods").
		Name(strings.TrimSpace(podName)).
		Namespace(strings.TrimSpace(namespace)).
		SubResource("exec")

	cmd := []string{"sh", "-l"}
	execOpts := &corev1.PodExecOptions{
		Container: strings.TrimSpace(container),
		Command:   cmd,
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}

	req.VersionedParams(execOpts, scheme.ParameterCodec)
	exec, err := remotecommand.NewSPDYExecutor(restCfg, "POST", req.URL())
	if err != nil {
		return apperror.Internal(fmt.Sprintf("创建 exec 通道失败: %v", err))
	}

	return exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:             stdin,
		Stdout:            stdout,
		Stderr:            stderr,
		Tty:               true,
		TerminalSizeQueue: sizeQueue,
	})
}

func (s *K8sPodService) GetLogs(ctx context.Context, query PodLogsQuery) (string, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, query.ClusterID)
	if err != nil {
		return "", err
	}
	tail := query.TailLines
	if tail <= 0 {
		tail = 300
	}
	opts := &corev1.PodLogOptions{
		Container: strings.TrimSpace(query.Container),
		TailLines: &tail,
	}
	stream, err := k.Client().CoreV1().Pods(query.Namespace).GetLogs(query.Name, opts).Stream(ctx)
	if err != nil {
		return "", apperror.Internal(fmt.Sprintf("获取 Pod 日志失败: %v", err))
	}
	defer stream.Close()
	var buf bytes.Buffer
	if _, err = io.Copy(&buf, stream); err != nil {
		return "", apperror.Internal(fmt.Sprintf("读取 Pod 日志失败: %v", err))
	}
	return filterLogLines(buf.String(), query.Keyword, query.StartTime, query.EndTime), nil
}

func (s *K8sPodService) StreamLogs(ctx context.Context, query PodLogsQuery, onLine func(string) error) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, query.ClusterID)
	if err != nil {
		return err
	}
	tail := query.TailLines
	if tail <= 0 {
		tail = 100
	}
	opts := &corev1.PodLogOptions{
		Container: strings.TrimSpace(query.Container),
		TailLines: &tail,
		Follow:    true,
	}
	stream, err := k.Client().CoreV1().Pods(query.Namespace).GetLogs(query.Name, opts).Stream(ctx)
	if err != nil {
		return apperror.Internal(fmt.Sprintf("获取 Pod 流日志失败: %v", err))
	}
	defer stream.Close()

	reader := bufio.NewReader(stream)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			line, e := reader.ReadString('\n')
			if line != "" {
				if cbErr := onLine(line); cbErr != nil {
					return cbErr
				}
			}
			if e != nil {
				if e == io.EOF {
					time.Sleep(200 * time.Millisecond)
					continue
				}
				return e
			}
		}
	}
}

func (s *K8sPodService) Restart(ctx context.Context, req PodRestartRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	grace := int64(0)
	if err := k.Client().CoreV1().Pods(req.Namespace).Delete(ctx, req.Name, metav1.DeleteOptions{GracePeriodSeconds: &grace}); err != nil {
		return apperror.Internal(fmt.Sprintf("重启 Pod 失败: %v", err))
	}
	return nil
}

func (s *K8sPodService) CreateByYAML(ctx context.Context, req PodCreateYAMLRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	_ = k.WithContext(ctx).Applier().Apply(req.Manifest)
	return nil
}

func (s *K8sPodService) CreateSimple(ctx context.Context, req PodCreateSimpleRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if err := validateRFC1123Subdomain(req.Name); err != nil {
		return err
	}
	if cn := strings.TrimSpace(req.ContainerName); cn != "" {
		if err := validateRFC1123Label(cn); err != nil {
			return err
		}
	} else {
		// default container name equals pod name; validate as label too
		if err := validateRFC1123Label(req.Name); err != nil {
			return err
		}
	}
	pod := buildSimplePod(req)
	if _, err := k.Client().CoreV1().Pods(req.Namespace).Create(ctx, pod, metav1.CreateOptions{}); err != nil {
		return apperror.Internal(fmt.Sprintf("快捷创建 Pod 失败: %v", err))
	}
	return nil
}

func (s *K8sPodService) UpdateSimple(ctx context.Context, req PodCreateSimpleRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if err := validateRFC1123Subdomain(req.Name); err != nil {
		return err
	}
	if cn := strings.TrimSpace(req.ContainerName); cn != "" {
		if err := validateRFC1123Label(cn); err != nil {
			return err
		}
	} else {
		if err := validateRFC1123Label(req.Name); err != nil {
			return err
		}
	}

	client := k.Client().CoreV1().Pods(req.Namespace)
	existing, err := client.Get(ctx, req.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return apperror.BadRequest("Pod 不存在，无法编辑")
		}
		return apperror.Internal(fmt.Sprintf("获取 Pod 失败: %v", err))
	}

	desired := buildSimplePod(req)
	if msg := workloadManagedPodHint(ctx, k, existing); msg != "" {
		return apperror.BadRequest(msg)
	}
	if canUpdateImageOnly(existing, desired) {
		// Kubernetes 生态做法：仅更新镜像时，不删除重建 Pod
		existing = existing.DeepCopy()
		existing.Spec.Containers[0].Image = desired.Spec.Containers[0].Image
		if _, err := client.Update(ctx, existing, metav1.UpdateOptions{}); err != nil {
			return apperror.Internal(fmt.Sprintf("更新 Pod 镜像失败: %v", err))
		}
		return nil
	}

	// 其他字段多为不可变：按同名删除后重建，并等待删除完成避免 Terminating 占用导致创建失败
	grace := int64(0)
	_ = client.Delete(ctx, req.Name, metav1.DeleteOptions{GracePeriodSeconds: &grace})
	deadline := time.Now().Add(30 * time.Second)
	for {
		_, err := client.Get(ctx, req.Name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				break
			}
			return apperror.Internal(fmt.Sprintf("等待 Pod 删除失败: %v", err))
		}
		if time.Now().After(deadline) {
			return apperror.BadRequest("Pod 正在删除中（Terminating），请稍后再试")
		}
		time.Sleep(500 * time.Millisecond)
	}

	if _, err := client.Create(ctx, desired, metav1.CreateOptions{}); err != nil {
		return apperror.Internal(fmt.Sprintf("编辑并重建 Pod 失败: %v", err))
	}
	return nil
}

func workloadManagedPodHint(ctx context.Context, k *kom.Kubectl, pod *corev1.Pod) string {
	if k == nil || pod == nil {
		return ""
	}
	owner := controllerOwner(pod.OwnerReferences)
	if owner == nil {
		return ""
	}
	switch owner.Kind {
	case "StatefulSet":
		return "该 Pod 由 StatefulSet 管理，直接编辑 Pod 可能会被控制器回滚/重建；请到 StatefulSet 中修改镜像或配置后滚动更新。"
	case "ReplicaSet":
		// 绝大多数情况 ReplicaSet 来自 Deployment，尽力探测 Deployment 名称
		rs, err := k.Client().AppsV1().ReplicaSets(pod.Namespace).Get(ctx, owner.Name, metav1.GetOptions{})
		if err == nil {
			if depName := rsOwnerDeploymentName(rs); depName != "" {
				return fmt.Sprintf("该 Pod 由 Deployment(%s) 管理，直接编辑 Pod 可能会被控制器回滚/重建；请到 Deployment 中修改镜像或配置后滚动更新。", depName)
			}
		}
		return "该 Pod 由 ReplicaSet（通常来自 Deployment）管理，直接编辑 Pod 可能会被控制器回滚/重建；请到对应工作负载中修改镜像或配置后滚动更新。"
	case "DaemonSet":
		return "该 Pod 由 DaemonSet 管理，直接编辑 Pod 可能会被控制器回滚/重建；请到 DaemonSet 中修改镜像或配置后滚动更新。"
	}
	return ""
}

func controllerOwner(refs []metav1.OwnerReference) *metav1.OwnerReference {
	for i := range refs {
		if refs[i].Controller != nil && *refs[i].Controller {
			return &refs[i]
		}
	}
	return nil
}

func rsOwnerDeploymentName(rs *appsv1.ReplicaSet) string {
	if rs == nil {
		return ""
	}
	for i := range rs.OwnerReferences {
		ref := rs.OwnerReferences[i]
		if ref.Controller != nil && *ref.Controller && ref.Kind == "Deployment" && ref.Name != "" {
			return ref.Name
		}
	}
	return ""
}

func canUpdateImageOnly(existing, desired *corev1.Pod) bool {
	if existing == nil || desired == nil {
		return false
	}
	if existing.Name != desired.Name || existing.Namespace != desired.Namespace {
		return false
	}
	// “快捷创建”只支持单容器；多容器场景保守走重建
	if len(existing.Spec.Containers) != 1 || len(desired.Spec.Containers) != 1 {
		return false
	}
	// 容器名变更属于重建场景
	if existing.Spec.Containers[0].Name != desired.Spec.Containers[0].Name {
		return false
	}
	// 仅镜像变更才允许走 Update；其他字段一律走重建，避免用户“以为已改但实际上没生效”
	if existing.Spec.Containers[0].Image == desired.Spec.Containers[0].Image {
		return false
	}

	if strings.TrimSpace(string(existing.Spec.Containers[0].ImagePullPolicy)) != strings.TrimSpace(string(desired.Spec.Containers[0].ImagePullPolicy)) {
		return false
	}
	if strings.TrimSpace(string(existing.Spec.RestartPolicy)) != strings.TrimSpace(string(desired.Spec.RestartPolicy)) {
		return false
	}
	if !reflect.DeepEqual(existing.Spec.NodeSelector, desired.Spec.NodeSelector) {
		return false
	}
	if strings.TrimSpace(existing.Spec.PriorityClassName) != strings.TrimSpace(desired.Spec.PriorityClassName) {
		return false
	}
	if !reflect.DeepEqual(existing.Spec.Affinity, desired.Spec.Affinity) {
		return false
	}
	if !reflect.DeepEqual(existing.Spec.Tolerations, desired.Spec.Tolerations) {
		return false
	}

	exC := existing.Spec.Containers[0]
	desC := desired.Spec.Containers[0]
	if !reflect.DeepEqual(exC.Command, desC.Command) {
		return false
	}
	if !reflect.DeepEqual(exC.Ports, desC.Ports) {
		return false
	}
	if !reflect.DeepEqual(envToMap(exC.Env), envToMap(desC.Env)) {
		return false
	}
	if !resourceRequirementsEqual(exC.Resources, desC.Resources) {
		return false
	}
	if !reflect.DeepEqual(existing.Labels, desired.Labels) {
		return false
	}
	return true
}

func envToMap(env []corev1.EnvVar) map[string]string {
	m := make(map[string]string, len(env))
	for _, e := range env {
		name := strings.TrimSpace(e.Name)
		if name == "" {
			continue
		}
		m[name] = e.Value
	}
	return m
}

func resourceRequirementsEqual(a, b corev1.ResourceRequirements) bool {
	// Requests/Limits 都为空时认为相等
	if len(a.Requests) == 0 && len(a.Limits) == 0 && len(b.Requests) == 0 && len(b.Limits) == 0 {
		return true
	}
	if len(a.Requests) != len(b.Requests) || len(a.Limits) != len(b.Limits) {
		return false
	}
	for k, av := range a.Requests {
		bv, ok := b.Requests[k]
		if !ok || av.Cmp(bv) != 0 {
			return false
		}
	}
	for k, av := range a.Limits {
		bv, ok := b.Limits[k]
		if !ok || av.Cmp(bv) != 0 {
			return false
		}
	}
	return true
}

func buildSimplePod(req PodCreateSimpleRequest) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
			Labels:    req.Labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  req.Name,
					Image: req.Image,
				},
			},
			RestartPolicy: corev1.RestartPolicyAlways,
		},
	}
	container := &pod.Spec.Containers[0]
	if cn := strings.TrimSpace(req.ContainerName); cn != "" {
		container.Name = cn
	}
	switch strings.TrimSpace(req.ImagePullPolicy) {
	case "Always", "IfNotPresent", "Never":
		container.ImagePullPolicy = corev1.PullPolicy(req.ImagePullPolicy)
	}
	switch strings.TrimSpace(req.RestartPolicy) {
	case "Always", "OnFailure", "Never":
		pod.Spec.RestartPolicy = corev1.RestartPolicy(req.RestartPolicy)
	}
	if req.Port > 0 {
		container.Ports = []corev1.ContainerPort{{ContainerPort: req.Port}}
	}
	if len(req.Env) > 0 {
		envs := make([]corev1.EnvVar, 0, len(req.Env))
		for k, v := range req.Env {
			name := strings.TrimSpace(k)
			if name == "" {
				continue
			}
			envs = append(envs, corev1.EnvVar{Name: name, Value: v})
		}
		container.Env = envs
	}
	reqs := corev1.ResourceList{}
	lims := corev1.ResourceList{}
	if v := strings.TrimSpace(req.RequestsCPU); v != "" {
		if q, e := resource.ParseQuantity(v); e == nil {
			reqs[corev1.ResourceCPU] = q
		}
	}
	if v := strings.TrimSpace(req.RequestsMemory); v != "" {
		if q, e := resource.ParseQuantity(v); e == nil {
			reqs[corev1.ResourceMemory] = q
		}
	}
	if v := strings.TrimSpace(req.LimitsCPU); v != "" {
		if q, e := resource.ParseQuantity(v); e == nil {
			lims[corev1.ResourceCPU] = q
		}
	}
	if v := strings.TrimSpace(req.LimitsMemory); v != "" {
		if q, e := resource.ParseQuantity(v); e == nil {
			lims[corev1.ResourceMemory] = q
		}
	}
	if len(reqs) > 0 || len(lims) > 0 {
		container.Resources = corev1.ResourceRequirements{Requests: reqs, Limits: lims}
	}
	if len(req.Tolerations) > 0 {
		tolerations := make([]corev1.Toleration, 0, len(req.Tolerations))
		for _, t := range req.Tolerations {
			op := strings.TrimSpace(t.Operator)
			if op == "" {
				op = string(corev1.TolerationOpEqual)
			}
			effect := strings.TrimSpace(t.Effect)
			item := corev1.Toleration{
				Key:      strings.TrimSpace(t.Key),
				Operator: corev1.TolerationOperator(op),
				Value:    strings.TrimSpace(t.Value),
				Effect:   corev1.TaintEffect(effect),
			}
			if t.TolerationSeconds != nil {
				item.TolerationSeconds = t.TolerationSeconds
			}
			tolerations = append(tolerations, item)
		}
		pod.Spec.Tolerations = tolerations
	}
	if len(req.NodeSelector) > 0 {
		pod.Spec.NodeSelector = req.NodeSelector
	}
	if v := strings.TrimSpace(req.PriorityClassName); v != "" {
		pod.Spec.PriorityClassName = v
	}
	if req.Affinity != nil {
		pod.Spec.Affinity = req.Affinity
	}
	if cmd := strings.TrimSpace(req.Command); cmd != "" {
		container.Command = []string{"sh", "-c", cmd}
	}
	return pod
}

func filterLogLines(text, keyword, startStr, endStr string) string {
	lines := strings.Split(text, "\n")
	kw := strings.ToLower(strings.TrimSpace(keyword))
	start, _ := parseAnyTime(startStr)
	end, _ := parseAnyTime(endStr)
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if kw != "" && !strings.Contains(strings.ToLower(line), kw) {
			continue
		}
		if !start.IsZero() || !end.IsZero() {
			ts, ok := extractLineTimestamp(line)
			if ok {
				if !start.IsZero() && ts.Before(start) {
					continue
				}
				if !end.IsZero() && ts.After(end) {
					continue
				}
			}
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func parseAnyTime(v string) (time.Time, error) {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Time{}, nil
	}
	layouts := []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05"}
	for _, l := range layouts {
		if t, err := time.Parse(l, v); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid time")
}

var logTSRegexp = regexp.MustCompile(`\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}Z?`)

func extractLineTimestamp(line string) (time.Time, bool) {
	match := logTSRegexp.FindString(line)
	if match == "" {
		return time.Time{}, false
	}
	if t, err := parseAnyTime(match); err == nil {
		return t, true
	}
	return time.Time{}, false
}
