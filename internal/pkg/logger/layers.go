package logger

// 统一分层标识，便于从日志定位错误来源（HTTP / API / Service / DAO / gRPC / Worker）。
const (
	LayerHTTP    = "http"
	LayerAPI     = "api"
	LayerService = "service"
	LayerDAO     = "dao"
	LayerGRPC    = "grpc"
	LayerWorker  = "worker"
)

const (
	channelInfo  = "info"
	channelError = "error"
	channelSQL   = "sql"
)
