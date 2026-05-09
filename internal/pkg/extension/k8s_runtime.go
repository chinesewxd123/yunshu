// Package extension 提供轻量扩展钩子（对齐 k8m「插件可订阅系统事件」的思想，但不实现完整插件生命周期与 DB 状态机）。
// 业务或二开可在 init 中 RegisterKomRegisterHook 接入审计/指标；不含 AI/MCP。
package extension

import "sync"

var (
	komMu   sync.RWMutex
	komHook []func(clusterID string, registerSucceeded bool, errMsg string)
)

// RegisterKomRegisterHook 在 kom 向某集群注册成功或失败时调用（缓存命中未重注册时不触发）。
func RegisterKomRegisterHook(fn func(clusterID string, registerSucceeded bool, errMsg string)) {
	if fn == nil {
		return
	}
	komMu.Lock()
	komHook = append(komHook, fn)
	komMu.Unlock()
}

// NotifyKomRegister 由 K8sRuntimeService 在注册结果确定后调用。
func NotifyKomRegister(clusterID string, registerSucceeded bool, errMsg string) {
	komMu.RLock()
	hooks := append([]func(string, bool, string){}, komHook...)
	komMu.RUnlock()
	for _, fn := range hooks {
		fn(clusterID, registerSucceeded, errMsg)
	}
}
