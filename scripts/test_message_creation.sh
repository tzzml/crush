#!/bin/bash

# 测试消息创建 API，验证用户消息是否只创建一次

set -e

PROJECT_DIR="/Users/zhuminglei/Projects/test2"
API_URL="http://localhost:8080"

echo "=== 测试消息创建 ==="
echo

# 1. 创建新会话
echo "1. 创建新会话..."
SESSION_RESPONSE=$(curl -s -X POST "${API_URL}/session?directory=${PROJECT_DIR}" \
  -H "Content-Type: application/json" \
  -d '{"title":"测试重复消息"}')

SESSION_ID=$(echo "$SESSION_RESPONSE" | jq -r '.session.id')
echo "   会话 ID: $SESSION_ID"
echo

# 2. 发送消息
echo "2. 发送消息..."
curl -s -X POST "${API_URL}/session/${SESSION_ID}/message?directory=${PROJECT_DIR}" \
  -H "Content-Type: application/json" \
  -d '{"prompt":"测试消息","stream":false}' | jq '.'
echo

# 3. 等待一下
sleep 2

# 4. 获取会话的所有消息
echo "3. 获取会话的所有消息..."
MESSAGES_RESPONSE=$(curl -s -X GET "${API_URL}/session/${SESSION_ID}/message?directory=${PROJECT_DIR}")

# 提取消息数量和内容
MESSAGE_COUNT=$(echo "$MESSAGES_RESPONSE" | jq '.total')
echo "   消息总数: $MESSAGE_COUNT"

# 提取用户消息数量
USER_MESSAGE_COUNT=$(echo "$MESSAGES_RESPONSE" | jq '[.messages[] | select(.role=="user")] | length')
echo "   用户消息数量: $USER_MESSAGE_COUNT"

# 显示所有消息的角色和内容
echo
echo "4. 消息列表："
echo "$MESSAGES_RESPONSE" | jq -r '.messages[] | "   \(.role): \(.content // [empty] | if type == "array" then "[附件]" else .[0:50] end)"'

echo
echo "=== 验证结果 ==="
if [ "$USER_MESSAGE_COUNT" -eq 1 ]; then
  echo "✅ 成功：用户消息只创建了 1 次"
  exit 0
else
  echo "❌ 失败：用户消息创建了 $USER_MESSAGE_COUNT 次（期望 1 次）"
  exit 1
fi
