package service

import (
	logx "yunshu/internal/pkg/logger"
	"yunshu/internal/service/svclog"
)

func alertLog() *logx.Component {
	return svclog.Worker("alert")
}
