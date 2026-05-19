package handler

import logx "yunshu/internal/pkg/logger"

func httpLog(component string) *logx.Component {
	return logx.Biz(component).WithLayer(logx.LayerHTTP)
}
