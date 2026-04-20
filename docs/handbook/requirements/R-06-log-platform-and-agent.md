# 需求说明：日志平台与 Agent

## 1. 目标

在服务器上部署 **log-agent** 二进制，通过 **gRPC** 向平台上报日志批次；平台侧维护 Agent 注册、心跳、项目维度日志检索与导出。

## 2. 子功能

| 子功能 | 说明 |
|--------|------|
| Agent 注册 | `register-secret` 或长期 token |
| 心跳 / 状态 | 项目下批量刷新 |
| 运行时配置拉取 | `enable-runtime-pull` |
| 发现扫描 | `agent_discovery` 上报 |
| 日志 | HTTP 流式/导出，与 `log_sources` 绑定 |

## 3. 注意事项

- gRPC 地址 `grpc.target_addr` / `listen_addr` 需与 Agent `-grpc-server` 一致。
- Agent 默认**出站连接**，`listen-port=0` 表示本机不监听服务端口。
- 平台 HTTP `8080` 与 gRPC `18080` 防火墙需分别放行。

## 4. 相关文档

`docs/log-platform-api.md`、部署手册中的 Agent 章节。
