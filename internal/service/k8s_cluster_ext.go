package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"yunshu/internal/repository"

	"gorm.io/gorm"
	"k8s.io/client-go/rest"
)

// getDirectConfigFromDict 从数据字典读取直连配置
func getDirectConfigFromDict(ctx context.Context, dictRepo *repository.DictEntryRepository, configKey string) (*DirectConfig, error) {
	// 优先按 label（配置键）查；兼容历史“按 value 作为键”查法。
	entry, err := dictRepo.GetByDictTypeAndLabel(ctx, "k8s_direct_config", configKey)
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		entry, err = dictRepo.GetByDictTypeAndValue(ctx, "k8s_direct_config", configKey)
	}
	if err != nil {
		return nil, fmt.Errorf("获取数据字典配置失败: %w", err)
	}

	var config DirectConfig
	if err := json.Unmarshal([]byte(entry.Value), &config); err != nil {
		return nil, fmt.Errorf("解析配置JSON失败: %w", err)
	}

	return &config, nil
}

// mergeDirectConfig 合并字典配置和用户配置（用户配置优先）
func mergeDirectConfig(base, override *DirectConfig) {
	if override.Server != "" {
		base.Server = override.Server
	}
	if override.CAData != "" {
		base.CAData = override.CAData
	}
	if override.Token != "" {
		base.Token = override.Token
	}
	if override.Username != "" {
		base.Username = override.Username
	}
	if override.Password != "" {
		base.Password = override.Password
	}
	if override.ClientCertData != "" {
		base.ClientCertData = override.ClientCertData
	}
	if override.ClientKeyData != "" {
		base.ClientKeyData = override.ClientKeyData
	}
	base.InsecureSkipTLSVerify = override.InsecureSkipTLSVerify || base.InsecureSkipTLSVerify
}

// preserveDirectAuthFromStored 编辑集群时前端常不回传已保存的 token/密码/证书（JSON 省略或空串），
// 若直接生成 kubeconfig 会丢失 Bearer，表现为：心跳能拿到版本（部分集群允许匿名 /version）、List Namespace 等返回 401 Unauthorized。
func preserveDirectAuthFromStored(storedJSON string, next *DirectConfig) {
	if next == nil || strings.TrimSpace(storedJSON) == "" {
		return
	}
	var prev DirectConfig
	if err := json.Unmarshal([]byte(storedJSON), &prev); err != nil {
		return
	}
	if strings.TrimSpace(next.Token) == "" && strings.TrimSpace(prev.Token) != "" {
		next.Token = prev.Token
	}
	if strings.TrimSpace(next.Password) == "" && strings.TrimSpace(prev.Password) != "" {
		next.Password = prev.Password
	}
	if strings.TrimSpace(next.ClientCertData) == "" && strings.TrimSpace(prev.ClientCertData) != "" {
		next.ClientCertData = prev.ClientCertData
	}
	if strings.TrimSpace(next.ClientKeyData) == "" && strings.TrimSpace(prev.ClientKeyData) != "" {
		next.ClientKeyData = prev.ClientKeyData
	}
}

// buildKubeconfigFromDirectConfig 从直连配置生成kubeconfig
func buildKubeconfigFromDirectConfig(config *DirectConfig) (string, error) {
	serverRaw := strings.TrimSpace(config.Server)
	if serverRaw == "" {
		return "", fmt.Errorf("API Server 地址不能为空")
	}
	// 兜底清洗：UI/复制粘贴可能混入空白字符，导致 token/base64 校验失败。
	token := compactNoSpace(config.Token)
	username := strings.TrimSpace(config.Username)
	password := strings.TrimSpace(config.Password)
	caRaw := compactNoSpace(config.CAData)
	certRaw := compactNoSpace(config.ClientCertData)
	keyRaw := compactNoSpace(config.ClientKeyData)
	// 解析服务器地址
	serverURL, err := url.Parse(serverRaw)
	if err != nil {
		return "", fmt.Errorf("无效的服务器地址: %w", err)
	}
	if serverURL.Scheme == "" || serverURL.Host == "" {
		return "", fmt.Errorf("无效的服务器地址: 需要完整 URL（如 https://10.0.0.1:6443）")
	}

	// 构建REST配置
	restConfig := &rest.Config{
		Host: serverRaw,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: config.InsecureSkipTLSVerify,
		},
	}

	// 设置CA证书。若已开启 insecure，client-go 不允许同时带 root CA。
	if !config.InsecureSkipTLSVerify && caRaw != "" {
		caData, err := base64.StdEncoding.DecodeString(caRaw)
		if err != nil {
			return "", fmt.Errorf("CA证书解码失败: %w", err)
		}
		restConfig.CAData = caData
	}

	// 设置认证方式
	if token != "" {
		slog.Info("k8s direct auth token debug",
			"token_len", len(token),
			"token_masked", maskSecretEdge(token, 8),
			"server", serverRaw,
		)
		restConfig.BearerToken = token
	} else if username != "" && password != "" {
		restConfig.Username = username
		restConfig.Password = password
	} else if certRaw != "" && keyRaw != "" {
		certData, err := base64.StdEncoding.DecodeString(certRaw)
		if err != nil {
			return "", fmt.Errorf("客户端证书解码失败: %w", err)
		}
		keyData, err := base64.StdEncoding.DecodeString(keyRaw)
		if err != nil {
			return "", fmt.Errorf("客户端密钥解码失败: %w", err)
		}
		restConfig.CertData = certData
		restConfig.KeyData = keyData
	}

	hasAuth := strings.TrimSpace(restConfig.BearerToken) != "" ||
		(strings.TrimSpace(restConfig.Username) != "" && strings.TrimSpace(restConfig.Password) != "") ||
		(len(restConfig.CertData) > 0 && len(restConfig.KeyData) > 0)
	if !hasAuth {
		return "", fmt.Errorf("直连未配置有效认证：请填写 Token、或用户名+密码、或客户端证书+私钥（部分集群匿名可读版本信息，但无法列举命名空间）")
	}

	// 将REST配置转换为kubeconfig格式
	kubeconfig := generateKubeconfigYAML(restConfig, serverURL.Hostname())
	return kubeconfig, nil
}

func compactNoSpace(s string) string {
	parts := strings.Fields(strings.TrimSpace(s))
	return strings.Join(parts, "")
}

func maskSecretEdge(s string, edge int) string {
	raw := strings.TrimSpace(s)
	if raw == "" {
		return ""
	}
	if edge <= 0 || len(raw) <= edge*2 {
		return fmt.Sprintf("***len=%d***", len(raw))
	}
	return raw[:edge] + "..." + raw[len(raw)-edge:]
}

// generateKubeconfigYAML 生成kubeconfig YAML格式
func generateKubeconfigYAML(config *rest.Config, clusterName string) string {
	var caData, certData, keyData string
	if len(config.CAData) > 0 {
		caData = base64.StdEncoding.EncodeToString(config.CAData)
	}
	if len(config.CertData) > 0 {
		certData = base64.StdEncoding.EncodeToString(config.CertData)
	}
	if len(config.KeyData) > 0 {
		keyData = base64.StdEncoding.EncodeToString(config.KeyData)
	}

	// 构建YAML
	yaml := fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: %s
`, config.Host)

	if caData != "" {
		yaml += fmt.Sprintf("    certificate-authority-data: %s\n", caData)
	}
	if config.TLSClientConfig.Insecure {
		yaml += "    insecure-skip-tls-verify: true\n"
	}

	yaml += fmt.Sprintf("  name: %s\n", clusterName)

	// Contexts
	yaml += fmt.Sprintf(`contexts:
- context:
    cluster: %s
    user: %s-user
  name: %s-context
current-context: %s-context
`, clusterName, clusterName, clusterName, clusterName)

	// Users
	yaml += fmt.Sprintf("users:\n- name: %s-user\n", clusterName)

	if config.BearerToken != "" {
		yaml += fmt.Sprintf("  user:\n    token: %s\n", config.BearerToken)
	} else if config.Username != "" {
		yaml += fmt.Sprintf("  user:\n    username: %s\n    password: %s\n", config.Username, config.Password)
	} else if certData != "" {
		yaml += fmt.Sprintf("  user:\n    client-certificate-data: %s\n    client-key-data: %s\n", certData, keyData)
	}

	return yaml
}
