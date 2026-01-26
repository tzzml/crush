# OpenAPI 和 Swagger 文档更新总结

## 更新内容

已成功将系统提示词 API 端点添加到 OpenAPI 3.0 文档中。

## 新增的 API 端点

### 1. GET /system-prompt
获取指定项目当前的系统提示词内容。

**路径**: `/system-prompt`

**方法**: `GET`

**参数**:
- `directory` (query, required): 项目路径

**响应** (200):
```json
{
  "system_prompt": "string",
  "length": 1234,
  "is_custom": true
}
```

**Schema**: `models.GetSystemPromptResponse`

### 2. PUT /system-prompt
动态修改指定项目的系统提示词，无需重启服务。

**路径**: `/system-prompt`

**方法**: `PUT`

**参数**:
- `directory` (query, required): 项目路径

**请求体**:
```json
{
  "system_prompt": "string"
}
```

**响应** (200):
```json
{
  "success": true,
  "system_prompt": "string",
  "message": "string (optional)"
}
```

**Schema**: `models.UpdateSystemPromptRequest` (请求)
**Schema**: `models.UpdateSystemPromptResponse` (响应)

## 新增的 Schema 定义

### models.GetSystemPromptResponse
```json
{
  "type": "object",
  "properties": {
    "system_prompt": {
      "type": "string",
      "description": "当前的系统提示词内容"
    },
    "length": {
      "type": "integer",
      "description": "系统提示词的字符数"
    },
    "is_custom": {
      "type": "boolean",
      "description": "是否为自定义提示词（非空表示已自定义）"
    }
  }
}
```

### models.UpdateSystemPromptRequest
```json
{
  "type": "object",
  "required": ["system_prompt"],
  "properties": {
    "system_prompt": {
      "type": "string",
      "description": "新的系统提示词内容"
    }
  }
}
```

### models.UpdateSystemPromptResponse
```json
{
  "type": "object",
  "properties": {
    "success": {
      "type": "boolean",
      "description": "是否成功更新"
    },
    "system_prompt": {
      "type": "string",
      "description": "更新后的系统提示词内容"
    },
    "message": {
      "type": "string",
      "description": "操作结果消息（可选）"
    }
  }
}
```

## Tag 标签

新增了 `System Prompt` 标签，用于组织和分类这些端点。

## 错误响应

所有端点都支持标准的错误响应：

- **400 Bad Request** - 请求参数错误
- **404 Not Found** - 项目不存在
- **500 Internal Server Error** - 服务器内部错误

## 访问 Swagger UI

更新 OpenAPI 文档后，可以通过以下地址访问 Swagger UI：

- **Swagger UI**: `http://localhost:8080/swagger`
- **Redoc**: `http://localhost:8080/redoc`
- **OpenAPI JSON**: `http://localhost:8080/swagger/openapi3.json`

## 验证

已验证：
- ✅ JSON 格式有效
- ✅ 新增端点 `/system-prompt` 已添加
- ✅ GET 方法已定义
- ✅ PUT 方法已定义
- ✅ 所有 Schema 定义已添加
- ✅ 参数和响应定义完整

## 文件更改

**修改文件**:
- `docs/openapi3.json` (+541 行)

## 下一步

1. 启动 API 服务器
2. 访问 `http://localhost:8080/swagger` 查看 Swagger UI
3. 在 "System Prompt" 标签下找到新的端点
4. 可以直接在 Swagger UI 中测试这些 API

## 示例使用

### 在 Swagger UI 中测试

1. 打开 `http://localhost:8080/swagger`
2. 展开 "System Prompt" 部分
3. 点击 "GET /system-prompt" 或 "PUT /system-prompt"
4. 点击 "Try it out"
5. 填写参数（directory: 项目路径）
6. 点击 "Execute" 执行请求

### 使用 curl

```bash
# GET - 获取系统提示词
curl -X GET "http://localhost:8080/system-prompt?directory=/path/to/project"

# PUT - 更新系统提示词
curl -X PUT "http://localhost:8080/system-prompt?directory=/path/to/project" \
  -H "Content-Type: application/json" \
  -d '{"system_prompt": "You are an expert Go developer."}'
```
