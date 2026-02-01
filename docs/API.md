# Crush REST API 设计文档

## 概述

Crush REST API 提供了程序化的方式来管理项目、会话和消息。当使用 `serve` 子命令启动时，程序会启动一个 HTTP API 服务器而不是交互式 TUI。

## 启动 API 服务器

```bash
# 启动 API 服务器（默认端口 8080，监听 localhost）
crush serve

# 指定端口和地址
crush serve --port 3000 --host 0.0.0.0

# 指定工作目录和数据目录
crush serve -c /path/to/project -D /path/to/data
```

### 命令行参数

**serve 子命令参数**：
- `--port`: API 服务器端口（默认: 8080）
- `--host`: 监听地址（默认: localhost）

**全局参数**（适用于所有命令）：
- `--cwd` / `-c`: 当前工作目录
- `--data-dir` / `-D`: 自定义数据目录
- `--debug` / `-d`: 启用调试日志
- `--yolo` / `-y`: 自动接受所有权限请求（危险模式）

**注意**：`--server` 标志已废弃，建议使用 `crush serve` 子命令。

## API 基础信息

### Base URL

```
http://localhost:8080
```

### 响应格式

所有 API 响应都使用 JSON 格式。错误响应格式：

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable error message",
    "details": {}
  }
}
```

### HTTP 状态码

- `200 OK`: 请求成功
- `201 Created`: 资源创建成功
- `400 Bad Request`: 请求参数错误
- `404 Not Found`: 资源不存在
- `500 Internal Server Error`: 服务器内部错误

## API 端点

### 1. Projects（项目管理）

#### 1.1 获取所有项目

```http
GET /project
```

**响应示例**：

```json
{
  "projects": [
    {
      "path": "/path/to/project1",
      "data_dir": "/home/user/.crush/project1",
      "last_accessed": "2024-01-15T10:30:00Z"
    }
  ]
}
```

#### 1.2 注册项目

```http
POST /project
Content-Type: application/json
```

**请求体**：

```json
{
  "path": "/path/to/project",
  "data_dir": "/home/user/.crush/project"  // 可选
}
```

#### 1.3 获取当前项目

```http
GET /project/current
```

### 2. Sessions（会话管理）

**注意**：所有会话相关的 API 都需要通过查询参数 `directory` 指定项目路径。

#### 2.1 获取会话列表

```http
GET /session?directory=/path/to/project
```

#### 2.2 创建新会话

```http
POST /session?directory=/path/to/project
Content-Type: application/json

{
  "title": "New conversation session"
}
```

#### 2.3 获取单个会话详情

```http
GET /session/{session_id}?directory=/path/to/project
```

#### 2.4 更新会话

```http
PUT /session/{session_id}?directory=/path/to/project
Content-Type: application/json

{
  "title": "Updated session title"
}
```

#### 2.5 删除会话

```http
DELETE /session/{session_id}?directory=/path/to/project
```

#### 2.6 中止 AI 处理

```http
POST /session/{session_id}/abort?directory=/path/to/project
```

### 3. Messages（消息管理）

#### 3.1 获取会话的所有消息

```http
GET /session/{session_id}/message?directory=/path/to/project
```

#### 3.2 发送消息并运行 AI

```http
POST /session/{session_id}/message?directory=/path/to/project
Content-Type: application/json

{
  "prompt": "Explain the use of context in Go",
  "stream": true
}
```

#### 3.3 获取单个消息

```http
GET /message/{id}
```

### 4. Files & Search（文件与搜索）

#### 4.1 搜索文本内容

```http
GET /find?directory={path}&query={query}
```

#### 4.2 搜索文件名

```http
GET /find/file?directory={path}&query={query}
```

#### 4.3 列出文件/目录

```http
GET /file?directory={path}
```

#### 4.4 读取文件内容

```http
GET /file/content?directory={path}&path={file_path}
```

#### 4.5 获取 Git 状态

```http
GET /file/status?directory={path}
```

### 5. Config & Permissions（配置与权限）

#### 5.1 获取项目配置

```http
GET /project/config?directory=/path/to/project
```

#### 5.2 获取权限列表

```http
GET /project/permissions?directory=/path/to/project
```

#### 5.3 回复权限请求

```http
POST /project/permissions/{request_id}/reply?directory=/path/to/project
Content-Type: application/json

{
  "granted": true,
  "persistent": false
}
```

### 6. Global & System（全局与系统）

#### 6.1 健康检核

```http
GET /global/health
```

#### 6.2 释放所有资源

```http
POST /global/dispose
```

#### 6.3 释放单个项目实例

```http
POST /instance/dispose
Content-Type: application/json

{
  "directory": "/path/to/project"
}
```

### 7. SSE 事件订阅

#### 7.1 订阅实时事件

```http
GET /event?directory=/path/to/project
Accept: text/event-stream
```

**响应格式 (Opencode 风格)**：

所有事件都遵循统一的包装格式：

```json
{
  "type": "event.name",
  "properties": {
    "key": "value"
  }
}
```

**常见事件类型**：
- `server.connected`: 成功连接到事件流
- `message.created`: 新消息已创建（或生成完成）
- `message.updated`: 消息内容更新（流式输出中）
- `message.removed`: 消息被删除
- `session.created`: 新会话已创建
- `session.updated`: 会话信息更新
- `session.deleted`: 会话被删除
- `lsp.server.state_changed`: LSP 服务器状态变化
- `lsp.client.diagnostics`: LSP 诊断结果更新

