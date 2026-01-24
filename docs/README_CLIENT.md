# Crush API 客户端使用指南

## 快速开始

### 1. 启动服务器

```bash
crush serve
```

### 2. 运行测试

```bash
# 完整 API 测试
python3 docs/test_api.py

# SSE 测试
python3 docs/test_sse.py
```

## 依赖安装

```bash
pip install requests sseclient-py
```

## 核心 API

### 项目管理
- `GET /project` - 列出项目
- `POST /project` - 注册项目

### 会话和消息
- `GET /session?directory={path}` - 列出会话
- `POST /session?directory={path}` - 创建会话
- `POST /session/{id}/message?directory={path}` - 发送消息

### 实时事件
- `GET /event?directory={path}` - 订阅 SSE 事件

## 代码示例

```python
import requests

BASE_URL = "http://localhost:8080"
PROJECT = "/tmp/my-project"

# 注册项目
requests.post(f"{BASE_URL}/project", json={"path": PROJECT})

# 创建会话
resp = requests.post(f"{BASE_URL}/session", 
                     params={"directory": PROJECT},
                     json={"title": "测试会话"})
session_id = resp.json()["session"]["id"]

# 发送消息
requests.post(f"{BASE_URL}/session/{session_id}/message",
              params={"directory": PROJECT},
              json={"prompt": "你好", "stream": False})
```

详细 API 文档请参考 [API.md](./API.md)。
