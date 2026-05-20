package service

import (
	logx "yunshu/internal/pkg/logger"
	"yunshu/internal/service/svclog"
)

func mysqlBackupLog() *logx.Component {
	return svclog.Worker("mysql.backup")
}
