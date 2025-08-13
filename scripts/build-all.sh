#!/bin/bash

# JobTracker 完整构建脚本

set -e

echo "=== JobTracker 完整构建脚本 ==="

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

# 创建构建目录
mkdir -p bin

# 构建MCP服务器
echo "构建MCP服务器..."
go build -ldflags="-s -w" -o bin/mcp-server ./cmd/mcp-server

# 构建JobTracker客户端
echo "构建JobTracker客户端..."
go build -ldflags="-s -w" -o bin/jobtracker ./cmd/jobtracker

echo "✅ 构建完成！"
echo ""
echo "可执行文件:"
echo "  - MCP服务器: bin/mcp-server"
echo "  - JobTracker: bin/jobtracker"
echo ""
echo "使用方法:"
echo "  1. 启动MCP服务器: ./bin/mcp-server"
echo "  2. 配置环境变量: 编辑 .env 文件"
echo "  3. 运行JobTracker: ./bin/jobtracker"
echo ""
echo "测试模式:"
echo "  ./bin/jobtracker --mock (无需MCP服务器)" 