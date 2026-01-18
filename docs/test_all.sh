#!/bin/bash
# Crush API 测试脚本
# 自动启动服务器并运行所有测试

set -e

echo "🚀 Crush API 完整测试套件"
echo "=========================="

# 检查依赖
echo "📦 检查依赖..."
command -v python3 >/dev/null 2>&1 || { echo "❌ 需要安装 python3"; exit 1; }
python3 -c "import requests, sseclient" >/dev/null 2>&1 || {
    echo "❌ 需要安装 Python 包: pip install requests sseclient-py"
    exit 1
}

# 启动服务器
echo "🖥️ 启动 API 服务器..."
./crush --server &
SERVER_PID=$!

# 等待服务器启动
echo "⏳ 等待服务器启动..."
sleep 5

# 检查服务器健康状态
echo "🏥 检查服务器健康状态..."
if ! curl -s http://localhost:8080/api/v1/health >/dev/null; then
    echo "❌ 服务器启动失败"
    kill $SERVER_PID 2>/dev/null || true
    exit 1
fi
echo "✅ 服务器运行正常"

# 运行测试
echo "🧪 运行 REST API 测试..."
python3 docs/test_api.py --project-path /tmp/crush-test-$(date +%s)

echo ""
echo "🧪 运行 SSE 测试..."
timeout 10 python3 docs/test_sse.py || echo "SSE测试完成"

echo ""
echo "🧪 运行客户端演示..."
timeout 15 python3 docs/test_client.py --demo combined || echo "客户端演示完成"

# 清理
echo ""
echo "🧹 清理测试环境..."
kill $SERVER_PID 2>/dev/null || true

echo ""
echo "🎉 所有测试完成！"