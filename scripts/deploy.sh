#!/bin/bash

# 显示部署消息
echo "正在部署Go负载均衡器..."

# 默认配置文件路径
CONFIG_FILE=${1:-"configs/example.yaml"}

# 检查配置文件是否存在
if [ ! -f "$CONFIG_FILE" ]; then
    echo "错误: 配置文件 '$CONFIG_FILE' 不存在"
    echo "使用: $0 <配置文件路径>"
    exit 1
fi

# 检查可执行文件是否存在
if [ ! -f "lb" ]; then
    echo "错误: 未找到负载均衡器可执行文件 'lb'"
    echo "请先运行 build.sh 构建项目"
    exit 1
fi

# 确保有正确的权限
chmod +x lb

# 显示配置信息
echo "使用配置文件: $CONFIG_FILE"
echo "配置内容摘要:"
grep "listen_addr\|algorithm" "$CONFIG_FILE"
echo "后端服务器:"
grep -A 2 "- url:" "$CONFIG_FILE"

# 询问是否继续
read -p "是否继续部署? (y/n): " confirm
if [ "$confirm" != "y" ]; then
    echo "部署已取消"
    exit 0
fi

# 停止先前运行的实例
echo "检查是否有正在运行的实例..."
PID=$(pgrep -f "./lb -config")
if [ ! -z "$PID" ]; then
    echo "正在停止实例 (PID: $PID)..."
    kill -15 $PID
    sleep 2
    
    # 检查是否成功停止
    if ps -p $PID > /dev/null; then
        echo "强制终止进程..."
        kill -9 $PID
    fi
fi

# 后台启动服务
echo "启动负载均衡器..."
nohup ./lb -config "$CONFIG_FILE" > lb.log 2>&1 &
NEW_PID=$!

# 检查是否成功启动
sleep 2
if ps -p $NEW_PID > /dev/null; then
    echo "负载均衡器已成功启动 (PID: $NEW_PID)"
    echo "日志输出重定向到 lb.log"
else
    echo "启动失败，请检查 lb.log 获取更多信息"
    exit 1
fi

echo "部署完成" 