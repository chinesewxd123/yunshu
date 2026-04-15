package alertnotify

import "strings"

// Dims 从 Alertmanager labels 中抽取的通用维度（容器云、主机、中间件等均可映射）。
type Dims struct {
	Cluster      string
	Namespace    string
	WorkloadKind string
	WorkloadName string
	Pod          string
	Node         string
	Container    string
	Service      string
	Ingress      string
	Instance     string
	Job          string
	Endpoint     string
	MetricsPath  string
}

// ExtractDims 从告警 labels 解析展示维度（best-effort，未知标签自动忽略）。
func ExtractDims(labels map[string]string) Dims {
	d := Dims{}
	if labels == nil {
		return d
	}
	get := func(keys ...string) string {
		for _, k := range keys {
			if v := strings.TrimSpace(labels[k]); v != "" {
				return v
			}
		}
		return ""
	}
	d.Cluster = get("cluster", "kubernetes_cluster", "k8s_cluster", "region", "env")
	d.Namespace = get("namespace", "kubernetes_namespace")
	d.Pod = get("pod", "pod_name")
	d.Node = get("node", "kubernetes_node", "nodename", "instance_node", "hostname", "host")
	d.Container = get("container", "container_name")
	d.Service = get("service", "service_name")
	d.Ingress = get("ingress", "ingress_name")
	d.Instance = get("instance")
	d.Job = get("job")
	d.Endpoint = get("endpoint")
	d.MetricsPath = get("metrics_path")

	// 不要用 Prometheus scrape 的 `job` 推断 K8s Job，否则会误显示为 job/kube-state-metrics
	candidates := [][2]string{
		{"deployment", "deployment"},
		{"statefulset", "statefulset"},
		{"daemonset", "daemonset"},
		{"job_name", "job"},
		{"cronjob", "cronjob"},
		{"workload_kind", "workload"},
		{"workload", "workload"},
	}
	for _, it := range candidates {
		if v := strings.TrimSpace(labels[it[0]]); v != "" {
			d.WorkloadKind = it[1]
			d.WorkloadName = v
			break
		}
	}
	if d.WorkloadName == "" {
		if v := get("app", "app_kubernetes_io/name"); v != "" {
			d.WorkloadKind = "app"
			d.WorkloadName = v
		}
	}
	return d
}

// SafeOr 空串时返回 fallback。
func SafeOr(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return strings.TrimSpace(v)
}

// FormatWorkload 格式化工作负载展示。
func FormatWorkload(kind, name string) string {
	kind = strings.TrimSpace(kind)
	name = strings.TrimSpace(name)
	if kind == "" && name == "" {
		return "-"
	}
	if kind == "" {
		return name
	}
	if name == "" {
		return kind
	}
	return kind + "/" + name
}

// InferEnvironmentDisplay 在缺少标准 cluster 标签时，为主机/中间件/裸金属等告警提供可展示的「环境 / 实例」文案（写入事件与模板中的环境标识）。
func InferEnvironmentDisplay(labels map[string]string, dims Dims) string {
	get := func(keys ...string) string {
		for _, k := range keys {
			if labels == nil {
				continue
			}
			if v := strings.TrimSpace(labels[k]); v != "" {
				return v
			}
		}
		return ""
	}
	if v := get("region", "env", "environment", "datacenter", "dc", "prometheus", "monitor", "team"); v != "" {
		return v
	}
	if v := get("hostname", "nodename"); v != "" {
		return v
	}
	if v := strings.TrimSpace(dims.Instance); v != "" {
		return v
	}
	if v := strings.TrimSpace(dims.Node); v != "" {
		return v
	}
	if v := strings.TrimSpace(dims.Job); v != "" {
		return v
	}
	return ""
}
