#!/bin/bash
# Yunshu 运维平台 - CentOS 部署脚本
# 适用于 2C2G50G 云服务器

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  Yunshu 运维平台 - CentOS 部署脚本${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# 检查系统
if [[ ! -f /etc/redhat-release ]] && [[ ! -f /etc/centos-release ]]; then
    echo -e "${YELLOW}警告：当前系统不是 CentOS/RHEL，脚本可能不完全适用${NC}"
fi

# 检查内存
echo -e "${YELLOW}[1/10] 检查系统资源...${NC}"
TOTAL_MEM=$(free -m | awk '/^Mem:/{print $2}')
echo "系统内存: ${TOTAL_MEM}MB"
if [ "$TOTAL_MEM" -lt 1800 ]; then
    echo -e "${YELLOW}警告：内存小于 2GB，建议升级配置${NC}"
fi

# 安装 Docker
echo -e "${YELLOW}[2/10] 检查并安装 Docker...${NC}"
if ! command -v docker &> /dev/null; then
    echo "正在安装 Docker..."
    sudo yum remove -y docker docker-client docker-client-latest docker-common docker-latest docker-latest-logrotate docker-logrotate docker-engine 2>/dev/null || true
    sudo yum install -y yum-utils
    sudo yum-config-manager --add-repo https://mirrors.aliyun.com/docker-ce/linux/centos/docker-ce.repo
    sudo yum install -y docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
    sudo systemctl start docker
    sudo systemctl enable docker
    echo -e "${GREEN}Docker 安装完成${NC}"
else
    echo -e "${GREEN}Docker 已安装${NC}"
    sudo systemctl start docker
fi

# 安装 docker-compose
echo -e "${YELLOW}[3/10] 检查并安装 docker-compose...${NC}"
if ! command -v docker-compose &> /dev/null; then
    echo "正在安装 docker-compose..."
    sudo curl -L "https://github.com/docker/compose/releases/download/v2.24.0/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    sudo chmod +x /usr/local/bin/docker-compose
    sudo ln -sf /usr/local/bin/docker-compose /usr/bin/docker-compose
    echo -e "${GREEN}docker-compose 安装完成${NC}"
else
    echo -e "${GREEN}docker-compose 已安装: $(docker-compose --version)${NC}"
fi

# 创建数据目录
echo -e "${YELLOW}[4/10] 创建数据目录...${NC}"
sudo mkdir -p /export/mysql_data /export/redis_data "$SCRIPT_DIR/logs"
sudo chown -R $(id -u):$(id -g) /export/mysql_data /export/redis_data "$SCRIPT_DIR/logs" 2>/dev/null || true
sudo chmod 777 /export/mysql_data /export/redis_data 2>/dev/null || true
echo -e "${GREEN}数据目录创建完成${NC}"

# 清理旧的 MySQL 数据（如果是重新部署）
echo -e "${YELLOW}[5/10] 检查是否需要清理旧数据...${NC}"
read -p "是否清理旧的 MySQL 数据？(首次部署请选 N，重新部署请选 y/N): " clean_data
if [[ $clean_data =~ ^[Yy]$ ]]; then
    echo "清理旧的 MySQL 数据..."
    sudo rm -rf /export/mysql_data/*
    echo -e "${GREEN}旧数据已清理${NC}"
fi

# 设置防火墙
echo -e "${YELLOW}[6/10] 配置防火墙...${NC}"
if command -v firewall-cmd &> /dev/null; then
    sudo firewall-cmd --permanent --add-port=80/tcp 2>/dev/null || true
    sudo firewall-cmd --permanent --add-port=8080/tcp 2>/dev/null || true
    sudo firewall-cmd --permanent --add-port=3306/tcp 2>/dev/null || true
    sudo firewall-cmd --permanent --add-port=6379/tcp 2>/dev/null || true
    sudo firewall-cmd --reload 2>/dev/null || true
    echo -e "${GREEN}防火墙端口已开放${NC}"
else
    echo -e "${YELLOW}未检测到 firewall-cmd，请手动开放端口 80, 8080${NC}"
fi

# 清理旧的容器
echo -e "${YELLOW}[7/10] 清理旧容器...${NC}"
docker-compose down 2>/dev/null || true
docker system prune -f --volumes=false 2>/dev/null || true
echo -e "${GREEN}旧容器已清理${NC}"

# 拉取镜像
echo -e "${YELLOW}[8/10] 拉取 Docker 镜像...${NC}"
docker-compose pull
echo -e "${GREEN}镜像拉取完成${NC}"

# 启动服务
echo -e "${YELLOW}[9/10] 启动服务...${NC}"
docker-compose up -d
echo -e "${GREEN}服务启动完成${NC}"

# 等待 MySQL 初始化
echo -e "${YELLOW}[10/10] 等待 MySQL 初始化 (约 30-60 秒)...${NC}"
echo "正在检查 MySQL 健康状态..."
for i in {1..30}; do
    if docker-compose ps mysql | grep -q "healthy"; then
        echo -e "${GREEN}MySQL 已就绪${NC}"
        break
    fi
    echo -n "."
    sleep 2
done

# 检查 SQL 导入状态
echo ""
echo -e "${YELLOW}检查数据库初始化状态...${NC}"
sleep 5
if docker-compose logs mysql | grep -i "initialized"; then
    echo -e "${GREEN}数据库初始化完成${NC}"
else
    echo -e "${YELLOW}正在检查 SQL 导入情况...${NC}"
    docker-compose logs --tail=20 mysql
fi

# 显示状态
echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  部署完成！服务状态：${NC}"
echo -e "${GREEN}========================================${NC}"
docker-compose ps

echo ""
echo -e "${GREEN}访问地址：${NC}"
echo "  - Web 界面: http://$(hostname -I | awk '{print $1}')"
echo "  - 后端 API: http://$(hostname -I | awk '{print $1}'):8080"
echo "  - Swagger 文档: http://$(hostname -I | awk '{print $1}'):8080/swagger"
echo ""
echo -e "${GREEN}常用命令：${NC}"
echo "  查看日志: docker-compose logs -f"
echo "  查看后端日志: docker-compose logs -f backend"
echo "  查看 MySQL 日志: docker-compose logs -f mysql"
echo "  重启服务: docker-compose restart"
echo "  停止服务: docker-compose down"
echo ""
echo -e "${YELLOW}注意：首次启动 SQL 文件导入可能需要 1-2 分钟，请耐心等待${NC}"
echo -e "${YELLOW}如果数据库表未创建，请查看 MySQL 日志: docker-compose logs mysql${NC}"
