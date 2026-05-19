package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/service/svcerr"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
)

// K8sResourceWatchQuery 将 client-go Watch 以 SSE 推送给浏览器（按需，非常驻 Informer 聚合）。
//
// 两种模式（二选一语义）：
// 1) 短名：不传 version，resource 为内置别名（如 pods、deployments）。
// 2) 任意 GVR：传 version（及可选 group，核心组留空）、resource 为 API 复数名（如 pods、horizontalpodautoscalers、customresources），由 RESTMapper 按当前集群 Discovery 校验。
type K8sResourceWatchQuery struct {
	ClusterID        uint   `form:"cluster_id" binding:"required"`
	Namespace        string `form:"namespace"`
	Group            string `form:"group"`
	Version          string `form:"version"`
	Resource         string `form:"resource"`
	LabelSelector    string `form:"label_selector"`
	FieldSelector    string `form:"field_selector"`
	ResourceVersion  string `form:"resource_version"`
	TimeoutSeconds   int    `form:"timeout_seconds"`
	HeartbeatSeconds int    `form:"heartbeat_seconds"`
}

type watchGVR struct {
	GVR        schema.GroupVersionResource
	Namespaced bool
}

// ResolveWatchTarget 解析 Watch 目标（短名表或 GVR+RESTMapper）；cfg 为已生效的集群 kubeconfig。
func ResolveWatchTarget(cfg *rest.Config, q *K8sResourceWatchQuery) (watchGVR, error) {
	ver := strings.TrimSpace(q.Version)
	if ver != "" {
		group := strings.TrimSpace(q.Group)
		res := strings.TrimSpace(q.Resource)
		if res == "" {
			return watchGVR{}, constants.ErrBadRequestWithMsg("GVR 模式需传入 resource（API 复数名，如 pods、jobs）")
		}
		return resolveGVRWithRESTMapper(cfg, group, ver, res)
	}
	slug := strings.TrimSpace(q.Resource)
	if slug == "" {
		return watchGVR{}, constants.ErrBadRequestWithMsg("请传入 resource 短名（如 pods），或同时传入 version+resource 使用 GVR 模式")
	}
	return resolveWatchGVR(slug)
}

func resolveGVRWithRESTMapper(cfg *rest.Config, group, version, resource string) (watchGVR, error) {
	disc, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return watchGVR{}, svcerr.Internal(context.Background(), "k8s.watch", "discovery_client", err, "discovery client: %v")
	}
	gr, err := restmapper.GetAPIGroupResources(disc)
	if err != nil {
		return watchGVR{}, svcerr.Internal(context.Background(), "k8s.watch", "api_groups", err, "discovery: %v")
	}
	mapper := restmapper.NewDiscoveryRESTMapper(gr)

	gvrIn := schema.GroupVersionResource{Group: group, Version: version, Resource: resource}
	resolved, err := mapper.ResourceFor(gvrIn)
	if err != nil {
		return watchGVR{}, constants.ErrBadRequestWithMsg(fmt.Sprintf("RESTMapper 无法解析 GVR %q/%q/%q: %v", group, version, resource, err))
	}
	gvks, err := mapper.KindsFor(resolved)
	if err != nil || len(gvks) == 0 {
		msg := ""
		if err != nil {
			msg = err.Error()
		}
		return watchGVR{}, constants.ErrBadRequestWithMsg(fmt.Sprintf("无法解析资源 Kind（%s）：%s", resolved.String(), msg))
	}
	gvk := gvks[0]
	mapList, err := mapper.RESTMappings(schema.GroupKind{Group: gvk.Group, Kind: gvk.Kind}, gvk.Version)
	if err != nil || len(mapList) == 0 {
		msg := ""
		if err != nil {
			msg = err.Error()
		}
		return watchGVR{}, constants.ErrBadRequestWithMsg(fmt.Sprintf("无法取得 RESTMapping（%s %s）：%s", gvk.Group, gvk.Kind, msg))
	}
	var chosen *apimeta.RESTMapping
	for _, m := range mapList {
		if m != nil && m.Resource == resolved {
			chosen = m
			break
		}
	}
	if chosen == nil {
		chosen = mapList[0]
	}
	namespaced := chosen.Scope.Name() == apimeta.RESTScopeNameNamespace
	return watchGVR{GVR: resolved, Namespaced: namespaced}, nil
}

// WatchResourceLabel 返回 SSE 与 eventbus 使用的资源标识（短名或规范化 GVR 路径）。
func WatchResourceLabel(def watchGVR, q *K8sResourceWatchQuery) string {
	if strings.TrimSpace(q.Version) != "" {
		return def.GVR.Group + "/" + def.GVR.Version + "/" + def.GVR.Resource
	}
	return strings.TrimSpace(q.Resource)
}

func resolveWatchGVR(name string) (watchGVR, error) {
	return builtinWatchGVRBySlug(name)
}

// StreamResourceWatch 将 Watch 结果以 SSE 写入 out（需已设置 Content-Type: text/event-stream）。
// def 须由 ResolveWatchTarget 与 q 使用同一集群 cfg 事先解析。
func (s *K8sRuntimeService) StreamResourceWatch(ctx context.Context, cfg *rest.Config, q K8sResourceWatchQuery, def watchGVR, out io.Writer, flush func()) error {
	ns := strings.TrimSpace(q.Namespace)
	if def.Namespaced && ns == "" {
		return constants.ErrBadRequestWithMsg("该资源为命名空间级，请传入 namespace")
	}
	if !def.Namespaced {
		ns = ""
	}

	timeout := q.TimeoutSeconds
	if timeout <= 0 {
		timeout = 3600
	}
	if timeout > 7200 {
		timeout = 7200
	}
	wctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return svcerr.Internal(ctx, "k8s.watch", "dynamic_client", err, "dynamic client: %v")
	}

	ri := dyn.Resource(def.GVR)
	opts := metav1.ListOptions{
		LabelSelector:   strings.TrimSpace(q.LabelSelector),
		FieldSelector:   strings.TrimSpace(q.FieldSelector),
		ResourceVersion: strings.TrimSpace(q.ResourceVersion),
	}

	var watcher watch.Interface
	if def.Namespaced {
		watcher, err = ri.Namespace(ns).Watch(wctx, opts)
	} else {
		watcher, err = ri.Watch(wctx, opts)
	}
	if err != nil {
		return svcerr.Internal(ctx, "k8s.watch", "watch", err, "watch: %v")
	}
	defer watcher.Stop()

	hb := q.HeartbeatSeconds
	if hb <= 0 {
		hb = 25
	}
	tick := time.NewTicker(time.Duration(hb) * time.Second)
	defer tick.Stop()

	writeFrame := func(evName string, payload map[string]any) error {
		b, jerr := json.Marshal(payload)
		if jerr != nil {
			return jerr
		}
		line := fmt.Sprintf("event: %s\ndata: %s\n\n", evName, string(b))
		if _, err := io.WriteString(out, line); err != nil {
			return err
		}
		if flush != nil {
			flush()
		}
		return nil
	}

	for {
		select {
		case <-wctx.Done():
			return nil
		case <-tick.C:
			if _, err := io.WriteString(out, ":heartbeat\n\n"); err != nil {
				return nil
			}
			if flush != nil {
				flush()
			}
		case ev, ok := <-watcher.ResultChan():
			if !ok {
				return nil
			}
			if ev.Type == watch.Error {
				st := statusFromObject(ev.Object)
				_ = writeFrame("error", map[string]any{
					"cluster_id": q.ClusterID,
					"reason":     "watch_error",
					"status":     st,
				})
				continue
			}
			u, convErr := runtime.DefaultUnstructuredConverter.ToUnstructured(ev.Object)
			if convErr != nil {
				_ = writeFrame("error", map[string]any{"cluster_id": q.ClusterID, "reason": convErr.Error()})
				continue
			}
			_ = writeFrame("watch", map[string]any{
				"cluster_id":    q.ClusterID,
				"resource":      WatchResourceLabel(def, &q),
				"group_version": def.GVR.Group + "/" + def.GVR.Version,
				"gvr":           def.GVR.Group + "/" + def.GVR.Version + "/" + def.GVR.Resource,
				"type":          string(ev.Type),
				"object":        u,
			})
		}
	}
}

func statusFromObject(obj any) map[string]any {
	if obj == nil {
		return nil
	}
	switch t := obj.(type) {
	case *unstructured.Unstructured:
		if t == nil {
			return nil
		}
		return t.UnstructuredContent()
	case *metav1.Status:
		if t == nil {
			return nil
		}
		out, err := runtime.DefaultUnstructuredConverter.ToUnstructured(t)
		if err != nil {
			return map[string]any{"message": t.Message, "code": t.Code}
		}
		return out
	default:
		out, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
		if err != nil {
			return map[string]any{"message": fmt.Sprintf("%v", obj)}
		}
		return out
	}
}
