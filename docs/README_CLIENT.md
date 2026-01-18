# Crush API 客户端测试指南

## 快速开始

### 1. 启动API服务器

```bash
# 在后台启动API服务器
crush --server

# 或者指定端口
crush --server --port 8080
```

### 2. 运行测试

#### 完整API测试（推荐）
```bash
# 测试所有功能，包括SSE
python3 docs/test_api.py

# 指定API服务器地址
python3 docs/test_api.py --base-url http://localhost:8080

# 指定测试项目路径
python3 docs/test_api.py --project-path /path/to/test/project
```

#### SSE专用测试
```bash
# 仅测试SSE实时事件
python3 docs/test_sse.py
```

#### 客户端演示
```bash
# 运行客户端演示
python3 docs/test_client.py

# 仅运行基本API演示
python3 docs/test_client.py --demo basic

# 仅运行SSE演示
python3 docs/test_client.py --demo sse

# 仅运行组合演示
python3 docs/test_client.py --demo combined
```

## 测试内容

### 1. REST API测试 (`test_api.py`)

- ✅ 项目管理（创建、列出）
- ✅ 会话管理（创建、获取）
- ✅ 消息发送（同步流式）
- ✅ SSE实时事件订阅
- ✅ 错误处理

### 2. SSE测试 (`test_sse.py`)

- ✅ SSE连接建立
- ✅ 实时事件接收
- ✅ 事件数据解析
- ✅ 连接稳定性

### 3. 客户端演示 (`test_client.py`)

- ✅ 基本API使用示例
- ✅ SSE事件订阅演示
- ✅ REST API + SSE组合使用
- ✅ 多线程事件处理

## 依赖安装

```bash
# 安装必需的Python包
pip install requests sseclient-py
```

## API端点

### REST API

- `GET /api/v1/projects` - 列出项目
- `POST /api/v1/projects` - 创建项目
- `GET /api/v1/projects/{path}/sessions` - 获取项目会话
- `POST /api/v1/projects/{path}/sessions` - 创建会话
- `GET /api/v1/projects/{path}/sessions/{id}` - 获取会话详情
- `POST /api/v1/projects/{path}/sessions/{id}/messages` - 发送消息

### SSE (Server-Sent Events)

- `GET /api/v1/projects/{project_path}/events` - 订阅项目的实时事件

**注意**：项目必须先通过 `POST /api/v1/projects/{project_path}/open` 打开

#### 支持的事件类型

- `connected` - 连接确认
- `updated` - 数据更新（LSP状态、诊断信息等）
- `created` - 新资源创建
- `deleted` - 资源删除

## 客户端代码示例

### Python客户端

```python
from docs.test_client import CrushClient

# 创建客户端
client = CrushClient("http://localhost:8080/api/v1")

# 使用REST API
projects = client.list_projects()
project = client.create_project("/tmp/my-project")
session = client.create_session("/tmp/my-project", "测试会话")
message = client.send_message("/tmp/my-project", session['id'], "你好")

# 订阅SSE事件
def handle_event(event_type, data):
    print(f"收到事件: {event_type}", data)

client.subscribe_events(callback=handle_event)
```

### JavaScript客户端

```javascript
// 订阅SSE事件
const eventSource = new EventSource('/api/v1/events');

eventSource.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('收到事件:', event.type, data);
};

eventSource.addEventListener('connected', (event) => {
  console.log('SSE连接已建立');
});

eventSource.addEventListener('updated', (event) => {
  const data = JSON.parse(event.data);
  console.log('数据更新:', data);
});
```

## 故障排除

### 常见问题

1. **连接失败**
   ```bash
   # 检查服务器是否运行
   curl http://localhost:8080/api/v1/health

   # 检查端口是否被占用
   lsof -i :8080
   ```

2. **SSE无事件**
   - 确认项目中有LSP服务器配置
   - 检查项目路径是否正确
   - 等待LSP初始化完成（可能需要几秒）

3. **Python依赖错误**
   ```bash
   pip install --upgrade requests sseclient-py
   ```

### 日志查看

```bash
# 查看服务器日志
tail -f ~/.crush/logs/crush.log

# 查看详细的API请求日志
tail -f /tmp/crush-server.log
```

## 性能测试

```bash
# 压力测试REST API
ab -n 100 -c 10 http://localhost:8080/api/v1/projects

# 测试SSE连接稳定性
timeout 300 python3 docs/test_sse.py
```