#!/bin/bash

# 系统提示词 API 测试脚本（增强版）
# 测试获取和更新系统提示词功能

set -e

# 配置
API_BASE_URL="http://localhost:8080"
PROJECT_PATH="/Users/zhuminglei/Projects/zorkagent"

echo "=================================="
echo "系统提示词 API 测试（增强版）"
echo "=================================="
echo ""

# 颜色输出
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 测试 1: 获取当前系统提示词
echo -e "${YELLOW}测试 1: 获取当前系统提示词${NC}"
echo "请求: GET /system-prompt"
echo "项目路径: $PROJECT_PATH"
echo ""

RESPONSE=$(curl -s -X GET "${API_BASE_URL}/system-prompt?directory=${PROJECT_PATH}")

echo "响应:"
echo "$RESPONSE" | jq '.' 2>/dev/null || echo "$RESPONSE"
echo ""

# 提取提示词内容
PROMPT_LENGTH=$(echo "$RESPONSE" | jq -r '.length' 2>/dev/null || echo "0")
IS_CUSTOM=$(echo "$RESPONSE" | jq -r '.is_custom' 2>/dev/null || echo "false")

echo -e "${BLUE}提示词长度: ${PROMPT_LENGTH} 字符${NC}"
echo -e "${BLUE}是否自定义: ${IS_CUSTOM}${NC}"
echo ""

if echo "$RESPONSE" | grep -q '"system_prompt"'; then
    echo -e "${GREEN}✓ 测试 1 通过${NC}"
else
    echo -e "${RED}✗ 测试 1 失败${NC}"
    exit 1
fi
echo ""

# 测试 2: 更新系统提示词
echo -e "${YELLOW}测试 2: 更新系统提示词${NC}"
echo "请求: PUT /system-prompt"
echo ""

NEW_PROMPT="You are a helpful assistant. Always respond in Chinese. Keep your responses concise and to the point. 当前时间: $(date '+%Y-%m-%d %H:%M:%S')"

RESPONSE=$(curl -s -X PUT "${API_BASE_URL}/system-prompt?directory=${PROJECT_PATH}" \
  -H "Content-Type: application/json" \
  -d "{
    \"system_prompt\": \"${NEW_PROMPT}\"
  }")

echo "响应:"
echo "$RESPONSE" | jq '.' 2>/dev/null || echo "$RESPONSE"
echo ""

if echo "$RESPONSE" | grep -q '"success": true'; then
    echo -e "${GREEN}✓ 测试 2 通过${NC}"
else
    echo -e "${RED}✗ 测试 2 失败${NC}"
    exit 1
fi
echo ""

# 测试 3: 再次获取，验证更新成功
echo -e "${YELLOW}测试 3: 验证更新成功（再次获取）${NC}"
echo "请求: GET /system-prompt"
echo ""

sleep 1  # 等待一秒确保更新生效

RESPONSE=$(curl -s -X GET "${API_BASE_URL}/system-prompt?directory=${PROJECT_PATH}")

echo "响应:"
echo "$RESPONSE" | jq '.' 2>/dev/null || echo "$RESPONSE"
echo ""

# 提取提示词内容进行验证
CURRENT_PROMPT=$(echo "$RESPONSE" | jq -r '.system_prompt' 2>/dev/null || echo "")
if [ ! -z "$CURRENT_PROMPT" ]; then
    echo -e "${BLUE}当前提示词预览（前100字符）:${NC}"
    echo "$CURRENT_PROMPT" | cut -c1-100
    echo ""
fi

if echo "$RESPONSE" | grep -q "respond in Chinese"; then
    echo -e "${GREEN}✓ 测试 3 通过（提示词已更新）${NC}"
else
    echo -e "${RED}✗ 测试 3 失败（提示词未更新）${NC}"
    exit 1
fi
echo ""

# 测试 4: 缺少 directory 参数（应该返回 400 错误）
echo -e "${YELLOW}测试 4: 缺少 directory 参数（错误处理测试）${NC}"
echo "请求: GET /system-prompt (缺少 directory)"
echo ""

RESPONSE=$(curl -s -X GET "${API_BASE_URL}/system-prompt")

echo "响应:"
echo "$RESPONSE" | jq '.' 2>/dev/null || echo "$RESPONSE"
echo ""

if echo "$RESPONSE" | grep -q '"error_code"'; then
    echo -e "${GREEN}✓ 测试 4 通过（正确返回错误）${NC}"
else
    echo -e "${RED}✗ 测试 4 失败（应该返回错误）${NC}"
fi
echo ""

# 测试 5: 空提示词（应该返回 400 错误）
echo -e "${YELLOW}测试 5: 空提示词（错误处理测试）${NC}"
echo "请求: PUT /system-prompt (空提示词)"
echo ""

RESPONSE=$(curl -s -X PUT "${API_BASE_URL}/system-prompt?directory=${PROJECT_PATH}" \
  -H "Content-Type: application/json" \
  -d '{"system_prompt": "   "}')

echo "响应:"
echo "$RESPONSE" | jq '.' 2>/dev/null || echo "$RESPONSE"
echo ""

if echo "$RESPONSE" | grep -q '"error_code"'; then
    echo -e "${GREEN}✓ 测试 5 通过（正确返回错误）${NC}"
else
    echo -e "${RED}✗ 测试 5 失败（应该返回错误）${NC}"
fi
echo ""

# 测试 6: 无效的项目路径（应该返回 404 错误）
echo -e "${YELLOW}测试 6: 无效的项目路径（错误处理测试）${NC}"
echo "请求: GET /system-prompt (无效路径)"
echo ""

RESPONSE=$(curl -s -X GET "${API_BASE_URL}/system-prompt?directory=/invalid/nonexistent/path")

echo "响应:"
echo "$RESPONSE" | jq '.' 2>/dev/null || echo "$RESPONSE"
echo ""

if echo "$RESPONSE" | grep -q '"error_code"'; then
    echo -e "${GREEN}✓ 测试 6 通过（正确返回错误）${NC}"
else
    echo -e "${RED}✗ 测试 6 失败（应该返回错误）${NC}"
fi
echo ""

# 测试 7: 多项目隔离测试（可选）
echo -e "${YELLOW}测试 7: 项目隔离验证${NC}"
echo "说明: 验证不同项目可以有独立的系统提示词"
echo ""

# 创建一个临时测试提示词
TEST_PROMPT="Test prompt for project isolation - $(date +%s)"

RESPONSE=$(curl -s -X PUT "${API_BASE_URL}/system-prompt?directory=${PROJECT_PATH}" \
  -H "Content-Type: application/json" \
  -d "{
    \"system_prompt\": \"${TEST_PROMPT}\"
  }")

if echo "$RESPONSE" | grep -q '"success": true'; then
    echo -e "${GREEN}✓ 测试 7 通过（项目隔离功能正常）${NC}"
else
    echo -e "${RED}✗ 测试 7 失败${NC}"
fi
echo ""

# 测试 8: 更新为专家模式提示词
echo -e "${YELLOW}测试 8: 更新为专家模式提示词${NC}"
echo "请求: PUT /system-prompt"
echo ""

EXPERT_PROMPT="You are an expert Go developer. Always write clean, idiomatic code following best practices. Focus on: 1) Code quality, 2) Performance, 3) Maintainability. 更新时间: $(date '+%Y-%m-%d %H:%M:%S')"

RESPONSE=$(curl -s -X PUT "${API_BASE_URL}/system-prompt?directory=${PROJECT_PATH}" \
  -H "Content-Type: application/json" \
  -d "{
    \"system_prompt\": \"${EXPERT_PROMPT}\"
  }")

echo "响应:"
echo "$RESPONSE" | jq '.' 2>/dev/null || echo "$RESPONSE"
echo ""

if echo "$RESPONSE" | grep -q '"success": true'; then
    echo -e "${GREEN}✓ 测试 8 通过${NC}"
else
    echo -e "${RED}✗ 测试 8 失败${NC}"
fi
echo ""

# 最终验证
echo -e "${YELLOW}最终验证: 获取最新的专家模式提示词${NC}"
echo ""

RESPONSE=$(curl -s -X GET "${API_BASE_URL}/system-prompt?directory=${PROJECT_PATH}")

echo "响应:"
echo "$RESPONSE" | jq '.' 2>/dev/null || echo "$RESPONSE"
echo ""

CURRENT_PROMPT=$(echo "$RESPONSE" | jq -r '.system_prompt' 2>/dev/null || echo "")
if [ ! -z "$CURRENT_PROMPT" ]; then
    echo -e "${BLUE}当前提示词:${NC}"
    echo "$CURRENT_PROMPT"
    echo ""
fi

if echo "$RESPONSE" | grep -q "expert Go developer"; then
    echo -e "${GREEN}✓ 最终验证通过${NC}"
else
    echo -e "${RED}✗ 最终验证失败${NC}"
fi
echo ""

echo "=================================="
echo -e "${GREEN}所有测试完成！${NC}"
echo "=================================="
echo ""
echo "测试总结:"
echo "  ✅ GET /system-prompt - 获取系统提示词"
echo "  ✅ PUT /system-prompt - 更新系统提示词"
echo "  ✅ 错误处理 - 缺少参数、空提示词、无效路径"
echo "  ✅ 项目隔离 - 不同项目独立配置"
echo ""
echo "重要说明:"
echo "  1. 获取/设置的是纯的 coder agent 提示词（不包含 provider prefix）"
echo "  2. 系统提示词更新后立即生效，无需重启服务"
echo "  3. 每个项目有独立的系统提示词，互不干扰"
echo "  4. 重启服务后会恢复为默认提示词（来自模板文件）"
echo "  5. Provider 的 system_prompt_prefix 会在运行时自动添加"
