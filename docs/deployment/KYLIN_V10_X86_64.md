# 部署手册：银河麒麟 Kylin Linux Advanced Server V10（x86_64）

适用于 **Kylin V10 Sword** 等与 RHEL 系兼容的 x86_64 服务器。以下路径、用户请按实际环境替换（示例用户 `deploy`，目录 `/opt/yunshu-cmdb`）。

## 1. 环境依赖

| 组件 | 建议版本 | 说明 |
|------|----------|------|
| Go | 1.23+（与 `go.mod` 一致） | 构建后端与 Agent |
| Node.js | 18+ LTS | 构建前端 |
| MySQL | 8.0+ | 元数据与 Casbin |
| Redis | 6+ | 会话/限流/告警去重等 |
| Nginx | 1.20+ | 静态资源与反向代理 |

安装示例（麒麟可用 `yum`/`dnf`，以实际源为准）：

```bash
sudo yum install -y git gcc nginx mysql redis
# Node：建议使用 nvm 或官方二进制安装
# Go：从 https://go.dev/dl/ 下载 linux-amd64 包并配置 GOROOT/GOPATH
```

## 2. 获取源码

```bash
sudo mkdir -p /opt/yunshu-cmdb && sudo chown "$USER":"$USER" /opt/yunshu-cmdb
cd /opt/yunshu-cmdb
git clone <your-repo-url> src
cd src
```

## 3. 配置文件

复制并编辑 `configs/config.yaml`：

- `mysql.*`：主机、库名、账号密码  
- `redis.*`  
- `auth.jwt_secret`：**生产必须更换**  
- `security.encryption_key`：**32 字节强度密钥**  
- `agent.register_secret`：Agent 注册密钥  
- `app.port`：HTTP 监听，默认 `8080`  
- `grpc.listen_addr` / `grpc.target_addr`：默认 `0.0.0.0:18080`  

## 4. 数据库迁移与种子

```bash
cd /opt/yunshu-cmdb/src
export CONFIG_PATH="$PWD/configs/config.yaml"   # 若使用 --config 可省略
go run . migrate
go run . seed
```

- `migrate`：执行 GORM `AutoMigrate` 及附带清理逻辑。  
- `seed`：默认管理员、权限元数据、Casbin 策略、菜单等（详见 `cmd/seed.go`）。

## 5. 后端构建与启动

### 5.1 源码直接启动（开发/调试）

```bash
cd /opt/yunshu-cmdb/src
go run . server --config configs/config.yaml
```

### 5.2 二进制构建与运行

```bash
cd /opt/yunshu-cmdb/src
go build -o bin/permission-system -trimpath -ldflags "-s -w" .
./bin/permission-system server --config configs/config.yaml
```

**端口**：HTTP `app.port`（8080）；gRPC `grpc.listen_addr`（18080）。防火墙需放行：

```bash
sudo firewall-cmd --permanent --add-port=8080/tcp
sudo firewall-cmd --permanent --add-port=18080/tcp
sudo firewall-cmd --reload
```

### 5.3 systemd（可选）

`/etc/systemd/system/yunshu-cmdb.service`：

```ini
[Unit]
Description=YunShu CMDB API
After=network.target mysql.service redis.service

[Service]
Type=simple
User=deploy
WorkingDirectory=/opt/yunshu-cmdb/src
ExecStart=/opt/yunshu-cmdb/src/bin/permission-system server --config /opt/yunshu-cmdb/src/configs/config.yaml
Restart=on-failure
Environment=GIN_MODE=release

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now yunshu-cmdb
```

## 6. 前端构建

```bash
cd /opt/yunshu-cmdb/src/web
npm ci
npm run build
```

产物目录：`web/dist/`。构建时若需指定后端 API 基地址，按项目惯例配置环境变量或 `vite` 配置（见 `web/vite.config.ts` / `.env.production`）。

## 7. Nginx：静态站点 + 反向代理 API

示例：对外 `443`/`80`，静态文件指向 `web/dist`，`/api/` 与 `/swagger` 等到后端 `127.0.0.1:8080`。

```nginx
server {
    listen 80;
    server_name cmdb.example.com;
    root /opt/yunshu-cmdb/src/web/dist;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /swagger {
        proxy_pass http://127.0.0.1:8080;
    }
}
```

WebSocket（如 Pod 终端、服务器终端）需额外：

```nginx
proxy_set_header Upgrade $http_upgrade;
proxy_set_header Connection "upgrade";
```

reload：

```bash
sudo nginx -t && sudo systemctl reload nginx
```

## 8. Log Agent 二进制与启动

在**日志所在节点**构建：

```bash
cd /opt/yunshu-cmdb/src
go build -o bin/log-agent -trimpath ./cmd/logagent
```

示例（令牌模式，参数以实际项目/服务器/日志源为准）：

```bash
./bin/log-agent \
  -grpc-server 10.0.0.1:18080 \
  -platform-url http://10.0.0.1:8080 \
  -project-id 1 \
  -server-id 2 \
  -log-source-id 3 \
  -token "<平台下发的长期 token>" \
  -source-type file \
  -path /var/log/app.log
```

**公网注册模式**（无 token 时，需 `-register-secret` 与平台 `agent.register_secret` 一致）：

```bash
./bin/log-agent \
  -grpc-server 10.0.0.1:18080 \
  -register-secret "<与 configs/config.yaml 中一致>" \
  -project-id 1 \
  ...
```

更多参数见 `cmd/logagent/main.go` 中 `flag` 定义。

## 9. 健康检查

- HTTP：`GET http://127.0.0.1:8080/api/v1/health`（以路由为准）  
- 前端：浏览器访问 Nginx 站点，登录后检查菜单与 API。

## 10. 常见问题

| 现象 | 排查 |
|------|------|
| 403 无访问权限 | 非 super-admin 需在「授权管理」勾选对应 API；或检查 Casbin 表是否随 seed 更新 |
| Agent 连不上 gRPC | 检查 `18080` 防火墙、TLS/地址是否与 `-grpc-server` 一致 |
| 前端空白 | `try_files` 是否指向 SPA；API 是否跨域（同域 Nginx 代理可避免 CORS） |

---

*与源码版本同步；若命令行参数变更以 `cmd/*.go` 为准。*
