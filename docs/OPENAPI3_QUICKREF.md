# OpenAPI 3.0 快速参考

## 📋 生成文档

```bash
# 从代码生成 OpenAPI 3.0 文档
make swagger

# 仅生成 Swagger 2.0
make swagger2

# 转换现有文档
python3 scripts/convert_to_openapi3.py input.json output.json

# 测试实现
scripts/test_openapi3.sh
```

## 🌐 访问文档

启动服务器 (`make run`) 后:

| 界面 | URL | 说明 |
|------|-----|------|
| **Swagger UI** | http://localhost:8080/swagger | 交互式 API 测试 |
| **Redoc** | http://localhost:8080/redoc | 美观的文档展示 |
| **OpenAPI 3.0** | http://localhost:8080/swagger/openapi3.json | JSON 规范 |
| **Swagger 2.0** | http://localhost:8080/swagger/doc.json | JSON 规范 |

## 📝 代码注解示例

```go
// @Summary 获取项目列表
// @Description 获取所有已注册的项目
// @Tags Project
// @Accept json
// @Produce json
// @Param directory query string true "项目路径"
// @Success 200 {object} models.ProjectsResponse
// @Failure 500 {object} models.ErrorResponse
// @Router /project [get]
func HandleListProjects(c context.Context, ctx *app.RequestContext) {
    // ...
}
```

## 🔄 主要变化 (Swagger 2.0 → OpenAPI 3.0)

| Swagger 2.0 | OpenAPI 3.0 |
|------------|-------------|
| `swagger: "2.0"` | `openapi: "3.0.0"` |
| `definitions` | `components/schemas` |
| `#/definitions/...` | `#/components/schemas/...` |
| `host`, `basePath`, `schemes` | `servers` |
| parameters `in: body` | `requestBody` |
| response `schema` | `content.<media-type>.schema` |

## 📦 生成的文件

```
docs/
├── openapi3.json      # OpenAPI 3.0 (推荐)
├── swagger.json       # Swagger 2.0
└── swagger.yaml       # Swagger 2.0 YAML
```

## 🛠️ 使用 OpenAPI 3.0 文档

### 生成客户端 SDK
```bash
# 使用 OpenAPI Generator
openapi-generator-cli generate -i docs/openapi3.json -g go -o ./client
```

### 导入到 Postman
1. 打开 Postman
2. Import → 选择 `docs/openapi3.json`
3. 自动生成所有 API 请求

### 在其他工具中使用
- **Insomnia**: 导入 openapi3.json
- **Swagger Codegen**: 生成服务器 stub
- **Redoc**: 静态文档生成

## ✅ 验证文档

```bash
# 检查 JSON 格式
cat docs/openapi3.json | python3 -m json.tool

# 检查 OpenAPI 规范
cat docs/openapi3.json | grep '"openapi"'

# 统计信息
cat docs/openapi3.json | python3 -c "
import json, sys
data = json.load(sys.stdin)
print(f'Version: {data[\"openapi\"]}')
print(f'Endpoints: {len(data[\"paths\"])}')
print(f'Schemas: {len(data[\"components\"][\"schemas\"])}')
"
```

## 🚀 开发工作流

```bash
# 1. 修改代码和注解
vim api/handlers/projects.go

# 2. 重新生成文档
make swagger

# 3. 测试文档
scripts/test_openapi3.sh

# 4. 启动服务器
make run

# 5. 访问文档
open http://localhost:8080/redoc
```

## 📚 更多信息

- [OpenAPI 3.0 规范](https://swagger.io/specification/)
- [Swag 注解指南](https://github.com/swaggo/swag)
- [项目文档](./OPENAPI3.md)
- [实现总结](./OPENAPI3_SUMMARY.md)
