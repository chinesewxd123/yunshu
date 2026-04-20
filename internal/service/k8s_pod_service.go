package service

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/k8sutil"

	kom "github.com/weibaohui/kom/kom"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
)

type PodListQuery = ClusterNamespaceOptionalKeywordQuery

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

type PodFileQuery struct {
	ClusterID uint   `form:"cluster_id" binding:"required"`
	Namespace string `form:"namespace" binding:"required"`
	Name      string `form:"name" binding:"required"`
	Container string `form:"container"`
	Path      string `form:"path"`
}

type PodExecRequest struct {
	ClusterNamespaceNameCommandRequest
	Container string `json:"container"`
}

type PodDeleteRequest = ClusterNamespaceNameRequest

type PodDetailQuery = ClusterNamespaceNameQuery
type PodEventQuery = ClusterNamespaceNameQuery
type PodRestartRequest = ClusterNamespaceNameRequest
type PodCreateYAMLRequest = ClusterManifestApplyRequest

type PodCreateSimpleRequest struct {
	ClusterNamespaceNameRequest
	Image   string `json:"image" binding:"required"`
	Command string `json:"command"`

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

type PodFileItem struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Type        string `json:"type"`
	IsDir       bool   `json:"is_dir"`
	Size        int64  `json:"size"`
	Permissions string `json:"permissions"`
	Owner       string `json:"owner"`
	Group       string `json:"group"`
	ModTime     string `json:"mod_time"`
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

// NewK8sPodService 创建相关逻辑。
func NewK8sPodService(runtime *K8sRuntimeService) *K8sPodService {
	return &K8sPodService{runtime: runtime}
}

// List 查询列表相关的业务逻辑。
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

// Detail 查询详情相关的业务逻辑。
func (s *K8sPodService) Detail(ctx context.Context, query PodDetailQuery) (*PodDetail, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, query.ClusterID)
	if err != nil {
		return nil, err
	}
	var pod corev1.Pod
	if err := k.WithContext(ctx).Resource(&corev1.Pod{}).Namespace(query.Namespace).Name(query.Name).Get(&pod).Error; err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取 Pod 详情失败: %v", err))
	}
	containers := make([]PodContainerInfo, 0, len(pod.Status.ContainerStatuses))
	for _, c := range pod.Status.ContainerStatuses {
		containers = append(containers, PodContainerInfo{
			Name:         c.Name,
			Image:        c.Image,
			Ready:        c.Ready,
			RestartCount: c.RestartCount,
			State:        k8sutil.ContainerState(c.State),
		})
	}
	initContainers := make([]PodContainerInfo, 0, len(pod.Status.InitContainerStatuses))
	for _, c := range pod.Status.InitContainerStatuses {
		initContainers = append(initContainers, PodContainerInfo{
			Name:         c.Name,
			Image:        c.Image,
			Ready:        c.Ready,
			RestartCount: c.RestartCount,
			State:        k8sutil.ContainerState(c.State),
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

// Events 执行对应的业务逻辑。
func (s *K8sPodService) Events(ctx context.Context, query PodEventQuery) ([]PodEventItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, query.ClusterID)
	if err != nil {
		return nil, err
	}
	var list []corev1.Event
	if err := k.WithContext(ctx).
		Resource(&corev1.Event{}).
		Namespace(query.Namespace).
		WithFieldSelector(fmt.Sprintf("involvedObject.name=%s,involvedObject.kind=Pod", query.Name)).
		List(&list).Error; err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取 Pod 事件失败: %v", err))
	}
	out := make([]PodEventItem, 0, len(list))
	for _, e := range list {
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

// Delete 删除相关的业务逻辑。
func (s *K8sPodService) Delete(ctx context.Context, req PodDeleteRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if err := k.WithContext(ctx).Resource(&corev1.Pod{}).Namespace(req.Namespace).Name(req.Name).Delete().Error; err != nil {
		return apperror.Internal(fmt.Sprintf("删除 Pod 失败: %v", err))
	}
	return nil
}

// Exec 执行对应的业务逻辑。
func (s *K8sPodService) Exec(ctx context.Context, req PodExecRequest) (string, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return "", err
	}
	container := strings.TrimSpace(req.Container)
	if container == "" {
		var pod corev1.Pod
		e := k.WithContext(ctx).Resource(&corev1.Pod{}).Namespace(req.Namespace).Name(req.Name).Get(&pod).Error
		if e == nil && len(pod.Spec.Containers) > 0 {
			container = pod.Spec.Containers[0].Name
		}
	}
	cmd := strings.Fields(strings.TrimSpace(req.Command))
	if len(cmd) == 0 {
		return "", apperror.BadRequest("命令不能为空")
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
// 保留 client-go 直连：需要 remotecommand.TerminalSizeQueue 来支持前端终端窗口 resize，
// 当前 kom StreamExecute 不暴露该能力（且默认 TTY=false），因此这里采用最小原生实现。
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

// GetLogs 获取相关的业务逻辑。
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
	var stream io.ReadCloser
	err = k.WithContext(ctx).
		Namespace(query.Namespace).
		Name(query.Name).
		Ctl().
		Pod().
		ContainerName(strings.TrimSpace(query.Container)).
		GetLogs(&stream, opts).Error
	if err != nil {
		return "", apperror.Internal(fmt.Sprintf("获取 Pod 日志失败: %v", err))
	}
	if stream == nil {
		return "", apperror.Internal("获取 Pod 日志失败: 日志流为空")
	}
	defer stream.Close()
	var buf bytes.Buffer
	if _, err = io.Copy(&buf, stream); err != nil {
		return "", apperror.Internal(fmt.Sprintf("读取 Pod 日志失败: %v", err))
	}
	return k8sutil.FilterLogLines(buf.String(), query.Keyword, query.StartTime, query.EndTime), nil
}

// StreamLogs 执行对应的业务逻辑。
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
	var stream io.ReadCloser
	err = k.WithContext(ctx).
		Namespace(query.Namespace).
		Name(query.Name).
		Ctl().
		Pod().
		ContainerName(strings.TrimSpace(query.Container)).
		GetLogs(&stream, opts).Error
	if err != nil {
		return apperror.Internal(fmt.Sprintf("获取 Pod 流日志失败: %v", err))
	}
	if stream == nil {
		return apperror.Internal("获取 Pod 流日志失败: 日志流为空")
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

// Restart 执行对应的业务逻辑。
func (s *K8sPodService) Restart(ctx context.Context, req PodRestartRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if err := k.WithContext(ctx).Resource(&corev1.Pod{}).Namespace(req.Namespace).Name(req.Name).Delete().Error; err != nil {
		return apperror.Internal(fmt.Sprintf("重启 Pod 失败: %v", err))
	}
	return nil
}

// ListFiles 查询列表相关的业务逻辑。
func (s *K8sPodService) ListFiles(ctx context.Context, query PodFileQuery) ([]PodFileItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, query.ClusterID)
	if err != nil {
		return nil, err
	}
	path := strings.TrimSpace(query.Path)
	if path == "" {
		path = "/"
	}
	files, err := k.WithContext(ctx).
		Namespace(query.Namespace).
		Name(query.Name).
		Ctl().
		Pod().
		ContainerName(strings.TrimSpace(query.Container)).
		ListAllFiles(path)
	if err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取 Pod 文件列表失败: %v", err))
	}
	out := make([]PodFileItem, 0, len(files))
	for _, f := range files {
		if f == nil {
			continue
		}
		out = append(out, PodFileItem{
			Name:        f.Name,
			Path:        f.Path,
			Type:        f.Type,
			IsDir:       f.IsDir,
			Size:        f.Size,
			Permissions: f.Permissions,
			Owner:       f.Owner,
			Group:       f.Group,
			ModTime:     f.ModTime,
		})
	}
	return out, nil
}

// ReadFile 执行对应的业务逻辑。
func (s *K8sPodService) ReadFile(ctx context.Context, query PodFileQuery) ([]byte, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, query.ClusterID)
	if err != nil {
		return nil, err
	}
	path := strings.TrimSpace(query.Path)
	if path == "" {
		return nil, apperror.BadRequest("路径不能为空")
	}
	data, err := k.WithContext(ctx).
		Namespace(query.Namespace).
		Name(query.Name).
		Ctl().
		Pod().
		ContainerName(strings.TrimSpace(query.Container)).
		DownloadFile(path)
	if err != nil {
		return nil, apperror.Internal(fmt.Sprintf("读取 Pod 文件失败: %v", err))
	}
	return data, nil
}

// DeleteFile 删除相关的业务逻辑。
func (s *K8sPodService) DeleteFile(ctx context.Context, query PodFileQuery) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, query.ClusterID)
	if err != nil {
		return err
	}
	path := strings.TrimSpace(query.Path)
	if path == "" {
		return apperror.BadRequest("路径不能为空")
	}
	if _, err := k.WithContext(ctx).
		Namespace(query.Namespace).
		Name(query.Name).
		Ctl().
		Pod().
		ContainerName(strings.TrimSpace(query.Container)).
		DeleteFile(path); err != nil {
		return apperror.Internal(fmt.Sprintf("删除 Pod 文件失败: %v", err))
	}
	return nil
}

// UploadFile 执行对应的业务逻辑。
func (s *K8sPodService) UploadFile(ctx context.Context, query PodFileQuery, filename string, r io.Reader) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, query.ClusterID)
	if err != nil {
		return err
	}
	dest := strings.TrimSpace(query.Path)
	if dest == "" {
		dest = "/tmp"
	}
	tmp, err := os.CreateTemp("", "pod-upload-*"+filepath.Ext(filename))
	if err != nil {
		return apperror.Internal(fmt.Sprintf("创建临时文件失败: %v", err))
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()
	if _, err := io.Copy(tmp, r); err != nil {
		return apperror.Internal(fmt.Sprintf("保存上传文件失败: %v", err))
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return apperror.Internal(fmt.Sprintf("处理上传文件失败: %v", err))
	}
	if err := k.WithContext(ctx).
		Namespace(query.Namespace).
		Name(query.Name).
		Ctl().
		Pod().
		ContainerName(strings.TrimSpace(query.Container)).
		UploadFile(dest, tmp); err != nil {
		return apperror.Internal(fmt.Sprintf("上传文件到 Pod 失败: %v", err))
	}
	return nil
}

// CreateByYAML 创建相关的业务逻辑。
func (s *K8sPodService) CreateByYAML(ctx context.Context, req PodCreateYAMLRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	_ = k.WithContext(ctx).Applier().Apply(req.Manifest)
	return nil
}

// CreateSimple 创建相关的业务逻辑。
func (s *K8sPodService) CreateSimple(ctx context.Context, req PodCreateSimpleRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if err := k8sutil.ValidateRFC1123Subdomain(req.Name); err != nil {
		return err
	}
	if cn := strings.TrimSpace(req.ContainerName); cn != "" {
		if err := k8sutil.ValidateRFC1123Label(cn); err != nil {
			return err
		}
	} else {
		// default container name equals pod name; validate as label too
		if err := k8sutil.ValidateRFC1123Label(req.Name); err != nil {
			return err
		}
	}
	pod := buildSimplePod(req)
	if err := k.WithContext(ctx).Resource(&corev1.Pod{}).Namespace(req.Namespace).Create(pod).Error; err != nil {
		return apperror.Internal(fmt.Sprintf("快捷创建 Pod 失败: %v", err))
	}
	return nil
}

// UpdateSimple 更新相关的业务逻辑。
func (s *K8sPodService) UpdateSimple(ctx context.Context, req PodCreateSimpleRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if err := k8sutil.ValidateRFC1123Subdomain(req.Name); err != nil {
		return err
	}
	if cn := strings.TrimSpace(req.ContainerName); cn != "" {
		if err := k8sutil.ValidateRFC1123Label(cn); err != nil {
			return err
		}
	} else {
		if err := k8sutil.ValidateRFC1123Label(req.Name); err != nil {
			return err
		}
	}

	var existing corev1.Pod
	if err := k.WithContext(ctx).Resource(&corev1.Pod{}).Namespace(req.Namespace).Name(req.Name).Get(&existing).Error; err != nil {
		if apierrors.IsNotFound(err) {
			return apperror.BadRequest("Pod 不存在，无法编辑")
		}
		return apperror.Internal(fmt.Sprintf("获取 Pod 失败: %v", err))
	}

	desired := buildSimplePod(req)
	if msg := workloadManagedPodHint(ctx, k, &existing); msg != "" {
		return apperror.BadRequest(msg)
	}
	if k8sutil.CanUpdateImageOnly(&existing, desired) {
		// Kubernetes 生态做法：仅更新镜像时，不删除重建 Pod
		copyPod := existing.DeepCopy()
		copyPod.Spec.Containers[0].Image = desired.Spec.Containers[0].Image
		if err := k.WithContext(ctx).Resource(&corev1.Pod{}).Namespace(req.Namespace).Update(copyPod).Error; err != nil {
			return apperror.Internal(fmt.Sprintf("更新 Pod 镜像失败: %v", err))
		}
		return nil
	}

	// 其他字段多为不可变：按同名删除后重建，并等待删除完成避免 Terminating 占用导致创建失败
	_ = k.WithContext(ctx).Resource(&corev1.Pod{}).Namespace(req.Namespace).Name(req.Name).Delete().Error
	deadline := time.Now().Add(30 * time.Second)
	for {
		var current corev1.Pod
		err := k.WithContext(ctx).Resource(&corev1.Pod{}).Namespace(req.Namespace).Name(req.Name).Get(&current).Error
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

	if err := k.WithContext(ctx).Resource(&corev1.Pod{}).Namespace(req.Namespace).Create(desired).Error; err != nil {
		return apperror.Internal(fmt.Sprintf("编辑并重建 Pod 失败: %v", err))
	}
	return nil
}

func workloadManagedPodHint(ctx context.Context, k *kom.Kubectl, pod *corev1.Pod) string {
	if k == nil || pod == nil {
		return ""
	}
	owner := k8sutil.ControllerOwner(pod.OwnerReferences)
	if owner == nil {
		return ""
	}
	switch owner.Kind {
	case "StatefulSet":
		return "该 Pod 由 StatefulSet 管理，直接编辑 Pod 可能会被控制器回滚/重建；请到 StatefulSet 中修改镜像或配置后滚动更新。"
	case "ReplicaSet":
		// 绝大多数情况 ReplicaSet 来自 Deployment，尽力探测 Deployment 名称
		var rs appsv1.ReplicaSet
		err := k.WithContext(ctx).Resource(&appsv1.ReplicaSet{}).Namespace(pod.Namespace).Name(owner.Name).Get(&rs).Error
		if err == nil {
			if depName := k8sutil.RSOwnerDeploymentName(&rs); depName != "" {
				return fmt.Sprintf("该 Pod 由 Deployment(%s) 管理，直接编辑 Pod 可能会被控制器回滚/重建；请到 Deployment 中修改镜像或配置后滚动更新。", depName)
			}
		}
		return "该 Pod 由 ReplicaSet（通常来自 Deployment）管理，直接编辑 Pod 可能会被控制器回滚/重建；请到对应工作负载中修改镜像或配置后滚动更新。"
	case "DaemonSet":
		return "该 Pod 由 DaemonSet 管理，直接编辑 Pod 可能会被控制器回滚/重建；请到 DaemonSet 中修改镜像或配置后滚动更新。"
	}
	return ""
}

func buildSimplePod(req PodCreateSimpleRequest) *corev1.Pod {
	tolerations := make([]k8sutil.SimpleTolerationInput, 0, len(req.Tolerations))
	for _, t := range req.Tolerations {
		tolerations = append(tolerations, k8sutil.SimpleTolerationInput{
			Key:               t.Key,
			Operator:          t.Operator,
			Value:             t.Value,
			Effect:            t.Effect,
			TolerationSeconds: t.TolerationSeconds,
		})
	}
	return k8sutil.BuildSimplePod(k8sutil.SimplePodBuildInput{
		Name:              req.Name,
		Namespace:         req.Namespace,
		Image:             req.Image,
		Command:           req.Command,
		ContainerName:     req.ContainerName,
		ImagePullPolicy:   req.ImagePullPolicy,
		RestartPolicy:     req.RestartPolicy,
		Port:              req.Port,
		Env:               req.Env,
		Labels:            req.Labels,
		RequestsCPU:       req.RequestsCPU,
		RequestsMemory:    req.RequestsMemory,
		LimitsCPU:         req.LimitsCPU,
		LimitsMemory:      req.LimitsMemory,
		Tolerations:       tolerations,
		NodeSelector:      req.NodeSelector,
		PriorityClassName: req.PriorityClassName,
		Affinity:          req.Affinity,
	})
}
