package k8seventforward

import (
	logx "yunshu/internal/pkg/logger"
	"yunshu/internal/service/svclog"
)

func forwardLog() *logx.Component {
	return svclog.Worker("k8s.event_forward")
}
