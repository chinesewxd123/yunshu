package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"yunshu/internal/pkg/constants"
	"yunshu/internal/pkg/eventbus"
	"yunshu/internal/pkg/response"
	"yunshu/internal/service"

	"github.com/gin-gonic/gin"
)

type K8sResourceWatchHandler struct {
	runtime *service.K8sRuntimeService
}

func NewK8sResourceWatchHandler(runtime *service.K8sRuntimeService) *K8sResourceWatchHandler {
	return &K8sResourceWatchHandler{runtime: runtime}
}

// Stream GET /api/v1/k8s/resource-watch/stream — Kubernetes Watch → SSE（按需，非常驻 Informer）。
func (h *K8sResourceWatchHandler) Stream(c *gin.Context) {
	var q service.K8sResourceWatchQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		response.Error(c, constants.ErrBadRequestWithMsg(err.Error()))
		return
	}

	_, cfg, err := h.runtime.GetClusterRestConfig(c.Request.Context(), q.ClusterID)
	if err != nil {
		response.Error(c, err)
		return
	}
	def, err := service.ResolveWatchTarget(cfg, &q)
	if err != nil {
		response.Error(c, err)
		return
	}
	ns := strings.TrimSpace(q.Namespace)
	if def.Namespaced && ns == "" {
		response.Error(c, constants.ErrBadRequestWithMsg("该资源为命名空间级，请传入 namespace 查询参数"))
		return
	}

	flush := func() {
		if f, ok := c.Writer.(http.Flusher); ok {
			f.Flush()
		}
	}

	resourceLabel := service.WatchResourceLabel(def, &q)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	c.Status(http.StatusOK)
	flush()

	eventbus.Default().Publish(eventbus.Event{
		Type: eventbus.K8sResourceWatchStarted,
		Payload: map[string]any{
			"cluster_id": q.ClusterID,
			"resource":   resourceLabel,
			"gvr":        fmt.Sprintf("%s/%s/%s", def.GVR.Group, def.GVR.Version, def.GVR.Resource),
			"namespace":  strings.TrimSpace(q.Namespace),
		},
	})
	defer eventbus.Default().Publish(eventbus.Event{
		Type: eventbus.K8sResourceWatchClientClose,
		Payload: map[string]any{
			"cluster_id": q.ClusterID,
			"resource":   resourceLabel,
		},
	})

	if err := h.runtime.StreamResourceWatch(c.Request.Context(), cfg, q, def, c.Writer, flush); err != nil {
		b, _ := json.Marshal(map[string]any{"message": err.Error()})
		_, _ = fmt.Fprintf(c.Writer, "event: error\ndata: %s\n\n", string(b))
		flush()
	}
}
