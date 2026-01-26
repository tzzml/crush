# Swagger 文档标签更新总结

## ✅ 更新完成

已成功更新 Swagger 文档中的 API 标签和描述。

## 修改内容

### 1. 系统提示词端点 - 移至 Session 组

**GET /system-prompt**
- 标签: `System Prompt` → `Session`
- 说明: 现在与会话相关的 API 放在一起

**PUT /system-prompt**
- 标签: `System Prompt` → `Session`
- 说明: 系统提示词与会话管理相关

### 2. 事件端点 - 更新描述

**GET /event**
- Summary: `服务器发送事件` → `订阅服务器事件`
- Description: 保持不变 `订阅项目的实时事件流`

## 验证结果

```bash
# GET /system-prompt 标签
"Session" ✅

# PUT /system-prompt 标签
"Session" ✅

# GET /event 摘要
"订阅服务器事件" ✅
```

## 重新生成命令

```bash
swag init -g cmd/server.go -o docs --parseInternal --parseDepth 1
```

## 查看 Swagger UI

启动服务后访问 `http://localhost:8080/swagger`，您会看到：

- **Session** 标签下包含：
  - GET /system-prompt - 获取系统提示词
  - PUT /system-prompt - 更新系统提示词
  - 以及其他会话相关的 API

- **Event** 标签下包含：
  - GET /event - 订阅服务器事件

## 修改的文件

- `api/handlers/system_prompt.go` - 更新两个端点的 @Tags 为 Session
- `api/handlers/events.go` - 更新 @Summary 为"订阅服务器事件"

## 生成的文档

- `docs/swagger.json` - 已更新 ✅
- `docs/swagger.yaml` - 已更新 ✅
- `docs/docs.go` - 已更新 ✅
