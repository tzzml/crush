# OpenAPI 3.0 文档

本项目支持从代码自动生成 OpenAPI 3.0 规范的 API 文档。

## 快速开始

### 生成 OpenAPI 3.0 文档

```bash
# 从代码生成 OpenAPI 3.0 文档 (推荐)
make swagger
```

这个命令会:
1. 使用 `swag` 工具从代码注解生成 Swagger 2.0 文档
2. 使用转换脚本将 Swagger 2.0 转换为 OpenAPI 3.0
3. 最终生成 `docs/openapi3.json` 文件

### 仅生成 Swagger 2.0 文档

```bash
make swagger2
```

### 手动转换现有文档

如果你有一个 Swagger 2.0 文档想要转换为 OpenAPI 3.0:

```bash
python3 scripts/convert_to_openapi3.py input.json output.json
```

## 文档文件

生成的文件位于 `docs/` 目录:

- `openapi3.json` - OpenAPI 3.0 JSON 格式 (推荐)
- `swagger.json` - Swagger 2.0 JSON 格式
- `swagger.yaml` - Swagger 2.0 YAML 格式

## 使用文档

### 在 Swagger UI 中查看

启动服务器后,访问:
- http://localhost:8080/swagger

### 使用其他工具

生成的 OpenAPI 3.0 文档可以用于:

1. **Redoc** - 更美观的文档展示
2. **Swagger UI** - 官方文档界面
3. **Postman** - API 测试
4. **OpenAPI Generator** - 生成客户端 SDK

## 代码注解

在 Go 代码中使用 swag 注解来定义 API:

```go
// @Summary 获取项目列表
// @Description 获取所有已注册的项目
// @Tags Project
// @Accept json
// @Produce json
// @Success 200 {object} models.ProjectsResponse
// @Router /project [get]
func HandleListProjects(c context.Context, ctx *app.RequestContext) {
    // ...
}
```

更多注解语法请参考: [swaggo/swag](https://github.com/swaggo/swag)

## 主要变化: Swagger 2.0 → OpenAPI 3.0

| Swagger 2.0 | OpenAPI 3.0 |
|------------|-------------|
| `swagger: "2.0"` | `openapi: "3.0.0"` |
| `definitions` | `components/schemas` |
| `host`, `basePath`, `schemes` | `servers` |
| parameters `in: body` | `requestBody` |
| response `schema` | `content.<media-type>.schema` |

## 自动化

在开发过程中,你可以使用 `dev` 命令自动生成文档并启动服务器:

```bash
make dev
```

这会在每次启动前自动重新生成文档。

## 清理

删除所有生成的文档文件:

```bash
make clean
```
