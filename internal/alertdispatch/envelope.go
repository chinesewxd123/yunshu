package alertdispatch

// Envelope 一次外发语义单元（对应 Alertmanager 处理后待下发到各通道的同一份上下文）。
type Envelope struct {
	Source   string
	Title    string
	Severity string
	Status   string
	Payload  map[string]interface{}
}

// NewEnvelope 构造投递信封；payload 允许为 nil，发送前由调用方或通道逻辑按需初始化。
func NewEnvelope(source, title, severity, status string, payload map[string]interface{}) *Envelope {
	return &Envelope{
		Source:   source,
		Title:    title,
		Severity: severity,
		Status:   status,
		Payload:  payload,
	}
}

// PayloadOrEmpty 返回非 nil 的 map，避免通道内对 payload 判空分支扩散。
func (e *Envelope) PayloadOrEmpty() map[string]interface{} {
	if e == nil || e.Payload == nil {
		return map[string]interface{}{}
	}
	return e.Payload
}
