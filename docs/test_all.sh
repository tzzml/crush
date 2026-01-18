#!/bin/bash
# Crush API 测试脚本

set -e

echo "🚀 Crush API 测试套件"

# 检查依赖
command -v python3 >/dev/null 2>&1 || { echo "❌ 需要安装 python3"; exit 1; }
python3 -c "import requests, sseclient" >/dev/null 2>&1 || {
    echo "❌ 需要安装: pip install requests sseclient-py"
    exit 1
}

# 启动服务器
echo "🖥️  启动服务器..."
./crush --server &
SERVER_PID=$!

# 等待服务器启动
sleep 3

# 检查服务器
if ! curl -s http://localhost:8080/api/v1/health >/dev/null; then
    echo "❌ 服务器启动失败"
    kill $SERVER_PID 2>/dev/null || true
    exit 1
fi

echo "✅ 服务器运行正常"

# 运行测试
echo "🧪 运行 API 测试..."
python3 docs/test_api.py --project-path /tmp/crush-test-$(date +%s)

echo "🧪 运行 SSE 测试..."
timeout 10 python3 docs/test_sse.py || true

# 清理
kill $SERVER_PID 2>/dev/null || true
echo "🎉 测试完成"
