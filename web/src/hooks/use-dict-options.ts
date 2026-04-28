import { useEffect, useMemo, useState } from "react";
import { getDictOptions } from "../services/dict";

type Option = { label: string; value: string | number };

function shouldCastDictValueToNumber(dictType: string, value: string): boolean {
  const rawType = String(dictType || "").trim();
  const rawValue = String(value || "").trim();
  if (!rawValue || !/^-?\d+$/.test(rawValue)) {
    return false;
  }
  if (rawType === "common_status") {
    return true;
  }
  return rawType === "server_port" || rawType.endsWith("_port");
}

const fallbackMap: Record<string, Option[]> = {
  common_status: [
    { label: "启用", value: 1 },
    { label: "停用", value: 0 },
  ],
  alert_channel_type: [
    { label: "通用 Webhook", value: "generic_webhook" },
    { label: "企业微信", value: "wechat_work" },
    { label: "钉钉", value: "dingding" },
    { label: "邮件", value: "email" },
  ],
  alert_severity: [
    { label: "信息", value: "info" },
    { label: "警告", value: "warning" },
    { label: "严重", value: "critical" },
  ],
  /** 告警监控平台 · 数据源表单「从字典填入」；实际条目以数据字典为准 */
  alert_datasource_base_url: [],
  alert_datasource_basic_user: [],
  alert_rule_category: [
    { label: "主机", value: "host" },
    { label: "Kubernetes", value: "k8s" },
    { label: "API", value: "api" },
  ],
  /** 企业微信/钉钉通知模式：必须以数据字典为准，不在此写死选项（避免与字典不一致时仍显示旧文案） */
  wecom_notify_mode: [],
  dingtalk_notify_mode: [],
  log_source_type: [
    { label: "文件", value: "file" },
    { label: "systemd journal", value: "journal" },
  ],
  server_group_category: [
    { label: "自建服务器", value: "self_hosted" },
    { label: "云服务器", value: "cloud" },
  ],
  server_os_type: [
    { label: "Linux", value: "linux" },
    { label: "Windows", value: "windows" },
  ],
  server_auth_type: [
    { label: "密码", value: "password" },
    { label: "私钥", value: "key" },
  ],
  server_port: [
    { label: "SSH 默认端口 22", value: 22 },
    { label: "RDP 默认端口 3389", value: 3389 },
  ],
  log_agent_health_status: [
    { label: "运行中", value: "running" },
    { label: "启动中", value: "starting" },
    { label: "已停止", value: "stopped" },
    { label: "错误", value: "error" },
    { label: "未知", value: "unknown" },
  ],
  /** 监控规则 PromQL 生成器中的标签键候选；条目以数据字典为准 */
  alert_promql_label_key: [
    { label: "instance", value: "instance" },
    { label: "job", value: "job" },
    { label: "cluster", value: "cluster" },
    { label: "namespace", value: "namespace" },
    { label: "pod", value: "pod" },
    { label: "service", value: "service" },
    { label: "node", value: "node" },
    { label: "severity", value: "severity" },
    { label: "alertname", value: "alertname" },
    { label: "path", value: "path" },
    { label: "device", value: "device" },
    { label: "fstype", value: "fstype" },
    { label: "mountpoint", value: "mountpoint" },
  ],
  /** 阈值单位字典 */
  alert_threshold_unit: [
    { label: "原始值", value: "raw" },
    { label: "百分比 (%)", value: "percent" },
    { label: "字节 (bytes)", value: "bytes" },
    { label: "毫秒 (ms)", value: "ms" },
    { label: "计数 (count)", value: "count" },
  ],
  /** 监控规则告警文案预设（value 建议存 JSON：{"summary":"...","description":"..."}） */
  alert_rule_template_preset: [],
  alert_webhook_url: [],
  wecom_webhook_url: [],
  dingtalk_webhook_url: [],
  agent_platform_url: [],
  wecom_corp_id: [],
  wecom_corp_secret: [],
  wecom_agent_id: [],
  dingtalk_app_key: [],
  dingtalk_app_secret: [],
  dingtalk_chat_id: [],
  dingtalk_sign_secret: [],
  /** 集群 kubeconfig 模板；条目以数据字典为准（值可为多行 yaml） */
  k8s_kubeconfig_template: [],
  /** 服务器 SSH 用户名模板（值填入「用户名」） */
  server_ssh_username: [],
  /** 服务器 SSH 密码模板（值填入「密码」） */
  server_ssh_password: [],
  cloud_alibaba_ak: [],
  cloud_alibaba_sk: [],
  cloud_tencent_ak: [],
  cloud_tencent_sk: [],
  cloud_jd_ak: [],
  cloud_jd_sk: [],
  server_cloud_alibaba_username: [],
  server_cloud_alibaba_password: [],
  server_cloud_alibaba_private_key: [],
  server_cloud_alibaba_port: [],
  server_cloud_tencent_username: [],
  server_cloud_tencent_password: [],
  server_cloud_tencent_private_key: [],
  server_cloud_tencent_port: [],
  server_cloud_jd_username: [],
  server_cloud_jd_password: [],
  server_cloud_jd_private_key: [],
  server_cloud_jd_port: [],
};

export function useDictOptions(dictType: string) {
  const [options, setOptions] = useState<Option[]>(fallbackMap[dictType] ?? []);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        const list = await getDictOptions(dictType);
        if (cancelled) return;
        const next = list.map((item) => {
          if (shouldCastDictValueToNumber(dictType, item.value)) {
            return { label: item.label, value: Number(item.value) };
          }
          return { label: item.label, value: item.value };
        });
        setOptions(next.length ? next : fallbackMap[dictType] ?? []);
      } catch {
        if (!cancelled) {
          setOptions(fallbackMap[dictType] ?? []);
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [dictType]);

  return useMemo(() => options, [options]);
}
