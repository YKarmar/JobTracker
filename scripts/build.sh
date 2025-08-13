#!/bin/bash

# JobTracker 构建脚本

set -e

echo "=== JobTracker 构建脚本 ==="

# 检查Go版本
echo "检查Go环境..."
if ! command -v go &> /dev/null; then
    echo "错误: 未找到Go环境，请先安装Go 1.22+"
    exit 1
fi

GO_VERSION=$(go version | cut -d' ' -f3 | sed 's/go//')
echo "Go版本: $GO_VERSION"

# 进入项目根目录
cd "$(dirname "$0")/.."

# 整理依赖
echo "整理依赖..."
go mod tidy

# 运行测试（如果有）
if [ -d "tests" ]; then
    echo "运行测试..."
    go test ./...
fi

# 构建项目
echo "构建项目..."
go build -ldflags="-s -w" -o bin/jobtracker ./cmd/jobtracker

# 创建输出目录
mkdir -p bin

echo "✅ 构建完成！"
echo "可执行文件位置: bin/jobtracker"
echo ""
echo "使用方法:"
echo "  1. 配置环境变量: cp .env.example .env && 编辑 .env"
echo "  2. 配置参数: 编辑 configs/config.yaml"
echo "  3. 运行程序: ./bin/jobtracker" 