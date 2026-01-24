# Swagger API 文档集成指南

本文档介绍如何在 Go + Hertz 项目中集成 Swagger API 文档。

## 目录

- [概述](#概述)
- [技术栈](#技术栈)
- [安装步骤](#安装步骤)
- [项目结构](#项目结构)
- [实现步骤](#实现步骤)
- [Swagger 注释语法](#swagger-注释语法)
- [使用流程](#使用流程)
- [常见问题](#常见问题)

## 概述

本项目使用 **swaggo/swag** 工具自动生成 Swagger/OpenAPI 3.0 规范的 API 文档，并提供交互式的 Swagger UI 界面。

**主要功能：**
- 📝 自动从代码注释生成 API 文档
- 🎨 提供美观的 Swagger UI 界面
- 🔐 支持 JWT 认证（自动 token 管理）
- 📊 支持在线测试 API 接口
- 📄 导出 OpenAPI JSON/YAML 规范

**访问地址：**
- Swagger UI: `http://localhost:54321/swagger`
- OpenAPI JSON: `http://localhost:54321/swagger/doc.json`

## 技术栈

- **swaggo/swag** v1.16.6 - Swagger 文档生成工具
- **CloudWeGo Hertz** - Go HTTP 框架
- **Swagger UI** v3.52.5 - 前端文档界面

## 安装步骤

### 1. 安装 swag 命令行工具

```bash
go install github.com/swaggo/swag/cmd/swag@latest
```

验证安装：
```bash
swag --version
# 输出: swag version v1.16.6
```

### 2. 添加项目依赖

在 `go.mod` 中确保包含以下依赖：

```go
require (
    github.com/swaggo/swag v1.16.6
    github.com/go-openapi/spec v0.20.4
    github.com/go-openapi/jsonreference v0.19.6
)
```

安装依赖：
```bash
go mod tidy
```

## 项目结构

```
backend/
├── cmd/
│   └── main.go              # 主入口（必须导入 docs 包）
├── docs/                    # swag 自动生成的文档目录
│   ├── docs.go              # Swagger 文档代码
│   ├── swagger.json         # OpenAPI JSON 规范
│   └── swagger.yaml         # OpenAPI YAML 规范
├── handlers/
│   ├── swagger.go           # Swagger UI 路由处理器
│   ├── auth_v2.go           # 业务处理器（含注释示例）
│   ├── chat.go
│   └── session.go
└── routes/
    └── routes.go            # 路由注册（含 Swagger 路由）
```

## 实现步骤

### 步骤 1: 在主程序中导入 docs 包

**文件：** `cmd/main.go`

```go
package main

import (
    // ... 其他导入
    _ "test-claude-agent-go/backend/docs" // ⚠️ 重要：必须导入 swagger 文档
)

func main() {
    // ... 服务器代码
}
```

**注意：** 导入路径使用 `项目模块名/docs`，必须与 `go.mod` 中的 `module` 声明一致。

---

### 步骤 2: 创建 Swagger UI 处理器

**文件：** `handlers/swagger.go`

```go
package handlers

import (
	"context"
	"net/http"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/swaggo/swag"
	_ "test-claude-agent-go/backend/docs"
)

// SwaggerHTML 是 Swagger UI 的 HTML 页面
const SwaggerHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>API - Swagger UI</title>
    <link rel="stylesheet" type="text/css" href="https://unpkg.com/swagger-ui-dist@3.52.5/swagger-ui.css">
    <style>
        body { margin: 0; padding: 0; }
        #swagger-ui { max-width: 1460px; margin: 0 auto; }
    </style>
</head>
<body>
    <div id="swagger-ui"></div>
    <script src="https://unpkg.com/swagger-ui-dist@3.52.5/swagger-ui-bundle.js"></script>
    <script src="https://unpkg.com/swagger-ui-dist@3.52.5/swagger-ui-standalone-preset.js"></script>
    <script>
        window.onload = function() {
            const ui = SwaggerUIBundle({
                url: '/swagger/doc.json',
                dom_id: '#swagger-ui',
                deepLinking: true,
                presets: [
                    SwaggerUIBundle.presets.apis,
                    SwaggerUIStandalonePreset
                ],
                plugins: [
                    SwaggerUIBundle.plugins.DownloadUrl
                ],
                layout: "StandaloneLayout",
                defaultModelsExpandDepth: 1,
                defaultModelExpandDepth: 2,
                docExpansion: "list",
                persistAuthorization: true,
                displayRequestDuration: true,
                requestInterceptor: (request) => {
                    // 自动添加 JWT token
                    const token = localStorage.getItem('BearerAuth');
                    if (token && request.headers) {
                        request.headers.Authorization = 'Bearer ' + token;
                    }
                    return request;
                },
                responseInterceptor: async (response) => {
                    // 登录成功后自动保存 token
                    if (response.ok && response.url && response.url.includes('/auth/login')) {
                        try {
                            const data = await response.json();
                            if (data.token) {
                                localStorage.setItem('BearerAuth', data.token);
                                setTimeout(() => location.reload(), 500);
                            }
                        } catch (e) {
                            console.error('Failed to parse response', e);
                        }
                    }
                    return response;
                }
            });
            window.ui = ui;
        };
    </script>
</body>
</html>`

// HandleSwaggerUI 处理 GET /swagger 请求
//
//	@Summary		Swagger UI
//	@Description	交互式 API 文档
//	@Tags			Documentation
//	@Accept			html
//	@Produce		html
//	@Router			/swagger [get]
func HandleSwaggerUI(c context.Context, ctx *app.RequestContext) {
	ctx.SetContentType("text/html; charset=utf-8")
	ctx.Response.SetBody([]byte(SwaggerHTML))
}

// HandleSwaggerJSON 处理 GET /swagger/doc.json 请求
//
//	@Summary		OpenAPI 规范
//	@Description	返回 OpenAPI 3.0 JSON 规范
//	@Tags			Documentation
//	@Accept			json
//	@Produce		json
//	@Router			/swagger/doc.json [get]
func HandleSwaggerJSON(c context.Context, ctx *app.RequestContext) {
	doc, err := swag.ReadDoc("swagger")
	if err != nil {
		ctx.JSON(500, map[string]string{"error": "failed to read swagger doc"})
		return
	}
	// ⚠️ 重要：直接返回原始 JSON，避免双重包裹
	ctx.SetStatusCode(200)
	ctx.SetContentType("application/json; charset=utf-8")
	ctx.Response.SetBody([]byte(doc))
}

// HandleIndexRedirect 处理 GET / 请求
//
//	@Summary		首页重定向
//	@Description	重定向到 Swagger UI
//	@Tags			Documentation
//	@Router			/ [get]
func HandleIndexRedirect(c context.Context, ctx *app.RequestContext) {
	ctx.Response.SetStatusCode(http.StatusFound)
	ctx.Response.Header.Set("Location", "/swagger")
}
```

---

### 步骤 3: 注册路由

**文件：** `routes/routes.go`

```go
// registerSwaggerRoutes 注册 Swagger 文档路由
func registerSwaggerRoutes(h *server.Hertz) {
	// 首页重定向到 Swagger UI
	h.GET("/", handlers.HandleIndexRedirect)
	// Swagger UI 页面
	h.GET("/swagger", handlers.HandleSwaggerUI)
	// OpenAPI JSON 规范
	h.GET("/swagger/doc.json", handlers.HandleSwaggerJSON)
}

// 在主路由注册函数中调用
func Register(h *server.Hertz, deps *Dependencies) {
	// ... 其他路由
	registerSwaggerRoutes(h)  // ⚠️ 添加这一行
}
```

---

### 步骤 4: 在 Handler 中添加 Swagger 注释

**文件：** `handlers/auth_v2.go`（示例）

```go
// Register handles POST /api/auth/register.
//
//	@Summary		用户注册
//	@Description	创建新用户账号并返回 JWT token
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Param			request	body		RegisterRequest	true	"注册请求"
//	@Success		201		{object}	AuthResponse	"注册成功"
//	@Failure		400		{object}	map[string]interface{}	"请求格式无效"
//	@Failure		409		{object}	map[string]interface{}	"用户名或邮箱已存在"
//	@Failure		500		{object}	map[string]interface{}	"服务器内部错误"
//	@Router			/api/auth/register [post]
func (h *AuthV2Handler) Register(c context.Context, ctx *app.RequestContext) {
	// 实现代码...
}
```

---

### 步骤 5: 生成 Swagger 文档

在项目根目录（`backend/`）运行：

```bash
swag init -g cmd/main.go -o docs --parseDependency --parseInternal
```

**参数说明：**
- `-g cmd/main.go` - 主入口文件路径
- `-o docs` - 输出目录
- `--parseDependency` - 解析依赖包中的注释
- `--parseInternal` - 解析 internal 包中的注释

**预期输出：**
```
2024/01/23 15:00:00 Generate docs
2024/01/23 15:00:00 Generate docs success
```

生成的文件：
- `docs/docs.go` - Go 代码
- `docs/swagger.json` - OpenAPI JSON
- `docs/swagger.yaml` - OpenAPI YAML

---

### 步骤 6: 启动服务器

```bash
go run cmd/main.go
```

访问 `http://localhost:54321/swagger` 查看 API 文档。

## Swagger 注释语法

### 基础注释格式

```go
// FunctionName 处理 HTTP 请求
//
//	@Summary		简短摘要（必填）
//	@Description	详细描述（可选）
//	@Tags			标签分组（必填，用于分类）
//	@Accept			json          // 接受的请求类型
//	@Produce		json          // 返回的响应类型
//	@Router			/path [method]
//	@Security		BearerAuth    // 认证方式（可选）
func HandlerName(c context.Context, ctx *app.RequestContext) {
    // ...
}
```

### 参数定义 (@Param)

```go
//	@Param		name		type		data source		required		description
//	@Param		id			path		int				true		"用户ID"
//	@Param		query		query		string			false		"搜索关键词"
//	@Param		page		query		int				false		"页码"
//	@Param		body		body		Request			true		"请求体"
```

**类型：** `path`, `query`, `header`, `body`, `formData`

**数据类型：** `string`, `int`, `bool`, `object`, `array`, 自定义结构体

### 响应定义 (@Success/@Failure)

```go
//	@Success	200	{object}	Response	"成功描述"
//	@Success	201	{object}	AuthResponse	"创建成功"
//	@Failure	400	{object}	ErrorResp	"请求错误"
//	@Failure	401	{object}	ErrorResp	"未授权"
//	@Failure	500	{object}	ErrorResp	"服务器错误"
```

**格式：** `@Success HTTP码 {类型} 数据结构 描述`

### 认证定义 (@Security)

```go
//	@Security	BearerAuth
```

需要在 Swagger UI 配置中定义认证类型（已在 `swagger.go` 中配置）。

### 完整示例

#### GET 请求示例

```go
// HandleSessions 获取会话列表
//
//	@Summary		获取会话列表
//	@Description	获取当前用户的所有会话
//	@Tags			Sessions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	[]Session	"成功"
//	@Failure		401	{object}	map[string]interface{}	"未授权"
//	@Router			/api/sessions [get]
func (h *SessionHandler) HandleSessions(c context.Context, ctx *app.RequestContext) {
    // ...
}
```

#### POST 请求示例

```go
// Login 用户登录
//
//	@Summary		用户登录
//	@Description	使用邮箱和密码登录，返回 JWT token
//	@Tags			Authentication
//	@Accept			json
//	@Produce		json
//	@Param			request	body		LoginRequest	true	"登录请求"
//	@Success		200		{object}	AuthResponse	"登录成功"
//	@Failure		401		{object}	map[string]interface{}	"认证失败"
//	@Router			/api/auth/login [post]
func (h *AuthV2Handler) Login(c context.Context, ctx *app.RequestContext) {
    // ...
}
```

#### DELETE 请求示例

```go
// HandleSessionPath 删除会话
//
//	@Summary		删除会话
//	@Description	删除指定 ID 的会话及其所有消息
//	@Tags			Sessions
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path		int		true	"会话ID"
//	@Success		204		"删除成功"
//	@Failure		404	{object}	map[string]interface{}	"会话不存在"
//	@Router			/api/sessions/{id} [delete]
func (h *SessionHandler) HandleSessionPath(c context.Context, ctx *app.RequestContext) {
    // ...
}
```

## 使用流程

### 开发工作流

```bash
# 1. 在 handler 函数前添加 Swagger 注释
# 编辑 handlers/xxx.go

# 2. 重新生成文档
swag init -g cmd/main.go -o docs --parseDependency --parseInternal

# 3. 重启服务器
go run cmd/main.go

# 4. 浏览器访问
open http://localhost:54321/swagger
```

### 可选：添加 Makefile 简化操作

**文件：** `Makefile`

```makefile
.PHONY: swagger run build clean

# 生成 swagger 文档
swagger:
	@echo "Generating swagger docs..."
	swag init -g cmd/main.go -o docs --parseDependency --parseInternal
	@echo "✓ Swagger docs generated"

# 运行服务器
run:
	@echo "Starting server..."
	go run cmd/main.go

# 构建
build:
	@echo "Building..."
	go build -o bin/server cmd/main.go

# 清理生成的文件
clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f docs/docs.go docs/swagger.json docs/swagger.yaml

# 开发模式（生成文档 + 运行）
dev: swagger run
```

使用：
```bash
make swagger   # 生成文档
make run       # 运行服务器
make dev       # 生成文档 + 运行
```

## 常见问题

### Q1: 生成文档时报错 "cannot find package"

**原因：** `-g` 参数指定的路径不正确。

**解决：** 确保路径相对于项目根目录：
```bash
# 错误示例
swag init -g main.go

# 正确示例
swag init -g cmd/main.go
```

### Q2: Swagger UI 显示 "No API definition found"

**原因：**
1. 未在 `main.go` 中导入 docs 包
2. 导入路径不正确
3. 未生成文档

**解决：**
```go
// 确保在 cmd/main.go 中导入
import _ "你的项目模块名/docs"
```

### Q3: 注释不生效

**原因：** 注释格式错误。

**解决：**
- 使用 `//	` (双斜杠 + tab) 格式
- 注释必须紧贴函数定义
- 确保所有必填字段都存在（`@Summary`, `@Tags`, `@Router`）

**正确格式：**
```go
// FunctionName 函数描述
//
//	@Summary	摘要
//	@Router		/path [method]
func FunctionName() {}
```

### Q4: 认证不生效

**原因：** 未添加 `@Security` 注释。

**解决：**
```go
//	@Security	BearerAuth
```

### Q5: Swagger JSON 显示为字符串

**原因：** 在 `HandleSwaggerJSON` 中使用了 `ctx.JSON()`。

**解决：** 必须直接写入原始 JSON：
```go
// ❌ 错误
ctx.JSON(200, doc)

// ✅ 正确
ctx.SetStatusCode(200)
ctx.SetContentType("application/json; charset=utf-8")
ctx.Response.SetBody([]byte(doc))
```

### Q6: 如何定义通用响应类型

**方法 1：定义结构体**
```go
// ErrorResponse 错误响应
type ErrorResponse struct {
    Error struct {
        Code    string `json:"code"`
        Message string `json:"message"`
    } `json:"error"`
}

// 在注释中使用
//	@Failure	400	{object}	ErrorResponse
```

**方法 2：使用 map**
```go
//	@Failure	400	{object}	map[string]interface{}
```

### Q7: 如何支持文件上传

```go
// UploadFile 上传文件
//
//	@Summary		上传文件
//	@Description	上传单个文件
//	@Tags			Files
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			file	formData	file	true	"文件"
//	@Success		200	{object}	UploadResponse
//	@Router			/files/upload [post]
func (h *FileHandler) UploadFile(c context.Context, ctx *app.RequestContext) {
    // ...
}
```

### Q8: 如何定义枚举类型

```go
// ProviderStatus 提供商状态
type ProviderStatus string

const (
    StatusActive   ProviderStatus = "active"
    StatusInactive ProviderStatus = "inactive"
)

// 使用 @Enum 标注
//	@Param		status	query	string	false	"状态"	Enums(active, inactive)
```

## 最佳实践

1. **注释规范**
   - 保持 `@Summary` 简洁（不超过 50 字符）
   - 在 `@Description` 中提供详细信息
   - 使用有意义的 `@Tags` 进行分组（如 "Authentication", "Sessions", "Chats"）

2. **错误响应**
   - 为所有可能的错误码定义响应
   - 使用一致的错误响应结构

3. **数据模型**
   - 为所有请求/响应定义明确的 struct
   - 使用 `json` tag 指定 JSON 字段名
   - 添加 `example` tag 提供示例值

4. **版本管理**
   - 将生成的 `docs/` 目录提交到版本控制
   - 每次 API 变更后更新文档

5. **安全性**
   - 需要认证的接口添加 `@Security` 注释
   - 敏感接口在描述中说明权限要求

## 参考资料

- [swaggo/swag 官方文档](https://github.com/swaggo/swag)
- [Swagger 注释规范](https://github.com/swaggo/swag/blob/master/README.md#general-api-info)
- [OpenAPI 规范](https://swagger.io/specification/)
- [Hertz 框架文档](https://cloudwego.io/docs/hertz/)

## 更新日志

- 2024-01-23: 初始版本，集成 swaggo/swag
- 支持 JWT 认证自动管理
- 提供完整的 Swagger UI 配置

---

**维护者：** 开发团队
**最后更新：** 2024-01-23
