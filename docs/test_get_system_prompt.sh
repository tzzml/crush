#!/bin/bash

# 测试 GET /system-prompt 端点
# 这个脚本验证获取系统提示词功能是否正常工作

set -e

BASE_URL="http://localhost:8080"
PROJECT_DIR="/Users/zhuminglei/Projects/zorkagent"

echo "========================================"
echo "测试 GET /system-prompt 端点"
echo "========================================"
echo ""

# 测试 1: 获取当前系统提示词
echo "测试 1: 获取当前系统提示词"
echo "请求: GET ${BASE_URL}/system-prompt?directory=${PROJECT_DIR}"
echo ""

RESPONSE=$(curl -s -X GET "${BASE_URL}/system-prompt?directory=${PROJECT_DIR}")
echo "响应:"
echo "$RESPONSE" | jq '.'
echo ""

# 检查响应
PROMPT_LENGTH=$(echo "$RESPONSE" | jq -r '.length')
SYSTEM_PROMPT=$(echo "$RESPONSE" | jq -r '.system_prompt')

echo "提示词长度: $PROMPT_LENGTH"
echo ""

if [ "$PROMPT_LENGTH" -gt 0 ]; then
    echo "✅ 成功: 系统提示词不为空"
    echo ""
    echo "提示词预览 (前 200 字符):"
    echo "$SYSTEM_PROMPT" | head -c 200
    echo "..."
else
    echo "❌ 失败: 系统提示词为空"
    exit 1
fi

echo ""
echo "========================================"
echo "测试完成"
echo "========================================"
