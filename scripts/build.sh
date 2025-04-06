#!/bin/bash

# 显示构建消息
echo "正在构建Go负载均衡器..."

# 设置版本信息
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
COMMIT_HASH=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# 设置输出文件名
OUTPUT_NAME=${OUTPUT_NAME:-"lb"}

# 设置GOPATH(如果未设置)
if [ -z "$GOPATH" ]; then
    GOPATH="$HOME/go"
    echo "GOPATH未设置，使用默认值: $GOPATH"
fi

# 获取依赖
echo "获取依赖..."
go mod tidy

# 检查静态分析问题
echo "运行代码检查..."
go vet ./...
if [ $? -ne 0 ]; then
    echo "警告: 代码检查发现问题，请检查上述警告"
    # 不阻止构建，只提示警告
fi

# 运行单元测试
echo "运行单元测试..."
go test -v ./... 

# 构建适用于当前平台的执行文件
echo "为当前平台构建..."
go build -o ${OUTPUT_NAME} -ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.CommitHash=${COMMIT_HASH}" cmd/lb/main.go

if [ $? -eq 0 ]; then
    echo "构建成功! 输出文件: ${OUTPUT_NAME}"
    echo "版本信息:"
    echo "  版本: ${VERSION}"
    echo "  构建时间: ${BUILD_TIME}"
    echo "  提交哈希: ${COMMIT_HASH}"
else
    echo "构建失败，请检查错误"
    exit 1
fi

# 可选的交叉编译
if [ "$1" = "--cross" ]; then
    echo "执行交叉编译..."
    
    # Linux (amd64)
    GOOS=linux GOARCH=amd64 go build -o ${OUTPUT_NAME}-linux-amd64 -ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.CommitHash=${COMMIT_HASH}" cmd/lb/main.go
    
    # Windows (amd64)
    GOOS=windows GOARCH=amd64 go build -o ${OUTPUT_NAME}-windows-amd64.exe -ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.CommitHash=${COMMIT_HASH}" cmd/lb/main.go
    
    # MacOS (amd64)
    GOOS=darwin GOARCH=amd64 go build -o ${OUTPUT_NAME}-darwin-amd64 -ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.CommitHash=${COMMIT_HASH}" cmd/lb/main.go
    
    echo "交叉编译完成"
fi

echo "构建过程结束" 