#!/bin/bash
# 测试 OpenAPI 3.0 实现

echo "🧪 测试 OpenAPI 3.0 实现"
echo "========================"
echo ""

# 1. 测试文件是否存在
echo "1️⃣ 检查生成的文件..."
if [ -f "docs/openapi3.json" ]; then
    echo "   ✓ openapi3.json 存在"
else
    echo "   ✗ openapi3.json 不存在"
    exit 1
fi

# 2. 验证 JSON 格式
echo ""
echo "2️⃣ 验证 JSON 格式..."
if python3 -m json.tool docs/openapi3.json > /dev/null 2>&1; then
    echo "   ✓ JSON 格式有效"
else
    echo "   ✗ JSON 格式无效"
    exit 1
fi

# 3. 检查 OpenAPI 版本
echo ""
echo "3️⃣ 检查 OpenAPI 版本..."
VERSION=$(cat docs/openapi3.json | grep -o '"openapi": "[^"]*"' | cut -d'"' -f4)
if [ "$VERSION" == "3.0.0" ]; then
    echo "   ✓ OpenAPI 版本: $VERSION"
else
    echo "   ✗ OpenAPI 版本不正确: $VERSION"
    exit 1
fi

# 4. 统计端点和模型
echo ""
echo "4️⃣ 统计 API 内容..."
ENDPOINTS=$(cat docs/openapi3.json | python3 -c "import json,sys; data=json.load(sys.stdin); print(len(data.get('paths', {})))")
SCHEMAS=$(cat docs/openapi3.json | python3 -c "import json,sys; data=json.load(sys.stdin); print(len(data.get('components', {}).get('schemas', {})))")
echo "   ✓ API 端点数: $ENDPOINTS"
echo "   ✓ 数据模型数: $SCHEMAS"

# 5. 检查关键组件
echo ""
echo "5️⃣ 检查 OpenAPI 3.0 关键组件..."
if cat docs/openapi3.json | grep -q '"components"'; then
    echo "   ✓ components 存在"
else
    echo "   ✗ components 不存在"
    exit 1
fi

if cat docs/openapi3.json | grep -q '"servers"'; then
    echo "   ✓ servers 存在"
else
    echo "   ⚠ servers 不存在 (可选)"
fi

# 6. 检查转换脚本
echo ""
echo "6️⃣ 检查转换脚本..."
if [ -f "scripts/convert_to_openapi3.py" ]; then
    echo "   ✓ 转换脚本存在"
else
    echo "   ✗ 转换脚本不存在"
    exit 1
fi

# 7. 显示一些示例端点
echo ""
echo "7️⃣ 示例 API 端点:"
cat docs/openapi3.json | python3 -c "
import json, sys
data = json.load(sys.stdin)
for i, (path, methods) in enumerate(list(data['paths'].items())[:5]):
    for method, details in methods.items():
        summary = details.get('summary', 'N/A')
        print(f'   {method.upper():6} {path:25} - {summary}')
"

echo ""
echo "========================"
echo "✅ 所有测试通过!"
echo ""
echo "📚 使用方式:"
echo "   make swagger      - 生成 OpenAPI 3.0 文档"
echo "   make run          - 启动服务器"
echo "   然后:"
echo "     http://localhost:8080/swagger   - Swagger UI"
echo "     http://localhost:8080/redoc     - Redoc UI"
echo "     http://localhost:8080/swagger/openapi3.json - OpenAPI 3.0 JSON"
