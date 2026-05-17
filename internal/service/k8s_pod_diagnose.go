package service

import (
	"context"
	"fmt"
	"strings"

	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/k8sutil"

	corev1 "k8s.io/api/core/v1"
)

// Diagnose 聚合 Pod 状态、事件与规则化排障建议（参考 kubectl describe / 常见故障模式，非外部二进制集成）。
func (s *K8sPodService) Diagnose(ctx context.Context, query PodDiagnoseQuery) (*PodDiagnoseResult, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, query.ClusterID)
	if err != nil {
		return nil, err
	}
	var pod corev1.Pod
	if err := k.WithContext(ctx).Resource(&corev1.Pod{}).Namespace(query.Namespace).Name(query.Name).Get(&pod).Error; err != nil {
		return nil, constants.ErrInternalWithMsg(fmt.Sprintf(constants.ErrFmtc52b9130d74c, err))
	}

	events, _ := s.Events(ctx, PodEventQuery(query))
	containers := make([]PodDiagnoseContainerIssue, 0, len(pod.Status.ContainerStatuses))
	readyCnt := 0
	for _, c := range pod.Status.ContainerStatuses {
		if c.Ready {
			readyCnt++
		}
		issue := PodDiagnoseContainerIssue{
			Name:         c.Name,
			State:        k8sutil.ContainerState(c.State),
			RestartCount: c.RestartCount,
		}
		if c.State.Waiting != nil {
			issue.Reason = c.State.Waiting.Reason
			issue.Message = c.State.Waiting.Message
		}
		if c.State.Terminated != nil {
			issue.Reason = c.State.Terminated.Reason
			issue.Message = c.State.Terminated.Message
		}
		if shouldFetchCrashLog(issue) {
			snippet, _ := s.fetchLogTail(ctx, query.ClusterID, query.Namespace, query.Name, c.Name, true, 80)
			issue.LogSnippet = snippet
		}
		containers = append(containers, issue)
	}
	ready := len(pod.Status.ContainerStatuses) > 0 && readyCnt == len(pod.Status.ContainerStatuses)
	hints := buildPodDiagnoseHints(pod, containers, events)
	summary := podDiagnoseSummary(pod, ready, hints)

	return &PodDiagnoseResult{
		Summary:    summary,
		Phase:      string(pod.Status.Phase),
		Ready:      ready,
		NodeName:   pod.Spec.NodeName,
		Hints:      hints,
		Events:     events,
		Containers: containers,
	}, nil
}

func shouldFetchCrashLog(issue PodDiagnoseContainerIssue) bool {
	r := strings.ToLower(issue.Reason)
	return strings.Contains(r, "crash") || strings.Contains(r, "error") || strings.Contains(r, "backoff") ||
		issue.RestartCount > 0 && issue.State != "Running"
}

func (s *K8sPodService) fetchLogTail(ctx context.Context, clusterID uint, ns, name, container string, previous bool, tail int64) (string, error) {
	logs, err := s.GetLogs(ctx, PodLogsQuery{
		ClusterID: clusterID,
		Namespace: ns,
		Name:      name,
		Container: container,
		Previous:  previous,
		TailLines: tail,
	})
	if err != nil {
		return "", err
	}
	const maxLen = 8000
	if len(logs) > maxLen {
		return logs[len(logs)-maxLen:], nil
	}
	return logs, nil
}

func podDiagnoseSummary(pod corev1.Pod, ready bool, hints []PodDiagnoseHint) string {
	phase := string(pod.Status.Phase)
	if ready && phase == "Running" {
		return "Pod 运行正常"
	}
	if len(hints) > 0 {
		return hints[0].Title
	}
	return fmt.Sprintf("Pod 阶段 %s，未就绪", phase)
}

func buildPodDiagnoseHints(pod corev1.Pod, containers []PodDiagnoseContainerIssue, events []PodEventItem) []PodDiagnoseHint {
	var hints []PodDiagnoseHint
	add := func(level, title, detail, action string) {
		hints = append(hints, PodDiagnoseHint{Level: level, Title: title, Detail: detail, Action: action})
	}

	phase := strings.ToLower(string(pod.Status.Phase))
	if phase == "pending" {
		add("warning", "Pod 处于 Pending", "通常因调度失败或镜像拉取未完成", "查看事件中的 FailedScheduling / FailedMount；检查节点资源与亲和性")
	}
	for _, c := range containers {
		reason := strings.ToLower(c.Reason)
		switch {
		case strings.Contains(reason, "imagepull"):
			add("error", fmt.Sprintf("容器 %s 镜像拉取失败", c.Name), c.Message, "确认镜像名、仓库凭证与 imagePullSecrets")
		case strings.Contains(reason, "crashloop"):
			add("error", fmt.Sprintf("容器 %s CrashLoopBackOff", c.Name), c.Message, "查看「上一实例」日志与就绪/存活探针配置")
		case strings.Contains(reason, "oom"):
			add("error", fmt.Sprintf("容器 %s 内存不足 (OOM)", c.Name), c.Message, "提高 limits.memory 或优化应用内存占用")
		case strings.Contains(reason, "createcontainerconfigerror"):
			add("error", fmt.Sprintf("容器 %s 配置错误", c.Name), c.Message, "检查 ConfigMap/Secret 挂载与 env 引用是否存在")
		case c.RestartCount >= 3 && !strings.Contains(strings.ToLower(c.State), "running"):
			add("warning", fmt.Sprintf("容器 %s 频繁重启 (%d 次)", c.Name, c.RestartCount), c.Message, "对比当前与 previous 日志，检查启动命令与依赖服务")
		}
	}
	for _, e := range events {
		r := strings.ToLower(e.Reason)
		msg := e.Message
		if strings.Contains(r, "failedscheduling") {
			add("error", "调度失败", msg, "检查节点标签、污点、资源 requests 与 PVC 绑定")
			break
		}
		if strings.Contains(r, "failedmount") {
			add("error", "卷挂载失败", msg, "确认 PV/PVC/Secret/ConfigMap 在目标命名空间可用")
			break
		}
	}
	for _, cond := range pod.Status.Conditions {
		if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionFalse && cond.Message != "" {
			add("info", "未通过 Ready 检查", cond.Message, "配置或放宽 readinessProbe；滚动发布时 maxUnavailable 建议为 0")
		}
	}
	if len(hints) == 0 && string(pod.Status.Phase) == "Running" {
		add("info", "未发现明显异常", "可结合实时日志与事件继续排查", "")
	}
	return hints
}
