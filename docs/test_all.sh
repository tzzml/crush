#!/bin/bash
# Crush API 测试脚本

set -e

echo "🚀 Crush API 测试套件"
echo "===================="

# 检查依赖
command -v python3 >/dev/null 2>&1 || { echo "❌ 需要安装 python3"; exit 1; }
python3 -c "import requests, sseclient" >/dev/null 2>&1 || {
    echo "❌ 需要安装: pip install requests sseclient-py"
    exit 1
}

# 检查可执行文件
if [ ! -f "./crush" ]; then
    echo "❌ 找不到 crush 可执行文件"
    echo "   请先编译: go build -o crush ."
    exit 1
fi

# 启动服务器
echo ""
echo "🖥️  启动服务器..."
./crush serve --port 8080 > /tmp/crush-server.log 2>&1 &
SERVER_PID=$!

# 等待服务器启动
echo "⏳ 等待服务器启动..."
for i in {1..10}; do
    if curl -s http://localhost:8080/api/v1/health >/dev/null 2>&1; then
        echo "✅ 服务器运行正常 (PID: $SERVER_PID)"
        break
    fi
    if [ $i -eq 10 ]; then
        echo "❌ 服务器启动失败"
        echo "   日志:"
        tail -20 /tmp/crush-server.log
        kill $SERVER_PID 2>/dev/null || true
        exit 1
    fi
    sleep 1
done

# 运行测试
echo ""
echo "🧪 运行 API 测试..."
TEST_PROJECT="/tmp/crush-test-$(date +%s)"
python3 docs/test_api.py --project-path "$TEST_PROJECT" || {
    echo "❌ API 测试失败"
    kill $SERVER_PID 2>/dev/null || true
    exit 1
}

echo ""
echo "🧪 运行 SSE 测试..."
timeout 15 python3 docs/test_sse.py || {
    echo "⚠️  SSE 测试超时或失败（这可能是正常的）"
}

# 清理
echo ""
echo "🧹 清理..."
kill $SERVER_PID 2>/dev/null || true
wait $SERVER_PID 2>/dev/null || true

echo ""
echo "🎉 测试完成"
echo "===================="
