# Yunshu 运维平台 - 部署配置对比分析
## 适用于 2C2G50G CentOS 云服务器

---

## 一、关键发现：配置冲突问题

### ❌ 发现的环境变量映射问题

| 环境变量 (docker-compose) | 目标配置 (config.yaml) | 映射结果 | 状态 |
|---------------------------|------------------------|----------|------|
| `MYSQL_HOST` | `mysql.host` | `mysql.host` | ✅ 正确 |
| `MYSQL_PORT` | `mysql.port` | `mysql.port` | ✅ 正确 |
| `MYSQL_USER` | `mysql.user` | `mysql.user` | ✅ 正确 |
| `MYSQL_PASSWORD` | `mysql.password` | `mysql.password` | ✅ 正确 |
| `MYSQL_DB_NAME` | `mysql.db_name` | `mysql.db_name` | ✅ 正确 |
| `REDIS_ADDR` | `redis.addr` | `redis.addr` | ✅ 正确 |
| `REDIS_PASSWORD` | `redis.password` | `redis.password` | ✅ 正确 |
| `JWT_SECRET` | `auth.jwt_secret` | `jwt_secret` ❌ | **❌ 不匹配** |
| `ENCRYPTION_KEY` | `security.encryption_key` | `encryption_key` ❌ | **❌ 不匹配** |

### 问题解释

Viper 的 `AutomaticEnv()` 将环境变量转为小写映射：
- `JWT_SECRET` → 寻找 `jwt_secret` 配置项
- 但实际配置项是嵌套的 `auth.jwt_secret`
- 因此 **JWT_SECRET 环境变量无法覆盖配置文件中的值**！

---

## 二、配置文件当前值（configs/config.yaml）

```yaml
# MySQL - 指向公网IP（Docker中需要覆盖为 yunshu-mysql）
mysql:
  host: 175.178.156.23        # 生产环境公网IP，Docker内需用 yunshu-mysql
  port: 3306
  user: root
  password: "123456"          # ⚠️ 弱密码，生产环境必须修改
  db_name: yunshu

# Redis - 指向公网IP（Docker中需要覆盖为 yunshu-redis）
redis:
  addr: 175.178.156.23:6379   # 生产环境公网IP，Docker内需用 yunshu-redis:6379
  password: "1234"            # ⚠️ 弱密码，生产环境必须修改
  db: 0

# 认证 - JWT密钥（环境变量无法覆盖！）
auth:
  jwt_secret: change-me-in-production  # ⚠️ 必须在 config.yaml 中修改

# 安全 - 加密密钥（环境变量无法覆盖！）
security:
  encryption_key: "change-me-32bytes-base64"  # ⚠️ 必须在 config.yaml 中修改
```

---

## 三、两种部署模式选择

### 模式A：Docker 部署（推荐用于云服务器）

所有服务容器化，**必须确保容器内连接使用服务名而非公网IP**。

#### 修改要求：
1. **修改 configs/config.yaml**：
   ```yaml
   mysql:
     host: yunshu-mysql        # 改为服务名
     password: "StrongPassword!"  # 强密码
   
   redis:
     addr: yunshu-redis:6379   # 改为服务名
     password: "StrongRedisPass!" # 强密码
   
   auth:
     jwt_secret: "YourStrongJWTSecretKey2026MustBe32BytesLong"  # 强密钥
   
   security:
     encryption_key: "your-32-byte-base64-key-here-please"  # 32字节base64
   ```

2. **同步修改 docker-compose.yml 中的密码环境变量**，保持一致

#### 优缺点：
- ✅ 无需处理复杂的环境变量映射
- ✅ 配置集中管理
- ❌ 密码明文存储在 config.yaml

---

### 模式B：原生部署（非 Docker）

直接在 CentOS 上运行编译后的二进制文件。

#### 部署要求：
1. 安装 MySQL 5.7、Redis 7.x
2. 导入 SQL 文件
3. 修改 config.yaml 使用 `127.0.0.1` 或公网IP
4. 编译并运行 `./yunshu server`

#### 优缺点：
- ✅ 资源占用更少（无 Docker 开销）
- ❌ 需要手动管理服务
- ❌ 部署更复杂

---

## 四、资源占用评估（2C2G 服务器）

### Docker 模式资源分配

| 服务 | CPU | 内存 | 说明 |
|------|-----|------|------|
| MySQL | 0.6 | 400m | 生产级配置，可处理中等负载 |
| Redis | 0.3 | 128m | 关闭持久化，纯缓存 |
| 后端 | 0.6 | 384m | Go服务内存优化 |
| 前端(Nginx) | 0.2 | 64m | 静态文件服务 |
| **预留** | 0.3 | ~200m | 系统 + Docker 开销 |
| **总计** | 2.0 | ~1.2G | 接近上限，建议关闭非必要服务 |

### 内存优化建议
1. **MySQL**: 已调整 `innodb_buffer_pool_size=128M`（适合小内存）
2. **Redis**: 关闭 RDB/AOF 持久化，纯内存缓存
3. **后端**: 512m 限制，Go 运行时 GC 会自适应
4. **系统**: CentOS 最小化安装，关闭不必要的服务

---

## 五、安全加固清单

### 必须修改的默认凭证

| 凭证 | 当前值 | 风险 | 修改位置 |
|------|--------|------|----------|
| MySQL root | `123456` | 极易被破解 | configs/config.yaml + docker-compose.yml |
| Redis | `1234` | 极易被破解 | configs/config.yaml + docker-compose.yml |
| JWT Secret | `change-me-in-production` | 可伪造token | configs/config.yaml |
| Encryption Key | `change-me-32bytes-base64` | 敏感数据可解密 | configs/config.yaml |
| Agent Secret | `change-me-agent-register-secret` | Agent可任意注册 | configs/config.yaml |

### 网络暴露检查

| 端口 | 服务 | 对外暴露 | 建议 |
|------|------|----------|------|
| 80 | Nginx(前端) | ✅ | 保留 |
| 8080 | Go后端 | ✅ | **建议限制IP或使用反向代理** |
| 3306 | MySQL | ✅ | ⚠️ **强烈建议关闭或限制IP** |
| 6379 | Redis | ✅ | ⚠️ **强烈建议关闭或限制IP** |
| 18080 | gRPC | ✅ | **建议限制IP** |

#### 安全加固操作
```bash
# 1. 使用防火墙限制端口访问
sudo firewall-cmd --permanent --remove-port=3306/tcp
sudo firewall-cmd --permanent --remove-port=6379/tcp
sudo firewall-cmd --permanent --remove-port=8080/tcp
sudo firewall-cmd --permanent --remove-port=18080/tcp
# 仅保留 80 端口对外
sudo firewall-cmd --reload

# 2. 如需远程管理，使用 SSH 隧道或 VPN
# 3. 或限制特定IP访问
sudo firewall-cmd --permanent --add-rich-rule='rule family="ipv4" source address="你的IP/32" port protocol="tcp" port="3306" accept'
```

---

## 六、SQL 自动导入机制

### 工作原理
MySQL Docker 镜像首次启动时（数据目录为空），会执行 `/docker-entrypoint-initdb.d/` 目录下的 `.sql`、`.sql.gz`、`.sh` 文件。

### 失败常见原因
1. **数据目录非空** - 已有 `/export/mysql_data` 数据，跳过初始化
2. **SQL 文件过大** - 导入超时
3. **权限问题** - 挂载的 SQL 文件无法读取

### 验证导入成功的方法
```bash
# 进入 MySQL 容器
docker exec -it yunshu-mysql mysql -uroot -p123456

# 查询表数量
SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='yunshu';
# 预期结果：20+ 张表
```

---

## 七、部署步骤（推荐流程）

### 第一步：准备服务器
```bash
# CentOS 系统更新
sudo yum update -y

# 安装基础工具
sudo yum install -y vim wget curl

# 创建数据目录
sudo mkdir -p /export/mysql_data /export/redis_data
sudo chmod 777 /export/mysql_data /export/redis_data
```

### 第二步：修改配置文件

**必须修改 configs/config.yaml**：
```yaml
# 数据库连接改为容器名
mysql:
  host: yunshu-mysql
  password: "YourStrongPassword123!"

redis:
  addr: yunshu-redis:6379
  password: "YourRedisPass123!"

auth:
  jwt_secret: "YourRandomJWTSecretKey32Bytes2026!"

security:
  encryption_key: "your-32-byte-base64-encoded-key-here"

agent:
  register_secret: "YourRandomAgentSecret2026"
```

**同步修改 docker-compose.yml**：
```yaml
environment:
  MYSQL_ROOT_PASSWORD: "YourStrongPassword123!"
  MYSQL_PASSWORD: "YourStrongPassword123!"
  REDIS_PASSWORD: "YourRedisPass123!"
```

### 第三步：执行部署
```bash
chmod +x deploy.sh
./deploy.sh
```

### 第四步：验证部署
```bash
# 检查所有服务状态
docker-compose ps

# 检查 MySQL 表是否导入
./check-sql.sh

# 检查后端日志
docker-compose logs -f backend
```

---

## 八、故障排查速查表

| 症状 | 可能原因 | 解决方案 |
|------|----------|----------|
| MySQL 启动后自动退出 | 数据目录权限问题 | `sudo chmod 777 /export/mysql_data` |
| SQL 未自动导入 | 数据目录已有数据 | `sudo rm -rf /export/mysql_data/*` 后重启 |
| 后端无法连接 MySQL | host 配置为公网IP | 改为 `yunshu-mysql` |
| 后端无法连接 Redis | addr 配置为公网IP | 改为 `yunshu-redis:6379` |
| 登录提示 token 错误 | JWT 密钥不匹配 | 检查 config.yaml 中的 jwt_secret |
| 内存不足 OOM | 2G 内存不够 | 增加 swap 或升级配置 |

---

## 九、总结

### 核心要点
1. **JWT_SECRET 和 ENCRYPTION_KEY 无法通过环境变量覆盖**，必须在 `configs/config.yaml` 中修改
2. **Docker 部署时**，MySQL/Redis 的 host 必须改为服务名（`yunshu-mysql`、`yunshu-redis`）
3. **所有默认密码必须修改**，特别是 MySQL、Redis、JWT、加密密钥
4. **端口安全**：3306/6379/8080 不应直接暴露公网
5. **SQL 自动导入**：只在首次启动且数据目录为空时执行

### 推荐的最小化安全配置
- MySQL: 仅监听容器内，不暴露 3306 到宿主机
- Redis: 仅监听容器内，不暴露 6379 到宿主机
- 后端: 通过 Nginx 反向代理，不直接暴露 8080
