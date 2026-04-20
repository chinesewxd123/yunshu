package service

import (
	"context"
	"encoding/base64"
	"fmt"
	"sort"
	"strings"

	"go-permission-system/internal/pkg/apperror"
	"go-permission-system/internal/pkg/k8sutil"

	kom "github.com/weibaohui/kom/kom"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

type ConfigListQuery = ClusterNamespaceKeywordQuery
type ConfigDetailQuery = ClusterNamespaceNameQuery
type ConfigApplyRequest = ClusterManifestApplyRequest
type ConfigDeleteRequest = ClusterNamespaceNameQuery

type ConfigMapItem struct {
	Name         string `json:"name"`
	Namespace    string `json:"namespace"`
	DataCount    int    `json:"data_count"`
	CreationTime string `json:"creation_time"`
}

type SecretItem struct {
	Name         string `json:"name"`
	Namespace    string `json:"namespace"`
	Type         string `json:"type"`
	DataCount    int    `json:"data_count"`
	CreationTime string `json:"creation_time"`
}

type ConfigDetail struct {
	YAML         string            `json:"yaml"`
	StringData   map[string]string `json:"string_data,omitempty"`
	DecodedData  map[string]string `json:"decoded_data,omitempty"`
	BinaryKeySet []string          `json:"binary_keys,omitempty"`
}

type K8sConfigService struct {
	runtime *K8sRuntimeService
	dyn     *DynamicResourceService
}

// NewK8sConfigService 创建相关逻辑。
func NewK8sConfigService(runtime *K8sRuntimeService) *K8sConfigService {
	return &K8sConfigService{runtime: runtime, dyn: NewDynamicResourceService(runtime)}
}

var (
	configMapGVK = schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"}
	secretGVK    = schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"}
)

// ListConfigMaps 查询列表相关的业务逻辑。
func (s *K8sConfigService) ListConfigMaps(ctx context.Context, q ConfigListQuery) ([]ConfigMapItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	listU, err := s.dyn.ListByGVK(ctx, k, configMapGVK, q.Namespace)
	if err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取 ConfigMaps 失败: %v", err))
	}
	list := make([]corev1.ConfigMap, 0, len(listU))
	for _, item := range listU {
		var cm corev1.ConfigMap
		if e := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &cm); e != nil {
			continue
		}
		list = append(list, cm)
	}
	kw := strings.ToLower(strings.TrimSpace(q.Keyword))
	out := make([]ConfigMapItem, 0, len(list))
	for _, cm := range list {
		if kw != "" && !strings.Contains(strings.ToLower(cm.Name), kw) {
			continue
		}
		out = append(out, ConfigMapItem{
			Name:         cm.Name,
			Namespace:    cm.Namespace,
			DataCount:    len(cm.Data) + len(cm.BinaryData),
			CreationTime: cm.CreationTimestamp.Time.Format("2006-01-02 15:04:05"),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// ConfigMapDetail 执行对应的业务逻辑。
func (s *K8sConfigService) ConfigMapDetail(ctx context.Context, q ConfigDetailQuery) (*ConfigDetail, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	u, err := s.dyn.GetByGVK(ctx, k, configMapGVK, q.Namespace, q.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, apperror.BadRequest("ConfigMap 资源不存在")
		}
		return nil, apperror.Internal(fmt.Sprintf("获取 ConfigMap 失败: %v", err))
	}
	var obj corev1.ConfigMap
	_ = runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &obj)
	copyObj := obj.DeepCopy()
	copyObj.APIVersion = "v1"
	copyObj.Kind = "ConfigMap"
	copyObj.ManagedFields = nil
	y, _ := yaml.Marshal(copyObj)
	return &ConfigDetail{YAML: string(y)}, nil
}

// DeleteConfigMap 删除相关的业务逻辑。
func (s *K8sConfigService) DeleteConfigMap(ctx context.Context, req ConfigDeleteRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if err := s.dyn.DeleteByGVK(ctx, k, configMapGVK, req.Namespace, req.Name); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return apperror.Internal(fmt.Sprintf("删除 ConfigMap 失败: %v", err))
	}
	return nil
}

// ListSecrets 查询列表相关的业务逻辑。
func (s *K8sConfigService) ListSecrets(ctx context.Context, q ConfigListQuery) ([]SecretItem, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	listU, err := s.dyn.ListByGVK(ctx, k, secretGVK, q.Namespace)
	if err != nil {
		return nil, apperror.Internal(fmt.Sprintf("获取 Secrets 失败: %v", err))
	}
	list := make([]corev1.Secret, 0, len(listU))
	for _, item := range listU {
		var sec corev1.Secret
		if e := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, &sec); e != nil {
			continue
		}
		list = append(list, sec)
	}
	kw := strings.ToLower(strings.TrimSpace(q.Keyword))
	out := make([]SecretItem, 0, len(list))
	for _, sct := range list {
		if kw != "" && !strings.Contains(strings.ToLower(sct.Name), kw) {
			continue
		}
		out = append(out, SecretItem{
			Name:         sct.Name,
			Namespace:    sct.Namespace,
			Type:         string(sct.Type),
			DataCount:    len(sct.Data),
			CreationTime: sct.CreationTimestamp.Time.Format("2006-01-02 15:04:05"),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// SecretDetail 执行对应的业务逻辑。
func (s *K8sConfigService) SecretDetail(ctx context.Context, q ConfigDetailQuery) (*ConfigDetail, error) {
	_, k, err := s.runtime.GetClusterKubectl(ctx, q.ClusterID)
	if err != nil {
		return nil, err
	}
	u, err := s.dyn.GetByGVK(ctx, k, secretGVK, q.Namespace, q.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, apperror.BadRequest("Secret 资源不存在")
		}
		return nil, apperror.Internal(fmt.Sprintf("获取 Secret 失败: %v", err))
	}
	var obj corev1.Secret
	_ = runtime.DefaultUnstructuredConverter.FromUnstructured(u.Object, &obj)
	copyObj := obj.DeepCopy()
	copyObj.APIVersion = "v1"
	copyObj.Kind = "Secret"
	copyObj.ManagedFields = nil
	y, _ := yaml.Marshal(copyObj)
	decoded := map[string]string{}
	binaryKeys := make([]string, 0)
	for k, v := range copyObj.Data {
		if len(v) == 0 {
			decoded[k] = ""
			continue
		}
		// best-effort: try utf-8 string; if not, provide base64
		if k8sutil.IsLikelyText(v) {
			decoded[k] = string(v)
		} else {
			decoded[k] = base64.StdEncoding.EncodeToString(v)
			binaryKeys = append(binaryKeys, k)
		}
	}
	sort.Strings(binaryKeys)
	return &ConfigDetail{
		YAML:         string(y),
		DecodedData:  decoded,
		BinaryKeySet: binaryKeys,
	}, nil
}

// DeleteSecret 删除相关的业务逻辑。
func (s *K8sConfigService) DeleteSecret(ctx context.Context, req ConfigDeleteRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if err := s.dyn.DeleteByGVK(ctx, k, secretGVK, req.Namespace, req.Name); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return apperror.Internal(fmt.Sprintf("删除 Secret 失败: %v", err))
	}
	return nil
}

// Apply 提交申请相关的业务逻辑。
func (s *K8sConfigService) Apply(ctx context.Context, req ConfigApplyRequest) error {
	_, k, err := s.runtime.GetClusterKubectl(ctx, req.ClusterID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(req.Manifest) == "" {
		return apperror.BadRequest("资源清单不能为空")
	}

	cfgRefs := extractConfigRefs(req.Manifest)

	// Kom SDK 的 apply 对 Secret 的 stringData 兼容性在部分场景可能不理想，
	// 为了让用户直接粘贴 YAML 一定可用，这里把 stringData 转成 data（base64），再删除 stringData。
	sanitized, err := sanitizeSecretStringDataForApply(req.Manifest)
	if err != nil {
		return err
	}

	err = s.dyn.ApplyManifest(ctx, k, sanitized, func(c context.Context) bool {
		if len(cfgRefs) == 0 {
			return false
		}
		return configRefsAllExist(c, s.dyn, k, cfgRefs)
	})
	if err != nil {
		return apperror.Internal(fmt.Sprintf("应用 YAML 失败: %v", err))
	}
	return nil
}

type configRef struct {
	Kind      string
	Name      string
	Namespace string
}

func extractConfigRefs(manifest string) []configRef {
	docs := k8sutil.SplitYAMLDocs(manifest)
	out := make([]configRef, 0)

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
		kind = strings.TrimSpace(kind)
		if kind != "Secret" && kind != "ConfigMap" {
			continue
		}

		metadata, _ := m["metadata"].(map[string]any)
		name := ""
		namespace := "default"
		if metadata != nil {
			if v, ok := metadata["name"]; ok && v != nil {
				name = fmt.Sprint(v)
			}
			if v, ok := metadata["namespace"]; ok && v != nil && fmt.Sprint(v) != "" {
				namespace = fmt.Sprint(v)
			}
		}

		if strings.TrimSpace(name) == "" {
			continue
		}
		out = append(out, configRef{Kind: kind, Name: strings.TrimSpace(name), Namespace: strings.TrimSpace(namespace)})
	}

	return out
}

func configRefsAllExist(ctx context.Context, dyn *DynamicResourceService, k *kom.Kubectl, refs []configRef) bool {
	if k == nil || dyn == nil {
		return false
	}
	for _, ref := range refs {
		if strings.TrimSpace(ref.Name) == "" {
			continue
		}
		ns := strings.TrimSpace(ref.Namespace)
		if ns == "" {
			ns = "default"
		}
		switch ref.Kind {
		case "Secret":
			if _, err := dyn.GetByGVK(ctx, k, secretGVK, ns, ref.Name); err != nil {
				return false
			}
		case "ConfigMap":
			if _, err := dyn.GetByGVK(ctx, k, configMapGVK, ns, ref.Name); err != nil {
				return false
			}
		default:
			continue
		}
	}
	return true
}

func sanitizeSecretStringDataForApply(manifest string) (string, error) {
	docs := k8sutil.SplitYAMLDocs(manifest)
	out := make([]string, 0, len(docs))

	for _, doc := range docs {
		docTrim := strings.TrimSpace(doc)
		if docTrim == "" {
			continue
		}

		var m map[string]any
		if err := yaml.Unmarshal([]byte(docTrim), &m); err != nil {
			// 非法 YAML 时由下游（Kom SDK）报更准确错误，这里保持原样
			out = append(out, docTrim)
			continue
		}

		kind, _ := m["kind"].(string)
		if strings.TrimSpace(kind) != "Secret" {
			out = append(out, docTrim)
			continue
		}

		stringData, ok := m["stringData"].(map[string]any)
		if !ok || len(stringData) == 0 {
			out = append(out, docTrim)
			continue
		}

		data, _ := m["data"].(map[string]any)
		if data == nil {
			data = map[string]any{}
		}

		// stringData 转为 data（base64），同名 key 的 data 以 stringData 优先（模拟 k8s 转换效果）
		for k, v := range stringData {
			raw := fmt.Sprint(v)
			data[k] = base64.StdEncoding.EncodeToString([]byte(raw))
		}

		m["data"] = data
		delete(m, "stringData")

		y, err := yaml.Marshal(m)
		if err != nil {
			// marshal 失败则保持原样
			out = append(out, docTrim)
			continue
		}
		out = append(out, string(y))
	}

	return strings.Join(out, "\n---\n"), nil
}
