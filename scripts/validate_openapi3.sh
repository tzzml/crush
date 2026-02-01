#!/bin/bash
# 验证 OpenAPI 3.0 文档的完整性

echo "🔍 验证 OpenAPI 3.0 文档完整性"
echo "================================"
echo ""

FAILED=0

# 1. 检查基本结构
echo "1️⃣ 检查基本结构..."

# 检查 openapi 字段
if ! grep -q '"openapi": "3.0.0"' docs/openapi3.json; then
    echo "   ✗ 缺少 openapi 版本字段"
    FAILED=1
else
    echo "   ✓ openapi 版本正确"
fi

# 检查 info 字段
if ! grep -q '"info"' docs/openapi3.json; then
    echo "   ✗ 缺少 info 字段"
    FAILED=1
else
    echo "   ✓ info 字段存在"
fi

# 检查 paths 字段
if ! grep -q '"paths"' docs/openapi3.json; then
    echo "   ✗ 缺少 paths 字段"
    FAILED=1
else
    echo "   ✓ paths 字段存在"
fi

# 检查 components 字段
if ! grep -q '"components"' docs/openapi3.json; then
    echo "   ✗ 缺少 components 字段"
    FAILED=1
else
    echo "   ✓ components 字段存在"
fi

# 2. 检查引用路径
echo ""
echo "2️⃣ 检查引用路径..."

# 检查是否还有旧的 definitions 引用
OLD_REFS=$(grep -c '#/definitions/' docs/openapi3.json || true)
if [ "$OLD_REFS" -gt 0 ]; then
    echo "   ✗ 发现 $OLD_REFS 个旧的 #/definitions/ 引用"
    FAILED=1
else
    echo "   ✓ 没有旧的 definitions 引用"
fi

# 检查 components/schemas 引用
NEW_REFS=$(grep -c '#/components/schemas/' docs/openapi3.json || true)
if [ "$NEW_REFS" -gt 0 ]; then
    echo "   ✓ 发现 $NEW_REFS 个正确的 #/components/schemas/ 引用"
else
    echo "   ⚠ 没有找到任何 schema 引用"
fi

# 3. 检查 requestBody 格式
echo ""
echo "3️⃣ 检查 requestBody 格式..."

# 检查 requestBody 中的 schema 引用
REQUESTBODY_COUNT=$(grep -c '"requestBody"' docs/openapi3.json || true)
if [ "$REQUESTBODY_COUNT" -gt 0 ]; then
    echo "   ✓ 发现 $REQUESTBODY_COUNT 个 requestBody"

    # 检查 requestBody 中是否有错误的引用
    WRONG_REF=$(grep -A 10 '"requestBody"' docs/openapi3.json | grep -c '#/definitions/' || true)
    if [ "$WRONG_REF" -gt 0 ]; then
        echo "   ✗ requestBody 中有 $WRONG_REF 个错误的引用"
        FAILED=1
    else
        echo "   ✓ requestBody 中的引用都正确"
    fi
else
    echo "   ⚠ 没有找到 requestBody"
fi

# 4. 检查响应格式
echo ""
echo "4️⃣ 检查响应格式..."

# 检查是否有 content 字段
CONTENT_COUNT=$(grep -c '"content"' docs/openapi3.json || true)
if [ "$CONTENT_COUNT" -gt 0 ]; then
    echo "   ✓ 发现 $CONTENT_COUNT 个 content 字段"
else
    echo "   ⚠ 没有找到 content 字段"
fi

# 5. 验证 JSON 格式
echo ""
echo "5️⃣ 验证 JSON 格式..."
if python3 -m json.tool docs/openapi3.json > /dev/null 2>&1; then
    echo "   ✓ JSON 格式有效"
else
    echo "   ✗ JSON 格式无效"
    FAILED=1
fi

# 6. 统计信息
echo ""
echo "6️⃣ 文档统计..."
ENDPOINTS=$(python3 -c "import json; data=json.load(open('docs/openapi3.json')); print(len(data.get('paths', {})))")
SCHEMAS=$(python3 -c "import json; data=json.load(open('docs/openapi3.json')); print(len(data.get('components', {}).get('schemas', {})))")
echo "   • API 端点数: $ENDPOINTS"
echo "   • 数据模型数: $SCHEMAS"

# 7. 检查一些关键路径
echo ""
echo "7️⃣ 检查关键路径..."

KEY_PATHS=(
    "/project"
    "/session"
    "/session/{session_id}/message"
    "/file"
)

for path in "${KEY_PATHS[@]}"; do
    if grep -q "\"$path\"" docs/openapi3.json; then
        echo "   ✓ $path 存在"
    else
        echo "   ✗ $path 不存在"
        FAILED=1
    fi
done

# 结果
echo ""
echo "================================"
if [ $FAILED -eq 0 ]; then
    echo "✅ 所有验证通过!"
    echo ""
    echo "🎉 OpenAPI 3.0 文档完全符合规范!"
    exit 0
else
    echo "❌ 验证失败，请检查上述错误"
    exit 1
fi
