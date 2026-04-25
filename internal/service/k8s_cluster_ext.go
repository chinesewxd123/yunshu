package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"

	"yunshu/internal/repository"

	"k8s.io/client-go/rest"
)

// getDirectConfigFromDict 从数据字典读取直连配置
func getDirectConfigFromDict(ctx context.Context, dictRepo *repository.DictEntryRepository, configKey string) (*DirectConfig, error) {
	entry, err := dictRepo.GetByDictTypeAndValue(ctx, "k8s_direct_config", configKey)
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

// buildKubeconfigFromDirectConfig 从直连配置生成kubeconfig
func buildKubeconfigFromDirectConfig(config *DirectConfig) (string, error) {
	// 解析服务器地址
	serverURL, err := url.Parse(config.Server)
	if err != nil {
		return "", fmt.Errorf("无效的服务器地址: %w", err)
	}

	// 构建REST配置
	restConfig := &rest.Config{
		Host: serverURL.String(),
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: config.InsecureSkipTLSVerify,
		},
	}

	// 设置CA证书
	if config.CAData != "" {
		caData, err := base64.StdEncoding.DecodeString(config.CAData)
		if err != nil {
			return "", fmt.Errorf("CA证书解码失败: %w", err)
		}
		restConfig.CAData = caData
	}

	// 设置认证方式
	if config.Token != "" {
		restConfig.BearerToken = config.Token
	} else if config.Username != "" && config.Password != "" {
		restConfig.Username = config.Username
		restConfig.Password = config.Password
	} else if config.ClientCertData != "" && config.ClientKeyData != "" {
		certData, err := base64.StdEncoding.DecodeString(config.ClientCertData)
		if err != nil {
			return "", fmt.Errorf("客户端证书解码失败: %w", err)
		}
		keyData, err := base64.StdEncoding.DecodeString(config.ClientKeyData)
		if err != nil {
			return "", fmt.Errorf("客户端密钥解码失败: %w", err)
		}
		restConfig.CertData = certData
		restConfig.KeyData = keyData
	}

	// 将REST配置转换为kubeconfig格式
	kubeconfig := generateKubeconfigYAML(restConfig, serverURL.Hostname())
	return kubeconfig, nil
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
