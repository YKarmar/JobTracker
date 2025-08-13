#!/bin/bash

# JobTracker 完整演示脚本

set -e

echo "🚀 === JobTracker 完整演示 ==="
echo ""

# 进入项目根目录
cd "$(dirname "$0")/.."

# 检查是否已构建
if [ ! -f "bin/mcp-server" ] || [ ! -f "bin/jobtracker" ]; then
    echo "⚠️  未找到构建文件，正在构建..."
    ./scripts/build-all.sh
    echo ""
fi

# 检查.env文件
if [ ! -f ".env" ]; then
    echo "⚠️  未找到.env文件，请先配置环境变量:"
    echo "   cp .env.example .env"
    echo "   编辑 .env 文件设置真实的邮箱和API密钥"
    echo ""
    echo "📧 支持的邮箱类型:"
    echo "   - Gmail (需要应用密码)"
    echo "   - Outlook/Hotmail"
    echo "   - Yahoo"
    echo "   - QQ/163/126等国内邮箱"
    echo ""
    exit 1
fi

echo "📧 MCP服务器支持的邮箱类型:"
echo "   ✅ Gmail (需要应用密码或OAuth)"
echo "   ✅ Outlook/Hotmail"
echo "   ✅ Yahoo"
echo "   ✅ QQ/163/126等国内邮箱"
echo ""

echo "🔧 启动演示模式..."
echo ""

# 设置演示用的环境变量
export USER_EMAIL="demo@gmail.com"
export DEEPSEEK_API_KEY="demo-key"

echo "方式1: 使用Mock模式 (推荐用于测试)"
echo "命令: ./bin/jobtracker --mock"
echo ""

echo "方式2: 使用真实MCP服务器"
echo "步骤:"
echo "  1. 启动MCP服务器: ./bin/mcp-server &"
echo "  2. 设置真实环境变量在.env文件中"
echo "  3. 运行客户端: ./bin/jobtracker"
echo ""

read -p "选择演示方式 [1=Mock模式, 2=真实MCP, q=退出]: " choice

case $choice in
    1)
        echo "🧪 启动Mock演示..."
        ./bin/jobtracker --mock
        ;;
    2)
        echo "🌐 启动MCP服务器..."
        ./bin/mcp-server &
        MCP_PID=$!
        
        echo "等待MCP服务器启动..."
        sleep 2
        
        echo "🎯 启动JobTracker客户端..."
        if ./bin/jobtracker; then
            echo "✅ 演示完成"
        else
            echo "❌ 客户端运行失败，请检查配置"
        fi
        
        echo "🛑 停止MCP服务器..."
        kill $MCP_PID 2>/dev/null || true
        ;;
    q|Q)
        echo "👋 退出演示"
        exit 0
        ;;
    *)
        echo "❌ 无效选择"
        exit 1
        ;;
esac

echo ""
echo "🎉 演示结束！" 