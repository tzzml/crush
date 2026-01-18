# Crush API 客户端使用指南

## 快速开始

### 1. 启动服务器

```bash
crush --server
```

### 2. 运行测试

```bash
# 完整 API 测试
python3 docs/test_api.py

# SSE 测试
python3 docs/test_sse.py

# 客户端演示
python3 docs/test_client.py
```

## 依赖安装

```bash
pip install requests sseclient-py
```

## 核心 API

### 项目管理
- `POST /api/v1/projects/{path}/open` - 打开项目
- `POST /api/v1/projects/{path}/close` - 关闭项目

### 会话和消息
- `GET /api/v1/projects/{path}/sessions` - 列出会话
- `POST /api/v1/projects/{path}/sessions` - 创建会话
- `POST /api/v1/projects/{path}/sessions/{id}/messages` - 发送消息

### 实时事件
- `GET /api/v1/projects/{path}/events` - 订阅 SSE 事件

**注意**：使用项目相关 API 前，必须先调用 `open` 打开项目。

## 代码示例

```python
from docs.test_client import CrushClient

client = CrushClient("http://localhost:8080/api/v1")

# 打开项目
client.open_project("/tmp/my-project")

# 创建会话
session = client.create_session("/tmp/my-project", "测试会话")

# 发送消息
message = client.send_message("/tmp/my-project", session['id'], "你好")

# 订阅事件
client.subscribe_events("/tmp/my-project", callback=lambda t, d: print(t, d))
```

详细 API 文档请参考 [API.md](./API.md)。
