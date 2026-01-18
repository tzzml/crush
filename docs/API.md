# Crush REST API 设计文档

## 概述

Crush REST API 提供了程序化的方式来管理项目、会话和消息。当使用 `--server` 标志启动时，程序会启动一个 HTTP API 服务器而不是交互式 TUI。

## 启动 API 服务器

```bash
# 启动 API 服务器（默认端口 8080，监听 localhost）
crush --server

# 指定端口和地址
crush --server --port 3000 --host 0.0.0.0

# 指定工作目录和数据目录
crush --server -c /path/to/project -D /path/to/data
```

### 命令行参数

- `--server`: 启动 API 服务器模式
- `--port` / `-p`: API 服务器端口（默认: 8080）
- `--host`: 监听地址（默认: localhost）
- `--cwd` / `-c`: 当前工作目录
- `--data-dir` / `-D`: 自定义数据目录
- `--debug` / `-d`: 启用调试日志
- `--yolo` / `-y`: 自动接受所有权限请求（危险模式）

## API 基础信息

### Base URL

```
http://localhost:8080/api/v1
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
GET /api/v1/projects
```

**响应示例**：

```json
{
  "projects": [
    {
      "path": "/path/to/project1",
      "data_dir": "/home/user/.crush/project1",
      "last_accessed": "2024-01-15T10:30:00Z"
    },
    {
      "path": "/path/to/project2",
      "data_dir": "/home/user/.crush/project2",
      "last_accessed": "2024-01-14T15:20:00Z"
    }
  ]
}
```

#### 1.2 添加/注册项目

```http
POST /api/v1/projects
Content-Type: application/json
```

**请求体**：

```json
{
  "path": "/path/to/project",
  "data_dir": "/home/user/.crush/project"  // 可选，如果不提供会自动生成
}
```

**响应示例**：

```json
{
  "project": {
    "path": "/path/to/project",
    "data_dir": "/home/user/.crush/project",
    "last_accessed": "2024-01-15T10:30:00Z"
  }
}
```

**说明**：
- 如果项目已存在，会更新 `last_accessed` 时间
- 如果 `data_dir` 未提供，会根据项目路径自动生成

#### 1.3 打开项目（创建 app 实例）

在使用项目的会话和消息功能之前，必须先打开项目以创建 app 实例。

```http
POST /api/v1/projects/{project_path}/open
Content-Type: application/json
```

**路径参数**：
- `project_path`: 项目的路径（需要 URL 编码）

**响应示例**：

```json
{
  "project_path": "/path/to/project",
  "status": "opened"
}
```

**错误码**：
- `PROJECT_NOT_FOUND`: 项目不存在
- `INTERNAL_ERROR`: 打开失败（如配置加载失败、数据库连接失败等）

**说明**：
- 此操作是幂等的：如果项目已经打开，再次调用会直接返回成功，不会报错

#### 1.4 关闭项目（清理 app 实例）

关闭项目会清理 app 实例并释放相关资源。

```http
POST /api/v1/projects/{project_path}/close
Content-Type: application/json
```

**路径参数**：
- `project_path`: 项目的路径（需要 URL 编码）

**响应示例**：

```json
{
  "project_path": "/path/to/project",
  "status": "closed"
}
```

**错误码**：
- `INTERNAL_ERROR`: 关闭失败

**说明**：
- 此操作是幂等的：如果项目已经关闭，再次调用会直接返回成功，不会报错

### 2. Sessions（会话管理）

**重要**：在使用会话和消息 API 之前，必须先调用 `POST /api/v1/projects/{project_path}/open` 打开项目。

#### 2.1 获取项目下的所有会话

```http
GET /api/v1/projects/{project_path}/sessions
```

**路径参数**：
- `project_path`: 项目的路径（需要 URL 编码）

**查询参数**：
- `limit` (可选): 返回数量限制（默认: 50）
- `offset` (可选): 偏移量（默认: 0）

**响应示例**：

```json
{
  "sessions": [
    {
      "id": "session-uuid-1",
      "parent_session_id": null,
      "title": "Explain the use of context in Go",
      "message_count": 10,
      "prompt_tokens": 150,
      "completion_tokens": 500,
      "cost": 0.0025,
      "summary_message_id": null,
      "todos": [],
      "created_at": 1705312200000,
      "updated_at": 1705312300000
    }
  ],
  "total": 25
}
```

**说明**：
- `project_path` 需要与已注册的项目路径匹配
- 会话按 `updated_at` 降序排列（最新的在前）

#### 2.2 创建新会话

```http
POST /api/v1/projects/{project_path}/sessions
Content-Type: application/json
```

**请求体**：

```json
{
  "title": "New conversation session"
}
```

**响应示例**：

```json
{
  "session": {
    "id": "session-uuid-2",
    "parent_session_id": null,
    "title": "New conversation session",
    "message_count": 0,
    "prompt_tokens": 0,
    "completion_tokens": 0,
    "cost": 0.0,
    "summary_message_id": null,
    "todos": [],
    "created_at": 1705312400000,
    "updated_at": 1705312400000
  }
}
```

#### 2.3 获取单个会话详情

```http
GET /api/v1/projects/{project_path}/sessions/{session_id}
```

**路径参数**：
- `project_path`: 项目的路径（需要 URL 编码）
- `session_id`: 会话的唯一标识符

**响应示例**：

```json
{
  "session": {
    "id": "session-uuid-1",
    "parent_session_id": null,
    "title": "Explain the use of context in Go",
    "message_count": 10,
    "prompt_tokens": 150,
    "completion_tokens": 500,
    "cost": 0.0025,
    "summary_message_id": "message-uuid-1",
    "todos": [
      {
        "content": "Review the code",
        "status": "pending",
        "active_form": ""
      }
    ],
    "created_at": 1705312200000,
    "updated_at": 1705312300000
  }
}
```

#### 2.4 删除会话

```http
DELETE /api/v1/projects/{project_path}/sessions/{session_id}
```

**路径参数**：
- `project_path`: 项目的路径（需要 URL 编码）
- `session_id`: 会话的唯一标识符

**响应**：

```json
{
  "message": "Session deleted successfully"
}
```

### 3. Messages（消息管理）

#### 3.1 获取会话的所有消息

```http
GET /api/v1/projects/{project_path}/sessions/{session_id}/messages
```

**路径参数**：
- `project_path`: 项目的路径（需要 URL 编码）
- `session_id`: 会话的唯一标识符

**查询参数**：
- `limit` (可选): 返回数量限制（默认: 100）
- `offset` (可选): 偏移量（默认: 0）

**响应示例**：

```json
{
  "messages": [
    {
      "id": "message-uuid-1",
      "session_id": "session-uuid-1",
      "role": "user",
      "content": "Explain the use of context in Go",
      "model": "gpt-4",
      "provider": "openai",
      "is_summary_message": false,
      "created_at": 1705312200000,
      "updated_at": 1705312200000,
      "finished_at": 1705312201000
    },
    {
      "id": "message-uuid-2",
      "session_id": "session-uuid-1",
      "role": "assistant",
      "content": "Context in Go is a powerful mechanism...",
      "model": "gpt-4",
      "provider": "openai",
      "is_summary_message": false,
      "created_at": 1705312201000,
      "updated_at": 1705312205000,
      "finished_at": 1705312205000
    }
  ],
  "total": 10
}
```

**说明**：
- 消息按 `created_at` 升序排列（最早的在前）
- `content` 字段包含完整的消息内容
- `finished_at` 为 null 表示消息还在生成中

#### 3.2 发送消息并运行 AI（同步）

```http
POST /api/v1/projects/{project_path}/sessions/{session_id}/messages
Content-Type: application/json
```

**路径参数**：
- `project_path`: 项目的路径（需要 URL 编码）
- `session_id`: 会话的唯一标识符

**请求体**：

```json
{
  "prompt": "Explain the use of context in Go",
  "stream": false
}
```

**响应示例**：

```json
{
  "message": {
    "id": "message-uuid-3",
    "session_id": "session-uuid-1",
    "role": "assistant",
    "content": "Context in Go is a powerful mechanism for...",
    "model": "gpt-4",
    "provider": "openai",
    "is_summary_message": false,
    "created_at": 1705312400000,
    "updated_at": 1705312405000,
    "finished_at": 1705312405000
  },
  "session": {
    "id": "session-uuid-1",
    "title": "Explain the use of context in Go",
    "message_count": 12,
    "prompt_tokens": 200,
    "completion_tokens": 600,
    "cost": 0.003
  }
}
```

**说明**：
- 当 `stream: false` 时，API 会等待 AI 完成响应后返回完整消息
- 响应时间取决于 AI 模型的响应速度
- 响应中包含更新后的会话信息（token 使用量、成本等）

#### 3.3 发送消息并运行 AI（流式）

```http
POST /api/v1/projects/{project_path}/sessions/{session_id}/messages
Content-Type: application/json
```

**路径参数**：
- `project_path`: 项目的路径（需要 URL 编码）
- `session_id`: 会话的唯一标识符

**请求体**：

```json
{
  "prompt": "Explain the use of context in Go",
  "stream": true
}
```

**响应**：使用 Server-Sent Events (SSE) 格式

```
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive

event: message
data: {"type": "start", "message_id": "message-uuid-3"}

event: chunk
data: {"type": "chunk", "content": "Context"}

event: chunk
data: {"type": "chunk", "content": " in Go"}

event: chunk
data: {"type": "chunk", "content": " is a powerful"}

event: done
data: {"type": "done", "message_id": "message-uuid-3", "session": {...}}
```

**SSE 事件类型**：
- `start`: 消息开始生成，包含 `message_id`
- `chunk`: 内容片段，包含 `content`
- `done`: 消息生成完成，包含完整的 `message` 和更新后的 `session`
- `error`: 发生错误，包含 `error` 对象

#### 3.4 获取单个消息

```http
GET /api/v1/projects/{project_path}/messages/{message_id}
```

**路径参数**：
- `project_path`: 项目的路径（需要 URL 编码）
- `message_id`: 消息的唯一标识符

**响应示例**：

```json
{
  "message": {
    "id": "message-uuid-1",
    "session_id": "session-uuid-1",
    "role": "user",
    "content": "Explain the use of context in Go",
    "model": null,
    "provider": null,
    "is_summary_message": false,
    "created_at": 1705312200000,
    "updated_at": 1705312200000,
    "finished_at": 1705312200000
  }
}
```

### 4. Events（实时事件订阅）

#### 4.1 订阅项目实时事件

订阅指定项目的实时事件，包括会话和消息的创建、更新、删除等。

```http
GET /api/v1/projects/{project_path}/events
Accept: text/event-stream
```

**路径参数**：
- `project_path`: 项目的路径（需要 URL 编码）

**前置条件**：
- 项目必须已通过 `POST /api/v1/projects/{project_path}/open` 打开

**响应**：使用 Server-Sent Events (SSE) 格式

```
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive

event: connected
data: {"status": "connected"}

event: created
data: {"type": "created", "payload": {"id": "session-uuid-1", "title": "New Session", ...}}

event: updated
data: {"type": "updated", "payload": {"id": "message-uuid-1", "content": "Hello", ...}}

event: deleted
data: {"type": "deleted", "payload": {"id": "session-uuid-1", ...}}
```

**SSE 事件类型**：
- `connected`: 连接建立确认
- `created`: 资源创建事件（会话、消息等）
- `updated`: 资源更新事件（消息内容更新、会话状态变化等）
- `deleted`: 资源删除事件
- `heartbeat`: 心跳事件（每30秒发送一次，保持连接）

**事件数据格式**：
- 所有事件数据都是 JSON 格式
- `payload` 字段包含具体的事件数据（会话或消息对象）

**错误处理**：
- 如果项目未打开，返回 `APP_NOT_OPENED` 错误
- 如果项目不存在，返回 `PROJECT_NOT_FOUND` 错误

**说明**：
- SSE 连接会持续保持，直到客户端断开或项目被关闭
- 只接收指定项目的事件，不会收到其他项目的事件
- 建议客户端实现重连机制以处理网络中断

**路径参数**：
- `project_path`: 项目的路径（需要 URL 编码）
- `message_id`: 消息的唯一标识符

**响应示例**：

```json
{
  "message": {
    "id": "message-uuid-1",
    "session_id": "session-uuid-1",
    "role": "user",
    "content": "Explain the use of context in Go",
    "model": null,
    "provider": null,
    "is_summary_message": false,
    "created_at": 1705312200000,
    "updated_at": 1705312200000,
    "finished_at": 1705312200000
  }
}
```

## 使用示例

### 完整工作流示例

#### 1. 启动 API 服务器

```bash
crush --server --port 8080
```

#### 2. 注册项目

```bash
curl -X POST http://localhost:8080/api/v1/projects \
  -H "Content-Type: application/json" \
  -d '{
    "path": "/path/to/my/project"
  }'
```

#### 3. 打开项目

```bash
curl -X POST "http://localhost:8080/api/v1/projects/%2Fpath%2Fto%2Fmy%2Fproject/open" \
  -H "Content-Type: application/json" \
  -d '{}'
```

响应：
```json
{
  "project_path": "/path/to/my/project",
  "status": "opened"
}
```

#### 4. 创建会话

```bash
curl -X POST "http://localhost:8080/api/v1/projects/%2Fpath%2Fto%2Fmy%2Fproject/sessions" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "My first API session"
  }'
```

响应：
```json
{
  "session": {
    "id": "abc-123-def",
    "title": "My first API session",
    ...
  }
}
```

#### 5. 发送消息（同步）

```bash
curl -X POST "http://localhost:8080/api/v1/projects/my-project/sessions/abc-123-def/messages" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "What is Go?",
    "stream": false
  }'
```

#### 6. 发送消息（流式）

```bash
curl -X POST "http://localhost:8080/api/v1/projects/my-project/sessions/abc-123-def/messages" \
  -H "Content-Type: application/json" \
  -d '{
    "prompt": "What is Go?",
    "stream": true
  }' \
  --no-buffer
```

#### 7. 获取消息历史

```bash
curl "http://localhost:8080/api/v1/projects/my-project/sessions/abc-123-def/messages"
```

#### 8. 获取所有会话

```bash
curl "http://localhost:8080/api/v1/projects/%2Fpath%2Fto%2Fmy%2Fproject/sessions"
```

#### 9. 订阅实时事件（SSE）

```bash
curl -N "http://localhost:8080/api/v1/projects/%2Fpath%2Fto%2Fmy%2Fproject/events" \
  -H "Accept: text/event-stream"
```

响应示例：
```
event: connected
data: {"status": "connected"}

event: created
data: {"type": "created", "payload": {"id": "session-123", "title": "New Session"}}

event: updated
data: {"type": "updated", "payload": {"id": "message-456", "content": "Hello"}}
```

#### 10. 关闭项目（可选）

```bash
curl -X POST "http://localhost:8080/api/v1/projects/%2Fpath%2Fto%2Fmy%2Fproject/close" \
  -H "Content-Type: application/json" \
  -d '{}'
```

响应：
```json
{
  "project_path": "/path/to/my/project",
  "status": "closed"
}
```

```bash
curl "http://localhost:8080/api/v1/projects/%2Fpath%2Fto%2Fmy%2Fproject/sessions"
```

#### 9. 订阅实时事件（SSE）

```bash
curl -N "http://localhost:8080/api/v1/projects/%2Fpath%2Fto%2Fmy%2Fproject/events" \
  -H "Accept: text/event-stream"
```

响应示例：
```
event: connected
data: {"status": "connected"}

event: created
data: {"type": "created", "payload": {"id": "session-123", "title": "New Session"}}

event: updated
data: {"type": "updated", "payload": {"id": "message-456", "content": "Hello"}}
```

#### 10. 关闭项目（可选）

```bash
curl -X POST "http://localhost:8080/api/v1/projects/%2Fpath%2Fto%2Fmy%2Fproject/close" \
  -H "Content-Type: application/json" \
  -d '{}'
```

响应：
```json
{
  "project_path": "/path/to/my/project",
  "status": "closed"
}
```

### JavaScript/TypeScript 示例

```typescript
// 创建会话
const createSession = async (projectPath: string, title: string) => {
  const response = await fetch(
    `http://localhost:8080/api/v1/projects/${encodeURIComponent(projectPath)}/sessions`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ title }),
    }
  );
  const data = await response.json();
  return data.session;
};

// 发送消息（流式）
const sendMessageStream = async (
  sessionId: string,
  prompt: string,
  onChunk: (content: string) => void
) => {
  const response = await fetch(
    `http://localhost:8080/api/v1/sessions/${sessionId}/messages`,
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ prompt, stream: true }),
    }
  );

  const reader = response.body?.getReader();
  const decoder = new TextDecoder();

  while (true) {
    const { done, value } = await reader!.read();
    if (done) break;

    const chunk = decoder.decode(value);
    const lines = chunk.split('\n\n');

    for (const line of lines) {
      if (line.startsWith('data: ')) {
        const data = JSON.parse(line.slice(6));
        if (data.type === 'chunk') {
          onChunk(data.content);
        }
      }
    }
  }
};

// 订阅实时事件
const subscribeEvents = async (projectPath: string, onEvent: (event: any) => void) => {
  const response = await fetch(
    `http://localhost:8080/api/v1/projects/${encodeURIComponent(projectPath)}/events`,
    {
      headers: { 'Accept': 'text/event-stream' },
    }
  );

  const reader = response.body?.getReader();
  const decoder = new TextDecoder();

  while (true) {
    const { done, value } = await reader!.read();
    if (done) break;

    const chunk = decoder.decode(value);
    const lines = chunk.split('\n\n');

    for (const line of lines) {
      if (line.startsWith('event: ')) {
        const eventType = line.slice(7);
      } else if (line.startsWith('data: ')) {
        const data = JSON.parse(line.slice(6));
        onEvent({ type: eventType, data });
      }
    }
  }
};

// 使用示例
const session = await createSession('/path/to/project', 'My Session');
await sendMessageStream(session.id, 'Hello!', (chunk) => {
  process.stdout.write(chunk);
});

// 订阅事件
subscribeEvents('/path/to/project', (event) => {
  console.log('收到事件:', event);
});
```

### Python 示例

```python
import requests
import json

BASE_URL = "http://localhost:8080/api/v1"

# 打开项目
def open_project(project_path: str):
    response = requests.post(
        f"{BASE_URL}/projects/{requests.utils.quote(project_path)}/open",
        json={}
    )
    return response.json()

# 创建会话
def create_session(project_path: str, title: str):
    response = requests.post(
        f"{BASE_URL}/projects/{requests.utils.quote(project_path)}/sessions",
        json={"title": title}
    )
    return response.json()["session"]

# 发送消息（同步）
def send_message(session_id: str, prompt: str):
    response = requests.post(
        f"{BASE_URL}/sessions/{session_id}/messages",
        json={"prompt": prompt, "stream": False}
    )
    return response.json()

# 发送消息（流式）
def send_message_stream(session_id: str, prompt: str):
    response = requests.post(
        f"{BASE_URL}/sessions/{session_id}/messages",
        json={"prompt": prompt, "stream": True},
        stream=True
    )
    
    for line in response.iter_lines():
        if line.startswith(b"data: "):
            data = json.loads(line[6:])
            if data.get("type") == "chunk":
                yield data.get("content", "")

# 订阅实时事件
def subscribe_events(project_path: str, on_event=None):
    import sseclient
    encoded_path = requests.utils.quote(project_path, safe="")
    sse_url = f"{BASE_URL}/projects/{encoded_path}/events"
    
    response = requests.get(sse_url, stream=True, headers={
        'Accept': 'text/event-stream',
        'Cache-Control': 'no-cache',
    })
    
    client = sseclient.SSEClient(response)
    for event in client.events():
        data = json.loads(event.data)
        if on_event:
            on_event(event.event, data)
        else:
            print(f"事件 [{event.event}]: {data}")

# 使用示例
project_path = "/path/to/project"
open_project(project_path)  # 必须先打开项目
session = create_session(project_path, "My Session")

# 在后台订阅事件
import threading
threading.Thread(
    target=subscribe_events,
    args=(project_path, lambda t, d: print(f"收到事件: {t}")),
    daemon=True
).start()

# 发送消息
for chunk in send_message_stream(session["id"], "Hello!"):
    print(chunk, end="", flush=True)
```

## 错误处理

### 常见错误码

- `PROJECT_NOT_FOUND`: 项目不存在
- `APP_NOT_OPENED`: 项目的 app 实例未打开（需要先调用 open API）

**注意**：
- `open` 和 `close` 操作是幂等的：重复调用不会报错，会直接返回成功
- `SESSION_NOT_FOUND`: 会话不存在
- `MESSAGE_NOT_FOUND`: 消息不存在
- `INVALID_PROJECT_PATH`: 项目路径无效
- `PROVIDER_NOT_CONFIGURED`: AI 提供商未配置
- `INVALID_REQUEST`: 请求参数无效
- `INTERNAL_ERROR`: 服务器内部错误

### 错误响应示例

```json
{
  "error": {
    "code": "SESSION_NOT_FOUND",
    "message": "Session with id 'abc-123' not found",
    "details": {
      "session_id": "abc-123"
    }
  }
}
```

## 注意事项

1. **项目生命周期管理**：在使用会话和消息功能之前，必须先调用 `POST /api/v1/projects/{project_path}/open` 打开项目。使用完毕后可以调用 `POST /api/v1/projects/{project_path}/close` 关闭项目以释放资源。
2. **项目路径编码**：在 URL 中使用项目路径时，需要进行 URL 编码
3. **会话隔离**：每个项目的会话数据存储在对应的 `data_dir` 中，相互独立
4. **并发限制**：建议对同一会话的并发请求进行限制，避免冲突
5. **流式响应**：使用流式响应时，确保客户端正确处理 SSE 格式
6. **权限管理**：在 `--yolo` 模式下，所有权限请求会自动批准；否则需要手动处理
7. **数据持久化**：所有数据存储在 SQLite 数据库中，位于项目的 `data_dir` 目录
8. **资源管理**：打开的 app 实例会占用内存和数据库连接，建议在不使用时及时关闭

## 未来扩展

可能的未来功能：

- WebSocket 支持实时双向通信
- 认证和授权机制
- 批量操作 API
- 文件上传和管理 API
- 会话导出/导入功能
- 统计和分析 API
