package k8sutil

import (
	"reflect"
	"regexp"
	"strings"

	"go-permission-system/internal/pkg/apperror"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	rfc1123LabelRe     = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	rfc1123SubdomainRe = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`)
)

func ValidateRFC1123Label(name string) error {
	n := strings.TrimSpace(name)
	if n == "" || !rfc1123LabelRe.MatchString(n) {
		return apperror.BadRequest("容器名称不合法：必须为 RFC1123 label（小写字母/数字/短横线，且首尾为字母或数字）")
	}
	return nil
}

func ValidateRFC1123Subdomain(name string) error {
	n := strings.TrimSpace(name)
	if n == "" || !rfc1123SubdomainRe.MatchString(n) {
		return apperror.BadRequest("Pod 名称不合法：必须为 RFC1123 subdomain（小写字母/数字/短横线/点，且首尾为字母或数字）")
	}
	return nil
}

func ContainerState(st corev1.ContainerState) string {
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

func ControllerOwner(refs []metav1.OwnerReference) *metav1.OwnerReference {
	for i := range refs {
		if refs[i].Controller != nil && *refs[i].Controller {
			return &refs[i]
		}
	}
	return nil
}

func RSOwnerDeploymentName(rs *appsv1.ReplicaSet) string {
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

func CanUpdateImageOnly(existing, desired *corev1.Pod) bool {
	if existing == nil || desired == nil {
		return false
	}
	if existing.Name != desired.Name || existing.Namespace != desired.Namespace {
		return false
	}
	if len(existing.Spec.Containers) != 1 || len(desired.Spec.Containers) != 1 {
		return false
	}
	if existing.Spec.Containers[0].Name != desired.Spec.Containers[0].Name {
		return false
	}
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

type SimpleTolerationInput struct {
	Key               string
	Operator          string
	Value             string
	Effect            string
	TolerationSeconds *int64
}

type SimplePodBuildInput struct {
	Name              string
	Namespace         string
	Image             string
	Command           string
	ContainerName     string
	ImagePullPolicy   string
	RestartPolicy     string
	Port              int32
	Env               map[string]string
	Labels            map[string]string
	RequestsCPU       string
	RequestsMemory    string
	LimitsCPU         string
	LimitsMemory      string
	Tolerations       []SimpleTolerationInput
	NodeSelector      map[string]string
	PriorityClassName string
	Affinity          *corev1.Affinity
}

func BuildSimplePod(req SimplePodBuildInput) *corev1.Pod {
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
