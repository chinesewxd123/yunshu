#!/bin/bash
# SQL 导入诊断和修复脚本

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  SQL 导入诊断工具${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

cd "$(dirname "$0")"

# 检查容器状态
echo -e "${YELLOW}[1/5] 检查 MySQL 容器状态...${NC}"
if ! docker ps | grep -q yunshu-mysql; then
    echo -e "${RED}MySQL 容器未运行，请先启动服务${NC}"
    echo "运行: docker-compose up -d mysql"
    exit 1
fi
echo -e "${GREEN}MySQL 容器运行中${NC}"

# 检查初始化日志
echo -e "${YELLOW}[2/5] 检查 MySQL 初始化日志...${NC}"
echo "最近的日志:"
docker-compose logs --tail=30 mysql | grep -E "(Initializing|entrypoint|Entrypoint|database|DATABASE|ready|/docker-entrypoint)" || echo "未找到相关日志"

# 检查挂载的 SQL 文件
echo -e "${YELLOW}[3/5] 检查容器内的 SQL 文件...${NC}"
docker exec yunshu-mysql ls -la /docker-entrypoint-initdb.d/ 2>/dev/null || echo -e "${RED}无法访问 /docker-entrypoint-initdb.d/${NC}"

# 检查数据库和表
echo -e "${YELLOW}[4/5] 检查数据库状态...${NC}"
docker exec yunshu-mysql mysql -uroot -p123456 -e "SHOW DATABASES;" 2>/dev/null || echo -e "${RED}无法连接 MySQL（使用默认密码）${NC}"

# 检查表数量
echo -e "${YELLOW}[5/5] 检查 yunshu 数据库表数量...${NC}"
TABLE_COUNT=$(docker exec yunshu-mysql mysql -uroot -p123456 -e "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='yunshu';" 2>/dev/null | tail -1)
if [ -n "$TABLE_COUNT" ] && [ "$TABLE_COUNT" -gt 0 ]; then
    echo -e "${GREEN}yunshu 数据库有 $TABLE_COUNT 个表${NC}"
    echo -e "${GREEN}SQL 导入成功！${NC}"
else
    echo -e "${RED}yunshu 数据库没有表或不存在${NC}"
    echo ""
    echo -e "${YELLOW}可能的原因：${NC}"
    echo "1. MySQL 数据目录 (/export/mysql_data) 已有数据，跳过初始化"
    echo "2. SQL 文件命名或路径问题"
    echo "3. SQL 文件执行出错"
    echo ""
    echo -e "${YELLOW}修复方案：${NC}"
    echo "A) 如果数据重要，请手动导入 SQL:"
    echo "   docker exec -i yunshu-mysql mysql -uroot -p123456 yunshu < configs/dump-permission_system-202604231146.sql"
    echo ""
    echo "B) 如果不需要旧数据，清空后重新初始化："
    echo "   docker-compose down"
    echo "   sudo rm -rf /export/mysql_data/*"
    echo "   docker-compose up -d mysql"
    echo "   sleep 30"
    echo "   docker-compose logs mysql | tail -50"
fi

echo ""
echo -e "${GREEN}诊断完成${NC}"
